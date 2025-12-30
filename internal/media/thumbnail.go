package media

import (
	"fmt"
	"image"
	_ "image/gif"   // Register GIF decoder
	"image/jpeg"
	_ "image/png"   // Register PNG decoder
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"  // Register WebP decoder
)

const (
	// ThumbnailWidth is the standard thumbnail width as per design spec
	ThumbnailWidth = 200
	// ThumbnailHeight is the standard thumbnail height as per design spec
	ThumbnailHeight = 200
)

// GenerateStandardThumbnail creates a 200x200 thumbnail as per design specification.
// This is a convenience wrapper around GenerateThumbnail with standard dimensions.
func GenerateStandardThumbnail(srcPath, dstPath string) error {
	return GenerateThumbnail(srcPath, dstPath, ThumbnailWidth, ThumbnailHeight)
}

// GenerateThumbnail creates a thumbnail from an image file
// Uses center crop to maintain exact dimensions
func GenerateThumbnail(srcPath, dstPath string, width, height int) error {
	// Open source image
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %w", err)
	}
	defer srcFile.Close()

	// Decode image
	img, _, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate center crop rectangle
	srcBounds := img.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Calculate aspect ratios
	srcAspect := float64(srcWidth) / float64(srcHeight)
	dstAspect := float64(width) / float64(height)

	var cropRect image.Rectangle
	if srcAspect > dstAspect {
		// Source is wider - crop width
		newWidth := int(float64(srcHeight) * dstAspect)
		x := (srcWidth - newWidth) / 2
		cropRect = image.Rect(x, 0, x+newWidth, srcHeight)
	} else {
		// Source is taller - crop height
		newHeight := int(float64(srcWidth) / dstAspect)
		y := (srcHeight - newHeight) / 2
		cropRect = image.Rect(0, y, srcWidth, y+newHeight)
	}

	// Create thumbnail image
	thumbnail := image.NewRGBA(image.Rect(0, 0, width, height))

	// Scale and draw
	draw.CatmullRom.Scale(thumbnail, thumbnail.Bounds(), img, cropRect, draw.Over, nil)

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save thumbnail
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer dstFile.Close()

	// Always save as JPEG for consistent format
	if err := jpeg.Encode(dstFile, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return nil
}
