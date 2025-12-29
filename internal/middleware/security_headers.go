package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/config"
)

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "SAMEORIGIN")

		// Enable XSS protection (for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Content Security Policy (CSP)
		// Allow inline styles and scripts for now (can be tightened later)
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://www.googletagmanager.com https://www.google-analytics.com; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"frame-src https://www.youtube.com https://player.vimeo.com; " +
			"connect-src 'self' https://www.google-analytics.com"
		c.Header("Content-Security-Policy", csp)

		// HTTP Strict Transport Security (HSTS) - only if TLS is enabled
		if config.GetBool("server.tls_enabled") {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
