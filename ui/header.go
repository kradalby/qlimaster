package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/quiz"
)

// renderHeader returns the single-line top bar shaped like
//
//	qlimaster                N teams | RRxQQ | Hc1,c2          Day YYYY-MM-DD
//
// left-aligned app name, centered stats, right-aligned date. The whole bar
// is painted on the header background.
func renderHeader(width int, cfg quiz.Config, teamCount int, dateLabel string) string {
	if width <= 0 {
		return ""
	}
	app := styles.AppName.Render(" qlimaster ")
	stats := styles.Stats.Render(headerStats(cfg, teamCount))
	date := styles.DateRight.Render(dateLabel + " ")

	// Use lipgloss.Place to horizontally assemble the three regions.
	appW := lipgloss.Width(app)
	dateW := lipgloss.Width(date)
	middleW := max(width-appW-dateW, 1)
	mid := lipgloss.NewStyle().
		Background(pal.BgHeader).
		Width(middleW).
		Align(lipgloss.Center).
		Render(stats)
	line := lipgloss.JoinHorizontal(lipgloss.Top, app, mid, date)
	// Ensure full width even when lipgloss rounding is involved.
	if lipgloss.Width(line) < width {
		line += strings.Repeat(" ", width-lipgloss.Width(line))
	}
	return styles.TopBarBase.Render(line)
}

// headerStats formats the middle section of the header bar.
func headerStats(cfg quiz.Config, teamCount int) string {
	cp := formatCheckpoints(cfg.Checkpoints)
	if cp == "" {
		return fmt.Sprintf("%d teams | %dR x %dQ", teamCount, cfg.Rounds, cfg.QuestionsPerRound)
	}
	return fmt.Sprintf("%d teams | %dR x %dQ | %s", teamCount, cfg.Rounds, cfg.QuestionsPerRound, cp)
}

func formatCheckpoints(cps []int) string {
	if len(cps) == 0 {
		return ""
	}
	parts := make([]string, len(cps))
	for i, c := range cps {
		parts[i] = strconv.Itoa(c)
	}
	return "H" + strings.Join(parts, ",")
}
