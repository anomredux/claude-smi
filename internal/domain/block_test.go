package domain

import (
	"testing"
	"time"
)

func TestBuildBlocks(t *testing.T) {
	base := time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)

	entries := []UsageEntry{
		{Timestamp: base, InputTokens: 100, OutputTokens: 50, Model: "opus", MessageID: "m1", RequestID: "r1"},
		{Timestamp: base.Add(1 * time.Hour), InputTokens: 200, OutputTokens: 100, Model: "opus", MessageID: "m2", RequestID: "r2"},
		{Timestamp: base.Add(3 * time.Hour), InputTokens: 50, OutputTokens: 25, Model: "haiku", MessageID: "m3", RequestID: "r3"},
		// 6 hours later -- new block
		{Timestamp: base.Add(6 * time.Hour), InputTokens: 300, OutputTokens: 150, Model: "opus", MessageID: "m4", RequestID: "r4"},
	}

	blocks := BuildBlocks(entries)

	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}

	// Block 1: 3 entries
	if blocks[0].MessageCount != 3 {
		t.Errorf("block[0] MessageCount = %d, want 3", blocks[0].MessageCount)
	}
	if blocks[0].TotalTokens != 525 { // 150 + 300 + 75
		t.Errorf("block[0] TotalTokens = %d, want 525", blocks[0].TotalTokens)
	}
	if blocks[0].InputTokens != 350 { // 100 + 200 + 50
		t.Errorf("block[0] InputTokens = %d, want 350", blocks[0].InputTokens)
	}
	if blocks[0].OutputTokens != 175 { // 50 + 100 + 25
		t.Errorf("block[0] OutputTokens = %d, want 175", blocks[0].OutputTokens)
	}

	// Block 2: 1 entry
	if blocks[1].MessageCount != 1 {
		t.Errorf("block[1] MessageCount = %d, want 1", blocks[1].MessageCount)
	}
	if blocks[1].InputTokens != 300 {
		t.Errorf("block[1] InputTokens = %d, want 300", blocks[1].InputTokens)
	}

	// Model breakdown
	if len(blocks[0].Models) != 2 {
		t.Errorf("block[0] has %d models, want 2", len(blocks[0].Models))
	}
}

func TestBuildBlocks_Empty(t *testing.T) {
	blocks := BuildBlocks(nil)
	if blocks != nil {
		t.Errorf("got %v, want nil", blocks)
	}
}

func TestBuildBlocks_ModelPercentages(t *testing.T) {
	base := time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)

	entries := []UsageEntry{
		{Timestamp: base, InputTokens: 75, OutputTokens: 0, Model: "opus"},
		{Timestamp: base.Add(1 * time.Hour), InputTokens: 25, OutputTokens: 0, Model: "haiku"},
	}

	blocks := BuildBlocks(entries)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}

	opus := blocks[0].Models["opus"]
	if opus.Percentage != 75.0 {
		t.Errorf("opus percentage = %f, want 75.0", opus.Percentage)
	}
	haiku := blocks[0].Models["haiku"]
	if haiku.Percentage != 25.0 {
		t.Errorf("haiku percentage = %f, want 25.0", haiku.Percentage)
	}
}
