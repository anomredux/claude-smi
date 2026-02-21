package domain

import "testing"

func TestUsageEntry_TotalTokens(t *testing.T) {
	tests := []struct {
		name  string
		entry UsageEntry
		want  int
	}{
		{
			name: "all token types",
			entry: UsageEntry{
				InputTokens:         100,
				OutputTokens:        50,
				CacheCreationTokens: 200,
				CacheReadTokens:     30,
			},
			want: 380,
		},
		{
			name:  "zero value",
			entry: UsageEntry{},
			want:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.TotalTokens(); got != tt.want {
				t.Errorf("TotalTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestUsageEntry_DedupKey(t *testing.T) {
	e := UsageEntry{MessageID: "msg_abc", RequestID: "req_123"}
	want := "msg_abc:req_123"
	if got := e.DedupKey(); got != want {
		t.Errorf("DedupKey() = %q, want %q", got, want)
	}
}
