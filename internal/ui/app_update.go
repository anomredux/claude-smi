package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anomredux/claude-smi/internal/animation"
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
				a.setScrollTarget(a.scrollTarget() - 3)
				return a, a.ensureAnimRunning()
			case tea.MouseButtonWheelDown:
				a.setScrollTarget(a.scrollTarget() + 3)
				return a, a.ensureAnimRunning()
			}
		}
		return a, nil

	case tea.KeyMsg:
		if a.overlay != OverlayNone {
			return a.updateOverlay(msg)
		}
		return a.handleGlobalKey(msg)

	case AnimFrameMsg:
		a.animator.Update()
		if a.animator.Settled() {
			a.animRunning = false
			return a, nil
		}
		return a, doAnimFrame()

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
		return a, a.ensureAnimRunning()

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
		return a, a.ensureAnimRunning()

	case apiUsageMsg:
		if msg.err != nil {
			a.notifications.SetMessage("API: " + msg.err.Error())
		} else if msg.data != nil {
			a.apiUsage = msg.data
			a.liveView.SetApiUsage(msg.data)
			return a, a.ensureAnimRunning()
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
		a.liveView.ResetAnimations()
		a.liveView = views.NewLiveView(a.tz, a.calc, a.animator)
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
		return a, a.ensureAnimRunning()
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "1":
		a.activeView = ViewLive
		a.snapScrollSpring()
	case "2":
		a.activeView = ViewBlocks
		a.snapScrollSpring()
	case "3":
		a.activeView = ViewDailyReport
		a.snapScrollSpring()
	case "tab":
		a.activeView = (a.activeView + 1) % ViewCount
		a.snapScrollSpring()
	case "shift+tab":
		a.activeView = (a.activeView + ViewCount - 1) % ViewCount
		a.snapScrollSpring()
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

// scrollTarget returns the current scroll target for the active view.
func (a *App) scrollTarget() int {
	return a.scroll.viewScrollY[a.activeView]
}

// setScrollTarget clamps and sets a new scroll target, driving the spring.
func (a *App) setScrollTarget(target int) {
	maxOffset := int(a.scroll.lastContentLines.Load()) - a.contentHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if target < 0 {
		target = 0
	}
	if target > maxOffset {
		target = maxOffset
	}
	a.scroll.viewScrollY[a.activeView] = target
	key := animation.ScrollKey(int(a.activeView))
	a.animator.SetWithPreset(key, float64(target), animation.PresetSnap)
}

// snapScrollSpring sets the scroll spring to current offset without animation.
func (a *App) snapScrollSpring() {
	key := animation.ScrollKey(int(a.activeView))
	a.animator.SetSnap(key, float64(a.scroll.viewScrollY[a.activeView]))
}

// handleScrollKey processes scroll-related key events. Returns true if consumed.
func (a *App) handleScrollKey(msg tea.KeyMsg) bool {
	contentHeight := a.contentHeight()
	pageSize := contentHeight - 2
	if pageSize < 1 {
		pageSize = 1
	}

	target := a.scrollTarget()

	switch msg.String() {
	case "j", "down":
		a.setScrollTarget(target + 1)
		return true
	case "k", "up":
		a.setScrollTarget(target - 1)
		return true
	case "pgdown", " ":
		a.setScrollTarget(target + pageSize)
		return true
	case "pgup":
		a.setScrollTarget(target - pageSize)
		return true
	case "home", "g":
		a.setScrollTarget(0)
		return true
	case "end", "G":
		a.setScrollTarget(int(a.scroll.lastContentLines.Load()))
		return true
	}
	return false
}
