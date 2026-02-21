package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// LiteLLMURL is the URL for the LiteLLM pricing JSON.
// Exported so tests can override it via httptest.
var LiteLLMURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// httpClient is a shared client with sensible timeouts for pricing fetches.
var httpClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        5,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
	},
}

// liteLLMEntry represents a single model entry from LiteLLM pricing JSON.
type liteLLMEntry struct {
	InputCostPerToken *float64 `json:"input_cost_per_token"`
	OutputCostPerToken *float64 `json:"output_cost_per_token"`
	CacheCreationCost *float64 `json:"cache_creation_input_token_cost"`
	CacheReadCost     *float64 `json:"cache_read_input_token_cost"`
}

// FetchLiteLLM fetches pricing from LiteLLM's GitHub-hosted JSON and returns
// a PricingTable containing only Claude models. Prices are converted from
// per-token to per-1M-tokens to match our internal format.
func FetchLiteLLM(ctx context.Context) (PricingTable, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", LiteLLMURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch litellm pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("litellm pricing: HTTP %d", resp.StatusCode)
	}

	var raw map[string]liteLLMEntry
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode litellm pricing: %w", err)
	}

	return filterClaudeModels(raw), nil
}

// filterClaudeModels extracts Claude models from the raw LiteLLM data
// and converts per-token prices to per-1M-token prices.
func filterClaudeModels(raw map[string]liteLLMEntry) PricingTable {
	table := make(PricingTable)
	for key, entry := range raw {
		// Only include bare claude- models (skip provider-prefixed like anthropic.claude-, vertex_ai/claude-)
		if !strings.HasPrefix(key, "claude-") {
			continue
		}
		// Must have at least input and output prices
		if entry.InputCostPerToken == nil || entry.OutputCostPerToken == nil {
			continue
		}

		mp := ModelPricing{
			Input:  *entry.InputCostPerToken * 1_000_000,
			Output: *entry.OutputCostPerToken * 1_000_000,
		}
		if entry.CacheCreationCost != nil {
			mp.CacheCreation = *entry.CacheCreationCost * 1_000_000
		}
		if entry.CacheReadCost != nil {
			mp.CacheRead = *entry.CacheReadCost * 1_000_000
		}

		table[key] = mp
	}
	return table
}
