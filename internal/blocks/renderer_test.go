// SPDX-License-Identifier: MIT
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

func TestRenderColumnsBlock(t *testing.T) {
	dataJSON := `{"column_count":2,"columns":[{"content":"Column 1"},{"content":"Column 2"}]}`
	html, err := RenderBlock("columns", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if !strings.Contains(html, "columns-block") {
		t.Errorf("Expected HTML to contain columns-block class, got: %s", html)
	}

	if !strings.Contains(html, "Column 1") {
		t.Errorf("Expected HTML to contain 'Column 1', got: %s", html)
	}

	if !strings.Contains(html, "Column 2") {
		t.Errorf("Expected HTML to contain 'Column 2', got: %s", html)
	}

	if !strings.Contains(html, "grid-template-columns: repeat(2, 1fr)") {
		t.Errorf("Expected HTML to have 2 column grid, got: %s", html)
	}
}

func TestRenderColumnsBlockWith3Columns(t *testing.T) {
	dataJSON := `{"column_count":3,"columns":[{"content":"Col 1"},{"content":"Col 2"},{"content":"Col 3"}]}`
	html, err := RenderBlock("columns", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if !strings.Contains(html, "grid-template-columns: repeat(3, 1fr)") {
		t.Errorf("Expected HTML to have 3 column grid, got: %s", html)
	}
}

func TestRenderColumnsBlockEscapesHTML(t *testing.T) {
	dataJSON := `{"column_count":2,"columns":[{"content":"<script>alert('xss')</script>"},{"content":"Safe content"}]}`
	html, err := RenderBlock("columns", dataJSON)

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

func TestRenderColumnsBlockValidatesColumnCount(t *testing.T) {
	// Test with invalid column count (should default to 2)
	dataJSON := `{"column_count":10,"columns":[{"content":"Test"}]}`
	html, err := RenderBlock("columns", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	// Should default to 2 columns when count is invalid
	if !strings.Contains(html, "grid-template-columns: repeat(2, 1fr)") {
		t.Errorf("Expected invalid column count to default to 2, got: %s", html)
	}
}
