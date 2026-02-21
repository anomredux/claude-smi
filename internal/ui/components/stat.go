package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

// Cached styles for stat card rendering.
var (
	statValueStyle = lipgloss.NewStyle().Foreground(theme.ColorBrightText).Bold(true)
	statLabelStyle = lipgloss.NewStyle().Foreground(theme.ColorMutedText)
)

// StatCard renders a Grafana-style stat panel: big number + small label.
type StatCard struct {
	Value string         // e.g. "282,846"
	Sub   string         // optional secondary text e.g. "(+12K cached)"
	Label string         // e.g. "Tokens/min"
	Width int            // character width
	Color lipgloss.Color // color for the value text (optional)
}

// Render returns the stat card as a block of lines.
func (s StatCard) Render() []string {
	w := s.Width
	if w < 8 {
		w = 8
	}

	// Style the value
	var styledVal string
	if s.Color != "" {
		styledVal = lipgloss.NewStyle().Foreground(s.Color).Bold(true).Render(s.Value)
	} else {
		styledVal = statValueStyle.Render(s.Value)
	}

	styledLabel := statLabelStyle.Render(s.Label)

	lines := []string{
		CenterText(styledVal, w),
	}
	if s.Sub != "" {
		styledSub := lipgloss.NewStyle().Foreground(theme.ColorMutedText).Render(s.Sub)
		lines = append(lines, CenterText(styledSub, w))
	}
	lines = append(lines, CenterText(styledLabel, w))

	return lines
}

// RenderStatRow renders multiple stat cards side by side.
func RenderStatRow(cards []StatCard, gap int) string {
	var blocks [][]string
	for _, c := range cards {
		blocks = append(blocks, c.Render())
	}
	lines := JoinHorizontal(blocks, gap)
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
