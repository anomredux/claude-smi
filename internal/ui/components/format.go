package components

import (
	"fmt"
	"time"
)

// FormatNumber formats an integer with comma separators (e.g. 1,234,567).
func FormatNumber(n int) string {
	negative := n < 0
	if negative {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		if negative {
			return "-" + s
		}
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	out := string(result)
	if negative {
		return "-" + out
	}
	return out
}

// FormatCompact formats a number with K/M suffix (e.g. 12345 → "12.3K").
func FormatCompact(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}

// FormatDuration formats a duration as "Xh Ym" or "Xm".
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
