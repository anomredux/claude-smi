package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

// Cached styles for gauge rendering.
var (
	gaugeLabelStyle = lipgloss.NewStyle().Foreground(theme.ColorBodyText)
)

// SemicircleGauge renders a semicircle arc gauge using braille characters.
type SemicircleGauge struct {
	Label   string  // e.g. "5h Session"
	Percent float64 // 0.0 ~ N (can exceed 1.0)
	Width   int     // character width of the gauge area
}

// Render returns the gauge as a block of lines.
func (g SemicircleGauge) Render() []string {
	w := g.Width
	if w < 10 {
		w = 10
	}

	// Semicircle dimensions
	arcH := w / 4
	if arcH < 3 {
		arcH = 3
	}
	if arcH > 6 {
		arcH = 6
	}

	canvas := NewBrailleCanvas(w, arcH)
	cx := float64(canvas.PixelWidth()) / 2
	cy := float64(canvas.PixelHeight()) - 1
	// outerR must fit within both width and height
	outerR := cy
	if cx-0.5 < outerR {
		outerR = cx - 0.5
	}
	innerR := outerR * 0.62 // thinner arc for cleaner look

	pct := g.Percent
	if pct > 1 {
		pct = 1 // cap visual fill at 100%
	}
	if pct < 0 {
		pct = 0
	}

	canvas.DrawSemicircle(cx, cy, outerR, innerR, pct)

	dimColor := theme.ColorGaugeDim
	arcLines := canvas.RenderGradient(dimColor)

	// Format percentage text with gradient color matching the utilization level
	pctText := fmt.Sprintf("%.1f%%", g.Percent*100)
	pctColor := theme.MultiStopGradient(pct, theme.ProgressGradient)
	styledPct := lipgloss.NewStyle().Foreground(lipgloss.Color(pctColor)).Bold(true).Render(pctText)

	// Build the block: label, arc, percentage
	var block []string
	block = append(block, CenterText(gaugeLabelStyle.Render(g.Label), w))
	block = append(block, "")
	block = append(block, arcLines...)
	block = append(block, CenterText(styledPct, w))

	return block
}

// RenderGaugeRow renders multiple gauges side by side.
func RenderGaugeRow(gauges []SemicircleGauge, gap int) string {
	var blocks [][]string
	for _, g := range gauges {
		blocks = append(blocks, g.Render())
	}
	lines := JoinHorizontal(blocks, gap)
	return strings.Join(lines, "\n")
}

