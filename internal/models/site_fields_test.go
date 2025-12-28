package models

import (
	"testing"
)

func TestSiteFieldsDefaults(t *testing.T) {
	db := setupTestDB(t)

	// Create a site
	site := &Site{
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}

	if err := db.Create(site).Error; err != nil {
		t.Fatalf("failed to create site: %v", err)
	}

	// Check defaults
	var retrieved Site
	if err := db.First(&retrieved, site.ID).Error; err != nil {
		t.Fatalf("failed to retrieve site: %v", err)
	}

	if retrieved.GoogleAnalyticsID != "" {
		t.Errorf("expected empty GoogleAnalyticsID, got %s", retrieved.GoogleAnalyticsID)
	}

	if retrieved.CopyrightText != "" {
		t.Errorf("expected empty CopyrightText, got %s", retrieved.CopyrightText)
	}
}
