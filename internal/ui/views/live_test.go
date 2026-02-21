package views

import (
	"testing"
	"time"

	"github.com/anomredux/claude-smi/internal/api"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/pricing"
)

func TestSessionModelBreakdown(t *testing.T) {
	v := NewLiveView(time.UTC, nil)

	entries := []domain.UsageEntry{
		{Model: "claude-sonnet-4-6", InputTokens: 100, OutputTokens: 50},
		{Model: "claude-sonnet-4-6", InputTokens: 200, OutputTokens: 100},
		{Model: "claude-haiku-4-5", InputTokens: 50, OutputTokens: 25},
	}

	breakdown := v.sessionModelBreakdown(entries)

	if len(breakdown) != 2 {
		t.Fatalf("expected 2 models; got %d", len(breakdown))
	}

	sonnet := breakdown["claude-sonnet-4-6"]
	if sonnet.Tokens != 450 {
		t.Errorf("sonnet tokens: got %d; want 450", sonnet.Tokens)
	}

	haiku := breakdown["claude-haiku-4-5"]
	if haiku.Tokens != 75 {
		t.Errorf("haiku tokens: got %d; want 75", haiku.Tokens)
	}

	// Check percentages sum to ~100
	totalPct := sonnet.Percentage + haiku.Percentage
	if totalPct < 99.9 || totalPct > 100.1 {
		t.Errorf("percentages should sum to ~100; got %f", totalPct)
	}
}

func TestSessionModelBreakdownEmpty(t *testing.T) {
	v := NewLiveView(time.UTC, nil)
	breakdown := v.sessionModelBreakdown(nil)
	if len(breakdown) != 0 {
		t.Errorf("expected empty breakdown for nil entries; got %d", len(breakdown))
	}
}

func TestSessionEntries_WithApiUsage(t *testing.T) {
	v := NewLiveView(time.UTC, nil)

	now := time.Now()
	resetAt := now.Add(3 * time.Hour) // 3 hours from now
	sessionStart := resetAt.Add(-api.SessionWindow)

	v.apiUsage = &api.UsageData{
		FiveHour: api.WindowData{
			Utilization: 50,
			ResetsAt:    resetAt.Format(time.RFC3339),
		},
	}

	v.entries = []domain.UsageEntry{
		{Timestamp: sessionStart.Add(-1 * time.Hour)}, // before session
		{Timestamp: sessionStart.Add(1 * time.Hour)},  // in session
		{Timestamp: sessionStart.Add(2 * time.Hour)},  // in session
	}

	filtered := v.sessionEntries()
	if len(filtered) != 2 {
		t.Errorf("expected 2 session entries; got %d", len(filtered))
	}
}

func TestSessionEntries_NoApiUsage(t *testing.T) {
	v := NewLiveView(time.UTC, nil)
	v.entries = []domain.UsageEntry{
		{Timestamp: time.Now()},
	}

	// No apiUsage, no active block → should return nil
	filtered := v.sessionEntries()
	if filtered != nil {
		t.Errorf("expected nil without API usage or active block; got %d entries", len(filtered))
	}
}

func TestRecomputeBurn(t *testing.T) {
	table, _ := pricing.LoadDefault()
	if table == nil {
		table = make(pricing.PricingTable)
	}
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	v := NewLiveView(time.UTC, calc)

	now := time.Now()
	resetAt := now.Add(3 * time.Hour)

	v.apiUsage = &api.UsageData{
		FiveHour: api.WindowData{
			Utilization: 50,
			ResetsAt:    resetAt.Format(time.RFC3339),
		},
	}

	v.entries = []domain.UsageEntry{
		{
			Timestamp:    now.Add(-30 * time.Minute),
			InputTokens:  1000,
			OutputTokens: 500,
			Model:        "claude-sonnet-4-6",
			CostUSD:      0.01,
		},
		{
			Timestamp:    now.Add(-15 * time.Minute),
			InputTokens:  2000,
			OutputTokens: 1000,
			Model:        "claude-sonnet-4-6",
			CostUSD:      0.02,
		},
	}

	v.recomputeBurn()

	if !v.burn.hasData {
		t.Fatal("expected burn to have data")
	}
	if v.burn.inputTokens != 3000 {
		t.Errorf("input tokens: got %d; want 3000", v.burn.inputTokens)
	}
	if v.burn.outputTokens != 1500 {
		t.Errorf("output tokens: got %d; want 1500", v.burn.outputTokens)
	}
	if v.burn.totalCost != 0.03 {
		t.Errorf("total cost: got %f; want 0.03", v.burn.totalCost)
	}
	if v.burn.tokensPerMin <= 0 {
		t.Error("expected positive tokens per minute")
	}
	if v.burn.costPerHour <= 0 {
		t.Error("expected positive cost per hour")
	}
}

func TestRecomputeBurn_Empty(t *testing.T) {
	v := NewLiveView(time.UTC, nil)
	v.recomputeBurn()

	if v.burn.hasData {
		t.Error("expected no burn data for empty entries")
	}
}
