package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// LoginHandler handles admin login requests
func LoginHandler(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Find user by email
	var user models.User
	if err := db.GetDB().Where("email = ?", email).First(&user).Error; err != nil {
		c.String(http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Verify password
	if !auth.CheckPassword(password, user.PasswordHash) {
		c.String(http.StatusUnauthorized, "Invalid email or password")
		return
	}

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
		c.String(http.StatusForbidden, "You don't have access to this site")
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(&user, site)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token: %v", err)
		return
	}

	// Set SameSite attribute before setting cookie
	c.SetSameSite(http.SameSiteLaxMode)

	// Set HTTP-only cookie
	c.SetCookie(
		"stinky_token",        // name
		token,                 // value
		28800,                 // max age (8 hours in seconds)
		"/",                   // path
		"",                    // domain (empty = current domain)
		false,                 // secure (set to true in production with HTTPS)
		true,                  // httpOnly
	)

	// Redirect to dashboard
	c.Redirect(http.StatusFound, "/admin/dashboard")
}

// LogoutHandler handles admin logout requests
func LogoutHandler(c *gin.Context) {
	// Clear cookie
	c.SetCookie(
		"stinky_token",
		"",
		-1,     // max age -1 deletes the cookie
		"/",
		"",
		false,
		true,
	)

	// Redirect to login
	c.Redirect(http.StatusFound, "/admin/login")
}

// DashboardHandler renders the admin dashboard
func DashboardHandler(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userVal, exists := c.Get("user")
	if !exists {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}
	user := userVal.(*models.User)

	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	c.String(http.StatusOK, "Admin Dashboard\n\nUser: %s\nSite: %s\nGlobal Admin: %v",
		user.Email, site.Subdomain, user.IsGlobalAdmin)
}
