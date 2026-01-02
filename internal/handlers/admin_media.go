// SPDX-License-Identifier: MIT
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/media"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// saveToCentralizedStorage saves an uploaded file to centralized media storage
func saveToCentralizedStorage(file *multipart.FileHeader) (string, error) {
	// Get centralized media directory from config
	mediaDir := config.GetString("storage.media_dir")
	if mediaDir == "" {
		return "", fmt.Errorf("storage.media_dir not configured")
	}

	// Create media directory if it doesn't exist
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	// Generate random filename to avoid conflicts
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random filename: %w", err)
	}
	randomName := hex.EncodeToString(randomBytes)

	// Get file extension
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg" // default
	}

	// Create full path (save to uploads subdirectory)
	filename := randomName + ext
	uploadsDir := filepath.Join(mediaDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}
	fullPath := filepath.Join(uploadsDir, filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Validate file content type using Magic Bytes
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file for validation: %w", err)
	}

	contentType := http.DetectContentType(buffer[:n])
	validTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	isValid := false
	for _, validType := range validTypes {
		if contentType == validType {
			isValid = true
			break
		}
	}
	if !isValid {
		return "", fmt.Errorf("invalid file type: %s (only images allowed)", contentType)
	}

	// Reset file pointer to beginning after validation
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file contents
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Return just the filename (not web path)
	return filename, nil
}

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

	// Auto-import existing uploads on first use
	var itemCount int64
	db.GetDB().Model(&models.MediaItem{}).Count(&itemCount)

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

	// Query media items - show ALL media across all sites
	query := db.GetDB().Model(&models.MediaItem{})

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

	db.GetDB().Model(&models.MediaItem{}).Count(&totalCount)

	query.Preload("User").
		Preload("Tags").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&mediaItems)

	// Filter orphaned if requested
	var displayItems []models.MediaItem
	if showOrphaned {
		for _, item := range mediaItems {
			usages := media.FindImageUsage(db.GetDB(), site.ID, "/assets/"+item.Filename)
			if len(usages) == 0 {
				displayItems = append(displayItems, item)
			}
		}
	} else {
		displayItems = mediaItems
	}

	// Calculate pagination
	totalPages := 0
	if !showOrphaned {
		totalPages = int(totalCount) / limit
		if int(totalCount)%limit != 0 {
			totalPages++
		}
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

	// Validate file sizes
	const MaxFileSize = 5 * 1024 * 1024 // 5MB
	for _, fileHeader := range files {
		if fileHeader.Size > MaxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s exceeds 5MB limit", fileHeader.Filename)})
			return
		}
	}

	for _, fileHeader := range files {
		// Save file to centralized storage
		filename, err := saveToCentralizedStorage(fileHeader)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to upload %s: %v", fileHeader.Filename, err)})
			return
		}

		// Get file info
		fileSize := fileHeader.Size
		mimeType := fileHeader.Header.Get("Content-Type")

		// Create database record with UploadedFromSiteID
		mediaItem := models.MediaItem{
			Filename:           filename,
			OriginalName:       fileHeader.Filename,
			FileSize:           fileSize,
			MimeType:           mimeType,
			UploadedBy:         user.ID,
			UploadedFromSiteID: &site.ID, // Track which site uploaded this
		}

		if err := db.GetDB().Create(&mediaItem).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save media item"})
			return
		}

		// Generate thumbnail in centralized location
		mediaDir := config.GetString("storage.media_dir")
		srcPath := filepath.Join(mediaDir, "uploads", filename)
		thumbsDir := filepath.Join(mediaDir, "uploads", "thumbs")
		if err := os.MkdirAll(thumbsDir, 0755); err != nil {
			fmt.Printf("Warning: Failed to create thumbs directory: %v\n", err)
		} else {
			thumbPath := filepath.Join(thumbsDir, filename)
			if err := media.GenerateThumbnail(srcPath, thumbPath, 200, 200); err != nil {
				// Log error but don't fail the upload
				fmt.Printf("Warning: Failed to generate thumbnail for %s: %v\n", filename, err)
			}
		}

		uploadedItems = append(uploadedItems, mediaItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"items":   uploadedItems,
	})
}

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

// MediaDeleteHandler handles image deletion with usage checking
func MediaDeleteHandler(c *gin.Context) {
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

	// Get media item ID
	mediaIDStr := c.Param("id")
	mediaID, err := strconv.ParseUint(mediaIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid media ID"})
		return
	}

	// Get media item (centralized - no site filtering)
	var mediaItem models.MediaItem
	if err := db.GetDB().First(&mediaItem, mediaID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media item not found"})
		return
	}

	// Permission check: only uploader or global admin can delete
	if mediaItem.UploadedBy != user.ID && !user.IsGlobalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this media"})
		return
	}

	// Check usage
	imageURL := "/assets/" + mediaItem.Filename
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

	// Get centralized media directory
	mediaDir := config.GetString("storage.media_dir")
	if mediaDir == "" {
		mediaDir = "/var/lib/stinkykitty/media"
	}

	// Delete file from centralized storage
	filePath := filepath.Join(mediaDir, mediaItem.Filename)
	if err := os.Remove(filePath); err != nil {
		// Log error but continue (file might already be deleted)
		fmt.Printf("Warning: Failed to delete file %s: %v\n", filePath, err)
	}

	// Delete thumbnail from centralized storage
	thumbPath := filepath.Join(mediaDir, "thumbs", mediaItem.Filename)
	os.Remove(thumbPath) // Ignore error

	// Delete database record (and tags via cascade)
	if err := db.GetDB().Delete(&mediaItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
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
		thumbURL := fmt.Sprintf("/assets/thumbs/%s", item.Filename)

		// Build tag badges
		var tagBadges string
		for _, tag := range item.Tags {
			tagBadges += fmt.Sprintf(`<span class="tag-badge">%s</span>`, tag.TagName)
		}

		// Get uploader email
		uploaderEmail := "Unknown"
		if item.User.Email != "" {
			uploaderEmail = item.User.Email
		}

		// Check if user can delete (uploader or global admin)
		canDelete := item.UploadedBy == user.ID || user.IsGlobalAdmin
		deleteButton := ""
		if canDelete {
			deleteButton = fmt.Sprintf(`<button class="btn-small btn-danger" onclick="deleteMedia(%d)">Delete</button>`, item.ID)
		}

		imageGrid += fmt.Sprintf(`
		<div class="media-card" data-id="%d">
			<div class="media-thumbnail">
				<img src="%s" alt="%s" loading="lazy">
			</div>
			<div class="media-info">
				<div class="media-filename" title="%s">%s</div>
				<div class="media-tags">%s</div>
				<div class="media-uploader">Uploaded by: %s</div>
				<div class="media-date">%s</div>
				<div class="media-actions">
					<button class="btn-small" onclick="editTags(%d)">Edit Tags</button>
					%s
				</div>
			</div>
		</div>
		`, item.ID, thumbURL, item.OriginalName, item.OriginalName, item.OriginalName,
			tagBadges, uploaderEmail, item.CreatedAt.Format("Jan 2, 2006"), item.ID, deleteButton)
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

		.media-uploader {
			font-size: 12px;
			color: var(--color-text-secondary);
			margin-bottom: var(--spacing-sm);
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
			uploadFiles();
		});

		// Click to browse
		uploadZone.addEventListener('click', (e) => {
			if (e.target.tagName !== 'LABEL') {
				fileInput.click();
			}
		});

		fileInput.addEventListener('change', () => {
			if (fileInput.files.length > 0) {
				uploadFiles();
			}
		});

		// Upload files via AJAX
		async function uploadFiles() {
			const formData = new FormData(uploadForm);

			try {
				const csrfToken = decodeURIComponent(
					document.cookie
						.split('; ')
						.find(row => row.startsWith('csrf_token='))
						?.substring('csrf_token='.length) || ''
				);

				const response = await fetch('/admin/media/upload', {
					method: 'POST',
					headers: {
						'X-CSRF-Token': csrfToken
					},
					body: formData
				});

				const result = await response.json();

				if (result.success) {
					// Reload page to show new images
					window.location.reload();
				} else {
					alert('Upload failed: ' + (result.error || 'Unknown error'));
				}
			} catch (error) {
				alert('Upload failed: ' + error.message);
			}
		}

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

		// Edit tags
		function editTags(id) {
			const tagName = prompt('Enter tag name:');
			if (!tagName) return;

			const csrfToken = decodeURIComponent(
				document.cookie
					.split('; ')
					.find(row => row.startsWith('csrf_token='))
					?.substring('csrf_token='.length) || ''
			);

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
			const csrfToken = decodeURIComponent(
				document.cookie
					.split('; ')
					.find(row => row.startsWith('csrf_token='))
					?.substring('csrf_token='.length) || ''
			);

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
	</script>
</body>
</html>`, site.SiteTitle, GetDesignSystemCSS(), csrfToken, search,
		(map[bool]string{true: "Show All", false: "Show Orphaned"})[showOrphaned],
		filterBadges, imageGrid, pagination)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
