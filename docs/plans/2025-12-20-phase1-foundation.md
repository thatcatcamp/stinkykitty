# Phase 1: StinkyKitty Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the foundational infrastructure for StinkyKitty including configuration management, CLI commands, and database models with multi-tenancy support.

**Architecture:** Viper-based configuration system with CLI-editable YAML config, Cobra command structure for `config`, `site`, `user`, and `server` commands, and GORM models for users, sites, pages, and content blocks with proper relationships.

**Tech Stack:** Go 1.25+, Cobra (CLI), Viper (config), GORM (ORM), SQLite (default DB)

---

## Task 1: Configuration Management System

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `cmd/stinky/config.go`

### Step 1: Write the failing test for config initialization

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfig(t *testing.T) {
	// Create temp directory for test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := InitConfig(configPath)
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestGetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	InitConfig(configPath)

	// Test getting a default value
	value := GetString("server.http_port")
	if value != "80" {
		t.Errorf("Expected default http_port to be 80, got %s", value)
	}
}

func TestSetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	InitConfig(configPath)

	err := Set("server.http_port", "8080")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value := GetString("server.http_port")
	if value != "8080" {
		t.Errorf("Expected http_port to be 8080, got %s", value)
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/config -v`

Expected: FAIL with "no such file or directory" or "undefined: InitConfig"

### Step 3: Implement config package

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var v *viper.Viper

// InitConfig initializes the configuration system
func InitConfig(configPath string) error {
	v = viper.New()

	// Set defaults
	setDefaults()

	// Set config file path
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to read existing config
	if err := v.ReadInConfig(); err != nil {
		// If config doesn't exist, create it with defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := v.SafeWriteConfig(); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	v.SetDefault("server.http_port", "80")
	v.SetDefault("server.https_port", "443")
	v.SetDefault("server.behind_proxy", false)

	// Storage defaults
	v.SetDefault("storage.data_dir", "/var/lib/stinkykitty")
	v.SetDefault("storage.sites_dir", "/var/lib/stinkykitty/sites")
	v.SetDefault("storage.backups_dir", "/var/lib/stinkykitty/backups")

	// Backup defaults
	v.SetDefault("backups.schedule", "0 3 * * *") // 3am daily
	v.SetDefault("backups.retention.daily", 7)
	v.SetDefault("backups.retention.weekly", 4)
	v.SetDefault("backups.retention.monthly", 12)

	// Database defaults
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.path", "/var/lib/stinkykitty/stinkykitty.db")
}

// GetString returns a config value as string
func GetString(key string) string {
	if v == nil {
		return ""
	}
	return v.GetString(key)
}

// GetInt returns a config value as int
func GetInt(key string) int {
	if v == nil {
		return 0
	}
	return v.GetInt(key)
}

// GetBool returns a config value as bool
func GetBool(key string) bool {
	if v == nil {
		return false
	}
	return v.GetBool(key)
}

// Set sets a config value and saves to file
func Set(key string, value interface{}) error {
	if v == nil {
		return fmt.Errorf("config not initialized")
	}

	v.Set(key, value)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// GetAll returns all config values as a map
func GetAll() map[string]interface{} {
	if v == nil {
		return nil
	}
	return v.AllSettings()
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/config -v`

Expected: PASS (all 3 tests)

### Step 5: Add config CLI commands

Create `cmd/stinky/config.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage StinkyKitty configuration",
	Long:  "View and modify StinkyKitty configuration values",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		value := config.GetString(args[0])
		fmt.Println(value)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := config.Set(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Set %s = %s\n", args[0], args[1])
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		all := config.GetAll()
		for key, value := range all {
			fmt.Printf("%s: %v\n", key, value)
		}
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}

// initConfig initializes the configuration system
func initConfig() error {
	configPath := os.Getenv("STINKY_CONFIG")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = home + "/.stinkykitty/config.yaml"
	}

	return config.InitConfig(configPath)
}
```

### Step 6: Update main.go to use init function

Modify `cmd/stinky/main.go` - no changes needed, init() functions run automatically.

### Step 7: Test CLI commands manually

Run: `go run cmd/stinky/main.go config list`

Expected: Output showing default config values

Run: `go run cmd/stinky/main.go config set server.http_port 8080`

Expected: "Set server.http_port = 8080"

Run: `go run cmd/stinky/main.go config get server.http_port`

Expected: "8080"

### Step 8: Commit

```bash
git add internal/config/ cmd/stinky/config.go
git commit -m "feat: add Viper-based configuration management

- Configuration system with YAML storage
- CLI-editable config (no manual YAML editing)
- Commands: config get, config set, config list
- Sensible defaults for server, storage, backups
- Tests for init, get, and set operations"
```

---

## Task 2: Database Models

**Files:**
- Create: `internal/models/models.go`
- Create: `internal/models/models_test.go`
- Create: `internal/db/db.go`

### Step 1: Write the failing test for database connection

Create `internal/models/models_test.go`:

```go
package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(&User{}, &Site{}, &Page{}, &Block{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)

	user := User{
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
	}

	result := db.Create(&user)
	if result.Error != nil {
		t.Fatalf("Failed to create user: %v", result.Error)
	}

	if user.ID == 0 {
		t.Error("User ID should be set after creation")
	}
}

func TestCreateSite(t *testing.T) {
	db := setupTestDB(t)

	// Create owner first
	user := User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:    "testcamp",
		OwnerID:      user.ID,
		SiteDir:      "/var/lib/stinkykitty/sites/site-123",
		DatabaseType: "sqlite",
		DatabasePath: "/var/lib/stinkykitty/sites/site-123/site.db",
	}

	result := db.Create(&site)
	if result.Error != nil {
		t.Fatalf("Failed to create site: %v", result.Error)
	}

	if site.ID == 0 {
		t.Error("Site ID should be set after creation")
	}
}

func TestSiteUserRelationship(t *testing.T) {
	db := setupTestDB(t)

	user := User{Email: "admin@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := Site{
		Subdomain:    "camp",
		OwnerID:      user.ID,
		SiteDir:      "/var/lib/stinkykitty/sites/site-456",
		DatabaseType: "sqlite",
	}
	db.Create(&site)

	siteUser := SiteUser{
		UserID: user.ID,
		SiteID: site.ID,
		Role:   "admin",
	}
	db.Create(&siteUser)

	// Query to verify relationship
	var retrievedSiteUser SiteUser
	result := db.Where("user_id = ? AND site_id = ?", user.ID, site.ID).First(&retrievedSiteUser)
	if result.Error != nil {
		t.Fatalf("Failed to retrieve site user: %v", result.Error)
	}

	if retrievedSiteUser.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", retrievedSiteUser.Role)
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/models -v`

Expected: FAIL with "undefined: User" or similar

### Step 3: Implement database models

Create `internal/models/models.go`:

```go
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
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/models -v`

Expected: PASS (all 3 tests)

### Step 5: Create database connection package

Create `internal/db/db.go`:

```go
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
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	return DB
}
```

### Step 6: Commit

```bash
git add internal/models/ internal/db/
git commit -m "feat: add GORM database models and connection

- User model with email and password hash
- Site model with multi-tenancy fields (subdomain, custom domain)
- SiteUser junction table for user-site-role relationships
- Page and Block models for content management
- Database connection package supporting SQLite and MariaDB
- Comprehensive tests for models and relationships"
```

---

## Task 3: User Management Commands

**Files:**
- Create: `cmd/stinky/user.go`
- Create: `internal/users/users.go`
- Create: `internal/users/users_test.go`

### Step 1: Write failing test for user creation

Create `internal/users/users_test.go`:

```go
package users

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	db.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{})
	return db
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Email)
	}

	if user.PasswordHash == "password123" {
		t.Error("Password should be hashed, not stored in plain text")
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := setupTestDB(t)

	CreateUser(db, "find@example.com", "password")

	user, err := GetUserByEmail(db, "find@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if user.Email != "find@example.com" {
		t.Errorf("Expected email find@example.com, got %s", user.Email)
	}
}

func TestListUsers(t *testing.T) {
	db := setupTestDB(t)

	CreateUser(db, "user1@example.com", "pass1")
	CreateUser(db, "user2@example.com", "pass2")

	users, err := ListUsers(db)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)

	user, _ := CreateUser(db, "delete@example.com", "password")

	err := DeleteUser(db, user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify user is deleted
	_, err = GetUserByEmail(db, "delete@example.com")
	if err == nil {
		t.Error("Expected error when getting deleted user, got nil")
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/users -v`

Expected: FAIL with "undefined: CreateUser"

### Step 3: Implement user management functions

Create `internal/users/users.go`:

```go
package users

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// CreateUser creates a new user with hashed password
func CreateUser(db *gorm.DB, email, password string) (*models.User, error) {
	// Check if user already exists
	var existing models.User
	result := db.Where("email = ?", email).First(&existing)
	if result.Error == nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email address
func GetUserByEmail(db *gorm.DB, email string) (*models.User, error) {
	var user models.User
	result := db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(db *gorm.DB, id uint) (*models.User, error) {
	var user models.User
	result := db.First(&user, id)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

// ListUsers returns all users
func ListUsers(db *gorm.DB) ([]models.User, error) {
	var users []models.User
	result := db.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list users: %w", result.Error)
	}
	return users, nil
}

// DeleteUser soft-deletes a user
func DeleteUser(db *gorm.DB, id uint) error {
	result := db.Delete(&models.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// ValidatePassword checks if a password matches the user's hash
func ValidatePassword(user *models.User, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/users -v`

Expected: PASS (all 4 tests)

### Step 5: Add user CLI commands

Create `cmd/stinky/user.go`:

```go
package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/users"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long:  "Create, list, and manage user accounts",
}

var userCreateCmd = &cobra.Command{
	Use:   "create <email>",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		email := args[0]

		// Get password from stdin
		fmt.Print("Enter password: ")
		var password string
		fmt.Scanln(&password)

		user, err := users.CreateUser(db.GetDB(), email, password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating user: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User created: %s (ID: %d)\n", user.Email, user.ID)
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		userList, err := users.ListUsers(db.GetDB())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing users: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tEMAIL\tCREATED")
		for _, u := range userList {
			fmt.Fprintf(w, "%d\t%s\t%s\n", u.ID, u.Email, u.CreatedAt.Format("2006-01-02"))
		}
		w.Flush()
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <email>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		email := args[0]
		user, err := users.GetUserByEmail(db.GetDB(), email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := users.DeleteUser(db.GetDB(), user.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting user: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("User deleted: %s\n", email)
	},
}

func init() {
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userDeleteCmd)
	rootCmd.AddCommand(userCmd)
}

// initSystemDB initializes the system database connection
func initSystemDB() error {
	if err := initConfig(); err != nil {
		return err
	}

	dbType := config.GetString("database.type")
	dbPath := config.GetString("database.path")

	return db.InitDB(dbType, dbPath)
}
```

### Step 6: Test user commands manually

Run: `go run cmd/stinky/main.go user create test@example.com`

Enter password when prompted.

Expected: "User created: test@example.com (ID: 1)"

Run: `go run cmd/stinky/main.go user list`

Expected: Table showing the created user

### Step 7: Commit

```bash
git add internal/users/ cmd/stinky/user.go
git commit -m "feat: add user management commands

- CreateUser with bcrypt password hashing
- GetUserByEmail and GetUserByID functions
- ListUsers and DeleteUser functions
- CLI commands: user create, user list, user delete
- Comprehensive tests for all user operations"
```

---

## Task 4: Site Management Commands

**Files:**
- Create: `cmd/stinky/site.go`
- Create: `internal/sites/sites.go`
- Create: `internal/sites/sites_test.go`

### Step 1: Write failing test for site creation

Create `internal/sites/sites_test.go`:

```go
package sites

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	db.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{})
	return db
}

func TestCreateSite(t *testing.T) {
	db := setupTestDB(t)

	// Create owner
	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	site, err := CreateSite(db, "testcamp", owner.ID, "/tmp/sites")
	if err != nil {
		t.Fatalf("CreateSite failed: %v", err)
	}

	if site.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain testcamp, got %s", site.Subdomain)
	}

	if site.OwnerID != owner.ID {
		t.Errorf("Expected owner ID %d, got %d", owner.ID, site.OwnerID)
	}
}

func TestGetSiteBySubdomain(t *testing.T) {
	db := setupTestDB(t)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	CreateSite(db, "findme", owner.ID, "/tmp/sites")

	site, err := GetSiteBySubdomain(db, "findme")
	if err != nil {
		t.Fatalf("GetSiteBySubdomain failed: %v", err)
	}

	if site.Subdomain != "findme" {
		t.Errorf("Expected subdomain findme, got %s", site.Subdomain)
	}
}

func TestAddUserToSite(t *testing.T) {
	db := setupTestDB(t)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	admin := models.User{Email: "admin@example.com", PasswordHash: "hash"}
	db.Create(&admin)

	site, _ := CreateSite(db, "camp", owner.ID, "/tmp/sites")

	err := AddUserToSite(db, site.ID, admin.ID, "admin")
	if err != nil {
		t.Fatalf("AddUserToSite failed: %v", err)
	}

	// Verify relationship
	var siteUser models.SiteUser
	result := db.Where("site_id = ? AND user_id = ?", site.ID, admin.ID).First(&siteUser)
	if result.Error != nil {
		t.Fatalf("Failed to find site user: %v", result.Error)
	}

	if siteUser.Role != "admin" {
		t.Errorf("Expected role admin, got %s", siteUser.Role)
	}
}

func TestListSites(t *testing.T) {
	db := setupTestDB(t)

	owner := models.User{Email: "owner@example.com", PasswordHash: "hash"}
	db.Create(&owner)

	CreateSite(db, "site1", owner.ID, "/tmp/sites")
	CreateSite(db, "site2", owner.ID, "/tmp/sites")

	sites, err := ListSites(db)
	if err != nil {
		t.Fatalf("ListSites failed: %v", err)
	}

	if len(sites) != 2 {
		t.Errorf("Expected 2 sites, got %d", len(sites))
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/sites -v`

Expected: FAIL with "undefined: CreateSite"

### Step 3: Implement site management functions

Create `internal/sites/sites.go`:

```go
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
	// Check if domain is already in use
	var existing models.Site
	result := db.Where("custom_domain = ?", domain).First(&existing)
	if result.Error == nil {
		return fmt.Errorf("domain %s is already in use", domain)
	}

	site, err := GetSiteByID(db, siteID)
	if err != nil {
		return err
	}

	site.CustomDomain = domain
	if err := db.Save(site).Error; err != nil {
		return fmt.Errorf("failed to add custom domain: %w", err)
	}

	return nil
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/sites -v`

Expected: PASS (all 4 tests)

### Step 5: Add site CLI commands

Create `cmd/stinky/site.go`:

```go
package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
	"github.com/thatcatcamp/stinkykitty/internal/users"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Manage sites",
	Long:  "Create, list, and manage camp sites",
}

var siteCreateCmd = &cobra.Command{
	Use:   "create <subdomain>",
	Short: "Create a new site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		ownerEmail, _ := cmd.Flags().GetString("owner")

		if ownerEmail == "" {
			fmt.Fprintf(os.Stderr, "Error: --owner flag is required\n")
			os.Exit(1)
		}

		// Get owner user
		owner, err := users.GetUserByEmail(db.GetDB(), ownerEmail)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: owner user not found: %v\n", err)
			os.Exit(1)
		}

		sitesDir := config.GetString("storage.sites_dir")
		site, err := sites.CreateSite(db.GetDB(), subdomain, owner.ID, sitesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Site created: %s (ID: %d)\n", site.Subdomain, site.ID)
		fmt.Printf("Site directory: %s\n", site.SiteDir)
	},
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sites",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		siteList, err := sites.ListSites(db.GetDB())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sites: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSUBDOMAIN\tCUSTOM DOMAIN\tOWNER\tCREATED")
		for _, s := range siteList {
			customDomain := s.CustomDomain
			if customDomain == "" {
				customDomain = "-"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
				s.ID, s.Subdomain, customDomain, s.Owner.Email, s.CreatedAt.Format("2006-01-02"))
		}
		w.Flush()
	},
}

var siteDeleteCmd = &cobra.Command{
	Use:   "delete <subdomain>",
	Short: "Delete a site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := sites.DeleteSite(db.GetDB(), site.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Site deleted: %s\n", subdomain)
	},
}

var siteAddUserCmd = &cobra.Command{
	Use:   "add-user <subdomain> <email>",
	Short: "Add a user to a site",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		email := args[1]
		role, _ := cmd.Flags().GetString("role")

		if role == "" {
			role = "editor" // default role
		}

		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: site not found: %v\n", err)
			os.Exit(1)
		}

		user, err := users.GetUserByEmail(db.GetDB(), email)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: user not found: %v\n", err)
			os.Exit(1)
		}

		if err := sites.AddUserToSite(db.GetDB(), site.ID, user.ID, role); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding user to site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added %s to %s with role: %s\n", email, subdomain, role)
	},
}

var siteListUsersCmd = &cobra.Command{
	Use:   "list-users <subdomain>",
	Short: "List users with access to a site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		siteUsers, err := sites.ListSiteUsers(db.GetDB(), site.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "EMAIL\tROLE")
		for _, su := range siteUsers {
			fmt.Fprintf(w, "%s\t%s\n", su.User.Email, su.Role)
		}
		w.Flush()
	},
}

var siteAddDomainCmd = &cobra.Command{
	Use:   "add-domain <subdomain> <domain>",
	Short: "Add a custom domain to a site",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		domain := args[1]

		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := sites.AddCustomDomain(db.GetDB(), site.ID, domain); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added custom domain %s to site %s\n", domain, subdomain)
	},
}

func init() {
	siteCreateCmd.Flags().String("owner", "", "Email of the site owner (required)")
	siteAddUserCmd.Flags().String("role", "editor", "User role (owner, admin, editor)")

	siteCmd.AddCommand(siteCreateCmd)
	siteCmd.AddCommand(siteListCmd)
	siteCmd.AddCommand(siteDeleteCmd)
	siteCmd.AddCommand(siteAddUserCmd)
	siteCmd.AddCommand(siteListUsersCmd)
	siteCmd.AddCommand(siteAddDomainCmd)
	rootCmd.AddCommand(siteCmd)
}
```

### Step 6: Test site commands manually

Run: `go run cmd/stinky/main.go site create testcamp --owner test@example.com`

Expected: "Site created: testcamp (ID: 1)"

Run: `go run cmd/stinky/main.go site list`

Expected: Table showing the created site

Run: `go run cmd/stinky/main.go site add-user testcamp admin@example.com --role admin`

Expected: "Added admin@example.com to testcamp with role: admin"

### Step 7: Commit

```bash
git add internal/sites/ cmd/stinky/site.go
git commit -m "feat: add site management commands

- CreateSite with automatic owner assignment
- Site lookup by subdomain, ID, and custom domain
- AddUserToSite with role validation (owner/admin/editor)
- RemoveUserFromSite and ListSiteUsers
- AddCustomDomain with uniqueness checks
- CLI commands: site create, list, delete, add-user, list-users, add-domain
- Comprehensive tests for all site operations"
```

---

## Task 5: Basic Server Command

**Files:**
- Create: `cmd/stinky/server.go`
- Modify: `cmd/stinky/main.go`

### Step 1: Create basic server command

Create `cmd/stinky/server.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server operations",
	Long:  "Start, stop, and manage the StinkyKitty HTTP server",
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create Gin router
		r := gin.Default()

		// Basic health check endpoint
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
				"service": "stinkykitty",
			})
		})

		// Placeholder for site routing
		r.GET("/", func(c *gin.Context) {
			c.String(200, "StinkyKitty CMS - Server running")
		})

		httpPort := config.GetString("server.http_port")
		addr := fmt.Sprintf(":%s", httpPort)

		fmt.Printf("Starting StinkyKitty server on %s\n", addr)
		if err := r.Run(addr); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	rootCmd.AddCommand(serverCmd)
}
```

### Step 2: Test server command

Run: `go run cmd/stinky/main.go server start`

Expected: Server starts on configured port, accessible via curl http://localhost:80/health

Press Ctrl+C to stop.

### Step 3: Commit

```bash
git add cmd/stinky/server.go
git commit -m "feat: add basic server command

- server start command with Gin router
- Health check endpoint at /health
- Reads HTTP port from config
- Foundation for multi-tenant routing (to be added)"
```

---

## Summary

This plan builds the foundational infrastructure for StinkyKitty:

1. **Configuration Management** - Viper-based YAML config with CLI editing
2. **Database Models** - GORM models for users, sites, pages, blocks with relationships
3. **User Management** - Create, list, delete users with bcrypt password hashing
4. **Site Management** - Create sites, manage users/roles, add custom domains
5. **Basic Server** - Gin HTTP server with health check endpoint

**Next phases** will build:
- Multi-tenant routing middleware (identify site by Host header)
- Content block system (hero, text, gallery, video, button)
- Page editor API endpoints
- Authentication (JWT tokens)
- Admin web UI
- SSL automation
- Backup/restore system

**Testing approach:**
- Unit tests for all business logic
- Manual CLI testing for user experience
- Frequent commits after each working piece

**Development principles:**
- DRY: Reuse database connection, config initialization
- YAGNI: Only build what's needed now
- TDD: Tests before implementation
- Small commits: One feature at a time
