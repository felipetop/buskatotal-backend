package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiterEntry struct {
	count    int
	windowAt time.Time
}

// RateLimiter limits requests per IP within a time window.
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateLimiterEntry
	maxReqs  int
	window   time.Duration
}

// NewRateLimiter creates a rate limiter that allows maxReqs per window per IP.
func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimiterEntry),
		maxReqs: maxReqs,
		window:  window,
	}
	// Cleanup old entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		entry, ok := rl.entries[ip]
		now := time.Now()

		if !ok || now.After(entry.windowAt) {
			rl.entries[ip] = &rateLimiterEntry{count: 1, windowAt: now.Add(rl.window)}
			rl.mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > rl.maxReqs {
			rl.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "muitas tentativas, aguarde alguns minutos"})
			c.Abort()
			return
		}

		rl.mu.Unlock()
		c.Next()
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for k, v := range rl.entries {
		if now.After(v.windowAt) {
			delete(rl.entries, k)
		}
	}
}
