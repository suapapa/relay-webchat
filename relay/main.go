package main

import (
	"cmp"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var (
	flagAddr           string
	flagRootPath       string
	flagAllowedOrigins string
	flagRateLimitRPM   int
	flagRateLimitBurst int
)

func getEnvInt(key string, fallback int) int {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}

func setupCORS(rawOrigins string) gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept"}

	rawOrigins = strings.TrimSpace(rawOrigins)
	if rawOrigins == "*" {
		config.AllowAllOrigins = true
		log.Println("CORS: allowing all origins (*)")
		return cors.New(config)
	}

	var origins []string
	if rawOrigins != "" {
		for _, o := range strings.Split(rawOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				origins = append(origins, o)
			}
		}
	}

	if len(origins) == 0 {
		origins = []string{
			"https://homin.dev",
			"https://www.homin.dev",
			"http://localhost:5173",
			"http://localhost:3000",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:3000",
		}
	}

	config.AllowOrigins = origins
	log.Printf("CORS: allowed origins: %v\n", origins)
	return cors.New(config)
}

func main() {
	// Set gin to release mode
	gin.SetMode(gin.ReleaseMode)

	secret := strings.TrimSpace(os.Getenv("RELAY_APIKEY"))
	if secret != "" {
		log.Println("using RELAY_APIKEY from environment")
	}

	flag.StringVar(&flagAddr, "addr", cmp.Or(os.Getenv("ADDR"), ":8080"), "address to listen on")
	flag.StringVar(&flagRootPath, "root", cmp.Or(os.Getenv("ROOT_PATH"), "/"), "root path of webchat-widget")
	flag.StringVar(&flagAllowedOrigins, "origins", os.Getenv("ALLOWED_ORIGINS"), "comma-separated allowed CORS origins (* for all)")
	flag.IntVar(&flagRateLimitRPM, "rate-limit-rpm", getEnvInt("RATE_LIMIT_RPM", 10), "rate limit requests per minute per IP (0 to disable)")
	flag.IntVar(&flagRateLimitBurst, "rate-limit-burst", getEnvInt("RATE_LIMIT_BURST", 5), "rate limit burst capacity per IP")
	flag.Parse()

	r := gin.Default()

	// Configure trusted proxies
	// In most Kubernetes setups, requests may come from various internal IPs (e.g., cluster ingress, service mesh).
	// To avoid proxy issues, it's common to trust all proxies, but be aware of security implications.
	// For stricter security, configure with known proxy IPs or CIDRs.
	r.SetTrustedProxies(nil) // Trust all proxies (suitable for most Kubernetes clusters)

	// Configure CORS
	r.Use(setupCORS(flagAllowedOrigins))

	relay := NewRelay()

	rootPath := flagRootPath
	if rootPath == "/" {
		rootPath = ""
	} else {
		log.Println("root path: ", rootPath)
	}

	var chatHandlers []gin.HandlerFunc
	if flagRateLimitRPM > 0 {
		limiter := NewIPRateLimiter(rate.Limit(float64(flagRateLimitRPM)/60.0), flagRateLimitBurst, 10*time.Minute)
		defer limiter.Close()
		chatHandlers = append(chatHandlers, RateLimitMiddleware(limiter))
		log.Printf("Rate limit enabled for /chat: %d req/min (burst %d)\n", flagRateLimitRPM, flagRateLimitBurst)
	} else {
		log.Println("Rate limit disabled for /chat")
	}
	chatHandlers = append(chatHandlers, relay.ChatHandler)

	// Chat endpoint for webchat-widget
	r.POST(rootPath+"/chat", chatHandlers...)

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
