package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(&User{}, &Site{}, &SiteUser{}, &Page{}, &Block{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)

	user := User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}

	result := db.Create(&user)
	if result.Error != nil {
		t.Fatalf("Failed to create user: %v", result.Error)
	}

	if user.ID == 0 {
		t.Error("User ID should be set after creation")
	}
}

func TestCreateSite(t *testing.T) {
	db := setupTestDB(t)

	// Create owner first
	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:    "testcamp",
		OwnerID:      user.ID,
		SiteDir:      "/var/lib/stinkykitty/sites/site-123",
		DatabaseType: "sqlite",
		DatabasePath: "/var/lib/stinkykitty/sites/site-123/site.db",
	}

	result := db.Create(&site)
	if result.Error != nil {
		t.Fatalf("Failed to create site: %v", result.Error)
	}

	if site.ID == 0 {
		t.Error("Site ID should be set after creation")
	}
}

func TestSiteUserRelationship(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "admin@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:    "camp",
		OwnerID:      user.ID,
		SiteDir:      "/var/lib/stinkykitty/sites/site-456",
		DatabaseType: "sqlite",
	}
	db.Create(&site)

	siteUser := SiteUser{
		UserID: user.ID,
		SiteID: site.ID,
		Role:   "admin",
	}
	db.Create(&siteUser)

	// Query to verify relationship
	var retrievedSiteUser SiteUser
	result := db.Where("user_id = ? AND site_id = ?", user.ID, site.ID).First(&retrievedSiteUser)
	if result.Error != nil {
		t.Fatalf("Failed to retrieve site user: %v", result.Error)
	}

	if retrievedSiteUser.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", retrievedSiteUser.Role)
	}
}
