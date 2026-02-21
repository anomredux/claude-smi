package domain

import (
	"sort"
	"time"
)

type DailyAggregate struct {
	Date                string // "2006-01-02"
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalCost           float64
	EntriesCount        int
}

// TotalTokens returns the sum of all token types for this day.
func (d DailyAggregate) TotalTokens() int {
	return d.InputTokens + d.OutputTokens + d.CacheCreationTokens + d.CacheReadTokens
}

type MonthlyAggregate struct {
	Month                   string                 // "2006-01"
	Days                    map[int]DailyAggregate // day number -> aggregate
	TotalCost               float64
	TotalTokens             int
	TotalInputTokens        int
	TotalOutputTokens       int
	TotalCacheCreation      int
	TotalCacheRead          int
	TotalCalls              int
}

// AggregateDaily groups entries by date in the given timezone.
func AggregateDaily(entries []UsageEntry, tz *time.Location) []DailyAggregate {
	groups := make(map[string]*DailyAggregate)

	for _, e := range entries {
		key := e.Timestamp.In(tz).Format("2006-01-02")
		agg, ok := groups[key]
		if !ok {
			agg = &DailyAggregate{Date: key}
			groups[key] = agg
		}
		agg.InputTokens += e.InputTokens
		agg.OutputTokens += e.OutputTokens
		agg.CacheCreationTokens += e.CacheCreationTokens
		agg.CacheReadTokens += e.CacheReadTokens
		agg.TotalCost += e.CostUSD
		agg.EntriesCount++
	}

	result := make([]DailyAggregate, 0, len(groups))
	for _, agg := range groups {
		result = append(result, *agg)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date > result[j].Date // descending
	})
	return result
}

// AggregateMonthly groups entries by month in the given timezone.
func AggregateMonthly(entries []UsageEntry, tz *time.Location, year int, month time.Month) MonthlyAggregate {
	key := time.Date(year, month, 1, 0, 0, 0, 0, tz).Format("2006-01")
	agg := MonthlyAggregate{
		Month: key,
		Days:  make(map[int]DailyAggregate),
	}

	for _, e := range entries {
		local := e.Timestamp.In(tz)
		if local.Year() != year || local.Month() != month {
			continue
		}
		day := local.Day()
		d := agg.Days[day]
		d.Date = local.Format("2006-01-02")
		d.InputTokens += e.InputTokens
		d.OutputTokens += e.OutputTokens
		d.CacheCreationTokens += e.CacheCreationTokens
		d.CacheReadTokens += e.CacheReadTokens
		d.TotalCost += e.CostUSD
		d.EntriesCount++
		agg.Days[day] = d

		agg.TotalCost += e.CostUSD
		agg.TotalTokens += e.TotalTokens()
		agg.TotalInputTokens += e.InputTokens
		agg.TotalOutputTokens += e.OutputTokens
		agg.TotalCacheCreation += e.CacheCreationTokens
		agg.TotalCacheRead += e.CacheReadTokens
		agg.TotalCalls++
	}

	return agg
}
