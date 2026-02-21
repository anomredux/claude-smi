package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/anomredux/claude-smi/internal/config"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/parser"
	"github.com/anomredux/claude-smi/internal/pricing"
	"github.com/anomredux/claude-smi/internal/ui"
)

// version is set by goreleaser via ldflags.
var version = "dev"

// maxEntries limits the number of entries to prevent OOM on huge histories.
const maxEntries = 500_000

func main() {
	var (
		configPath  = flag.String("config", config.DefaultPath(), "config file path")
		dataDir     = flag.String("data-dir", defaultDataDir(), "Claude Code data directory")
		noTUI       = flag.Bool("no-tui", false, "output JSON to stdout instead of TUI")
		view        = flag.String("view", "daily", "view for --no-tui: daily, blocks")
		timezone    = flag.String("timezone", "", "override timezone (e.g., Asia/Seoul)")
		since       = flag.String("since", "", "filter entries from this date (YYYY-MM-DD)")
		until       = flag.String("until", "", "filter entries until this date (YYYY-MM-DD)")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println("claude-smi", version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI overrides
	if *timezone != "" {
		if _, err := time.LoadLocation(*timezone); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid timezone: %s\n", *timezone)
			os.Exit(1)
		}
		cfg.General.Timezone = *timezone
	}

	// Validate date filters
	for _, df := range []struct{ name, val string }{{"--since", *since}, {"--until", *until}} {
		if df.val != "" {
			if _, err := time.Parse("2006-01-02", df.val); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid %s date (use YYYY-MM-DD): %s\n", df.name, df.val)
				os.Exit(1)
			}
		}
	}

	if *noTUI {
		runNoTUI(cfg, *dataDir, *view, *since, *until)
		return
	}

	app := ui.NewApp(cfg)
	app.DataDir = *dataDir
	app.SinceFilter = *since
	app.UntilFilter = *until
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runNoTUI(cfg config.Config, dataDir, view, since, until string) {
	// Load timezone
	tz, err := time.LoadLocation(cfg.General.Timezone)
	if err != nil {
		tz = time.UTC
	}

	// Scan and parse all JSONL files
	entries := parser.ScanAndParse(dataDir)
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	// Dedup
	entries = parser.Dedup(entries)

	// Apply pricing: start with embedded defaults, overlay with LiteLLM
	table, err := pricing.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading pricing: %v\n", err)
		os.Exit(1)
	}
	if fetched, fetchErr := pricing.FetchLiteLLM(context.Background()); fetchErr == nil {
		table.Merge(fetched)
	}
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)
	calc.ApplyAll(entries)

	// Apply time range filter
	entries, err = domain.FilterByTimeRange(entries, since, until, tz)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date filter: %v\n", err)
		os.Exit(1)
	}

	var data interface{}
	switch view {
	case "daily":
		data = domain.AggregateDaily(entries, tz)
	case "blocks":
		data = domain.BuildBlocks(entries)
	default:
		fmt.Fprintf(os.Stderr, "Unknown view: %s (use daily or blocks)\n", view)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".claude", "projects")
	}
	return filepath.Join(home, ".claude", "projects")
}
