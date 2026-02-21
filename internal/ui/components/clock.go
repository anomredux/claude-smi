package components

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/theme"
)

// BigDigits maps each digit (0-9) and colon to 5-line block art.
// Each digit is 5 chars wide, colon is 3 chars wide.
// Uses full blocks (█) and half-blocks (▀▄) for bold, clean look.
var BigDigits = map[rune][]string{
	'0': {"█▀▀▀█", "█   █", "█   █", "█   █", "█▄▄▄█"},
	'1': {"  ▄█ ", "   █ ", "   █ ", "   █ ", "  ▄█▄"},
	'2': {"▀▀▀▀█", "    █", "█▀▀▀▀", "█    ", "█▄▄▄▄"},
	'3': {"▀▀▀▀█", "    █", " ▀▀▀█", "    █", "▄▄▄▄█"},
	'4': {"█   █", "█   █", "▀▀▀▀█", "    █", "    █"},
	'5': {"█▀▀▀▀", "█    ", "▀▀▀▀█", "    █", "▄▄▄▄█"},
	'6': {"█▀▀▀▀", "█    ", "█▀▀▀█", "█   █", "█▄▄▄█"},
	'7': {"▀▀▀▀█", "    █", "   █ ", "  █  ", "  █  "},
	'8': {"█▀▀▀█", "█   █", "█▀▀▀█", "█   █", "█▄▄▄█"},
	'9': {"█▀▀▀█", "█   █", "▀▀▀▀█", "    █", "▄▄▄▄█"},
	':': {"   ", " █ ", "   ", " █ ", "   "},
}

// blankColon is the invisible colon (same width, all spaces).
var blankColon = []string{"   ", "   ", "   ", "   ", "   "}

// SessionClock renders a large digital clock with session info.
type SessionClock struct {
	Remaining    time.Duration
	SessionStart string // formatted time e.g. "13:00"
	SessionEnd   string // formatted time e.g. "17:59"
	Timezone     string // e.g. "KST"
	Width        int
}

// Render returns the clock as a string block.
func (c SessionClock) Render() string {
	remaining := c.Remaining
	if remaining < 0 {
		remaining = 0
	}

	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60
	seconds := int(remaining.Seconds()) % 60

	// Colon blink synced with countdown: visible first 500ms of each second
	fracSec := remaining.Seconds() - math.Floor(remaining.Seconds())
	colonOn := fracSec >= 0.5

	timeStr := fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)

	// Gradient: SkyBlue → Lavender → Mauve (matches theme progression)
	hourStyle := lipgloss.NewStyle().Foreground(theme.ColorSkyBlue).Bold(true)
	colonStyle := lipgloss.NewStyle().Foreground(theme.ColorPeach).Bold(true)
	minuteStyle := lipgloss.NewStyle().Foreground(theme.ColorLavender).Bold(true)
	secondStyle := lipgloss.NewStyle().Foreground(theme.ColorMauve)

	// Find colon positions for zone-based coloring
	var colonPositions []int
	for i, ch := range timeStr {
		if ch == ':' {
			colonPositions = append(colonPositions, i)
		}
	}

	digitLines := make([]string, 5)
	charIdx := 0
	for _, ch := range timeStr {
		if ch == ':' {
			// Blink: show colon or blank (synced with seconds countdown)
			glyph := BigDigits[':']
			if !colonOn {
				glyph = blankColon
			}
			for row := 0; row < 5; row++ {
				if len(digitLines[row]) > 0 {
					digitLines[row] += " "
				}
				digitLines[row] += colonStyle.Render(glyph[row])
			}
			charIdx++
			continue
		}

		digit, ok := BigDigits[ch]
		if !ok {
			charIdx++
			continue
		}

		// Pick style based on position relative to colons
		var style lipgloss.Style
		if len(colonPositions) >= 2 && charIdx > colonPositions[1] {
			style = secondStyle
		} else if len(colonPositions) >= 1 && charIdx > colonPositions[0] {
			style = minuteStyle
		} else {
			style = hourStyle
		}

		for row := 0; row < 5; row++ {
			if len(digitLines[row]) > 0 {
				digitLines[row] += " "
			}
			digitLines[row] += style.Render(digit[row])
		}
		charIdx++
	}

	var lines []string
	for _, dl := range digitLines {
		lines = append(lines, CenterText(dl, c.Width))
	}

	// blank line + "until reset" label
	lines = append(lines, "")
	untilLabel := lipgloss.NewStyle().Foreground(theme.ColorPeach).Render(i18n.T("until_reset"))
	lines = append(lines, CenterText(untilLabel, c.Width))

	// Session time range with timezone (no extra blank line)
	tzSuffix := ""
	if c.Timezone != "" {
		tzSuffix = " " + c.Timezone
	}
	sessionInfo := theme.BodyStyle.Render(
		fmt.Sprintf("%s → %s%s", c.SessionStart, c.SessionEnd, tzSuffix))
	lines = append(lines, CenterText(sessionInfo, c.Width))

	return strings.Join(lines, "\n")
}
