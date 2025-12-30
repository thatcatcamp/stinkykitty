package media

import (
	"testing"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindImageUsage(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Site{}, &models.Page{}, &models.Block{}, &models.User{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{Subdomain: "test", OwnerID: user.ID, SiteDir: "/tmp"}
	db.Create(&site)

	page := models.Page{SiteID: site.ID, Slug: "/", Title: "Home", Published: true}
	db.Create(&page)

	// Create image block with specific URL
	imageBlock := models.Block{
		PageID: page.ID,
		Type:   "image",
		Order:  1,
		Data:   `{"url":"/uploads/test123.jpg","alt":"Test image","caption":""}`,
	}
	db.Create(&imageBlock)

	// Create text block (no image)
	textBlock := models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  2,
		Data:   `{"content":"Some text"}`,
	}
	db.Create(&textBlock)

	// Test finding usage
	usages := FindImageUsage(db, site.ID, "/uploads/test123.jpg")

	if len(usages) != 1 {
		t.Errorf("Expected 1 usage, got %d", len(usages))
	}

	if len(usages) > 0 {
		if usages[0].PageTitle != "Home" {
			t.Errorf("Expected page title 'Home', got '%s'", usages[0].PageTitle)
		}
		if usages[0].BlockType != "image" {
			t.Errorf("Expected block type 'image', got '%s'", usages[0].BlockType)
		}
	}

	// Test finding non-existent image
	usages2 := FindImageUsage(db, site.ID, "/uploads/nonexistent.jpg")
	if len(usages2) != 0 {
		t.Errorf("Expected 0 usages for nonexistent image, got %d", len(usages2))
	}
}

func TestContainsImageURL(t *testing.T) {
	tests := []struct {
		name      string
		blockType string
		blockData string
		imageURL  string
		expected  bool
	}{
		{
			name:      "image block with matching URL",
			blockType: "image",
			blockData: `{"url":"/uploads/cat.jpg","alt":"Cat"}`,
			imageURL:  "/uploads/cat.jpg",
			expected:  true,
		},
		{
			name:      "image block with different URL",
			blockType: "image",
			blockData: `{"url":"/uploads/dog.jpg","alt":"Dog"}`,
			imageURL:  "/uploads/cat.jpg",
			expected:  false,
		},
		{
			name:      "text block",
			blockType: "text",
			blockData: `{"content":"Some text"}`,
			imageURL:  "/uploads/cat.jpg",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := models.Block{
				Type: tt.blockType,
				Data: tt.blockData,
			}
			result := containsImageURL(block, tt.imageURL)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
