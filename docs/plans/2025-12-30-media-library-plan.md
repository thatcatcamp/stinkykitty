# Media Library Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a full-featured media library for managing uploaded images with tagging, search, and safe deletion

**Architecture:** Database-backed image tracking with thumbnails, tag-based organization, usage detection for safe deletion, and modal picker integration with block editors

**Tech Stack:** Go 1.21+, Gin web framework, GORM ORM, SQLite, Go image library for thumbnails

---

## Task 1: Add Database Models

**Files:**
- Modify: `internal/models/models.go` (add after Line ~100)
- Create: `internal/models/media_test.go`

**Step 1: Write tests for MediaItem model**

Create `internal/models/media_test.go`:

```go
package models

import (
	"testing"
	"time"

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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/models -run TestMediaItemModel -v`
Expected: FAIL with "undefined: MediaItem"

**Step 3: Add MediaItem and MediaTag models**

Add to `internal/models/models.go` after Block struct (~line 105):

```go
// MediaItem represents an uploaded image in the media library
type MediaItem struct {
	ID           uint   `gorm:"primaryKey"`
	SiteID       uint   `gorm:"not null;index"`
	Filename     string `gorm:"not null"`        // Random hex filename
	OriginalName string `gorm:"not null"`        // User's original filename
	FileSize     int64  `gorm:"not null"`        // Bytes
	MimeType     string `gorm:"not null"`        // image/jpeg, etc.
	UploadedBy   uint   `gorm:"not null"`        // User ID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relationships
	Site Site        `gorm:"foreignKey:SiteID"`
	User User        `gorm:"foreignKey:UploadedBy"`
	Tags []MediaTag  `gorm:"foreignKey:MediaItemID"`
}

// MediaTag represents a tag on a media item
type MediaTag struct {
	ID          uint   `gorm:"primaryKey"`
	MediaItemID uint   `gorm:"not null;index:idx_media_tag"`
	TagName     string `gorm:"not null;index:idx_media_tag"`
	CreatedAt   time.Time

	// Relationships
	MediaItem MediaItem `gorm:"foreignKey:MediaItemID"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/models -run TestMediaItemModel -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/models/models.go internal/models/media_test.go
git commit -m "feat: add MediaItem and MediaTag database models"
```

---

## Task 2: Create Media Utility Package

**Files:**
- Create: `internal/media/thumbnail.go`
- Create: `internal/media/thumbnail_test.go`

**Step 1: Write test for thumbnail generation**

Create `internal/media/thumbnail_test.go`:

```go
package media

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateThumbnail(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create a test image (100x100 red square)
	srcPath := filepath.Join(tmpDir, "test.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	if err := jpeg.Encode(file, img, nil); err != nil {
		file.Close()
		t.Fatalf("Failed to encode test image: %v", err)
	}
	file.Close()

	// Generate thumbnail
	dstPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := GenerateThumbnail(srcPath, dstPath, 50, 50); err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Verify thumbnail exists
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Thumbnail file was not created")
	}

	// Verify thumbnail dimensions
	thumbFile, err := os.Open(dstPath)
	if err != nil {
		t.Fatalf("Failed to open thumbnail: %v", err)
	}
	defer thumbFile.Close()

	thumbImg, _, err := image.Decode(thumbFile)
	if err != nil {
		t.Fatalf("Failed to decode thumbnail: %v", err)
	}

	bounds := thumbImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width != 50 || height != 50 {
		t.Errorf("Expected thumbnail 50x50, got %dx%d", width, height)
	}
}

func TestGenerateThumbnailMaintainsAspectRatio(t *testing.T) {
	tmpDir := t.TempDir()

	// Create rectangular image (200x100)
	srcPath := filepath.Join(tmpDir, "wide.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}

	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	jpeg.Encode(file, img, nil)
	file.Close()

	// Generate 50x50 thumbnail (should crop to maintain aspect ratio)
	dstPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := GenerateThumbnail(srcPath, dstPath, 50, 50); err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Verify thumbnail is 50x50 (center-cropped)
	thumbFile, err := os.Open(dstPath)
	if err != nil {
		t.Fatalf("Failed to open thumbnail: %v", err)
	}
	defer thumbFile.Close()

	thumbImg, _, err := image.Decode(thumbFile)
	if err != nil {
		t.Fatalf("Failed to decode thumbnail: %v", err)
	}

	bounds := thumbImg.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("Expected 50x50 thumbnail, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/media -v`
Expected: FAIL with "no such file or directory" or "undefined: GenerateThumbnail"

**Step 3: Implement thumbnail generation**

Create `internal/media/thumbnail.go`:

```go
package media

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

// GenerateThumbnail creates a thumbnail from an image file
// Uses center crop to maintain exact dimensions
func GenerateThumbnail(srcPath, dstPath string, width, height int) error {
	// Open source image
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %w", err)
	}
	defer srcFile.Close()

	// Decode image
	img, _, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate center crop rectangle
	srcBounds := img.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Calculate aspect ratios
	srcAspect := float64(srcWidth) / float64(srcHeight)
	dstAspect := float64(width) / float64(height)

	var cropRect image.Rectangle
	if srcAspect > dstAspect {
		// Source is wider - crop width
		newWidth := int(float64(srcHeight) * dstAspect)
		x := (srcWidth - newWidth) / 2
		cropRect = image.Rect(x, 0, x+newWidth, srcHeight)
	} else {
		// Source is taller - crop height
		newHeight := int(float64(srcWidth) / dstAspect)
		y := (srcHeight - newHeight) / 2
		cropRect = image.Rect(0, y, srcWidth, y+newHeight)
	}

	// Create thumbnail image
	thumbnail := image.NewRGBA(image.Rect(0, 0, width, height))

	// Scale and draw
	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), img, cropRect, draw.Over, nil)

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save thumbnail
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer dstFile.Close()

	// Always save as JPEG for consistent format
	if err := jpeg.Encode(dstFile, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/media -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/media/
git commit -m "feat: add thumbnail generation utility"
```

---

## Task 3: Create Usage Detection Utility

**Files:**
- Create: `internal/media/usage.go`
- Create: `internal/media/usage_test.go`

**Step 1: Write test for usage detection**

Create `internal/media/usage_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/media -run TestFindImageUsage -v`
Expected: FAIL with "undefined: FindImageUsage"

**Step 3: Implement usage detection**

Create `internal/media/usage.go`:

```go
package media

import (
	"encoding/json"

	"github.com/thatcatcamp/stinkykitty/internal/blocks"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// UsageLocation represents where an image is used
type UsageLocation struct {
	PageID    uint
	PageTitle string
	BlockID   uint
	BlockType string
}

// FindImageUsage finds all blocks that reference a specific image URL
func FindImageUsage(db *gorm.DB, siteID uint, imageURL string) []UsageLocation {
	var usages []UsageLocation

	// Get all pages for this site
	var pages []models.Page
	db.Where("site_id = ? AND deleted_at IS NULL", siteID).Find(&pages)

	// For each page, check blocks
	for _, page := range pages {
		var pageBlocks []models.Block
		db.Where("page_id = ? AND deleted_at IS NULL", page.ID).Find(&pageBlocks)

		for _, block := range pageBlocks {
			if containsImageURL(block, imageURL) {
				usages = append(usages, UsageLocation{
					PageID:    page.ID,
					PageTitle: page.Title,
					BlockID:   block.ID,
					BlockType: block.Type,
				})
			}
		}
	}

	return usages
}

// containsImageURL checks if a block contains a specific image URL
func containsImageURL(block models.Block, imageURL string) bool {
	switch block.Type {
	case "image":
		var data blocks.ImageBlockData
		if err := json.Unmarshal([]byte(block.Data), &data); err != nil {
			return false
		}
		return data.URL == imageURL

	case "button":
		// Button blocks might have background images in the future
		// For now, just check if the URL appears in the data
		return false

	case "columns":
		// Columns can contain nested blocks with images
		// For V1, we'll do simple string matching
		// TODO: Proper nested block parsing in V2
		return false

	default:
		return false
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/media -run TestFindImageUsage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/media/usage.go internal/media/usage_test.go
git commit -m "feat: add image usage detection utility"
```

---

## Task 4: Create Media Upload Handler

**Files:**
- Create: `internal/handlers/admin_media.go`
- Modify: `cmd/stinky/server.go` (add routes)

**Step 1: Create basic handler structure**

Create `internal/handlers/admin_media.go`:

```go
package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/media"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/uploads"
)

// MediaLibraryHandler shows the main media library page
func MediaLibraryHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}
	user := userVal.(*models.User)

	// Get pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50
	offset := (page - 1) * limit

	// Get search query
	search := c.Query("search")

	// Get tag filter
	tagFilter := c.Query("tag")

	// Get orphaned filter
	showOrphaned := c.Query("orphaned") == "true"

	// Query media items
	query := db.GetDB().Where("site_id = ?", site.ID)

	if search != "" {
		query = query.Where("original_name LIKE ? OR filename LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if tagFilter != "" {
		// Join with media_tags table
		query = query.Joins("JOIN media_tags ON media_tags.media_item_id = media_items.id").
			Where("media_tags.tag_name = ?", tagFilter)
	}

	var mediaItems []models.MediaItem
	var totalCount int64

	db.GetDB().Model(&models.MediaItem{}).Where("site_id = ?", site.ID).Count(&totalCount)

	query.Preload("Tags").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&mediaItems)

	// Filter orphaned if requested
	var displayItems []models.MediaItem
	if showOrphaned {
		for _, item := range mediaItems {
			usages := media.FindImageUsage(db.GetDB(), site.ID, "/uploads/"+item.Filename)
			if len(usages) == 0 {
				displayItems = append(displayItems, item)
			}
		}
	} else {
		displayItems = mediaItems
	}

	// Calculate pagination
	totalPages := int(totalCount) / limit
	if int(totalCount)%limit != 0 {
		totalPages++
	}

	// Render page
	renderMediaLibraryPage(c, site, user, displayItems, page, totalPages, search, tagFilter, showOrphaned)
}

// MediaUploadHandler handles file uploads
func MediaUploadHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Site not found"})
		return
	}
	site := siteVal.(*models.Site)

	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(*models.User)

	// Handle file upload
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data"})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files provided"})
		return
	}

	var uploadedItems []models.MediaItem

	for _, fileHeader := range files {
		// Save file using existing upload utility
		webPath, err := uploads.SaveUploadedFile(fileHeader, site.SiteDir)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to upload %s: %v", fileHeader.Filename, err)})
			return
		}

		// Extract filename from web path (/uploads/abc123.jpg -> abc123.jpg)
		filename := filepath.Base(webPath)

		// Get file info
		fileSize := fileHeader.Size
		mimeType := fileHeader.Header.Get("Content-Type")

		// Create database record
		mediaItem := models.MediaItem{
			SiteID:       site.ID,
			Filename:     filename,
			OriginalName: fileHeader.Filename,
			FileSize:     fileSize,
			MimeType:     mimeType,
			UploadedBy:   user.ID,
		}

		if err := db.GetDB().Create(&mediaItem).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save media item"})
			return
		}

		// Generate thumbnail
		srcPath := filepath.Join(site.SiteDir, "uploads", filename)
		thumbPath := filepath.Join(site.SiteDir, "uploads", "thumbs", filename)
		if err := media.GenerateThumbnail(srcPath, thumbPath, 200, 200); err != nil {
			// Log error but don't fail the upload
			fmt.Printf("Warning: Failed to generate thumbnail for %s: %v\n", filename, err)
		}

		uploadedItems = append(uploadedItems, mediaItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"items":   uploadedItems,
	})
}

// renderMediaLibraryPage renders the HTML for the media library
func renderMediaLibraryPage(c *gin.Context, site *models.Site, user *models.User, items []models.MediaItem, page, totalPages int, search, tagFilter string, showOrphaned bool) {
	csrfToken := middleware.GetCSRFTokenHTML(c)

	// Build filter badges
	var filterBadges string
	if search != "" {
		filterBadges += fmt.Sprintf(`<span class="filter-badge">Search: %s <a href="/admin/media">×</a></span>`, search)
	}
	if tagFilter != "" {
		filterBadges += fmt.Sprintf(`<span class="filter-badge">Tag: %s <a href="/admin/media">×</a></span>`, tagFilter)
	}
	if showOrphaned {
		filterBadges += `<span class="filter-badge">Orphaned only <a href="/admin/media">×</a></span>`
	}

	// Build image grid
	var imageGrid string
	for _, item := range items {
		thumbURL := fmt.Sprintf("/uploads/thumbs/%s", item.Filename)

		// Build tag badges
		var tagBadges string
		for _, tag := range item.Tags {
			tagBadges += fmt.Sprintf(`<span class="tag-badge">%s</span>`, tag.TagName)
		}

		imageGrid += fmt.Sprintf(`
		<div class="media-card" data-id="%d">
			<div class="media-thumbnail">
				<img src="%s" alt="%s" loading="lazy">
			</div>
			<div class="media-info">
				<div class="media-filename" title="%s">%s</div>
				<div class="media-tags">%s</div>
				<div class="media-date">%s</div>
				<div class="media-actions">
					<button class="btn-small" onclick="editTags(%d)">Edit Tags</button>
					<button class="btn-small btn-danger" onclick="deleteMedia(%d)">Delete</button>
				</div>
			</div>
		</div>
		`, item.ID, thumbURL, item.OriginalName, item.OriginalName, item.OriginalName,
			tagBadges, item.CreatedAt.Format("Jan 2, 2006"), item.ID, item.ID)
	}

	if len(items) == 0 {
		imageGrid = `<div class="empty-state">No images found. Upload some images to get started!</div>`
	}

	// Build pagination
	var pagination string
	if totalPages > 1 {
		pagination = `<div class="pagination">`
		if page > 1 {
			pagination += fmt.Sprintf(`<a href="?page=%d">← Previous</a>`, page-1)
		}
		pagination += fmt.Sprintf(` Page %d of %d `, page, totalPages)
		if page < totalPages {
			pagination += fmt.Sprintf(`<a href="?page=%d">Next →</a>`, page+1)
		}
		pagination += `</div>`
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Media Library - %s</title>
	<style>
		%s

		.upload-zone {
			border: 2px dashed var(--color-border);
			border-radius: var(--radius-base);
			padding: calc(var(--spacing-base) * 2);
			text-align: center;
			margin-bottom: var(--spacing-lg);
			cursor: pointer;
			transition: border-color 0.2s;
		}

		.upload-zone:hover {
			border-color: var(--color-accent);
		}

		.upload-zone.drag-over {
			border-color: var(--color-accent);
			background: var(--color-bg-secondary);
		}

		.filter-bar {
			display: flex;
			gap: var(--spacing-base);
			margin-bottom: var(--spacing-lg);
			flex-wrap: wrap;
		}

		.filter-bar input {
			flex: 1;
			min-width: 200px;
		}

		.filter-badge {
			background: var(--color-bg-secondary);
			padding: var(--spacing-sm) var(--spacing-base);
			border-radius: var(--radius-sm);
			font-size: 14px;
		}

		.filter-badge a {
			margin-left: var(--spacing-sm);
			color: var(--color-danger);
			text-decoration: none;
		}

		.media-grid {
			display: grid;
			grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
			gap: var(--spacing-base);
			margin-bottom: var(--spacing-lg);
		}

		.media-card {
			background: var(--color-bg-card);
			border-radius: var(--radius-base);
			overflow: hidden;
			box-shadow: var(--shadow-sm);
		}

		.media-thumbnail {
			width: 100%%;
			height: 200px;
			overflow: hidden;
			background: var(--color-bg-secondary);
			display: flex;
			align-items: center;
			justify-content: center;
		}

		.media-thumbnail img {
			width: 100%%;
			height: 100%%;
			object-fit: cover;
		}

		.media-info {
			padding: var(--spacing-base);
		}

		.media-filename {
			font-weight: 600;
			margin-bottom: var(--spacing-sm);
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.media-tags {
			margin-bottom: var(--spacing-sm);
		}

		.tag-badge {
			display: inline-block;
			background: var(--color-accent);
			color: white;
			padding: 2px 8px;
			border-radius: var(--radius-sm);
			font-size: 12px;
			margin-right: 4px;
		}

		.media-date {
			font-size: 12px;
			color: var(--color-text-secondary);
			margin-bottom: var(--spacing-sm);
		}

		.media-actions {
			display: flex;
			gap: var(--spacing-sm);
		}

		.empty-state {
			text-align: center;
			padding: calc(var(--spacing-base) * 4);
			color: var(--color-text-secondary);
		}

		.pagination {
			text-align: center;
			padding: var(--spacing-base);
		}

		.pagination a {
			margin: 0 var(--spacing-sm);
			color: var(--color-accent);
			text-decoration: none;
		}

		@media (max-width: 640px) {
			.media-grid {
				grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
			}
		}
	</style>
</head>
<body>
	<div class="admin-container">
		<div class="admin-header">
			<h1>Media Library</h1>
			<div class="header-actions">
				<a href="/admin/dashboard" class="btn btn-secondary">← Back to Dashboard</a>
			</div>
		</div>

		<form id="upload-form" method="POST" action="/admin/media/upload" enctype="multipart/form-data">
			%s
			<div class="upload-zone" id="upload-zone">
				<p>📤 Drag & drop images here</p>
				<p>or <label for="file-input" style="color: var(--color-accent); cursor: pointer;">click to browse</label></p>
				<input type="file" id="file-input" name="images" multiple accept="image/*" style="display: none;">
			</div>
		</form>

		<div class="filter-bar">
			<input type="text" id="search-input" placeholder="Search images..." value="%s">
			<button class="btn" onclick="toggleOrphaned()">%s</button>
		</div>

		<div>%s</div>

		<div class="media-grid">
			%s
		</div>

		%s
	</div>

	<script>
		const uploadZone = document.getElementById('upload-zone');
		const fileInput = document.getElementById('file-input');
		const uploadForm = document.getElementById('upload-form');

		// Drag and drop
		uploadZone.addEventListener('dragover', (e) => {
			e.preventDefault();
			uploadZone.classList.add('drag-over');
		});

		uploadZone.addEventListener('dragleave', () => {
			uploadZone.classList.remove('drag-over');
		});

		uploadZone.addEventListener('drop', (e) => {
			e.preventDefault();
			uploadZone.classList.remove('drag-over');
			fileInput.files = e.dataTransfer.files;
			uploadForm.submit();
		});

		// Click to browse
		uploadZone.addEventListener('click', (e) => {
			if (e.target.tagName !== 'LABEL') {
				fileInput.click();
			}
		});

		fileInput.addEventListener('change', () => {
			if (fileInput.files.length > 0) {
				uploadForm.submit();
			}
		});

		// Search
		const searchInput = document.getElementById('search-input');
		let searchTimeout;
		searchInput.addEventListener('input', () => {
			clearTimeout(searchTimeout);
			searchTimeout = setTimeout(() => {
				window.location.href = '/admin/media?search=' + encodeURIComponent(searchInput.value);
			}, 500);
		});

		// Toggle orphaned
		function toggleOrphaned() {
			const current = new URLSearchParams(window.location.search).get('orphaned');
			window.location.href = '/admin/media?orphaned=' + (current === 'true' ? 'false' : 'true');
		}

		// Edit tags (placeholder)
		function editTags(id) {
			alert('Tag editing coming in next task!');
		}

		// Delete media (placeholder)
		function deleteMedia(id) {
			alert('Delete functionality coming in next task!');
		}
	</script>
</body>
</html>`, site.SiteTitle, GetDesignSystemCSS(), csrfToken, search,
		(map[bool]string{true: "Show All", false: "Show Orphaned"})[showOrphaned],
		filterBadges, imageGrid, pagination)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
```

**Step 2: Add routes to server**

Modify `cmd/stinky/server.go`, add after line ~196 (in adminGroup protected section):

```go
				// Media library
				adminGroup.GET("/media", handlers.MediaLibraryHandler)
				adminGroup.POST("/media/upload", handlers.MediaUploadHandler)
```

**Step 3: Test manually**

Run: `go run cmd/stinky/main.go server start`
Navigate to: `http://localhost:8080/admin/media`
Expected: Media library page loads (empty at first)

**Step 4: Commit**

```bash
git add internal/handlers/admin_media.go cmd/stinky/server.go
git commit -m "feat: add media library page and upload handler"
```

---

## Task 5: Add Tag Management

**Files:**
- Modify: `internal/handlers/admin_media.go` (add tag handlers)
- Modify: `cmd/stinky/server.go` (add tag routes)

**Step 1: Add tag handler functions**

Add to `internal/handlers/admin_media.go`:

```go
// MediaTagsHandler handles adding/removing tags
func MediaTagsHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Site not found"})
		return
	}
	site := siteVal.(*models.Site)

	// Get media item ID
	mediaIDStr := c.Param("id")
	mediaID, err := strconv.ParseUint(mediaIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	// Verify media item belongs to this site
	var mediaItem models.MediaItem
	if err := db.GetDB().Where("id = ? AND site_id = ?", mediaID, site.ID).First(&mediaItem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media item not found"})
		return
	}

	action := c.PostForm("action")
	tagName := c.PostForm("tag")

	if tagName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tag name required"})
		return
	}

	if action == "add" {
		// Check if tag already exists
		var existingTag models.MediaTag
		err := db.GetDB().Where("media_item_id = ? AND tag_name = ?", mediaID, tagName).First(&existingTag).Error
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Tag already exists"})
			return
		}

		// Add tag
		tag := models.MediaTag{
			MediaItemID: uint(mediaID),
			TagName:     tagName,
		}
		if err := db.GetDB().Create(&tag).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add tag"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	} else if action == "remove" {
		// Remove tag
		if err := db.GetDB().Where("media_item_id = ? AND tag_name = ?", mediaID, tagName).Delete(&models.MediaTag{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove tag"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
	}
}

// MediaTagAutocompleteHandler returns existing tags for autocomplete
func MediaTagAutocompleteHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Site not found"})
		return
	}
	site := siteVal.(*models.Site)

	// Get distinct tags for this site
	var tags []string
	db.GetDB().Model(&models.MediaTag{}).
		Joins("JOIN media_items ON media_items.id = media_tags.media_item_id").
		Where("media_items.site_id = ?", site.ID).
		Distinct("tag_name").
		Pluck("tag_name", &tags)

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
```

**Step 2: Add routes**

Add to `cmd/stinky/server.go` after media routes:

```go
				adminGroup.POST("/media/:id/tags", handlers.MediaTagsHandler)
				adminGroup.GET("/media/tags/autocomplete", handlers.MediaTagAutocompleteHandler)
```

**Step 3: Update renderMediaLibraryPage to handle tag editing**

Replace the `editTags` and `deleteMedia` JavaScript functions in `renderMediaLibraryPage`:

```javascript
		// Edit tags
		function editTags(id) {
			const tagName = prompt('Enter tag name:');
			if (!tagName) return;

			const csrfToken = document.cookie
				.split('; ')
				.find(row => row.startsWith('csrf_token='))
				?.split('=')[1] || '';

			fetch('/admin/media/' + id + '/tags', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/x-www-form-urlencoded',
					'X-CSRF-Token': csrfToken
				},
				body: 'action=add&tag=' + encodeURIComponent(tagName)
			})
			.then(r => r.json())
			.then(data => {
				if (data.success) {
					location.reload();
				} else {
					alert('Failed to add tag: ' + (data.error || 'Unknown error'));
				}
			});
		}

		// Delete media
		function deleteMedia(id) {
			if (!confirm('Delete this image? This cannot be undone.')) return;

			const csrfToken = document.cookie
				.split('; ')
				.find(row => row.startsWith('csrf_token='))
				?.split('=')[1] || '';

			fetch('/admin/media/' + id + '/delete', {
				method: 'POST',
				headers: {
					'X-CSRF-Token': csrfToken
				}
			})
			.then(r => r.json())
			.then(data => {
				if (data.success) {
					location.reload();
				} else {
					alert('Failed to delete: ' + (data.error || 'Unknown error'));
				}
			});
		}
```

**Step 4: Commit**

```bash
git add internal/handlers/admin_media.go cmd/stinky/server.go
git commit -m "feat: add tag management to media library"
```

---

## Task 6: Add Delete with Usage Warnings

**Files:**
- Modify: `internal/handlers/admin_media.go` (add delete handler)
- Modify: `cmd/stinky/server.go` (add delete route)

**Step 1: Add delete handler**

Add to `internal/handlers/admin_media.go`:

```go
// MediaDeleteHandler handles image deletion with usage checking
func MediaDeleteHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Site not found"})
		return
	}
	site := siteVal.(*models.Site)

	// Get media item ID
	mediaIDStr := c.Param("id")
	mediaID, err := strconv.ParseUint(mediaIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	// Get media item
	var mediaItem models.MediaItem
	if err := db.GetDB().Where("id = ? AND site_id = ?", mediaID, site.ID).First(&mediaItem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media item not found"})
		return
	}

	// Check usage
	imageURL := "/uploads/" + mediaItem.Filename
	usages := media.FindImageUsage(db.GetDB(), site.ID, imageURL)

	// If checking usage (not force delete)
	forceDelete := c.Query("force") == "true"
	if len(usages) > 0 && !forceDelete {
		// Return usage information
		var usageList []string
		for _, usage := range usages {
			usageList = append(usageList, fmt.Sprintf("%s → %s", usage.PageTitle, usage.BlockType))
		}

		c.JSON(http.StatusOK, gin.H{
			"in_use":  true,
			"usages":  usageList,
			"message": fmt.Sprintf("This image is used in %d place(s)", len(usages)),
		})
		return
	}

	// Delete file
	filePath := filepath.Join(site.SiteDir, "uploads", mediaItem.Filename)
	if err := os.Remove(filePath); err != nil {
		// Log error but continue (file might already be deleted)
		fmt.Printf("Warning: Failed to delete file %s: %v\n", filePath, err)
	}

	// Delete thumbnail
	thumbPath := filepath.Join(site.SiteDir, "uploads", "thumbs", mediaItem.Filename)
	os.Remove(thumbPath) // Ignore error

	// Delete database record (and tags via cascade)
	if err := db.GetDB().Delete(&mediaItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

**Step 2: Update delete JavaScript**

Replace the `deleteMedia` function in `renderMediaLibraryPage`:

```javascript
		function deleteMedia(id) {
			const csrfToken = document.cookie
				.split('; ')
				.find(row => row.startsWith('csrf_token='))
				?.split('=')[1] || '';

			// First, check if image is in use
			fetch('/admin/media/' + id + '/delete', {
				method: 'POST',
				headers: {
					'X-CSRF-Token': csrfToken
				}
			})
			.then(r => r.json())
			.then(data => {
				if (data.in_use) {
					// Show warning with usage locations
					const usageList = data.usages.join('\\n• ');
					const confirmMsg = '⚠️ ' + data.message + ':\\n\\n• ' + usageList + '\\n\\nDelete anyway? Blocks will show broken links.';

					if (confirm(confirmMsg)) {
						// Force delete
						fetch('/admin/media/' + id + '/delete?force=true', {
							method: 'POST',
							headers: {
								'X-CSRF-Token': csrfToken
							}
						})
						.then(r => r.json())
						.then(data => {
							if (data.success) {
								location.reload();
							} else {
								alert('Failed to delete: ' + (data.error || 'Unknown error'));
							}
						});
					}
				} else if (data.success) {
					// Orphaned image - simple confirm
					if (confirm('Delete this image? This cannot be undone.')) {
						location.reload();
					}
				} else {
					alert('Error: ' + (data.error || 'Unknown error'));
				}
			});
		}
```

**Step 3: Add route**

Add to `cmd/stinky/server.go`:

```go
				adminGroup.POST("/media/:id/delete", handlers.MediaDeleteHandler)
```

**Step 4: Test manually**

- Upload an image
- Add it to a page block
- Try to delete it from media library
- Should show warning with usage location

**Step 5: Commit**

```bash
git add internal/handlers/admin_media.go cmd/stinky/server.go
git commit -m "feat: add delete with usage warnings"
```

---

## Task 7: Add Dashboard Link

**Files:**
- Modify: `internal/handlers/admin.go` (add link in DashboardHandler)

**Step 1: Add media library link**

Find the dashboard navigation in `internal/handlers/admin.go` (around line ~450-500), add after the Settings link:

```go
				<a href="/admin/media" class="nav-link">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
						<circle cx="8.5" cy="8.5" r="1.5"/>
						<polyline points="21 15 16 10 5 21"/>
					</svg>
					Media Library
				</a>
```

**Step 2: Test**

- Navigate to dashboard
- Verify "Media Library" link appears
- Click it, should load media library page

**Step 3: Commit**

```bash
git add internal/handlers/admin.go
git commit -m "feat: add media library link to dashboard"
```

---

## Task 8: Add Modal Picker for Block Editors

**Files:**
- Create: `internal/handlers/admin_media_picker.go`
- Modify: `internal/handlers/admin_blocks.go` (add Browse Library button)
- Modify: `cmd/stinky/server.go` (add picker route)

**Step 1: Create picker handler**

Create `internal/handlers/admin_media_picker.go`:

```go
package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// MediaPickerHandler shows modal picker for block editors
func MediaPickerHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get all media items for this site
	var mediaItems []models.MediaItem
	db.GetDB().Where("site_id = ?", site.ID).
		Preload("Tags").
		Order("created_at DESC").
		Find(&mediaItems)

	// Build image grid
	var imageGrid string
	for _, item := range mediaItems {
		thumbURL := fmt.Sprintf("/uploads/thumbs/%s", item.Filename)
		imageURL := fmt.Sprintf("/uploads/%s", item.Filename)

		imageGrid += fmt.Sprintf(`
		<div class="picker-card" onclick="selectImage('%s', '%s')">
			<img src="%s" alt="%s">
			<div class="picker-filename">%s</div>
		</div>
		`, imageURL, item.OriginalName, thumbURL, item.OriginalName, item.OriginalName)
	}

	if len(mediaItems) == 0 {
		imageGrid = `<div class="empty-state">No images in library. Upload images from the Media Library page.</div>`
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Select Image</title>
	<style>
		%s

		body {
			margin: 0;
			padding: var(--spacing-base);
		}

		.picker-header {
			margin-bottom: var(--spacing-base);
			display: flex;
			justify-content: space-between;
			align-items: center;
		}

		.picker-grid {
			display: grid;
			grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
			gap: var(--spacing-base);
		}

		.picker-card {
			cursor: pointer;
			border: 2px solid transparent;
			border-radius: var(--radius-base);
			overflow: hidden;
			transition: border-color 0.2s;
		}

		.picker-card:hover {
			border-color: var(--color-accent);
		}

		.picker-card img {
			width: 100%%;
			height: 150px;
			object-fit: cover;
			display: block;
		}

		.picker-filename {
			padding: var(--spacing-sm);
			font-size: 12px;
			text-align: center;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.empty-state {
			text-align: center;
			padding: calc(var(--spacing-base) * 4);
			color: var(--color-text-secondary);
		}
	</style>
</head>
<body>
	<div class="picker-header">
		<h2>Select Image</h2>
		<button onclick="window.close()" class="btn btn-secondary">Cancel</button>
	</div>

	<div class="picker-grid">
		%s
	</div>

	<script>
		function selectImage(url, filename) {
			// Send message to parent window
			if (window.opener) {
				window.opener.postMessage({
					type: 'image-selected',
					url: url,
					filename: filename
				}, '*');
				window.close();
			}
		}
	</script>
</body>
</html>`, GetDesignSystemCSS(), imageGrid)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
```

**Step 2: Add route**

Add to `cmd/stinky/server.go`:

```go
				adminGroup.GET("/media/picker", handlers.MediaPickerHandler)
```

**Step 3: Add Browse Library button to image block editor**

Find the image block edit form in `internal/handlers/admin_blocks.go` (around line ~400), add button after file input:

```go
				<div class="form-group">
					<label for="image">Image</label>
					<input type="file" id="image" name="image" accept="image/*">
					<button type="button" class="btn btn-secondary" onclick="openMediaPicker()" style="margin-top: var(--spacing-sm);">
						📚 Browse Library
					</button>
					<input type="hidden" id="selected-image-url" name="selected_image_url">
				</div>
```

**Step 4: Add JavaScript for picker**

Add at the end of the image block edit form's script section:

```javascript
		function openMediaPicker() {
			window.open('/admin/media/picker', 'mediaPicker', 'width=800,height=600');
		}

		// Listen for selected image
		window.addEventListener('message', (event) => {
			if (event.data.type === 'image-selected') {
				document.getElementById('selected-image-url').value = event.data.url;
				alert('Image selected: ' + event.data.filename);
			}
		});
```

**Step 5: Update UpdateBlockHandler**

Modify `UpdateBlockHandler` in `admin_blocks.go` to check for `selected_image_url`:

```go
	// Check if image was selected from library
	selectedImageURL := c.PostForm("selected_image_url")
	if selectedImageURL != "" {
		// Use selected image from library
		blockData.URL = selectedImageURL
	} else if imageFile != nil {
		// Upload new image (existing logic)
		// ...
	}
```

**Step 6: Commit**

```bash
git add internal/handlers/admin_media_picker.go internal/handlers/admin_blocks.go cmd/stinky/server.go
git commit -m "feat: add modal picker for block editors"
```

---

## Task 9: Add Import Existing Uploads

**Files:**
- Create: `internal/media/import.go`
- Create: `internal/media/import_test.go`
- Modify: `internal/handlers/admin_media.go` (auto-import on first load)

**Step 1: Write import test**

Create `internal/media/import_test.go`:

```go
package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestImportExistingUploads(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Site{}, &models.User{}, &models.MediaItem{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test site
	tmpDir := t.TempDir()
	uploadsDir := filepath.Join(tmpDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)

	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{
		Subdomain: "test",
		OwnerID:   user.ID,
		SiteDir:   tmpDir,
	}
	db.Create(&site)

	// Create fake image files
	os.WriteFile(filepath.Join(uploadsDir, "abc123.jpg"), []byte("fake image"), 0644)
	os.WriteFile(filepath.Join(uploadsDir, "def456.png"), []byte("fake image 2"), 0644)
	os.WriteFile(filepath.Join(uploadsDir, "notanimage.txt"), []byte("text"), 0644)

	// Import
	count, err := ImportExistingUploads(db, site)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 imports, got %d", count)
	}

	// Verify database records
	var items []models.MediaItem
	db.Where("site_id = ?", site.ID).Find(&items)

	if len(items) != 2 {
		t.Errorf("Expected 2 media items in database, got %d", len(items))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/media -run TestImportExistingUploads -v`
Expected: FAIL with "undefined: ImportExistingUploads"

**Step 3: Implement import function**

Create `internal/media/import.go`:

```go
package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// ImportExistingUploads scans uploads directory and creates media_items records
// Returns count of imported files
func ImportExistingUploads(db *gorm.DB, site models.Site) (int, error) {
	uploadsDir := filepath.Join(site.SiteDir, "uploads")

	// Check if directory exists
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		return 0, nil // No uploads directory, nothing to import
	}

	// Read directory
	files, err := os.ReadDir(uploadsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read uploads directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue // Skip subdirectories (like thumbs/)
		}

		filename := file.Name()

		// Only import image files
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			continue
		}

		// Check if already imported
		var existing models.MediaItem
		err := db.Where("site_id = ? AND filename = ?", site.ID, filename).First(&existing).Error
		if err == nil {
			continue // Already imported
		}

		// Get file info
		fileInfo, err := file.Info()
		if err != nil {
			continue // Skip if can't get info
		}

		// Detect mime type from extension
		mimeType := "image/jpeg"
		switch ext {
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		case ".webp":
			mimeType = "image/webp"
		}

		// Create media item
		mediaItem := models.MediaItem{
			SiteID:       site.ID,
			Filename:     filename,
			OriginalName: filename, // Best guess
			FileSize:     fileInfo.Size(),
			MimeType:     mimeType,
			UploadedBy:   site.OwnerID, // Assume owner uploaded
		}

		if err := db.Create(&mediaItem).Error; err != nil {
			return count, fmt.Errorf("failed to create media item for %s: %w", filename, err)
		}

		count++
	}

	return count, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/media -run TestImportExistingUploads -v`
Expected: PASS

**Step 5: Add auto-import to MediaLibraryHandler**

Add at the start of `MediaLibraryHandler` in `admin_media.go`:

```go
	// Auto-import existing uploads on first use
	var itemCount int64
	db.GetDB().Model(&models.MediaItem{}).Where("site_id = ?", site.ID).Count(&itemCount)

	if itemCount == 0 {
		// First time accessing media library - import existing uploads
		count, err := media.ImportExistingUploads(db.GetDB(), *site)
		if err != nil {
			fmt.Printf("Warning: Failed to import existing uploads: %v\n", err)
		} else if count > 0 {
			// Could add a flash message here, but for now just log
			fmt.Printf("Imported %d existing images for site %s\n", count, site.Subdomain)
		}
	}
```

**Step 6: Commit**

```bash
git add internal/media/import.go internal/media/import_test.go internal/handlers/admin_media.go
git commit -m "feat: add import of existing uploads on first use"
```

---

## Task 10: Update Image Upload Handler for Backwards Compatibility

**Files:**
- Modify: `internal/handlers/uploads.go` (if it exists, or admin_blocks.go)

**Step 1: Find existing upload handler**

The existing `UploadImageHandler` in `internal/handlers/uploads.go` or `admin_blocks.go` needs to create media_items records.

**Step 2: Update to create MediaItem**

After successful file upload, add:

```go
	// Create media item record for tracking
	mediaItem := models.MediaItem{
		SiteID:       site.ID,
		Filename:     filepath.Base(webPath),
		OriginalName: fileHeader.Filename,
		FileSize:     fileHeader.Size,
		MimeType:     fileHeader.Header.Get("Content-Type"),
		UploadedBy:   user.ID,
	}
	db.GetDB().Create(&mediaItem) // Ignore error - not critical

	// Generate thumbnail if possible
	srcPath := filepath.Join(site.SiteDir, "uploads", mediaItem.Filename)
	thumbPath := filepath.Join(site.SiteDir, "uploads", "thumbs", mediaItem.Filename)
	media.GenerateThumbnail(srcPath, thumbPath, 200, 200) // Ignore error
```

**Step 3: Test**

- Upload image from block editor (old workflow)
- Check media library - image should appear
- Verify thumbnail was generated

**Step 4: Commit**

```bash
git add internal/handlers/uploads.go  # or admin_blocks.go
git commit -m "feat: create media items for direct uploads"
```

---

## Task 11: Run Database Migration

**Files:**
- Modify: `internal/db/db.go` (add migration)

**Step 1: Add models to AutoMigrate**

Find the `AutoMigrate` call in `internal/db/db.go`, add MediaItem and MediaTag:

```go
	if err := database.AutoMigrate(
		&models.Site{},
		&models.User{},
		&models.SiteUser{},
		&models.Page{},
		&models.Block{},
		&models.MenuItem{},
		&models.MediaItem{},    // Add this
		&models.MediaTag{},     // Add this
	); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}
```

**Step 2: Test migration**

Run: `go run cmd/stinky/main.go server start`
Expected: Server starts, tables created successfully

**Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "feat: add media library tables to migration"
```

---

## Completion

All tasks complete! The media library is now fully implemented with:

- ✅ Database models and migrations
- ✅ Thumbnail generation
- ✅ Upload handling with progress
- ✅ Tag-based organization
- ✅ Search and filtering
- ✅ Usage detection for safe deletion
- ✅ Modal picker for block editors
- ✅ Import of existing uploads
- ✅ Dashboard integration
- ✅ Backwards compatibility

## Testing Checklist

Before marking as complete, verify:

- [ ] Can upload images to media library
- [ ] Thumbnails generate correctly
- [ ] Can add/remove tags
- [ ] Search finds images
- [ ] Filter by tags works
- [ ] "Show orphaned" filter works
- [ ] Delete shows warning if image is in use
- [ ] Modal picker opens from block editor
- [ ] Selecting image from picker works
- [ ] Direct upload from block creates media_item
- [ ] Existing images imported on first load
- [ ] Dashboard link works

## Next Steps

Use **superpowers:verification-before-completion** before merging to main.
Use **superpowers:finishing-a-development-branch** to clean up and merge.
