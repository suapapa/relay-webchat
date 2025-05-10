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
	// Set gin to release mode
	gin.SetMode(gin.ReleaseMode)

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

	// Configure trusted proxies
	// In most Kubernetes setups, requests may come from various internal IPs (e.g., cluster ingress, service mesh).
	// To avoid proxy issues, it's common to trust all proxies, but be aware of security implications.
	// For stricter security, configure with known proxy IPs or CIDRs.
	r.SetTrustedProxies(nil) // Trust all proxies (suitable for most Kubernetes clusters)

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
