package components

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/theme"
)

// brailleDots maps [row][col] to braille dot bit positions.
// Each braille character is a 2-wide × 4-tall pixel grid.
var brailleDots = [4][2]int{
	{0x01, 0x08}, // row 0
	{0x02, 0x10}, // row 1
	{0x04, 0x20}, // row 2
	{0x40, 0x80}, // row 3
}

// BrailleCanvas is a pixel grid that renders to braille characters.
type BrailleCanvas struct {
	Width  int // character width
	Height int // character height
	pixels [][]BraillePixel
}

// BraillePixel holds the state of a single sub-character dot.
type BraillePixel struct {
	On       bool
	ColorIdx int // -1 = dim/unfilled, 0+ = palette index
}

// NewBrailleCanvas creates a canvas of the given character dimensions.
func NewBrailleCanvas(charW, charH int) *BrailleCanvas {
	pxW, pxH := charW*2, charH*4
	pixels := make([][]BraillePixel, pxH)
	for y := range pixels {
		pixels[y] = make([]BraillePixel, pxW)
		for x := range pixels[y] {
			pixels[y][x].ColorIdx = -1
		}
	}
	return &BrailleCanvas{Width: charW, Height: charH, pixels: pixels}
}

// PixelWidth returns the horizontal pixel resolution.
func (c *BrailleCanvas) PixelWidth() int { return c.Width * 2 }

// PixelHeight returns the vertical pixel resolution.
func (c *BrailleCanvas) PixelHeight() int { return c.Height * 4 }

// Set turns on a pixel with the given color index.
func (c *BrailleCanvas) Set(x, y, colorIdx int) {
	if x >= 0 && x < c.PixelWidth() && y >= 0 && y < c.PixelHeight() {
		c.pixels[y][x] = BraillePixel{On: true, ColorIdx: colorIdx}
	}
}

// Render converts the pixel grid to styled braille strings.
// palette maps color index → hex color string.
// dimColor is used for unfilled (colorIdx == -1) pixels.
func (c *BrailleCanvas) Render(palette []string, dimColor string) []string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(dimColor))
	styles := make([]lipgloss.Style, len(palette))
	for i, hex := range palette {
		styles[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
	}

	var lines []string
	for cr := 0; cr < c.Height; cr++ {
		var sb strings.Builder
		for cc := 0; cc < c.Width; cc++ {
			code := 0x2800
			// Count which color is dominant in this character cell
			counts := make(map[int]int)
			total := 0
			for dr := 0; dr < 4; dr++ {
				for dc := 0; dc < 2; dc++ {
					px, py := cc*2+dc, cr*4+dr
					if px < c.PixelWidth() && py < c.PixelHeight() && c.pixels[py][px].On {
						code |= brailleDots[dr][dc]
						counts[c.pixels[py][px].ColorIdx]++
						total++
					}
				}
			}
			if code == 0x2800 {
				sb.WriteString(" ")
				continue
			}
			ch := string(rune(code))
			// Find dominant color
			bestIdx, bestCnt := -1, 0
			for idx, cnt := range counts {
				if cnt > bestCnt {
					bestIdx = idx
					bestCnt = cnt
				}
			}
			if bestIdx >= 0 && bestIdx < len(styles) {
				sb.WriteString(styles[bestIdx].Render(ch))
			} else {
				sb.WriteString(dimStyle.Render(ch))
			}
		}
		lines = append(lines, sb.String())
	}
	return lines
}

// RenderGradient converts the pixel grid to styled braille strings using a gradient.
// angleFn maps (x, y) → normalized position [0,1] for gradient color lookup.
// filled pixels with colorIdx >= 0 get gradient color, colorIdx == -1 get dimColor.
func (c *BrailleCanvas) RenderGradient(dimColor string) []string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(dimColor))
	style := lipgloss.NewStyle()

	var lines []string
	for cr := 0; cr < c.Height; cr++ {
		var sb strings.Builder
		for cc := 0; cc < c.Width; cc++ {
			code := 0x2800
			filledCount := 0
			dimCount := 0
			var sumAngle float64

			for dr := 0; dr < 4; dr++ {
				for dc := 0; dc < 2; dc++ {
					px, py := cc*2+dc, cr*4+dr
					if px < c.PixelWidth() && py < c.PixelHeight() && c.pixels[py][px].On {
						code |= brailleDots[dr][dc]
						if c.pixels[py][px].ColorIdx >= 0 {
							filledCount++
							// Store angle as float from ColorIdx (encoded as angle*1000)
							sumAngle += float64(c.pixels[py][px].ColorIdx) / 1000.0
						} else {
							dimCount++
						}
					}
				}
			}

			if code == 0x2800 {
				sb.WriteString(" ")
				continue
			}

			ch := string(rune(code))
			if filledCount > dimCount && filledCount > 0 {
				t := sumAngle / float64(filledCount)
				if t > 1 {
					t = 1
				}
				hex := theme.MultiStopGradient(t, theme.ProgressGradient)
				sb.WriteString(style.Foreground(lipgloss.Color(hex)).Render(ch))
			} else {
				sb.WriteString(dimStyle.Render(ch))
			}
		}
		lines = append(lines, sb.String())
	}
	return lines
}

// ── Palette for pie chart slices ──

// SlicePalette returns the 5-color palette for pie chart slices.
func SlicePalette() []string {
	return []string{
		string(theme.ColorSkyBlue),  // C2
		string(theme.ColorLavender), // C1
		string(theme.ColorMauve),    // C3
		string(theme.ColorPeach),    // C4
		string(theme.ColorGold),     // C5
	}
}

// ── Geometry helpers ──

// DrawRing sets pixels on a ring (donut) shape.
// cx, cy: center in pixel coords. outerR, innerR: radii.
// startAngle, endAngle: in radians (0 = top, clockwise).
// colorIdx: color to assign to filled pixels.
func (c *BrailleCanvas) DrawRing(cx, cy, outerR, innerR, startAngle, endAngle float64, colorIdx int) {
	for y := 0; y < c.PixelHeight(); y++ {
		for x := 0; x < c.PixelWidth(); x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			// Angle from top, clockwise
			angle := math.Atan2(-dx, -dy)
			if angle < 0 {
				angle += 2 * math.Pi
			}
			if angle >= startAngle && angle <= endAngle {
				c.Set(x, y, colorIdx)
			}
		}
	}
}

// DrawSemicircle sets pixels on a semicircle arc (top half).
// cx, cy: center (bottom-center of the semicircle).
// outerR, innerR: radii.
// fillFraction: 0..1 how much is filled (left to right).
// Uses gradient encoding: colorIdx = int(normalizedAngle * 1000).
func (c *BrailleCanvas) DrawSemicircle(cx, cy, outerR, innerR, fillFraction float64) {
	fillAngle := -math.Pi + fillFraction*math.Pi
	if fillAngle > 0 {
		fillAngle = 0
	}

	for y := 0; y < c.PixelHeight(); y++ {
		for x := 0; x < c.PixelWidth(); x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR || dy > 1 {
				continue
			}
			angle := math.Atan2(dy, dx) // -π (left) → 0 (right)
			if angle <= fillAngle {
				// Filled: encode normalized position as colorIdx
				t := (angle + math.Pi) / math.Pi // 0..1
				c.Set(x, y, int(t*1000))
			} else {
				// Unfilled
				c.Set(x, y, -1)
			}
		}
	}
}

// ── Text helpers ──

// VisualWidth returns the visible character count, ignoring ANSI escape codes.
func VisualWidth(s string) int {
	n := 0
	esc := false
	for _, r := range s {
		if r == '\033' {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		n++
	}
	return n
}

// PadRight pads a string to the given visual width.
func PadRight(s string, width int) string {
	gap := width - VisualWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// PadLeft pads a string on the left to the given visual width.
func PadLeft(s string, width int) string {
	gap := width - VisualWidth(s)
	if gap <= 0 {
		return s
	}
	return strings.Repeat(" ", gap) + s
}

// CenterText centers a string within the given visual width.
func CenterText(s string, width int) string {
	gap := width - VisualWidth(s)
	if gap <= 0 {
		return s
	}
	left := gap / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", gap-left)
}

// CenterBlock centers a multi-line block within the given width.
// Unlike CenterText (per-line), this shifts the entire block uniformly
// so internal column alignment is preserved.
func CenterBlock(content string, width int) string {
	lines := strings.Split(content, "\n")
	maxW := 0
	for _, line := range lines {
		if vw := VisualWidth(line); vw > maxW {
			maxW = vw
		}
	}
	pad := (width - maxW) / 2
	if pad <= 0 {
		return content
	}
	prefix := strings.Repeat(" ", pad)
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// JoinHorizontal joins multiple blocks of lines side by side with a gap.
func JoinHorizontal(blocks [][]string, gap int) []string {
	maxH := 0
	for _, b := range blocks {
		if len(b) > maxH {
			maxH = len(b)
		}
	}
	// Calculate visual widths per block
	widths := make([]int, len(blocks))
	for i, b := range blocks {
		for _, line := range b {
			if vl := VisualWidth(line); vl > widths[i] {
				widths[i] = vl
			}
		}
	}
	spacer := strings.Repeat(" ", gap)
	var result []string
	for row := 0; row < maxH; row++ {
		var sb strings.Builder
		for i, b := range blocks {
			if i > 0 {
				sb.WriteString(spacer)
			}
			if row < len(b) {
				sb.WriteString(PadRight(b[row], widths[i]))
			} else {
				sb.WriteString(strings.Repeat(" ", widths[i]))
			}
		}
		result = append(result, sb.String())
	}
	return result
}

// ColoredSquare returns a colored "■" for legend entries using lipgloss.
func ColoredSquare(hex string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("■")
}
