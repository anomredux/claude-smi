package animation

import (
	"math"
	"sync"

	"github.com/charmbracelet/harmonica"
)

// Animator provides read/write access to animated spring values.
// Views should depend on this interface, not the concrete SpringAnimator.
type Animator interface {
	Get(key SpringKey) float64
	Set(key SpringKey, target float64)
	SetWithPreset(key SpringKey, target float64, preset SpringPreset)
	SetSnap(key SpringKey, value float64)
	Delete(key SpringKey)
}

// nullAnimator is a no-op implementation of Animator.
// Used to eliminate nil checks throughout the codebase.
type nullAnimator struct{}

func (nullAnimator) Get(SpringKey) float64                          { return 0 }
func (nullAnimator) Set(SpringKey, float64)                         {}
func (nullAnimator) SetWithPreset(SpringKey, float64, SpringPreset) {}
func (nullAnimator) SetSnap(SpringKey, float64)                     {}
func (nullAnimator) Delete(SpringKey)                               {}

// NullAnimator returns a no-op Animator that safely ignores all calls.
func NullAnimator() Animator { return nullAnimator{} }

// SpringPreset defines spring tuning for different animation contexts.
type SpringPreset struct {
	Frequency float64 // Hz — higher = snappier
	Damping   float64 // 1.0 = critically damped (no overshoot)
}

var (
	// PresetSnap: fast response, no overshoot. For scroll.
	PresetSnap = SpringPreset{Frequency: 8.0, Damping: 1.0}
	// PresetSmooth: gentle ease-in. For gauges and counters.
	PresetSmooth = SpringPreset{Frequency: 4.0, Damping: 1.0}
)

// frameFPS is the internal simulation rate matching the AnimFrameMsg interval.
// Each call to Update advances the simulation by 1/frameFPS seconds.
const frameFPS = 20 // 50ms per frame

// convergenceThreshold — absolute threshold for small-range values (0-1).
// For large-range values, use relative comparison.
const convergenceThreshold = 0.0005

// springEntry tracks one named spring value converging toward a target.
type springEntry struct {
	spring harmonica.Spring
	preset SpringPreset
	pos    float64
	vel    float64
	target float64
}

// settled returns true if the spring has effectively converged.
func (e *springEntry) settled() bool {
	delta := math.Abs(e.pos - e.target)
	// Use the larger of absolute and relative thresholds
	// to handle all value ranges including target=0.
	tol := convergenceThreshold
	if absTarget := math.Abs(e.target); absTarget > 1.0 {
		if rel := absTarget * 0.0001; rel > tol {
			tol = rel
		}
	}
	return delta < tol && math.Abs(e.vel) < tol
}

// SpringAnimator manages named spring-animated float64 values.
// Thread-safe: Set/SetSnap/Update hold a write lock; Get holds a read lock.
// Implements Animator interface. App holds the concrete type for Update/Settled.
type SpringAnimator struct {
	mu      sync.RWMutex
	springs map[SpringKey]*springEntry
}

// Compile-time check: SpringAnimator implements Animator.
var _ Animator = (*SpringAnimator)(nil)

// NewSpringAnimator returns a ready-to-use animator.
func NewSpringAnimator() *SpringAnimator {
	return &SpringAnimator{
		springs: make(map[SpringKey]*springEntry),
	}
}

func newSpring(p SpringPreset) harmonica.Spring {
	return harmonica.NewSpring(harmonica.FPS(frameFPS), p.Frequency, p.Damping)
}

// sanitizeTarget rejects NaN/Inf targets by snapping to 0.
func sanitizeTarget(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// Set pushes a new target using PresetSmooth. Creates the spring if absent.
func (a *SpringAnimator) Set(key SpringKey, target float64) {
	if a == nil {
		return
	}
	a.SetWithPreset(key, target, PresetSmooth)
}

// SetWithPreset pushes a new target using a specific preset.
// If the spring already exists with a different preset, it is recreated.
func (a *SpringAnimator) SetWithPreset(key SpringKey, target float64, preset SpringPreset) {
	if a == nil {
		return
	}
	target = sanitizeTarget(target)
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.springs[key]
	if !ok {
		e = &springEntry{spring: newSpring(preset), preset: preset}
		a.springs[key] = e
	} else if e.preset != preset {
		// Preset changed — recreate spring, preserve current position/velocity.
		pos, vel := e.pos, e.vel
		e.spring = newSpring(preset)
		e.preset = preset
		e.pos = pos
		e.vel = vel
	}
	e.target = target
}

// SetSnap sets the value immediately without animation.
func (a *SpringAnimator) SetSnap(key SpringKey, value float64) {
	if a == nil {
		return
	}
	value = sanitizeTarget(value)
	a.mu.Lock()
	defer a.mu.Unlock()
	e, ok := a.springs[key]
	if !ok {
		e = &springEntry{spring: newSpring(PresetSmooth), preset: PresetSmooth}
		a.springs[key] = e
	}
	e.pos = value
	e.vel = 0
	e.target = value
}

// Get returns the current interpolated value. Returns 0 for unknown keys.
func (a *SpringAnimator) Get(key SpringKey) float64 {
	if a == nil {
		return 0
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	e, ok := a.springs[key]
	if !ok {
		return 0
	}
	return e.pos
}

// Delete removes a spring by key.
func (a *SpringAnimator) Delete(key SpringKey) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.springs, key)
}

// Settled returns true when all springs have converged to their targets.
func (a *SpringAnimator) Settled() bool {
	if a == nil {
		return true
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, e := range a.springs {
		if !e.settled() {
			return false
		}
	}
	return true
}

// Update advances all active springs by one frame (1/frameFPS seconds).
// Settled springs are snapped to their target and skipped.
func (a *SpringAnimator) Update() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, e := range a.springs {
		if e.settled() {
			e.pos = e.target
			e.vel = 0
			continue
		}
		e.pos, e.vel = e.spring.Update(e.pos, e.vel, e.target)
		// Guard against NaN/Inf from degenerate spring output
		if math.IsNaN(e.pos) || math.IsInf(e.pos, 0) {
			e.pos = e.target
			e.vel = 0
		}
	}
}
