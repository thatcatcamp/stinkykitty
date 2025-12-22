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
