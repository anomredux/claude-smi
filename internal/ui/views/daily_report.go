package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/theme"
	"github.com/anomredux/claude-smi/internal/ui/components"
)

// Package-level cached styles for daily report view rendering.
var (
	colorWeekendFaded = theme.ColorWeekendFaded

	// Per-column colors: Sun Mon Tue Wed Thu Fri Sat
	dayColumnColors = [7]lipgloss.Color{
		colorWeekendFaded,  // Sun
		theme.ColorSkyBlue, // Mon
		theme.ColorLavender, // Tue
		theme.ColorMauve,   // Wed
		theme.ColorPeach,   // Thu
		theme.ColorGold,    // Fri
		colorWeekendFaded,  // Sat
	}
)

type DailyReportView struct {
	entries  []domain.UsageEntry
	tz       *time.Location
	year     int
	month    time.Month
	AnimTick uint
}

func NewDailyReportView(tz *time.Location) *DailyReportView {
	now := time.Now().In(tz)
	return &DailyReportView{tz: tz, year: now.Year(), month: now.Month()}
}

func (v *DailyReportView) SetData(entries []domain.UsageEntry) {
	v.entries = entries
}

func (v *DailyReportView) Update(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "left", "h":
			v.month--
			if v.month < time.January {
				v.month = time.December
				v.year--
			}
			return KeyHandledCmd
		case "right", "l":
			v.month++
			if v.month > time.December {
				v.month = time.January
				v.year++
			}
			return KeyHandledCmd
		}
	}
	return nil
}

func (v *DailyReportView) Render(width, height int, compact bool) string {
	agg := domain.AggregateMonthly(v.entries, v.tz, v.year, v.month)
	cardWidth := width - 4

	card := components.Card{
		Title:   theme.AnimatedGradientText(fmt.Sprintf("%s %d", v.month.String(), v.year), v.AnimTick),
		Width:   cardWidth,
		Compact: compact,
	}

	innerW := card.InnerWidth()
	var sections []string

	// 5 stat cards: Cost, Input, Output, Cache W, Cache R
	statGap := 2
	statW := (innerW - statGap*4) / 5
	if statW < 10 {
		statW = 10
	}

	stats := []components.StatCard{
		{Value: fmt.Sprintf("$%.2f", agg.TotalCost), Label: i18n.T("cost"), Width: statW, Color: theme.ColorSkyBlue},
		{Value: components.FormatCompact(agg.TotalInputTokens), Label: i18n.T("input_tokens"), Width: statW, Color: theme.ColorLavender},
		{Value: components.FormatCompact(agg.TotalOutputTokens), Label: i18n.T("output_tokens"), Width: statW, Color: theme.ColorMauve},
		{Value: components.FormatCompact(agg.TotalCacheRead), Label: i18n.T("cache_read"), Width: statW, Color: theme.ColorPeach},
		{Value: components.FormatCompact(agg.TotalCacheCreation), Label: i18n.T("cache_create"), Width: statW, Color: theme.ColorGold},
	}

	sections = append(sections, components.CenterBlock(components.RenderStatRow(stats, statGap), innerW))
	sections = append(sections, "")

	sections = append(sections, v.renderCalendar(agg, card.InnerWidth()))

	card.Content = strings.Join(sections, "\n")
	return card.Render()
}

func (v *DailyReportView) renderCalendar(agg domain.MonthlyAggregate, innerWidth int) string {
	cellWidth := innerWidth / 7
	if cellWidth < 10 {
		cellWidth = 10
	}
	if cellWidth > 18 {
		cellWidth = 18
	}

	dayNames := []string{
		i18n.T("day_sun"), i18n.T("day_mon"), i18n.T("day_tue"), i18n.T("day_wed"),
		i18n.T("day_thu"), i18n.T("day_fri"), i18n.T("day_sat"),
	}
	var headerCells []string
	for i, d := range dayNames {
		style := lipgloss.NewStyle().Width(cellWidth).Align(lipgloss.Center)
		if i == 0 || i == 6 { // Sun, Sat
			style = style.Foreground(theme.ColorWeekendRed)
		} else {
			style = style.Foreground(theme.ColorMutedText)
		}
		headerCells = append(headerCells, style.Render(d))
	}
	header := lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, strings.Join(headerCells, ""))

	firstDay := time.Date(v.year, v.month, 1, 0, 0, 0, 0, v.tz)
	daysInMonth := time.Date(v.year, v.month+1, 0, 0, 0, 0, 0, v.tz).Day()

	weekday := int(firstDay.Weekday()) // Sunday=0, Monday=1, ..., Saturday=6

	var rows []string
	rows = append(rows, header)

	var currentWeek []int
	for i := 0; i < weekday; i++ {
		currentWeek = append(currentWeek, 0)
	}

	for day := 1; day <= daysInMonth; day++ {
		currentWeek = append(currentWeek, day)

		if len(currentWeek) == 7 {
			rows = append(rows, v.renderWeekRow(currentWeek, agg, cellWidth, innerWidth))
			currentWeek = nil
		}
	}

	if len(currentWeek) > 0 {
		for len(currentWeek) < 7 {
			currentWeek = append(currentWeek, 0)
		}
		rows = append(rows, v.renderWeekRow(currentWeek, agg, cellWidth, innerWidth))
	}

	return strings.Join(rows, "\n")
}

// fmtCalToken formats a token count for the calendar cell with aligned suffix and label.
// Number+suffix is right-aligned, then " (label)" with ( at a fixed column.
// e.g. " 12.3K (I )", "  1.2M (O )", "  523  (CR)", " 52.3M (CW)"
func fmtCalToken(n int, label string) string {
	var numPart, suffix string
	if n < 1000 {
		numPart = fmt.Sprintf("%d", n)
		suffix = " " // space placeholder
	} else if n < 1_000_000 {
		numPart = fmt.Sprintf("%.1f", float64(n)/1000)
		suffix = "K"
	} else {
		numPart = fmt.Sprintf("%.1f", float64(n)/1_000_000)
		suffix = "M"
	}
	return fmt.Sprintf("%5s%s (%s)", numPart, suffix, label)
}

// isWeekend returns true if the column index is Sunday (0) or Saturday (6).
func isWeekend(colIdx int) bool {
	return colIdx == 0 || colIdx == 6
}

// renderWeekRow renders a week as 7 lines: day number, input, output, cache read, cache write, cost, blank spacer.
func (v *DailyReportView) renderWeekRow(week []int, agg domain.MonthlyAggregate, cellWidth, innerWidth int) string {
	base := lipgloss.NewStyle().Width(cellWidth).Align(lipgloss.Center)
	blank := base.Render("")

	var line1, line2, line3, line4, line5, line6, line7 []string

	for col, day := range week {
		if day == 0 {
			line1 = append(line1, blank)
			line2 = append(line2, blank)
			line3 = append(line3, blank)
			line4 = append(line4, blank)
			line5 = append(line5, blank)
			line6 = append(line6, blank)
			line7 = append(line7, blank)
			continue
		}

		d := agg.Days[day]
		dayStr := fmt.Sprintf("%d", day)
		weekend := isWeekend(col)

		if d.EntriesCount == 0 {
			dayStyle := base.Foreground(theme.ColorMutedText)
			if weekend {
				dayStyle = base.Foreground(theme.ColorWeekendRed)
			}
			line1 = append(line1, dayStyle.Render(dayStr))
			line2 = append(line2, blank)
			line3 = append(line3, blank)
			line4 = append(line4, blank)
			line5 = append(line5, blank)
			line6 = append(line6, blank)
			line7 = append(line7, blank)
			continue
		}

		// Day number background: weekday = colormap color, weekend = faded white
		bgColor := dayColumnColors[col]

		dayFg := theme.ColorBaseBg
		if weekend {
			dayFg = theme.ColorWeekendRed
		}
		line1 = append(line1, base.Background(bgColor).Foreground(dayFg).Render(dayStr))
		ds := base.Foreground(theme.ColorBodyText)
		line2 = append(line2, ds.Render(fmtCalToken(d.InputTokens, "I")))
		line3 = append(line3, ds.Render(fmtCalToken(d.OutputTokens, "O")))
		line4 = append(line4, ds.Render(fmtCalToken(d.CacheReadTokens, "CR")))
		line5 = append(line5, ds.Render(fmtCalToken(d.CacheCreationTokens, "CW")))
		line6 = append(line6, ds.Render(fmt.Sprintf("$%.2f", d.TotalCost)))
		line7 = append(line7, blank)
	}

	place := func(cells []string) string {
		return lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, strings.Join(cells, ""))
	}

	return place(line1) + "\n" + place(line2) + "\n" + place(line3) + "\n" + place(line4) + "\n" + place(line5) + "\n" + place(line6) + "\n" + place(line7)
}
