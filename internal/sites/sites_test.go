package sites

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	db.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{}, &models.Page{}, &models.Block{})
	return db
}

func TestCreateSite(t *testing.T) {
	db := setupTestDB(t)

	// Create owner
	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	site, err := CreateSite(db, "testcamp", owner.ID, "/tmp/sites")
	if err != nil {
		t.Fatalf("CreateSite failed: %v", err)
	}

	if site.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain testcamp, got %s", site.Subdomain)
	}

	if site.OwnerID != owner.ID {
		t.Errorf("Expected owner ID %d, got %d", owner.ID, site.OwnerID)
	}
}

func TestGetSiteBySubdomain(t *testing.T) {
	db := setupTestDB(t)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	CreateSite(db, "findme", owner.ID, "/tmp/sites")

	site, err := GetSiteBySubdomain(db, "findme")
	if err != nil {
		t.Fatalf("GetSiteBySubdomain failed: %v", err)
	}

	if site.Subdomain != "findme" {
		t.Errorf("Expected subdomain findme, got %s", site.Subdomain)
	}
}

func TestAddUserToSite(t *testing.T) {
	db := setupTestDB(t)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	admin := models.User{Email: "admin@example.com", PasswordHash: "hash"}
	db.Create(&admin)

	site, _ := CreateSite(db, "camp", owner.ID, "/tmp/sites")

	err := AddUserToSite(db, site.ID, admin.ID, "admin")
	if err != nil {
		t.Fatalf("AddUserToSite failed: %v", err)
	}

	// Verify relationship
	var siteUser models.SiteUser
	result := db.Where("site_id = ? AND user_id = ?", site.ID, admin.ID).First(&siteUser)
	if result.Error != nil {
		t.Fatalf("Failed to find site user: %v", result.Error)
	}

	if siteUser.Role != "admin" {
		t.Errorf("Expected role admin, got %s", siteUser.Role)
	}
}

func TestListSites(t *testing.T) {
	db := setupTestDB(t)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	CreateSite(db, "site1", owner.ID, "/tmp/sites")
	CreateSite(db, "site2", owner.ID, "/tmp/sites")

	sites, err := ListSites(db)
	if err != nil {
		t.Fatalf("ListSites failed: %v", err)
	}

	if len(sites) != 2 {
		t.Errorf("Expected 2 sites, got %d", len(sites))
	}
}
