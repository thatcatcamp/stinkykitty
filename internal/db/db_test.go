// SPDX-License-Identifier: MIT
package db

import (
	"testing"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return testDB
}

func TestSiteModelHasThemeFields(t *testing.T) {
	// Get test database instance
	testDB := setupTestDB(t)

	// Run migrations
	if err := testDB.AutoMigrate(&models.Site{}); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Check if theme_palette column exists
	if !testDB.Migrator().HasColumn(&models.Site{}, "theme_palette") {
		t.Fatal("theme_palette column not found in sites table")
	}

	// Check if dark_mode column exists
	if !testDB.Migrator().HasColumn(&models.Site{}, "dark_mode") {
		t.Fatal("dark_mode column not found in sites table")
	}
}

func TestThemeFieldDefaults(t *testing.T) {
	testDB := setupTestDB(t)

	if err := testDB.AutoMigrate(&models.Site{}); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Create a site without specifying theme fields
	site := &models.Site{
		Subdomain: "test-camp",
		SiteTitle: "Test Camp",
		SiteDir:   "/tmp/test-camp",
		OwnerID:   1,
		// Note: not setting ThemePalette or DarkMode
	}

	if err := testDB.Create(site).Error; err != nil {
		t.Fatalf("failed to create site: %v", err)
	}

	// Retrieve the site from database to check defaults
	var retrievedSite models.Site
	if err := testDB.First(&retrievedSite, site.ID).Error; err != nil {
		t.Fatalf("failed to retrieve site: %v", err)
	}

	// Verify defaults were applied
	if retrievedSite.ThemePalette != "slate" {
		t.Errorf("expected default theme_palette 'slate', got '%s'", retrievedSite.ThemePalette)
	}
	if retrievedSite.DarkMode != false {
		t.Errorf("expected default dark_mode false, got %v", retrievedSite.DarkMode)
	}
}
