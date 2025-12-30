// SPDX-License-Identifier: MIT
package media

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
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

func TestGenerateThumbnailFromPNG(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test PNG image (100x100 blue square)
	srcPath := filepath.Join(tmpDir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}

	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Failed to create test PNG: %v", err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatalf("Failed to encode test PNG: %v", err)
	}
	file.Close()

	// Generate thumbnail from PNG
	dstPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := GenerateThumbnail(srcPath, dstPath, 50, 50); err != nil {
		t.Fatalf("GenerateThumbnail from PNG failed: %v", err)
	}

	// Verify thumbnail exists and is JPEG
	thumbFile, err := os.Open(dstPath)
	if err != nil {
		t.Fatalf("Failed to open thumbnail: %v", err)
	}
	defer thumbFile.Close()

	thumbImg, format, err := image.Decode(thumbFile)
	if err != nil {
		t.Fatalf("Failed to decode thumbnail: %v", err)
	}

	// Verify output format is JPEG
	if format != "jpeg" {
		t.Errorf("Expected JPEG output, got %s", format)
	}

	// Verify dimensions
	bounds := thumbImg.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("Expected 50x50 thumbnail, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestGenerateStandardThumbnail(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test image (300x300)
	srcPath := filepath.Join(tmpDir, "test.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 300; x++ {
			img.Set(x, y, color.RGBA{128, 128, 128, 255})
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

	// Generate standard thumbnail
	dstPath := filepath.Join(tmpDir, "thumb.jpg")
	if err := GenerateStandardThumbnail(srcPath, dstPath); err != nil {
		t.Fatalf("GenerateStandardThumbnail failed: %v", err)
	}

	// Verify thumbnail is 200x200
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
	if bounds.Dx() != 200 || bounds.Dy() != 200 {
		t.Errorf("Expected standard 200x200 thumbnail, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}
