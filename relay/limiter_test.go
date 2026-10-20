package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestIPRateLimiter_Allow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Burst = 2, rate = 1 token/sec
	limiter := NewIPRateLimiter(rate.Limit(1), 2, 100*time.Millisecond)
	defer limiter.Close()

	ip1 := "192.168.1.10"
	ip2 := "192.168.1.20"

	// IP1: first 2 requests should be allowed (burst = 2)
	if !limiter.Allow(ip1) {
		t.Errorf("expected ip1 request 1 to be allowed")
	}
	if !limiter.Allow(ip1) {
		t.Errorf("expected ip1 request 2 to be allowed")
	}
	// IP1: 3rd request immediately should be blocked
	if limiter.Allow(ip1) {
		t.Errorf("expected ip1 request 3 to be rate-limited")
	}

	// IP2: should still be allowed (separate bucket)
	if !limiter.Allow(ip2) {
		t.Errorf("expected ip2 request 1 to be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewIPRateLimiter(rate.Limit(0.1), 1, time.Minute)
	defer limiter.Close()

	r := gin.New()
	r.POST("/chat", RateLimitMiddleware(limiter), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"reply": "ok"})
	})

	// 1st request: should succeed (200 OK)
	req1, _ := http.NewRequest(http.MethodPost, "/chat", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// 2nd request immediately: should get 429 Too Many Requests
	req2, _ := http.NewRequest(http.MethodPost, "/chat", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w2.Code)
	}
	if w2.Header().Get("Retry-After") != "60" {
		t.Errorf("expected Retry-After header to be 60, got %s", w2.Header().Get("Retry-After"))
	}
}

func TestSetupCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(setupCORS("https://homin.dev,http://localhost:3000"))
	r.POST("/chat", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"reply": "ok"})
	})

	// OPTIONS preflight from allowed origin
	reqPreflight, _ := http.NewRequest(http.MethodOptions, "/chat", nil)
	reqPreflight.Header.Set("Origin", "https://homin.dev")
	reqPreflight.Header.Set("Access-Control-Request-Method", "POST")
	wPreflight := httptest.NewRecorder()
	r.ServeHTTP(wPreflight, reqPreflight)
	if wPreflight.Code != http.StatusNoContent && wPreflight.Code != http.StatusOK {
		t.Errorf("expected 204 or 200 for preflight, got %d", wPreflight.Code)
	}
	if wPreflight.Header().Get("Access-Control-Allow-Origin") != "https://homin.dev" {
		t.Errorf("expected Access-Control-Allow-Origin header, got %s", wPreflight.Header().Get("Access-Control-Allow-Origin"))
	}

	// OPTIONS preflight from unauthorized origin
	reqDisallowed, _ := http.NewRequest(http.MethodOptions, "/chat", nil)
	reqDisallowed.Header.Set("Origin", "https://evil.example.com")
	reqDisallowed.Header.Set("Access-Control-Request-Method", "POST")
	wDisallowed := httptest.NewRecorder()
	r.ServeHTTP(wDisallowed, reqDisallowed)
	if wDisallowed.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disallowed origin preflight, got %d", wDisallowed.Code)
	}
}

func TestChatHandler_NoProcessorReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)

	relay := NewRelay()
	r := gin.New()
	r.POST("/chat", relay.ChatHandler)

	req, _ := http.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"message":"/start"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "No processor connection") {
		t.Errorf("expected body to contain 'No processor connection', got %s", w.Body.String())
	}
}
