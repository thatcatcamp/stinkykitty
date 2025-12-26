package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/search"
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
        <form method="POST" action="/admin/pages/%s/blocks/%s">
            <input type="hidden" name="url" value="%s">
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
