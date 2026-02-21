package parser

import (
	"sort"

	"github.com/anomredux/claude-smi/internal/domain"
)

// Dedup removes duplicate entries based on MessageID:RequestID.
// Keeps the first occurrence (earliest timestamp).
// Note: sorts the input slice in place.
func Dedup(entries []domain.UsageEntry) []domain.UsageEntry {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	seen := make(map[string]struct{}, len(entries))
	result := make([]domain.UsageEntry, 0, len(entries))

	for _, e := range entries {
		key := e.DedupKey()
		if key == ":" {
			// Both MessageID and RequestID empty -- keep as-is (can't dedup)
			result = append(result, e)
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, e)
	}

	return result
}
