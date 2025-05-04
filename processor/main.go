package main

import (
	"flag"
	"log"

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
			break
		}

		log.Printf("Received message: %s", string(message))
		if err := conn.WriteMessage(messageType, append([]byte("processor: "), message...)); err != nil {
			log.Printf("Write error: %v", err)
			break
		}
	}
}
