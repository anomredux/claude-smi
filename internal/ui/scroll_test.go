package ui

import (
	"testing"

	"github.com/anomredux/claude-smi/internal/animation"
)

func TestSetScrollTarget_Clamps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		target        int
		lastContent   int
		contentHeight int
		want          int
	}{
		{"within range", 5, 50, 30, 5},
		{"negative clamped to 0", -3, 50, 30, 0},
		{"exceeds max clamped", 100, 50, 30, 20},
		{"content fits screen", 5, 20, 30, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &scrollState{}
			s.lastContentLines.Store(int64(tt.lastContent))
			a := App{
				scroll:   s,
				height:   tt.contentHeight + 4, // contentHeight = height - 4
				animator: animation.NewSpringAnimator(),
			}
			a.setScrollTarget(tt.target)
			got := a.scroll.viewScrollY[a.activeView]
			if got != tt.want {
				t.Errorf("setScrollTarget(%d) = %d; want %d", tt.target, got, tt.want)
			}
		})
	}
}

func TestSnapScrollSpring(t *testing.T) {
	t.Parallel()
	a := App{
		scroll:   &scrollState{},
		animator: animation.NewSpringAnimator(),
	}
	a.scroll.viewScrollY[ViewLive] = 15
	a.activeView = ViewLive
	a.snapScrollSpring()

	key := animation.ScrollKey(int(ViewLive))
	if got := a.animator.Get(key); got != 15.0 {
		t.Errorf("spring snapped to %f; want 15.0", got)
	}
}
