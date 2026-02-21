package pricing

import (
	"math"
	"testing"

	"github.com/anomredux/claude-smi/internal/domain"
)

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestCalculator_Auto(t *testing.T) {
	table := PricingTable{
		"claude-opus-4-6": {Input: 5.0, Output: 25.0, CacheCreation: 6.25, CacheRead: 0.50},
	}
	calc := NewCalculator(table, CostModeAuto)

	t.Run("uses costUSD when present", func(t *testing.T) {
		e := &domain.UsageEntry{CostUSD: 1.23, Model: "claude-opus-4-6", InputTokens: 1000}
		if got := calc.Calculate(e); got != 1.23 {
			t.Errorf("got %f, want 1.23", got)
		}
	})

	t.Run("calculates from tokens when costUSD is 0", func(t *testing.T) {
		e := &domain.UsageEntry{
			Model:        "claude-opus-4-6",
			InputTokens:  1000,
			OutputTokens: 500,
		}
		// 1000 * 5/1M + 500 * 25/1M = 0.005 + 0.0125 = 0.0175
		got := calc.Calculate(e)
		if !almostEqual(got, 0.0175, 0.0001) {
			t.Errorf("got %f, want ~0.0175", got)
		}
	})
}

func TestCalculator_Display(t *testing.T) {
	table := PricingTable{
		"claude-opus-4-6": {Input: 5.0, Output: 25.0},
	}
	calc := NewCalculator(table, CostModeDisplay)

	e := &domain.UsageEntry{CostUSD: 0, Model: "claude-opus-4-6", InputTokens: 1000}
	if got := calc.Calculate(e); got != 0 {
		t.Errorf("display mode should return costUSD as-is, got %f", got)
	}
}

func TestCalculator_Calculate(t *testing.T) {
	table := PricingTable{
		"claude-opus-4-6": {Input: 5.0, Output: 25.0},
	}
	calc := NewCalculator(table, CostModeCalculate)

	e := &domain.UsageEntry{CostUSD: 999, Model: "claude-opus-4-6", InputTokens: 1000}
	got := calc.Calculate(e)
	// 1000 * 5/1M = 0.005
	if !almostEqual(got, 0.005, 0.0001) {
		t.Errorf("calculate mode should ignore costUSD, got %f, want ~0.005", got)
	}
}

func TestCalculator_ApplyAll(t *testing.T) {
	table := PricingTable{
		"claude-opus-4-6": {Input: 5.0, Output: 25.0},
	}
	calc := NewCalculator(table, CostModeAuto)

	entries := []domain.UsageEntry{
		{Model: "claude-opus-4-6", InputTokens: 1000, OutputTokens: 500},
		{Model: "claude-opus-4-6", InputTokens: 2000, OutputTokens: 1000},
	}
	calc.ApplyAll(entries)

	if entries[0].CostUSD == 0 {
		t.Errorf("ApplyAll should set CostUSD on first entry")
	}
	if entries[1].CostUSD == 0 {
		t.Errorf("ApplyAll should set CostUSD on second entry")
	}
}

func TestPricingTable_Lookup(t *testing.T) {
	table := PricingTable{
		"claude-opus-4-6": {Input: 5.0},
	}

	t.Run("exact match", func(t *testing.T) {
		p, ok := table.Lookup("claude-opus-4-6")
		if !ok || p.Input != 5.0 {
			t.Errorf("exact match failed")
		}
	})

	t.Run("prefix match", func(t *testing.T) {
		p, ok := table.Lookup("claude-opus-4-6-20260101")
		if !ok || p.Input != 5.0 {
			t.Errorf("prefix match failed")
		}
	})

	t.Run("no match", func(t *testing.T) {
		_, ok := table.Lookup("unknown-model")
		if ok {
			t.Errorf("should not match unknown model")
		}
	})
}

func TestPricingTable_Lookup_Deterministic(t *testing.T) {
	// When multiple keys could prefix-match, the longest prefix should win.
	table := PricingTable{
		"claude-opus":     {Input: 10.0},
		"claude-opus-4-6": {Input: 5.0},
	}
	p, ok := table.Lookup("claude-opus-4-6-20260101")
	if !ok {
		t.Fatal("expected match")
	}
	if p.Input != 5.0 {
		t.Errorf("should match longest prefix: got Input=%f, want 5.0", p.Input)
	}
}

func TestCalculator_UnknownModel(t *testing.T) {
	table := PricingTable{
		"claude-opus-4-6": {Input: 5.0, Output: 25.0},
	}
	calc := NewCalculator(table, CostModeCalculate)

	e := &domain.UsageEntry{Model: "totally-unknown", InputTokens: 1000}
	got := calc.Calculate(e)
	if got != 0 {
		t.Errorf("unknown model should return 0, got %f", got)
	}
}

func TestLoadDefault(t *testing.T) {
	table, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault failed: %v", err)
	}
	if len(table) < 3 {
		t.Errorf("expected at least 3 models, got %d", len(table))
	}
	opus, ok := table["claude-opus-4-6"]
	if !ok {
		t.Fatal("missing claude-opus-4-6")
	}
	if opus.Input != 5.0 {
		t.Errorf("opus input = %f, want 5.0", opus.Input)
	}
}
