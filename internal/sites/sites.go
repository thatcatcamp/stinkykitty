package sites

import (
	"fmt"
	"path/filepath"

	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// CreateSite creates a new site with the given subdomain and owner
func CreateSite(db *gorm.DB, subdomain string, ownerID uint, sitesDir string) (*models.Site, error) {
	// Check if subdomain already exists
	var existing models.Site
	result := db.Where("subdomain = ?", subdomain).First(&existing)
	if result.Error == nil {
		return nil, fmt.Errorf("subdomain %s already exists", subdomain)
	}

	// Generate unique site directory name
	siteDir := filepath.Join(sitesDir, fmt.Sprintf("site-%s", subdomain))
	dbPath := filepath.Join(siteDir, "site.db")

	site := &models.Site{
		Subdomain:    subdomain,
		OwnerID:      ownerID,
		SiteDir:      siteDir,
		DatabaseType: "sqlite",
		DatabasePath: dbPath,
		StorageType:  "local",
	}

	if err := db.Create(site).Error; err != nil {
		return nil, fmt.Errorf("failed to create site: %w", err)
	}

	// Add owner to site_users with owner role
	siteUser := &models.SiteUser{
		UserID: ownerID,
		SiteID: site.ID,
		Role:   "owner",
	}

	if err := db.Create(siteUser).Error; err != nil {
		return nil, fmt.Errorf("failed to add owner to site: %w", err)
	}

	// Auto-create homepage for new site
	homepage := &models.Page{
		SiteID:    site.ID,
		Slug:      "/",
		Title:     subdomain,
		Published: false,
	}
	if err := db.Create(homepage).Error; err != nil {
		return nil, fmt.Errorf("failed to create homepage: %w", err)
	}

	return site, nil
}

// GetSiteBySubdomain retrieves a site by subdomain
func GetSiteBySubdomain(db *gorm.DB, subdomain string) (*models.Site, error) {
	var site models.Site
	result := db.Where("subdomain = ?", subdomain).First(&site)
	if result.Error != nil {
		return nil, fmt.Errorf("site not found: %w", result.Error)
	}
	return &site, nil
}

// GetSiteByID retrieves a site by ID
func GetSiteByID(db *gorm.DB, id uint) (*models.Site, error) {
	var site models.Site
	result := db.First(&site, id)
	if result.Error != nil {
		return nil, fmt.Errorf("site not found: %w", result.Error)
	}
	return &site, nil
}

// GetSiteByDomain retrieves a site by custom domain
func GetSiteByDomain(db *gorm.DB, domain string) (*models.Site, error) {
	var site models.Site
	result := db.Where("custom_domain = ?", domain).First(&site)
	if result.Error != nil {
		return nil, fmt.Errorf("site not found: %w", result.Error)
	}
	return &site, nil
}

// ListSites returns all sites
func ListSites(db *gorm.DB) ([]models.Site, error) {
	var sites []models.Site
	result := db.Preload("Owner").Find(&sites)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list sites: %w", result.Error)
	}
	return sites, nil
}

// AddUserToSite adds a user to a site with a specific role
func AddUserToSite(db *gorm.DB, siteID, userID uint, role string) error {
	// Validate role
	validRoles := map[string]bool{"owner": true, "admin": true, "editor": true}
	if !validRoles[role] {
		return fmt.Errorf("invalid role: %s (must be owner, admin, or editor)", role)
	}

	// Check if relationship already exists
	var existing models.SiteUser
	result := db.Where("site_id = ? AND user_id = ?", siteID, userID).First(&existing)
	if result.Error == nil {
		return fmt.Errorf("user already has access to this site")
	}

	siteUser := &models.SiteUser{
		UserID: userID,
		SiteID: siteID,
		Role:   role,
	}

	if err := db.Create(siteUser).Error; err != nil {
		return fmt.Errorf("failed to add user to site: %w", err)
	}

	return nil
}

// RemoveUserFromSite removes a user's access to a site
func RemoveUserFromSite(db *gorm.DB, siteID, userID uint) error {
	result := db.Where("site_id = ? AND user_id = ?", siteID, userID).Delete(&models.SiteUser{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove user from site: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user does not have access to this site")
	}
	return nil
}

// ListSiteUsers returns all users with access to a site
func ListSiteUsers(db *gorm.DB, siteID uint) ([]models.SiteUser, error) {
	var siteUsers []models.SiteUser
	result := db.Where("site_id = ?", siteID).Preload("User").Find(&siteUsers)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list site users: %w", result.Error)
	}
	return siteUsers, nil
}

// DeleteSite soft-deletes a site
func DeleteSite(db *gorm.DB, id uint) error {
	result := db.Delete(&models.Site{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete site: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("site not found")
	}
	return nil
}

// AddCustomDomain adds a custom domain to a site
func AddCustomDomain(db *gorm.DB, siteID uint, domain string) error {
	// Check if domain is already in use (including soft-deleted sites)
	var existing models.Site
	result := db.Unscoped().Where("custom_domain = ?", domain).First(&existing)
	if result.Error == nil {
		if existing.DeletedAt.Valid {
			// Domain is on a deleted site - clear it so it can be reused
			db.Unscoped().Model(&existing).Update("custom_domain", nil)
		} else {
			return fmt.Errorf("domain %s is already in use by site '%s'", domain, existing.Subdomain)
		}
	}

	site, err := GetSiteByID(db, siteID)
	if err != nil {
		return err
	}

	// Set the custom domain using a pointer
	site.CustomDomain = &domain
	if err := db.Save(site).Error; err != nil {
		return fmt.Errorf("failed to add custom domain: %w", err)
	}

	return nil
}
