package theme

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Base palette
var (
	ColorLavender = lipgloss.Color("#9f99d1") // C1
	ColorSkyBlue  = lipgloss.Color("#86bada") // C2
	ColorMauve    = lipgloss.Color("#dbaad7") // C3
	ColorPeach    = lipgloss.Color("#f6bcb0") // C4
	ColorGold     = lipgloss.Color("#ffe3b3") // C5
)

// Background tones (dark theme)
var (
	ColorBaseBg     = lipgloss.Color("#1a1b2e")
	ColorCardBg     = lipgloss.Color("#232438")
	ColorElevatedBg = lipgloss.Color("#2a2b42")
	ColorBorder     = lipgloss.Color("#3a3b52")
	ColorMutedText  = lipgloss.Color("#6b6d8a")
	ColorBodyText   = lipgloss.Color("#c8cad8")
	ColorBrightText = lipgloss.Color("#ecedf5")
)

// Gradient stops for progress bar (5-color)
var ProgressGradient = []string{
	"#86bada", // C2 - start
	"#9f99d1", // C1
	"#dbaad7", // C3
	"#f6bcb0", // C4
	"#ffe3b3", // C5 - end
}

// Text gradient colors (midpoints)
var (
	TextGradient1 = lipgloss.Color("#93aad5") // G1: Lavender -> Blue
	TextGradient2 = lipgloss.Color("#e9b3c4") // G2: Mauve -> Peach
	TextGradient3 = lipgloss.Color("#cfbee2") // G3: Gold -> Lavender
)

// LerpColor interpolates between two hex colors.
func LerpColor(from, to string, t float64) string {
	r1, g1, b1 := HexToRGB(from)
	r2, g2, b2 := HexToRGB(to)

	r := uint8(float64(r1) + t*(float64(r2)-float64(r1)))
	g := uint8(float64(g1) + t*(float64(g2)-float64(g1)))
	b := uint8(float64(b1) + t*(float64(b2)-float64(b1)))

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func HexToRGB(hex string) (uint8, uint8, uint8) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// GradientText applies a gradient color across a string.
func GradientText(text, fromHex, toHex string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(text) * 20) // pre-allocate for ANSI escape overhead
	style := lipgloss.NewStyle()
	for i, r := range runes {
		t := float64(i) / float64(max(len(runes)-1, 1))
		color := LerpColor(fromHex, toHex, t)
		sb.WriteString(style.Foreground(lipgloss.Color(color)).Render(string(r)))
	}
	return sb.String()
}

// AnimatedGradientText applies a narrow sliding gradient across text.
// Shows 2 colors at a time, up to 3 during transitions between color pairs.
// tick is incremented by the UI animation timer (100ms per tick).
// Optional bg parameter sets a background color on each character.
func AnimatedGradientText(text string, tick uint, bg ...lipgloss.Color) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	stops := ProgressGradient // SkyBlue → Lavender → Mauve → Peach → Gold
	n := float64(len(stops))

	// Narrow window: text spans ~1.5 color segments → 2 colors, up to 3 during transitions
	windowSize := 1.5 / n

	// Phase: full cycle in 10 seconds (100 ticks × 100ms)
	phase := float64(tick) * 0.01
	phase = phase - math.Floor(phase)

	var sb strings.Builder
	sb.Grow(len(text) * 20)
	baseStyle := lipgloss.NewStyle()
	if len(bg) > 0 {
		baseStyle = baseStyle.Background(bg[0])
	}
	for i, r := range runes {
		charT := float64(i) / float64(max(len(runes)-1, 1))
		t := phase + charT*windowSize
		t = t - math.Floor(t) // wrap to [0, 1)
		color := multiStopGradientWrap(t, stops)
		sb.WriteString(baseStyle.Foreground(lipgloss.Color(color)).Render(string(r)))
	}
	return sb.String()
}

// multiStopGradientWrap interpolates through stops with wrapping (last → first).
func multiStopGradientWrap(t float64, stops []string) string {
	t = t - math.Floor(t)
	n := len(stops)
	pos := t * float64(n)
	idx := int(pos)
	if idx >= n {
		idx = 0
	}
	localT := pos - math.Floor(pos)
	next := (idx + 1) % n
	return LerpColor(stops[idx], stops[next], localT)
}

// MultiStopGradient interpolates through multiple color stops.
func MultiStopGradient(t float64, stops []string) string {
	if len(stops) < 2 {
		return stops[0]
	}
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}

	segments := len(stops) - 1
	segment := int(t * float64(segments))
	if segment >= segments {
		segment = segments - 1
	}
	localT := t*float64(segments) - float64(segment)

	return LerpColor(stops[segment], stops[segment+1], localT)
}

// Semantic colors extracted from UI files
var (
	ColorOverlayBg    = lipgloss.Color("#111122") // overlay dimmed background
	ColorWeekendRed   = lipgloss.Color("#f07070") // weekend/holiday text (WCAG AA ~4.5:1 on #232438)
	ColorWeekendFaded = lipgloss.Color("#4f5060") // faded weekend header
	ColorGaugeDim     = "#2a2b42"                 // dim arc in semicircle gauge (raw hex for Braille canvas)
	ColorPieChartBg   = "#373855"                 // pie chart background (raw hex for Braille canvas)
)

// Common styles
var (
	CardStyle = lipgloss.NewStyle().
			Background(ColorCardBg).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorBrightText).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMutedText)

	BodyStyle = lipgloss.NewStyle().
			Foreground(ColorBodyText)

	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorMauve)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorPeach).
			Bold(true)
)
