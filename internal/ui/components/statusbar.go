package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/theme"
)

// StatusBar renders the bottom status bar with key hints.
type StatusBar struct {
	Width int
}

// Render returns the status bar: separator + key hints.
func (s StatusBar) Render() string {
	keys := s.renderKeyHints()
	sep := theme.MutedStyle.Render(strings.Repeat("─", s.Width))
	return sep + "\n" + keys
}

// 5 key hints mapped to 5-color gradient: SkyBlue → Lavender → Mauve → Peach → Gold
var keyColors = []lipgloss.Color{
	theme.ColorSkyBlue,
	theme.ColorLavender,
	theme.ColorMauve,
	theme.ColorPeach,
	theme.ColorGold,
}

func (s StatusBar) renderKeyHints() string {
	hints := []struct{ key, desc string }{
		{"?", i18n.T("status_help")},
		{"s", i18n.T("status_settings")},
		{"p", i18n.T("status_project")},
		{"r", i18n.T("status_refresh")},
		{"q", i18n.T("status_quit")},
	}

	var parts []string
	for i, h := range hints {
		color := keyColors[i%len(keyColors)]
		keyStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
		parts = append(parts, keyStyle.Render(h.key)+" "+theme.MutedStyle.Render(h.desc))
	}

	return "  " + strings.Join(parts, "  ")
}
