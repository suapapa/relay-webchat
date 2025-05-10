package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var (
	flagWebSocketServer  string
	flagRetriveCnt       int
	flagPromptPreProcess bool
	flagEmbedderType     string
	flagSecretFile       string
	hominDevAI           *HominDevAI
)

func main() {
	defer func() { log.Println("Exiting...") }()

	flag.StringVar(&flagWebSocketServer, "ws", "ws://localhost:8080", "WebSocket server address")
	flag.IntVar(&flagRetriveCnt, "retrive", 50, "Retrive count")
	flag.BoolVar(&flagPromptPreProcess, "pre-process", false, "Pre-process prompt")
	flag.StringVar(&flagEmbedderType, "embedder", "ollama", "Embedder type (ollama, openai)")
	flag.StringVar(&flagSecretFile, "secret", "/secret/token", "Secret file")
	flag.Parse()

	var secret string
	secretB, err := os.ReadFile(flagSecretFile)
	if err != nil {
		fmt.Printf("WARN: failed to read secret: %v\n", err)
	} else {
		log.Println("using secret from file")
		secret = strings.TrimSpace(string(secretB))
	}

	stat := &Stat{}
	log.Printf("WebSocket server: %s", flagWebSocketServer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hominDevAI, err = NewHominDevAI(ctx)
	if err != nil {
		log.Fatalf("Failed to create HominDevAI: %v", err)
	}

	var conn *websocket.Conn
	connectWS := func() error {
		var err error

		reqHeader := http.Header{}
		if secret != "" {
			reqHeader.Set("Authorization", fmt.Sprintf("Bearer %s", secret))
		}

		conn, _, err = websocket.DefaultDialer.Dial(flagWebSocketServer, reqHeader)
		if err != nil {
			return fmt.Errorf("failed to connect to WebSocket server: %v", err)
		}
		return nil
	}

	if err := connectWS(); err != nil {
		log.Fatalf("%v", err)
	}
	defer conn.Close()

	// get termination signals (systemctl restart sends SIGTERM, not os.Interrupt)
	chCtrlC := make(chan os.Signal, 1)
	signal.Notify(chCtrlC, os.Interrupt, syscall.SIGTERM)

	msgChan := make(chan struct {
		msgType int
		msg     []byte
		err     error
	})

	// Start message reading goroutine
	readErrRetryCnt, writeErrRetryCnt := 0, 0
	go func() {
		for {
			msgType, msgBytes, err := conn.ReadMessage()
			msgChan <- struct {
				msgType int
				msg     []byte
				err     error
			}{msgType, msgBytes, err}
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("Reconnecting...")
					if err := connectWS(); err != nil {
						log.Fatalf("Failed to reconnect: %v", err)
					}
				} else {
					log.Printf("Read error: %v", err)
					readErrRetryCnt++
					if readErrRetryCnt > 3 {
						log.Fatalf("Failed to reconnect after 3 attempts: %v", err)
					}
				}
			} else {
				readErrRetryCnt = 0
			}
		}
	}()

	for {
		select {
		case <-chCtrlC:
			log.Println("Ctrl-C pressed, exiting...")
			// Close WebSocket connection gracefully
			if conn != nil {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				conn.Close()
			}
			cancel()
		case <-ctx.Done():
			log.Println("Context canceled, exiting...")
			return
		case msgData := <-msgChan:
			if msgData.err != nil {
				log.Printf("Connection error: %v", msgData.err)
				time.Sleep(1 * time.Second)
				continue
			}

			msg := string(msgData.msg)
			msg = strings.TrimSpace(msg)

			var reply string
			if len([]rune(msg)) > 200 {
				log.Printf("Message is too long: %d", len([]rune(msg)))
				reply = "메시지가 너무 길어요. 200자 이하로 짧게 줄여주세요."
			} else if len(msg) == 0 {
				log.Printf("Empty message")
				continue
			} else {
				log.Printf("Received message: %s", msg)

				var cmd Cmd
				if !strings.HasPrefix(msg, "/") {
					cmd, err = hominDevAI.PreProcessFLow.Run(context.Background(), msg)
					if err != nil {
						log.Printf("Failed to run intent flow: %v", err)
						return
					}
				} else {
					msgParts := strings.Split(msg, " ")
					if len(msgParts) > 0 {
						cmd = Cmd{
							Action: msgParts[0],
							Args:   msgParts[1:],
						}
					}
				}
				log.Printf("Command: %s", cmd)

				switch cmd.Action {
				case "/keyword":
					searchKeywords := strings.Join(cmd.Args, ",")
					posts, err := retrivePost(searchKeywords, flagRetriveCnt)
					if err != nil {
						log.Printf("Failed to retrive post for keywords, %s: %v", searchKeywords, err)
						return
					}
					stat.TotalKeywordCnt++
					reply = fmt.Sprintf("%s 에 대한 검색 결과:\n%s", searchKeywords, makePostReply(posts))
				case "/search":
					posts, err := retrivePost(msg, flagRetriveCnt)
					if err != nil {
						log.Printf("Failed to retrive post for msg, %s: %v", msg, err)
						return
					}

					stat.TotalSearchCnt++
					if len(posts) == 0 {
						reply = "검색 결과가 없습니다."
					} else {
						reply = "검색 결과:\n" + makePostReply(posts)
					}
				case "/smallchat":
					stat.TotalSmallChatCnt++
					reply = strings.Join(cmd.Args, "\n")
				case "/about", "/start", "/help":
					reply = makeAboutReply()
				case "/stat":
					reply = stat.String()
				default:
					log.Printf("Unknown command: %s", cmd.Action)
					stat.TotalUnknownCnt++
					reply = makeAboutReply()
				}
			}

			if err := conn.WriteMessage(msgData.msgType, []byte(reply)); err != nil {
				log.Printf("Write error: %v", err)
				time.Sleep(1 * time.Second)
				writeErrRetryCnt++
				if writeErrRetryCnt > 3 {
					log.Fatalf("Failed to write after 3 attempts: %v", err)
				}
			} else {
				writeErrRetryCnt = 0
			}
		}
	} // for
}

func makePostReply(posts []*Post) string {
	// reply := "검색 결과:\n"
	var reply string
	for i, post := range posts {
		if i >= 5 {
			break
		}
		reply += fmt.Sprintf("- [%s](%s) - %s\n", post.Title, post.Url, strings.Join(post.Texts, ","))
	}
	return reply
}

func makeAboutReply() string {
	siteURL := `https://homin.dev/blog/post/20250507_rag_blog_search_webchat_got/`

	return fmt.Sprintf(
		`내 이름은 **블검봇**.

Homin Lee's blog를 검색합니다. 편하게 물어보세요.
제가 만들어진 내용은 [여기](%s)에 있습니다.`,
		siteURL,
	)
}

type Stat struct {
	TotalSmallChatCnt int
	TotalKeywordCnt   int
	TotalSearchCnt    int
	TotalUnknownCnt   int
}

func (s *Stat) String() string {
	return fmt.Sprintf(`- TotalSmallChat: %d
- TotalKeyword: %d
- TotalSearch: %d
- TotalUnknown: %d`,
		s.TotalSmallChatCnt, s.TotalKeywordCnt, s.TotalSearchCnt, s.TotalUnknownCnt)
}
