package models

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// User represents a global user account
type User struct {
	ID            uint           `gorm:"primaryKey"`
	Email         string         `gorm:"uniqueIndex;not null"`
	PasswordHash  string         `gorm:"not null"`
	IsGlobalAdmin bool           `gorm:"default:false"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relationships
	OwnedSites []Site     `gorm:"foreignKey:OwnerID"`
	SiteUsers  []SiteUser `gorm:"foreignKey:UserID"`
}

// Site represents a camp website
type Site struct {
	ID            uint           `gorm:"primaryKey"`
	Subdomain     string         `gorm:"uniqueIndex"`
	CustomDomain  *string        `gorm:"uniqueIndex"`
	OwnerID       uint           `gorm:"not null"`
	SiteDir       string         `gorm:"not null"` // Directory path for this site
	DatabaseType  string         `gorm:"default:sqlite"` // "sqlite" or "mariadb"
	DatabasePath  string         // For SQLite
	DatabaseHost  string         // For MariaDB
	DatabaseName  string         // For MariaDB
	StorageType   string         `gorm:"default:local"` // "local" or "s3"
	S3Bucket      string         // For S3 storage
	PrimaryColor  string         `gorm:"default:#2563eb"`
	SecondaryColor string        `gorm:"default:#64748b"`
	SiteTitle     string
	SiteTagline   string
	LogoPath      string
	FontPair      string         `gorm:"default:system"`
	AllowedIPs    string         `gorm:"type:text"` // JSON array of CIDR ranges
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relationships
	Owner     User       `gorm:"foreignKey:OwnerID"`
	SiteUsers []SiteUser `gorm:"foreignKey:SiteID"`
	Pages     []Page     `gorm:"foreignKey:SiteID"`
	MenuItems []MenuItem `gorm:"foreignKey:SiteID"`
}

// SiteUser represents the many-to-many relationship between users and sites
type SiteUser struct {
	ID        uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"uniqueIndex:idx_user_site;not null"`
	SiteID    uint           `gorm:"uniqueIndex:idx_user_site;not null"`
	Role      string         `gorm:"not null"` // "owner", "admin", "editor"
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
	Site Site `gorm:"foreignKey:SiteID"`
}

// Page represents a content page on a site
type Page struct {
	ID        uint           `gorm:"primaryKey"`
	SiteID    uint           `gorm:"not null;index:idx_site_slug,unique"`
	Slug      string         `gorm:"not null;index:idx_site_slug,unique"` // "/" for homepage, "/about", etc
	Title     string         `gorm:"not null"`
	Published bool           `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Site   Site    `gorm:"foreignKey:SiteID"`
	Blocks []Block `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
}

// Block represents a content block on a page
type Block struct {
	ID        uint           `gorm:"primaryKey"`
	PageID    uint           `gorm:"not null;index"`
	Type      string         `gorm:"not null"` // "text", "hero", "gallery", etc
	Order     int            `gorm:"not null;default:0"`
	Data      string         `gorm:"type:text"` // JSON blob with block-specific content
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Page Page `gorm:"foreignKey:PageID"`
}

// MenuItem represents a navigation menu item
type MenuItem struct {
	ID        uint           `gorm:"primaryKey"`
	SiteID    uint           `gorm:"not null;index"`
	Label     string         `gorm:"not null"`        // Display text
	URL       string         `gorm:"not null"`        // Page slug or external URL
	Order     int            `gorm:"not null;default:0"` // Display order
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Site Site `gorm:"foreignKey:SiteID"`
}

// TableName overrides for consistent naming
func (User) TableName() string {
	return "users"
}

func (Site) TableName() string {
	return "sites"
}

func (SiteUser) TableName() string {
	return "site_users"
}

func (Page) TableName() string {
	return "pages"
}

func (Block) TableName() string {
	return "blocks"
}

func (MenuItem) TableName() string {
	return "menu_items"
}

// GetAllowedIPs returns the list of allowed IP ranges for this site
func (s *Site) GetAllowedIPs() ([]string, error) {
	if s.AllowedIPs == "" {
		return []string{}, nil
	}

	var ips []string
	if err := json.Unmarshal([]byte(s.AllowedIPs), &ips); err != nil {
		return nil, err
	}

	return ips, nil
}

// SetAllowedIPs sets the allowed IP ranges for this site
func (s *Site) SetAllowedIPs(ips []string) error {
	if len(ips) == 0 {
		s.AllowedIPs = ""
		return nil
	}

	data, err := json.Marshal(ips)
	if err != nil {
		return err
	}

	s.AllowedIPs = string(data)
	return nil
}

// AddAllowedIP adds a new IP range to the allowlist
func (s *Site) AddAllowedIP(cidr string) error {
	ips, err := s.GetAllowedIPs()
	if err != nil {
		return err
	}

	// Check if already exists
	for _, ip := range ips {
		if ip == cidr {
			return nil // Already exists, no error
		}
	}

	ips = append(ips, cidr)
	return s.SetAllowedIPs(ips)
}

// RemoveAllowedIP removes an IP range from the allowlist
func (s *Site) RemoveAllowedIP(cidr string) error {
	ips, err := s.GetAllowedIPs()
	if err != nil {
		return err
	}

	if len(ips) == 0 {
		return fmt.Errorf("IP range not found: %s", cidr)
	}

	// Find and remove the IP
	found := false
	newIPs := make([]string, 0, len(ips))
	for _, ip := range ips {
		if ip == cidr {
			found = true
			continue
		}
		newIPs = append(newIPs, ip)
	}

	if !found {
		return fmt.Errorf("IP range not found: %s", cidr)
	}

	return s.SetAllowedIPs(newIPs)
}
