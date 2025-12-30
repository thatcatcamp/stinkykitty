// SPDX-License-Identifier: MIT
package handlers

import (
	"encoding/json"
	"fmt"
	htmlpkg "html"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/media"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/search"
	"github.com/thatcatcamp/stinkykitty/internal/uploads"
)

// CreateBlockHandler creates a new block for a page
func CreateBlockHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID from URL parameter
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	// Load the page
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

	// Get block type from POST form
	blockType := c.PostForm("type")
	validTypes := map[string]bool{
		"text":    true,
		"image":   true,
		"heading": true,
		"quote":   true,
		"button":  true,
		"video":   true,
		"spacer":  true,
		"contact": true,
		"columns": true,
	}
	if !validTypes[blockType] {
		c.String(http.StatusBadRequest, "Invalid block type")
		return
	}

	// Calculate the next order number
	// Find max order of existing blocks + 1, or 0 if no blocks
	var maxOrder struct {
		MaxOrder *int
	}
	db.GetDB().Model(&models.Block{}).
		Where("page_id = ?", pageID).
		Select("MAX(\"order\") as max_order").
		Scan(&maxOrder)

	nextOrder := 0
	if maxOrder.MaxOrder != nil {
		nextOrder = *maxOrder.MaxOrder + 1
	}

	// Determine initial data based on block type
	var blockData string
	switch blockType {
	case "text":
		blockData = `{"content":""}`
	case "image":
		blockData = c.PostForm("data")
		if blockData == "" {
			c.String(http.StatusBadRequest, "Image block data is required")
			return
		}
	case "heading":
		blockData = `{"level":2,"text":""}`
	case "quote":
		blockData = `{"quote":"","author":""}`
	case "button":
		blockData = `{"text":"Click Here","url":"","style":"primary"}`
	case "video":
		blockData = `{"url":""}`
	case "spacer":
		blockData = `{"height":40}`
	case "contact":
		blockData = `{"title":"Get in Touch","subtitle":""}`
	case "columns":
		blockData = `{"column_count":2,"columns":[{"content":""},{"content":""}]}`
	}

	// Create new block
	block := models.Block{
		PageID: uint(pageID),
		Type:   blockType,
		Order:  nextOrder,
		Data:   blockData,
	}

	if err := db.GetDB().Create(&block).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create block")
		return
	}

	// Re-index the page in FTS
	if err := search.IndexPage(db.GetDB(), &page); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to index page %d: %v\n", page.ID, err)
	}

	// For blocks that need immediate editing, redirect to edit page
	// For blocks that are ready to use (image, spacer), redirect to page editor
	needsEditing := blockType != "image" && blockType != "spacer"
	if needsEditing {
		c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/blocks/"+strconv.Itoa(int(block.ID))+"/edit")
	} else {
		c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
	}
}

// EditBlockHandler shows a form to edit a block
func EditBlockHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID and block ID from URL parameters
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	blockIDStr := c.Param("block_id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid block ID")
		return
	}

	// Load the page
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

	// Load the block
	var block models.Block
	result = db.GetDB().Where("id = ? AND page_id = ?", blockID, pageID).First(&block)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Block not found")
		return
	}

	// Parse JSON Data field based on block type
	var html string
	if block.Type == "text" {
		var content string
		var data struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(block.Data), &data); err == nil {
			content = data.Content
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Edit Block</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 40px auto;
            padding: 0 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-top: 0;
        }
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #555;
        }
        textarea {
            width: 100%%;
            min-height: 300px;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-family: system-ui, -apple-system, sans-serif;
            font-size: 14px;
            box-sizing: border-box;
        }
        textarea:focus {
            outline: none;
            border-color: #2563eb;
        }
        .button-group {
            margin-top: 20px;
            display: flex;
            gap: 10px;
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
            background: #2563eb;
            color: white;
        }
        button[type="submit"]:hover {
            background: #1d4ed8;
        }
        a.cancel {
            padding: 10px 20px;
            background: #6b7280;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 600;
        }
        a.cancel:hover {
            background: #4b5563;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Text Block</h1>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="content">Content:</label>
            <textarea id="content" name="content" rows="10">%s</textarea>
            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr, content, pageIDStr)
	} else if block.Type == "image" {
		var imageData struct {
			URL     string `json:"url"`
			Alt     string `json:"alt"`
			Caption string `json:"caption"`
		}
		if err := json.Unmarshal([]byte(block.Data), &imageData); err != nil {
			c.String(http.StatusInternalServerError, "Failed to parse image data")
			return
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Edit Image Block</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            max-width: 800px;
            margin: 40px auto;
            padding: 0 20px;
            background: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-top: 0;
        }
        .preview {
            margin-bottom: 20px;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 4px;
        }
        .preview img {
            max-width: 100%%;
            height: auto;
            display: block;
        }
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #555;
        }
        input[type="text"] {
            width: 100%%;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-family: system-ui, -apple-system, sans-serif;
            font-size: 14px;
            box-sizing: border-box;
            margin-bottom: 15px;
        }
        input[type="text"]:focus {
            outline: none;
            border-color: #2563eb;
        }
        .button-group {
            margin-top: 20px;
            display: flex;
            gap: 10px;
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
            background: #2563eb;
            color: white;
        }
        button[type="submit"]:hover {
            background: #1d4ed8;
        }
        a.cancel {
            padding: 10px 20px;
            background: #6b7280;
            color: white;
            text-decoration: none;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 600;
        }
        a.cancel:hover {
            background: #4b5563;
        }
        .help-text {
            font-size: 12px;
            color: #666;
            margin-top: -10px;
            margin-bottom: 15px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Image Block</h1>
        <div class="preview">
            <img src="%s" alt="%s">
        </div>
        <form method="POST" action="/admin/pages/%s/blocks/%s" enctype="multipart/form-data">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <input type="hidden" name="url" value="%s">

            <label for="image">Upload New Image (optional):</label>
            <input type="file" id="image" name="image" accept="image/*">
            <button type="button" onclick="openMediaPicker()" style="margin-top: 8px; padding: 10px 20px; background: #6b7280; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600;">
                Browse Library
            </button>
            <input type="hidden" id="selected-image-url" name="selected_image_url">
            <p class="help-text">Upload a new image or browse the media library.</p>

            <label for="alt">Alt Text:</label>
            <input type="text" id="alt" name="alt" value="%s" required>
            <p class="help-text">Required for accessibility. Describe what's in the image.</p>

            <label for="caption">Caption (optional):</label>
            <input type="text" id="caption" name="caption" value="%s">
            <p class="help-text">Optional caption to display below the image.</p>

            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
    <script>
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
    </script>
</body>
</html>`, imageData.URL, imageData.Alt, pageIDStr, blockIDStr, imageData.URL, imageData.Alt, imageData.Caption, pageIDStr)
	} else if block.Type == "heading" {
		var headingData struct {
			Level int    `json:"level"`
			Text  string `json:"text"`
		}
		if err := json.Unmarshal([]byte(block.Data), &headingData); err != nil {
			c.String(http.StatusInternalServerError, "Failed to parse heading data")
			return
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Edit Heading Block</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #555; }
        input[type="text"], select { width: 100%%; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; margin-bottom: 15px; }
        input:focus, select:focus { outline: none; border-color: #2563eb; }
        .button-group { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
        button[type="submit"] { background: #2563eb; color: white; }
        button[type="submit"]:hover { background: #1d4ed8; }
        a.cancel { padding: 10px 20px; background: #6b7280; color: white; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 600; }
        a.cancel:hover { background: #4b5563; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Heading Block</h1>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="level">Heading Level:</label>
            <select id="level" name="level">
                <option value="2"%s>H2 - Large Heading</option>
                <option value="3"%s>H3 - Medium Heading</option>
                <option value="4"%s>H4 - Small Heading</option>
                <option value="5"%s>H5 - Smaller Heading</option>
                <option value="6"%s>H6 - Smallest Heading</option>
            </select>
            <label for="text">Heading Text:</label>
            <input type="text" id="text" name="text" value="%s" required>
            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr,
			func() string {
				if headingData.Level == 2 {
					return " selected"
				}
				return ""
			}(),
			func() string {
				if headingData.Level == 3 {
					return " selected"
				}
				return ""
			}(),
			func() string {
				if headingData.Level == 4 {
					return " selected"
				}
				return ""
			}(),
			func() string {
				if headingData.Level == 5 {
					return " selected"
				}
				return ""
			}(),
			func() string {
				if headingData.Level == 6 {
					return " selected"
				}
				return ""
			}(),
			headingData.Text, pageIDStr)
	} else if block.Type == "quote" {
		var quoteData struct {
			Quote  string `json:"quote"`
			Author string `json:"author"`
		}
		if err := json.Unmarshal([]byte(block.Data), &quoteData); err != nil {
			c.String(http.StatusInternalServerError, "Failed to parse quote data")
			return
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Edit Quote Block</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #555; }
        textarea, input[type="text"] { width: 100%%; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; margin-bottom: 15px; font-family: system-ui; }
        textarea { min-height: 120px; }
        textarea:focus, input:focus { outline: none; border-color: #2563eb; }
        .button-group { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
        button[type="submit"] { background: #2563eb; color: white; }
        button[type="submit"]:hover { background: #1d4ed8; }
        a.cancel { padding: 10px 20px; background: #6b7280; color: white; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 600; }
        a.cancel:hover { background: #4b5563; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Quote Block</h1>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="quote">Quote:</label>
            <textarea id="quote" name="quote" required>%s</textarea>
            <label for="author">Author (optional):</label>
            <input type="text" id="author" name="author" value="%s">
            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr, quoteData.Quote, quoteData.Author, pageIDStr)
	} else if block.Type == "button" {
		var buttonData struct {
			Text  string `json:"text"`
			URL   string `json:"url"`
			Style string `json:"style"`
		}
		if err := json.Unmarshal([]byte(block.Data), &buttonData); err != nil {
			c.String(http.StatusInternalServerError, "Failed to parse button data")
			return
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Edit Button Block</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #555; }
        input[type="text"], select { width: 100%%; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; margin-bottom: 15px; }
        input:focus, select:focus { outline: none; border-color: #2563eb; }
        .button-group { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
        button[type="submit"] { background: #2563eb; color: white; }
        button[type="submit"]:hover { background: #1d4ed8; }
        a.cancel { padding: 10px 20px; background: #6b7280; color: white; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 600; }
        a.cancel:hover { background: #4b5563; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Button Block</h1>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="text">Button Text:</label>
            <input type="text" id="text" name="text" value="%s" required>
            <label for="url">Link URL:</label>
            <input type="text" id="url" name="url" value="%s" required placeholder="https://example.com or /page">
            <label for="style">Button Style:</label>
            <select id="style" name="style">
                <option value="primary"%s>Primary (Blue)</option>
                <option value="secondary"%s>Secondary (Gray)</option>
            </select>
            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr, buttonData.Text, buttonData.URL,
			func() string {
				if buttonData.Style == "primary" {
					return " selected"
				}
				return ""
			}(),
			func() string {
				if buttonData.Style == "secondary" {
					return " selected"
				}
				return ""
			}(),
			pageIDStr)
	} else if block.Type == "video" {
		var videoData struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(block.Data), &videoData); err != nil {
			c.String(http.StatusInternalServerError, "Failed to parse video data")
			return
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Edit Video Block</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #555; }
        input[type="text"] { width: 100%%; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; margin-bottom: 15px; }
        input:focus { outline: none; border-color: #2563eb; }
        .help-text { font-size: 12px; color: #666; margin-top: -10px; margin-bottom: 15px; }
        .button-group { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
        button[type="submit"] { background: #2563eb; color: white; }
        button[type="submit"]:hover { background: #1d4ed8; }
        a.cancel { padding: 10px 20px; background: #6b7280; color: white; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 600; }
        a.cancel:hover { background: #4b5563; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Video Block</h1>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="url">Video URL:</label>
            <input type="text" id="url" name="url" value="%s" required placeholder="YouTube or Vimeo URL">
            <p class="help-text">Paste a YouTube or Vimeo video URL (e.g., https://www.youtube.com/watch?v=...)</p>
            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr, videoData.URL, pageIDStr)
	} else if block.Type == "spacer" {
		var spacerData struct {
			Height int `json:"height"`
		}
		if err := json.Unmarshal([]byte(block.Data), &spacerData); err != nil {
			c.String(http.StatusInternalServerError, "Failed to parse spacer data")
			return
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Edit Spacer Block</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #555; }
        input[type="number"] { width: 100%%; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; margin-bottom: 15px; }
        input:focus { outline: none; border-color: #2563eb; }
        .help-text { font-size: 12px; color: #666; margin-top: -10px; margin-bottom: 15px; }
        .button-group { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
        button[type="submit"] { background: #2563eb; color: white; }
        button[type="submit"]:hover { background: #1d4ed8; }
        a.cancel { padding: 10px 20px; background: #6b7280; color: white; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 600; }
        a.cancel:hover { background: #4b5563; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Spacer Block</h1>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="height">Height (pixels):</label>
            <input type="number" id="height" name="height" value="%d" required min="1" max="500">
            <p class="help-text">Vertical spacing in pixels (recommended: 20-100)</p>
            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr, spacerData.Height, pageIDStr)
	} else if block.Type == "contact" {
		// Parse contact block data
		var contactData struct {
			Title    string `json:"title"`
			Subtitle string `json:"subtitle"`
		}
		if err := json.Unmarshal([]byte(block.Data), &contactData); err != nil {
			contactData.Title = "Get in Touch"
		}

		// Escape contact form data before using
		contactTitle := contactData.Title
		contactSubtitle := contactData.Subtitle
		// Note: We'll escape these in the template using Go's escaping

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Edit Contact Block</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #555; }
        input[type="text"],
        textarea { width: 100%%; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; margin-bottom: 15px; }
        input:focus,
        textarea:focus { outline: none; border-color: #2563eb; }
        .help-text { font-size: 12px; color: #666; margin-top: -10px; margin-bottom: 15px; }
        .button-group { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
        button[type="submit"] { background: #2563eb; color: white; }
        button[type="submit"]:hover { background: #1d4ed8; }
        a.cancel { padding: 10px 20px; background: #6b7280; color: white; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 600; }
        a.cancel:hover { background: #4b5563; }
        .note { background: #f0f4f8; padding: 15px; border-radius: 4px; margin-bottom: 20px; color: #555; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit Contact Form Block</h1>
        <div class="note">
            <strong>Note:</strong> This block displays a contact form where visitors can send you messages. Their email address is shown to you, but not to other visitors.
        </div>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="title">Form Title:</label>
            <input type="text" id="title" name="title" value="%s" placeholder="Get in Touch">
            <p class="help-text">The heading displayed above the contact form</p>

            <label for="subtitle">Form Subtitle (optional):</label>
            <textarea id="subtitle" name="subtitle" placeholder="We'd love to hear from you!">%s</textarea>
            <p class="help-text">Additional text displayed under the title</p>

            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr, contactTitle, contactSubtitle, pageIDStr)
	} else if block.Type == "columns" {
		// Parse columns block data
		var columnsData struct {
			ColumnCount int `json:"column_count"`
			Columns     []struct {
				Content string `json:"content"`
			} `json:"columns"`
		}
		if err := json.Unmarshal([]byte(block.Data), &columnsData); err != nil {
			columnsData.ColumnCount = 2
			columnsData.Columns = make([]struct {
				Content string `json:"content"`
			}, 2)
		}

		// Ensure we have columns matching the column count
		if len(columnsData.Columns) == 0 {
			if columnsData.ColumnCount < 2 || columnsData.ColumnCount > 4 {
				columnsData.ColumnCount = 2
			}
			columnsData.Columns = make([]struct {
				Content string `json:"content"`
			}, columnsData.ColumnCount)
		}

		// Build column inputs HTML
		var columnInputsHTML string
		for i, col := range columnsData.Columns {
			columnInputsHTML += fmt.Sprintf(`
				<div class="column-input">
					<label for="column_%d">Column %d:</label>
					<div class="toolbar">
						<button type="button" class="toolbar-btn" onclick="insertImage(%d)" title="Insert Image">🖼️ Image</button>
						<button type="button" class="toolbar-btn" onclick="insertButton(%d)" title="Insert Button">🔘 Button</button>
						<button type="button" class="toolbar-btn" onclick="insertLink(%d)" title="Insert Link">🔗 Link</button>
						<button type="button" class="toolbar-btn" onclick="insertHeading(%d)" title="Insert Heading">📝 Heading</button>
						<button type="button" class="toolbar-btn" onclick="makeText(%d, 'bold')" title="Bold Text"><b>B</b></button>
						<button type="button" class="toolbar-btn" onclick="makeText(%d, 'italic')" title="Italic Text"><i>I</i></button>
					</div>
					<textarea id="column_%d" name="column_%d" rows="8">%s</textarea>
				</div>
			`, i, i+1, i, i, i, i, i, i, i, i, htmlpkg.EscapeString(col.Content))
		}

		html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Edit Columns Block</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; background: #f5f5f5; }
        .container { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-top: 0; }
        label { display: block; margin-bottom: 8px; font-weight: 600; color: #555; }
        select, textarea { width: 100%%; padding: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; margin-bottom: 15px; font-family: system-ui; }
        textarea { min-height: 100px; }
        select:focus, textarea:focus { outline: none; border-color: #2563eb; }
        .help-text { font-size: 12px; color: #666; margin-top: -10px; margin-bottom: 15px; }
        .button-group { margin-top: 20px; display: flex; gap: 10px; }
        button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600; }
        button[type="submit"] { background: #2563eb; color: white; }
        button[type="submit"]:hover { background: #1d4ed8; }
        a.cancel { padding: 10px 20px; background: #6b7280; color: white; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 600; }
        a.cancel:hover { background: #4b5563; }
        .columns-container { display: grid; grid-template-columns: 1fr; gap: 15px; margin-bottom: 20px; }
        .column-input { background: #f8f9fa; padding: 15px; border-radius: 4px; }
        .note { background: #f0f4f8; padding: 15px; border-radius: 4px; margin-bottom: 20px; color: #555; }
        .toolbar { display: flex; gap: 5px; margin-bottom: 8px; flex-wrap: wrap; }
        .toolbar-btn { padding: 6px 12px; background: #e5e7eb; border: 1px solid #d1d5db; border-radius: 4px; cursor: pointer; font-size: 13px; transition: all 0.2s; }
        .toolbar-btn:hover { background: #d1d5db; }
        .toolbar-btn:active { transform: scale(0.95); }
    </style>
    <script>
        function insertAtCursor(textareaId, text) {
            const textarea = document.getElementById('column_' + textareaId);
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            const currentText = textarea.value;
            textarea.value = currentText.substring(0, start) + text + currentText.substring(end);
            textarea.focus();
            textarea.selectionStart = textarea.selectionEnd = start + text.length;
        }

        function insertImage(colIndex) {
            const url = prompt('Enter image URL (e.g., /uploads/image.jpg):');
            if (url) {
                const html = '<img src="' + url + '" style="width: 100%%; height: auto;">\n';
                insertAtCursor(colIndex, html);
            }
        }

        function insertButton(colIndex) {
            const text = prompt('Enter button text:');
            if (text) {
                const link = prompt('Enter button link (optional, press OK to skip):');
                let html;
                if (link) {
                    html = '<a href="' + link + '" style="display: inline-block; background: var(--color-accent, #2563eb); color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; box-shadow: 0 2px 4px rgba(0,0,0,0.1); transition: all 0.2s;">' + text + '</a>\n';
                } else {
                    html = '<button style="background: var(--color-accent, #2563eb); color: white; padding: 12px 24px; border: none; border-radius: 6px; cursor: pointer; font-weight: 600; box-shadow: 0 2px 4px rgba(0,0,0,0.1); transition: all 0.2s;">' + text + '</button>\n';
                }
                insertAtCursor(colIndex, html);
            }
        }

        function insertLink(colIndex) {
            const url = prompt('Enter link URL:');
            if (url) {
                const text = prompt('Enter link text:');
                if (text) {
                    const html = '<a href="' + url + '">' + text + '</a>';
                    insertAtCursor(colIndex, html);
                }
            }
        }

        function insertHeading(colIndex) {
            const text = prompt('Enter heading text:');
            if (text) {
                const html = '<h2>' + text + '</h2>\n';
                insertAtCursor(colIndex, html);
            }
        }

        function makeText(colIndex, style) {
            const textarea = document.getElementById('column_' + colIndex);
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            const selectedText = textarea.value.substring(start, end);

            if (selectedText) {
                let wrapped;
                if (style === 'bold') {
                    wrapped = '<strong>' + selectedText + '</strong>';
                } else if (style === 'italic') {
                    wrapped = '<em>' + selectedText + '</em>';
                }
                textarea.value = textarea.value.substring(0, start) + wrapped + textarea.value.substring(end);
                textarea.focus();
                textarea.selectionStart = start;
                textarea.selectionEnd = start + wrapped.length;
            } else {
                alert('Please select some text first!');
            }
        }

        function updateColumnInputs() {
            const count = parseInt(document.getElementById('column_count').value);
            const container = document.getElementById('columns-container');
            const currentCount = container.children.length;

            if (count > currentCount) {
                // Add new columns
                for (let i = currentCount; i < count; i++) {
                    const div = document.createElement('div');
                    div.className = 'column-input';
                    div.innerHTML = '<label for="column_' + i + '">Column ' + (i + 1) + ':</label>' +
                        '<div class="toolbar">' +
                        '<button type="button" class="toolbar-btn" onclick="insertImage(' + i + ')">🖼️ Image</button>' +
                        '<button type="button" class="toolbar-btn" onclick="insertButton(' + i + ')">🔘 Button</button>' +
                        '<button type="button" class="toolbar-btn" onclick="insertLink(' + i + ')">🔗 Link</button>' +
                        '<button type="button" class="toolbar-btn" onclick="insertHeading(' + i + ')">📝 Heading</button>' +
                        '<button type="button" class="toolbar-btn" onclick="makeText(' + i + ', \'bold\')"><b>B</b></button>' +
                        '<button type="button" class="toolbar-btn" onclick="makeText(' + i + ', \'italic\')"><i>I</i></button>' +
                        '</div>' +
                        '<textarea id="column_' + i + '" name="column_' + i + '" rows="8"></textarea>';
                    container.appendChild(div);
                }
            } else if (count < currentCount) {
                // Remove columns
                while (container.children.length > count) {
                    container.removeChild(container.lastChild);
                }
            }
        }
    </script>
</head>
<body>
    <div class="container">
        <h1>Edit Columns Block</h1>
        <div class="note">
            <strong>Note:</strong> Create a multi-column layout with 2, 3, or 4 columns. Content will be displayed side by side on larger screens.
        </div>
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            ` + middleware.GetCSRFTokenHTML(c) + `
            <label for="column_count">Number of Columns:</label>
            <select id="column_count" name="column_count" onchange="updateColumnInputs()">
                <option value="2"%s>2 Columns</option>
                <option value="3"%s>3 Columns</option>
                <option value="4"%s>4 Columns</option>
            </select>
            <p class="help-text">Select how many columns you want in this layout</p>

            <div id="columns-container" class="columns-container">
                %s
            </div>

            <div class="button-group">
                <button type="submit">Save &amp; Return</button>
                <a href="/admin/pages/%s/edit" class="cancel">Cancel</a>
            </div>
        </form>
    </div>
</body>
</html>`, pageIDStr, blockIDStr,
			func() string {
				if columnsData.ColumnCount == 2 {
					return " selected"
				}
				return ""
			}(),
			func() string {
				if columnsData.ColumnCount == 3 {
					return " selected"
				}
				return ""
			}(),
			func() string {
				if columnsData.ColumnCount == 4 {
					return " selected"
				}
				return ""
			}(),
			columnInputsHTML, pageIDStr)
	} else {
		c.String(http.StatusBadRequest, "Block type '%s' does not support editing yet", block.Type)
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// UpdateBlockHandler updates a block's content
func UpdateBlockHandler(c *gin.Context) {
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
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}
	user := userVal.(*models.User)

	// Get page ID and block ID from URL parameters
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	blockIDStr := c.Param("block_id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid block ID")
		return
	}

	// Load the page
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

	// Load the block
	var block models.Block
	result = db.GetDB().Where("id = ? AND page_id = ?", blockID, pageID).First(&block)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Block not found")
		return
	}

	// Update block data based on type
	switch block.Type {
	case "text":
		content := c.PostForm("content")
		data := map[string]string{"content": content}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)

	case "image":
		url := c.PostForm("url")
		alt := c.PostForm("alt")
		caption := c.PostForm("caption")

		// Check if image was selected from library
		selectedImageURL := c.PostForm("selected_image_url")
		if selectedImageURL != "" {
			// Use selected image from library
			url = selectedImageURL
		} else {
			// Check if new image was uploaded
			fileHeader, err := c.FormFile("image")
			if err == nil && fileHeader != nil {
				// Upload new image (reuse existing upload utility)
				webPath, err := uploads.SaveUploadedFile(fileHeader, site.SiteDir)
				if err != nil {
					c.String(http.StatusBadRequest, fmt.Sprintf("Failed to upload image: %v", err))
					return
				}
				url = webPath

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
			}
			// Otherwise, keep existing URL
		}

		data := map[string]string{
			"url":     url,
			"alt":     alt,
			"caption": caption,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)

	case "heading":
		levelStr := c.PostForm("level")
		text := c.PostForm("text")
		level, err := strconv.Atoi(levelStr)
		if err != nil || level < 2 || level > 6 {
			level = 2
		}
		data := map[string]interface{}{
			"level": level,
			"text":  text,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)

	case "quote":
		quote := c.PostForm("quote")
		author := c.PostForm("author")
		data := map[string]string{
			"quote":  quote,
			"author": author,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)

	case "button":
		text := c.PostForm("text")
		url := c.PostForm("url")
		style := c.PostForm("style")
		if style != "primary" && style != "secondary" {
			style = "primary"
		}
		data := map[string]string{
			"text":  text,
			"url":   url,
			"style": style,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)

	case "video":
		url := c.PostForm("url")
		data := map[string]string{"url": url}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)

	case "spacer":
		heightStr := c.PostForm("height")
		height, err := strconv.Atoi(heightStr)
		if err != nil || height < 1 || height > 500 {
			height = 40
		}
		data := map[string]int{"height": height}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)
	case "contact":
		title := c.PostForm("title")
		if title == "" {
			title = "Get in Touch"
		}
		subtitle := c.PostForm("subtitle")
		data := map[string]string{"title": title, "subtitle": subtitle}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)

	case "columns":
		columnCountStr := c.PostForm("column_count")
		columnCount, err := strconv.Atoi(columnCountStr)
		if err != nil || columnCount < 2 || columnCount > 4 {
			columnCount = 2
		}

		// Collect column contents
		columns := make([]map[string]string, columnCount)
		for i := 0; i < columnCount; i++ {
			content := c.PostForm(fmt.Sprintf("column_%d", i))
			columns[i] = map[string]string{"content": content}
		}

		data := map[string]interface{}{
			"column_count": columnCount,
			"columns":      columns,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to encode block data")
			return
		}
		block.Data = string(jsonData)
	}

	// Save to database
	if err := db.GetDB().Save(&block).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to update block")
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

// DeleteBlockHandler deletes a block from a page
func DeleteBlockHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID and block ID from URL parameters
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	blockIDStr := c.Param("block_id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid block ID")
		return
	}

	// Load the block
	var block models.Block
	result := db.GetDB().Where("id = ? AND page_id = ?", blockID, pageID).First(&block)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Block not found")
		return
	}

	// Load the page to verify ownership
	var page models.Page
	result = db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Delete the block from database
	if err := db.GetDB().Delete(&block).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete block")
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

// MoveBlockUpHandler moves a block up in the order (swaps with previous block)
func MoveBlockUpHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID and block ID from URL parameters
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	blockIDStr := c.Param("block_id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid block ID")
		return
	}

	// Load the block
	var block models.Block
	result := db.GetDB().Where("id = ? AND page_id = ?", blockID, pageID).First(&block)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Block not found")
		return
	}

	// Load the page to verify ownership
	var page models.Page
	result = db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Find the previous block (where order < current.Order, ordered DESC, limit 1)
	var previousBlock models.Block
	result = db.GetDB().Where("page_id = ? AND \"order\" < ?", pageID, block.Order).
		Order("\"order\" DESC").
		First(&previousBlock)

	// If no previous block found, this is already the first block - do nothing
	if result.Error != nil {
		c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
		return
	}

	// Swap the order values
	currentOrder := block.Order
	previousOrder := previousBlock.Order

	block.Order = previousOrder
	previousBlock.Order = currentOrder

	// Save both blocks
	if err := db.GetDB().Save(&block).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to update block order")
		return
	}

	if err := db.GetDB().Save(&previousBlock).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to update block order")
		return
	}

	// Redirect back to page editor
	c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
}

// MoveBlockDownHandler moves a block down in the order (swaps with next block)
func MoveBlockDownHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get page ID and block ID from URL parameters
	pageIDStr := c.Param("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	blockIDStr := c.Param("block_id")
	blockID, err := strconv.Atoi(blockIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid block ID")
		return
	}

	// Load the block
	var block models.Block
	result := db.GetDB().Where("id = ? AND page_id = ?", blockID, pageID).First(&block)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Block not found")
		return
	}

	// Load the page to verify ownership
	var page models.Page
	result = db.GetDB().Where("id = ?", pageID).First(&page)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Page not found")
		return
	}

	// Security check: verify page belongs to current site
	if page.SiteID != site.ID {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Find the next block (where order > current.Order, ordered ASC, limit 1)
	var nextBlock models.Block
	result = db.GetDB().Where("page_id = ? AND \"order\" > ?", pageID, block.Order).
		Order("\"order\" ASC").
		First(&nextBlock)

	// If no next block found, this is already the last block - do nothing
	if result.Error != nil {
		c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
		return
	}

	// Swap the order values
	currentOrder := block.Order
	nextOrder := nextBlock.Order

	block.Order = nextOrder
	nextBlock.Order = currentOrder

	// Save both blocks
	if err := db.GetDB().Save(&block).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to update block order")
		return
	}

	if err := db.GetDB().Save(&nextBlock).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to update block order")
		return
	}

	// Redirect back to page editor
	c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
}
