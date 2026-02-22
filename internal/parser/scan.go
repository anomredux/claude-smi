package parser

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/anomredux/claude-smi/internal/domain"
)

// Scanner abstracts data scanning for testing.
type Scanner interface {
	ScanAndParse(ctx context.Context, dataDir string) []domain.UsageEntry
}

// DefaultScanner implements Scanner using the real file system.
type DefaultScanner struct{}

// ScanAndParse walks the data directory, parses all .jsonl files,
// and returns the combined usage entries.
func (DefaultScanner) ScanAndParse(ctx context.Context, dataDir string) []domain.UsageEntry {
	return ScanAndParse(ctx, dataDir)
}

// ScanAndParse walks the data directory, parses all .jsonl files,
// and returns the combined usage entries.
func ScanAndParse(ctx context.Context, dataDir string) []domain.UsageEntry {
	// First pass: collect file paths using WalkDir (avoids unnecessary Stat calls)
	var paths []string
	_ = filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})

	// Pre-allocate with reasonable estimate (avg ~50 entries per file)
	all := make([]domain.UsageEntry, 0, len(paths)*50)

	for _, path := range paths {
		if ctx.Err() != nil {
			break
		}

		f, err := os.Open(path)
		if err != nil {
			continue
		}

		projectPath := filepath.Dir(path)
		result := ParseReader(f, projectPath)
		all = append(all, result.Entries...)
		f.Close()
	}

	return all
}

// FileChange describes a file that has changed since the last read.
type FileChange struct {
	Path   string
	Offset int64
}

// ParseIncremental reads only the new data from each changed file (from the
// given offset) and returns the new entries along with updated offsets.
func ParseIncremental(ctx context.Context, changes []FileChange) (entries []domain.UsageEntry, newOffsets map[string]int64) {
	newOffsets = make(map[string]int64, len(changes))

	for _, fc := range changes {
		if ctx.Err() != nil {
			break
		}

		f, err := os.Open(fc.Path)
		if err != nil {
			continue
		}

		// Seek to the last known offset
		if fc.Offset > 0 {
			if _, err := f.Seek(fc.Offset, io.SeekStart); err != nil {
				f.Close()
				continue
			}
		}

		projectPath := filepath.Dir(fc.Path)
		result := ParseReader(f, projectPath)
		entries = append(entries, result.Entries...)

		// Record the new offset
		pos, err := f.Seek(0, io.SeekCurrent)
		if err == nil {
			newOffsets[fc.Path] = pos
		}
		f.Close()
	}

	return entries, newOffsets
}
