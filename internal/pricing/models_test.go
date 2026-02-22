package pricing

import "testing"

func TestPricingTable_Lookup_EmptyTable(t *testing.T) {
	table := PricingTable{}
	_, ok := table.Lookup("claude-opus-4-6")
	if ok {
		t.Error("empty table should return false")
	}
}

func TestPricingTable_Lookup_EmptyModel(t *testing.T) {
	table := PricingTable{
		"claude-opus-4-6": {Input: 5.0, Output: 25.0},
	}
	// Empty model matches via reverse prefix fallback (every key HasPrefix "")
	_, ok := table.Lookup("")
	if !ok {
		t.Error("empty model should match via reverse prefix fallback")
	}
}

func TestPricingTable_Lookup_ReversePrefixMatch(t *testing.T) {
	// When a table key is a prefix of the query (normal prefix match)
	// AND when the query is a prefix of a table key (reverse prefix match)
	table := PricingTable{
		"claude-sonnet-4-6-20260101": {Input: 3.0, Output: 15.0},
	}

	// Query is prefix of key → should match via fallback sorted iteration
	p, ok := table.Lookup("claude-sonnet-4-6")
	if !ok {
		t.Fatal("reverse prefix match should work")
	}
	if p.Input != 3.0 {
		t.Errorf("got Input=%f, want 3.0", p.Input)
	}
}

func TestPricingTable_Merge_OverrideAndAdd(t *testing.T) {
	base := PricingTable{
		"claude-opus-4-6":  {Input: 5.0, Output: 25.0},
		"claude-haiku-4-5": {Input: 1.0, Output: 5.0},
	}
	overlay := PricingTable{
		"claude-opus-4-6":    {Input: 4.0, Output: 20.0}, // override
		"claude-sonnet-4-6":  {Input: 3.0, Output: 15.0}, // new
	}

	base.Merge(overlay)

	if len(base) != 3 {
		t.Errorf("expected 3 models after merge, got %d", len(base))
	}

	// Overridden
	if base["claude-opus-4-6"].Input != 4.0 {
		t.Errorf("opus input should be overridden to 4.0, got %f", base["claude-opus-4-6"].Input)
	}

	// Preserved
	if base["claude-haiku-4-5"].Input != 1.0 {
		t.Errorf("haiku should be preserved, got %f", base["claude-haiku-4-5"].Input)
	}

	// Added
	if base["claude-sonnet-4-6"].Input != 3.0 {
		t.Errorf("sonnet should be added, got %f", base["claude-sonnet-4-6"].Input)
	}
}

func TestPricingTable_Merge_WithEmpty(t *testing.T) {
	base := PricingTable{
		"claude-opus-4-6": {Input: 5.0},
	}
	base.Merge(PricingTable{})

	if len(base) != 1 {
		t.Errorf("merging empty should not change base, got %d", len(base))
	}
}

func TestPricingTable_Lookup_LongestPrefixWins(t *testing.T) {
	table := PricingTable{
		"claude":            {Input: 1.0},
		"claude-opus":       {Input: 2.0},
		"claude-opus-4":     {Input: 3.0},
		"claude-opus-4-6":   {Input: 5.0},
	}

	tests := []struct {
		name  string
		model string
		want  float64
	}{
		{"exact match", "claude-opus-4-6", 5.0},
		{"longest prefix claude-opus-4-6-xxx", "claude-opus-4-6-20260101", 5.0},
		{"longest prefix claude-opus-4-xxx", "claude-opus-4-5", 3.0},
		{"longest prefix claude-opus-xxx", "claude-opus-latest", 2.0},
		{"longest prefix claude-xxx", "claude-haiku", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := table.Lookup(tt.model)
			if !ok {
				t.Fatalf("expected match for %q", tt.model)
			}
			if p.Input != tt.want {
				t.Errorf("Lookup(%q).Input = %f, want %f", tt.model, p.Input, tt.want)
			}
		})
	}
}

func TestLoadDefault_HasExpectedModels(t *testing.T) {
	table, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}

	expected := []string{
		"claude-opus-4-6",
		"claude-sonnet-4-6",
		"claude-haiku-4-5",
	}

	for _, model := range expected {
		if _, ok := table[model]; !ok {
			t.Errorf("missing expected model %q", model)
		}
	}
}

func TestLoadDefault_PricingValues(t *testing.T) {
	table, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}

	opus := table["claude-opus-4-6"]
	if opus.Input <= 0 || opus.Output <= 0 {
		t.Errorf("opus pricing should be positive: Input=%f, Output=%f", opus.Input, opus.Output)
	}
	if opus.Output <= opus.Input {
		t.Errorf("output price should exceed input price: Input=%f, Output=%f", opus.Input, opus.Output)
	}
}
