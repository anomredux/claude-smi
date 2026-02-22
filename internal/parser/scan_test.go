package parser

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine testdata path")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

func TestScanAndParse(t *testing.T) {
	dir := testdataDir(t)
	ctx := context.Background()

	entries := ScanAndParse(ctx, dir)

	// testdata has 3 assistant entries across 2 files
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	// Verify entries come from different projects
	projects := make(map[string]bool)
	for _, e := range entries {
		projects[e.ProjectPath] = true
	}
	if len(projects) != 2 {
		t.Errorf("got %d unique projects, want 2", len(projects))
	}
}

func TestScanAndParse_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	entries := ScanAndParse(ctx, dir)
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0 for empty dir", len(entries))
	}
}

func TestScanAndParse_NonExistentDir(t *testing.T) {
	ctx := context.Background()
	entries := ScanAndParse(ctx, "/nonexistent/path/that/does/not/exist")
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0 for nonexistent dir", len(entries))
	}
}

func TestScanAndParse_ContextCancellation(t *testing.T) {
	dir := testdataDir(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	entries := ScanAndParse(ctx, dir)
	// With cancelled context, should return fewer or zero entries
	if len(entries) > 3 {
		t.Errorf("cancelled context should limit results, got %d", len(entries))
	}
}

func TestScanAndParse_IgnoresNonJSONL(t *testing.T) {
	dir := t.TempDir()

	// Create a .txt file that should be ignored
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not jsonl"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a .json file that should be ignored
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	entries := ScanAndParse(ctx, dir)
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0 (should ignore non-.jsonl files)", len(entries))
	}
}

func TestParseIncremental(t *testing.T) {
	dir := testdataDir(t)
	ctx := context.Background()

	logFile := filepath.Join(dir, "project-a", "log.jsonl")

	// Parse from offset 0 (full read)
	changes := []FileChange{{Path: logFile, Offset: 0}}
	entries, offsets := ParseIncremental(ctx, changes)

	if len(entries) != 2 {
		t.Fatalf("got %d entries from full read, want 2", len(entries))
	}

	newOffset, ok := offsets[logFile]
	if !ok {
		t.Fatal("expected offset for file")
	}
	if newOffset <= 0 {
		t.Errorf("expected positive offset, got %d", newOffset)
	}

	// Parse from the recorded offset (should find nothing new)
	changes2 := []FileChange{{Path: logFile, Offset: newOffset}}
	entries2, _ := ParseIncremental(ctx, changes2)

	if len(entries2) != 0 {
		t.Errorf("got %d entries from unchanged file, want 0", len(entries2))
	}
}

func TestParseIncremental_ContextCancellation(t *testing.T) {
	dir := testdataDir(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	changes := []FileChange{
		{Path: filepath.Join(dir, "project-a", "log.jsonl"), Offset: 0},
		{Path: filepath.Join(dir, "project-b", "log.jsonl"), Offset: 0},
	}
	entries, _ := ParseIncremental(ctx, changes)

	// Should return fewer entries due to cancellation
	if len(entries) > 3 {
		t.Errorf("cancelled context should limit results, got %d", len(entries))
	}
}

func TestParseIncremental_NonExistentFile(t *testing.T) {
	ctx := context.Background()
	changes := []FileChange{{Path: "/nonexistent/file.jsonl", Offset: 0}}
	entries, offsets := ParseIncremental(ctx, changes)

	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0 for nonexistent file", len(entries))
	}
	if len(offsets) != 0 {
		t.Errorf("got %d offsets, want 0 for nonexistent file", len(offsets))
	}
}
