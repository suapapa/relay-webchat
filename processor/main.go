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
	"sync"
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
	var connMutex sync.Mutex

	connectWS := func() error {
		connMutex.Lock()
		defer connMutex.Unlock()

		// Close existing connection if any
		if conn != nil {
			conn.Close()
			conn = nil
		}

		reqHeader := http.Header{}
		if secret != "" {
			reqHeader.Set("Authorization", fmt.Sprintf("Bearer %s", secret))
		}

		var err error
		conn, _, err = websocket.DefaultDialer.Dial(flagWebSocketServer, reqHeader)
		if err != nil {
			return fmt.Errorf("failed to connect to WebSocket server: %v", err)
		}

		// Set connection parameters for better reliability
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

		// Set up ping/pong handler for the new connection
		conn.SetPongHandler(func(appData string) error {
			log.Println("Received pong")
			// Reset read deadline on pong
			connMutex.Lock()
			if conn != nil {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			}
			connMutex.Unlock()
			return nil
		})

		return nil
	}

	if err := connectWS(); err != nil {
		log.Fatalf("%v", err)
	}
	defer func() {
		connMutex.Lock()
		if conn != nil {
			conn.Close()
		}
		connMutex.Unlock()
	}()

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
	stopReading := make(chan struct{})

	go func() {
		defer close(msgChan)
		for {
			select {
			case <-stopReading:
				return
			default:
			}

			connMutex.Lock()
			currentConn := conn
			connMutex.Unlock()

			if currentConn == nil {
				time.Sleep(1 * time.Second)
				continue
			}

			// Set read deadline for this read operation
			currentConn.SetReadDeadline(time.Now().Add(60 * time.Second))
			msgType, msgBytes, err := currentConn.ReadMessage()

			select {
			case msgChan <- struct {
				msgType int
				msg     []byte
				err     error
			}{msgType, msgBytes, err}:
			case <-stopReading:
				return
			}

			if err != nil {
				log.Printf("Read error: %v", err)

				// Check if it's a close error or network error
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("Connection closed normally, reconnecting...")
				} else {
					log.Printf("Connection error, attempting reconnection...")
				}

				// Attempt reconnection
				readErrRetryCnt++
				if readErrRetryCnt > 5 {
					log.Fatalf("Failed to reconnect after 5 attempts: %v", err)
				}

				// Wait before reconnecting
				time.Sleep(time.Duration(readErrRetryCnt) * time.Second)

				if err := connectWS(); err != nil {
					log.Printf("Failed to reconnect: %v", err)
					continue
				}

				log.Printf("Successfully reconnected")
				readErrRetryCnt = 0
			} else {
				readErrRetryCnt = 0
			}
		}
	}()

	// Pong handler is set up in connectWS function

	// Send periodic ping (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				connMutex.Lock()
				currentConn := conn
				connMutex.Unlock()

				if currentConn != nil {
					currentConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := currentConn.WriteMessage(websocket.PingMessage, nil); err != nil {
						log.Printf("Ping error: %v", err)
						// Don't return, keep trying
					}
				}
			case <-stopReading:
				return
			}
		}
	}()

	for {
		select {
		case <-chCtrlC:
			log.Println("Ctrl-C pressed, exiting...")
			// Signal stop to reading goroutine
			close(stopReading)
			// Close WebSocket connection gracefully
			connMutex.Lock()
			if conn != nil {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				conn.Close()
			}
			connMutex.Unlock()
			cancel()
			return
		case <-ctx.Done():
			log.Println("Context canceled, exiting...")
			close(stopReading)
			return
		case msgData, ok := <-msgChan:
			if !ok {
				log.Println("Message channel closed, exiting...")
				return
			}
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

			connMutex.Lock()
			currentConn := conn
			connMutex.Unlock()

			if currentConn != nil {
				currentConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := currentConn.WriteMessage(msgData.msgType, []byte(reply)); err != nil {
					log.Printf("Write error: %v", err)
					writeErrRetryCnt++
					if writeErrRetryCnt > 3 {
						log.Printf("Failed to write after 3 attempts, attempting reconnection: %v", err)
						// Trigger reconnection
						connMutex.Lock()
						if conn != nil {
							conn.Close()
							conn = nil
						}
						connMutex.Unlock()
						time.Sleep(1 * time.Second)
						if err := connectWS(); err != nil {
							log.Printf("Failed to reconnect after write error: %v", err)
						}
					}
				} else {
					writeErrRetryCnt = 0
				}
			} else {
				log.Printf("No connection available for writing")
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
	siteURL := `https://homin.dev/blog/post/20250507_rag_blog_search_webchat_bot/`

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
