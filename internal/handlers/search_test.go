package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/search"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func setupSearchTestDB(t *testing.T) *gorm.DB {
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Migrate models
	if err := testDB.AutoMigrate(&models.User{}, &models.Site{}, &models.Page{}, &models.Block{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Initialize FTS index
	sqlDB, _ := testDB.DB()
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

	return testDB
}

func TestSearchHandler(t *testing.T) {
	// Setup
	testDB := setupSearchTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := models.Site{
		ID:        1,
		Subdomain: "testsite",
	}
	testDB.Create(&site)

	// Create test page
	page := models.Page{
		SiteID:    site.ID,
		Slug:      "/test",
		Title:     "Amazing Cat Adventures",
		Published: true,
	}
	testDB.Create(&page)

	// Create test block
	block := models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"This page is all about cats and their incredible journeys."}`,
	}
	testDB.Create(&block)

	// Index the page
	if err := search.IndexPage(testDB, &page); err != nil {
		t.Fatalf("Failed to index page: %v", err)
	}

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to inject site into context
	router.Use(func(c *gin.Context) {
		c.Set("site", &site)
		c.Next()
	})

	// Add theme CSS to context
	router.Use(func(c *gin.Context) {
		c.Set("themeCSS", "")
		c.Next()
	})

	router.GET("/search", SearchHandler)

	// Test 1: Search for "cat" - should find the page
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/search?q=cat", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that HTML response contains the page title
	body := w.Body.String()
	if !containsString(body, page.Title) {
		t.Errorf("Expected HTML to contain page title %q", page.Title)
	}

	// Check that it shows 1 result
	if !containsString(body, "Found 1 result") {
		t.Errorf("Expected HTML to show 1 result")
	}

	// Test 2: Search for "dog" - should find nothing
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/search?q=dog", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that HTML response shows no results
	body = w.Body.String()
	if !containsString(body, "No results found") {
		t.Errorf("Expected HTML to show no results message")
	}

	// Test 3: Missing query parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/search", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing query, got %d", w.Code)
	}
}

func TestSearchHandlerIsolation(t *testing.T) {
	// Setup
	testDB := setupSearchTestDB(t)
	db.SetDB(testDB)

	// Create two test sites
	site1 := models.Site{ID: 1, Subdomain: "site1"}
	site2 := models.Site{ID: 2, Subdomain: "site2"}
	testDB.Create(&site1)
	testDB.Create(&site2)

	// Create page for site1
	page1 := models.Page{
		SiteID:    site1.ID,
		Slug:      "/cats",
		Title:     "All About Cats",
		Published: true,
	}
	testDB.Create(&page1)

	block1 := models.Block{
		PageID: page1.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Cats are wonderful pets."}`,
	}
	testDB.Create(&block1)

	// Create page for site2
	page2 := models.Page{
		SiteID:    site2.ID,
		Slug:      "/dogs",
		Title:     "Dog Training",
		Published: true,
	}
	testDB.Create(&page2)

	block2 := models.Block{
		PageID: page2.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Dogs need training."}`,
	}
	testDB.Create(&block2)

	// Index both pages
	search.IndexPage(testDB, &page1)
	search.IndexPage(testDB, &page2)

	// Setup Gin router with site1 context
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("site", &site1)
		c.Set("themeCSS", "")
		c.Next()
	})
	router.GET("/search", SearchHandler)

	// Search for "cats" on site1 - should find page1
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/search?q=cats", nil)
	router.ServeHTTP(w, req)

	body := w.Body.String()
	if !containsString(body, page1.Title) {
		t.Fatalf("Expected to find page1 title on site1")
	}
	if !containsString(body, "Found 1 result") {
		t.Fatalf("Expected 1 result on site1")
	}

	// Search for "dogs" on site1 - should find nothing (isolation)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/search?q=dogs", nil)
	router.ServeHTTP(w, req)

	body = w.Body.String()
	if !containsString(body, "No results found") {
		t.Fatalf("Expected 0 results on site1 (isolation)")
	}

	// Now test with site2 context
	router2 := gin.New()
	router2.Use(func(c *gin.Context) {
		c.Set("site", &site2)
		c.Set("themeCSS", "")
		c.Next()
	})
	router2.GET("/search", SearchHandler)

	// Search for "dogs" on site2 - should find page2
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/search?q=dogs", nil)
	router2.ServeHTTP(w, req)

	body = w.Body.String()
	if !containsString(body, page2.Title) {
		t.Fatalf("Expected to find page2 title on site2")
	}
	if !containsString(body, "Found 1 result") {
		t.Fatalf("Expected 1 result on site2")
	}

	// Search for "cats" on site2 - should find nothing (isolation)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/search?q=cats", nil)
	router2.ServeHTTP(w, req)

	body = w.Body.String()
	if !containsString(body, "No results found") {
		t.Fatalf("Expected 0 results on site2 (isolation)")
	}
}
