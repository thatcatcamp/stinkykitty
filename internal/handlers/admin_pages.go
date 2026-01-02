// SPDX-License-Identifier: MIT
package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/search"
	"gorm.io/gorm"
)

// NewPageFormHandler shows the form to create a new page
func NewPageFormHandler(c *gin.Context) {
	csrfToken := middleware.GetCSRFTokenHTML(c)
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
            ` + csrfToken + `
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

	// Index the page in FTS (won't be searchable until published)
	if err := search.IndexPage(db.GetDB(), &page); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to index page %d: %v\n", page.ID, err)
	}

	// Redirect to edit page
	c.Redirect(http.StatusFound, "/admin/pages/"+strconv.Itoa(int(page.ID))+"/edit")
}

// EditPageHandler shows the page editor
func EditPageHandler(c *gin.Context) {
	// Get user and site from context
	userVal, exists := c.Get("user")
	if !exists {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}
	user := userVal.(*models.User)

	// Get CSRF token
	csrfToken := middleware.GetCSRFTokenHTML(c)

	var site *models.Site
	siteVal, exists := c.Get("site")
	if exists {
		site = siteVal.(*models.Site)
	}

	// Try to get site from query parameter if needed (for redirects after creation)
	if site == nil {
		siteIDStr := c.Query("site")
		if siteIDStr != "" {
			var siteID uint
			if _, err := fmt.Sscanf(siteIDStr, "%d", &siteID); err == nil {
				var queriedSite models.Site
				if err := db.GetDB().First(&queriedSite, siteID).Error; err == nil {
					site = &queriedSite
				}
			}
		}
	}

	if site == nil {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}

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
		} else if block.Type == "heading" {
			blockTypeLabel = "Heading Block"
		} else if block.Type == "quote" {
			blockTypeLabel = "Quote Block"
		} else if block.Type == "button" {
			blockTypeLabel = "Button Block"
		} else if block.Type == "video" {
			blockTypeLabel = "Video Block"
		} else if block.Type == "spacer" {
			blockTypeLabel = "Spacer Block"
		} else if block.Type == "contact" {
			blockTypeLabel = "Contact Form Block"
		} else if block.Type == "columns" {
			blockTypeLabel = "Columns Block"
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
				` + csrfToken + `
				<button type="submit" class="btn-icon">↑</button>
			</form>`
		} else {
			moveUpBtn = `<button class="btn-icon" disabled style="opacity: 0.3;">↑</button>`
		}

		moveDownBtn := ""
		if showMoveDown {
			moveDownBtn = `<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/move-down" style="display:inline;">
				` + csrfToken + `
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
						` + csrfToken + `
						<button type="submit" class="btn-small btn-danger">Delete</button>
					</form>
				</div>
			</div>
		`
	}

	if blocksHTML == "" {
		blocksHTML = `<div class="empty-state">No blocks yet. Add a block to get started.</div>`
	}

	var publishButton string
	if page.Published {
		publishButton = `
                            <form method="POST" action="/admin/pages/` + pageIDStr + `/unpublish" style="display:inline;">
                                ` + csrfToken + `
                                <button type="submit" class="btn btn-secondary">Unpublish</button>
                            </form>`
	} else {
		publishButton = `
                            <form method="POST" action="/admin/pages/` + pageIDStr + `/publish" style="display:inline;">
                                ` + csrfToken + `
                                <button type="submit" class="btn btn-success">Publish</button>
                            </form>`
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Edit Page - ` + page.Title + `</title>
    <style>
        ` + GetDesignSystemCSS() + `

        body {
            padding: 0;
        }

        .page-layout {
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }

        .header {
            background: var(--color-bg-card);
            border-bottom: 1px solid var(--color-border);
            padding: var(--spacing-base) var(--spacing-md);
            box-shadow: var(--shadow-sm);
            position: sticky;
            top: 0;
            z-index: 10;
        }

        .header-content {
            max-width: 1200px;
            margin: 0 auto;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .header-left h1 {
            font-size: 18px;
            color: var(--color-text-primary);
        }

        .header-right {
            display: flex;
            align-items: center;
            gap: var(--spacing-base);
            font-size: 14px;
        }

        .header-right small {
            color: var(--color-text-secondary);
        }

        .logout-btn {
            background: transparent;
            color: var(--color-accent);
            padding: var(--spacing-sm) var(--spacing-base);
            font-size: 14px;
        }

        .logout-btn:hover {
            color: var(--color-accent-hover);
        }

        .container {
            flex: 1;
            max-width: 1200px;
            margin: 0 auto;
            width: 100%;
            padding: var(--spacing-md);
        }

        .back-link {
            display: inline-block;
            margin-bottom: var(--spacing-md);
            color: var(--color-accent);
            font-size: 14px;
        }

        .page-header {
            margin-bottom: var(--spacing-lg);
        }

        .page-title-section {
            background: var(--color-bg-card);
            padding: var(--spacing-md);
            border-radius: var(--radius-base);
            border: 1px solid var(--color-border);
            margin-bottom: var(--spacing-md);
            display: flex;
            flex-wrap: wrap;
            gap: var(--spacing-md);
            align-items: flex-end;
        }

        .page-title-section > form {
            flex: 1;
            min-width: 250px;
        }

        .page-title-section input {
            width: 100%;
            font-size: 20px;
            font-weight: 600;
            padding: var(--spacing-base);
            margin-bottom: var(--spacing-base);
        }

        .page-title-section > form:last-of-type {
            margin-top: 0;
        }

        .page-actions {
            display: flex;
            gap: var(--spacing-base);
        }

        .btn {
            background: var(--color-accent);
            color: white;
            padding: var(--spacing-sm) var(--spacing-md);
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
            transition: background var(--transition);
        }

        .btn:hover {
            background: var(--color-accent-hover);
        }

        .btn-secondary {
            background: var(--color-text-secondary);
            color: white;
        }

        .btn-secondary:hover {
            background: #5a6268;
        }

        .btn-success {
            background: var(--color-success);
        }

        .btn-success:hover {
            background: #059669;
        }

        .section {
            margin-bottom: var(--spacing-lg);
        }

        .section h2 {
            font-size: 18px;
            margin-bottom: var(--spacing-md);
            color: var(--color-text-primary);
        }

        .blocks-list {
            display: flex;
            flex-direction: column;
            gap: var(--spacing-base);
        }

        .block-item {
            background: var(--color-bg-card);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-base);
            padding: var(--spacing-base);
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            gap: var(--spacing-md);
            transition: box-shadow var(--transition), background var(--transition);
        }

        .block-item:hover {
            box-shadow: var(--shadow-md);
            background: #fafbfc;
        }

        .block-info {
            flex: 1;
            min-width: 0;
        }

        .block-type {
            font-weight: 600;
            margin-bottom: var(--spacing-sm);
            font-size: 14px;
            color: var(--color-text-primary);
        }

        .block-preview {
            font-size: 13px;
            color: var(--color-text-secondary);
            font-family: "Monaco", "Courier New", monospace;
            background: #f8f9fa;
            padding: var(--spacing-sm);
            border-radius: var(--radius-sm);
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            max-width: 100%;
        }

        .block-actions {
            display: flex;
            gap: var(--spacing-sm);
            align-items: center;
            flex-shrink: 0;
        }

        .btn-icon {
            padding: var(--spacing-sm) calc(var(--spacing-sm) * 1.25);
            font-size: 14px;
            background: var(--color-text-secondary);
            color: white;
            border: none;
            border-radius: var(--radius-sm);
            cursor: pointer;
            transition: background var(--transition);
        }

        .btn-icon:hover {
            background: #4b5563;
        }

        .btn-icon:disabled {
            opacity: 0.3;
            cursor: not-allowed;
        }

        .btn-small {
            padding: var(--spacing-sm) var(--spacing-base);
            font-size: 13px;
            background: var(--color-accent);
            color: white;
            text-decoration: none;
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            transition: background var(--transition);
        }

        .btn-small:hover {
            background: var(--color-accent-hover);
        }

        .btn-danger {
            background: var(--color-danger);
        }

        .btn-danger:hover {
            background: #dc2626;
        }

        .empty-state {
            padding: var(--spacing-lg);
            text-align: center;
            color: var(--color-text-secondary);
            border: 2px dashed var(--color-border);
            border-radius: var(--radius-base);
            background: #fafbfc;
        }

        .add-block {
            display: flex;
            flex-wrap: wrap;
            gap: var(--spacing-base);
            padding: var(--spacing-md);
            background: var(--color-bg-card);
            border-radius: var(--radius-base);
            border: 1px solid var(--color-border);
        }

        .add-block .btn {
            padding: var(--spacing-sm) var(--spacing-md);
            font-size: 13px;
        }

        .btn-text { background: var(--color-accent); }
        .btn-heading { background: #6c757d; }
        .btn-image { background: #17a2b8; }
        .btn-quote { background: #6f42c1; }
        .btn-button { background: var(--color-success); }
        .btn-video { background: var(--color-danger); }
        .btn-spacer { background: #e0e0e0; color: var(--color-text-primary); }
        .btn-columns { background: #f59e0b; }
    </style>
</head>
<body>
    <div class="page-layout">
        <div class="header">
            <div class="header-content">
                <div class="header-left">
                    <h1>Your Site</h1>
                </div>
                <div class="header-right">
                    <small>` + site.Subdomain + `</small>
                    <small>` + user.Email + `</small>
                    <form method="POST" action="/admin/logout" style="display:inline;">
                        ` + csrfToken + `
                        <button type="submit" class="logout-btn">Sign Out</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="container">
            <a href="/admin/pages?site=` + fmt.Sprintf("%d", site.ID) + `" class="back-link">← Back to Pages</a>

            <div class="page-header">
                <div class="page-title-section">
                    <form method="POST" action="/admin/pages/` + pageIDStr + `">
                        ` + csrfToken + `
                        <input type="text" name="title" value="` + page.Title + `" placeholder="Page Title" required>
                        <div class="page-actions">
                            <button type="submit" class="btn">Save Draft</button>
                        </div>
                    </form>
                    ` + publishButton + `
                </div>
            </div>

            <div class="section">
                <h2>Content Blocks</h2>
                <div class="blocks-list">
                    ` + blocksHTML + `
                </div>
                <div class="add-block">
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="text">
                        <button type="submit" class="btn btn-text">+ Text</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="heading">
                        <button type="submit" class="btn btn-heading">+ Heading</button>
                    </form>
                    <a href="/admin/pages/` + pageIDStr + `/blocks/new-image" class="btn btn-image">+ Image</a>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="quote">
                        <button type="submit" class="btn btn-quote">+ Quote</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="button">
                        <button type="submit" class="btn btn-button">+ Button</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="video">
                        <button type="submit" class="btn btn-video">+ Video</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="spacer">
                        <button type="submit" class="btn btn-spacer">+ Spacer</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="contact">
                        <button type="submit" class="btn btn-contact">+ Contact Form</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        ` + csrfToken + `
                        <input type="hidden" name="type" value="columns">
                        <button type="submit" class="btn btn-columns">+ Columns</button>
                    </form>
                </div>
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

	// Re-index the page in FTS
	if err := search.IndexPage(db.GetDB(), &page); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to index page %d: %v\n", page.ID, err)
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

	// Index the page in FTS (now it's searchable)
	if err := search.IndexPage(db.GetDB(), &page); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to index page %d: %v\n", page.ID, err)
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

	// Remove from FTS index (no longer searchable)
	if err := search.IndexPage(db.GetDB(), &page); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to update index for page %d: %v\n", page.ID, err)
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

	// Remove from FTS index
	if err := search.RemovePageFromIndex(db.GetDB(), page.ID); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to remove page %d from index: %v\n", page.ID, err)
	}

	// Redirect back to dashboard
	c.Redirect(http.StatusFound, "/admin/dashboard")
}

// NewImageBlockFormHandler displays the form for adding a new image block
func NewImageBlockFormHandler(c *gin.Context) {
	// Get CSRF token
	csrfToken := middleware.GetCSRFTokenHTML(c)

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
</head>
<body>
    <div class="container">
        <a href="/admin/pages/` + pageIDStr + `/edit" class="back-link">← Back to Edit Page</a>

        <h1>Add Image Block</h1>

        <form id="imageBlockForm" method="POST" action="/admin/pages/` + pageIDStr + `/blocks" onsubmit="handleSubmit(event)">
            ` + csrfToken + `
            <div class="form-group">
                <label>Select Image Source</label>
                <div style="display: flex; gap: 10px; align-items: center; margin-bottom: 15px;">
                    <button type="button" class="btn" onclick="document.getElementById('imageFile').click()">Upload File</button>
                    <span>or</span>
                    <button type="button" class="btn" style="background: #28a745;" onclick="openMediaPicker()">Choose from Library</button>
                </div>
                <input type="file" id="imageFile" accept="image/*" onchange="handleImageUpload()" style="display: none;">
                <p class="help-text">Supported formats: JPG, PNG, GIF, WebP</p>
                <div id="uploadProgress"></div>
            </div>

            <div id="preview" class="preview">
                <img id="previewImg" alt="Preview">
                <p id="previewFilename" class="help-text" style="word-break: break-all;"></p>
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

    <script>
        let uploadedImageURL = '';

        function openMediaPicker() {
            const width = 800;
            const height = 600;
            const left = (window.screen.width / 2) - (width / 2);
            const top = (window.screen.height / 2) - (height / 2);
            
            window.open(
                '/admin/media/picker',
                'mediaPicker',
                'width=' + width + ',height=' + height + ',left=' + left + ',top=' + top + ',scrollbars=yes'
            );
        }

        // Listen for message from picker
        window.addEventListener('message', function(event) {
            if (event.origin !== window.location.origin) return;

            if (event.data && event.data.type === 'image-selected') {
                uploadedImageURL = event.data.url;
                
                // Show preview
                const preview = document.getElementById('preview');
                const previewImg = document.getElementById('previewImg');
                const previewFilename = document.getElementById('previewFilename');
                
                previewImg.src = event.data.url;
                previewFilename.textContent = 'Selected: ' + event.data.filename;
                preview.style.display = 'block';

                // Enable submit
                document.getElementById('submitBtn').disabled = false;
            }
        });

        async function handleImageUpload() {
            const fileInput = document.getElementById('imageFile');
            const file = fileInput.files[0];
            if (!file) return;

            const formData = new FormData();
            formData.append('image', file);

            const progress = document.getElementById('uploadProgress');
            progress.style.display = 'block';
            progress.textContent = 'Uploading...';
            progress.style.background = '#e7f3ff';
            progress.style.color = '#0066cc';

            const csrfToken = decodeURIComponent(
                document.cookie
                    .split('; ')
                    .find(row => row.startsWith('csrf_token='))
                    ?.substring('csrf_token='.length) || ''
            );

            try {
                const response = await fetch('/admin/upload/image', {
                    method: 'POST',
                    headers: { 'X-CSRF-Token': csrfToken },
                    body: formData
                });

                if (!response.ok) throw new Error('Upload failed');

                const data = await response.json();
                uploadedImageURL = data.url;

                // Show preview
                const preview = document.getElementById('preview');
                const previewImg = document.getElementById('previewImg');
                const previewFilename = document.getElementById('previewFilename');
                
                previewImg.src = data.url;
                previewFilename.textContent = 'Uploaded: ' + file.name;
                preview.style.display = 'block';

                progress.textContent = 'Upload complete!';
                progress.style.background = '#d4edda';
                progress.style.color = '#155724';

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
                alert('Please select or upload an image first');
                return;
            }

            const alt = document.getElementById('altText').value;
            const caption = document.getElementById('caption').value;

            const blockData = {
                url: uploadedImageURL,
                alt: alt,
                caption: caption
            };

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
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// PagesListHandler shows the list of pages for a site
func PagesListHandler(c *gin.Context) {
	// Get CSRF token
	csrfToken := middleware.GetCSRFTokenHTML(c)

	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}
	user := userVal.(*models.User)

	// Get site ID from query parameter
	siteIDStr := c.Query("site")
	if siteIDStr == "" {
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	siteID, err := strconv.ParseUint(siteIDStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid site ID")
		return
	}

	// Get site from context (set by middleware)
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Verify site ID matches
	if site.ID != uint(siteID) {
		c.String(http.StatusForbidden, "You don't have access to this site")
		return
	}

	// Load all pages for this site
	var pages []models.Page
	db.GetDB().Where("site_id = ?", site.ID).Order("slug ASC").Find(&pages)

	// Build pages list HTML
	var pagesList string
	homepageExists := false

	for _, page := range pages {
		if page.Slug == "/" {
			homepageExists = true
			status := "Draft"
			if page.Published {
				status = "Published"
			}
			pagesList += `
				<div class="page-item">
					<strong>Homepage</strong> <span class="status">` + status + `</span>
					<div class="actions">
						<a href="/admin/pages/` + strconv.FormatUint(uint64(page.ID), 10) + `/edit" class="btn-small">Edit</a>
					</div>
				</div>
			`
		} else {
			status := "Draft"
			if page.Published {
				status = "Published"
			}
			pagesList += `
				<div class="page-item">
					<strong>` + page.Title + `</strong> <code>` + page.Slug + `</code> <span class="status">` + status + `</span>
					<div class="actions">
						<a href="/admin/pages/` + strconv.FormatUint(uint64(page.ID), 10) + `/edit" class="btn-small">Edit</a>
						<form method="POST" action="/admin/pages/` + strconv.FormatUint(uint64(page.ID), 10) + `/delete" style="display:inline;" onsubmit="return confirm('Delete this page?')">
							` + csrfToken + `
							<button type="submit" class="btn-small btn-danger">Delete</button>
						</form>
					</div>
				</div>
			`
		}
	}

	if !homepageExists {
		pagesList += `
			<div class="page-item placeholder">
				<em>No homepage yet</em>
				<form method="POST" action="/admin/pages" style="display:inline;">
					` + csrfToken + `
					<input type="hidden" name="slug" value="/">
					<input type="hidden" name="title" value="` + site.Subdomain + `">
					<button type="submit" class="btn-small">Create Homepage</button>
				</form>
			</div>
		`
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Pages - ` + site.Subdomain + `</title>
    <style>
        ` + GetDesignSystemCSS() + `

        body { padding: 0; }

        .dashboard-layout {
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }

        .header {
            background: var(--color-bg-card);
            border-bottom: 1px solid var(--color-border);
            padding: var(--spacing-base) var(--spacing-md);
            box-shadow: var(--shadow-sm);
            position: sticky;
            top: 0;
            z-index: 10;
        }

        .header-content {
            max-width: 1200px;
            margin: 0 auto;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .header-left {
            display: flex;
            align-items: center;
            gap: var(--spacing-base);
        }

        .header-left h1 {
            font-size: 18px;
            color: var(--color-text-primary);
            margin: 0;
        }

        .header-left a {
            color: var(--color-accent);
            text-decoration: none;
            font-size: 14px;
        }

        .header-right {
            display: flex;
            align-items: center;
            gap: var(--spacing-base);
        }

        .header-right small {
            color: var(--color-text-secondary);
            font-size: 14px;
        }

        .logout-btn {
            background: var(--color-accent);
            color: white;
            padding: var(--spacing-sm) var(--spacing-md);
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
        }

        .logout-btn:hover {
            background: var(--color-accent-hover);
        }

        .container {
            flex: 1;
            max-width: 1200px;
            margin: 0 auto;
            width: 100%;
            padding: var(--spacing-md);
        }

        .section {
            margin-bottom: var(--spacing-lg);
        }

        .section-title {
            font-size: 18px;
            font-weight: 600;
            margin-bottom: var(--spacing-md);
            color: var(--color-text-primary);
        }

        .page-item {
            padding: var(--spacing-md);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-base);
            margin-bottom: var(--spacing-base);
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: var(--spacing-md);
        }

        .page-item.placeholder {
            border-style: dashed;
            color: var(--color-text-secondary);
        }

        .status {
            font-size: 12px;
            padding: 2px 8px;
            background: var(--color-warning);
            border-radius: 3px;
            margin-left: var(--spacing-base);
        }

        .actions {
            display: flex;
            gap: 8px;
        }

        .btn {
            padding: var(--spacing-sm) var(--spacing-md);
            background: var(--color-accent);
            color: white;
            text-decoration: none;
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
        }

        .btn:hover {
            background: var(--color-accent-hover);
        }

        .btn-small {
            padding: var(--spacing-sm) var(--spacing-base);
            font-size: 13px;
            background: var(--color-accent);
            color: white;
            text-decoration: none;
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            transition: background var(--transition);
        }

        .btn-small:hover {
            background: var(--color-accent-hover);
        }

        .btn-danger {
            background: var(--color-danger);
        }

        .btn-danger:hover {
            background: #c82333;
        }

        code {
            background: var(--color-bg-primary);
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 13px;
        }
    </style>
</head>
<body>
    <div class="dashboard-layout">
        <div class="header">
            <div class="header-content">
                <div class="header-left">
                    <h1>` + site.Subdomain + `</h1>
                    <a href="/admin/dashboard">← Back to Camps</a>
                </div>
                <div class="header-right">
                    <small>` + user.Email + `</small>
                    <form method="POST" action="/admin/logout" style="display:inline;">
                        ` + csrfToken + `
                        <button type="submit" class="logout-btn">Sign Out</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="container">
            <div class="section">
                <h2 class="section-title">Pages</h2>
                ` + pagesList + `
                <div style="margin-top: 15px;">
                    <a href="/admin/pages/new" class="btn">+ Create New Page</a>
                    <a href="/admin/menu" class="btn" style="background: #17a2b8; margin-left: 10px;">Navigation Menu</a>
                    <a href="/admin/settings" class="btn" style="background: #6366f1; margin-left: 10px;">Theme Settings</a>
                    <a href="/admin/export?site=` + fmt.Sprintf("%d", site.ID) + `" class="btn" style="background: #10b981; margin-left: 10px;">Download Site</a>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
