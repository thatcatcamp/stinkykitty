package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMediaItemModel(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&MediaItem{}, &MediaTag{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Test creating media item
	item := MediaItem{
		SiteID:       1,
		Filename:     "abc123.jpg",
		OriginalName: "cat-photo.jpg",
		FileSize:     102400,
		MimeType:     "image/jpeg",
		UploadedBy:   1,
	}

	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("Failed to create media item: %v", err)
	}

	if item.ID == 0 {
		t.Error("Expected ID to be set after create")
	}

	// Test retrieving media item
	var retrieved MediaItem
	if err := db.First(&retrieved, item.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve media item: %v", err)
	}

	if retrieved.Filename != "abc123.jpg" {
		t.Errorf("Expected filename 'abc123.jpg', got '%s'", retrieved.Filename)
	}
}

func TestMediaTagModel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&MediaItem{}, &MediaTag{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create media item
	item := MediaItem{
		SiteID:       1,
		Filename:     "test.jpg",
		OriginalName: "test.jpg",
		FileSize:     1024,
		MimeType:     "image/jpeg",
		UploadedBy:   1,
	}
	db.Create(&item)

	// Test creating tag
	tag := MediaTag{
		MediaItemID: item.ID,
		TagName:     "summer",
	}

	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	// Test retrieving tags for media item
	var tags []MediaTag
	if err := db.Where("media_item_id = ?", item.ID).Find(&tags).Error; err != nil {
		t.Fatalf("Failed to retrieve tags: %v", err)
	}

	if len(tags) != 1 {
		t.Errorf("Expected 1 tag, got %d", len(tags))
	}

	if tags[0].TagName != "summer" {
		t.Errorf("Expected tag 'summer', got '%s'", tags[0].TagName)
	}
}

func TestMediaItemTagRelationship(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable foreign key constraints for SQLite
	db.Exec("PRAGMA foreign_keys = ON")

	// Migrate all related models
	if err := db.AutoMigrate(&User{}, &Site{}, &MediaItem{}, &MediaTag{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test user
	user := User{
		Email:        "test@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create test site
	site := Site{
		Subdomain: "test",
		OwnerID:   user.ID,
		SiteDir:   "/test",
	}
	if err := db.Create(&site).Error; err != nil {
		t.Fatalf("Failed to create site: %v", err)
	}

	// Create media item with tags
	item := MediaItem{
		SiteID:       site.ID,
		Filename:     "test.jpg",
		OriginalName: "original.jpg",
		FileSize:     1024,
		MimeType:     "image/jpeg",
		UploadedBy:   user.ID,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("Failed to create media item: %v", err)
	}

	// Add tags
	tags := []MediaTag{
		{MediaItemID: item.ID, TagName: "summer"},
		{MediaItemID: item.ID, TagName: "2024"},
	}
	if err := db.Create(&tags).Error; err != nil {
		t.Fatalf("Failed to create tags: %v", err)
	}

	// Test preloading Site relationship
	var retrieved MediaItem
	if err := db.Preload("Site").First(&retrieved, item.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve media item with site: %v", err)
	}
	if retrieved.Site.Subdomain != "test" {
		t.Errorf("Expected site subdomain 'test', got '%s'", retrieved.Site.Subdomain)
	}

	// Test preloading User relationship
	if err := db.Preload("User").First(&retrieved, item.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve media item with user: %v", err)
	}
	if retrieved.User.Email != "test@example.com" {
		t.Errorf("Expected user email 'test@example.com', got '%s'", retrieved.User.Email)
	}

	// Test preloading Tags relationship
	if err := db.Preload("Tags").First(&retrieved, item.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve media item with tags: %v", err)
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}

	// Test cascade delete (use Unscoped to perform hard delete, not soft delete)
	if err := db.Unscoped().Delete(&item).Error; err != nil {
		t.Fatalf("Failed to delete media item: %v", err)
	}

	var remainingTags []MediaTag
	db.Unscoped().Where("media_item_id = ?", item.ID).Find(&remainingTags)
	if len(remainingTags) != 0 {
		t.Errorf("Expected 0 tags after cascade delete, got %d", len(remainingTags))
	}
}
