package parser

import (
	"strings"
	"testing"
)

func TestParseReader(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"assistant","timestamp":"2026-02-19T13:56:04.070Z","sessionId":"sess_1","requestId":"req_1","message":{"id":"msg_1","model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":200,"cache_read_input_tokens":30}}}`,
		`{"type":"user","timestamp":"2026-02-19T13:55:55.480Z","sessionId":"sess_1","message":{"role":"user","content":"hello"}}`,
		`{"type":"progress","timestamp":"2026-02-19T14:07:06.815Z","sessionId":"sess_1"}`,
		`{"type":"assistant","timestamp":"2026-02-19T14:00:00.000Z","sessionId":"sess_1","requestId":"req_2","message":{"id":"msg_2","model":"claude-haiku-4-5-20251001","usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":100}}}`,
		`invalid json line`,
		``,
	}, "\n")

	result := ParseReader(strings.NewReader(input), "/test/project")

	if len(result.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(result.Entries))
	}
	if result.SkipCount != 2 { // user + progress
		t.Errorf("SkipCount = %d, want 2", result.SkipCount)
	}
	if result.ErrorCount != 1 { // invalid json
		t.Errorf("ErrorCount = %d, want 1", result.ErrorCount)
	}

	// First entry
	e := result.Entries[0]
	if e.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want %q", e.Model, "claude-opus-4-6")
	}
	if e.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", e.InputTokens)
	}
	if e.CacheCreationTokens != 200 {
		t.Errorf("CacheCreationTokens = %d, want 200", e.CacheCreationTokens)
	}
	if e.CacheReadTokens != 30 {
		t.Errorf("CacheReadTokens = %d, want 30", e.CacheReadTokens)
	}
	if e.ProjectPath != "/test/project" {
		t.Errorf("ProjectPath = %q, want %q", e.ProjectPath, "/test/project")
	}
	if e.SessionID != "sess_1" {
		t.Errorf("SessionID = %q, want %q", e.SessionID, "sess_1")
	}

	// Second entry
	e2 := result.Entries[1]
	if e2.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("Model = %q, want %q", e2.Model, "claude-haiku-4-5-20251001")
	}
}

func TestParseReader_Empty(t *testing.T) {
	result := ParseReader(strings.NewReader(""), "/empty")
	if len(result.Entries) != 0 {
		t.Errorf("got %d entries, want 0", len(result.Entries))
	}
}

func TestParseReader_CostUSD(t *testing.T) {
	input := `{"type":"assistant","timestamp":"2026-02-19T13:56:04.070Z","sessionId":"s1","requestId":"r1","costUSD":1.5,"message":{"id":"m1","model":"opus","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`
	result := ParseReader(strings.NewReader(input), "/test")
	if len(result.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(result.Entries))
	}
	if result.Entries[0].CostUSD != 1.5 {
		t.Errorf("CostUSD = %f, want 1.5", result.Entries[0].CostUSD)
	}
}

func TestParseReader_NoUsage(t *testing.T) {
	input := `{"type":"assistant","timestamp":"2026-02-19T13:56:04.070Z","sessionId":"s1","requestId":"r1","message":{"id":"m1","model":"opus"}}`
	result := ParseReader(strings.NewReader(input), "/test")
	if len(result.Entries) != 0 {
		t.Errorf("got %d entries, want 0 (no usage data)", len(result.Entries))
	}
	if result.SkipCount != 1 {
		t.Errorf("SkipCount = %d, want 1", result.SkipCount)
	}
}
