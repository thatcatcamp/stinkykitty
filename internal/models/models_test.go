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

// Tests for AllowedIPs field and helper methods
func TestSiteAllowedIPsField(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	// Test creating site with AllowedIPs
	allowedIPs := `["10.0.0.0/24", "192.168.1.0/24"]`
	site := Site{
		Subdomain:  "testcamp",
		OwnerID:    user.ID,
		SiteDir:    "/tmp/test",
		AllowedIPs: allowedIPs,
	}

	result := db.Create(&site)
	if result.Error != nil {
		t.Fatalf("Failed to create site with AllowedIPs: %v", result.Error)
	}

	// Retrieve and verify
	var retrieved Site
	db.First(&retrieved, site.ID)

	if retrieved.AllowedIPs != allowedIPs {
		t.Errorf("Expected AllowedIPs %s, got %s", allowedIPs, retrieved.AllowedIPs)
	}
}

func TestSiteGetAllowedIPs(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:  "testcamp",
		OwnerID:    user.ID,
		SiteDir:    "/tmp/test",
		AllowedIPs: `["10.0.0.0/24", "192.168.1.100/32"]`,
	}
	db.Create(&site)

	// Test GetAllowedIPs
	ips, err := site.GetAllowedIPs()
	if err != nil {
		t.Fatalf("GetAllowedIPs failed: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(ips))
	}

	if ips[0] != "10.0.0.0/24" {
		t.Errorf("Expected first IP '10.0.0.0/24', got '%s'", ips[0])
	}

	if ips[1] != "192.168.1.100/32" {
		t.Errorf("Expected second IP '192.168.1.100/32', got '%s'", ips[1])
	}
}

func TestSiteGetAllowedIPsEmpty(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// Test GetAllowedIPs on empty field
	ips, err := site.GetAllowedIPs()
	if err != nil {
		t.Fatalf("GetAllowedIPs failed: %v", err)
	}

	if len(ips) != 0 {
		t.Errorf("Expected empty array, got %d IPs", len(ips))
	}
}

func TestSiteSetAllowedIPs(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// Test SetAllowedIPs
	newIPs := []string{"172.16.0.0/12", "10.1.2.3/32"}
	err := site.SetAllowedIPs(newIPs)
	if err != nil {
		t.Fatalf("SetAllowedIPs failed: %v", err)
	}

	// Verify it was set correctly
	ips, err := site.GetAllowedIPs()
	if err != nil {
		t.Fatalf("GetAllowedIPs failed: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(ips))
	}

	if ips[0] != "172.16.0.0/12" {
		t.Errorf("Expected first IP '172.16.0.0/12', got '%s'", ips[0])
	}
}

func TestSiteAddAllowedIP(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:  "testcamp",
		OwnerID:    user.ID,
		SiteDir:    "/tmp/test",
		AllowedIPs: `["10.0.0.0/24"]`,
	}
	db.Create(&site)

	// Test AddAllowedIP
	err := site.AddAllowedIP("192.168.1.0/24")
	if err != nil {
		t.Fatalf("AddAllowedIP failed: %v", err)
	}

	// Verify it was added
	ips, err := site.GetAllowedIPs()
	if err != nil {
		t.Fatalf("GetAllowedIPs failed: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(ips))
	}

	if ips[1] != "192.168.1.0/24" {
		t.Errorf("Expected second IP '192.168.1.0/24', got '%s'", ips[1])
	}
}

func TestSiteAddAllowedIPToEmpty(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// Test AddAllowedIP to empty list
	err := site.AddAllowedIP("10.0.0.0/24")
	if err != nil {
		t.Fatalf("AddAllowedIP failed: %v", err)
	}

	// Verify it was added
	ips, err := site.GetAllowedIPs()
	if err != nil {
		t.Fatalf("GetAllowedIPs failed: %v", err)
	}

	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}

	if ips[0] != "10.0.0.0/24" {
		t.Errorf("Expected IP '10.0.0.0/24', got '%s'", ips[0])
	}
}

func TestSiteRemoveAllowedIP(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:  "testcamp",
		OwnerID:    user.ID,
		SiteDir:    "/tmp/test",
		AllowedIPs: `["10.0.0.0/24", "192.168.1.0/24", "172.16.0.0/12"]`,
	}
	db.Create(&site)

	// Test RemoveAllowedIP
	err := site.RemoveAllowedIP("192.168.1.0/24")
	if err != nil {
		t.Fatalf("RemoveAllowedIP failed: %v", err)
	}

	// Verify it was removed
	ips, err := site.GetAllowedIPs()
	if err != nil {
		t.Fatalf("GetAllowedIPs failed: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(ips))
	}

	// Verify the correct one was removed
	for _, ip := range ips {
		if ip == "192.168.1.0/24" {
			t.Error("IP should have been removed")
		}
	}
}

func TestSiteRemoveAllowedIPNotFound(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:  "testcamp",
		OwnerID:    user.ID,
		SiteDir:    "/tmp/test",
		AllowedIPs: `["10.0.0.0/24"]`,
	}
	db.Create(&site)

	// Test RemoveAllowedIP for non-existent IP
	err := site.RemoveAllowedIP("192.168.1.0/24")
	if err == nil {
		t.Error("Expected error when removing non-existent IP")
	}

	// Verify original IP still exists
	ips, err := site.GetAllowedIPs()
	if err != nil {
		t.Fatalf("GetAllowedIPs failed: %v", err)
	}

	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}
}

func TestSiteRemoveAllowedIPEmpty(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// Test RemoveAllowedIP on empty list
	err := site.RemoveAllowedIP("10.0.0.0/24")
	if err == nil {
		t.Error("Expected error when removing from empty list")
	}
}
