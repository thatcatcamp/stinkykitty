package blocks

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// RenderBlock renders a block to HTML based on its type and data
func RenderBlock(blockType string, dataJSON string) (string, error) {
	switch blockType {
	case "text":
		return renderTextBlock(dataJSON)
	case "image":
		return renderImageBlock(dataJSON)
	case "heading":
		return renderHeadingBlock(dataJSON)
	case "quote":
		return renderQuoteBlock(dataJSON)
	case "button":
		return renderButtonBlock(dataJSON)
	case "video":
		return renderVideoBlock(dataJSON)
	case "spacer":
		return renderSpacerBlock(dataJSON)
	case "contact":
		return renderContactBlock(dataJSON)
	case "columns":
		return renderColumnsBlock(dataJSON)
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

// HeadingBlockData represents the JSON structure for heading blocks
type HeadingBlockData struct {
	Level int    `json:"level"` // 2-6 for h2-h6
	Text  string `json:"text"`
}

// renderHeadingBlock renders a heading block
func renderHeadingBlock(dataJSON string) (string, error) {
	var data HeadingBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse heading block data: %w", err)
	}

	// Default to h2 if level is invalid
	if data.Level < 2 || data.Level > 6 {
		data.Level = 2
	}

	safeText := html.EscapeString(data.Text)
	return fmt.Sprintf(`<h%d class="heading-block">%s</h%d>`, data.Level, safeText, data.Level), nil
}

// QuoteBlockData represents the JSON structure for quote blocks
type QuoteBlockData struct {
	Quote  string `json:"quote"`
	Author string `json:"author"`
}

// renderQuoteBlock renders a quote block
func renderQuoteBlock(dataJSON string) (string, error) {
	var data QuoteBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse quote block data: %w", err)
	}

	safeQuote := html.EscapeString(data.Quote)
	safeAuthor := html.EscapeString(data.Author)

	htmlStr := `<blockquote class="quote-block" style="border-left: 4px solid #ddd; padding-left: 20px; margin: 1.5em 0; font-style: italic; color: #555;">`
	htmlStr += fmt.Sprintf(`<p style="margin: 0 0 10px 0; font-size: 1.1em;">%s</p>`, safeQuote)

	if safeAuthor != "" {
		htmlStr += fmt.Sprintf(`<footer style="font-size: 0.9em; color: #888; font-style: normal;">— %s</footer>`, safeAuthor)
	}

	htmlStr += `</blockquote>`
	return htmlStr, nil
}

// ButtonBlockData represents the JSON structure for button blocks
type ButtonBlockData struct {
	Text  string `json:"text"`
	URL   string `json:"url"`
	Style string `json:"style"` // "primary" or "secondary"
}

// renderButtonBlock renders a button/CTA block
func renderButtonBlock(dataJSON string) (string, error) {
	var data ButtonBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse button block data: %w", err)
	}

	safeText := html.EscapeString(data.Text)
	safeURL := html.EscapeString(data.URL)

	// Choose button color based on style
	bgColor := "#007bff"
	if data.Style == "secondary" {
		bgColor = "#6c757d"
	}

	return fmt.Sprintf(`<div class="button-block" style="margin: 1.5em 0;">
		<a href="%s" style="display: inline-block; padding: 12px 24px; background: %s; color: white; text-decoration: none; border-radius: 4px; font-weight: 500; transition: opacity 0.2s;" onmouseover="this.style.opacity='0.9'" onmouseout="this.style.opacity='1'">%s</a>
	</div>`, safeURL, bgColor, safeText), nil
}

// VideoBlockData represents the JSON structure for video blocks
type VideoBlockData struct {
	URL string `json:"url"` // YouTube or Vimeo URL
}

// renderVideoBlock renders a video embed block
func renderVideoBlock(dataJSON string) (string, error) {
	var data VideoBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse video block data: %w", err)
	}

	// Convert YouTube/Vimeo URLs to embed URLs
	embedURL := convertToEmbedURL(data.URL)
	if embedURL == "" {
		return "", fmt.Errorf("invalid video URL: %s", data.URL)
	}

	return fmt.Sprintf(`<div class="video-block" style="position: relative; padding-bottom: 56.25%%; height: 0; overflow: hidden; margin: 1.5em 0;">
		<iframe src="%s" style="position: absolute; top: 0; left: 0; width: 100%%; height: 100%%;" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>
	</div>`, embedURL), nil
}

// convertToEmbedURL converts YouTube/Vimeo URLs to embed format
func convertToEmbedURL(url string) string {
	// YouTube patterns
	if strings.Contains(url, "youtube.com/watch?v=") {
		parts := strings.Split(url, "v=")
		if len(parts) == 2 {
			videoID := strings.Split(parts[1], "&")[0]
			return "https://www.youtube.com/embed/" + videoID
		}
	}
	if strings.Contains(url, "youtu.be/") {
		parts := strings.Split(url, "youtu.be/")
		if len(parts) == 2 {
			videoID := strings.Split(parts[1], "?")[0]
			return "https://www.youtube.com/embed/" + videoID
		}
	}

	// Vimeo patterns
	if strings.Contains(url, "vimeo.com/") {
		parts := strings.Split(url, "vimeo.com/")
		if len(parts) == 2 {
			videoID := strings.Split(parts[1], "/")[0]
			videoID = strings.Split(videoID, "?")[0]
			return "https://player.vimeo.com/video/" + videoID
		}
	}

	// Already an embed URL
	if strings.Contains(url, "youtube.com/embed/") || strings.Contains(url, "player.vimeo.com/video/") {
		return url
	}

	return ""
}

// SpacerBlockData represents the JSON structure for spacer blocks
type SpacerBlockData struct {
	Height int `json:"height"` // Height in pixels
}

// renderSpacerBlock renders a spacer block
func renderSpacerBlock(dataJSON string) (string, error) {
	var data SpacerBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse spacer block data: %w", err)
	}

	// Default to 40px if not specified or invalid
	if data.Height <= 0 {
		data.Height = 40
	}

	return fmt.Sprintf(`<div class="spacer-block" style="height: %dpx;"></div>`, data.Height), nil
}

// ContactBlockData represents the JSON structure for contact blocks
type ContactBlockData struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

// renderContactBlock renders an embedded contact form
func renderContactBlock(dataJSON string) (string, error) {
	var data ContactBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse contact block data: %w", err)
	}

	if data.Title == "" {
		data.Title = "Get in Touch"
	}

	// Escape HTML to prevent XSS
	title := html.EscapeString(data.Title)
	subtitle := html.EscapeString(data.Subtitle)

	formHTML := fmt.Sprintf(`<div class="contact-form-block" style="margin: 40px 0;">
	<h2>%s</h2>`, title)

	if subtitle != "" {
		formHTML += fmt.Sprintf(`
	<p>%s</p>`, subtitle)
	}

	formHTML += `
	<form method="POST" action="/contact" style="max-width: 500px;">
		<div style="margin-bottom: 20px;">
			<label for="contact-name" style="display: block; margin-bottom: 8px; font-weight: 500;">Name:</label>
			<input type="text" id="contact-name" name="name" required style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box;">
		</div>
		<div style="margin-bottom: 20px;">
			<label for="contact-email" style="display: block; margin-bottom: 8px; font-weight: 500;">Email:</label>
			<input type="email" id="contact-email" name="email" required style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box;">
		</div>
		<div style="margin-bottom: 20px;">
			<label for="contact-subject" style="display: block; margin-bottom: 8px; font-weight: 500;">Subject:</label>
			<input type="text" id="contact-subject" name="subject" required style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box;">
		</div>
		<div style="margin-bottom: 20px;">
			<label for="contact-message" style="display: block; margin-bottom: 8px; font-weight: 500;">Message:</label>
			<textarea id="contact-message" name="message" required rows="6" style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; box-sizing: border-box; font-family: inherit;"></textarea>
		</div>
		<button type="submit" style="background: var(--color-primary, #2563eb); color: white; padding: 12px 24px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; font-weight: 600;">Send Message</button>
	</form>
</div>`

	return formHTML, nil
}

// ColumnsBlockData represents the JSON structure for column blocks
type ColumnsBlockData struct {
	ColumnCount int      `json:"column_count"` // 2, 3, or 4
	Columns     []Column `json:"columns"`
}

type Column struct {
	Content string `json:"content"` // HTML content for this column
}

// renderColumnsBlock renders a multi-column layout block
func renderColumnsBlock(dataJSON string) (string, error) {
	var data ColumnsBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse columns block data: %w", err)
	}

	// Validate column count
	if data.ColumnCount < 2 || data.ColumnCount > 4 {
		data.ColumnCount = 2
	}

	// Create HTML sanitization policy that allows images, links, buttons, and formatting
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("button")
	policy.AllowAttrs("class", "style").OnElements("button", "div", "p", "h1", "h2", "h3", "h4", "h5", "h6", "span")
	policy.AllowAttrs("src", "alt", "title", "width", "height", "style").OnElements("img")

	htmlStr := `<div class="columns-block" style="display: grid; grid-template-columns: repeat(` + fmt.Sprintf("%d", data.ColumnCount) + `, 1fr); gap: var(--spacing-lg, 24px); margin: var(--spacing-lg, 24px) 0;">`

	// Render each column
	for _, col := range data.Columns {
		// Sanitize content using bluemonday policy (allows safe HTML)
		safeContent := policy.Sanitize(col.Content)
		// Convert remaining newlines to <br> for display
		safeContent = strings.ReplaceAll(safeContent, "\n", "<br>")

		htmlStr += fmt.Sprintf(`
			<div class="column" style="min-width: 0;">
				%s
			</div>
		`, safeContent)
	}

	htmlStr += `</div>`

	return htmlStr, nil
}
