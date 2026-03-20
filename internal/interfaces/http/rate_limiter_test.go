package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(5, 1*time.Minute)
	router := gin.New()
	router.GET("/test", rl.Handler(), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Minute)
	router := gin.New()
	router.GET("/test", rl.Handler(), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		router.ServeHTTP(w, req)
	}

	// 4th request should be blocked
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Minute)
	router := gin.New()
	router.GET("/test", rl.Handler(), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// Exhaust IP A
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.1.1.1:1234"
		router.ServeHTTP(w, req)
	}

	// IP B should still work
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "2.2.2.2:1234"
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("different IP should not be blocked, got %d", w.Code)
	}

	// IP A should be blocked
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "1.1.1.1:1234"
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("exhausted IP should be blocked, got %d", w2.Code)
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	router := gin.New()
	router.GET("/test", rl.Handler(), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// Exhaust limit
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		router.ServeHTTP(w, req)
	}

	// Should be blocked
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should work again
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Errorf("after window reset expected 200, got %d", w2.Code)
	}
}
