package ui

import (
	"testing"

	"github.com/anomredux/claude-smi/internal/theme"
)

func TestLerpColor(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		t    float64
		want string
	}{
		{"start", "#000000", "#ffffff", 0.0, "#000000"},
		{"end", "#000000", "#ffffff", 1.0, "#ffffff"},
		{"midpoint", "#000000", "#ffffff", 0.5, "#7f7f7f"},
		{"same color", "#ff0000", "#ff0000", 0.5, "#ff0000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := theme.LerpColor(tt.from, tt.to, tt.t)
			if got != tt.want {
				t.Errorf("LerpColor(%s, %s, %f) = %s, want %s", tt.from, tt.to, tt.t, got, tt.want)
			}
		})
	}
}

func TestHexToRGB(t *testing.T) {
	r, g, b := theme.HexToRGB("#ff8040")
	if r != 0xff || g != 0x80 || b != 0x40 {
		t.Errorf("got (%d, %d, %d), want (255, 128, 64)", r, g, b)
	}

	r, g, b = theme.HexToRGB("ff8040")
	if r != 0xff || g != 0x80 || b != 0x40 {
		t.Errorf("without hash: got (%d, %d, %d), want (255, 128, 64)", r, g, b)
	}
}

func TestGradientText(t *testing.T) {
	result := theme.GradientText("Hello", "#000000", "#ffffff")
	if result == "" {
		t.Error("GradientText returned empty string")
	}

	result = theme.GradientText("", "#000000", "#ffffff")
	if result != "" {
		t.Error("GradientText should return empty for empty input")
	}
}
