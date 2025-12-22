package blocks

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// RenderBlock renders a block to HTML based on its type and data
func RenderBlock(blockType string, dataJSON string) (string, error) {
	switch blockType {
	case "text":
		return renderTextBlock(dataJSON)
	default:
		return "", fmt.Errorf("unknown block type: %s", blockType)
	}
}

// TextBlockData represents the JSON structure for text blocks
type TextBlockData struct {
	Content string `json:"content"`
}

// renderTextBlock renders a text block with HTML escaping and line break preservation
func renderTextBlock(dataJSON string) (string, error) {
	var data TextBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse text block data: %w", err)
	}

	// Escape HTML to prevent XSS
	safe := html.EscapeString(data.Content)

	// Preserve line breaks
	formatted := strings.ReplaceAll(safe, "\n", "<br>")

	return fmt.Sprintf(`<div class="text-block">%s</div>`, formatted), nil
}
