package uploads

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// SaveUploadedFile saves an uploaded file and returns its web-accessible path
func SaveUploadedFile(file *multipart.FileHeader, siteDir string) (string, error) {
	// Create uploads directory if it doesn't exist
	uploadsDir := filepath.Join(siteDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}

	// Generate random filename to avoid conflicts
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random filename: %w", err)
	}
	randomName := hex.EncodeToString(randomBytes)

	// Get file extension
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg" // default
	}

	// Create full path
	filename := randomName + ext
	fullPath := filepath.Join(uploadsDir, filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file contents
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Return web path (relative to site root)
	return "/uploads/" + filename, nil
}

// IsImageFile checks if the uploaded file is an image based on extension
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}
