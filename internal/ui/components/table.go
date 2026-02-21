package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

// Package-level cached styles for row/cursor rendering.
var (
	rowEvenStyle = lipgloss.NewStyle()
	rowOddStyle  = lipgloss.NewStyle().Background(theme.ColorElevatedBg)
	cursorStyle  = lipgloss.NewStyle().Foreground(theme.ColorGold)
	cursorActive = cursorStyle.Render("▶ ")
	cursorBlank  = "  "
)

// RowBackground returns a subtle background style for alternating rows.
// Even rows (0, 2, 4...) get no background, odd rows get ElevatedBg.
func RowBackground(index int) lipgloss.Style {
	if index%2 == 1 {
		return rowOddStyle
	}
	return rowEvenStyle
}

// CursorIndicator returns "▶ " in Gold if selected, "  " otherwise.
func CursorIndicator(selected bool) string {
	if selected {
		return cursorActive
	}
	return cursorBlank
}
