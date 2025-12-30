package media

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateThumbnail(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create a test image (100x100 red square)
	srcPath := filepath.Join(tmpDir, "test.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	if err := jpeg.Encode(file, img, nil); err != nil {
		file.Close()
		t.Fatalf("Failed to encode test image: %v", err)
	}
	file.Close()

	// Generate thumbnail
	dstPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := GenerateThumbnail(srcPath, dstPath, 50, 50); err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Verify thumbnail exists
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("Thumbnail file was not created")
	}

	// Verify thumbnail dimensions
	thumbFile, err := os.Open(dstPath)
	if err != nil {
		t.Fatalf("Failed to open thumbnail: %v", err)
	}
	defer thumbFile.Close()

	thumbImg, _, err := image.Decode(thumbFile)
	if err != nil {
		t.Fatalf("Failed to decode thumbnail: %v", err)
	}

	bounds := thumbImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width != 50 || height != 50 {
		t.Errorf("Expected thumbnail 50x50, got %dx%d", width, height)
	}
}

func TestGenerateThumbnailMaintainsAspectRatio(t *testing.T) {
	tmpDir := t.TempDir()

	// Create rectangular image (200x100)
	srcPath := filepath.Join(tmpDir, "wide.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}

	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	jpeg.Encode(file, img, nil)
	file.Close()

	// Generate 50x50 thumbnail (should crop to maintain aspect ratio)
	dstPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := GenerateThumbnail(srcPath, dstPath, 50, 50); err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Verify thumbnail is 50x50 (center-cropped)
	thumbFile, err := os.Open(dstPath)
	if err != nil {
		t.Fatalf("Failed to open thumbnail: %v", err)
	}
	defer thumbFile.Close()

	thumbImg, _, err := image.Decode(thumbFile)
	if err != nil {
		t.Fatalf("Failed to decode thumbnail: %v", err)
	}

	bounds := thumbImg.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("Expected 50x50 thumbnail, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}
