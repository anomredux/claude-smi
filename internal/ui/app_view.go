package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/theme"
	"github.com/anomredux/claude-smi/internal/ui/components"
)

func (a App) View() string {
	if !a.ready {
		return i18n.T("initializing")
	}

	if a.width < 80 || a.height < 24 {
		return lipgloss.Place(a.width, a.height,
			lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(theme.ColorPeach).Render(
				i18n.T("terminal_too_small")+"\n"+
					i18n.Tf("current_size", a.width, a.height),
			),
		)
	}

	if a.overlay != OverlayNone {
		overlay := a.renderOverlay()
		return lipgloss.Place(a.width, a.height,
			lipgloss.Center, lipgloss.Center,
			overlay,
			lipgloss.WithWhitespaceBackground(theme.ColorOverlayBg),
		)
	}

	compact := a.height < 30

	tabBar := a.renderTabs()
	statusBar := a.renderStatusBar()

	contentHeight := a.height - 4 // 2 tab + 2 status
	if contentHeight < 5 {
		contentHeight = 5
	}

	var content string
	if a.projectPicking {
		content = a.renderProjectPicker()
	} else {
		content = a.renderActiveView(contentHeight, compact)
	}

	content = lipgloss.NewStyle().
		Width(a.width).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(content)

	banner := a.notifications.RenderBanner(a.width)
	if banner != "" {
		return tabBar + "\n" + content + "\n" + banner
	}

	return tabBar + "\n" + content + "\n" + statusBar
}

func (a App) renderTabs() string {
	viewNames := []string{i18n.T("tab_live"), i18n.T("tab_blocks"), i18n.T("tab_daily_report")}

	var projectDisplay string
	if len(a.activeProjects) == 1 {
		for p := range a.activeProjects {
			projectDisplay = p
		}
	} else if len(a.activeProjects) > 1 {
		projectDisplay = fmt.Sprintf("%d projects", len(a.activeProjects))
	}

	return components.TabBar{
		ViewNames:     viewNames,
		ActiveIndex:   int(a.activeView),
		Width:         a.width,
		ActiveProject: projectDisplay,
	}.Render()
}

func (a App) renderActiveView(contentHeight int, compact bool) string {
	switch a.activeView {
	case ViewLive:
		return a.liveView.Render(a.width, contentHeight, compact)
	case ViewBlocks:
		return a.blocksView.Render(a.width, contentHeight, compact)
	case ViewDailyReport:
		return a.dailyReportView.Render(a.width, contentHeight, compact)
	}
	return ""
}

func (a App) renderStatusBar() string {
	return components.StatusBar{Width: a.width}.Render()
}

func (a *App) renderProjectPicker() string {
	innerW := a.width - 4

	titleLine := "  " + theme.AnimatedGradientText(i18n.T("select_project"), a.animTick)
	helpLine := lipgloss.PlaceHorizontal(innerW, lipgloss.Right,
		theme.MutedStyle.Render(i18n.T("picker_help")))

	var lines []string
	lines = append(lines, titleLine)
	lines = append(lines, helpLine)
	lines = append(lines, "")

	visibleRows := a.height - 10
	if visibleRows < 3 {
		visibleRows = 3
	}
	totalOptions := len(a.projects) + 1

	if a.projectCursor < a.projectScroll {
		a.projectScroll = a.projectCursor
	}
	if a.projectCursor >= a.projectScroll+visibleRows {
		a.projectScroll = a.projectCursor - visibleRows + 1
	}

	cursorStyle := lipgloss.NewStyle().Foreground(theme.ColorGold).Bold(true)
	normalStyle := theme.BodyStyle
	checkOn := lipgloss.NewStyle().Foreground(theme.ColorSkyBlue).Render("[x]")
	checkOff := theme.MutedStyle.Render("[ ]")

	for displayIdx := a.projectScroll; displayIdx < totalOptions && displayIdx < a.projectScroll+visibleRows; displayIdx++ {
		isCursor := displayIdx == a.projectCursor
		arrow := "  "
		if isCursor {
			arrow = cursorStyle.Render("> ")
		}

		if displayIdx == 0 {
			label := i18n.T("all_projects")
			check := checkOff
			if len(a.activeProjects) == 0 {
				check = checkOn
			}
			style := normalStyle
			if isCursor {
				style = cursorStyle
			}
			lines = append(lines, fmt.Sprintf("  %s%s %s", arrow, check, style.Render(label)))
		} else {
			p := a.projects[displayIdx-1]
			projectName := filepath.Base(p)
			parentDir := filepath.Base(filepath.Dir(p))
			label := parentDir + "/" + projectName
			check := checkOff
			if a.activeProjects[p] {
				check = checkOn
			}
			style := normalStyle
			if isCursor {
				style = cursorStyle
			}
			lines = append(lines, fmt.Sprintf("  %s%s %s", arrow, check, style.Render(label)))
		}
	}

	if totalOptions > visibleRows {
		lines = append(lines, "")
		lines = append(lines, theme.MutedStyle.Render(
			fmt.Sprintf("  [%d-%d / %d]", a.projectScroll+1, min(a.projectScroll+visibleRows, totalOptions), totalOptions)))
	}

	return strings.Join(lines, "\n")
}

func (a App) renderOverlay() string {
	switch a.overlay {
	case OverlayHelp:
		return a.helpOverlay.Render(a.width, a.height)
	case OverlaySettings:
		if a.settingsOverlay != nil {
			return a.settingsOverlay.Render(a.width, a.height)
		}
	}
	return theme.CardStyle.Width(60).Height(20).Render("Overlay")
}
