package parser

import (
	"bufio"
	"encoding/json"
	"io"
	"time"

	"github.com/anomredux/claude-smi/internal/domain"
)

// rawRecord maps the JSONL structure we care about.
type rawRecord struct {
	Type      string   `json:"type"`
	Timestamp string   `json:"timestamp"`
	SessionID string   `json:"sessionId"`
	RequestID string   `json:"requestId"`
	CostUSD   *float64 `json:"costUSD"`
	Message   *struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage *struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// ParseResult holds parsed entries and error stats.
type ParseResult struct {
	Entries    []domain.UsageEntry
	SkipCount  int
	ErrorCount int
}

// ParseReader reads JSONL from an io.Reader, streaming line by line.
// projectPath is derived from the file path for project filtering.
func ParseReader(r io.Reader, projectPath string) ParseResult {
	var result ParseResult
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var rec rawRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			result.ErrorCount++
			continue
		}

		// Only assistant records have usage data
		if rec.Type != "assistant" {
			result.SkipCount++
			continue
		}

		if rec.Message == nil || rec.Message.Usage == nil {
			result.SkipCount++
			continue
		}

		ts, err := time.Parse(time.RFC3339Nano, rec.Timestamp)
		if err != nil {
			ts, err = time.Parse("2006-01-02T15:04:05.000Z", rec.Timestamp)
			if err != nil {
				result.ErrorCount++
				continue
			}
		}

		entry := domain.UsageEntry{
			Timestamp:           ts.UTC(),
			InputTokens:         rec.Message.Usage.InputTokens,
			OutputTokens:        rec.Message.Usage.OutputTokens,
			CacheCreationTokens: rec.Message.Usage.CacheCreationInputTokens,
			CacheReadTokens:     rec.Message.Usage.CacheReadInputTokens,
			Model:               rec.Message.Model,
			MessageID:           rec.Message.ID,
			RequestID:           rec.RequestID,
			SessionID:           rec.SessionID,
			ProjectPath:         projectPath,
		}

		if rec.CostUSD != nil {
			entry.CostUSD = *rec.CostUSD
		}

		result.Entries = append(result.Entries, entry)
	}

	if err := scanner.Err(); err != nil {
		result.ErrorCount++
	}

	return result
}
