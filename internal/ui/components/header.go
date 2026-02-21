package components

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

// TabBar renders a numbered tab bar with active highlighting and a bottom separator.
type TabBar struct {
	ViewNames     []string
	ActiveIndex   int
	Width         int
	ActiveProject string
}

// Package-level cached styles for tab bar rendering.
var (
	tabActiveStyle = lipgloss.NewStyle().
			Foreground(theme.ColorGold).
			Background(theme.ColorElevatedBg).
			Bold(true).
			Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(theme.ColorMutedText).
				Padding(0, 1)
)

// Render returns the styled tab bar with bottom separator line.
func (tb TabBar) Render() string {
	var tabs []string
	for i, name := range tb.ViewNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if i == tb.ActiveIndex {
			tabs = append(tabs, tabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, tabInactiveStyle.Render(label))
		}
	}

	line := strings.Join(tabs, "")

	if tb.ActiveProject != "" {
		projectName := filepath.Base(tb.ActiveProject)
		line += "  " +
			lipgloss.NewStyle().Foreground(theme.ColorMauve).Render("["+projectName+"]")
	}

	tabLine := lipgloss.NewStyle().
		Width(tb.Width).
		Padding(0, 1).
		Render(line)

	sep := theme.MutedStyle.Render(strings.Repeat("─", tb.Width))

	return tabLine + "\n" + sep
}
