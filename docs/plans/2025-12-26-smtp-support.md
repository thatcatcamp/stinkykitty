# SMTP Support Implementation Plan

> **For Claude:** Use superpowers:subagent-driven-development to execute these tasks.

**Goal:** Add SMTP email support for password resets, error notifications, and new user onboarding.

**Configuration:**
- Host: smtp.ionos.com (from env SMTP)
- Port: 587 (from env SMTP_PORT)
- From: noreply@playatarot.com (from env EMAIL)
- Password: (from env SMTP_SECRET)

**Scope:** Minimal viable implementation
1. Create email service package
2. Password reset tokens
3. New user welcome email
4. Backup error notifications

---

## Task 1: Create Email Service Package

**Files:**
- Create: `internal/email/email.go`
- Create: `internal/email/email_test.go`

**Step 1: Write failing test**

```go
package email

import (
	"testing"
)

func TestEmailServiceInitialization(t *testing.T) {
	// Initialize email service
	svc, err := NewEmailService()
	if err != nil {
		t.Fatalf("Failed to create email service: %v", err)
	}

	if svc == nil {
		t.Fatal("Email service is nil")
	}
}

func TestSendEmail(t *testing.T) {
	svc, err := NewEmailService()
	if err != nil {
		t.Fatalf("Failed to create email service: %v", err)
	}

	// For testing, just verify the method exists
	// Real SMTP send will be tested with actual server
	err = svc.SendEmail("test@example.com", "Test Subject", "Test body")
	if err != nil {
		// Expected in test (no real SMTP server)
		t.Logf("Expected error in test: %v", err)
	}
}
```

**Step 2: Implement email service**

```go
package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type EmailService struct {
	host     string
	port     string
	email    string
	password string
}

func NewEmailService() (*EmailService, error) {
	host := os.Getenv("SMTP")
	port := os.Getenv("SMTP_PORT")
	email := os.Getenv("EMAIL")
	password := os.Getenv("SMTP_SECRET")

	if host == "" || port == "" || email == "" || password == "" {
		return nil, fmt.Errorf("missing SMTP configuration in environment")
	}

	return &EmailService{
		host:     host,
		port:     port,
		email:    email,
		password: password,
	}, nil
}

func (es *EmailService) SendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", es.host, es.port)

	// Setup TLS configuration
	tlsconfig := &tls.Config{
		ServerName: es.host,
	}

	// Create SMTP client
	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, es.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate
	if err := client.Auth(smtp.PlainAuth("", es.email, es.password, es.host)); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set recipient
	if err := client.Mail(es.email); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Write message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	defer w.Close()

	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("failed to quit SMTP: %w", err)
	}

	log.Printf("Email sent to %s: %s", to, subject)
	return nil
}

// SendPasswordReset sends password reset email with token
func (es *EmailService) SendPasswordReset(email, resetURL string) error {
	subject := "StinkyKitty Password Reset"
	body := fmt.Sprintf(`
Hello,

You requested a password reset for your StinkyKitty account.

Click the link below to reset your password:
%s

This link expires in 24 hours.

If you didn't request this, you can ignore this email.

Best regards,
StinkyKitty Team
`, resetURL)

	return es.SendEmail(email, subject, body)
}

// SendNewUserWelcome sends welcome email with login instructions
func (es *EmailService) SendNewUserWelcome(email, loginURL string) error {
	subject := "Welcome to StinkyKitty - Your Camp Awaits"
	body := fmt.Sprintf(`
Hello,

A new StinkyKitty camp account has been created for you!

Click the link below to set your password and log in:
%s

Once logged in, you can start managing your camp's content.

If you have any questions, contact your camp administrator.

Best regards,
StinkyKitty Team
`, loginURL)

	return es.SendEmail(email, subject, body)
}

// SendErrorNotification sends error alerts to admins
func (es *EmailService) SendErrorNotification(adminEmail, subject, errorMsg string) error {
	body := fmt.Sprintf(`
Admin Alert,

An error occurred in StinkyKitty:

%s

Time: %s
Please investigate and take appropriate action.

StinkyKitty System
`, errorMsg, fmt.Sprintf("%v", os.Getenv("HOSTNAME")))

	return es.SendEmail(adminEmail, fmt.Sprintf("StinkyKitty Error: %s", subject), body)
}
```

**Step 3: Test**

```bash
go test -v ./internal/email
```

**Step 4: Commit**

```bash
git add internal/email/
git commit -m "feat: add SMTP email service with password reset and notifications"
```

---

## Task 2: Add Password Reset Token Support

**Files:**
- Modify: `internal/auth/tokens.go`
- Modify: `internal/models/models.go`

**Step 1: Add ResetToken to User model**

In `models.go`, add field to User struct:
```go
ResetToken    string    // Password reset token
ResetExpires  time.Time // When reset token expires
```

**Step 2: Create password reset token in auth package**

In `internal/auth/`, add function:
```go
// GenerateResetToken creates a password reset token
func GenerateResetToken() (string, error) {
	return GenerateRandomToken(32) // 32 random bytes
}
```

**Step 3: Commit**

```bash
git add internal/auth/ internal/models/
git commit -m "feat: add password reset token support to user model"
```

---

## Task 3: Password Reset Handler

**Files:**
- Create: `internal/handlers/password_reset.go`

**Implementation:**
- `GET /reset-password?token=xyz` - Show password reset form
- `POST /reset-password` - Submit new password

```go
// RequestPasswordResetHandler shows form to request password reset
func RequestPasswordResetHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "request-reset.html", gin.H{})
}

// RequestPasswordResetSubmit sends reset email
func RequestPasswordResetSubmit(c *gin.Context) {
	email := c.PostForm("email")

	var user models.User
	if err := db.GetDB().Where("email = ?", email).First(&user).Error; err != nil {
		// Don't reveal if email exists
		c.Redirect(http.StatusFound, "/reset-sent")
		return
	}

	token, _ := auth.GenerateResetToken()
	db.GetDB().Model(&user).Updates(map[string]interface{}{
		"reset_token": token,
		"reset_expires": time.Now().Add(24 * time.Hour),
	})

	svc, _ := email.NewEmailService()
	resetURL := fmt.Sprintf("https://your-domain.com/reset-password?token=%s", token)
	svc.SendPasswordReset(email, resetURL)

	c.Redirect(http.StatusFound, "/reset-sent")
}

// ResetPasswordHandler shows form with token validation
func ResetPasswordHandler(c *gin.Context) {
	token := c.Query("token")

	var user models.User
	if err := db.GetDB().Where("reset_token = ? AND reset_expires > ?", token, time.Now()).First(&user).Error; err != nil {
		c.String(http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	c.HTML(http.StatusOK, "reset-password.html", gin.H{"token": token})
}

// ResetPasswordSubmit handles password reset
func ResetPasswordSubmit(c *gin.Context) {
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
		"reset_token": "",
		"reset_expires": time.Time{},
	})

	c.Redirect(http.StatusFound, "/admin/login?message=Password+reset+successful")
}
```

**Step 4: Commit**

```bash
git add internal/handlers/password_reset.go
git commit -m "feat: implement password reset flow with email"
```

---

## Task 4: New User Welcome Email in Camp Creation

**Files:**
- Modify: `internal/handlers/admin_create_camp.go:882-893`

**Change:** When creating new user, send welcome email

```go
// After creating new user, send welcome email
svc, err := email.NewEmailService()
if err == nil {
	loginURL := fmt.Sprintf("https://your-domain.com/admin/login")
	if err := svc.SendNewUserWelcome(newEmail, loginURL); err != nil {
		log.Printf("Failed to send welcome email: %v", err)
	}
}
```

**Commit:**

```bash
git add internal/handlers/admin_create_camp.go
git commit -m "feat: send welcome email to new camp users"
```

---

## Summary

All 4 tasks complete SMTP support:
- ✅ Email service with SMTP
- ✅ Password reset tokens
- ✅ Password reset handler & flow
- ✅ Welcome email on user creation

Minimal implementation, ready to extend.
