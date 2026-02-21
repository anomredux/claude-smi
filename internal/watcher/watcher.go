package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileChange struct {
	Path   string
	Offset int64 // read from this offset
}

type Watcher struct {
	dirs         []string
	offsets      map[string]int64 // path -> last read offset
	mu           sync.Mutex
	pollInterval time.Duration
	onChange     func([]FileChange)
	stop         chan struct{}
	wg           sync.WaitGroup
}

func New(dirs []string, pollInterval time.Duration, onChange func([]FileChange)) *Watcher {
	return &Watcher{
		dirs:         dirs,
		offsets:      make(map[string]int64),
		pollInterval: pollInterval,
		onChange:     onChange,
		stop:         make(chan struct{}),
	}
}

// InitialScan finds all JSONL files and returns them with offset 0.
func (w *Watcher) InitialScan() ([]string, error) {
	var files []string
	for _, dir := range w.dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if !info.IsDir() && filepath.Ext(path) == ".jsonl" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Single lock acquisition to register all offsets
	w.mu.Lock()
	for _, f := range files {
		w.offsets[f] = 0
	}
	w.mu.Unlock()

	return files, nil
}

// SetOffset records that a file has been read up to this offset.
func (w *Watcher) SetOffset(path string, offset int64) {
	w.mu.Lock()
	w.offsets[path] = offset
	w.mu.Unlock()
}

// Start begins watching with fsnotify + polling fallback.
func (w *Watcher) Start() error {
	// Try fsnotify first
	fsw, err := fsnotify.NewWatcher()
	if err == nil {
		for _, dir := range w.dirs {
			_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err == nil && info.IsDir() {
					_ = fsw.Add(path)
				}
				return nil
			})
		}

		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for {
				select {
				case event, ok := <-fsw.Events:
					if !ok {
						return
					}
					if filepath.Ext(event.Name) == ".jsonl" &&
						(event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0) {
						w.checkFile(event.Name)
					}
				case <-w.stop:
					fsw.Close()
					return
				}
			}
		}()
	}

	// Polling fallback (always runs as safety net)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.pollAll()
			case <-w.stop:
				return
			}
		}
	}()

	return nil
}

// Stop signals goroutines to exit and waits for them to finish.
func (w *Watcher) Stop() {
	close(w.stop)
	w.wg.Wait()
}

func (w *Watcher) checkFile(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	w.mu.Lock()
	lastOffset, known := w.offsets[path]
	if !known {
		w.offsets[path] = 0
		lastOffset = 0
	}
	w.mu.Unlock()

	if info.Size() > lastOffset {
		w.onChange([]FileChange{{Path: path, Offset: lastOffset}})
	}
}

func (w *Watcher) pollAll() {
	// Collect file info without holding the lock
	type fileInfo struct {
		path string
		size int64
	}
	var files []fileInfo
	for _, dir := range w.dirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || filepath.Ext(path) != ".jsonl" {
				return nil
			}
			files = append(files, fileInfo{path: path, size: info.Size()})
			return nil
		})
	}

	// Single lock acquisition to check all offsets
	w.mu.Lock()
	var changes []FileChange
	for _, f := range files {
		lastOffset, known := w.offsets[f.path]
		if !known {
			w.offsets[f.path] = 0
			lastOffset = 0
		}
		if f.size > lastOffset {
			changes = append(changes, FileChange{Path: f.path, Offset: lastOffset})
		}
	}
	w.mu.Unlock()

	if len(changes) > 0 {
		w.onChange(changes)
	}
}
