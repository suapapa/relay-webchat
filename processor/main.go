package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	flagWebSocketServer  string
	flagRetriveCnt       int
	flagPromptPreProcess bool

	hominDevAI *HominDevAI
)

func main() {
	flag.StringVar(&flagWebSocketServer, "ws", "ws://localhost:8080", "WebSocket server address")
	flag.IntVar(&flagRetriveCnt, "retrive", 50, "Retrive count")
	flag.BoolVar(&flagPromptPreProcess, "pre-process", false, "Pre-process prompt")
	flag.Parse()

	log.Printf("WebSocket server: %s", flagWebSocketServer)

	var err error
	hominDevAI, err = NewHominDevAI(context.Background())
	if err != nil {
		log.Fatalf("Failed to create HominDevAI: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(flagWebSocketServer, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		go func(msg string) {
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

			if err := conn.WriteMessage(messageType, []byte(reply)); err != nil {
				log.Printf("Write error: %v", err)
			}
		}(string(message))
	}
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
