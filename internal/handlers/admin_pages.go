package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// NewPageFormHandler shows the form to create a new page
func NewPageFormHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Create New Page</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            background: #f5f5f5;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            margin: 0 0 20px 0;
            font-size: 28px;
            color: #333;
        }
        .back-link {
            color: #007bff;
            text-decoration: none;
            font-size: 14px;
            margin-bottom: 20px;
            display: inline-block;
        }
        .back-link:hover {
            text-decoration: underline;
        }
        form {
            margin-top: 20px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #333;
        }
        input[type="text"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
            box-sizing: border-box;
        }
        input[type="text"]:focus {
            outline: none;
            border-color: #007bff;
        }
        .help-text {
            font-size: 12px;
            color: #666;
            margin-top: 5px;
        }
        .button-group {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }
        button {
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
        }
        button[type="submit"] {
            background: #007bff;
            color: white;
        }
        button[type="submit"]:hover {
            background: #0056b3;
        }
        a.cancel {
            padding: 10px 20px;
            background: #6c757d;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 600;
        }
        a.cancel:hover {
            background: #5a6268;
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/admin/dashboard" class="back-link">← Back to Dashboard</a>

        <h1>Create New Page</h1>

        <form method="POST" action="/admin/pages">
            <div class="form-group">
                <label for="slug">Slug:</label>
                <input type="text" id="slug" name="slug" required placeholder="/about">
                <div class="help-text">The URL path for this page (e.g., /about, /contact)</div>
            </div>

            <div class="form-group">
                <label for="title">Title:</label>
                <input type="text" id="title" name="title" required placeholder="About Us">
                <div class="help-text">The page title that will appear in the editor and browser tab</div>
            </div>

            <div class="button-group">
                <button type="submit">Create Page</button>
                <a href="/admin/dashboard" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// CreatePageHandler creates a new page
func CreatePageHandler(c *gin.Context) {
	siteVal, _ := c.Get("site")
	site := siteVal.(*models.Site)

	slug := c.PostForm("slug")
	title := c.PostForm("title")

	// Validate
	if slug == "" {
		c.String(http.StatusBadRequest, "Slug is required")
		return
	}
	if title == "" {
		c.String(http.StatusBadRequest, "Title is required")
		return
	}

	// Check if page with this slug already exists
	var existing models.Page
	result := db.GetDB().Where("site_id = ? AND slug = ?", site.ID, slug).First(&existing)
	if result.Error == nil {
		c.String(http.StatusBadRequest, "Page with this slug already exists")
		return
	}

	// Create page
	page := models.Page{
		SiteID:    site.ID,
		Slug:      slug,
		Title:     title,
		Published: false,
	}

	if err := db.GetDB().Create(&page).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create page")
		return
	}

	// Redirect to edit page
	c.Redirect(http.StatusFound, "/admin/pages/"+strconv.Itoa(int(page.ID))+"/edit")
}

// EditPageHandler shows the page editor
func EditPageHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID from URL
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	// Load page with blocks
	var page models.Page
	result := db.GetDB().
		Preload("Blocks", func(db *gorm.DB) *gorm.DB {
			return db.Order("\"order\" ASC")
		}).
		Where("id = ?", pageID).
		First(&page)

	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Build blocks HTML
	var blocksHTML string
	for i, block := range page.Blocks {
		// Get block type label
		blockTypeLabel := "Text Block"
		if block.Type == "image" {
			blockTypeLabel = "Image Block"
		}

		// Extract preview from JSON content
		preview := ""
		if len(block.Data) > 100 {
			preview = block.Data[:100] + "..."
		} else {
			preview = block.Data
		}
		if preview == "" {
			preview = "(empty)"
		}

		// Determine if move buttons should be enabled
		showMoveUp := i > 0
		showMoveDown := i < len(page.Blocks)-1

		moveUpBtn := ""
		if showMoveUp {
			moveUpBtn = `<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/move-up" style="display:inline;">
				<button type="submit" class="btn-icon">↑</button>
			</form>`
		} else {
			moveUpBtn = `<button class="btn-icon" disabled style="opacity: 0.3;">↑</button>`
		}

		moveDownBtn := ""
		if showMoveDown {
			moveDownBtn = `<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/move-down" style="display:inline;">
				<button type="submit" class="btn-icon">↓</button>
			</form>`
		} else {
			moveDownBtn = `<button class="btn-icon" disabled style="opacity: 0.3;">↓</button>`
		}

		blocksHTML += `
			<div class="block-item">
				<div class="block-info">
					<div class="block-type">` + blockTypeLabel + `</div>
					<div class="block-preview">` + preview + `</div>
				</div>
				<div class="block-actions">
					` + moveUpBtn + `
					` + moveDownBtn + `
					<a href="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/edit" class="btn-small">Edit</a>
					<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/delete" style="display:inline;" onsubmit="return confirm('Delete this block?')">
						<button type="submit" class="btn-small btn-danger">Delete</button>
					</form>
				</div>
			</div>
		`
	}

	if blocksHTML == "" {
		blocksHTML = `<div class="empty-state">No blocks yet. Add a block to get started.</div>`
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Edit Page - ` + page.Title + `</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 900px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { margin: 0 0 20px 0; font-size: 28px; color: #333; }
        .back-link { color: #007bff; text-decoration: none; font-size: 14px; margin-bottom: 20px; display: inline-block; }
        .back-link:hover { text-decoration: underline; }
        .page-header { margin-bottom: 30px; }
        .page-header input { width: 100%; padding: 12px; font-size: 18px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
        .page-actions { margin-top: 15px; display: flex; gap: 10px; }
        .btn { padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; border: none; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #0056b3; }
        .btn-secondary { background: #6c757d; }
        .btn-secondary:hover { background: #5a6268; }
        .btn-success { background: #28a745; }
        .btn-success:hover { background: #218838; }
        .section { margin-bottom: 30px; }
        .section h2 { font-size: 18px; margin-bottom: 15px; color: #444; }
        .block-item { padding: 15px; border: 1px solid #e0e0e0; border-radius: 4px; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center; }
        .block-info { flex: 1; }
        .block-type { font-weight: 600; margin-bottom: 5px; font-size: 14px; color: #333; }
        .block-preview { font-size: 13px; color: #666; font-family: monospace; background: #f8f8f8; padding: 8px; border-radius: 3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .block-actions { display: flex; gap: 8px; align-items: center; }
        .btn-small { padding: 6px 12px; font-size: 13px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; border: none; cursor: pointer; }
        .btn-small:hover { background: #0056b3; }
        .btn-danger { background: #dc3545; }
        .btn-danger:hover { background: #c82333; }
        .btn-icon { padding: 6px 10px; font-size: 16px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer; }
        .btn-icon:hover { background: #5a6268; }
        .empty-state { padding: 40px; text-align: center; color: #999; border: 2px dashed #e0e0e0; border-radius: 4px; }
        .add-block { margin-top: 15px; display: flex; gap: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <a href="/admin/dashboard" class="back-link">← Back to Dashboard</a>

        <h1>Edit Page</h1>

        <div class="page-header">
            <form method="POST" action="/admin/pages/` + pageIDStr + `">
                <input type="text" name="title" value="` + page.Title + `" placeholder="Page Title" required>
                <div class="page-actions">
                    <button type="submit" class="btn">Save Draft</button>
                </div>
            </form>`

	// Show Publish or Unpublish button based on current status
	if page.Published {
		html += `
            <form method="POST" action="/admin/pages/` + pageIDStr + `/unpublish" style="display:inline;">
                <button type="submit" class="btn btn-secondary">Unpublish</button>
            </form>`
	} else {
		html += `
            <form method="POST" action="/admin/pages/` + pageIDStr + `/publish" style="display:inline;">
                <button type="submit" class="btn btn-success">Publish</button>
            </form>`
	}

	html += `
        </div>

        <div class="section">
            <h2>Content Blocks</h2>
            ` + blocksHTML + `
            <div class="add-block">
                <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                    <input type="hidden" name="type" value="text">
                    <button type="submit" class="btn">+ Add Text Block</button>
                </form>
                <a href="/admin/pages/` + pageIDStr + `/blocks/new-image" class="btn" style="background: #17a2b8;">+ Add Image Block</a>
            </div>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// UpdatePageHandler updates page title (Save Draft)
func UpdatePageHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID from URL
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	// Load page from database
	var page models.Page
	result := db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Get title from form
	title := c.PostForm("title")
	if title == "" {
		c.String(http.StatusBadRequest, "Title is required")
		return
	}

	// Update page title (keeps Published unchanged - this is "Save Draft")
	page.Title = title
	if err := db.GetDB().Save(&page).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to update page")
		return
	}

	// Redirect back to page editor
	c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
}

// PublishPageHandler publishes a page
func PublishPageHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID from URL
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	// Load page from database
	var page models.Page
	result := db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Set page.Published = true
	page.Published = true
	if err := db.GetDB().Save(&page).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to publish page")
		return
	}

	// Redirect back to page editor
	c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
}

// UnpublishPageHandler unpublishes a page
func UnpublishPageHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID from URL
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	// Load page from database
	var page models.Page
	result := db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Set page.Published = false
	page.Published = false
	if err := db.GetDB().Save(&page).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to unpublish page")
		return
	}

	// Redirect back to page editor
	c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
}

// DeletePageHandler deletes a page
func DeletePageHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID from URL
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	// Load page from database
	var page models.Page
	result := db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Don't allow deleting homepage
	if page.Slug == "/" {
		c.String(http.StatusForbidden, "Cannot delete homepage")
		return
	}

	// Delete the page (soft delete)
	if err := db.GetDB().Delete(&page).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete page")
		return
	}

	// Redirect back to dashboard
	c.Redirect(http.StatusFound, "/admin/dashboard")
}

// NewImageBlockFormHandler displays the form for adding a new image block
func NewImageBlockFormHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID from URL
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	// Load page from database
	var page models.Page
	result := db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Add Image Block</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 700px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { margin: 0 0 20px 0; font-size: 24px; color: #333; }
        .back-link { color: #007bff; text-decoration: none; font-size: 14px; margin-bottom: 20px; display: inline-block; }
        .back-link:hover { text-decoration: underline; }
        .form-group { margin-bottom: 20px; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #333; font-size: 14px; }
        input[type="text"], input[type="file"] { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; font-size: 14px; }
        input[type="file"] { padding: 8px; }
        .help-text { font-size: 13px; color: #666; margin-top: 5px; }
        .btn { padding: 12px 24px; background: #17a2b8; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #138496; }
        .btn-secondary { background: #6c757d; margin-left: 10px; }
        .btn-secondary:hover { background: #5a6268; }
        .preview { margin-top: 20px; padding: 15px; background: #f8f9fa; border-radius: 4px; display: none; }
        .preview img { max-width: 100%; height: auto; display: block; margin-bottom: 10px; }
        #uploadProgress { display: none; margin-top: 10px; padding: 10px; background: #e7f3ff; border-radius: 4px; color: #0066cc; }
    </style>
    <script>
        let uploadedImageURL = '';

        async function handleImageUpload() {
            const fileInput = document.getElementById('imageFile');
            const file = fileInput.files[0];

            if (!file) {
                alert('Please select an image file');
                return;
            }

            const formData = new FormData();
            formData.append('image', file);

            const progress = document.getElementById('uploadProgress');
            progress.style.display = 'block';
            progress.textContent = 'Uploading...';

            try {
                const response = await fetch('/admin/upload/image', {
                    method: 'POST',
                    body: formData
                });

                if (!response.ok) {
                    throw new Error('Upload failed');
                }

                const data = await response.json();
                uploadedImageURL = data.url;

                // Show preview
                const preview = document.getElementById('preview');
                const previewImg = document.getElementById('previewImg');
                previewImg.src = data.url;
                preview.style.display = 'block';

                progress.textContent = 'Upload complete!';
                progress.style.background = '#d4edda';
                progress.style.color = '#155724';

                // Enable submit button
                document.getElementById('submitBtn').disabled = false;
            } catch (error) {
                progress.textContent = 'Upload failed. Please try again.';
                progress.style.background = '#f8d7da';
                progress.style.color = '#721c24';
            }
        }

        function handleSubmit(event) {
            event.preventDefault();

            if (!uploadedImageURL) {
                alert('Please upload an image first');
                return;
            }

            const alt = document.getElementById('altText').value;
            const caption = document.getElementById('caption').value;

            // Create JSON data
            const blockData = {
                url: uploadedImageURL,
                alt: alt,
                caption: caption
            };

            // Submit form with JSON data
            const form = document.getElementById('imageBlockForm');
            const dataInput = document.createElement('input');
            dataInput.type = 'hidden';
            dataInput.name = 'data';
            dataInput.value = JSON.stringify(blockData);
            form.appendChild(dataInput);

            const typeInput = document.createElement('input');
            typeInput.type = 'hidden';
            typeInput.name = 'type';
            typeInput.value = 'image';
            form.appendChild(typeInput);

            form.submit();
        }
    </script>
</head>
<body>
    <div class="container">
        <a href="/admin/pages/` + pageIDStr + `/edit" class="back-link">← Back to Edit Page</a>

        <h1>Add Image Block</h1>

        <form id="imageBlockForm" method="POST" action="/admin/pages/` + pageIDStr + `/blocks" onsubmit="handleSubmit(event)">
            <div class="form-group">
                <label for="imageFile">Select Image</label>
                <input type="file" id="imageFile" accept="image/*" onchange="handleImageUpload()" required>
                <p class="help-text">Supported formats: JPG, PNG, GIF, WebP</p>
                <div id="uploadProgress"></div>
            </div>

            <div id="preview" class="preview">
                <img id="previewImg" alt="Preview">
            </div>

            <div class="form-group">
                <label for="altText">Alt Text</label>
                <input type="text" id="altText" placeholder="Describe the image for accessibility" required>
                <p class="help-text">Required for accessibility. Describe what's in the image.</p>
            </div>

            <div class="form-group">
                <label for="caption">Caption (optional)</label>
                <input type="text" id="caption" placeholder="Optional caption to display below image">
            </div>

            <button type="submit" id="submitBtn" class="btn" disabled>Add Image Block</button>
            <a href="/admin/pages/` + pageIDStr + `/edit" class="btn btn-secondary" style="text-decoration: none;">Cancel</a>
        </form>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
