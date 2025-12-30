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
