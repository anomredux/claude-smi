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
	_ = os.WriteFile(filepath.Join(dir, "test1.jsonl"), []byte(`{"type":"test"}`), 0600)
	_ = os.WriteFile(filepath.Join(dir, "test2.jsonl"), []byte(`{"type":"test2"}`), 0600)
	_ = os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte(`not a jsonl`), 0600)

	// Create subdir with JSONL
	subdir := filepath.Join(dir, "subagents")
	_ = os.MkdirAll(subdir, 0750)
	_ = os.WriteFile(filepath.Join(subdir, "agent.jsonl"), []byte(`{"type":"test3"}`), 0600)

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
	_ = os.WriteFile(testFile, []byte(`{"line":1}`), 0600)

	var mu sync.Mutex
	var changes []FileChange

	w := New([]string{dir}, 100*time.Millisecond, func(c []FileChange) {
		mu.Lock()
		changes = append(changes, c...)
		mu.Unlock()
	})

	// Initial scan sets offset to 0
	_, _ = w.InitialScan()

	// Start watcher
	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
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
