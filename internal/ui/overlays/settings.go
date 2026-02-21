package overlays

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/config"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/theme"
)

// ConfigChangedMsg signals that config has been updated.
type ConfigChangedMsg struct {
	Config config.Config
}

type settingsField struct {
	label   string
	key     string
	options []string
	value   string
}

type SettingsOverlay struct {
	cfg      config.Config
	cfgPath  string
	fields   []settingsField
	cursor   int
	dirty    bool
	animTick uint
}

func NewSettingsOverlay(cfg config.Config, cfgPath string) *SettingsOverlay {
	s := &SettingsOverlay{
		cfg:     cfg,
		cfgPath: cfgPath,
	}
	s.buildFields()
	return s
}

func (s *SettingsOverlay) SetAnimTick(tick uint) {
	s.animTick = tick
}

func (s *SettingsOverlay) buildFields() {
	s.fields = []settingsField{
		{label: i18n.T("setting_timezone"), key: "timezone", options: commonTimezones(), value: s.cfg.General.Timezone},
		{label: i18n.T("setting_refresh"), key: "interval", options: []string{"5", "10", "15", "30", "60"}, value: fmt.Sprintf("%d", s.cfg.General.Interval)},
		{label: i18n.T("setting_language"), key: "language", options: []string{"en"}, value: s.cfg.General.Language},
	}
}

func (s *SettingsOverlay) Update(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if s.cursor < len(s.fields)-1 {
			s.cursor++
		}
	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
		}
	case "enter", " ", "l", "right":
		s.cycleOption(1)
	case "h", "left":
		s.cycleOption(-1)
	case "esc", "s":
		if s.dirty {
			_ = config.Save(s.cfg, s.cfgPath)
			return true, func() tea.Msg { return ConfigChangedMsg{Config: s.cfg} }
		}
		return true, nil
	}
	return false, nil
}

func (s *SettingsOverlay) cycleOption(dir int) {
	f := &s.fields[s.cursor]
	idx := -1
	for i, o := range f.options {
		if o == f.value {
			idx = i
			break
		}
	}
	if idx < 0 {
		idx = 0
	}
	idx = (idx + dir + len(f.options)) % len(f.options)
	f.value = f.options[idx]
	s.dirty = true
	s.applyToConfig(f.key, f.value)
}

func (s *SettingsOverlay) applyToConfig(key, value string) {
	switch key {
	case "timezone":
		s.cfg.General.Timezone = value
	case "interval":
		var n int
		fmt.Sscanf(value, "%d", &n)
		if n > 0 {
			s.cfg.General.Interval = n
		}
	case "language":
		s.cfg.General.Language = value
	}
}

func (s *SettingsOverlay) Render(width, height int) string {
	bg := theme.ColorCardBg
	title := theme.AnimatedGradientText(i18n.T("settings"), s.animTick, bg)

	var rows []string
	for i, f := range s.fields {
		labelStyle := lipgloss.NewStyle().Foreground(theme.ColorBodyText).Background(bg)
		valueStyle := lipgloss.NewStyle().Foreground(theme.ColorSkyBlue).Background(bg)

		if i == s.cursor {
			labelStyle = lipgloss.NewStyle().Foreground(theme.ColorGold).Bold(true).Background(bg)
			valueStyle = lipgloss.NewStyle().Foreground(theme.ColorBrightText).Bold(true).Background(bg)
		}

		arrow := "  "
		if i == s.cursor {
			arrow = lipgloss.NewStyle().Foreground(theme.ColorGold).Background(bg).Render("> ")
		}

		label := fmt.Sprintf("%-16s", f.label)
		rows = append(rows, fmt.Sprintf("  %s%s%s",
			arrow,
			labelStyle.Render(label),
			valueStyle.Render(" "+f.value),
		))
	}

	content := title + "\n\n" + strings.Join(rows, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(theme.ColorMutedText).Background(bg).Render(i18n.T("settings_help"))

	boxWidth := 50
	if width < 54 {
		boxWidth = width - 4
	}

	return theme.CardStyle.
		Width(boxWidth).
		Render(content)
}

func commonTimezones() []string {
	return []string{
		"UTC",
		"US/Eastern", "US/Central", "US/Mountain", "US/Pacific",
		"Europe/London", "Europe/Berlin", "Europe/Paris",
		"Asia/Tokyo", "Asia/Seoul", "Asia/Shanghai", "Asia/Singapore",
		"Australia/Sydney",
	}
}
