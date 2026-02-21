package parser

import (
	"os"
	"path/filepath"

	"github.com/anomredux/claude-smi/internal/domain"
)

// ScanAndParse walks the data directory, parses all .jsonl files,
// and returns the combined usage entries.
func ScanAndParse(dataDir string) []domain.UsageEntry {
	// First pass: collect file paths
	var paths []string
	_ = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})

	// Pre-allocate with reasonable estimate (avg ~50 entries per file)
	all := make([]domain.UsageEntry, 0, len(paths)*50)

	for _, path := range paths {
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
