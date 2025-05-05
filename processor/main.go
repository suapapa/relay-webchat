package main

import (
	"flag"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var flagWebSocketServer string

func main() {
	flag.StringVar(&flagWebSocketServer, "ws", "ws://localhost:8080", "WebSocket server address")
	flag.Parse()

	log.Printf("WebSocket server: %s", flagWebSocketServer)

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
			reply := "[대문](https://homin.dev)으로 돌아가기\n- item1\n- item2\n- item3"
			if err := conn.WriteMessage(messageType, []byte(reply)); err != nil {
				log.Printf("Write error: %v", err)
			}
		}(string(message))
	}
}
