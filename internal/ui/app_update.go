package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/anomredux/claude-smi/internal/config"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/pricing"
	"github.com/anomredux/claude-smi/internal/ui/overlays"
	"github.com/anomredux/claude-smi/internal/ui/views"
)

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if !a.ready {
			a.ready = true
			return a, doTick(time.Duration(a.Config.General.Interval) * time.Second)
		}
		return a, nil

	case tea.MouseMsg:
		if a.overlay == OverlayNone && !a.projectPicking {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				a.scroll.viewScrollY[a.activeView] -= 3
				if a.scroll.viewScrollY[a.activeView] < 0 {
					a.scroll.viewScrollY[a.activeView] = 0
				}
			case tea.MouseButtonWheelDown:
				a.scroll.viewScrollY[a.activeView] += 3
				a.clampScrollY()
			}
		}
		return a, nil

	case tea.KeyMsg:
		if a.overlay != OverlayNone {
			return a.updateOverlay(msg)
		}
		return a.handleGlobalKey(msg)

	case BlinkMsg:
		a.animTick++
		a.propagateAnimTick()
		return a, doBlink()

	case TickMsg:
		a.notifications.Expire()
		var loadCmd tea.Cmd
		if a.initialLoaded {
			loadCmd = a.loadIncremental
		} else {
			loadCmd = a.loadData
		}
		return a, tea.Batch(
			loadCmd,
			fetchApiUsage,
			doTick(time.Duration(a.Config.General.Interval)*time.Second),
		)

	case dataLoadedMsg:
		// Store offsets from full scan
		if msg.offsets != nil {
			a.fileOffsetsMu.Lock()
			a.fileOffsets = msg.offsets
			a.fileOffsetsMu.Unlock()
		}
		a.initialLoaded = true
		a.processData(msg.entries)
		return a, nil

	case incrementalLoadedMsg:
		if len(msg.entries) > 0 {
			// Update offsets
			a.fileOffsetsMu.Lock()
			for path, offset := range msg.offsets {
				a.fileOffsets[path] = offset
			}
			a.fileOffsetsMu.Unlock()

			// Merge new entries with existing and reprocess
			merged := make([]domain.UsageEntry, 0, len(a.entries)+len(msg.entries))
			merged = append(merged, a.entries...)
			merged = append(merged, msg.entries...)
			a.processData(merged)
		}
		return a, nil

	case apiUsageMsg:
		if msg.err != nil {
			a.notifications.SetMessage("API: " + msg.err.Error())
		} else if msg.data != nil {
			a.apiUsage = msg.data
			a.liveView.SetApiUsage(msg.data)
		}
		return a, nil

	case pricingMsg:
		if msg.err != nil {
			a.notifications.SetMessage("Pricing: " + msg.err.Error())
		} else if msg.table != nil {
			baseTable, _ := pricing.LoadDefault()
			if baseTable == nil {
				baseTable = make(pricing.PricingTable)
			}
			baseTable.Merge(msg.table)
			a.calc.UpdateTable(baseTable)
			a.processData(a.entries)
		}
		return a, nil

	case overlays.ConfigChangedMsg:
		a.Config = msg.Config
		i18n.SetLanguage(a.Config.General.Language)
		newTz, err := time.LoadLocation(a.Config.General.Timezone)
		if err == nil {
			a.tz = newTz
		}
		a.liveView = views.NewLiveView(a.tz, a.calc)
		a.blocksView = views.NewBlocksView(a.tz)
		a.dailyReportView = views.NewDailyReportView(a.tz)
		a.processData(a.entries)
		return a, nil
	}

	return a, nil
}

func (a App) handleGlobalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.projectPicking {
		return a.handleProjectPicker(msg)
	}

	var cmd tea.Cmd
	switch a.activeView {
	case ViewLive:
		cmd = a.liveView.Update(msg)
	case ViewBlocks:
		cmd = a.blocksView.Update(msg)
		if a.blocksView.ScrollReset {
			a.scroll.viewScrollY[a.activeView] = 0
			a.blocksView.ScrollReset = false
		}
	case ViewDailyReport:
		cmd = a.dailyReportView.Update(msg)
	}
	if cmd != nil {
		return a, cmd
	}

	// App-level scroll handling (keys not consumed by view)
	if a.handleScrollKey(msg) {
		return a, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "1":
		a.activeView = ViewLive
	case "2":
		a.activeView = ViewBlocks
	case "3":
		a.activeView = ViewDailyReport
	case "tab":
		a.activeView = (a.activeView + 1) % ViewCount
	case "shift+tab":
		a.activeView = (a.activeView + ViewCount - 1) % ViewCount
	case "?":
		a.overlay = OverlayHelp
	case "s":
		a.settingsOverlay = overlays.NewSettingsOverlay(a.Config, config.DefaultPath())
		a.overlay = OverlaySettings
	case "r":
		a.loading = true
		a.initialLoaded = false // force full reload
		return a, a.loadData
	case "p":
		if len(a.projects) > 0 {
			a.projectPicking = true
			a.projectCursor = 0
		}
	}
	return a, nil
}

func (a App) handleProjectPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	totalOptions := len(a.projects) + 1

	switch msg.String() {
	case "esc":
		a.projectPicking = false
		a.processData(a.entries)
	case "j", "down":
		a.projectCursor++
		if a.projectCursor >= totalOptions {
			a.projectCursor = 0
		}
	case "k", "up":
		a.projectCursor--
		if a.projectCursor < 0 {
			a.projectCursor = totalOptions - 1
		}
	case "enter", " ":
		if a.projectCursor == 0 {
			a.activeProjects = make(map[string]bool)
		} else {
			p := a.projects[a.projectCursor-1]
			if a.activeProjects[p] {
				delete(a.activeProjects, p)
			} else {
				a.activeProjects[p] = true
			}
		}
		a.processData(a.entries)
	}
	return a, nil
}

func (a App) updateOverlay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.overlay {
	case OverlayHelp:
		switch msg.String() {
		case "esc", "?":
			a.overlay = OverlayNone
		}
	case OverlaySettings:
		if a.settingsOverlay != nil {
			closed, cmd := a.settingsOverlay.Update(msg)
			if closed {
				a.overlay = OverlayNone
			}
			return a, cmd
		}
	}
	return a, nil
}

// handleScrollKey processes scroll-related key events. Returns true if consumed.
func (a *App) handleScrollKey(msg tea.KeyMsg) bool {
	contentHeight := a.contentHeight()
	pageSize := contentHeight - 2
	if pageSize < 1 {
		pageSize = 1
	}

	switch msg.String() {
	case "j", "down":
		a.scroll.viewScrollY[a.activeView]++
		a.clampScrollY()
		return true
	case "k", "up":
		a.scroll.viewScrollY[a.activeView]--
		if a.scroll.viewScrollY[a.activeView] < 0 {
			a.scroll.viewScrollY[a.activeView] = 0
		}
		return true
	case "pgdown", " ":
		a.scroll.viewScrollY[a.activeView] += pageSize
		a.clampScrollY()
		return true
	case "pgup":
		a.scroll.viewScrollY[a.activeView] -= pageSize
		if a.scroll.viewScrollY[a.activeView] < 0 {
			a.scroll.viewScrollY[a.activeView] = 0
		}
		return true
	case "home", "g":
		a.scroll.viewScrollY[a.activeView] = 0
		return true
	case "end", "G":
		a.scroll.viewScrollY[a.activeView] = a.scroll.lastContentLines
		a.clampScrollY()
		return true
	}
	return false
}

// clampScrollY ensures the current view's scroll offset does not exceed
// the maximum based on lastContentLines from the previous render.
func (a *App) clampScrollY() {
	maxOffset := a.scroll.lastContentLines - a.contentHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if a.scroll.viewScrollY[a.activeView] > maxOffset {
		a.scroll.viewScrollY[a.activeView] = maxOffset
	}
}

func (a *App) propagateAnimTick() {
	a.liveView.AnimTick = a.animTick
	a.blocksView.AnimTick = a.animTick
	a.dailyReportView.AnimTick = a.animTick
	a.helpOverlay.AnimTick = a.animTick
	if a.settingsOverlay != nil {
		a.settingsOverlay.SetAnimTick(a.animTick)
	}
}
