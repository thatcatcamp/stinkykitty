package blocks

import (
	"strings"
	"testing"
)

func TestRenderTextBlock(t *testing.T) {
	dataJSON := `{"content":"Hello world"}`
	html, err := RenderBlock("text", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if !strings.Contains(html, "Hello world") {
		t.Errorf("Expected HTML to contain 'Hello world', got: %s", html)
	}

	if !strings.Contains(html, `class="text-block"`) {
		t.Errorf("Expected HTML to have text-block class, got: %s", html)
	}
}

func TestRenderTextBlockWithLineBreaks(t *testing.T) {
	dataJSON := `{"content":"Line 1\nLine 2"}`
	html, err := RenderBlock("text", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if !strings.Contains(html, "<br>") {
		t.Errorf("Expected HTML to contain <br> for line breaks, got: %s", html)
	}
}

func TestRenderTextBlockEscapesHTML(t *testing.T) {
	dataJSON := `{"content":"<script>alert('xss')</script>"}`
	html, err := RenderBlock("text", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if strings.Contains(html, "<script>") {
		t.Errorf("HTML should be escaped, got: %s", html)
	}

	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("Expected escaped HTML, got: %s", html)
	}
}

func TestRenderUnknownBlockType(t *testing.T) {
	_, err := RenderBlock("unknown", `{}`)

	if err == nil {
		t.Error("Expected error for unknown block type")
	}
}
