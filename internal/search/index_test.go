package search

import (
	"testing"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate models
	if err := db.AutoMigrate(&models.User{}, &models.Site{}, &models.Page{}, &models.Block{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Initialize FTS index
	sqlDB, _ := db.DB()
	_, err = sqlDB.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
			page_id UNINDEXED,
			site_id UNINDEXED,
			title,
			content,
			tokenize='porter unicode61'
		)
	`)
	if err != nil {
		t.Skipf("Skipping test: FTS5 not available in test environment: %v", err)
	}

	return db
}

func TestFTSIndexing(t *testing.T) {
	db := setupTestDB(t)

	// Create test sites
	site1 := models.Site{ID: 1, Subdomain: "site1"}
	site2 := models.Site{ID: 2, Subdomain: "site2"}
	db.Create(&site1)
	db.Create(&site2)

	// Create test pages for site1
	page1 := models.Page{
		SiteID:    site1.ID,
		Slug:      "/",
		Title:     "Welcome to My Cat Blog",
		Published: true,
	}
	db.Create(&page1)

	block1 := models.Block{
		PageID: page1.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"This is a blog about cats and their amazing adventures."}`,
	}
	db.Create(&block1)

	// Create test pages for site2
	page2 := models.Page{
		SiteID:    site2.ID,
		Slug:      "/",
		Title:     "Dog Training Guide",
		Published: true,
	}
	db.Create(&page2)

	block2 := models.Block{
		PageID: page2.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Learn how to train your dog effectively."}`,
	}
	db.Create(&block2)

	// Index both pages
	if err := IndexPage(db, &page1); err != nil {
		t.Fatalf("Failed to index page1: %v", err)
	}
	if err := IndexPage(db, &page2); err != nil {
		t.Fatalf("Failed to index page2: %v", err)
	}

	// Test 1: Search for "cat" in site1 - should find page1
	results, err := Search(db, site1.ID, "cat")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].PageID != page1.ID {
		t.Errorf("Expected page ID %d, got %d", page1.ID, results[0].PageID)
	}

	// Test 2: Search for "cat" in site2 - should find nothing (isolation)
	results, err = Search(db, site2.ID, "cat")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Expected 0 results (site isolation), got %d", len(results))
	}

	// Test 3: Search for "dog" in site2 - should find page2
	results, err = Search(db, site2.ID, "dog")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].PageID != page2.ID {
		t.Errorf("Expected page ID %d, got %d", page2.ID, results[0].PageID)
	}

	// Test 4: Search for "dog" in site1 - should find nothing (isolation)
	results, err = Search(db, site1.ID, "dog")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Expected 0 results (site isolation), got %d", len(results))
	}

	// Test 5: Unpublish page1 and verify it's removed from search
	page1.Published = false
	if err := IndexPage(db, &page1); err != nil {
		t.Fatalf("Failed to update index: %v", err)
	}
	results, err = Search(db, site1.ID, "cat")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Expected 0 results (unpublished page), got %d", len(results))
	}

	// Test 6: Re-publish page1 and verify it's searchable again
	page1.Published = true
	if err := IndexPage(db, &page1); err != nil {
		t.Fatalf("Failed to update index: %v", err)
	}
	results, err = Search(db, site1.ID, "cat")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result (re-published page), got %d", len(results))
	}
}

func TestRemovePageFromIndex(t *testing.T) {
	db := setupTestDB(t)

	// Create test site and page
	site := models.Site{ID: 1, Subdomain: "test"}
	db.Create(&site)

	page := models.Page{
		SiteID:    site.ID,
		Slug:      "/test",
		Title:     "Test Page",
		Published: true,
	}
	db.Create(&page)

	block := models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Test content for searching."}`,
	}
	db.Create(&block)

	// Index the page
	if err := IndexPage(db, &page); err != nil {
		t.Fatalf("Failed to index page: %v", err)
	}

	// Verify it's searchable
	results, err := Search(db, site.ID, "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Remove from index
	if err := RemovePageFromIndex(db, page.ID); err != nil {
		t.Fatalf("Failed to remove page from index: %v", err)
	}

	// Verify it's no longer searchable
	results, err = Search(db, site.ID, "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Expected 0 results after removal, got %d", len(results))
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello world</p>", "Hello world"},
		{"<b>Bold</b> and <i>italic</i>", "Bold and italic"},
		{"No tags here", "No tags here"},
		{"<div><span>Nested</span> tags</div>", "Nested tags"},
	}

	for _, test := range tests {
		result := stripHTML(test.input)
		if result != test.expected {
			t.Errorf("stripHTML(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
