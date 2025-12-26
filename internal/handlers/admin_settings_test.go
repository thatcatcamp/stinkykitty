package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func TestAdminSettingsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	// Create site with existing palette
	site := models.Site{
		ID:           1,
		Subdomain:    "test",
		ThemePalette: "indigo",
		DarkMode:     true,
	}
	database.Create(&site)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/settings", nil)
	c.Set("site", &site)

	AdminSettingsHandler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Should contain palette dropdown
	if !strings.Contains(body, `<select id="palette"`) {
		t.Error("Response should contain palette dropdown")
	}

	// Should have current palette selected
	if !strings.Contains(body, `<option value="indigo" selected>`) {
		t.Error("Current palette (indigo) should be selected")
	}

	// Should have dark mode checked
	if !strings.Contains(body, `<input type="checkbox" id="dark_mode" name="dark_mode" value="true" checked>`) {
		t.Error("Dark mode checkbox should be checked")
	}

	// Should contain all 12 palettes
	palettes := []string{"slate", "indigo", "rose", "emerald", "navy", "purple", "teal", "amber", "rose-mono", "green-mono", "blue-mono", "neutral"}
	for _, palette := range palettes {
		if !strings.Contains(body, `value="`+palette+`"`) {
			t.Errorf("Response should contain palette option: %s", palette)
		}
	}
}

func TestAdminSettingsHandlerDefaultPalette(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	// Create site with default palette
	site := models.Site{
		ID:           1,
		Subdomain:    "test",
		ThemePalette: "slate",
		DarkMode:     false,
	}
	database.Create(&site)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/settings", nil)
	c.Set("site", &site)

	AdminSettingsHandler(c)

	body := w.Body.String()

	// Should have slate selected
	if !strings.Contains(body, `<option value="slate" selected>`) {
		t.Error("Default palette (slate) should be selected")
	}

	// Dark mode should not be checked
	if strings.Contains(body, `<input type="checkbox" id="dark_mode" name="dark_mode" value="true" checked>`) {
		t.Error("Dark mode checkbox should not be checked")
	}
}

func TestAdminSettingsSaveHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	// Create site
	site := models.Site{
		ID:           1,
		Subdomain:    "test",
		ThemePalette: "slate",
		DarkMode:     false,
	}
	database.Create(&site)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("palette", "indigo")
	form.Add("dark_mode", "true")

	c.Request = httptest.NewRequest("POST", "/admin/settings", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Set("site", &site)

	AdminSettingsSaveHandler(c)

	// Should redirect to settings page
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d", c.Writer.Status())
	}

	if w.Result().Header.Get("Location") != "/admin/settings" {
		t.Errorf("Expected redirect to /admin/settings, got %s", w.Result().Header.Get("Location"))
	}

	// Check database was updated
	var updatedSite models.Site
	database.First(&updatedSite, site.ID)

	if updatedSite.ThemePalette != "indigo" {
		t.Errorf("Expected palette to be indigo, got %s", updatedSite.ThemePalette)
	}

	if !updatedSite.DarkMode {
		t.Error("Expected dark mode to be enabled")
	}
}

func TestAdminSettingsSaveHandlerDisableDarkMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	// Create site with dark mode enabled
	site := models.Site{
		ID:           1,
		Subdomain:    "test",
		ThemePalette: "indigo",
		DarkMode:     true,
	}
	database.Create(&site)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("palette", "rose")
	// dark_mode not set = unchecked = false

	c.Request = httptest.NewRequest("POST", "/admin/settings", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Set("site", &site)

	AdminSettingsSaveHandler(c)

	// Check database was updated
	var updatedSite models.Site
	database.First(&updatedSite, site.ID)

	if updatedSite.ThemePalette != "rose" {
		t.Errorf("Expected palette to be rose, got %s", updatedSite.ThemePalette)
	}

	if updatedSite.DarkMode {
		t.Error("Expected dark mode to be disabled")
	}
}

func TestAdminSettingsSaveHandlerInvalidPalette(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	// Create site
	site := models.Site{
		ID:           1,
		Subdomain:    "test",
		ThemePalette: "indigo",
		DarkMode:     false,
	}
	database.Create(&site)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("palette", "invalid-palette-name")
	form.Add("dark_mode", "true")

	c.Request = httptest.NewRequest("POST", "/admin/settings", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Set("site", &site)

	AdminSettingsSaveHandler(c)

	// Should still redirect
	if c.Writer.Status() != http.StatusFound {
		t.Errorf("Expected status 302, got %d", c.Writer.Status())
	}

	// Check database was updated with default palette
	var updatedSite models.Site
	database.First(&updatedSite, site.ID)

	if updatedSite.ThemePalette != "slate" {
		t.Errorf("Expected palette to default to slate, got %s", updatedSite.ThemePalette)
	}

	// Dark mode should still be saved
	if !updatedSite.DarkMode {
		t.Error("Expected dark mode to be enabled")
	}
}

func TestAdminSettingsHandlerNoSite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/settings", nil)
	// No site in context

	AdminSettingsHandler(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestAdminSettingsSaveHandlerNoSite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	database := setupHandlerTestDB(t)
	db.SetDB(database)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	form := url.Values{}
	form.Add("palette", "indigo")

	c.Request = httptest.NewRequest("POST", "/admin/settings", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// No site in context

	AdminSettingsSaveHandler(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}
