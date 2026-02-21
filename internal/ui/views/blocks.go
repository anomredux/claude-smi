package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/theme"
	"github.com/anomredux/claude-smi/internal/ui/components"
)

type BlocksView struct {
	blocks       []domain.SessionBlock
	tz           *time.Location
	cursor       int
	detail       bool
	scroll       int
	detailScroll int
	AnimTick     uint
}

func NewBlocksView(tz *time.Location) *BlocksView {
	return &BlocksView{tz: tz}
}

func (v *BlocksView) SetData(blocks []domain.SessionBlock) {
	v.blocks = blocks
	if v.cursor >= len(blocks) {
		v.cursor = max(0, len(blocks)-1)
	}
}

func (v *BlocksView) Update(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "j", "down":
			if v.detail {
				v.detailScroll++
			} else if v.cursor < len(v.blocks)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.detail {
				if v.detailScroll > 0 {
					v.detailScroll--
				}
			} else if v.cursor > 0 {
				v.cursor--
			}
		case "enter":
			if !v.detail && len(v.blocks) > 0 {
				v.detail = true
				v.detailScroll = 0
			}
		case "esc", "backspace":
			if v.detail {
				v.detail = false
			}
		}
	}
	return nil
}

func (v *BlocksView) Render(width, height int, compact bool) string {
	cardWidth := width - 4

	if len(v.blocks) == 0 {
		card := components.Card{
			Title:   theme.AnimatedGradientText(i18n.T("session_blocks"), v.AnimTick),
			Width:   cardWidth,
			Compact: compact,
		}
		card.Content = theme.MutedStyle.Render(i18n.T("no_blocks_found"))
		return card.Render()
	}

	if v.detail {
		return v.renderDetail(cardWidth, compact)
	}
	return v.renderList(cardWidth, height, compact)
}

func (v *BlocksView) renderList(cardWidth, contentHeight int, compact bool) string {
	title := fmt.Sprintf("%s (%d)", i18n.T("session_blocks"), len(v.blocks))
	card := components.Card{
		Title:   theme.AnimatedGradientText(title, v.AnimTick),
		Width:   cardWidth,
		Compact: compact,
	}

	innerW := card.InnerWidth()

	// Column layout — distribute widths proportionally
	type colDef struct {
		header string
		width  int
		align  lipgloss.Position
	}

	// Fixed columns with total ~88 chars; innerW typically 90+ for 100-char terminals
	cols := []colDef{
		{"#", 3, lipgloss.Right},
		{i18n.T("start"), 13, lipgloss.Left},
		{i18n.T("end"), 13, lipgloss.Left},
		{i18n.T("input_tokens"), 8, lipgloss.Right},
		{i18n.T("output_tokens"), 8, lipgloss.Right},
		{i18n.T("cache_read"), 8, lipgloss.Right},
		{i18n.T("cache_create"), 8, lipgloss.Right},
		{i18n.T("cost"), 9, lipgloss.Right},
		{i18n.T("status"), 7, lipgloss.Right},
	}

	// Adjust widths if we have extra space — give it to Start/End
	totalFixed := 0
	for _, c := range cols {
		totalFixed += c.width
	}
	gaps := len(cols) - 1 // 1 space between each column
	remaining := innerW - totalFixed - gaps
	if remaining > 0 {
		// Distribute extra space to Start and End columns
		extra := remaining / 2
		cols[1].width += extra
		cols[2].width += remaining - extra
	}

	cellStyle := func(width int, align lipgloss.Position, highlighted bool) lipgloss.Style {
		s := lipgloss.NewStyle().Width(width).Align(align)
		if highlighted {
			s = s.Background(theme.ColorElevatedBg)
		}
		return s
	}

	// Help text right-aligned below title
	helpLine := lipgloss.PlaceHorizontal(innerW, lipgloss.Right,
		theme.MutedStyle.Render(i18n.T("blocks_help")))

	// Per-column header colors matching data cell colors
	headerColors := []lipgloss.Color{
		theme.ColorBrightText, // #
		theme.ColorBrightText, // Start
		theme.ColorBrightText, // End
		theme.ColorLavender,   // Input
		theme.ColorMauve,      // Output
		theme.ColorPeach,      // Cache R
		theme.ColorGold,       // Cache W
		theme.ColorSkyBlue,    // Cost
		theme.ColorBrightText, // Status
	}

	// Render header
	var headerCells []string
	for i, c := range cols {
		s := cellStyle(c.width, c.align, false).Foreground(headerColors[i]).Bold(true)
		headerCells = append(headerCells, s.Render(c.header))
	}
	headerLine := strings.Join(headerCells, " ")

	// Separator
	sepWidth := 0
	for _, c := range cols {
		sepWidth += c.width
	}
	sepWidth += gaps
	separator := theme.MutedStyle.Render(strings.Repeat("─", sepWidth))

	var rows []string
	rows = append(rows, helpLine)
	rows = append(rows, headerLine)
	rows = append(rows, separator)

	// Compute max visible rows for scrolling
	visibleRows := contentHeight - 8 // card border + header + separator + footer
	if visibleRows < 3 {
		visibleRows = 3
	}

	// Scrolling: keep cursor in view
	if v.cursor < v.scroll {
		v.scroll = v.cursor
	}
	if v.cursor >= v.scroll+visibleRows {
		v.scroll = v.cursor - visibleRows + 1
	}

	for displayIdx := 0; displayIdx < len(v.blocks); displayIdx++ {
		if displayIdx < v.scroll || displayIdx >= v.scroll+visibleRows {
			continue
		}

		blockIdx := len(v.blocks) - 1 - displayIdx
		b := v.blocks[blockIdx]
		hl := displayIdx == v.cursor // highlighted row?

		// # column
		numCell := cellStyle(cols[0].width, cols[0].align, hl).
			Foreground(theme.ColorMutedText).
			Render(fmt.Sprintf("%d", displayIdx+1))

		// Start/End — display end as EndTime - 1min to show e.g. "20:59"
		startStr := b.StartTime.In(v.tz).Format("Jan 02 15:04")
		endDisplay := b.EndTime.Add(-1 * time.Minute)
		endStr := endDisplay.In(v.tz).Format("Jan 02 15:04")

		startCell := cellStyle(cols[1].width, cols[1].align, hl).
			Foreground(theme.ColorBodyText).
			Render(startStr)
		endCell := cellStyle(cols[2].width, cols[2].align, hl).
			Foreground(theme.ColorBodyText).
			Render(endStr)

		// Token columns with color gradient (same as report stat cards)
		inputCell := cellStyle(cols[3].width, cols[3].align, hl).
			Foreground(theme.ColorLavender).
			Render(components.FormatCompact(b.InputTokens))
		outputCell := cellStyle(cols[4].width, cols[4].align, hl).
			Foreground(theme.ColorMauve).
			Render(components.FormatCompact(b.OutputTokens))
		crCell := cellStyle(cols[5].width, cols[5].align, hl).
			Foreground(theme.ColorPeach).
			Render(components.FormatCompact(b.CacheReadTokens))
		cwCell := cellStyle(cols[6].width, cols[6].align, hl).
			Foreground(theme.ColorGold).
			Render(components.FormatCompact(b.CacheCreationTokens))

		// Cost
		costCell := cellStyle(cols[7].width, cols[7].align, hl).
			Foreground(theme.ColorSkyBlue).
			Render(fmt.Sprintf("$%.2f", b.TotalCost))

		// Status — animated for active
		var statusCell string
		if b.Status == domain.BlockActive {
			var animText string
			if hl {
				animText = theme.AnimatedGradientText(string(b.Status), v.AnimTick, theme.ColorElevatedBg)
			} else {
				animText = theme.AnimatedGradientText(string(b.Status), v.AnimTick)
			}
			statusCell = cellStyle(cols[8].width, cols[8].align, hl).
				Render(animText)
		} else {
			statusCell = cellStyle(cols[8].width, cols[8].align, hl).
				Foreground(theme.ColorBrightText).
				Render(string(b.Status))
		}

		// Gap between cells also needs highlight background
		gap := " "
		if hl {
			gap = lipgloss.NewStyle().Background(theme.ColorElevatedBg).Render(" ")
		}

		row := strings.Join([]string{
			numCell, startCell, endCell,
			inputCell, outputCell, cwCell, crCell,
			costCell, statusCell,
		}, gap)

		rows = append(rows, row)
	}

	// Scroll indicator
	if len(v.blocks) > visibleRows {
		indicator := theme.MutedStyle.Render(
			fmt.Sprintf("  [%d-%d / %d]", v.scroll+1, min(v.scroll+visibleRows, len(v.blocks)), len(v.blocks)))
		rows = append(rows, indicator)
	}

	card.Content = strings.Join(rows, "\n")
	return card.Render()
}

func (v *BlocksView) renderDetail(cardWidth int, compact bool) string {
	blockIdx := len(v.blocks) - 1 - v.cursor
	if blockIdx < 0 || blockIdx >= len(v.blocks) {
		return theme.MutedStyle.Render(i18n.T("invalid_block"))
	}
	b := v.blocks[blockIdx]

	startStr := b.StartTime.In(v.tz).Format("Jan 02 15:04")
	endStr := b.EndTime.Add(-1 * time.Minute).In(v.tz).Format("15:04")

	// Summary card
	summaryCard := components.Card{
		Title: theme.AnimatedGradientText(
			i18n.Tf("block_detail", startStr, endStr),
			v.AnimTick),
		Width:   cardWidth,
		Compact: compact,
	}

	// Stat cards row for block detail
	statGap := 2
	innerW := summaryCard.InnerWidth()
	statW := (innerW - statGap*4) / 5
	if statW < 10 {
		statW = 10
	}
	stats := []components.StatCard{
		{Value: fmt.Sprintf("$%.2f", b.TotalCost), Label: i18n.T("cost"), Width: statW, Color: theme.ColorSkyBlue},
		{Value: components.FormatCompact(b.InputTokens), Label: i18n.T("input_tokens"), Width: statW, Color: theme.ColorLavender},
		{Value: components.FormatCompact(b.OutputTokens), Label: i18n.T("output_tokens"), Width: statW, Color: theme.ColorMauve},
		{Value: components.FormatCompact(b.CacheReadTokens), Label: i18n.T("cache_read"), Width: statW, Color: theme.ColorPeach},
		{Value: components.FormatCompact(b.CacheCreationTokens), Label: i18n.T("cache_create"), Width: statW, Color: theme.ColorGold},
	}
	summaryCard.Content = components.CenterBlock(components.RenderStatRow(stats, statGap), innerW)

	// Model breakdown card — table layout
	modelCard := components.Card{
		Title:   theme.GradientText(i18n.T("model_breakdown"), string(theme.ColorLavender), string(theme.ColorSkyBlue)),
		Width:   cardWidth,
		Compact: compact,
	}
	modelInnerW := modelCard.InnerWidth()

	// Column definitions: Model | Tokens | Cost | %
	mColModel := 30
	mColTokens := 12
	mColCost := 10
	mColPct := 7
	mGaps := 3 // 3 gaps between 4 columns
	mRemaining := modelInnerW - mColModel - mColTokens - mColCost - mColPct - mGaps
	if mRemaining > 0 {
		mColModel += mRemaining
	}

	mCell := func(text string, width int, align lipgloss.Position) string {
		return lipgloss.NewStyle().Width(width).Align(align).Render(text)
	}

	// Header — colored to match data columns
	mHeader := strings.Join([]string{
		lipgloss.NewStyle().Width(mColModel).Align(lipgloss.Left).Foreground(theme.ColorBrightText).Bold(true).Render(i18n.T("model")),
		lipgloss.NewStyle().Width(mColTokens).Align(lipgloss.Right).Foreground(theme.ColorLavender).Bold(true).Render(i18n.T("tokens")),
		lipgloss.NewStyle().Width(mColCost).Align(lipgloss.Right).Foreground(theme.ColorSkyBlue).Bold(true).Render(i18n.T("cost")),
		lipgloss.NewStyle().Width(mColPct).Align(lipgloss.Right).Foreground(theme.ColorGold).Bold(true).Render(i18n.T("percent")),
	}, " ")
	mSep := theme.MutedStyle.Render(strings.Repeat("─", mColModel+mColTokens+mColCost+mColPct+mGaps))

	var modelRows []string
	modelRows = append(modelRows, mHeader)
	modelRows = append(modelRows, mSep)

	// Sort models by percentage descending
	sortedModels := make([]domain.ModelBreakdown, 0, len(b.Models))
	for _, mb := range b.Models {
		sortedModels = append(sortedModels, mb)
	}
	sort.Slice(sortedModels, func(i, j int) bool {
		return sortedModels[i].Percentage > sortedModels[j].Percentage
	})

	rowIdx := 0
	for _, mb := range sortedModels {
		bgStyle := components.RowBackground(rowIdx)

		row := strings.Join([]string{
			mCell(lipgloss.NewStyle().Foreground(theme.ColorBrightText).Render(mb.Model), mColModel, lipgloss.Left),
			mCell(lipgloss.NewStyle().Foreground(theme.ColorLavender).Render(components.FormatNumber(mb.Tokens)), mColTokens, lipgloss.Right),
			mCell(lipgloss.NewStyle().Foreground(theme.ColorSkyBlue).Render(fmt.Sprintf("$%.4f", mb.Cost)), mColCost, lipgloss.Right),
			mCell(lipgloss.NewStyle().Foreground(theme.ColorGold).Render(fmt.Sprintf("%.1f%%", mb.Percentage)), mColPct, lipgloss.Right),
		}, " ")

		modelRows = append(modelRows, bgStyle.Render(row))
		rowIdx++
	}
	if rowIdx == 0 {
		modelRows = append(modelRows, theme.MutedStyle.Render(i18n.T("no_data")))
	}
	modelCard.Content = strings.Join(modelRows, "\n")

	// Recent entries card
	entriesCard := components.Card{
		Title:   theme.GradientText(i18n.T("recent_entries"), string(theme.ColorMauve), string(theme.ColorPeach)),
		Width:   cardWidth,
		Compact: compact,
	}
	start := len(b.Entries) - 20
	if start < 0 {
		start = 0
	}
	var entryRows []string
	for i := len(b.Entries) - 1; i >= start; i-- {
		e := b.Entries[i]
		row := fmt.Sprintf("%s  %-25s  in:%d out:%d cache:%d",
			theme.MutedStyle.Render(e.Timestamp.In(v.tz).Format("15:04:05")),
			theme.BodyStyle.Render(e.Model),
			e.InputTokens, e.OutputTokens, e.CacheCreationTokens+e.CacheReadTokens)
		entryRows = append(entryRows, row)
	}
	if len(entryRows) == 0 {
		entryRows = append(entryRows, theme.MutedStyle.Render(i18n.T("no_data")))
	}
	entriesCard.Content = strings.Join(entryRows, "\n")

	footer := components.HelpFooter(i18n.T("detail_back_help"))

	return summaryCard.Render() + "\n" + modelCard.Render() + "\n" + entriesCard.Render() + "\n" + footer
}
