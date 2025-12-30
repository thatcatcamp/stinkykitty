// SPDX-License-Identifier: MIT
package media

import (
	"fmt"
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
		{
			name:      "image block with invalid JSON",
			blockType: "image",
			blockData: `{invalid json}`,
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

func TestFindImageUsageExcludesSoftDeleted(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Site{}, &models.Page{}, &models.Block{}, &models.User{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{Subdomain: "test", OwnerID: user.ID, SiteDir: "/tmp"}
	db.Create(&site)

	// Create active page with image
	activePage := models.Page{SiteID: site.ID, Slug: "/active", Title: "Active", Published: true}
	db.Create(&activePage)

	activeBlock := models.Block{
		PageID: activePage.ID,
		Type:   "image",
		Order:  1,
		Data:   `{"url":"/uploads/shared.jpg","alt":"Shared image"}`,
	}
	db.Create(&activeBlock)

	// Create soft-deleted page with image
	deletedPage := models.Page{SiteID: site.ID, Slug: "/deleted", Title: "Deleted", Published: false}
	db.Create(&deletedPage)
	db.Delete(&deletedPage) // Soft delete

	deletedPageBlock := models.Block{
		PageID: deletedPage.ID,
		Type:   "image",
		Order:  1,
		Data:   `{"url":"/uploads/shared.jpg","alt":"Shared image"}`,
	}
	db.Create(&deletedPageBlock)

	// Create page with soft-deleted block
	pageWithDeletedBlock := models.Page{SiteID: site.ID, Slug: "/page2", Title: "Page 2", Published: true}
	db.Create(&pageWithDeletedBlock)

	deletedBlock := models.Block{
		PageID: pageWithDeletedBlock.ID,
		Type:   "image",
		Order:  1,
		Data:   `{"url":"/uploads/shared.jpg","alt":"Shared image"}`,
	}
	db.Create(&deletedBlock)
	db.Delete(&deletedBlock) // Soft delete

	// Test - should only find the active page/block
	usages := FindImageUsage(db, site.ID, "/uploads/shared.jpg")

	if len(usages) != 1 {
		t.Errorf("Expected 1 usage (only active), got %d", len(usages))
	}

	if len(usages) > 0 {
		if usages[0].PageTitle != "Active" {
			t.Errorf("Expected page 'Active', got '%s'", usages[0].PageTitle)
		}
	}
}

func TestFindImageUsageMultiplePages(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Site{}, &models.Page{}, &models.Block{}, &models.User{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{Subdomain: "test", OwnerID: user.ID, SiteDir: "/tmp"}
	db.Create(&site)

	// Create multiple pages using the same image
	for i, title := range []string{"Home", "About", "Contact"} {
		page := models.Page{SiteID: site.ID, Slug: fmt.Sprintf("/%d", i), Title: title, Published: true}
		db.Create(&page)

		block := models.Block{
			PageID: page.ID,
			Type:   "image",
			Order:  1,
			Data:   `{"url":"/uploads/reused.jpg","alt":"Reused image"}`,
		}
		db.Create(&block)
	}

	// Test - should find all 3 usages
	usages := FindImageUsage(db, site.ID, "/uploads/reused.jpg")

	if len(usages) != 3 {
		t.Errorf("Expected 3 usages, got %d", len(usages))
	}

	// Verify all page titles are present
	pageTitles := make(map[string]bool)
	for _, usage := range usages {
		pageTitles[usage.PageTitle] = true
	}

	for _, expected := range []string{"Home", "About", "Contact"} {
		if !pageTitles[expected] {
			t.Errorf("Expected to find page '%s'", expected)
		}
	}
}
