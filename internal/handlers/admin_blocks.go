package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
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
	pageIDStr := c.Param("page_id")
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

	// Get block type from POST form (currently only "text" is valid)
	blockType := c.PostForm("type")
	if blockType != "text" {
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

	// Create new block with empty Data: {"content":""}
	block := models.Block{
		PageID: uint(pageID),
		Type:   blockType,
		Order:  nextOrder,
		Data:   `{"content":""}`,
	}

	if err := db.GetDB().Create(&block).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create block")
		return
	}

	// Redirect to /admin/pages/:page_id/blocks/:id/edit for immediate editing
	c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/blocks/"+strconv.Itoa(int(block.ID))+"/edit")
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
	pageIDStr := c.Param("page_id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	blockIDStr := c.Param("id")
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

	// Parse JSON Data field to extract content for text blocks
	var content string
	if block.Type == "text" {
		var data struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(block.Data), &data); err == nil {
			content = data.Content
		}
	}

	// Render HTML form
	html := fmt.Sprintf(`<!DOCTYPE html>
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
	pageIDStr := c.Param("page_id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid page ID")
		return
	}

	blockIDStr := c.Param("id")
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

	// For text blocks: get content from POST form
	if block.Type == "text" {
		content := c.PostForm("content")

		// Update block.Data with JSON: {"content":"..."}
		data := map[string]string{
			"content": content,
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

	// Redirect back to page editor
	c.Redirect(http.StatusFound, "/admin/pages/"+pageIDStr+"/edit")
}
