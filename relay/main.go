package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var flagAddr string
var flagRootPath string

func main() {
	var secret string
	secretB, err := os.ReadFile("/secret/token")
	if err != nil {
		fmt.Printf("WARN: failed to read secret: %v\n", err)
	} else {
		log.Println("using secret from file")
		secret = strings.TrimSpace(string(secretB))
	}

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

	// Chat endpoint for webchat-widget
	r.POST(rootPath+"/chat", relay.ChatHandler)

	// WebSocket endpoint for processor
	r.GET(rootPath+"/ws", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		expected := "Bearer " + secret
		if secret != "" && authHeader != expected {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		relay.handleWebSocket(c)
	})

	// Start server
	r.Run(flagAddr)
}
