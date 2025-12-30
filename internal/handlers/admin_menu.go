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
	"gorm.io/gorm"
)

// MenuHandler displays the navigation menu management page
func MenuHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get CSRF token
	csrfToken := middleware.GetCSRFTokenHTML(c)

	// Load all menu items ordered by position
	var menuItems []models.MenuItem
	db.GetDB().Where("site_id = ?", site.ID).
		Order("`order` ASC").
		Find(&menuItems)

	// Load all published pages for the dropdown
	var pages []models.Page
	db.GetDB().Where("site_id = ? AND published = ?", site.ID, true).
		Order("title ASC").
		Find(&pages)

	// Build menu items HTML
	var menuItemsHTML string
	for i, item := range menuItems {
		showMoveUp := i > 0
		showMoveDown := i < len(menuItems)-1

		moveUpBtn := ""
		if showMoveUp {
			moveUpBtn = `<form method="POST" action="/admin/menu/` + strconv.Itoa(int(item.ID)) + `/move-up" style="display:inline;">
				` + csrfToken + `
				<button type="submit" class="btn-icon">↑</button>
			</form>`
		} else {
			moveUpBtn = `<button class="btn-icon" disabled style="opacity: 0.3;">↑</button>`
		}

		moveDownBtn := ""
		if showMoveDown {
			moveDownBtn = `<form method="POST" action="/admin/menu/` + strconv.Itoa(int(item.ID)) + `/move-down" style="display:inline;">
				` + csrfToken + `
				<button type="submit" class="btn-icon">↓</button>
			</form>`
		} else {
			moveDownBtn = `<button class="btn-icon" disabled style="opacity: 0.3;">↓</button>`
		}

		menuItemsHTML += fmt.Sprintf(`
			<div class="menu-item">
				<div class="menu-info">
					<div class="menu-label">%s</div>
					<div class="menu-url">%s</div>
				</div>
				<div class="menu-actions">
					%s
					%s
					<form method="POST" action="/admin/menu/%d/delete" style="display:inline;" onsubmit="return confirm('Delete this menu item?')">
						%s
						<button type="submit" class="btn-small btn-danger">Delete</button>
					</form>
				</div>
			</div>
		`, item.Label, item.URL, moveUpBtn, moveDownBtn, item.ID, csrfToken)
	}

	if menuItemsHTML == "" {
		menuItemsHTML = `<div class="empty-state">No menu items yet. Add one below.</div>`
	}

	// Build pages dropdown options
	pagesOptions := ""
	for _, page := range pages {
		pagesOptions += fmt.Sprintf(`<option value="%s">%s</option>`, page.Slug, page.Title)
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Navigation Menu</title>
    <style>
        body { font-family: system-ui, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 900px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { margin: 0 0 20px 0; font-size: 28px; color: #333; }
        .back-link { color: #007bff; text-decoration: none; font-size: 14px; margin-bottom: 20px; display: inline-block; }
        .back-link:hover { text-decoration: underline; }
        .section { margin-bottom: 30px; }
        .section h2 { font-size: 18px; margin-bottom: 15px; color: #444; }
        .menu-item { padding: 15px; border: 1px solid #e0e0e0; border-radius: 4px; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center; }
        .menu-info { flex: 1; }
        .menu-label { font-weight: 600; margin-bottom: 5px; font-size: 14px; color: #333; }
        .menu-url { font-size: 13px; color: #666; font-family: monospace; }
        .menu-actions { display: flex; gap: 8px; align-items: center; }
        .btn-small { padding: 6px 12px; font-size: 13px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; border: none; cursor: pointer; }
        .btn-small:hover { background: #0056b3; }
        .btn-danger { background: #dc3545; }
        .btn-danger:hover { background: #c82333; }
        .btn-icon { padding: 6px 10px; font-size: 16px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer; }
        .btn-icon:hover { background: #5a6268; }
        .empty-state { padding: 40px; text-align: center; color: #999; border: 2px dashed #e0e0e0; border-radius: 4px; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; font-weight: 600; font-size: 14px; color: #333; }
        input[type="text"], select { width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; }
        select { cursor: pointer; }
        .btn { padding: 10px 20px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #218838; }
        .help-text { font-size: 13px; color: #666; margin-top: 5px; }
        .link-type-toggle { margin-bottom: 15px; }
        .link-type-toggle label { display: inline-block; margin-right: 20px; font-weight: normal; cursor: pointer; }
        .link-type-toggle input[type="radio"] { margin-right: 5px; }
        #page-select, #custom-url { display: none; }
    </style>
    <script>
        function toggleLinkType() {
            const linkType = document.querySelector('input[name="link_type"]:checked').value;
            const pageSelect = document.getElementById('page-select');
            const customUrl = document.getElementById('custom-url');

            if (linkType === 'page') {
                pageSelect.style.display = 'block';
                customUrl.style.display = 'none';
            } else {
                pageSelect.style.display = 'none';
                customUrl.style.display = 'block';
            }
        }

        // Set initial state
        document.addEventListener('DOMContentLoaded', function() {
            toggleLinkType();
        });
    </script>
</head>
<body>
    <div class="container">
        <a href="/admin/dashboard" class="back-link">← Back to Dashboard</a>

        <h1>Navigation Menu</h1>

        <div class="section">
            <h2>Menu Items</h2>
            ` + menuItemsHTML + `
        </div>

        <div class="section">
            <h2>Add Menu Item</h2>
            <form method="POST" action="/admin/menu">
                ` + csrfToken + `
                <div class="form-group">
                    <label for="label">Link Text</label>
                    <input type="text" id="label" name="label" placeholder="e.g., About Us" required>
                </div>

                <div class="link-type-toggle">
                    <label>
                        <input type="radio" name="link_type" value="page" checked onchange="toggleLinkType()">
                        Link to a page
                    </label>
                    <label>
                        <input type="radio" name="link_type" value="custom" onchange="toggleLinkType()">
                        Custom URL
                    </label>
                </div>

                <div class="form-group" id="page-select">
                    <label for="page">Select Page</label>
                    <select id="page" name="page_url">
                        <option value="">-- Select a page --</option>
                        ` + pagesOptions + `
                    </select>
                </div>

                <div class="form-group" id="custom-url">
                    <label for="custom_url">Custom URL</label>
                    <input type="text" id="custom_url" name="custom_url" placeholder="https://example.com or /custom-path">
                    <p class="help-text">Can be an external URL (https://...) or internal path (/about)</p>
                </div>

                <button type="submit" class="btn">Add Menu Item</button>
            </form>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// CreateMenuItemHandler creates a new menu item
func CreateMenuItemHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	label := c.PostForm("label")
	linkType := c.PostForm("link_type")

	var url string
	if linkType == "page" {
		url = c.PostForm("page_url")
	} else {
		url = c.PostForm("custom_url")
	}

	if label == "" || url == "" {
		c.String(http.StatusBadRequest, "Label and URL are required")
		return
	}

	// Calculate next order
	var maxOrder struct {
		MaxOrder *int
	}
	db.GetDB().Model(&models.MenuItem{}).
		Where("site_id = ?", site.ID).
		Select("MAX(`order`) as max_order").
		Scan(&maxOrder)

	nextOrder := 0
	if maxOrder.MaxOrder != nil {
		nextOrder = *maxOrder.MaxOrder + 1
	}

	// Create menu item
	menuItem := models.MenuItem{
		SiteID: site.ID,
		Label:  label,
		URL:    url,
		Order:  nextOrder,
	}

	if err := db.GetDB().Create(&menuItem).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create menu item")
		return
	}

	c.Redirect(http.StatusFound, "/admin/menu")
}

// DeleteMenuItemHandler deletes a menu item
func DeleteMenuItemHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get menu item ID from URL
	itemIDStr := c.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid menu item ID")
		return
	}

	// Load menu item
	var menuItem models.MenuItem
	result := db.GetDB().Where("id = ? AND site_id = ?", itemID, site.ID).First(&menuItem)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Menu item not found")
		return
	}

	// Delete the menu item
	if err := db.GetDB().Delete(&menuItem).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete menu item")
		return
	}

	c.Redirect(http.StatusFound, "/admin/menu")
}

// MoveMenuItemUpHandler moves a menu item up in the order
func MoveMenuItemUpHandler(c *gin.Context) {
	moveMenuItem(c, -1)
}

// MoveMenuItemDownHandler moves a menu item down in the order
func MoveMenuItemDownHandler(c *gin.Context) {
	moveMenuItem(c, 1)
}

func moveMenuItem(c *gin.Context, direction int) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get menu item ID from URL
	itemIDStr := c.Param("id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid menu item ID")
		return
	}

	// Load current menu item
	var currentItem models.MenuItem
	result := db.GetDB().Where("id = ? AND site_id = ?", itemID, site.ID).First(&currentItem)
	if result.Error != nil {
		c.String(http.StatusNotFound, "Menu item not found")
		return
	}

	// Find adjacent item to swap with
	var adjacentItem models.MenuItem
	var query *gorm.DB

	if direction < 0 {
		// Moving up: find item with next lower order
		query = db.GetDB().Where("site_id = ? AND `order` < ?", site.ID, currentItem.Order).
			Order("`order` DESC").Limit(1)
	} else {
		// Moving down: find item with next higher order
		query = db.GetDB().Where("site_id = ? AND `order` > ?", site.ID, currentItem.Order).
			Order("`order` ASC").Limit(1)
	}

	if err := query.First(&adjacentItem).Error; err != nil {
		// No adjacent item found, can't move
		c.Redirect(http.StatusFound, "/admin/menu")
		return
	}

	// Swap orders
	currentOrder := currentItem.Order
	currentItem.Order = adjacentItem.Order
	adjacentItem.Order = currentOrder

	// Save both items
	db.GetDB().Save(&currentItem)
	db.GetDB().Save(&adjacentItem)

	c.Redirect(http.StatusFound, "/admin/menu")
}
