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
	flagWebSocketServer string
	flagRetriveCnt      int

	hominDevAI *HominDevAI
)

func main() {
	flag.StringVar(&flagWebSocketServer, "ws", "ws://localhost:8080", "WebSocket server address")
	flag.IntVar(&flagRetriveCnt, "retrive", 50, "Retrive count")
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
			log.Printf("Received message: %s", msg)

			cmd, err := hominDevAI.IntentFLow.Run(context.Background(), msg)
			if err != nil {
				log.Printf("Failed to run intent flow: %v", err)
				return
			}

			var reply string
			switch cmd.Action {
			case "/search":
				result, err := hominDevAI.SearchPostFlow.Run(context.Background(), msg)
				if err != nil {
					log.Printf("Failed to run search post flow: %v", err)
					return
				}

				if len(result.Posts) == 0 {
					reply = "검색 결과가 없어요."
					break
				}

				// 중복제거. Url 이 같으면 하나만 출력
				seen := make(map[string]bool)
				for _, post := range result.Posts {
					if !seen[post.Url] {
						seen[post.Url] = true
						reply += fmt.Sprintf("- [%s](%s)\n", post.Title, post.Url)
					}
				}

				reply = "검색 결과:\n" + strings.TrimRight(reply, "\n")
			case "/smallchat":
				reply = strings.Join(cmd.Args, "\n")
			default:
				log.Printf("Unknown command: %s", cmd.Action)
				reply = "ABOUT - TBU"
			}

			if err := conn.WriteMessage(messageType, []byte(reply)); err != nil {
				log.Printf("Write error: %v", err)
			}
		}(string(message))
	}
}
