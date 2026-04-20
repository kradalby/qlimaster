package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/quiz"
)

// renderHeader returns the three-line top banner:
//
//	line 1: solid pink band with the app title centered
//	line 2: stats band with teams/config/date
//	line 3: thin rule
//
// All three lines are exactly width columns wide.
func renderHeader(width int, cfg quiz.Config, teamCount int, dateLabel string) string {
	if width <= 0 {
		return ""
	}
	title := styles.AppName.Render(" qlimaster · pub quiz score manager ")
	line1 := styles.TopBarBase.Render(centerInWidth(title, width, pal.BgHeader))

	statsLeft := styles.Stats.Render("  " + strconv.Itoa(teamCount) + " teams")
	statsMid := styles.Stats.Render(middleStats(cfg))
	statsRight := styles.DateRight.Render(dateLabel + "  ")
	line2 := threeRegionLine(statsLeft, statsMid, statsRight, width, pal.BgHeader)

	line3 := styles.Separator.Render(strings.Repeat("─", width))
	return lipgloss.JoinVertical(lipgloss.Left, line1, line2, line3)
}

// middleStats formats the centre region of the stats banner. The final
// round's checkpoint is omitted because it duplicates the Total column.
func middleStats(cfg quiz.Config) string {
	cp := formatCheckpoints(filterNonFinalCheckpoints(cfg.Checkpoints, cfg.Rounds))
	main := fmt.Sprintf("%d rounds · %d questions", cfg.Rounds, cfg.QuestionsPerRound)
	if cp != "" {
		return main + " · " + cp
	}
	return main
}

func formatCheckpoints(cps []int) string {
	if len(cps) == 0 {
		return ""
	}
	parts := make([]string, len(cps))
	for i, c := range cps {
		parts[i] = "R" + strconv.Itoa(c)
	}
	return "subtotals at " + strings.Join(parts, ", ")
}

// centerInWidth centers s in width columns, filling the surrounding space
// with the supplied adaptive colour.
func centerInWidth(s string, width int, bg lipgloss.AdaptiveColor) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	leftN := (width - w) / 2
	rightN := width - w - leftN
	pad := lipgloss.NewStyle().Background(bg)
	return pad.Render(strings.Repeat(" ", leftN)) + s + pad.Render(strings.Repeat(" ", rightN))
}

// threeRegionLine composes a left/middle/right line of exactly width
// columns, right-padding the middle region with the background colour.
func threeRegionLine(left, middle, right string, width int, bg lipgloss.AdaptiveColor) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	mw := max(width-lw-rw, 1)
	mid := lipgloss.NewStyle().Background(bg).Width(mw).Align(lipgloss.Center).Render(middle)
	line := lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
	if w := lipgloss.Width(line); w < width {
		line += lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-w))
	}
	return line
}
