package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
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

// GetDB returns the database connection
func GetDB() *gorm.DB {
	return DB
}

// SetDB sets the database connection (used for testing)
func SetDB(database *gorm.DB) {
	DB = database
}
