package api

import (
	"testing"
	"time"
)

func TestParseCredentialJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid credentials",
			input: `{"claudeAiOauth":{"accessToken":"test-token-123"}}`,
			want:  "test-token-123",
		},
		{
			name:    "empty access token",
			input:   `{"claudeAiOauth":{"accessToken":""}}`,
			wantErr: true,
		},
		{
			name:    "missing claudeAiOauth key",
			input:   `{"other":"data"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCredentialJSON(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q; want %q", got, tt.want)
			}
		})
	}
}

func TestUsageData_SessionStart(t *testing.T) {
	data := UsageData{
		FiveHour: WindowData{
			Utilization: 50.0,
			ResetsAt:    "2025-01-15T17:00:00Z",
		},
	}

	start, err := data.SessionStart()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("got %v; want %v", start, expected)
	}
}

func TestUsageData_SessionEnd(t *testing.T) {
	data := UsageData{
		FiveHour: WindowData{
			ResetsAt: "2025-01-15T17:00:00Z",
		},
	}

	end, err := data.SessionEnd()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := time.Date(2025, 1, 15, 17, 0, 0, 0, time.UTC)
	if !end.Equal(expected) {
		t.Errorf("got %v; want %v", end, expected)
	}
}

func TestUsageData_SessionRemaining(t *testing.T) {
	// Set reset time in the past
	past := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	data := UsageData{
		FiveHour: WindowData{ResetsAt: past},
	}

	remaining, err := data.SessionRemaining()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expected 0 for past reset; got %v", remaining)
	}

	// Set reset time in the future
	future := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	data.FiveHour.ResetsAt = future

	remaining, err = data.SessionRemaining()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining <= 0 {
		t.Errorf("expected positive remaining; got %v", remaining)
	}
}

func TestUsageData_InvalidResetsAt(t *testing.T) {
	data := UsageData{
		FiveHour: WindowData{ResetsAt: "not-a-date"},
	}

	_, err := data.SessionStart()
	if err == nil {
		t.Error("expected error for invalid ResetsAt")
	}

	_, err = data.SessionEnd()
	if err == nil {
		t.Error("expected error for invalid ResetsAt")
	}

	_, err = data.SessionRemaining()
	if err == nil {
		t.Error("expected error for invalid ResetsAt")
	}
}

func TestWindowData_ResetTime(t *testing.T) {
	w := WindowData{ResetsAt: "2025-06-15T10:30:00Z"}
	got, err := w.ResetTime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("got %v; want %v", got, expected)
	}
}
