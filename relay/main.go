package main

import (
	"flag"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var flagAddr string

func main() {
	flag.StringVar(&flagAddr, "addr", ":8080", "address to listen on")
	flag.Parse()

	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	relay := NewRelay()

	// Chat endpoint
	r.POST("/chat", relay.ChatHandler)

	// WebSocket 엔드포인트 설정
	r.GET("/ws", relay.handleWebSocket)

	// Start server
	r.Run(flagAddr)
}
