// SPDX-License-Identifier: MIT
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HTTPSRedirectMiddleware redirects HTTP requests to HTTPS
// Exceptions: ACME challenges (/.well-known/acme-challenge/)
func HTTPSRedirectMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if already HTTPS
		if c.Request.TLS != nil {
			c.Next()
			return
		}

		// Skip for ACME challenges
		if strings.HasPrefix(c.Request.URL.Path, "/.well-known/acme-challenge/") {
			c.Next()
			return
		}

		// Redirect to HTTPS
		httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
		c.Redirect(http.StatusMovedPermanently, httpsURL)
		c.Abort()
	}
}
