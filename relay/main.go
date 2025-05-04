package main

import (
	"flag"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var flagAddr string
var flagRootPath string

func main() {
	flag.StringVar(&flagAddr, "addr", ":8080", "address to listen on")
	flag.StringVar(&flagRootPath, "root", "/", "root path of webchat-widget")
	flag.Parse()

	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	relay := NewRelay()

	rootPath := flagRootPath
	if rootPath == "/" {
		rootPath = ""
	} else {
		log.Println("root path: ", rootPath)
	}

	// Chat endpoint
	r.POST(rootPath+"/chat", relay.ChatHandler)

	// WebSocket 엔드포인트 설정
	r.GET(rootPath+"/ws", relay.handleWebSocket)

	// Start server
	r.Run(flagAddr)
}
