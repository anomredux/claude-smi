package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anomredux/claude-smi/internal/api"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/pricing"
	"github.com/anomredux/claude-smi/internal/ui/views"
)

func main() {
	tz := time.UTC
	table, _ := pricing.LoadDefault()
	if table == nil {
		table = make(pricing.PricingTable)
	}
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	lv := views.NewLiveView(tz, calc)

	// Try fetching real API data
	apiData, err := api.FetchUsage(context.Background())
	if err != nil {
		fmt.Printf("API fetch failed (using mock): %v\n\n", err)
		// Use mock API data
		now := time.Now().UTC()
		mockResetTime := now.Add(1*time.Hour + 55*time.Minute)
		apiData = &api.UsageData{
			FiveHour: api.WindowData{
				Utilization: 42.0,
				ResetsAt:    mockResetTime.Format(time.RFC3339),
			},
			SevenDay: api.WindowData{
				Utilization: 18.5,
				ResetsAt:    now.Add(5 * 24 * time.Hour).Format(time.RFC3339),
			},
			FetchedAt: now,
		}
	} else {
		fmt.Println("Using real API data!")
		fmt.Printf("  5h: %.1f%% (resets: %s)\n", apiData.FiveHour.Utilization, apiData.FiveHour.ResetsAt)
		fmt.Printf("  7d: %.1f%% (resets: %s)\n\n", apiData.SevenDay.Utilization, apiData.SevenDay.ResetsAt)
	}

	lv.SetApiUsage(apiData)

	// Create entries within the session window
	sessionStart, _ := apiData.SessionStart()
	entries := []domain.UsageEntry{
		{
			Timestamp:    sessionStart.Add(30 * time.Minute),
			InputTokens:  5000,
			OutputTokens: 3000,
			CostUSD:      12.50,
			Model:        "claude-opus-4-6",
		},
		{
			Timestamp:    sessionStart.Add(1 * time.Hour),
			InputTokens:  8000,
			OutputTokens: 4000,
			CostUSD:      18.30,
			Model:        "claude-opus-4-6",
		},
		{
			Timestamp:    sessionStart.Add(2 * time.Hour),
			InputTokens:  1200,
			OutputTokens: 800,
			CostUSD:      0.45,
			Model:        "claude-haiku-4-5-20251001",
		},
	}

	blocks := domain.BuildBlocks(entries)
	lv.SetData(entries, blocks, nil)

	fmt.Println(lv.Render(100, 40, false))
}
