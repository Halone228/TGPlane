package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// limiterEntry holds a token bucket and a last-seen timestamp for cleanup.
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter returns a per-key token-bucket middleware.
// keyFunc extracts the rate-limit key from the request (e.g. IP or API key).
// rps is the sustained request rate; burst is the maximum burst size.
func RateLimiter(rps float64, burst int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	var mu sync.Mutex
	limiters := make(map[string]*limiterEntry)

	// Background goroutine cleans up entries idle for more than 5 minutes.
	go func() {
		for range time.Tick(time.Minute) {
			mu.Lock()
			for k, e := range limiters {
				if time.Since(e.lastSeen) > 5*time.Minute {
					delete(limiters, k)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		e, ok := limiters[key]
		if !ok {
			e = &limiterEntry{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
			limiters[key] = e
		}
		e.lastSeen = time.Now()
		return e.limiter
	}

	return func(c *gin.Context) {
		key := keyFunc(c)
		if !getLimiter(key).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}

// IPRateLimiter limits by client IP.
func IPRateLimiter(rps float64, burst int) gin.HandlerFunc {
	return RateLimiter(rps, burst, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// KeyRateLimiter limits by API key (falls back to IP if no key present).
func KeyRateLimiter(rps float64, burst int) gin.HandlerFunc {
	return RateLimiter(rps, burst, func(c *gin.Context) string {
		if key := c.GetHeader("X-Api-Key"); key != "" {
			return "key:" + key
		}
		return "ip:" + c.ClientIP()
	})
}
