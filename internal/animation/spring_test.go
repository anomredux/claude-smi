package animation

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

// assertConvergesTo runs Update steps and checks the spring converges.
func assertConvergesTo(t *testing.T, a *SpringAnimator, key SpringKey, target float64, steps int, tol float64) {
	t.Helper()
	for i := 0; i < steps; i++ {
		a.Update()
	}
	got := a.Get(key)
	if math.Abs(got-target) > tol {
		t.Errorf("after %d steps, Get(%s) = %f; want ~%f (tol=%f)", steps, key, got, target, tol)
	}
}

func TestSpringAnimator_Convergence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    SpringKey
		target float64
		preset SpringPreset
		steps  int
		tol    float64
	}{
		{"smooth to 100", "test:a", 100.0, PresetSmooth, 40, 0.5},
		{"smooth to 0.75", "test:b", 0.75, PresetSmooth, 40, 0.01},
		{"snap preset to 100", "test:c", 100.0, PresetSnap, 20, 1.0},
		{"large target", "test:d", 1_000_000.0, PresetSmooth, 60, 100.0},
		{"zero target from nonzero", "test:e", 0.0, PresetSmooth, 40, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewSpringAnimator()
			// For "zero target from nonzero", start at 100
			if tt.target == 0 {
				a.SetSnap(tt.key, 100.0)
			}
			a.SetWithPreset(tt.key, tt.target, tt.preset)
			assertConvergesTo(t, a, tt.key, tt.target, tt.steps, tt.tol)
		})
	}
}

func TestSpringAnimator_GetUnknownKey(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	if got := a.Get("missing"); got != 0 {
		t.Errorf("Get(missing) = %f; want 0", got)
	}
}

func TestSpringAnimator_SetDoesNotInstantlyConverge(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.Set("x", 100.0)
	if got := a.Get("x"); got == 100.0 {
		t.Error("Get should not equal target before Update")
	}
}

func TestSpringAnimator_SetSnap(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.SetSnap("pos", 42.0)
	if got := a.Get("pos"); got != 42.0 {
		t.Errorf("SetSnap: Get(pos) = %f; want 42.0", got)
	}
}

func TestSpringAnimator_SetSnapThenAnimate(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.SetSnap("x", 50.0)
	a.Set("x", 100.0)
	assertConvergesTo(t, a, "x", 100.0, 40, 0.5)
}

func TestSpringAnimator_Retarget(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.Set("x", 50.0)
	for i := 0; i < 10; i++ {
		a.Update()
	}
	a.Set("x", 200.0)
	assertConvergesTo(t, a, "x", 200.0, 60, 0.5)
}

func TestSpringAnimator_MultipleKeys(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.Set("a", 10.0)
	a.Set("b", 20.0)
	assertConvergesTo(t, a, "a", 10.0, 40, 0.5)
	// "b" also converged (Update advances all springs)
	if got := a.Get("b"); math.Abs(got-20.0) > 0.5 {
		t.Errorf("Get(b) = %f; want ~20.0", got)
	}
}

func TestSpringAnimator_Delete(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.SetSnap("x", 42.0)
	a.Delete("x")
	if got := a.Get("x"); got != 0 {
		t.Errorf("after Delete, Get(x) = %f; want 0", got)
	}
}

func TestSpringAnimator_SettledSkipsUpdate(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.SetSnap("done", 50.0)
	a.Update()
	if got := a.Get("done"); got != 50.0 {
		t.Errorf("settled spring drifted: got %f, want 50.0", got)
	}
}

func TestSpringAnimator_SettledReactivates(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.SetSnap("x", 10.0)
	a.Update() // settled
	a.Set("x", 50.0)
	assertConvergesTo(t, a, "x", 50.0, 40, 0.5)
}

func TestSpringAnimator_Settled(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	if !a.Settled() {
		t.Error("empty animator should be settled")
	}
	a.Set("x", 100.0)
	if a.Settled() {
		t.Error("animator with active spring should not be settled")
	}
	// Loop until settled instead of hardcoded step count
	const maxSteps = 500
	for i := 0; i < maxSteps; i++ {
		a.Update()
		if a.Settled() {
			return // success
		}
	}
	t.Errorf("animator not settled after %d steps", maxSteps)
}

func TestSpringAnimator_NilReceiver(t *testing.T) {
	t.Parallel()
	var a *SpringAnimator
	// All methods must be nil-safe
	if got := a.Get("x"); got != 0 {
		t.Errorf("nil.Get = %f; want 0", got)
	}
	a.Set("x", 1.0)     // no panic
	a.SetSnap("x", 1.0) // no panic
	a.Update()          // no panic
	a.Delete("x")       // no panic
	if !a.Settled() {
		t.Error("nil.Settled should return true")
	}
}

func TestSpringAnimator_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()
	a.Set("x", 100.0)

	var wg sync.WaitGroup
	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			a.Update()
		}
	}()
	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = a.Get("x")
		}
	}()
	wg.Wait()
}

func TestSpringAnimator_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target float64
		steps  int
		tol    float64
	}{
		{"negative target", -50.0, 40, 0.5},
		{"very large target", 1e9, 80, 1e5},
		{"very small target", 0.001, 40, 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewSpringAnimator()
			a.Set("x", tt.target)
			assertConvergesTo(t, a, "x", tt.target, tt.steps, tt.tol)
		})
	}
}

func TestSpringAnimator_NaNInfSanitization(t *testing.T) {
	t.Parallel()
	a := NewSpringAnimator()

	// NaN target should be sanitized to 0
	a.Set("nan", math.NaN())
	if got := a.Get("nan"); got != 0 {
		t.Errorf("NaN target: Get = %f; want 0", got)
	}

	// +Inf target should be sanitized to 0
	a.Set("inf", math.Inf(1))
	if got := a.Get("inf"); got != 0 {
		t.Errorf("+Inf target: Get = %f; want 0", got)
	}

	// -Inf target should be sanitized to 0
	a.Set("neginf", math.Inf(-1))
	if got := a.Get("neginf"); got != 0 {
		t.Errorf("-Inf target: Get = %f; want 0", got)
	}
}

func BenchmarkSpringAnimatorUpdate(b *testing.B) {
	sizes := []int{10, 50, 100}
	for _, n := range sizes {
		// Benchmark with active (non-settled) springs
		b.Run(fmt.Sprintf("active_springs=%d", n), func(b *testing.B) {
			a := NewSpringAnimator()
			for i := 0; i < n; i++ {
				a.Set(SpringKey(fmt.Sprintf("bench:%d", i)), float64(i)*100)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Re-target to keep springs active
				if i%20 == 0 {
					for j := 0; j < n; j++ {
						a.Set(SpringKey(fmt.Sprintf("bench:%d", j)), float64(j)*100+float64(i))
					}
				}
				a.Update()
			}
		})
	}
}
