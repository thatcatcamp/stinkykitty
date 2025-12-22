package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// RequireAuth middleware validates JWT token and checks site access
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from cookie
		cookie, err := c.Cookie("stinky_token")
		if err != nil || cookie == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := ValidateToken(cookie)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Load user from database
		var user models.User
		if err := db.GetDB().First(&user, claims.UserID).Error; err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Get site from context (set by site resolution middleware)
		siteVal, exists := c.Get("site")
		if !exists {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		site := siteVal.(*models.Site)

		// Check if user has access to this site
		hasAccess := false

		// Global admins can access any site
		if user.IsGlobalAdmin {
			hasAccess = true
		}

		// Site owner can access
		if site.OwnerID == user.ID {
			hasAccess = true
		}

		// Check if user is in SiteUsers (member of site)
		if !hasAccess {
			var siteUser models.SiteUser
			err := db.GetDB().Where("site_id = ? AND user_id = ?", site.ID, user.ID).First(&siteUser).Error
			if err == nil {
				hasAccess = true
			}
		}

		if !hasAccess {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You don't have access to this site"})
			return
		}

		// Set user in context for handlers
		c.Set("user", &user)
		c.Next()
	}
}

// RequireGlobalAdmin middleware requires global admin privileges
func RequireGlobalAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First run RequireAuth
		RequireAuth()(c)

		if c.IsAborted() {
			return
		}

		// Check if user is global admin
		userVal, exists := c.Get("user")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		user := userVal.(*models.User)
		if !user.IsGlobalAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Global administrator access required"})
			return
		}

		c.Next()
	}
}
