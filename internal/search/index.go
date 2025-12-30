// SPDX-License-Identifier: MIT
package search

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// InitFTSIndex creates the FTS5 virtual table for search
func InitFTSIndex(db *gorm.DB) error {
	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Create FTS5 virtual table
	// Note: We use default tokenizer instead of 'porter unicode61' for better
	// compatibility with SQLite builds that don't have the porter extension
	_, err = sqlDB.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
			page_id UNINDEXED,
			site_id UNINDEXED,
			title,
			content
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create FTS index: %w", err)
	}

	return nil
}

// IndexPage adds or updates a page in the FTS index
func IndexPage(db *gorm.DB, page *models.Page) error {
	// Get all blocks for the page
	var blocks []models.Block
	if err := db.Where("page_id = ?", page.ID).Order("`order` ASC").Find(&blocks).Error; err != nil {
		return fmt.Errorf("failed to load blocks: %w", err)
	}

	// Extract text content from all blocks
	var contentParts []string
	for _, block := range blocks {
		content := extractTextFromBlock(block)
		if content != "" {
			contentParts = append(contentParts, content)
		}
	}
	fullContent := strings.Join(contentParts, " ")

	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Delete existing entry if any
	_, err = sqlDB.Exec(`DELETE FROM pages_fts WHERE page_id = ?`, page.ID)
	if err != nil {
		return fmt.Errorf("failed to delete old index entry: %w", err)
	}

	// Only index published pages
	if !page.Published {
		return nil
	}

	// Insert new entry
	_, err = sqlDB.Exec(`
		INSERT INTO pages_fts (page_id, site_id, title, content)
		VALUES (?, ?, ?, ?)
	`, page.ID, page.SiteID, page.Title, fullContent)
	if err != nil {
		return fmt.Errorf("failed to insert index entry: %w", err)
	}

	return nil
}

// RemovePageFromIndex removes a page from the FTS index
func RemovePageFromIndex(db *gorm.DB, pageID uint) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	_, err = sqlDB.Exec(`DELETE FROM pages_fts WHERE page_id = ?`, pageID)
	if err != nil {
		return fmt.Errorf("failed to remove from index: %w", err)
	}

	return nil
}

// SearchResult represents a single search result
type SearchResult struct {
	PageID  uint    `json:"page_id"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	URL     string  `json:"url"`
	Rank    float64 `json:"rank"`
}

// Search performs a full-text search within a site
func Search(db *gorm.DB, siteID uint, query string) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Perform FTS5 search with snippet generation
	rows, err := sqlDB.Query(`
		SELECT
			fts.page_id,
			p.title,
			p.slug,
			snippet(pages_fts, 3, '<mark>', '</mark>', '...', 50) as snippet,
			rank
		FROM pages_fts fts
		INNER JOIN pages p ON fts.page_id = p.id
		WHERE pages_fts MATCH ? AND fts.site_id = ?
		ORDER BY rank
		LIMIT 50
	`, query, siteID)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var slug string
		if err := rows.Scan(&result.PageID, &result.Title, &slug, &result.Snippet, &result.Rank); err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		result.URL = slug
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// extractTextFromBlock extracts searchable text from a block's JSON data
func extractTextFromBlock(block models.Block) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(block.Data), &data); err != nil {
		return ""
	}

	var parts []string

	// Extract based on block type
	switch block.Type {
	case "text":
		if content, ok := data["content"].(string); ok {
			// Strip HTML tags for search indexing
			parts = append(parts, stripHTML(content))
		}
	case "hero":
		if title, ok := data["title"].(string); ok {
			parts = append(parts, title)
		}
		if subtitle, ok := data["subtitle"].(string); ok {
			parts = append(parts, subtitle)
		}
	case "image":
		if caption, ok := data["caption"].(string); ok {
			parts = append(parts, caption)
		}
		if alt, ok := data["alt"].(string); ok {
			parts = append(parts, alt)
		}
	}

	return strings.Join(parts, " ")
}

// stripHTML removes HTML tags from a string (simple implementation)
func stripHTML(s string) string {
	// Simple tag removal - not perfect but good enough for search indexing
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// RebuildSiteIndex rebuilds the entire FTS index for a site
func RebuildSiteIndex(db *gorm.DB, siteID uint) error {
	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Remove all existing entries for this site
	_, err = sqlDB.Exec(`DELETE FROM pages_fts WHERE site_id = ?`, siteID)
	if err != nil {
		return fmt.Errorf("failed to clear site index: %w", err)
	}

	// Get all published pages for the site
	var pages []models.Page
	if err := db.Where("site_id = ? AND published = ?", siteID, true).Find(&pages).Error; err != nil {
		return fmt.Errorf("failed to load pages: %w", err)
	}

	// Index each page
	for _, page := range pages {
		if err := IndexPage(db, &page); err != nil {
			return fmt.Errorf("failed to index page %d: %w", page.ID, err)
		}
	}

	return nil
}
