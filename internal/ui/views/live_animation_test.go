package views

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/anomredux/claude-smi/internal/animation"
	"github.com/anomredux/claude-smi/internal/api"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/pricing"
)

func TestLiveView_GaugeAnimation(t *testing.T) {
	t.Parallel()
	animator := animation.NewSpringAnimator()
	table, _ := pricing.LoadDefault()
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	lv := NewLiveView(time.UTC, calc, animator)

	// Simulate API data arrival
	lv.SetApiUsage(&api.UsageData{
		FiveHour: api.WindowData{Utilization: 75.0},
		SevenDay: api.WindowData{Utilization: 30.0},
	})

	// Spring targets should be set — run one Update to verify it moves
	animator.Update()
	if got := animator.Get(animation.KeyGauge5h); got == 0 {
		t.Error("gauge:5h spring not moving after SetApiUsage + Update")
	}

	// After convergence, value should match
	for i := 0; i < 120; i++ {
		animator.Update()
	}
	got5h := animator.Get(animation.KeyGauge5h)
	if math.Abs(got5h-0.75) > 0.01 {
		t.Errorf("gauge:5h = %f; want ~0.75", got5h)
	}
	got7d := animator.Get(animation.KeyGauge7d)
	if math.Abs(got7d-0.30) > 0.01 {
		t.Errorf("gauge:7d = %f; want ~0.30", got7d)
	}
}

func TestLiveView_RenderUtilizationUsesAnimatedValues(t *testing.T) {
	t.Parallel()
	animator := animation.NewSpringAnimator()
	table, _ := pricing.LoadDefault()
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	lv := NewLiveView(time.UTC, calc, animator)

	lv.SetApiUsage(&api.UsageData{
		FiveHour: api.WindowData{Utilization: 50.0},
		SevenDay: api.WindowData{Utilization: 50.0},
	})

	// Before any Update, rendered output should show low percentages
	output := lv.Render(100, 9999, false)
	// After convergence
	for i := 0; i < 120; i++ {
		animator.Update()
	}
	outputAfter := lv.Render(100, 9999, false)

	// Both should render without panic
	if len(output) == 0 || len(outputAfter) == 0 {
		t.Error("render produced empty output")
	}
	// After convergence, "50.0%" should appear
	if !strings.Contains(outputAfter, "50.0%") {
		t.Logf("output: %s", outputAfter)
		t.Error("expected 50.0%% in converged output")
	}
}

// mockApiData returns UsageData with a session window covering "now".
func mockApiData() *api.UsageData {
	resetAt := time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339)
	return &api.UsageData{
		FiveHour:  api.WindowData{Utilization: 10.0, ResetsAt: resetAt},
		SevenDay:  api.WindowData{Utilization: 5.0, ResetsAt: resetAt},
		FetchedAt: time.Now(),
	}
}

func TestLiveView_CounterAnimation(t *testing.T) {
	t.Parallel()
	animator := animation.NewSpringAnimator()
	table, _ := pricing.LoadDefault()
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	lv := NewLiveView(time.UTC, calc, animator)
	lv.SetApiUsage(mockApiData())

	entries := []domain.UsageEntry{
		{
			Timestamp:   time.Now().Add(-10 * time.Minute),
			Model:       "claude-sonnet-4-5-20250514",
			InputTokens: 5000, OutputTokens: 3000,
			CostUSD: 0.05,
		},
	}
	lv.SetData(entries, nil, nil)

	for i := 0; i < 120; i++ {
		animator.Update()
	}

	gotInput := animator.Get(animation.KeyBurnInput)
	if math.Abs(gotInput-5000) > 50 {
		t.Errorf("burn:input = %f; want ~5000", gotInput)
	}
}

func TestLiveView_CounterSnapsOnFirstLoad(t *testing.T) {
	t.Parallel()
	animator := animation.NewSpringAnimator()
	table, _ := pricing.LoadDefault()
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	lv := NewLiveView(time.UTC, calc, animator)
	lv.SetApiUsage(mockApiData())

	entries := []domain.UsageEntry{
		{
			Timestamp:   time.Now().Add(-5 * time.Minute),
			Model:       "claude-sonnet-4-5-20250514",
			InputTokens: 10000, OutputTokens: 5000,
			CostUSD: 0.10,
		},
	}

	// First SetData should snap (no meaningless 0→N animation)
	lv.SetData(entries, nil, nil)

	// Values should be at target immediately (snapped, not animated)
	gotInput := animator.Get(animation.KeyBurnInput)
	if math.Abs(gotInput-10000) > 1 {
		t.Errorf("first load should snap: burn:input = %f; want 10000", gotInput)
	}
}

func TestLiveView_CounterClearedOnNoData(t *testing.T) {
	t.Parallel()
	animator := animation.NewSpringAnimator()
	table, _ := pricing.LoadDefault()
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	lv := NewLiveView(time.UTC, calc, animator)
	lv.SetApiUsage(mockApiData())

	entries := []domain.UsageEntry{
		{Timestamp: time.Now(), Model: "m", InputTokens: 1000, CostUSD: 0.01},
	}
	lv.SetData(entries, nil, nil)
	lv.SetData(nil, nil, nil) // no data

	// Springs should be deleted
	if got := animator.Get(animation.KeyBurnInput); got != 0 {
		t.Errorf("after clear, burn:input = %f; want 0", got)
	}
}
