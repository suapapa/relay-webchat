package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	flagWebSocketServer  string
	flagRetriveCnt       int
	flagPromptPreProcess bool
	flagEmbedderType     string

	hominDevAI *HominDevAI
)

func main() {
	flag.StringVar(&flagWebSocketServer, "ws", "ws://localhost:8080", "WebSocket server address")
	flag.IntVar(&flagRetriveCnt, "retrive", 50, "Retrive count")
	flag.BoolVar(&flagPromptPreProcess, "pre-process", false, "Pre-process prompt")
	flag.StringVar(&flagEmbedderType, "embedder", "ollama", "Embedder type (ollama, openai)")
	flag.Parse()

	log.Printf("WebSocket server: %s", flagWebSocketServer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	hominDevAI, err = NewHominDevAI(ctx)
	if err != nil {
		log.Fatalf("Failed to create HominDevAI: %v", err)
	}

	var conn *websocket.Conn
	connectWS := func() error {
		var err error
		conn, _, err = websocket.DefaultDialer.Dial(flagWebSocketServer, nil)
		if err != nil {
			return fmt.Errorf("failed to connect to WebSocket server: %v", err)
		}
		return nil
	}

	if err := connectWS(); err != nil {
		log.Fatalf("%v", err)
	}
	defer conn.Close()

	// get ctrl-c
	chCtrlC := make(chan os.Signal, 1)
	signal.Notify(chCtrlC, os.Interrupt)

	readErrRetryCnt, writeErrRetryCnt := 0, 0
	for {
		select {
		case <-chCtrlC:
			log.Println("Ctrl-C pressed, exiting...")
			return
		case <-ctx.Done():
			log.Println("Context canceled, exiting...")
			return
		default:
			// Set a read deadline of 5 seconds
			// if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			// 	log.Printf("Failed to set read deadline: %v", err)
			// 	continue
			// }
			msgType, msgBytes, err := conn.ReadMessage()
			if err != nil {
				// 타임아웃 에러 처리
				if os.IsTimeout(err) || websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					// Try to reconnect
					if err := connectWS(); err != nil {
						log.Fatalf("Failed to reconnect: %v", err)
					}
					// log.Println("Successfully reconnected")
					continue
				}

				log.Printf("Connection error: %v", err)
				time.Sleep(1 * time.Second)
				readErrRetryCnt++
				if readErrRetryCnt > 3 {
					log.Fatalf("Failed to reconnect after 3 attempts: %v", err)
				}
			}
			readErrRetryCnt = 0

			msg := string(msgBytes)
			var reply string
			if len([]rune(msg)) > 200 {
				log.Printf("Message is too long: %d", len([]rune(msg)))
				reply = "메시지가 너무 길어요. 200자 이하로 짧게 줄여주세요."
			} else {
				log.Printf("Received message: %s", msg)

				cmd, err := hominDevAI.PreProcessFLow.Run(context.Background(), msg)
				if err != nil {
					log.Printf("Failed to run intent flow: %v", err)
					return
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
					reply = fmt.Sprintf("%s 에 대한 검색 결과:\n%s", searchKeywords, makePostReply(posts))
				case "/search":
					posts, err := retrivePost(msg, flagRetriveCnt)
					if err != nil {
						log.Printf("Failed to retrive post for msg, %s: %v", msg, err)
						return
					}
					reply = "검색 결과:\n" + makePostReply(posts)
				case "/smallchat":
					reply = strings.Join(cmd.Args, "\n")
				default:
					log.Printf("Unknown command: %s", cmd.Action)
					reply = makeAboutReply()
				}
			}

			// if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			// 	log.Printf("Failed to set read deadline: %v", err)
			// 	continue
			// }

			if err := conn.WriteMessage(msgType, []byte(reply)); err != nil {
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
	}

	log.Println("Exiting...")
}

func makePostReply(posts []*Post) string {
	// reply := "검색 결과:\n"
	var reply string
	for _, post := range posts {
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
