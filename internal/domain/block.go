package domain

import "time"

const BlockDuration = 5 * time.Hour

type BlockStatus string

const (
	BlockActive BlockStatus = "active"
	BlockDone   BlockStatus = "done"
)

type SessionBlock struct {
	StartTime           time.Time
	EndTime             time.Time // StartTime + 5h
	Entries             []UsageEntry
	TotalTokens         int
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	TotalCost           float64
	MessageCount        int
	Status              BlockStatus
	Models              map[string]ModelBreakdown
}

type ModelBreakdown struct {
	Model      string
	Tokens     int
	Cost       float64
	Percentage float64
}

// BuildBlocks groups entries into fixed 5-hour session blocks.
// Entries must be sorted by timestamp (ascending).
func BuildBlocks(entries []UsageEntry) []SessionBlock {
	if len(entries) == 0 {
		return nil
	}

	var blocks []SessionBlock
	var current *SessionBlock

	for _, e := range entries {
		if current == nil || e.Timestamp.After(current.EndTime) || e.Timestamp.Equal(current.EndTime) {
			// Start new block — truncate start to the hour boundary
			if current != nil {
				blocks = append(blocks, *current)
			}
			startHour := e.Timestamp.Truncate(time.Hour)
			current = &SessionBlock{
				StartTime: startHour,
				EndTime:   startHour.Add(BlockDuration),
				Models:    make(map[string]ModelBreakdown),
			}
		}

		current.Entries = append(current.Entries, e)
		current.TotalTokens += e.TotalTokens()
		current.InputTokens += e.InputTokens
		current.OutputTokens += e.OutputTokens
		current.CacheCreationTokens += e.CacheCreationTokens
		current.CacheReadTokens += e.CacheReadTokens
		current.TotalCost += e.CostUSD
		current.MessageCount++

		mb := current.Models[e.Model]
		mb.Model = e.Model
		mb.Tokens += e.TotalTokens()
		mb.Cost += e.CostUSD
		current.Models[e.Model] = mb
	}

	if current != nil {
		blocks = append(blocks, *current)
	}

	// Set status
	now := time.Now().UTC()
	for i := range blocks {
		if now.Before(blocks[i].EndTime) && i == len(blocks)-1 {
			blocks[i].Status = BlockActive
		} else {
			blocks[i].Status = BlockDone
		}
		// Calculate model percentages
		for k, mb := range blocks[i].Models {
			if blocks[i].TotalTokens > 0 {
				mb.Percentage = float64(mb.Tokens) / float64(blocks[i].TotalTokens) * 100
			}
			blocks[i].Models[k] = mb
		}
	}

	return blocks
}
