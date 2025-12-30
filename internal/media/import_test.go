package media

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestImportExistingUploads(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.Site{}, &models.User{}, &models.MediaItem{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test site
	tmpDir := t.TempDir()
	uploadsDir := filepath.Join(tmpDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)

	user := models.User{Email: "test@example.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{
		Subdomain: "test",
		OwnerID:   user.ID,
		SiteDir:   tmpDir,
	}
	db.Create(&site)

	// Create fake image files
	os.WriteFile(filepath.Join(uploadsDir, "abc123.jpg"), []byte("fake image"), 0644)
	os.WriteFile(filepath.Join(uploadsDir, "def456.png"), []byte("fake image 2"), 0644)
	os.WriteFile(filepath.Join(uploadsDir, "notanimage.txt"), []byte("text"), 0644)

	// Import
	count, err := ImportExistingUploads(db, site)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 imports, got %d", count)
	}

	// Verify database records
	var items []models.MediaItem
	db.Where("site_id = ?", site.ID).Find(&items)

	if len(items) != 2 {
		t.Errorf("Expected 2 media items in database, got %d", len(items))
	}
}
