package animation

import (
	"math"
	"testing"
)

// TestSpringAnimator_FullLifecycle simulates the App's usage pattern:
// Set targets, run Update frames, read values, retarget, delete.
func TestSpringAnimator_FullLifecycle(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()

	// Initial snap (view switch / first load)
	a.SetSnap(KeyScrollLive, 0)
	a.SetSnap(KeyGauge5h, 0)
	a.SetSnap(KeyBurnInput, 0)

	// Simulate data arrival — animate to targets
	a.SetWithPreset(KeyScrollLive, 10, PresetSnap)
	a.Set(KeyGauge5h, 0.85)
	a.Set(KeyBurnInput, 50000)

	// Run 120 frames (~6 seconds at 20 FPS)
	for i := 0; i < 120; i++ {
		a.Update()
	}

	// All should converge
	if got := a.Get(KeyScrollLive); math.Abs(got-10) > 0.5 {
		t.Errorf("scroll = %f; want ~10", got)
	}
	if got := a.Get(KeyGauge5h); math.Abs(got-0.85) > 0.01 {
		t.Errorf("gauge = %f; want ~0.85", got)
	}
	if got := a.Get(KeyBurnInput); math.Abs(got-50000) > 50 {
		t.Errorf("burn = %f; want ~50000", got)
	}

	// Delete and verify cleanup
	a.Delete(KeyBurnInput)
	if got := a.Get(KeyBurnInput); got != 0 {
		t.Errorf("after delete, burn = %f; want 0", got)
	}

	// All remaining should be settled
	if !a.Settled() {
		t.Error("should be settled after convergence")
	}
}
