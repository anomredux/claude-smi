package components

import "github.com/anomredux/claude-smi/internal/theme"

// HelpFooter renders muted help text with standard indentation.
func HelpFooter(text string) string {
	return theme.MutedStyle.Render("  " + text)
}
