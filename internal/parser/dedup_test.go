package parser

import (
	"testing"
	"time"

	"github.com/anomredux/claude-smi/internal/domain"
)

func TestDedup(t *testing.T) {
	now := time.Now()
	entries := []domain.UsageEntry{
		{Timestamp: now.Add(1 * time.Minute), MessageID: "msg_1", RequestID: "req_1", InputTokens: 100},
		{Timestamp: now, MessageID: "msg_1", RequestID: "req_1", InputTokens: 50},                      // duplicate, earlier
		{Timestamp: now.Add(2 * time.Minute), MessageID: "msg_2", RequestID: "req_2", InputTokens: 200},
		{Timestamp: now.Add(3 * time.Minute), MessageID: "", RequestID: "", InputTokens: 10},            // no key, kept
	}

	result := Dedup(entries)

	if len(result) != 3 {
		t.Fatalf("got %d entries, want 3", len(result))
	}
	// First should be the earlier duplicate (now), kept as first occurrence
	if result[0].InputTokens != 50 {
		t.Errorf("first entry InputTokens = %d, want 50 (earlier duplicate)", result[0].InputTokens)
	}
	if result[1].InputTokens != 200 {
		t.Errorf("second entry InputTokens = %d, want 200", result[1].InputTokens)
	}
	if result[2].InputTokens != 10 {
		t.Errorf("third entry InputTokens = %d, want 10 (no key)", result[2].InputTokens)
	}
}

func TestDedup_Empty(t *testing.T) {
	result := Dedup(nil)
	if len(result) != 0 {
		t.Errorf("got %d entries, want 0", len(result))
	}
}
