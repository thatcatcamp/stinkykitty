package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Create in-memory SQLite database
	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Auto-migrate all models
	err = testDB.AutoMigrate(&models.Site{}, &models.Page{}, &models.Block{})
	if err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return testDB
}

func TestEditPageHandler_PageNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create router and request
	router := gin.New()
	router.GET("/admin/pages/:id/edit", EditPageHandler)

	req := httptest.NewRequest("GET", "/admin/pages/999/edit", nil)
	w := httptest.NewRecorder()

	// Set site in context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	c.Set("site", site)

	// Execute
	EditPageHandler(c)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
	if w.Body.String() != "Page not found" {
		t.Errorf("Expected 'Page not found', got %s", w.Body.String())
	}
}

func TestEditPageHandler_InvalidPageID(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create router and request
	router := gin.New()
	router.GET("/admin/pages/:id/edit", EditPageHandler)

	req := httptest.NewRequest("GET", "/admin/pages/invalid/edit", nil)
	w := httptest.NewRecorder()

	// Set site in context
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	c.Set("site", site)

	// Execute
	EditPageHandler(c)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEditPageHandler_WrongSite(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create two test sites
	site1 := &models.Site{
		ID:        1,
		Subdomain: "site1",
		OwnerID:   1,
		SiteDir:   "/tmp/site1",
	}
	testDB.Create(site1)

	site2 := &models.Site{
		ID:        2,
		Subdomain: "site2",
		OwnerID:   1,
		SiteDir:   "/tmp/site2",
	}
	testDB.Create(site2)

	// Create page for site1
	page := &models.Page{
		SiteID:    site1.ID,
		Slug:      "/",
		Title:     "Homepage",
		Published: false,
	}
	testDB.Create(page)

	// Try to access from site2
	req := httptest.NewRequest("GET", "/admin/pages/1/edit", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site2) // Different site!

	// Execute
	EditPageHandler(c)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}
}

func TestEditPageHandler_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create page
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/",
		Title:     "Test Homepage",
		Published: false,
	}
	testDB.Create(page)

	// Create some blocks
	block1 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content": "This is the first block with some content that should be truncated because it is longer than one hundred characters"}`,
	}
	testDB.Create(block1)

	block2 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  1,
		Data:   `{"content": "Short block"}`,
	}
	testDB.Create(block2)

	// Create request
	req := httptest.NewRequest("GET", "/admin/pages/1/edit", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	EditPageHandler(c)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check for page title
	if !contains(body, "Test Homepage") {
		t.Error("Page title not found in response")
	}

	// Check for buttons
	if !contains(body, "Save Draft") {
		t.Error("'Save Draft' button not found")
	}
	if !contains(body, "Publish") {
		t.Error("'Publish' button not found")
	}
	if !contains(body, "Add Text Block") {
		t.Error("'Add Text Block' button not found")
	}

	// Check for block content
	if !contains(body, "Text Block") {
		t.Error("Block type label not found")
	}

	// Check for action buttons
	if !contains(body, "Edit") {
		t.Error("'Edit' button not found")
	}
	if !contains(body, "Delete") {
		t.Error("'Delete' button not found")
	}

	// Check for move buttons
	if !contains(body, "↑") {
		t.Error("Move up button not found")
	}
	if !contains(body, "↓") {
		t.Error("Move down button not found")
	}

	// Check for back link
	if !contains(body, "Back to Dashboard") {
		t.Error("'Back to Dashboard' link not found")
	}

	// Check that forms POST to correct endpoints
	if !contains(body, "/admin/pages/1") {
		t.Error("Save form action not found")
	}
	if !contains(body, "/admin/pages/1/publish") {
		t.Error("Publish form action not found")
	}
	if !contains(body, "/admin/pages/1/blocks") {
		t.Error("Add block form action not found")
	}
}

func TestEditPageHandler_EmptyBlocks(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create page with no blocks
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/",
		Title:     "Empty Page",
		Published: false,
	}
	testDB.Create(page)

	// Create request
	req := httptest.NewRequest("GET", "/admin/pages/1/edit", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	EditPageHandler(c)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Should show empty state message
	if !contains(body, "No blocks yet") {
		t.Error("Empty state message not found")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Tests for PublishPageHandler

func TestPublishPageHandler_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create unpublished page
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/test",
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/pages/1/publish", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	PublishPageHandler(c)

	// Assert redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d", c.Writer.Status())
	}
	location := w.Header().Get("Location")
	if location != "/admin/pages/1/edit" {
		t.Errorf("Expected redirect to /admin/pages/1/edit, got %s", location)
	}

	// Verify page is now published
	var updatedPage models.Page
	testDB.First(&updatedPage, 1)
	if !updatedPage.Published {
		t.Error("Expected page.Published to be true")
	}
}

func TestPublishPageHandler_SecurityCheck(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create two test sites
	site1 := &models.Site{
		ID:        1,
		Subdomain: "site1",
		OwnerID:   1,
		SiteDir:   "/tmp/site1",
	}
	testDB.Create(site1)

	site2 := &models.Site{
		ID:        2,
		Subdomain: "site2",
		OwnerID:   1,
		SiteDir:   "/tmp/site2",
	}
	testDB.Create(site2)

	// Create page for site1
	page := &models.Page{
		SiteID:    site1.ID,
		Slug:      "/test",
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Try to publish from site2
	req := httptest.NewRequest("POST", "/admin/pages/1/publish", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site2) // Different site!

	// Execute
	PublishPageHandler(c)

	// Assert forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify page is still unpublished
	var unchangedPage models.Page
	testDB.First(&unchangedPage, 1)
	if unchangedPage.Published {
		t.Error("Expected page.Published to remain false")
	}
}

func TestPublishPageHandler_PageNotFound(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Try to publish non-existent page
	req := httptest.NewRequest("POST", "/admin/pages/999/publish", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	c.Set("site", site)

	// Execute
	PublishPageHandler(c)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
	if w.Body.String() != "Page not found" {
		t.Errorf("Expected 'Page not found', got %s", w.Body.String())
	}
}

// Tests for UpdatePageHandler

func TestUpdatePageHandler_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create page
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/test",
		Title:     "Old Title",
		Published: true,
	}
	testDB.Create(page)

	// Create request with form data
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("title", "New Title")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	UpdatePageHandler(c)

	// Assert redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d", c.Writer.Status())
	}
	location := w.Header().Get("Location")
	if location != "/admin/pages/1/edit" {
		t.Errorf("Expected redirect to /admin/pages/1/edit, got %s", location)
	}

	// Verify title was updated and Published status unchanged
	var updatedPage models.Page
	testDB.First(&updatedPage, 1)
	if updatedPage.Title != "New Title" {
		t.Errorf("Expected title 'New Title', got %s", updatedPage.Title)
	}
	if !updatedPage.Published {
		t.Error("Expected page.Published to remain true (unchanged)")
	}
}

func TestUpdatePageHandler_SecurityCheck(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create two test sites
	site1 := &models.Site{
		ID:        1,
		Subdomain: "site1",
		OwnerID:   1,
		SiteDir:   "/tmp/site1",
	}
	testDB.Create(site1)

	site2 := &models.Site{
		ID:        2,
		Subdomain: "site2",
		OwnerID:   1,
		SiteDir:   "/tmp/site2",
	}
	testDB.Create(site2)

	// Create page for site1
	page := &models.Page{
		SiteID:    site1.ID,
		Slug:      "/test",
		Title:     "Original Title",
		Published: false,
	}
	testDB.Create(page)

	// Try to update from site2
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("title", "Hacked Title")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site2) // Different site!

	// Execute
	UpdatePageHandler(c)

	// Assert forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify title was not changed
	var unchangedPage models.Page
	testDB.First(&unchangedPage, 1)
	if unchangedPage.Title != "Original Title" {
		t.Errorf("Expected title to remain 'Original Title', got %s", unchangedPage.Title)
	}
}

func TestUpdatePageHandler_EmptyTitle(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create page
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/test",
		Title:     "Original Title",
		Published: false,
	}
	testDB.Create(page)

	// Try to update with empty title
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("title", "")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	UpdatePageHandler(c)

	// Assert bad request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
	if w.Body.String() != "Title is required" {
		t.Errorf("Expected 'Title is required', got %s", w.Body.String())
	}

	// Verify title was not changed
	var unchangedPage models.Page
	testDB.First(&unchangedPage, 1)
	if unchangedPage.Title != "Original Title" {
		t.Errorf("Expected title to remain 'Original Title', got %s", unchangedPage.Title)
	}
}

// Tests for UnpublishPageHandler

func TestUnpublishPageHandler_Success(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create test site
	site := &models.Site{
		ID:        1,
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}
	testDB.Create(site)

	// Create published page
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/test",
		Title:     "Test Page",
		Published: true,
	}
	testDB.Create(page)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/pages/1/unpublish", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	UnpublishPageHandler(c)

	// Assert redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d", c.Writer.Status())
	}
	location := w.Header().Get("Location")
	if location != "/admin/pages/1/edit" {
		t.Errorf("Expected redirect to /admin/pages/1/edit, got %s", location)
	}

	// Verify page is now unpublished
	var updatedPage models.Page
	testDB.First(&updatedPage, 1)
	if updatedPage.Published {
		t.Error("Expected page.Published to be false")
	}
}

func TestUnpublishPageHandler_SecurityCheck(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	testDB := setupTestDB(t)
	db.SetDB(testDB)

	// Create two test sites
	site1 := &models.Site{
		ID:        1,
		Subdomain: "site1",
		OwnerID:   1,
		SiteDir:   "/tmp/site1",
	}
	testDB.Create(site1)

	site2 := &models.Site{
		ID:        2,
		Subdomain: "site2",
		OwnerID:   1,
		SiteDir:   "/tmp/site2",
	}
	testDB.Create(site2)

	// Create published page for site1
	page := &models.Page{
		SiteID:    site1.ID,
		Slug:      "/test",
		Title:     "Test Page",
		Published: true,
	}
	testDB.Create(page)

	// Try to unpublish from site2
	req := httptest.NewRequest("POST", "/admin/pages/1/unpublish", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site2) // Different site!

	// Execute
	UnpublishPageHandler(c)

	// Assert forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify page is still published
	var unchangedPage models.Page
	testDB.First(&unchangedPage, 1)
	if !unchangedPage.Published {
		t.Error("Expected page.Published to remain true")
	}
}
