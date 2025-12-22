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

// LoginFormHandler shows the login form
func LoginFormHandler(c *gin.Context) {
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Admin Login - ` + site.Subdomain + `</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 400px; margin: 50px auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { margin: 0 0 20px 0; font-size: 24px; color: #333; }
        .site-name { color: #666; font-size: 14px; margin-bottom: 20px; }
        input { width: 100%; padding: 12px; margin: 8px 0; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; font-size: 14px; }
        button { width: 100%; padding: 12px; background: #007bff; color: white; border: none; border-radius: 4px; font-size: 16px; cursor: pointer; margin-top: 10px; }
        button:hover { background: #0056b3; }
        .error { color: #dc3545; margin-top: 10px; font-size: 14px; }
        label { font-size: 14px; color: #555; display: block; margin-top: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Admin Login</h1>
        <div class="site-name">` + site.Subdomain + `</div>
        <form method="POST" action="/admin/login">
            <label>Email</label>
            <input type="email" name="email" required autofocus>
            <label>Password</label>
            <input type="password" name="password" required>
            <button type="submit">Log In</button>
        </form>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
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
