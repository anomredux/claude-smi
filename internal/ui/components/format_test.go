package components

import (
	"testing"
	"time"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{100, "100"},
		{10000, "10,000"},
		{-1, "-1"},
		{-999, "-999"},
		{-1000, "-1,000"},
		{-1234567, "-1,234,567"},
	}
	for _, tt := range tests {
		got := FormatNumber(tt.input)
		if got != tt.want {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{30 * time.Minute, "30m"},
		{90 * time.Minute, "1h 30m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
		{5 * time.Minute, "5m"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.input)
		if got != tt.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
