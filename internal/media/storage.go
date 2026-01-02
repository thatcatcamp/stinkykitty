// SPDX-License-Identifier: MIT
package media

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/thatcatcamp/stinkykitty/internal/config"
)

// SaveToCentralizedStorage saves an uploaded file to the centralized media directory
// and returns the generated filename.
func SaveToCentralizedStorage(file *multipart.FileHeader) (string, error) {
	// Get centralized media directory from config
	mediaDir := config.GetString("storage.media_dir")
	if mediaDir == "" {
		mediaDir = "/var/lib/stinkykitty/media"
	}

	// Create uploads subdirectory if it doesn't exist
	uploadsDir := filepath.Join(mediaDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
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

	// Validate file content type using Magic Bytes
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file for validation: %w", err)
	}

	contentType := http.DetectContentType(buffer[:n])
	validTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	isValid := false
	for _, validType := range validTypes {
		if contentType == validType {
			isValid = true
			break
		}
	}
	if !isValid {
		return "", fmt.Errorf("invalid file type: %s (only images allowed)", contentType)
	}

	// Reset file pointer after validation
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to reset file pointer: %w", err)
	}

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

	return filename, nil
}
