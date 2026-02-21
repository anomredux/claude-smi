package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCard_InnerWidth(t *testing.T) {
	c := Card{Width: 80}
	if got := c.InnerWidth(); got != 76 {
		t.Errorf("InnerWidth() = %d, want 76", got)
	}
	c.Compact = true
	if got := c.InnerWidth(); got != 78 {
		t.Errorf("Compact InnerWidth() = %d, want 78", got)
	}
}

func TestCard_RenderFull_ContainsBorders(t *testing.T) {
	c := Card{
		Title:   "Test Title",
		Width:   40,
		Content: "Hello World",
	}
	out := c.Render()
	if !strings.Contains(out, "╭") {
		t.Error("expected top-left corner")
	}
	if !strings.Contains(out, "╯") {
		t.Error("expected bottom-right corner")
	}
	if !strings.Contains(out, "Test Title") {
		t.Error("expected title in output")
	}
	if !strings.Contains(out, "Hello World") {
		t.Error("expected content in output")
	}
}

func TestCard_RenderFull_WidthConsistent(t *testing.T) {
	c := Card{
		Title:   "Title",
		Width:   50,
		Content: "Line 1\nLine 2",
	}
	out := c.Render()
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != 50 {
			t.Errorf("line %d width = %d, want 50: %q", i, w, line)
		}
	}
}

func TestCard_RenderCompact_NoBorders(t *testing.T) {
	c := Card{
		Title:   "Test",
		Width:   40,
		Content: "Content",
		Compact: true,
	}
	out := c.Render()
	if strings.Contains(out, "╭") {
		t.Error("compact mode should not have border corners")
	}
	if !strings.Contains(out, "─") {
		t.Error("compact mode should have separator")
	}
	if !strings.Contains(out, "Content") {
		t.Error("expected content in output")
	}
}

func TestCard_EmptyContent(t *testing.T) {
	c := Card{Title: "Empty", Width: 30, Content: ""}
	out := c.Render()
	if !strings.Contains(out, "╭") {
		t.Error("should still render borders with empty content")
	}
}
