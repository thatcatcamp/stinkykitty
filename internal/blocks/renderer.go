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
	case "image":
		return renderImageBlock(dataJSON)
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

// ImageBlockData represents the JSON structure for image blocks
type ImageBlockData struct {
	URL     string `json:"url"`
	Alt     string `json:"alt"`
	Caption string `json:"caption"`
}

// renderImageBlock renders an image block
func renderImageBlock(dataJSON string) (string, error) {
	var data ImageBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse image block data: %w", err)
	}

	// Escape HTML in alt and caption
	safeAlt := html.EscapeString(data.Alt)
	safeCaption := html.EscapeString(data.Caption)

	// Build image HTML
	htmlStr := fmt.Sprintf(`<div class="image-block">
		<img src="%s" alt="%s" style="max-width: 100%%; height: auto; display: block;">`, data.URL, safeAlt)

	if safeCaption != "" {
		htmlStr += fmt.Sprintf(`<p style="font-size: 14px; color: #666; margin-top: 8px; font-style: italic;">%s</p>`, safeCaption)
	}

	htmlStr += `</div>`

	return htmlStr, nil
}
