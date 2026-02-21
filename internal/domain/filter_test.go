package domain

import (
	"testing"
	"time"
)

func TestFilterByTimeRange(t *testing.T) {
	tz := time.UTC
	base := time.Date(2026, 2, 15, 12, 0, 0, 0, tz)

	entries := []UsageEntry{
		{Timestamp: base.AddDate(0, 0, -5)}, // Feb 10
		{Timestamp: base.AddDate(0, 0, -3)}, // Feb 12
		{Timestamp: base},                    // Feb 15
		{Timestamp: base.AddDate(0, 0, 3)},  // Feb 18
		{Timestamp: base.AddDate(0, 0, 7)},  // Feb 22
	}

	tests := []struct {
		name    string
		since   string
		until   string
		want    int
		wantErr bool
	}{
		{"no filter", "", "", 5, false},
		{"since only", "2026-02-14", "", 3, false},
		{"until only", "", "2026-02-16", 3, false},
		{"both", "2026-02-12", "2026-02-18", 3, false},
		{"exact day", "2026-02-15", "2026-02-15", 1, false},
		{"no match", "2026-03-01", "2026-03-02", 0, false},
		{"invalid since", "not-a-date", "", 0, true},
		{"invalid until", "", "bad-date", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilterByTimeRange(entries, tt.since, tt.until, tz)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.want {
				t.Errorf("got %d entries, want %d", len(result), tt.want)
			}
		})
	}
}

func TestFilterByTimeRange_EmptyEntries(t *testing.T) {
	result, err := FilterByTimeRange(nil, "2026-01-01", "2026-12-31", time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d entries, want 0", len(result))
	}
}
