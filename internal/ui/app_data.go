package ui

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/anomredux/claude-smi/internal/api"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/parser"
	"github.com/anomredux/claude-smi/internal/pricing"
)

func (a App) loadData() tea.Msg {
	dataDir := a.DataDir
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return dataLoadedMsg{}
		}
		dataDir = filepath.Join(home, ".claude", "projects")
	}

	entries := parser.ScanAndParse(dataDir)
	return dataLoadedMsg{entries: entries}
}

func fetchApiUsage() tea.Msg {
	data, err := api.FetchUsage()
	return apiUsageMsg{data: data, err: err}
}

func fetchPricing() tea.Msg {
	table, err := pricing.FetchLiteLLM(context.Background())
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
