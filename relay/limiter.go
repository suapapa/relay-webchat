package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter tracks token-bucket rate limiters per IP address.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	limit   rate.Limit
	burst   int
	ttl     time.Duration
	stopCh  chan struct{}
}

// NewIPRateLimiter creates a new rate limiter with the given limit, burst, and eviction TTL.
func NewIPRateLimiter(r rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		limit:   r,
		burst:   burst,
		ttl:     ttl,
		stopCh:  make(chan struct{}),
	}

	go limiter.cleanupLoop(ttl)

	return limiter
}

func (i *IPRateLimiter) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			i.cleanup()
		case <-i.stopCh:
			return
		}
	}
}

func (i *IPRateLimiter) cleanup() {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()
	for ip, client := range i.clients {
		if now.Sub(client.lastSeen) > i.ttl {
			delete(i.clients, ip)
		}
	}
}

// Close stops the background cleanup goroutine.
func (i *IPRateLimiter) Close() {
	select {
	case <-i.stopCh:
	default:
		close(i.stopCh)
	}
}

// Allow reports whether a request from ip should be allowed.
func (i *IPRateLimiter) Allow(ip string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	client, exists := i.clients[ip]
	now := time.Now()
	if !exists {
		client = &clientLimiter{
			limiter:  rate.NewLimiter(i.limit, i.burst),
			lastSeen: now,
		}
		i.clients[ip] = client
	} else {
		client.lastSeen = now
	}

	return client.limiter.Allow()
}

// RateLimitMiddleware returns a Gin middleware that enforces IP-based rate limiting.
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"reply": "요청이 너무 많습니다. 잠시 후 다시 시도해 주세요.",
				"error": "too many requests",
			})
			return
		}
		c.Next()
	}
}
