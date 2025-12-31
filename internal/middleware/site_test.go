// SPDX-License-Identifier: MIT
package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{}, &models.Page{}, &models.Block{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}
	return db
}

func TestSiteResolutionBySubdomain(t *testing.T) {
	db := setupTestDB(t)

	// Create test site
	user := models.User{Email: "owner@test.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// Create Gin context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Host = "testcamp.stinkykitty.org"

	// Create middleware
	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")

	// Execute middleware
	middleware(c)

	// Check site was set in context
	siteFromCtx, exists := c.Get("site")
	if !exists {
		t.Fatal("Site not set in context")
	}

	resolvedSite := siteFromCtx.(*models.Site)
	if resolvedSite.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain 'testcamp', got '%s'", resolvedSite.Subdomain)
	}
}

func TestSiteResolutionByCustomDomain(t *testing.T) {
	db := setupTestDB(t)

	user := models.User{Email: "owner@test.com", PasswordHash: "hash"}
	db.Create(&user)

	customDomain := "thatcatcamp.com"
	site := models.Site{
		Subdomain:    "testcamp",
		CustomDomain: &customDomain,
		OwnerID:      user.ID,
		SiteDir:      "/tmp/test",
	}
	db.Create(&site)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Host = "thatcatcamp.com"

	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")
	middleware(c)

	siteFromCtx, exists := c.Get("site")
	if !exists {
		t.Fatal("Site not set in context")
	}

	resolvedSite := siteFromCtx.(*models.Site)
	if resolvedSite.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain 'testcamp', got '%s'", resolvedSite.Subdomain)
	}
}

func TestSiteResolutionNotFound(t *testing.T) {
	db := setupTestDB(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Host = "nonexistent.stinkykitty.org"

	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")
	middleware(c)

	// Should return 404
	if w.Code != 404 {
		t.Errorf("Expected 404, got %d", w.Code)
	}

	// Should not set site in context
	_, exists := c.Get("site")
	if exists {
		t.Error("Site should not be set for nonexistent subdomain")
	}
}

func TestSiteResolutionCache(t *testing.T) {
	db := setupTestDB(t)

	user := models.User{Email: "owner@test.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// First request - should cache
	gin.SetMode(gin.TestMode)
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("GET", "/", nil)
	c1.Request.Host = "testcamp.stinkykitty.org"

	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")
	middleware(c1)

	// Second request - should use cache
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/", nil)
	c2.Request.Host = "testcamp.stinkykitty.org"

	middleware(c2)

	siteFromCtx, exists := c2.Get("site")
	if !exists {
		t.Fatal("Site not set in context")
	}

	resolvedSite := siteFromCtx.(*models.Site)
	if resolvedSite.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain 'testcamp', got '%s'", resolvedSite.Subdomain)
	}
}
