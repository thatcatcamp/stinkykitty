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

	err = database.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{}, &models.Page{}, &models.Block{})
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
	// Check Gin context status instead of httptest recorder
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d. Location: %s",
			c.Writer.Status(), w.Result().Header.Get("Location"))
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
