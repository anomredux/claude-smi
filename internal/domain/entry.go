package domain

import "time"

type UsageEntry struct {
	Timestamp           time.Time
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	CostUSD             float64
	Model               string
	MessageID           string
	RequestID           string
	SessionID           string
	ProjectPath         string // derived from file path
}

// TotalTokens returns input + output + cache tokens for limit comparison.
func (e UsageEntry) TotalTokens() int {
	return e.InputTokens + e.OutputTokens + e.CacheCreationTokens + e.CacheReadTokens
}

// DedupKey returns the unique key for deduplication.
func (e UsageEntry) DedupKey() string {
	return e.MessageID + ":" + e.RequestID
}
