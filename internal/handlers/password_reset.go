package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/email"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func RequestPasswordResetHandler(c *gin.Context) {
	csrfToken := middleware.GetCSRFTokenHTML(c)
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Reset Password - StinkyKitty</title>
	<style>` + GetDesignSystemCSS() + `</style>
</head>
<body>
	<div class="container" style="max-width: 400px; margin: 50px auto;">
		<h1>Reset Password</h1>
		<form method="POST" action="/admin/reset-password">
			` + csrfToken + `
			<div style="margin: 20px 0;">
				<label>Email Address:</label>
				<input type="email" name="email" required style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
			</div>
			<button type="submit" style="background: #2563eb; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer;">Send Reset Link</button>
		</form>
		<p style="margin-top: 20px;"><a href="/admin/login">Back to Login</a></p>
	</div>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func RequestPasswordResetSubmitHandler(c *gin.Context) {
	emailAddr := c.PostForm("email")

	var user models.User
	if err := db.GetDB().Where("email = ?", emailAddr).First(&user).Error; err != nil {
		c.Redirect(http.StatusFound, "/admin/reset-sent")
		return
	}

	token, _ := auth.GenerateResetToken()
	db.GetDB().Model(&user).Updates(map[string]interface{}{
		"reset_token":   token,
		"reset_expires": time.Now().Add(24 * time.Hour),
	})

	svc, err := email.NewEmailService()
	if err == nil {
		baseDomain := config.GetString("server.base_domain")
		if baseDomain == "" {
			baseDomain = "campasaur.us"
		}
		resetURL := fmt.Sprintf("https://%s/admin/reset-confirm?token=%s", baseDomain, token)
		svc.SendPasswordReset(emailAddr, resetURL)
	}

	c.Redirect(http.StatusFound, "/admin/reset-sent")
}

func ResetSentHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Check Your Email - StinkyKitty</title>
	<style>` + GetDesignSystemCSS() + `</style>
</head>
<body>
	<div class="container" style="max-width: 600px; margin: 50px auto; text-align: center;">
		<h1>Check Your Email</h1>
		<p>If an account exists with that email, a password reset link has been sent.</p>
		<p><a href="/admin/login">Back to Login</a></p>
	</div>
</body>
</html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func ResetConfirmHandler(c *gin.Context) {
	token := c.Query("token")

	var user models.User
	if err := db.GetDB().Where("reset_token = ? AND reset_expires > ?", token, time.Now()).First(&user).Error; err != nil {
		c.String(http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	csrfToken := middleware.GetCSRFTokenHTML(c)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Set New Password - StinkyKitty</title>
	<style>%s</style>
</head>
<body>
	<div class="container" style="max-width: 400px; margin: 50px auto;">
		<h1>Set New Password</h1>
		<form method="POST" action="/admin/reset-confirm">
			%s
			<input type="hidden" name="token" value="%s">
			<div style="margin: 20px 0;">
				<label>New Password:</label>
				<input type="password" name="password" required minlength="8" style="width: 100%%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
			</div>
			<button type="submit" style="background: #2563eb; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer;">Reset Password</button>
		</form>
	</div>
</body>
</html>`, GetDesignSystemCSS(), csrfToken, token)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func ResetConfirmSubmitHandler(c *gin.Context) {
	token := c.PostForm("token")
	password := c.PostForm("password")

	var user models.User
	if err := db.GetDB().Where("reset_token = ? AND reset_expires > ?", token, time.Now()).First(&user).Error; err != nil {
		c.String(http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	hash, _ := auth.HashPassword(password)
	db.GetDB().Model(&user).Updates(map[string]interface{}{
		"password_hash": hash,
		"reset_token":   "",
		"reset_expires": time.Time{},
	})

	// Get the sites the user has access to
	type UserSite struct {
		Subdomain    string
		CustomDomain *string
	}
	var userSites []UserSite

	// Get sites where user is owner or has site_users entry
	db.GetDB().Raw(`
		SELECT DISTINCT sites.subdomain, sites.custom_domain
		FROM sites
		LEFT JOIN site_users ON sites.id = site_users.site_id
		WHERE sites.owner_id = ? OR site_users.user_id = ?
		ORDER BY sites.subdomain
	`, user.ID, user.ID).Scan(&userSites)

	baseDomain := config.GetString("server.base_domain")
	if baseDomain == "" {
		baseDomain = "campasaur.us"
	}

	// Build site links
	var siteLinksHTML string
	if len(userSites) == 0 {
		siteLinksHTML = `<p>You don't have access to any camps yet. Contact an administrator.</p>`
	} else if len(userSites) == 1 {
		// Single site - redirect directly
		site := userSites[0]
		var domain string
		if site.CustomDomain != nil && *site.CustomDomain != "" {
			domain = *site.CustomDomain
		} else {
			domain = site.Subdomain + "." + baseDomain
		}
		loginURL := fmt.Sprintf("https://%s/admin/login", domain)
		c.Redirect(http.StatusFound, loginURL)
		return
	} else {
		// Multiple sites - show list
		siteLinksHTML = `<p>Choose your camp to log in:</p><ul style="list-style: none; padding: 0;">`
		for _, site := range userSites {
			var domain string
			if site.CustomDomain != nil && *site.CustomDomain != "" {
				domain = *site.CustomDomain
			} else {
				domain = site.Subdomain + "." + baseDomain
			}
			loginURL := fmt.Sprintf("https://%s/admin/login", domain)
			siteLinksHTML += fmt.Sprintf(`<li style="margin: 10px 0;"><a href="%s" style="color: #2563eb; text-decoration: none; font-weight: 500;">%s</a></li>`, loginURL, domain)
		}
		siteLinksHTML += `</ul>`
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Password Reset Successful - StinkyKitty</title>
	<style>%s</style>
</head>
<body>
	<div class="container" style="max-width: 600px; margin: 50px auto; text-align: center;">
		<h1>Password Reset Successful!</h1>
		<p style="color: #16a34a; font-weight: 500;">✓ Your password has been updated.</p>
		%s
	</div>
</body>
</html>`, GetDesignSystemCSS(), siteLinksHTML)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
