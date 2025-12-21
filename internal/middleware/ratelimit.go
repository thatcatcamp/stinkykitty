package middleware

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens    int
	capacity  int
	refillAt  time.Time
	interval  time.Duration
	mu        sync.Mutex
}

// RateLimiter manages token buckets per IP
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket
	capacity int
	interval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(capacity int, interval time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		buckets:  make(map[string]*TokenBucket),
		capacity: capacity,
		interval: interval,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// cleanup removes old buckets every 5 minutes
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.buckets {
			bucket.mu.Lock()
			// Remove buckets not accessed in 10 minutes
			if now.Sub(bucket.refillAt) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ip string) (bool, int) {
	rl.mu.RLock()
	bucket, exists := rl.buckets[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		bucket = &TokenBucket{
			tokens:   rl.capacity,
			capacity: rl.capacity,
			refillAt: time.Now().Add(rl.interval),
			interval: rl.interval,
		}
		rl.buckets[ip] = bucket
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens if interval has passed
	now := time.Now()
	if now.After(bucket.refillAt) {
		bucket.tokens = bucket.capacity
		bucket.refillAt = now.Add(bucket.interval)
	}

	// Try to consume a token
	if bucket.tokens > 0 {
		bucket.tokens--
		return true, bucket.tokens
	}

	return false, 0
}

// RateLimitMiddleware creates a rate limiting middleware for specific paths
func RateLimitMiddleware(limiter *RateLimiter, paths ...string) gin.HandlerFunc {
	pathMap := make(map[string]bool)
	for _, path := range paths {
		pathMap[path] = true
	}

	return func(c *gin.Context) {
		// Check if this path requires rate limiting
		if !pathMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Extract client IP
		clientIP := getClientIP(c)

		// Check rate limit
		allowed, remaining := limiter.Allow(clientIP)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.capacity))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			// Calculate retry-after (seconds until next refill)
			retryAfter := int(limiter.interval.Seconds())
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatus(429)
			return
		}

		c.Next()
	}
}

// getClientIP extracts the client IP address
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
