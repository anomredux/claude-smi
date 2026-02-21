package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/anomredux/claude-smi/internal/api"
	"github.com/anomredux/claude-smi/internal/domain"
	"github.com/anomredux/claude-smi/internal/i18n"
	"github.com/anomredux/claude-smi/internal/pricing"
	"github.com/anomredux/claude-smi/internal/theme"
	"github.com/anomredux/claude-smi/internal/ui/components"
)

type LiveView struct {
	entries      []domain.UsageEntry
	blocks       []domain.SessionBlock
	daily        []domain.DailyAggregate
	tz *time.Location
	calc     *pricing.Calculator
	apiUsage *api.UsageData
	AnimTick uint

	// Cached burn rate (recomputed only on data change)
	burn burnCache

	// Cached model breakdown (recomputed only on data change)
	cachedSessionEntries []domain.UsageEntry
	cachedModelBreakdown map[string]domain.ModelBreakdown
}

type burnCache struct {
	inputTokens  int
	outputTokens int
	cacheCreate  int
	cacheRead    int
	totalCost    float64
	cacheSavings float64
	tokensPerMin float64
	costPerHour  float64
	hasData      bool
}

func NewLiveView(tz *time.Location, calc *pricing.Calculator) *LiveView {
	return &LiveView{tz: tz, calc: calc}
}

func (v *LiveView) SetData(entries []domain.UsageEntry, blocks []domain.SessionBlock, daily []domain.DailyAggregate) {
	v.entries = entries
	v.blocks = blocks
	v.daily = daily
	v.recomputeBurn()
}

func (v *LiveView) SetApiUsage(data *api.UsageData) {
	v.apiUsage = data
	if len(v.entries) > 0 {
		v.recomputeBurn()
	}
}

func (v *LiveView) recomputeBurn() {
	sEntries := v.sessionEntries()
	v.cachedSessionEntries = sEntries
	v.cachedModelBreakdown = v.sessionModelBreakdown(sEntries)

	if len(sEntries) == 0 {
		v.burn = burnCache{}
		return
	}

	var bc burnCache
	bc.hasData = true
	for _, e := range sEntries {
		bc.inputTokens += e.InputTokens
		bc.outputTokens += e.OutputTokens
		bc.cacheCreate += e.CacheCreationTokens
		bc.cacheRead += e.CacheReadTokens
		bc.totalCost += e.CostUSD
		if v.calc != nil {
			bc.cacheSavings += v.calc.CacheSavings(&e)
		}
	}
	activeTokens := bc.inputTokens + bc.outputTokens

	elapsed := time.Since(sEntries[0].Timestamp)
	if elapsed < time.Minute {
		elapsed = time.Minute
	}

	bc.tokensPerMin = float64(activeTokens) / elapsed.Minutes()
	bc.costPerHour = bc.totalCost / elapsed.Hours()
	v.burn = bc
}

func (v *LiveView) Update(msg tea.Msg) tea.Cmd {
	return nil
}

func (v *LiveView) Render(width, height int, compact bool) string {
	cardWidth := width - 4

	var sections []string
	sections = append(sections, v.renderSessionTimer(cardWidth, compact))
	sections = append(sections, v.renderUtilization(cardWidth, compact))
	sections = append(sections, v.renderBurnRate(cardWidth, compact))
	sections = append(sections, v.renderModelBreakdown(cardWidth, compact))

	return strings.Join(sections, "\n")
}

// sessionTimes returns normalized session start/end times and remaining duration.
// The API's resets_at jitters between nn:00 and nn:59, so we round to the
// nearest hour to get stable boundaries: nn:00 start → nn+4:59 end.
func (v *LiveView) sessionTimes() (start, end time.Time, remaining time.Duration, ok bool) {
	if v.apiUsage == nil {
		return time.Time{}, time.Time{}, 0, false
	}
	endRaw, err := v.apiUsage.SessionEnd()
	if err != nil {
		return time.Time{}, time.Time{}, 0, false
	}
	resetHour := endRaw.Round(time.Hour)
	start = resetHour.Add(-api.SessionWindow)
	end = resetHour.Add(-time.Minute)
	remaining = time.Until(endRaw)
	if remaining < 0 {
		remaining = 0
	}
	return start, end, remaining, true
}

// sessionEntries returns entries filtered to the current API session window.
func (v *LiveView) sessionEntries() []domain.UsageEntry {
	if v.apiUsage != nil {
		sessionStart, err := v.apiUsage.SessionStart()
		if err == nil {
			var filtered []domain.UsageEntry
			for _, e := range v.entries {
				if !e.Timestamp.Before(sessionStart) {
					filtered = append(filtered, e)
				}
			}
			return filtered
		}
	}
	if block := v.activeBlock(); block != nil {
		return block.Entries
	}
	return nil
}

// sessionModelBreakdown builds model breakdown from session-filtered entries.
func (v *LiveView) sessionModelBreakdown(entries []domain.UsageEntry) map[string]domain.ModelBreakdown {
	models := make(map[string]domain.ModelBreakdown)
	var totalTokens int
	for _, e := range entries {
		mb := models[e.Model]
		mb.Model = e.Model
		mb.Tokens += e.TotalTokens()
		mb.Cost += e.CostUSD
		totalTokens += e.TotalTokens()
		models[e.Model] = mb
	}
	for k, mb := range models {
		if totalTokens > 0 {
			mb.Percentage = float64(mb.Tokens) / float64(totalTokens) * 100
		}
		models[k] = mb
	}
	return models
}

func (v *LiveView) activeBlock() *domain.SessionBlock {
	for i := len(v.blocks) - 1; i >= 0; i-- {
		if v.blocks[i].Status == domain.BlockActive {
			return &v.blocks[i]
		}
	}
	return nil
}

// ── Section 1: Session Timer — Digital Clock ──

func (v *LiveView) renderSessionTimer(cardWidth int, compact bool) string {
	card := components.Card{
		Title:   theme.AnimatedGradientText(i18n.T("session_timer"), v.AnimTick),
		Width:   cardWidth,
		Compact: compact,
	}

	start, end, remaining, ok := v.sessionTimes()
	if !ok {
		card.Content = theme.MutedStyle.Render(i18n.T("no_active_block"))
		return card.Render()
	}

	innerW := card.InnerWidth()
	clock := components.SessionClock{
		Remaining:    remaining,
		SessionStart: start.In(v.tz).Format("15:04"),
		SessionEnd:   end.In(v.tz).Format("15:04"),
		Timezone:     start.In(v.tz).Format("MST"),
		Width:        innerW,
	}

	card.Content = clock.Render()
	return card.Render()
}

// ── Section 2: Utilization — 2 Semicircle Gauges (5h + 7d) ──

func (v *LiveView) renderUtilization(cardWidth int, compact bool) string {
	card := components.Card{
		Title:   theme.AnimatedGradientText(i18n.T("active_session_block"), v.AnimTick),
		Width:   cardWidth,
		Compact: compact,
	}

	if v.apiUsage == nil {
		card.Content = theme.MutedStyle.Render(i18n.T("no_active_block"))
		return card.Render()
	}

	innerW := card.InnerWidth()

	gaugeGap := 4
	gaugeW := (innerW - gaugeGap) / 2
	if gaugeW < 12 {
		gaugeW = 12
	}
	if gaugeW > 24 {
		gaugeW = 24
	}

	fiveHourPct := v.apiUsage.FiveHour.Utilization / 100.0
	sevenDayPct := v.apiUsage.SevenDay.Utilization / 100.0

	gauges := []components.SemicircleGauge{
		{
			Label:   i18n.T("five_hour"),
			Percent: fiveHourPct,
			Width:   gaugeW,
		},
		{
			Label:   i18n.T("seven_day"),
			Percent: sevenDayPct,
			Width:   gaugeW,
		},
	}

	card.Content = components.CenterBlock(components.RenderGaugeRow(gauges, gaugeGap), innerW)

	return card.Render()
}

// ── Section 3: Burn Rate — 4 Stat Cards (session-filtered) ──

func (v *LiveView) renderBurnRate(cardWidth int, compact bool) string {
	card := components.Card{
		Title:   theme.AnimatedGradientText(i18n.T("burn_rate"), v.AnimTick),
		Width:   cardWidth,
		Compact: compact,
	}

	bc := v.burn
	if !bc.hasData {
		card.Content = theme.MutedStyle.Render(i18n.T("no_active_session"))
		return card.Render()
	}

	innerW := card.InnerWidth()
	statGap := 2

	// Row 1: Input / Output tokens (with cache) + Est. Session Cost
	thirdW := (innerW - statGap*2) / 3
	if thirdW < 12 {
		thirdW = 12
	}

	// Gradient: SkyBlue → Lavender → Mauve → Peach → Gold
	row1 := []components.StatCard{
		{
			Value: components.FormatNumber(bc.inputTokens),
			Sub:   i18n.Tf("cached", components.FormatCompact(bc.cacheCreate)),
			Label: i18n.T("input_tokens"),
			Width: thirdW,
			Color: theme.ColorSkyBlue,
		},
		{
			Value: components.FormatNumber(bc.outputTokens),
			Sub:   i18n.Tf("cached", components.FormatCompact(bc.cacheRead)),
			Label: i18n.T("output_tokens"),
			Width: thirdW,
			Color: theme.ColorLavender,
		},
		{
			Value: fmt.Sprintf("$%.2f", bc.totalCost),
			Sub:   i18n.Tf("cache_saved", fmt.Sprintf("%.2f", bc.cacheSavings)),
			Label: i18n.T("session_cost"),
			Width: thirdW,
			Color: theme.ColorMauve,
		},
	}

	// Row 2: Rates
	halfW := (innerW - statGap) / 2
	if halfW < 14 {
		halfW = 14
	}

	row2 := []components.StatCard{
		{
			Value: components.FormatNumber(int(bc.tokensPerMin)),
			Label: i18n.T("tokens_per_min"),
			Width: halfW,
			Color: theme.ColorPeach,
		},
		{
			Value: fmt.Sprintf("$%.2f", bc.costPerHour),
			Label: i18n.T("cost_per_hour"),
			Width: halfW,
			Color: theme.ColorGold,
		},
	}

	content := components.CenterBlock(components.RenderStatRow(row1, statGap), innerW) + "\n\n" +
		components.CenterBlock(components.RenderStatRow(row2, statGap), innerW)
	card.Content = content

	return card.Render()
}

// ── Section 4: Model Breakdown — Pie Chart (session-filtered) ──

func (v *LiveView) renderModelBreakdown(cardWidth int, compact bool) string {
	card := components.Card{
		Title:   theme.AnimatedGradientText(i18n.T("model_breakdown"), v.AnimTick),
		Width:   cardWidth,
		Compact: compact,
	}

	models := v.cachedModelBreakdown
	if len(models) == 0 {
		card.Content = theme.MutedStyle.Render(i18n.T("no_data"))
		return card.Render()
	}

	// Collect slices in deterministic order (sort by model name)
	// to avoid Go map iteration randomness causing angle shifts.
	modelNames := make([]string, 0, len(models))
	for name := range models {
		modelNames = append(modelNames, name)
	}
	sort.Strings(modelNames)

	var slices []components.PieSlice
	for _, name := range modelNames {
		mb := models[name]
		slices = append(slices, components.PieSlice{
			Label:      mb.Model,
			Value:      mb.Cost,
			Percentage: mb.Percentage,
		})
	}

	innerW := card.InnerWidth()
	chartSize := innerW / 5
	if chartSize < 10 {
		chartSize = 10
	}
	if chartSize > 16 {
		chartSize = 16
	}

	pie := components.PieChart{
		Slices:    slices,
		ChartSize: chartSize,
		Width:     innerW,
	}

	card.Content = components.CenterBlock(pie.Render(), innerW)

	return card.Render()
}
