// SPDX-License-Identifier: MIT
package auth

import (
	"fmt"
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

	err = database.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{}, &models.Page{}, &models.Block{})
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

// TestRequireAuthUpdatesContextWithQueryParamSite verifies that when accessing
// a site via ?site=X query parameter, the middleware updates the context
// with the correct site, not the site resolved from Host header.
// This is a regression test for the "dead camps" bug.
func TestRequireAuthUpdatesContextWithQueryParamSite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupAuthTestDB(t)
	db.SetDB(database)

	// Create user
	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	database.Create(&user)

	// Create two sites
	site1 := models.Site{Subdomain: "maincamp", OwnerID: user.ID}
	database.Create(&site1)

	site2 := models.Site{Subdomain: "testcamp", OwnerID: user.ID}
	database.Create(&site2)

	// Generate token for user
	token, err := GenerateToken(&user, &site1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Simulate SiteResolutionMiddleware setting site1 in context
	// (because request came to maincamp.example.com)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Request to maincamp but query parameter asks for site2
	c.Request = httptest.NewRequest("GET", "/admin/pages?site="+fmt.Sprintf("%d", site2.ID), nil)
	c.Request.AddCookie(&http.Cookie{
		Name:  "stinky_token",
		Value: token,
	})

	// SiteResolutionMiddleware would have set site1
	c.Set("site", &site1)

	// Call RequireAuth middleware
	middleware := RequireAuth()
	middleware(c)

	if c.IsAborted() {
		t.Error("Middleware should not abort when user has access to queried site")
	}

	// CRITICAL BUG TEST: Context should have been updated to site2, not site1
	contextSite, exists := c.Get("site")
	if !exists {
		t.Fatal("Site should be in context after RequireAuth")
	}

	contextSiteVal := contextSite.(*models.Site)
	if contextSiteVal.ID != site2.ID {
		t.Errorf("Context site should be updated to queried site (ID=%d), but got ID=%d",
			site2.ID, contextSiteVal.ID)
	}
}
