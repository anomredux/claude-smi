package pricing

import "github.com/anomredux/claude-smi/internal/domain"

type CostMode string

const (
	CostModeAuto      CostMode = "auto"
	CostModeDisplay   CostMode = "display"
	CostModeCalculate CostMode = "calculate"
)

type Calculator struct {
	table PricingTable
	mode  CostMode
}

func NewCalculator(table PricingTable, mode CostMode) *Calculator {
	return &Calculator{table: table, mode: mode}
}

// UpdateTable replaces the pricing table used for cost calculations.
func (c *Calculator) UpdateTable(table PricingTable) {
	c.table = table
}

// Calculate returns the cost in USD for a single entry.
func (c *Calculator) Calculate(e *domain.UsageEntry) float64 {
	switch c.mode {
	case CostModeDisplay:
		return e.CostUSD
	case CostModeCalculate:
		return c.calculateFromTokens(e)
	default: // auto
		if e.CostUSD > 0 {
			return e.CostUSD
		}
		return c.calculateFromTokens(e)
	}
}

func (c *Calculator) calculateFromTokens(e *domain.UsageEntry) float64 {
	pricing, ok := c.table.Lookup(e.Model)
	if !ok {
		return 0
	}

	cost := float64(e.InputTokens) * pricing.Input / 1_000_000
	cost += float64(e.OutputTokens) * pricing.Output / 1_000_000
	cost += float64(e.CacheCreationTokens) * pricing.CacheCreation / 1_000_000
	cost += float64(e.CacheReadTokens) * pricing.CacheRead / 1_000_000

	return cost
}

// ApplyAll calculates and sets CostUSD on all entries.
func (c *Calculator) ApplyAll(entries []domain.UsageEntry) {
	for i := range entries {
		entries[i].CostUSD = c.Calculate(&entries[i])
	}
}

// CacheSavings returns the cost saved by cache reads for a single entry.
// Savings = cache_read_tokens × (input_rate - cache_read_rate) / 1M.
func (c *Calculator) CacheSavings(e *domain.UsageEntry) float64 {
	if e.CacheReadTokens == 0 {
		return 0
	}
	pricing, ok := c.table.Lookup(e.Model)
	if !ok {
		return 0
	}
	return float64(e.CacheReadTokens) * (pricing.Input - pricing.CacheRead) / 1_000_000
}
