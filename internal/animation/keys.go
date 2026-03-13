package animation

import "fmt"

// SpringKey is a typed key for spring-animated values.
// Defined type (not alias) provides compile-time safety against bare strings.
type SpringKey string

// Scroll spring keys (one per view).
const (
	KeyScrollLive   SpringKey = "scroll:live"
	KeyScrollBlocks SpringKey = "scroll:blocks"
	KeyScrollReport SpringKey = "scroll:report"
)

// ScrollKey returns the scroll spring key for a view index.
// Generates deterministically so new views don't require updating a switch.
func ScrollKey(viewIndex int) SpringKey {
	return SpringKey(fmt.Sprintf("scroll:%d", viewIndex))
}

// Gauge spring keys.
const (
	KeyGauge5h SpringKey = "gauge:5h"
	KeyGauge7d SpringKey = "gauge:7d"
)

// Counter spring keys (burn rate).
const (
	KeyBurnInput       SpringKey = "burn:input"
	KeyBurnOutput      SpringKey = "burn:output"
	KeyBurnCacheCreate SpringKey = "burn:cacheCreate"
	KeyBurnCacheRead   SpringKey = "burn:cacheRead"
	KeyBurnCost        SpringKey = "burn:cost"
	KeyBurnSavings     SpringKey = "burn:savings"
	KeyBurnTokPerMin   SpringKey = "burn:tokPerMin"
	KeyBurnCostPerHr   SpringKey = "burn:costPerHr"
)
