package media

import (
	"encoding/json"

	"github.com/thatcatcamp/stinkykitty/internal/blocks"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// UsageLocation represents where an image is used
type UsageLocation struct {
	PageID    uint
	PageTitle string
	BlockID   uint
	BlockType string
}

// FindImageUsage finds all blocks that reference a specific image URL
func FindImageUsage(db *gorm.DB, siteID uint, imageURL string) []UsageLocation {
	var usages []UsageLocation

	// Get all pages for this site
	var pages []models.Page
	db.Where("site_id = ? AND deleted_at IS NULL", siteID).Find(&pages)

	// For each page, check blocks
	for _, page := range pages {
		var pageBlocks []models.Block
		db.Where("page_id = ? AND deleted_at IS NULL", page.ID).Find(&pageBlocks)

		for _, block := range pageBlocks {
			if containsImageURL(block, imageURL) {
				usages = append(usages, UsageLocation{
					PageID:    page.ID,
					PageTitle: page.Title,
					BlockID:   block.ID,
					BlockType: block.Type,
				})
			}
		}
	}

	return usages
}

// containsImageURL checks if a block contains a specific image URL
func containsImageURL(block models.Block, imageURL string) bool {
	switch block.Type {
	case "image":
		var data blocks.ImageBlockData
		if err := json.Unmarshal([]byte(block.Data), &data); err != nil {
			return false
		}
		return data.URL == imageURL

	case "button":
		// Button blocks might have background images in the future
		// For now, just check if the URL appears in the data
		return false

	case "columns":
		// Columns can contain nested blocks with images
		// For V1, we'll do simple string matching
		// TODO: Proper nested block parsing in V2
		return false

	default:
		return false
	}
}
