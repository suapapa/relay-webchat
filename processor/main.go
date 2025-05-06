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
					reply = makePostReply(posts)
				case "/search":
					posts, err := retrivePost(msg, flagRetriveCnt)
					if err != nil {
						log.Printf("Failed to retrive post for msg, %s: %v", msg, err)
						return
					}
					reply = makePostReply(posts)
				case "/smallchat":
					reply = strings.Join(cmd.Args, "\n")
				default:
					log.Printf("Unknown command: %s", cmd.Action)
					reply = "ABOUT - TBU"
				}
			}

			if err := conn.WriteMessage(messageType, []byte(reply)); err != nil {
				log.Printf("Write error: %v", err)
			}
		}(string(message))
	}
}

func makePostReply(posts []*Post) string {
	reply := "검색 결과:\n"
	for _, post := range posts {
		reply += fmt.Sprintf("- [%s](%s)\n", post.Title, post.Url)
	}
	return reply
}
