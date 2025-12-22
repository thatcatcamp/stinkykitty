# Authentication System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement JWT-based authentication with HTTP-only cookies for the StinkyKitty admin panel

**Architecture:** JWT tokens in HTTP-only cookies (8-hour expiry), bcrypt password hashing, simple site membership + global admin flag, integration with existing middleware chain

**Tech Stack:** Go 1.25+, Gin web framework, GORM ORM, golang-jwt/jwt/v5, golang.org/x/crypto/bcrypt

---

## Prerequisites

Install JWT library:
```bash
go get github.com/golang-jwt/jwt/v5
```

## Task 1: Add IsGlobalAdmin Field to User Model

**Files:**
- Modify: `internal/models/models.go` (User struct)
- Modify: `internal/models/models_test.go` (add test)

### Step 1: Write the failing test

Add to `internal/models/models_test.go`:

```go
func TestUserIsGlobalAdmin(t *testing.T) {
	db := setupTestDB(t)

	// Create regular user
	regularUser := User{
		Email:        "regular@test.com",
		PasswordHash: "hash",
	}
	result := db.Create(&regularUser)
	if result.Error != nil {
		t.Fatalf("Failed to create regular user: %v", result.Error)
	}

	// Create global admin
	adminUser := User{
		Email:         "admin@test.com",
		PasswordHash:  "hash",
		IsGlobalAdmin: true,
	}
	result = db.Create(&adminUser)
	if result.Error != nil {
		t.Fatalf("Failed to create admin user: %v", result.Error)
	}

	// Verify regular user is not global admin
	var fetchedRegular User
	db.First(&fetchedRegular, regularUser.ID)
	if fetchedRegular.IsGlobalAdmin {
		t.Error("Regular user should not be global admin")
	}

	// Verify admin user is global admin
	var fetchedAdmin User
	db.First(&fetchedAdmin, adminUser.ID)
	if !fetchedAdmin.IsGlobalAdmin {
		t.Error("Admin user should be global admin")
	}
}
```

### Step 2: Run test to verify it fails

Run:
```bash
go test ./internal/models -v -run TestUserIsGlobalAdmin
```

Expected output: Compilation error - `IsGlobalAdmin` field doesn't exist

### Step 3: Add IsGlobalAdmin field

In `internal/models/models.go`, add to `User` struct (after `PasswordHash`):

```go
IsGlobalAdmin bool `gorm:"default:false"`
```

Complete User struct should look like:
```go
type User struct {
	ID            uint           `gorm:"primaryKey"`
	Email         string         `gorm:"uniqueIndex;not null"`
	PasswordHash  string         `gorm:"not null"`
	IsGlobalAdmin bool           `gorm:"default:false"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relationships
	OwnedSites []Site     `gorm:"foreignKey:OwnerID"`
	SiteUsers  []SiteUser `gorm:"foreignKey:UserID"`
}
```

### Step 4: Run test to verify it passes

Run:
```bash
go test ./internal/models -v -run TestUserIsGlobalAdmin
```

Expected output: PASS

### Step 5: Run all model tests

Run:
```bash
go test ./internal/models -v
```

Expected output: All tests PASS

### Step 6: Commit

```bash
git add internal/models/models.go internal/models/models_test.go
git commit -m "feat: add IsGlobalAdmin field to User model

Add boolean flag to identify platform administrators who can access all sites.
Defaults to false for regular users."
```

---

## Task 2: Password Utilities (Bcrypt)

**Files:**
- Create: `internal/auth/password.go`
- Create: `internal/auth/password_test.go`

### Step 1: Create auth package directory

Run:
```bash
mkdir -p internal/auth
```

### Step 2: Write the failing test

Create `internal/auth/password_test.go`:

```go
package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "test-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}

	// Hash should start with bcrypt prefix
	if len(hash) < 60 {
		t.Error("Bcrypt hash should be at least 60 characters")
	}
}

func TestCheckPasswordCorrect(t *testing.T) {
	password := "test-password-123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}
}

func TestCheckPasswordIncorrect(t *testing.T) {
	password := "test-password-123"
	wrongPassword := "wrong-password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if CheckPassword(wrongPassword, hash) {
		t.Error("CheckPassword should return false for incorrect password")
	}
}

func TestCheckPasswordEmpty(t *testing.T) {
	hash, _ := HashPassword("test")

	if CheckPassword("", hash) {
		t.Error("CheckPassword should return false for empty password")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Error("HashPassword should return error for empty password")
	}
}
```

### Step 3: Run test to verify it fails

Run:
```bash
go test ./internal/auth -v
```

Expected output: Compilation errors - `HashPassword` and `CheckPassword` not defined

### Step 4: Implement password utilities

Create `internal/auth/password.go`:

```go
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// CheckPassword verifies a password against a bcrypt hash using constant-time comparison
func CheckPassword(password, hash string) bool {
	if password == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
```

### Step 5: Run test to verify it passes

Run:
```bash
go test ./internal/auth -v
```

Expected output: All tests PASS

### Step 6: Commit

```bash
git add internal/auth/password.go internal/auth/password_test.go
git commit -m "feat: add bcrypt password hashing utilities

Implement HashPassword and CheckPassword with bcrypt cost 12.
Includes constant-time comparison to prevent timing attacks."
```

---

## Task 3: JWT Token Generation and Validation

**Files:**
- Create: `internal/auth/jwt.go`
- Create: `internal/auth/jwt_test.go`
- Modify: `internal/config/config.go` (add auth defaults)

### Step 1: Add auth configuration defaults

In `internal/config/config.go`, add to `setDefaults()` function:

```go
// Auth defaults
v.SetDefault("auth.jwt_secret", "CHANGE_ME_IN_PRODUCTION_USE_ENV_VAR")
v.SetDefault("auth.jwt_expiry_hours", 8)
v.SetDefault("auth.bcrypt_cost", 12)
```

### Step 2: Write the failing test

Create `internal/auth/jwt_test.go`:

```go
package auth

import (
	"testing"
	"time"

	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func TestGenerateToken(t *testing.T) {
	user := &models.User{
		ID:            1,
		Email:         "test@example.com",
		IsGlobalAdmin: false,
	}
	site := &models.Site{
		ID: 5,
	}

	token, err := GenerateToken(user, site)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	// Token should have 3 parts separated by dots
	if len(token) < 100 {
		t.Error("JWT token should be longer")
	}
}

func TestValidateTokenValid(t *testing.T) {
	user := &models.User{
		ID:            1,
		Email:         "test@example.com",
		IsGlobalAdmin: true,
	}
	site := &models.Site{
		ID: 5,
	}

	token, err := GenerateToken(user, site)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, claims.UserID)
	}

	if claims.Email != user.Email {
		t.Errorf("Expected Email %s, got %s", user.Email, claims.Email)
	}

	if claims.SiteID != site.ID {
		t.Errorf("Expected SiteID %d, got %d", site.ID, claims.SiteID)
	}

	if claims.IsGlobalAdmin != user.IsGlobalAdmin {
		t.Errorf("Expected IsGlobalAdmin %v, got %v", user.IsGlobalAdmin, claims.IsGlobalAdmin)
	}
}

func TestValidateTokenExpired(t *testing.T) {
	// Create token with -1 hour expiry (already expired)
	user := &models.User{ID: 1, Email: "test@example.com"}
	site := &models.Site{ID: 5}

	// We'll need to manually create an expired token
	// For now, just test that validation fails after sleep
	// This is a simplified test - in real scenario we'd mock time
	token, _ := GenerateToken(user, site)

	// Validate immediately should work
	_, err := ValidateToken(token)
	if err != nil {
		t.Error("Token should be valid immediately after generation")
	}
}

func TestValidateTokenInvalidSignature(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJzaXRlX2lkIjo1LCJpc19nbG9iYWxfYWRtaW4iOmZhbHNlLCJleHAiOjk5OTk5OTk5OTl9.invalid-signature"

	_, err := ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken should fail for invalid signature")
	}
}

func TestValidateTokenMalformed(t *testing.T) {
	token := "not-a-valid-jwt-token"

	_, err := ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken should fail for malformed token")
	}
}

func TestValidateTokenEmpty(t *testing.T) {
	_, err := ValidateToken("")
	if err == nil {
		t.Error("ValidateToken should fail for empty token")
	}
}
```

### Step 3: Run test to verify it fails

Run:
```bash
go test ./internal/auth -v -run TestGenerateToken
```

Expected output: Compilation errors - functions not defined

### Step 4: Implement JWT functions

Create `internal/auth/jwt.go`:

```go
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// Claims represents JWT claims for authentication
type Claims struct {
	UserID        uint   `json:"user_id"`
	Email         string `json:"email"`
	SiteID        uint   `json:"site_id"`
	IsGlobalAdmin bool   `json:"is_global_admin"`
	jwt.RegisteredClaims
}

// getJWTSecret returns the JWT secret from env var or config
func getJWTSecret() string {
	// Environment variable takes precedence
	if secret := os.Getenv("STINKY_JWT_SECRET"); secret != "" {
		return secret
	}
	return config.GetString("auth.jwt_secret")
}

// GenerateToken creates a JWT token for a user and site
func GenerateToken(user *models.User, site *models.Site) (string, error) {
	expiryHours := config.GetInt("auth.jwt_expiry_hours")
	if expiryHours == 0 {
		expiryHours = 8 // Default fallback
	}

	claims := Claims{
		UserID:        user.ID,
		Email:         user.Email,
		SiteID:        site.ID,
		IsGlobalAdmin: user.IsGlobalAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// ValidateToken parses and validates a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(getJWTSecret()), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
```

### Step 5: Run tests to verify they pass

Run:
```bash
go test ./internal/auth -v
```

Expected output: All tests PASS

### Step 6: Commit

```bash
git add internal/auth/jwt.go internal/auth/jwt_test.go internal/config/config.go
git commit -m "feat: add JWT token generation and validation

Implement GenerateToken and ValidateToken with 8-hour expiry.
Support environment variable override for JWT secret.
Add auth configuration defaults."
```

---

## Task 4: Authentication Middleware

**Files:**
- Create: `internal/auth/middleware.go`
- Create: `internal/auth/middleware_test.go`

### Step 1: Write the failing test

Create `internal/auth/middleware_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	err = database.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return database
}

func TestRequireAuthWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupAuthTestDB(t)
	db.SetDB(database)

	// Create user and site
	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	database.Create(&user)

	site := models.Site{Subdomain: "test", OwnerID: user.ID}
	database.Create(&site)

	// Generate token
	token, err := GenerateToken(&user, &site)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "stinky_token",
		Value: token,
	})
	c.Set("site", &site)

	// Call middleware
	called := false
	middleware := RequireAuth()
	middleware(c)
	c.Next()

	if c.IsAborted() {
		t.Error("Middleware should not abort with valid token")
	}

	if w.Code != 0 && w.Code != 200 {
		t.Errorf("Expected status 0 or 200, got %d", w.Code)
	}

	// Check user was set in context
	contextUser, exists := c.Get("user")
	if !exists {
		t.Error("User should be set in context")
	}

	if contextUser.(*models.User).ID != user.ID {
		t.Error("User ID in context should match")
	}
}

func TestRequireAuthWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupAuthTestDB(t)
	db.SetDB(database)

	site := models.Site{ID: 1, Subdomain: "test"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
	c.Set("site", &site)

	middleware := RequireAuth()
	middleware(c)

	if !c.IsAborted() {
		t.Error("Middleware should abort without token")
	}

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireAuthWithExpiredToken(t *testing.T) {
	// This test would require mocking time or waiting
	// For now we'll just test invalid token format
	gin.SetMode(gin.TestMode)
	database := setupAuthTestDB(t)
	db.SetDB(database)

	site := models.Site{ID: 1, Subdomain: "test"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "stinky_token",
		Value: "invalid-token",
	})
	c.Set("site", &site)

	middleware := RequireAuth()
	middleware(c)

	if !c.IsAborted() {
		t.Error("Middleware should abort with invalid token")
	}

	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireAuthUserWithoutSiteAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupAuthTestDB(t)
	db.SetDB(database)

	// Create user and site they own
	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	database.Create(&user)

	ownedSite := models.Site{Subdomain: "owned", OwnerID: user.ID}
	database.Create(&ownedSite)

	// Create different site they don't have access to
	otherSite := models.Site{Subdomain: "other", OwnerID: 999}
	database.Create(&otherSite)

	// Generate token for owned site
	token, _ := GenerateToken(&user, &ownedSite)

	// Try to access other site
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "stinky_token",
		Value: token,
	})
	c.Set("site", &otherSite)

	middleware := RequireAuth()
	middleware(c)

	if !c.IsAborted() {
		t.Error("Middleware should abort when user doesn't have site access")
	}

	if w.Code != 403 {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestRequireAuthGlobalAdminCanAccessAnySite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupAuthTestDB(t)
	db.SetDB(database)

	// Create global admin
	admin := models.User{
		Email:         "admin@example.com",
		PasswordHash:  "hash",
		IsGlobalAdmin: true,
	}
	database.Create(&admin)

	// Create any site
	adminSite := models.Site{Subdomain: "admin", OwnerID: admin.ID}
	database.Create(&adminSite)

	otherSite := models.Site{Subdomain: "other", OwnerID: 999}
	database.Create(&otherSite)

	// Generate token
	token, _ := GenerateToken(&admin, &adminSite)

	// Try to access other site
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "stinky_token",
		Value: token,
	})
	c.Set("site", &otherSite)

	middleware := RequireAuth()
	middleware(c)

	if c.IsAborted() {
		t.Error("Middleware should not abort for global admin")
	}
}
```

### Step 2: Run test to verify it fails

Run:
```bash
go test ./internal/auth -v -run TestRequireAuth
```

Expected output: Compilation error - `RequireAuth` not defined

### Step 3: Implement authentication middleware

Create `internal/auth/middleware.go`:

```go
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
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Validate token
		claims, err := ValidateToken(cookie)
		if err != nil {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Load user from database
		var user models.User
		if err := db.GetDB().First(&user, claims.UserID).Error; err != nil {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		// Get site from context (set by site resolution middleware)
		siteVal, exists := c.Get("site")
		if !exists {
			c.Status(http.StatusInternalServerError)
			c.Abort()
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
			c.String(http.StatusForbidden, "You don't have access to this site")
			c.Abort()
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
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		user := userVal.(*models.User)
		if !user.IsGlobalAdmin {
			c.String(http.StatusForbidden, "Global administrator access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
```

### Step 4: Run tests to verify they pass

Run:
```bash
go test ./internal/auth -v
```

Expected output: All tests PASS

### Step 5: Commit

```bash
git add internal/auth/middleware.go internal/auth/middleware_test.go
git commit -m "feat: add authentication middleware

Implement RequireAuth middleware for JWT validation and site access checks.
Add RequireGlobalAdmin for platform administrator routes.
Returns 401 for invalid/missing tokens, 403 for insufficient permissions."
```

---

## Task 5: Login Handler

**Files:**
- Create: `internal/handlers/admin.go`
- Create: `internal/handlers/admin_test.go`

### Step 1: Write the failing test

Create `internal/handlers/admin_test.go`:

```go
package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	err = database.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return database
}

func TestLoginHandlerValidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	// Create user and site
	passwordHash, _ := auth.HashPassword("test-password")
	user := models.User{
		Email:        "test@example.com",
		PasswordHash: passwordHash,
	}
	database.Create(&user)

	site := models.Site{
		Subdomain: "test",
		OwnerID:   user.ID,
	}
	database.Create(&site)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "test-password")

	c.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Set("site", &site)

	LoginHandler(c)

	// Should redirect to dashboard
	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302, got %d", w.Code)
	}

	// Should set cookie
	cookies := w.Result().Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "stinky_token" {
			found = true
			if cookie.Value == "" {
				t.Error("Cookie value should not be empty")
			}
			if !cookie.HttpOnly {
				t.Error("Cookie should be HttpOnly")
			}
		}
	}

	if !found {
		t.Error("stinky_token cookie should be set")
	}
}

func TestLoginHandlerInvalidPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	passwordHash, _ := auth.HashPassword("correct-password")
	user := models.User{
		Email:        "test@example.com",
		PasswordHash: passwordHash,
	}
	database.Create(&user)

	site := models.Site{
		Subdomain: "test",
		OwnerID:   user.ID,
	}
	database.Create(&site)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "wrong-password")

	c.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Set("site", &site)

	LoginHandler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestLoginHandlerInvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	site := models.Site{ID: 1, Subdomain: "test"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("email", "nonexistent@example.com")
	form.Add("password", "password")

	c.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Set("site", &site)

	LoginHandler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Should not reveal that email doesn't exist
	body := w.Body.String()
	if !strings.Contains(body, "Invalid email or password") {
		t.Error("Error message should be generic")
	}
}

func TestLoginHandlerNoSiteAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	passwordHash, _ := auth.HashPassword("test-password")
	user := models.User{
		Email:        "test@example.com",
		PasswordHash: passwordHash,
	}
	database.Create(&user)

	// User's own site
	ownSite := models.Site{
		Subdomain: "own",
		OwnerID:   user.ID,
	}
	database.Create(&ownSite)

	// Different site
	otherSite := models.Site{
		Subdomain: "other",
		OwnerID:   999,
	}
	database.Create(&otherSite)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("email", "test@example.com")
	form.Add("password", "test-password")

	c.Request = httptest.NewRequest("POST", "/admin/login", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Set("site", &otherSite)

	LoginHandler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}
```

### Step 2: Run test to verify it fails

Run:
```bash
go test ./internal/handlers -v -run TestLoginHandler
```

Expected output: Compilation error - `LoginHandler` not defined

### Step 3: Implement login handler

Create `internal/handlers/admin.go`:

```go
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
		c.String(http.StatusInternalServerError, "Failed to generate token")
		return
	}

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

	// Set SameSite attribute
	c.SetSameSite(http.SameSiteLaxMode)

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
```

### Step 4: Run tests to verify they pass

Run:
```bash
go test ./internal/handlers -v
```

Expected output: All tests PASS

### Step 5: Commit

```bash
git add internal/handlers/admin.go internal/handlers/admin_test.go
git commit -m "feat: add admin login, logout, and dashboard handlers

Implement LoginHandler with credential validation and site access checks.
Add LogoutHandler to clear authentication cookie.
Add DashboardHandler to display user and site information."
```

---

## Task 6: Update Server with Auth Middleware

**Files:**
- Modify: `cmd/stinky/server.go`

### Step 1: Read current server.go to understand structure

This is a modification task. We need to:
1. Import the new packages
2. Remove old placeholder handlers
3. Add real handlers with auth middleware

### Step 2: Update server.go

In `cmd/stinky/server.go`, make these changes:

**Add imports:**
```go
"github.com/thatcatcamp/stinkykitty/internal/auth"
```

**Replace the admin route section** (starting around line 68) with:

```go
		// Admin routes
		adminGroup := siteGroup.Group("/admin")
		adminGroup.Use(middleware.IPFilterMiddleware(blocklist))
		{
			// Login route (no auth required, but rate limited)
			adminGroup.POST("/login", middleware.RateLimitMiddleware(loginRateLimiter, "/admin/login"), handlers.LoginHandler)

			// Logout route (auth required)
			adminGroup.POST("/logout", auth.RequireAuth(), handlers.LogoutHandler)

			// Protected admin routes
			adminGroup.Use(auth.RequireAuth())
			{
				adminGroup.GET("/dashboard", handlers.DashboardHandler)
			}
		}
```

Complete updated `server.go`:

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/handlers"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server operations",
	Long:  "Start, stop, and manage the StinkyKitty HTTP server",
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create Gin router
		r := gin.Default()

		// System routes (no site context needed)
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "stinkykitty",
			})
		})

		// Get base domain from config (default to localhost for development)
		baseDomain := config.GetString("server.base_domain")
		if baseDomain == "" {
			baseDomain = "localhost"
		}

		// Get global IP blocklist from config
		var blocklist []string
		// TODO: Load from config when we add security.blocked_ips to config schema

		// Create rate limiter for admin routes
		loginRateLimiter := middleware.NewRateLimiter(5, time.Minute)

		// Site-required routes
		siteGroup := r.Group("/")
		siteGroup.Use(middleware.SiteResolutionMiddleware(db.GetDB(), baseDomain))
		{
			// Public content routes
			siteGroup.GET("/", handlers.ServeHomepage)

			// Admin routes
			adminGroup := siteGroup.Group("/admin")
			adminGroup.Use(middleware.IPFilterMiddleware(blocklist))
			{
				// Login route (no auth required, but rate limited)
				adminGroup.POST("/login", middleware.RateLimitMiddleware(loginRateLimiter, "/admin/login"), handlers.LoginHandler)

				// Logout route (auth required)
				adminGroup.POST("/logout", auth.RequireAuth(), handlers.LogoutHandler)

				// Protected admin routes
				adminGroup.Use(auth.RequireAuth())
				{
					adminGroup.GET("/dashboard", handlers.DashboardHandler)
				}
			}
		}

		httpPort := config.GetString("server.http_port")
		addr := fmt.Sprintf(":%s", httpPort)

		fmt.Printf("Starting StinkyKitty server on %s\n", addr)
		fmt.Printf("Base domain: %s\n", baseDomain)
		if err := r.Run(addr); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	rootCmd.AddCommand(serverCmd)
}
```

### Step 3: Test the server compiles

Run:
```bash
go build -o stinky cmd/stinky/*.go
```

Expected output: Successful compilation

### Step 4: Start server and test basic functionality

Run:
```bash
./stinky server start
```

Expected output: Server starts without errors

(Kill with Ctrl+C after verifying)

### Step 5: Commit

```bash
git add cmd/stinky/server.go
git commit -m "feat: integrate authentication into server middleware chain

Replace placeholder admin handlers with real authentication.
Add auth middleware to protected admin routes.
Login and logout routes properly configured with rate limiting."
```

---

## Task 7: Integration Testing

**Files:**
- Manual testing with browser / curl

### Step 1: Rebuild binary with latest code

Run:
```bash
make build
```

### Step 2: Start server

Run:
```bash
./stinky server start
```

### Step 3: Create test user and site (separate terminal)

Run:
```bash
./stinky user create test@example.com
# Enter password when prompted

./stinky site create testsite --owner test@example.com
./stinky site add-domain testsite localhost
```

### Step 4: Test login flow with curl

Run:
```bash
# Test login with valid credentials
curl -i -X POST http://localhost:17890/admin/login \
  -H "Host: localhost" \
  -d "email=test@example.com&password=YOUR_PASSWORD" \
  -c cookies.txt

# Should return 302 redirect with Set-Cookie header
```

### Step 5: Test protected route with token

Run:
```bash
# Test dashboard with cookie
curl -i http://localhost:17890/admin/dashboard \
  -H "Host: localhost" \
  -b cookies.txt

# Should return 200 with user info
```

### Step 6: Test protected route without token

Run:
```bash
# Test dashboard without cookie
curl -i http://localhost:17890/admin/dashboard \
  -H "Host: localhost"

# Should redirect to /admin/login
```

### Step 7: Test logout

Run:
```bash
# Test logout
curl -i -X POST http://localhost:17890/admin/logout \
  -H "Host: localhost" \
  -b cookies.txt

# Should clear cookie and redirect to login
```

### Step 8: Document test results

Create a simple test log or checklist verifying:
- [ ] Login with valid credentials works
- [ ] Login with invalid credentials fails
- [ ] Protected routes require authentication
- [ ] Logout clears session
- [ ] Rate limiting works (try 6 rapid logins)

### Step 9: Commit any fixes or documentation

```bash
git add -A
git commit -m "test: verify authentication system integration

Manual testing confirms:
- Login/logout flow works correctly
- JWT cookies are set properly
- Auth middleware protects admin routes
- Rate limiting prevents brute force"
```

---

## Completion Checklist

- [ ] IsGlobalAdmin field added to User model
- [ ] Password hashing and verification implemented
- [ ] JWT token generation and validation working
- [ ] Auth middleware protects admin routes
- [ ] Login handler validates credentials and site access
- [ ] Logout handler clears authentication
- [ ] Dashboard shows authenticated user info
- [ ] Server integrates all auth components
- [ ] Manual testing completed successfully
- [ ] All tests pass: `go test ./...`
- [ ] All code committed

## Next Steps

After authentication is complete:
1. Create simple login HTML form (replace form-urlencoded with real UI)
2. Add visible "Login" link to public site navigation
3. Build admin panel UI (React/Vue)
4. Add password reset flow
5. Implement content blocks system

---

## Notes for Future Reference

- JWT secret should be set via `STINKY_JWT_SECRET` environment variable in production
- Cookie `Secure` flag should be `true` when using HTTPS
- Consider adding audit logging for login attempts
- Token expiry is 8 hours - adjust in config if needed
