package ui

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/anomredux/claude-smi/internal/api"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/parser"
	"github.com/anomredux/claude-smi/internal/pricing"
)

// loadData performs a full scan of all JSONL files and records file sizes
// as offsets for subsequent incremental loads.
func (a App) loadData() tea.Msg {
	dataDir := a.DataDir
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return dataLoadedMsg{}
		}
		dataDir = filepath.Join(home, ".claude", "projects")
	}

	ctx := context.Background()
	entries := parser.ScanAndParse(ctx, dataDir)

	// Record file sizes as offsets for incremental parsing
	offsets := make(map[string]int64)
	_ = filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if info, err := d.Info(); err == nil {
			offsets[path] = info.Size()
		}
		return nil
	})

	return dataLoadedMsg{entries: entries, offsets: offsets}
}

// loadIncremental scans for files that have grown since the last read
// and parses only the new data.
func (a App) loadIncremental() tea.Msg {
	dataDir := a.DataDir
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return incrementalLoadedMsg{}
		}
		dataDir = filepath.Join(home, ".claude", "projects")
	}

	// Build a list of changed files
	a.fileOffsetsMu.Lock()
	currentOffsets := make(map[string]int64, len(a.fileOffsets))
	for k, v := range a.fileOffsets {
		currentOffsets[k] = v
	}
	a.fileOffsetsMu.Unlock()

	var changes []parser.FileChange
	_ = filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		lastOffset, known := currentOffsets[path]
		if !known || info.Size() > lastOffset {
			changes = append(changes, parser.FileChange{
				Path:   path,
				Offset: lastOffset,
			})
		}
		return nil
	})

	if len(changes) == 0 {
		return incrementalLoadedMsg{}
	}

	ctx := context.Background()
	entries, newOffsets := parser.ParseIncremental(ctx, changes)
	return incrementalLoadedMsg{entries: entries, offsets: newOffsets}
}

func fetchApiUsage() tea.Msg {
	ctx := context.Background()
	data, err := api.FetchUsage(ctx)
	return apiUsageMsg{data: data, err: err}
}

func fetchPricing() tea.Msg {
	ctx := context.Background()
	table, err := pricing.FetchLiteLLM(ctx)
	return pricingMsg{table: table, err: err}
}

func (a *App) processData(entries []domain.UsageEntry) {
	entries = parser.Dedup(entries)
	a.calc.ApplyAll(entries)

	if timeFiltered, err := domain.FilterByTimeRange(entries, a.SinceFilter, a.UntilFilter, a.tz); err == nil {
		entries = timeFiltered
	}

	a.entries = entries

	// Extract unique project paths
	projectSet := make(map[string]struct{})
	for _, e := range entries {
		if e.ProjectPath != "" {
			projectSet[e.ProjectPath] = struct{}{}
		}
	}
	a.projects = make([]string, 0, len(projectSet))
	for p := range projectSet {
		a.projects = append(a.projects, p)
	}
	sort.Strings(a.projects)

	// Apply project filter
	filtered := entries
	if len(a.activeProjects) > 0 {
		filtered = make([]domain.UsageEntry, 0)
		for _, e := range entries {
			if a.activeProjects[e.ProjectPath] {
				filtered = append(filtered, e)
			}
		}
	}
	a.filteredEntries = filtered

	a.blocks = domain.BuildBlocks(filtered)
	a.daily = domain.AggregateDaily(filtered, a.tz)

	// Update views
	a.liveView.SetData(filtered, a.blocks, a.daily)
	if a.apiUsage != nil {
		a.liveView.SetApiUsage(a.apiUsage)
	}
	a.blocksView.SetData(a.blocks)
	a.dailyReportView.SetData(filtered)

	a.loading = false
}
