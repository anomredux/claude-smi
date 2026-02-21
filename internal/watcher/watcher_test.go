package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestInitialScan(t *testing.T) {
	dir := t.TempDir()

	// Create test JSONL files
	os.WriteFile(filepath.Join(dir, "test1.jsonl"), []byte(`{"type":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "test2.jsonl"), []byte(`{"type":"test2"}`), 0644)
	os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte(`not a jsonl`), 0644)

	// Create subdir with JSONL
	subdir := filepath.Join(dir, "subagents")
	os.MkdirAll(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "agent.jsonl"), []byte(`{"type":"test3"}`), 0644)

	w := New([]string{dir}, 5*time.Second, nil)
	files, err := w.InitialScan()
	if err != nil {
		t.Fatalf("InitialScan error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("got %d files, want 3", len(files))
	}
}

func TestSetOffset(t *testing.T) {
	w := New([]string{"/tmp"}, 5*time.Second, nil)
	w.SetOffset("/tmp/test.jsonl", 1024)

	w.mu.Lock()
	offset := w.offsets["/tmp/test.jsonl"]
	w.mu.Unlock()

	if offset != 1024 {
		t.Errorf("offset = %d, want 1024", offset)
	}
}

func TestPollDetectsChanges(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.jsonl")
	os.WriteFile(testFile, []byte(`{"line":1}`), 0644)

	var mu sync.Mutex
	var changes []FileChange

	w := New([]string{dir}, 100*time.Millisecond, func(c []FileChange) {
		mu.Lock()
		changes = append(changes, c...)
		mu.Unlock()
	})

	// Initial scan sets offset to 0
	w.InitialScan()

	// Start watcher
	w.Start()
	defer w.Stop()

	// Wait for poll to detect the file (offset 0, size > 0)
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	got := len(changes)
	mu.Unlock()

	if got == 0 {
		t.Error("expected at least one change detected")
	}
}
