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
	flagFindIntent      bool

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
				posts, err := retrivePost(msg, flagRetriveCnt)
				if err != nil {
					log.Printf("Failed to retrive post: %v", err)
					return
				}
				for _, post := range posts {
					reply += fmt.Sprintf("- [%s](%s)\n", post.Title, post.Url)
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
