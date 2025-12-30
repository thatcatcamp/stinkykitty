// SPDX-License-Identifier: MIT
package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(5, time.Minute)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c.Request.RemoteAddr = "10.0.0.1:1234"

	middleware := RateLimitMiddleware(limiter, "/admin/login")
	middleware(c)

	if w.Code == 429 {
		t.Error("Expected request to be allowed")
	}
}

func TestRateLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(2, time.Minute)

	clientIP := "10.0.0.1:1234"

	// First request - allowed
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c1.Request.RemoteAddr = clientIP

	middleware := RateLimitMiddleware(limiter, "/admin/login")
	middleware(c1)

	if w1.Code == 429 {
		t.Error("First request should be allowed")
	}

	// Second request - allowed
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c2.Request.RemoteAddr = clientIP

	middleware(c2)

	if w2.Code == 429 {
		t.Error("Second request should be allowed")
	}

	// Third request - should be rate limited
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c3.Request.RemoteAddr = clientIP

	middleware(c3)

	if w3.Code != 429 {
		t.Errorf("Third request should be rate limited, got %d", w3.Code)
	}

	// Check headers
	if w3.Header().Get("X-RateLimit-Limit") != "2" {
		t.Errorf("Expected X-RateLimit-Limit: 2, got %s", w3.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimitDifferentPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(5, time.Minute)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
	c.Request.RemoteAddr = "10.0.0.1:1234"

	// Middleware only applies to /admin/login
	middleware := RateLimitMiddleware(limiter, "/admin/login")
	middleware(c)

	// Should not be rate limited (different path)
	if w.Code == 429 {
		t.Error("Different path should not be rate limited")
	}
}
