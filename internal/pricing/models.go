package pricing

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
)

//go:embed pricing.json
var defaultPricingJSON []byte

type ModelPricing struct {
	Input         float64 `json:"input"`          // per 1M tokens
	Output        float64 `json:"output"`
	CacheCreation float64 `json:"cache_creation"`
	CacheRead     float64 `json:"cache_read"`
}

type PricingTable map[string]ModelPricing

func LoadDefault() (PricingTable, error) {
	var table PricingTable
	if err := json.Unmarshal(defaultPricingJSON, &table); err != nil {
		return nil, err
	}
	return table, nil
}

// Merge adds entries from other into pt. Existing keys are overwritten.
func (pt PricingTable) Merge(other PricingTable) {
	for k, v := range other {
		pt[k] = v
	}
}

// Lookup finds pricing for a model, trying exact match then longest prefix match.
func (pt PricingTable) Lookup(model string) (ModelPricing, bool) {
	if p, ok := pt[model]; ok {
		return p, true
	}
	// Collect all matching keys and pick the longest match for determinism.
	var bestKey string
	var bestPricing ModelPricing
	for key, p := range pt {
		if strings.HasPrefix(model, key) && len(key) > len(bestKey) {
			bestKey = key
			bestPricing = p
		}
	}
	if bestKey != "" {
		return bestPricing, true
	}
	// Fall back to sorted keys for deterministic iteration.
	keys := make([]string, 0, len(pt))
	for k := range pt {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.HasPrefix(key, model) {
			return pt[key], true
		}
	}
	return ModelPricing{}, false
}
