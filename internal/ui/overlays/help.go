package overlays

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/theme"
)

type HelpOverlay struct {
	AnimTick uint
}

func NewHelpOverlay() *HelpOverlay {
	return &HelpOverlay{}
}

func (h *HelpOverlay) Render(width, height int) string {
	title := theme.AnimatedGradientText(i18n.T("keyboard_shortcuts"), h.AnimTick, theme.ColorCardBg)

	bindings := []struct {
		key  string
		desc string
	}{
		{"1 / 2 / 3", i18n.T("help_switch_views")},
		{"Tab / Shift+Tab", i18n.T("help_cycle_views")},
		{"", ""},
		{"j / k / Down / Up", i18n.T("help_navigate")},
		{"Enter", i18n.T("help_drill_down")},
		{"Esc / Backspace", i18n.T("help_go_back")},
		{"", ""},
		{"? ", i18n.T("help_toggle_help")},
		{"s", i18n.T("help_open_settings")},
		{"r", i18n.T("help_force_refresh")},
		{"p", i18n.T("help_project_filter")},
		{"", ""},
		{"h / l / Left / Right", i18n.T("help_navigate_months")},
		{"", ""},
		{"PgUp / PgDn", i18n.T("help_page_scroll")},
		{"g / G", i18n.T("help_top_bottom")},
		{"Mouse Wheel", i18n.T("help_mouse_scroll")},
		{"", ""},
		{"q / Ctrl+C", i18n.T("help_quit")},
	}

	maxKeyLen := 0
	for _, b := range bindings {
		if len(b.key) > maxKeyLen {
			maxKeyLen = len(b.key)
		}
	}

	bg := theme.ColorCardBg
	keyStyle := lipgloss.NewStyle().Foreground(theme.ColorGold).Bold(true).Background(bg)
	descStyle := lipgloss.NewStyle().Foreground(theme.ColorBodyText).Background(bg)

	var rows []string
	for _, b := range bindings {
		if b.key == "" {
			rows = append(rows, "")
			continue
		}
		padded := fmt.Sprintf("%-*s", maxKeyLen, b.key)
		rows = append(rows, fmt.Sprintf("  %s%s",
			keyStyle.Render(padded),
			descStyle.Render("  "+b.desc),
		))
	}

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(theme.ColorMutedText).Background(bg).Render(i18n.T("help_close"))

	boxWidth := 65
	if width < 69 {
		boxWidth = width - 4
	}

	return theme.CardStyle.
		Width(boxWidth).
		Render(content)
}
