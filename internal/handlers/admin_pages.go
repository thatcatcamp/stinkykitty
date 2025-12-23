package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

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
