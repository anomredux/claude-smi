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
		all = append(all, parseFileEntries(path)...)
	}

	return all
}

// parseFileEntries opens a .jsonl file and returns its parsed entries.
func parseFileEntries(path string) []domain.UsageEntry {
	f, err := os.Open(path) //nolint:gosec // G304: path from controlled directory walk
	if err != nil {
		return nil
	}
	defer f.Close()
	return ParseReader(f, filepath.Dir(path)).Entries
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
		e, pos, ok := parseFileFrom(fc)
		entries = append(entries, e...)
		if ok {
			newOffsets[fc.Path] = pos
		}
	}

	return entries, newOffsets
}

// parseFileFrom reads new entries from a file starting at the given offset.
// Returns the entries, the new file offset, and whether the offset is valid.
func parseFileFrom(fc FileChange) ([]domain.UsageEntry, int64, bool) {
	f, err := os.Open(fc.Path) //nolint:gosec // G304: path from controlled file watch
	if err != nil {
		return nil, 0, false
	}
	defer f.Close()

	if fc.Offset > 0 {
		if _, err := f.Seek(fc.Offset, io.SeekStart); err != nil {
			return nil, 0, false
		}
	}

	result := ParseReader(f, filepath.Dir(fc.Path))

	pos, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return result.Entries, 0, false
	}
	return result.Entries, pos, true
}
