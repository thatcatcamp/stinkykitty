package handlers

import (
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
