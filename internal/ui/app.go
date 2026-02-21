package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/anomredux/claude-smi/internal/api"
	"github.com/anomredux/claude-smi/internal/config"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/pricing"
	"github.com/anomredux/claude-smi/internal/ui/overlays"
	"github.com/anomredux/claude-smi/internal/ui/views"
)

type ViewType int

const (
	ViewLive ViewType = iota
	ViewBlocks
	ViewDailyReport
	ViewCount // sentinel: number of views
)

type OverlayType int

const (
	OverlayNone OverlayType = iota
	OverlayHelp
	OverlaySettings
)

// TickMsg triggers periodic data refresh.
type TickMsg time.Time

// BlinkMsg triggers UI-only refresh for smooth animation (250ms).
type BlinkMsg time.Time

// dataLoadedMsg carries freshly parsed data.
type dataLoadedMsg struct {
	entries []domain.UsageEntry
}

// apiUsageMsg carries usage data fetched from the OAuth API.
type apiUsageMsg struct {
	data *api.UsageData
	err  error
}

// pricingMsg carries dynamically fetched pricing data from LiteLLM.
type pricingMsg struct {
	table pricing.PricingTable
	err   error
}

type App struct {
	activeView ViewType
	overlay    OverlayType

	// Views
	liveView        *views.LiveView
	blocksView      *views.BlocksView
	dailyReportView *views.DailyReportView

	// Overlays
	helpOverlay     *overlays.HelpOverlay
	settingsOverlay *overlays.SettingsOverlay

	// Shared data
	entries         []domain.UsageEntry
	filteredEntries []domain.UsageEntry
	blocks          []domain.SessionBlock
	daily           []domain.DailyAggregate
	Config          config.Config
	calc            *pricing.Calculator
	tz              *time.Location
	apiUsage        *api.UsageData // from OAuth API

	// Animation state
	animTick uint

	// Project filter
	projects       []string        // available project paths
	activeProjects map[string]bool // selected projects; empty = all
	projectPicking bool            // project picker active
	projectCursor  int
	projectScroll  int

	// Notifications
	notifications *NotificationManager

	// Data
	DataDir     string
	SinceFilter string // YYYY-MM-DD
	UntilFilter string // YYYY-MM-DD

	// Terminal
	width  int
	height int

	// State
	loading bool
	ready   bool
}

func NewApp(cfg config.Config) App {
	i18n.SetLanguage(cfg.General.Language)

	tz, err := time.LoadLocation(cfg.General.Timezone)
	if err != nil {
		tz = time.UTC
	}

	table, _ := pricing.LoadDefault()
	if table == nil {
		table = make(pricing.PricingTable)
	}
	calc := pricing.NewCalculator(table, pricing.CostModeAuto)

	return App{
		activeView:     ViewLive,
		overlay:        OverlayNone,
		Config:         cfg,
		tz:             tz,
		calc:           calc,
		activeProjects: make(map[string]bool),
		liveView:       views.NewLiveView(tz, calc),
		blocksView:     views.NewBlocksView(tz),
		dailyReportView: views.NewDailyReportView(tz),
		helpOverlay:    overlays.NewHelpOverlay(),
		notifications:  NewNotificationManager(cfg.Notifications.Bell),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("claude-smi"),
		a.loadData,
		fetchApiUsage,
		fetchPricing,
		doBlink(),
	)
}

func doBlink() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return BlinkMsg(t)
	})
}

func doTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
