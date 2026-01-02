// SPDX-License-Identifier: MIT
package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/search"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMediaTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	err = database.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{}, &models.MediaItem{}, &models.MediaTag{}, &models.Page{}, &models.Block{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Try to init FTS index, but don't fail if FTS5 is missing in the test environment
	_ = search.InitFTSIndex(database)

	return database
}

func TestMediaUploadHandler_CentralizedStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupMediaTestDB(t)
	db.SetDB(database)

	// Create temporary directory for centralized media storage
	tempMediaDir := t.TempDir()

	// Initialize config with test values
	tempConfigDir := t.TempDir()
	configPath := filepath.Join(tempConfigDir, "config.yaml")
	if err := config.InitConfig(configPath); err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}
	config.Set("storage.media_dir", tempMediaDir)

	// Create user
	passwordHash, _ := auth.HashPassword("test-password")
	user := models.User{
		Email:        "test@example.com",
		PasswordHash: passwordHash,
	}
	database.Create(&user)

	// Create site (with its own site dir, which should NOT be used for media)
	tempSiteDir := t.TempDir()
	site := models.Site{
		Subdomain: "testsite",
		OwnerID:   user.ID,
		SiteDir:   tempSiteDir,
	}
	database.Create(&site)

	// Create a test image file in memory
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a simple 1x1 PNG image (valid PNG magic bytes)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, // IEND chunk
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	part, err := writer.CreateFormFile("images", "test-image.png")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(pngData)); err != nil {
		t.Fatalf("Failed to write image data: %v", err)
	}
	writer.Close()

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("POST", "/admin/media/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("site", &site)
	c.Set("user", &user)

	// Call the handler
	MediaUploadHandler(c)

	// Assert response is successful
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
	}

	// Verify MediaItem was created in database
	var mediaItems []models.MediaItem
	database.Find(&mediaItems)

	if len(mediaItems) != 1 {
		t.Fatalf("Expected 1 media item, found %d", len(mediaItems))
	}

	mediaItem := mediaItems[0]

	// CRITICAL: Verify UploadedFromSiteID is set (NOT SiteID)
	if mediaItem.UploadedFromSiteID == nil {
		t.Error("Expected UploadedFromSiteID to be set, but it was nil")
	} else if *mediaItem.UploadedFromSiteID != site.ID {
		t.Errorf("Expected UploadedFromSiteID to be %d, got %d", site.ID, *mediaItem.UploadedFromSiteID)
	}

	// Verify SiteID is NOT set (media is centralized, not site-specific)
	if mediaItem.SiteID != 0 {
		t.Errorf("Expected SiteID to be 0 (not set), got %d", mediaItem.SiteID)
	}

	// Verify UploadedBy is set
	if mediaItem.UploadedBy != user.ID {
		t.Errorf("Expected UploadedBy to be %d, got %d", user.ID, mediaItem.UploadedBy)
	}

	// Verify original filename
	if mediaItem.OriginalName != "test-image.png" {
		t.Errorf("Expected OriginalName to be 'test-image.png', got '%s'", mediaItem.OriginalName)
	}

	// Verify file exists in CENTRALIZED location (not site-specific)
	centralizedPath := filepath.Join(tempMediaDir, mediaItem.Filename)
	if _, err := os.Stat(centralizedPath); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at centralized location %s, but it doesn't", centralizedPath)
	}

	// Verify file does NOT exist in site-specific location
	siteSpecificPath := filepath.Join(tempSiteDir, "uploads", mediaItem.Filename)
	if _, err := os.Stat(siteSpecificPath); !os.IsNotExist(err) {
		t.Errorf("File should NOT exist at site-specific location %s, but it does", siteSpecificPath)
	}
}
func TestUpdateBlockHandler_ImageCentralized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupMediaTestDB(t)
	db.SetDB(database)

	tempMediaDir := t.TempDir()
	tempConfigDir := t.TempDir()
	configPath := filepath.Join(tempConfigDir, "config.yaml")
	config.InitConfig(configPath)
	config.Set("storage.media_dir", tempMediaDir)

	passwordHash, _ := auth.HashPassword("test-password")
	user := models.User{Email: "test@example.com", PasswordHash: passwordHash}
	database.Create(&user)

	site := models.Site{Subdomain: "testsite", OwnerID: user.ID, SiteDir: t.TempDir()}
	database.Create(&site)

	page := models.Page{SiteID: site.ID, Title: "Test Page"}
	database.Create(&page)

	block := models.Block{PageID: page.ID, Type: "image", Data: `{"url": "", "alt": ""}`}
	database.Create(&block)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE}
	part, _ := writer.CreateFormFile("image", "block-image.png")
	io.Copy(part, bytes.NewReader(pngData))
	writer.WriteField("alt", "New Alt Text")
	writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/pages/1/blocks/1", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = []gin.Param{{Key: "id", Value: "1"}, {Key: "block_id", Value: "1"}}
	c.Set("site", &site)
	c.Set("user", &user)

	UpdateBlockHandler(c)

	// Note: We expect 200 instead of 302 here because InitFTSIndex fails in the test environment (no FTS5),
	// causing the handler to return early before the redirect.
	if w.Code != http.StatusOK && w.Code != http.StatusFound {
		t.Errorf("Expected status 200 or 302, got %d. Response: %s", w.Code, w.Body.String())
	}

	var updatedBlock models.Block
	database.First(&updatedBlock, block.ID)
	var imageData map[string]interface{}
	json.Unmarshal([]byte(updatedBlock.Data), &imageData)

	url, _ := imageData["url"].(string)
	if !bytes.HasPrefix([]byte(url), []byte("/assets/")) {
		t.Errorf("Expected URL to start with /assets/, got %s", url)
	}

	filename := filepath.Base(url)
	centralizedPath := filepath.Join(tempMediaDir, "uploads", filename)
	if _, err := os.Stat(centralizedPath); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s, but it doesn't", centralizedPath)
	}
}
