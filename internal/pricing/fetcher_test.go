package pricing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLiteLLM(t *testing.T) {
	// Mock LiteLLM JSON response
	mockData := map[string]interface{}{
		"claude-sonnet-4-6": map[string]interface{}{
			"input_cost_per_token":                3e-06,
			"output_cost_per_token":               1.5e-05,
			"cache_creation_input_token_cost":      3.75e-06,
			"cache_read_input_token_cost":          3e-07,
		},
		"claude-opus-4-6": map[string]interface{}{
			"input_cost_per_token":                5e-06,
			"output_cost_per_token":               2.5e-05,
			"cache_creation_input_token_cost":      6.25e-06,
			"cache_read_input_token_cost":          5e-07,
		},
		// Provider-prefixed models should be excluded
		"anthropic.claude-sonnet-4-6": map[string]interface{}{
			"input_cost_per_token":  3e-06,
			"output_cost_per_token": 1.5e-05,
		},
		"vertex_ai/claude-sonnet-4-6": map[string]interface{}{
			"input_cost_per_token":  3e-06,
			"output_cost_per_token": 1.5e-05,
		},
		// Non-Claude models should be excluded
		"gpt-4o": map[string]interface{}{
			"input_cost_per_token":  2.5e-06,
			"output_cost_per_token": 1e-05,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockData)
	}))
	defer ts.Close()

	// Override the URL for testing
	origURL := LiteLLMURL
	LiteLLMURL = ts.URL
	defer func() { LiteLLMURL = origURL }()

	table, err := FetchLiteLLM(context.Background())
	if err != nil {
		t.Fatalf("FetchLiteLLM failed: %v", err)
	}

	// Should only have 2 bare claude- models
	if len(table) != 2 {
		t.Errorf("expected 2 models, got %d", len(table))
	}

	// Check sonnet pricing (per-token → per-1M-tokens conversion)
	sonnet, ok := table["claude-sonnet-4-6"]
	if !ok {
		t.Fatal("missing claude-sonnet-4-6")
	}
	if !almostEqual(sonnet.Input, 3.0, 0.001) {
		t.Errorf("sonnet Input = %f, want 3.0", sonnet.Input)
	}
	if !almostEqual(sonnet.Output, 15.0, 0.001) {
		t.Errorf("sonnet Output = %f, want 15.0", sonnet.Output)
	}
	if !almostEqual(sonnet.CacheCreation, 3.75, 0.001) {
		t.Errorf("sonnet CacheCreation = %f, want 3.75", sonnet.CacheCreation)
	}
	if !almostEqual(sonnet.CacheRead, 0.30, 0.001) {
		t.Errorf("sonnet CacheRead = %f, want 0.30", sonnet.CacheRead)
	}

	// Check opus pricing
	opus, ok := table["claude-opus-4-6"]
	if !ok {
		t.Fatal("missing claude-opus-4-6")
	}
	if !almostEqual(opus.Input, 5.0, 0.001) {
		t.Errorf("opus Input = %f, want 5.0", opus.Input)
	}
	if !almostEqual(opus.Output, 25.0, 0.001) {
		t.Errorf("opus Output = %f, want 25.0", opus.Output)
	}
}

func TestFetchLiteLLM_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	origURL := LiteLLMURL
	LiteLLMURL = ts.URL
	defer func() { LiteLLMURL = origURL }()

	_, err := FetchLiteLLM(context.Background())
	if err == nil {
		t.Error("expected error on HTTP 500")
	}
}

func TestFetchLiteLLM_NoClaude(t *testing.T) {
	mockData := map[string]interface{}{
		"gpt-4o": map[string]interface{}{
			"input_cost_per_token":  2.5e-06,
			"output_cost_per_token": 1e-05,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mockData)
	}))
	defer ts.Close()

	origURL := LiteLLMURL
	LiteLLMURL = ts.URL
	defer func() { LiteLLMURL = origURL }()

	table, err := FetchLiteLLM(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(table) != 0 {
		t.Errorf("expected 0 models, got %d", len(table))
	}
}

func TestPricingTable_Merge(t *testing.T) {
	base := PricingTable{
		"claude-opus-4-6":  {Input: 15.0, Output: 75.0},
		"claude-haiku-4-5": {Input: 0.80, Output: 4.0},
	}
	other := PricingTable{
		"claude-opus-4-6":   {Input: 16.0, Output: 80.0}, // override
		"claude-sonnet-4-6": {Input: 3.0, Output: 15.0},  // new
	}

	base.Merge(other)

	if len(base) != 3 {
		t.Errorf("expected 3 models after merge, got %d", len(base))
	}
	// opus should be overwritten
	if base["claude-opus-4-6"].Input != 16.0 {
		t.Errorf("opus Input should be overwritten to 16.0, got %f", base["claude-opus-4-6"].Input)
	}
	// haiku should remain
	if base["claude-haiku-4-5"].Input != 0.80 {
		t.Errorf("haiku should remain, got %f", base["claude-haiku-4-5"].Input)
	}
	// sonnet should be added
	if base["claude-sonnet-4-6"].Input != 3.0 {
		t.Errorf("sonnet should be added, got %f", base["claude-sonnet-4-6"].Input)
	}
}

func TestFilterClaudeModels_MissingPrices(t *testing.T) {
	inputOnly := 3e-06
	raw := map[string]liteLLMEntry{
		// Missing output price -> should be excluded
		"claude-incomplete": {
			InputCostPerToken: &inputOnly,
		},
	}

	table := filterClaudeModels(raw)
	if len(table) != 0 {
		t.Errorf("model without output price should be excluded, got %d", len(table))
	}
}
