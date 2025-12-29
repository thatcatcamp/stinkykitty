package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"html"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfFormField  = "csrf_token"
	csrfTokenLen   = 32
)

// CSRFMiddleware provides Cross-Site Request Forgery protection
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate or retrieve CSRF token
		token, err := c.Cookie(csrfCookieName)
		if err != nil || token == "" {
			// Generate new token
			token, err = generateCSRFToken()
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			// Set cookie with CSRF token
			c.SetCookie(
				csrfCookieName,
				token,
				3600*8, // 8 hours
				"/",
				"",
				false, // secure - should be true in production
				false, // httpOnly - false so JavaScript can read it if needed
			)
		}

		// Store token in context for template access
		c.Set("csrf_token", token)

		// For state-changing operations, validate the token
		if c.Request.Method == "POST" || c.Request.Method == "PUT" ||
			c.Request.Method == "PATCH" || c.Request.Method == "DELETE" {

			// Get token from header or form
			clientToken := c.GetHeader(csrfHeaderName)
			if clientToken == "" {
				clientToken = c.PostForm(csrfFormField)
			}

			// Validate token
			if clientToken != token {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Invalid CSRF token",
				})
				return
			}
		}

		c.Next()
	}
}

// generateCSRFToken creates a cryptographically secure random token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, csrfTokenLen)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetCSRFToken retrieves the CSRF token from the context for use in templates
func GetCSRFToken(c *gin.Context) string {
	token, exists := c.Get("csrf_token")
	if !exists {
		return ""
	}
	return token.(string)
}

// GetCSRFTokenHTML returns an HTML hidden input field containing the CSRF token
// for the current request. This is a convenience function for embedding CSRF
// protection in HTML forms.
//
// Returns an empty string if no CSRF token is available in the context, which
// indicates the CSRF middleware was not applied to this route.
//
// Example usage:
//
//	formHTML := `<form method="POST">` + middleware.GetCSRFTokenHTML(c) + `...`
//
// The output is safe for direct inclusion in HTML as the token value is escaped.
func GetCSRFTokenHTML(c *gin.Context) string {
	token := GetCSRFToken(c)
	if token == "" {
		return ""
	}
	return `<input type="hidden" name="` + csrfFormField + `" value="` + html.EscapeString(token) + `">`
}
