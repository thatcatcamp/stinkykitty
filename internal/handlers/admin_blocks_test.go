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
)

func TestCreateBlockHandler_Success(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create POST request with form data
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("type", "text")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	CreateBlockHandler(c)

	// Assert redirect - check using c.Writer.Status() like other tests
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d. Body: %s", c.Writer.Status(), w.Body.String())
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "/admin/pages/1/blocks/") || !strings.HasSuffix(location, "/edit") {
		t.Errorf("Expected redirect to /admin/pages/1/blocks/:id/edit, got %s", location)
	}

	// Verify block was created in database
	var block models.Block
	result := testDB.Where("page_id = ?", 1).First(&block)
	if result.Error != nil {
		t.Errorf("Block was not created in database: %v", result.Error)
	}

	if block.Type != "text" {
		t.Errorf("Expected block type 'text', got %s", block.Type)
	}

	if block.Order != 0 {
		t.Errorf("Expected block order 0, got %d", block.Order)
	}

	if block.Data != `{"content":""}` {
		t.Errorf("Expected block data '{\"content\":\"\"}', got %s", block.Data)
	}
}

func TestCreateBlockHandler_InvalidPageID(t *testing.T) {
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

	// Create POST request with invalid page ID
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("type", "text")

	c.Request = httptest.NewRequest("POST", "/admin/pages/invalid/blocks", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	c.Set("site", site)

	// Execute
	CreateBlockHandler(c)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Body.String() != "Invalid page ID" {
		t.Errorf("Expected 'Invalid page ID', got %s", w.Body.String())
	}
}

func TestCreateBlockHandler_PageNotFound(t *testing.T) {
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

	// Create POST request for non-existent page
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("type", "text")

	c.Request = httptest.NewRequest("POST", "/admin/pages/999/blocks", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	c.Set("site", site)

	// Execute
	CreateBlockHandler(c)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	if w.Body.String() != "Page not found" {
		t.Errorf("Expected 'Page not found', got %s", w.Body.String())
	}
}

func TestCreateBlockHandler_PageFromDifferentSite(t *testing.T) {
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
		Title:     "Site1 Page",
		Published: false,
	}
	testDB.Create(page)

	// Try to create block from site2
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("type", "text")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site2) // Different site!

	// Execute
	CreateBlockHandler(c)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify no block was created
	var count int64
	testDB.Model(&models.Block{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 blocks to be created, got %d", count)
	}
}

func TestCreateBlockHandler_OrderCalculation(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Test 1: First block should have order 0
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("type", "text")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	CreateBlockHandler(c)

	var firstBlock models.Block
	testDB.Where("page_id = ?", 1).Order("id ASC").First(&firstBlock)
	if firstBlock.Order != 0 {
		t.Errorf("First block should have order 0, got %d", firstBlock.Order)
	}

	// Test 2: Second block should have order 1
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)

	form2 := url.Values{}
	form2.Add("type", "text")

	c2.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks", strings.NewReader(form2.Encode()))
	c2.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c2.Params = gin.Params{{Key: "id", Value: "1"}}
	c2.Set("site", site)

	CreateBlockHandler(c2)

	var secondBlock models.Block
	testDB.Where("page_id = ?", 1).Order("id DESC").First(&secondBlock)
	if secondBlock.Order != 1 {
		t.Errorf("Second block should have order 1, got %d", secondBlock.Order)
	}

	// Test 3: Third block should have order 2
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)

	form3 := url.Values{}
	form3.Add("type", "text")

	c3.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks", strings.NewReader(form3.Encode()))
	c3.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c3.Params = gin.Params{{Key: "id", Value: "1"}}
	c3.Set("site", site)

	CreateBlockHandler(c3)

	var thirdBlock models.Block
	testDB.Where("page_id = ?", 1).Order("id DESC").First(&thirdBlock)
	if thirdBlock.Order != 2 {
		t.Errorf("Third block should have order 2, got %d", thirdBlock.Order)
	}

	// Verify all three blocks exist
	var allBlocks []models.Block
	testDB.Where("page_id = ?", 1).Order("\"order\" ASC").Find(&allBlocks)
	if len(allBlocks) != 3 {
		t.Errorf("Expected 3 blocks, got %d", len(allBlocks))
	}

	// Verify orders are sequential
	for i, block := range allBlocks {
		if block.Order != i {
			t.Errorf("Block at index %d should have order %d, got %d", i, i, block.Order)
		}
	}
}

func TestCreateBlockHandler_InvalidBlockType(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create POST request with invalid block type
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("type", "invalid")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("site", site)

	// Execute
	CreateBlockHandler(c)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Body.String() != "Invalid block type" {
		t.Errorf("Expected 'Invalid block type', got %s", w.Body.String())
	}

	// Verify no block was created
	var count int64
	testDB.Model(&models.Block{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 blocks to be created, got %d", count)
	}
}

func TestEditBlockHandler_Success(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create block with content
	block := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Hello, World!"}`,
	}
	testDB.Create(block)

	// Create GET request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/admin/pages/1/blocks/1/edit", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site)

	// Execute
	EditBlockHandler(c)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that form is rendered with content
	body := w.Body.String()
	if !strings.Contains(body, "Edit Text Block") {
		t.Errorf("Expected 'Edit Text Block' in response, got %s", body)
	}

	if !strings.Contains(body, "Hello, World!") {
		t.Errorf("Expected 'Hello, World!' in response, got %s", body)
	}

	if !strings.Contains(body, `action="/admin/pages/1/blocks/1"`) {
		t.Errorf("Expected form action to be /admin/pages/1/blocks/1, got %s", body)
	}
}

func TestEditBlockHandler_BlockNotFound(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create GET request for non-existent block
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/admin/pages/1/blocks/999/edit", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "999"},
	}
	c.Set("site", site)

	// Execute
	EditBlockHandler(c)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	if w.Body.String() != "Block not found" {
		t.Errorf("Expected 'Block not found', got %s", w.Body.String())
	}
}

func TestEditBlockHandler_SecurityCheck(t *testing.T) {
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
		Title:     "Site1 Page",
		Published: false,
	}
	testDB.Create(page)

	// Create block for site1
	block := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Secret content"}`,
	}
	testDB.Create(block)

	// Try to edit block from site2
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/admin/pages/1/blocks/1/edit", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site2) // Different site!

	// Execute
	EditBlockHandler(c)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}
}

func TestUpdateBlockHandler_Success(t *testing.T) {
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

	// Create test user
	user := &models.User{
		ID:    1,
		Email: "test@example.com",
	}
	testDB.Create(user)

	// Create page
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/",
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create block with content
	block := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Old content"}`,
	}
	testDB.Create(block)

	// Create POST request with new content
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("content", "New content here!")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site)
	c.Set("user", user)

	// Execute
	UpdateBlockHandler(c)

	// Assert redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d", c.Writer.Status())
	}

	location := w.Header().Get("Location")
	if location != "/admin/pages/1/edit" {
		t.Errorf("Expected redirect to /admin/pages/1/edit, got %s", location)
	}

	// Verify block was updated in database
	var updatedBlock models.Block
	result := testDB.Where("id = ?", 1).First(&updatedBlock)
	if result.Error != nil {
		t.Errorf("Failed to load updated block: %v", result.Error)
	}

	if updatedBlock.Data != `{"content":"New content here!"}` {
		t.Errorf("Expected block data to be '{\"content\":\"New content here!\"}', got %s", updatedBlock.Data)
	}
}

func TestUpdateBlockHandler_EmptyContent(t *testing.T) {
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

	// Create test user
	user := &models.User{
		ID:    1,
		Email: "test@example.com",
	}
	testDB.Create(user)

	// Create page
	page := &models.Page{
		SiteID:    site.ID,
		Slug:      "/",
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create block with content
	block := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Old content"}`,
	}
	testDB.Create(block)

	// Create POST request with empty content
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("content", "")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site)
	c.Set("user", user)

	// Execute
	UpdateBlockHandler(c)

	// Assert redirect (should still succeed)
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d", c.Writer.Status())
	}

	// Verify block was updated with empty content
	var updatedBlock models.Block
	result := testDB.Where("id = ?", 1).First(&updatedBlock)
	if result.Error != nil {
		t.Errorf("Failed to load updated block: %v", result.Error)
	}

	if updatedBlock.Data != `{"content":""}` {
		t.Errorf("Expected block data to be '{\"content\":\"\"}', got %s", updatedBlock.Data)
	}
}

func TestUpdateBlockHandler_SecurityCheck(t *testing.T) {
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

	// Create test user
	user := &models.User{
		ID:    1,
		Email: "test@example.com",
	}
	testDB.Create(user)

	// Create page for site1
	page := &models.Page{
		SiteID:    site1.ID,
		Slug:      "/",
		Title:     "Site1 Page",
		Published: false,
	}
	testDB.Create(page)

	// Create block for site1
	block := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Original content"}`,
	}
	testDB.Create(block)

	// Try to update block from site2
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("content", "Hacked content!")

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site2) // Different site!
	c.Set("user", user)

	// Execute
	UpdateBlockHandler(c)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify block was NOT updated
	var unchangedBlock models.Block
	result := testDB.Where("id = ?", 1).First(&unchangedBlock)
	if result.Error != nil {
		t.Errorf("Failed to load block: %v", result.Error)
	}

	if unchangedBlock.Data != `{"content":"Original content"}` {
		t.Errorf("Block should not have been updated, got %s", unchangedBlock.Data)
	}
}

func TestDeleteBlockHandler_Success(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create block to delete
	block := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Content to delete"}`,
	}
	testDB.Create(block)

	// Verify block exists
	var countBefore int64
	testDB.Model(&models.Block{}).Where("page_id = ?", page.ID).Count(&countBefore)
	if countBefore != 1 {
		t.Errorf("Expected 1 block before deletion, got %d", countBefore)
	}

	// Create POST request to delete
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1/delete", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site)

	// Execute
	DeleteBlockHandler(c)

	// Assert redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d. Body: %s", c.Writer.Status(), w.Body.String())
	}

	location := w.Header().Get("Location")
	if location != "/admin/pages/1/edit" {
		t.Errorf("Expected redirect to /admin/pages/1/edit, got %s", location)
	}

	// Verify block was deleted from database
	var countAfter int64
	testDB.Model(&models.Block{}).Where("page_id = ?", page.ID).Count(&countAfter)
	if countAfter != 0 {
		t.Errorf("Expected 0 blocks after deletion, got %d", countAfter)
	}

	// Verify block cannot be found
	var deletedBlock models.Block
	result := testDB.Where("id = ?", block.ID).First(&deletedBlock)
	if result.Error == nil {
		t.Errorf("Block should not be found after deletion, but it was")
	}
}

func TestDeleteBlockHandler_BlockNotFound(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Try to delete non-existent block
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/999/delete", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "999"},
	}
	c.Set("site", site)

	// Execute
	DeleteBlockHandler(c)

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	if w.Body.String() != "Block not found" {
		t.Errorf("Expected 'Block not found', got %s", w.Body.String())
	}
}

func TestDeleteBlockHandler_SecurityCheck(t *testing.T) {
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
		Title:     "Site1 Page",
		Published: false,
	}
	testDB.Create(page)

	// Create block for site1
	block := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Protected content"}`,
	}
	testDB.Create(block)

	// Try to delete block from site2
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1/delete", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site2) // Different site!

	// Execute
	DeleteBlockHandler(c)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify block was NOT deleted
	var stillExists models.Block
	result := testDB.Where("id = ?", block.ID).First(&stillExists)
	if result.Error != nil {
		t.Errorf("Block should still exist after failed deletion attempt, but got error: %v", result.Error)
	}

	if stillExists.Data != `{"content":"Protected content"}` {
		t.Errorf("Block data should be unchanged, got %s", stillExists.Data)
	}
}

func TestDeleteBlockHandler_InvalidPageID(t *testing.T) {
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

	// Try to delete with invalid page ID
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/invalid/blocks/1/delete", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "invalid"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site)

	// Execute
	DeleteBlockHandler(c)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Body.String() != "Invalid page ID" {
		t.Errorf("Expected 'Invalid page ID', got %s", w.Body.String())
	}
}

func TestDeleteBlockHandler_InvalidBlockID(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Try to delete with invalid block ID
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/invalid/delete", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "invalid"},
	}
	c.Set("site", site)

	// Execute
	DeleteBlockHandler(c)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Body.String() != "Invalid block ID" {
		t.Errorf("Expected 'Invalid block ID', got %s", w.Body.String())
	}
}

func TestMoveBlockUpHandler_Success(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create three blocks
	block1 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Block 1"}`,
	}
	testDB.Create(block1)

	block2 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  1,
		Data:   `{"content":"Block 2"}`,
	}
	testDB.Create(block2)

	block3 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  2,
		Data:   `{"content":"Block 3"}`,
	}
	testDB.Create(block3)

	// Move block2 (order 1) up - should swap with block1 (order 0)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/2/move-up", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "2"},
	}
	c.Set("site", site)

	// Execute
	MoveBlockUpHandler(c)

	// Assert redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d. Body: %s", c.Writer.Status(), w.Body.String())
	}

	location := w.Header().Get("Location")
	if location != "/admin/pages/1/edit" {
		t.Errorf("Expected redirect to /admin/pages/1/edit, got %s", location)
	}

	// Verify order swap occurred
	var updatedBlock1 models.Block
	testDB.Where("id = ?", block1.ID).First(&updatedBlock1)
	if updatedBlock1.Order != 1 {
		t.Errorf("Block 1 should now have order 1, got %d", updatedBlock1.Order)
	}

	var updatedBlock2 models.Block
	testDB.Where("id = ?", block2.ID).First(&updatedBlock2)
	if updatedBlock2.Order != 0 {
		t.Errorf("Block 2 should now have order 0, got %d", updatedBlock2.Order)
	}

	var updatedBlock3 models.Block
	testDB.Where("id = ?", block3.ID).First(&updatedBlock3)
	if updatedBlock3.Order != 2 {
		t.Errorf("Block 3 should still have order 2, got %d", updatedBlock3.Order)
	}
}

func TestMoveBlockUpHandler_FirstBlock(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create two blocks
	block1 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Block 1"}`,
	}
	testDB.Create(block1)

	block2 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  1,
		Data:   `{"content":"Block 2"}`,
	}
	testDB.Create(block2)

	// Try to move block1 (order 0) up - should do nothing
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1/move-up", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site)

	// Execute
	MoveBlockUpHandler(c)

	// Assert redirect (should still redirect without error)
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d. Body: %s", c.Writer.Status(), w.Body.String())
	}

	// Verify orders unchanged
	var updatedBlock1 models.Block
	testDB.Where("id = ?", block1.ID).First(&updatedBlock1)
	if updatedBlock1.Order != 0 {
		t.Errorf("Block 1 should still have order 0, got %d", updatedBlock1.Order)
	}

	var updatedBlock2 models.Block
	testDB.Where("id = ?", block2.ID).First(&updatedBlock2)
	if updatedBlock2.Order != 1 {
		t.Errorf("Block 2 should still have order 1, got %d", updatedBlock2.Order)
	}
}

func TestMoveBlockUpHandler_SecurityCheck(t *testing.T) {
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
		Title:     "Site1 Page",
		Published: false,
	}
	testDB.Create(page)

	// Create blocks for site1
	block1 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Block 1"}`,
	}
	testDB.Create(block1)

	block2 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  1,
		Data:   `{"content":"Block 2"}`,
	}
	testDB.Create(block2)

	// Try to move block from site2
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/2/move-up", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "2"},
	}
	c.Set("site", site2) // Different site!

	// Execute
	MoveBlockUpHandler(c)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify orders unchanged
	var unchangedBlock1 models.Block
	testDB.Where("id = ?", block1.ID).First(&unchangedBlock1)
	if unchangedBlock1.Order != 0 {
		t.Errorf("Block 1 should still have order 0, got %d", unchangedBlock1.Order)
	}

	var unchangedBlock2 models.Block
	testDB.Where("id = ?", block2.ID).First(&unchangedBlock2)
	if unchangedBlock2.Order != 1 {
		t.Errorf("Block 2 should still have order 1, got %d", unchangedBlock2.Order)
	}
}

func TestMoveBlockDownHandler_Success(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create three blocks
	block1 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Block 1"}`,
	}
	testDB.Create(block1)

	block2 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  1,
		Data:   `{"content":"Block 2"}`,
	}
	testDB.Create(block2)

	block3 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  2,
		Data:   `{"content":"Block 3"}`,
	}
	testDB.Create(block3)

	// Move block2 (order 1) down - should swap with block3 (order 2)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/2/move-down", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "2"},
	}
	c.Set("site", site)

	// Execute
	MoveBlockDownHandler(c)

	// Assert redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d. Body: %s", c.Writer.Status(), w.Body.String())
	}

	location := w.Header().Get("Location")
	if location != "/admin/pages/1/edit" {
		t.Errorf("Expected redirect to /admin/pages/1/edit, got %s", location)
	}

	// Verify order swap occurred
	var updatedBlock1 models.Block
	testDB.Where("id = ?", block1.ID).First(&updatedBlock1)
	if updatedBlock1.Order != 0 {
		t.Errorf("Block 1 should still have order 0, got %d", updatedBlock1.Order)
	}

	var updatedBlock2 models.Block
	testDB.Where("id = ?", block2.ID).First(&updatedBlock2)
	if updatedBlock2.Order != 2 {
		t.Errorf("Block 2 should now have order 2, got %d", updatedBlock2.Order)
	}

	var updatedBlock3 models.Block
	testDB.Where("id = ?", block3.ID).First(&updatedBlock3)
	if updatedBlock3.Order != 1 {
		t.Errorf("Block 3 should now have order 1, got %d", updatedBlock3.Order)
	}
}

func TestMoveBlockDownHandler_LastBlock(t *testing.T) {
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
		Title:     "Test Page",
		Published: false,
	}
	testDB.Create(page)

	// Create two blocks
	block1 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Block 1"}`,
	}
	testDB.Create(block1)

	block2 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  1,
		Data:   `{"content":"Block 2"}`,
	}
	testDB.Create(block2)

	// Try to move block2 (order 1) down - should do nothing
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/2/move-down", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "2"},
	}
	c.Set("site", site)

	// Execute
	MoveBlockDownHandler(c)

	// Assert redirect (should still redirect without error)
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d. Body: %s", c.Writer.Status(), w.Body.String())
	}

	// Verify orders unchanged
	var updatedBlock1 models.Block
	testDB.Where("id = ?", block1.ID).First(&updatedBlock1)
	if updatedBlock1.Order != 0 {
		t.Errorf("Block 1 should still have order 0, got %d", updatedBlock1.Order)
	}

	var updatedBlock2 models.Block
	testDB.Where("id = ?", block2.ID).First(&updatedBlock2)
	if updatedBlock2.Order != 1 {
		t.Errorf("Block 2 should still have order 1, got %d", updatedBlock2.Order)
	}
}

func TestMoveBlockDownHandler_SecurityCheck(t *testing.T) {
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
		Title:     "Site1 Page",
		Published: false,
	}
	testDB.Create(page)

	// Create blocks for site1
	block1 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Block 1"}`,
	}
	testDB.Create(block1)

	block2 := &models.Block{
		PageID: page.ID,
		Type:   "text",
		Order:  1,
		Data:   `{"content":"Block 2"}`,
	}
	testDB.Create(block2)

	// Try to move block from site2
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1/move-down", nil)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "block_id", Value: "1"},
	}
	c.Set("site", site2) // Different site!

	// Execute
	MoveBlockDownHandler(c)

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	if w.Body.String() != "Access denied" {
		t.Errorf("Expected 'Access denied', got %s", w.Body.String())
	}

	// Verify orders unchanged
	var unchangedBlock1 models.Block
	testDB.Where("id = ?", block1.ID).First(&unchangedBlock1)
	if unchangedBlock1.Order != 0 {
		t.Errorf("Block 1 should still have order 0, got %d", unchangedBlock1.Order)
	}

	var unchangedBlock2 models.Block
	testDB.Where("id = ?", block2.ID).First(&unchangedBlock2)
	if unchangedBlock2.Order != 1 {
		t.Errorf("Block 2 should still have order 1, got %d", unchangedBlock2.Order)
	}
}
