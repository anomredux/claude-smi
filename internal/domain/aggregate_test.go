package domain

import (
	"testing"
	"time"
)

func TestAggregateDaily(t *testing.T) {
	seoul, _ := time.LoadLocation("Asia/Seoul")
	utc := time.UTC

	// 2026-02-21 23:30 UTC = 2026-02-22 08:30 KST
	entries := []UsageEntry{
		{Timestamp: time.Date(2026, 2, 21, 23, 30, 0, 0, utc), InputTokens: 100, CostUSD: 1.0},
		{Timestamp: time.Date(2026, 2, 21, 10, 0, 0, 0, utc), InputTokens: 200, CostUSD: 2.0},
	}

	t.Run("UTC grouping", func(t *testing.T) {
		result := AggregateDaily(entries, utc)
		if len(result) != 1 {
			t.Fatalf("UTC: got %d days, want 1", len(result))
		}
		if result[0].Date != "2026-02-21" {
			t.Errorf("date = %s, want 2026-02-21", result[0].Date)
		}
		if result[0].InputTokens != 300 {
			t.Errorf("InputTokens = %d, want 300", result[0].InputTokens)
		}
	})

	t.Run("Seoul timezone splits into 2 days", func(t *testing.T) {
		result := AggregateDaily(entries, seoul)
		if len(result) != 2 {
			t.Fatalf("Seoul: got %d days, want 2", len(result))
		}
	})
}

func TestAggregateMonthly(t *testing.T) {
	utc := time.UTC
	entries := []UsageEntry{
		{Timestamp: time.Date(2026, 2, 1, 10, 0, 0, 0, utc), InputTokens: 100, CostUSD: 1.0},
		{Timestamp: time.Date(2026, 2, 15, 10, 0, 0, 0, utc), InputTokens: 200, CostUSD: 2.0},
		{Timestamp: time.Date(2026, 2, 15, 14, 0, 0, 0, utc), InputTokens: 50, CostUSD: 0.5},
		// Different month -- should be excluded
		{Timestamp: time.Date(2026, 3, 1, 10, 0, 0, 0, utc), InputTokens: 999, CostUSD: 99.0},
	}

	agg := AggregateMonthly(entries, utc, 2026, time.February)

	if agg.Month != "2026-02" {
		t.Errorf("Month = %s, want 2026-02", agg.Month)
	}
	if agg.TotalCalls != 3 {
		t.Errorf("TotalCalls = %d, want 3", agg.TotalCalls)
	}
	if len(agg.Days) != 2 {
		t.Errorf("Days count = %d, want 2", len(agg.Days))
	}
	day15 := agg.Days[15]
	if day15.EntriesCount != 2 {
		t.Errorf("day 15 entries = %d, want 2", day15.EntriesCount)
	}
}

func TestAggregateDaily_Empty(t *testing.T) {
	result := AggregateDaily(nil, time.UTC)
	if len(result) != 0 {
		t.Errorf("got %d days, want 0", len(result))
	}
}

func TestAggregateMonthly_Empty(t *testing.T) {
	agg := AggregateMonthly(nil, time.UTC, 2026, time.February)
	if agg.TotalCalls != 0 {
		t.Errorf("TotalCalls = %d, want 0", agg.TotalCalls)
	}
	if len(agg.Days) != 0 {
		t.Errorf("Days count = %d, want 0", len(agg.Days))
	}
}

func TestDailyAggregate_TotalTokens(t *testing.T) {
	d := DailyAggregate{
		InputTokens:         100,
		OutputTokens:        50,
		CacheCreationTokens: 25,
		CacheReadTokens:     10,
	}
	if got := d.TotalTokens(); got != 185 {
		t.Errorf("TotalTokens() = %d, want 185", got)
	}
}
