package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/email"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func RequestPasswordResetHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Reset Password - StinkyKitty</title>
	<style>` + GetDesignSystemCSS() + `</style>
</head>
<body>
	<div class="container" style="max-width: 400px; margin: 50px auto;">
		<h1>Reset Password</h1>
		<form method="POST" action="/reset-password">
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
	c.HTML(http.StatusOK, "", html)
}

func RequestPasswordResetSubmitHandler(c *gin.Context) {
	emailAddr := c.PostForm("email")

	var user models.User
	if err := db.GetDB().Where("email = ?", emailAddr).First(&user).Error; err != nil {
		c.Redirect(http.StatusFound, "/reset-sent")
		return
	}

	token, _ := auth.GenerateResetToken()
	db.GetDB().Model(&user).Updates(map[string]interface{}{
		"reset_token":   token,
		"reset_expires": time.Now().Add(24 * time.Hour),
	})

	svc, err := email.NewEmailService()
	if err == nil {
		resetURL := "https://campasaur.us/reset-confirm?token=" + token
		svc.SendPasswordReset(emailAddr, resetURL)
	}

	c.Redirect(http.StatusFound, "/reset-sent")
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
	c.HTML(http.StatusOK, "", html)
}

func ResetConfirmHandler(c *gin.Context) {
	token := c.Query("token")

	var user models.User
	if err := db.GetDB().Where("reset_token = ? AND reset_expires > ?", token, time.Now()).First(&user).Error; err != nil {
		c.String(http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Set New Password - StinkyKitty</title>
	<style>%s</style>
</head>
<body>
	<div class="container" style="max-width: 400px; margin: 50px auto;">
		<h1>Set New Password</h1>
		<form method="POST" action="/reset-confirm">
			<input type="hidden" name="token" value="%s">
			<div style="margin: 20px 0;">
				<label>New Password:</label>
				<input type="password" name="password" required minlength="8" style="width: 100%%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
			</div>
			<button type="submit" style="background: #2563eb; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer;">Reset Password</button>
		</form>
	</div>
</body>
</html>`, GetDesignSystemCSS(), token)
	c.HTML(http.StatusOK, "", html)
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

	c.Redirect(http.StatusFound, "/admin/login?message=Password+reset+successful")
}
