package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a global user account
type User struct {
	ID           uint           `gorm:"primaryKey"`
	Email        string         `gorm:"uniqueIndex;not null"`
	PasswordHash string         `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relationships
	OwnedSites []Site     `gorm:"foreignKey:OwnerID"`
	SiteUsers  []SiteUser `gorm:"foreignKey:UserID"`
}

// Site represents a camp website
type Site struct {
	ID            uint           `gorm:"primaryKey"`
	Subdomain     string         `gorm:"uniqueIndex"`
	CustomDomain  string         `gorm:"uniqueIndex"`
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
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relationships
	Owner     User       `gorm:"foreignKey:OwnerID"`
	SiteUsers []SiteUser `gorm:"foreignKey:SiteID"`
	Pages     []Page     `gorm:"foreignKey:SiteID"`
}

// SiteUser represents the many-to-many relationship between users and sites
type SiteUser struct {
	ID        uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"not null"`
	SiteID    uint           `gorm:"not null"`
	Role      string         `gorm:"not null"` // "owner", "admin", "editor"
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relationships
	User User `gorm:"foreignKey:UserID"`
	Site Site `gorm:"foreignKey:SiteID"`
}

// Page represents a page on a site
type Page struct {
	ID        uint           `gorm:"primaryKey"`
	SiteID    uint           `gorm:"not null"`
	Slug      string         `gorm:"not null"` // URL slug
	Title     string         `gorm:"not null"`
	Published bool           `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relationships
	Site   Site    `gorm:"foreignKey:SiteID"`
	Blocks []Block `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
}

// Block represents a content block on a page
type Block struct {
	ID        uint           `gorm:"primaryKey"`
	PageID    uint           `gorm:"not null"`
	Type      string         `gorm:"not null"` // "hero", "text", "gallery", "video", "button"
	Order     int            `gorm:"not null"` // Display order on page
	Config    string         `gorm:"type:text"` // JSON configuration for the block
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relationships
	Page Page `gorm:"foreignKey:PageID"`
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
