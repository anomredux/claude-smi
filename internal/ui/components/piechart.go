package components

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

// PieSlice represents one slice of the pie chart.
type PieSlice struct {
	Label      string
	Value      float64 // absolute value (e.g. cost)
	Percentage float64 // 0-100
}

// PieChart renders a braille circle pie chart with a legend.
type PieChart struct {
	Slices    []PieSlice
	ChartSize int // character width/height of the circle area
	Width     int // total available width (chart + legend)
}

// Render returns the pie chart with legend as a block of lines.
func (p PieChart) Render() string {
	if len(p.Slices) == 0 {
		return theme.MutedStyle.Render("  No data")
	}

	chartSize := p.ChartSize
	if chartSize < 8 {
		chartSize = 8
	}

	palette := SlicePalette()
	maxColors := len(palette)

	// Prepare slices: sort by percentage descending, group small ones as "Others"
	slices := p.prepareSlices(maxColors)

	// Draw the pie chart
	chartH := chartSize / 2 // semicircle aspect ratio
	if chartH < 4 {
		chartH = 4
	}
	canvas := NewBrailleCanvas(chartSize, chartH)
	cx := float64(canvas.PixelWidth()) / 2
	cy := float64(canvas.PixelHeight()) / 2
	outerR := math.Min(cx, cy) - 0.5
	innerR := outerR * 0.45 // donut shape

	// Compute draw percentages with minimum arc enforcement.
	// Small slices need enough pixels to render stably on the braille grid.
	// Legend still shows real percentages.
	drawPcts := enforceMinArc(slices, outerR)

	// Draw slices
	startAngle := 0.0
	for i := range slices {
		sliceAngle := drawPcts[i] / 100.0 * 2 * math.Pi
		if sliceAngle < 0.001 {
			continue
		}
		endAngle := startAngle + sliceAngle
		if endAngle > 2*math.Pi {
			endAngle = 2 * math.Pi
		}
		colorIdx := i
		if colorIdx >= maxColors {
			colorIdx = maxColors - 1
		}
		canvas.DrawRing(cx, cy, outerR, innerR, startAngle, endAngle, colorIdx)
		startAngle = endAngle
	}

	chartLines := canvas.Render(palette, theme.ColorPieChartBg)

	// Build legend with aligned columns
	legendLines := p.buildLegend(slices, palette)

	// Join chart and legend horizontally
	combined := JoinHorizontal([][]string{chartLines, legendLines}, 3)
	return strings.Join(combined, "\n")
}

// prepareSlices sorts slices, groups excess into "Others", removes 0%.
func (p PieChart) prepareSlices(maxColors int) []PieSlice {
	// Filter out 0% slices (rounded to 2 decimal places)
	var filtered []PieSlice
	for _, s := range p.Slices {
		rounded := math.Round(s.Percentage*100) / 100
		if rounded > 0 {
			filtered = append(filtered, s)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	// Sort by percentage descending, then label for stable ordering
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Percentage != filtered[j].Percentage {
			return filtered[i].Percentage > filtered[j].Percentage
		}
		return filtered[i].Label < filtered[j].Label
	})

	// If within color limit, return as-is
	if len(filtered) <= maxColors {
		return filtered
	}

	// Group excess into "Others"
	result := make([]PieSlice, 0, maxColors)
	result = append(result, filtered[:maxColors-1]...)

	others := PieSlice{Label: "Others"}
	for _, s := range filtered[maxColors-1:] {
		others.Value += s.Value
		others.Percentage += s.Percentage
	}
	result = append(result, others)

	return result
}

// enforceMinArc returns adjusted percentages for drawing.
// Small slices are bumped up so they span at least 4 braille pixels of arc,
// borrowing from the largest slice. Original slice data is not modified.
func enforceMinArc(slices []PieSlice, outerR float64) []float64 {
	pcts := make([]float64, len(slices))
	for i, s := range slices {
		pcts[i] = s.Percentage
	}
	if len(pcts) <= 1 {
		return pcts
	}

	// Minimum arc = 4 pixels at outer radius → minimum percentage
	minArc := 4.0 / outerR                    // radians
	minPct := minArc / (2 * math.Pi) * 100.0  // percentage

	// Boost small slices, track deficit
	var deficit float64
	largestIdx := 0
	for i, p := range pcts {
		if p > pcts[largestIdx] {
			largestIdx = i
		}
		if p > 0 && p < minPct {
			deficit += minPct - p
			pcts[i] = minPct
		}
	}

	// Subtract deficit from the largest slice
	if deficit > 0 {
		pcts[largestIdx] -= deficit
	}

	return pcts
}

// buildLegend creates the legend with properly aligned columns.
func (p PieChart) buildLegend(slices []PieSlice, palette []string) []string {
	if len(slices) == 0 {
		return nil
	}

	// Calculate column widths
	maxLabel := 0
	maxPct := 0
	maxVal := 0

	type legendEntry struct {
		colorHex string
		label    string
		pctStr   string
		valStr   string
	}

	entries := make([]legendEntry, len(slices))
	for i, s := range slices {
		colorIdx := i
		if colorIdx >= len(palette) {
			colorIdx = len(palette) - 1
		}
		pctStr := fmt.Sprintf("%.1f%%", s.Percentage)
		valStr := fmt.Sprintf("$%.2f", s.Value)

		entries[i] = legendEntry{
			colorHex: palette[colorIdx],
			label:    s.Label,
			pctStr:   pctStr,
			valStr:   valStr,
		}

		if len(s.Label) > maxLabel {
			maxLabel = len(s.Label)
		}
		if len(pctStr) > maxPct {
			maxPct = len(pctStr)
		}
		if len(valStr) > maxVal {
			maxVal = len(valStr)
		}
	}

	labelStyle := lipgloss.NewStyle().Foreground(theme.ColorBodyText)

	var lines []string
	for _, e := range entries {
		square := ColoredSquare(e.colorHex)
		label := labelStyle.Render(fmt.Sprintf("%-*s", maxLabel, e.label))
		pctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(e.colorHex))
		pct := pctStyle.Render(fmt.Sprintf("%*s", maxPct, e.pctStr))
		valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(e.colorHex))
		val := valStyle.Render(fmt.Sprintf("%*s", maxVal, e.valStr))
		lines = append(lines, fmt.Sprintf("%s %s  %s  %s", square, label, pct, val))
	}

	return lines
}
