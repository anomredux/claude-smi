package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

// Card wraps content in a rounded-border box with a title in the top border.
// When Compact is true, renders title + separator instead of full border.
type Card struct {
	Title   string // pre-styled (e.g. gradient text with ANSI codes)
	Width   int    // total outer width
	Content string // pre-rendered content lines
	Compact bool   // true = no borders, just title + separator
}

// InnerWidth returns the usable content width inside the card.
func (c Card) InnerWidth() int {
	if c.Compact {
		return c.Width - 2 // compact: just indentation, no border
	}
	return c.Width - 4 // 2 border chars + 2 padding spaces
}

// Render returns the styled card string.
func (c Card) Render() string {
	if c.Compact {
		return c.renderCompact()
	}
	return c.renderFull()
}

func (c Card) renderCompact() string {
	sepWidth := c.Width - 4
	if sepWidth < 1 {
		sepWidth = 1
	}
	sep := theme.MutedStyle.Render("  " + strings.Repeat("─", sepWidth))

	if c.Content == "" {
		return c.Title + "\n" + sep
	}
	return c.Title + "\n" + sep + "\n" + c.Content
}

func (c Card) renderFull() string {
	bs := lipgloss.NewStyle().Foreground(theme.ColorBorder)
	innerWidth := c.Width - 2

	// Top border: ╭─ Title ────────╮
	titlePart := ""
	titleVisualWidth := 0
	if c.Title != "" {
		titlePart = " " + c.Title + " "
		titleVisualWidth = lipgloss.Width(titlePart)
	}
	dashesAfterTitle := innerWidth - 1 - titleVisualWidth
	if dashesAfterTitle < 0 {
		// Title is wider than card; truncate the border extension
		dashesAfterTitle = 0
	}
	topLine := bs.Render("╭─") + titlePart + bs.Render(strings.Repeat("─", dashesAfterTitle)+"╮")

	// Body: │ content...          │
	contentWidth := innerWidth - 2
	contentLines := strings.Split(c.Content, "\n")
	var bodyLines []string
	for _, line := range contentLines {
		w := lipgloss.Width(line)
		pad := contentWidth - w
		if pad < 0 {
			pad = 0
		}
		bodyLines = append(bodyLines,
			bs.Render("│")+" "+line+strings.Repeat(" ", pad)+" "+bs.Render("│"))
	}

	// Bottom border: ╰──────────────╯
	bottomLine := bs.Render("╰" + strings.Repeat("─", innerWidth) + "╯")

	return topLine + "\n" + strings.Join(bodyLines, "\n") + "\n" + bottomLine
}
