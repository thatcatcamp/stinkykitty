package db

import (
	"fmt"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB(dbType, dbPath string) error {
	var err error
	var dialector gorm.Dialector

	switch dbType {
	case "sqlite":
		dialector = sqlite.Open(dbPath)
	case "mysql", "mariadb":
		dialector = mysql.Open(dbPath) // dbPath is DSN for MySQL
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate all models
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Site{},
		&models.SiteUser{},
		&models.Page{},
		&models.Block{},
		&models.MenuItem{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// InitFTSIndex initializes the FTS5 search index
// Note: Must be called separately after InitDB, and only for SQLite databases
func InitFTSIndex() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Create FTS5 virtual table
	_, err = sqlDB.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
			page_id UNINDEXED,
			site_id UNINDEXED,
			title,
			content,
			tokenize='porter unicode61'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create FTS index: %w", err)
	}

	return nil
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	return DB
}

// SetDB sets the database connection (used for testing)
func SetDB(database *gorm.DB) {
	DB = database
}
