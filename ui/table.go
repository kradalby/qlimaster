package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/score"
)

// renderTable renders the full data table (header, separators, rows,
// averages) using the precomputed layout.
func (m Model) renderTable(l Layout) string {
	header := m.renderHeaderRow(l)
	sepTop := m.renderSeparator(l)
	body := m.renderBodyRows(l)
	sepBot := m.renderSeparator(l)
	avg := m.renderAveragesRow(l)
	return lipgloss.JoinVertical(lipgloss.Left, header, sepTop, body, sepBot, avg)
}

func (m Model) renderHeaderRow(l Layout) string {
	parts := []string{
		padCell("Pos", l.PosWidth, alignRight),
		padCell("Team", l.TeamWidth, alignLeft),
	}
	if l.ShowPlayers {
		parts = append(parts, padCell("Players", l.PlayersWidth, alignLeft))
	}
	for _, r := range l.VisibleRounds {
		parts = append(parts, padCell("R"+strconv.Itoa(r), l.RoundWidth, alignRight))
		if slices.Contains(l.VisibleCheckpts, r) {
			parts = append(parts, padCell("H"+strconv.Itoa(r), l.CheckptWidth, alignRight))
		}
	}
	parts = append(parts, padCell("Total", l.TotalWidth, alignRight))

	line := strings.Join(parts, " | ")
	return padLine(styles.TableHeader.Render(line), l.Width)
}

func (m Model) renderSeparator(l Layout) string {
	return padLine(styles.Separator.Render(strings.Repeat("-", l.Width)), l.Width)
}

func (m Model) renderBodyRows(l Layout) string {
	sorted := quiz.SortByRanking(m.quiz)
	ranking := quiz.Rank(m.quiz)

	if len(sorted) == 0 || l.TableHeight <= 0 {
		return padLine("", l.Width) // keep vertical space consistent
	}

	// Viewport clamp: if there are more rows than TableHeight, scroll to
	// keep the row cursor visible.
	start, end := windowRange(l.TableHeight, len(sorted), m.rowCursor)
	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, m.renderDataRow(l, sorted[i], ranking.PositionOf(sorted[i].ID), i == m.rowCursor))
	}
	// Pad to TableHeight lines so the table occupies its allotted area.
	for len(lines) < l.TableHeight {
		lines = append(lines, padLine("", l.Width))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderAveragesRow(l Layout) string {
	parts := []string{
		padCell("avg", l.PosWidth, alignRight),
		padCell("", l.TeamWidth, alignLeft),
	}
	if l.ShowPlayers {
		parts = append(parts, padCell("", l.PlayersWidth, alignLeft))
	}
	for _, r := range l.VisibleRounds {
		avg, ok := quiz.RoundAverage(m.quiz, r)
		parts = append(parts, padCell(formatAvg(avg, ok), l.RoundWidth, alignRight))
		if slices.Contains(l.VisibleCheckpts, r) {
			cavg, cok := quiz.CheckpointAverage(m.quiz, r)
			parts = append(parts, padCell(formatAvg(cavg, cok), l.CheckptWidth, alignRight))
		}
	}
	tavg, tok := quiz.TotalAverage(m.quiz)
	parts = append(parts, padCell(formatAvg(tavg, tok), l.TotalWidth, alignRight))

	line := strings.Join(parts, " | ")
	return padLine(styles.Averages.Render(line), l.Width)
}

func (m Model) renderDataRow(l Layout, t quiz.Team, position int, focused bool) string {
	posCell := padCell(strconv.Itoa(position), l.PosWidth, alignRight)
	posCell = positionStyle(position).Render(posCell)

	teamName := truncate(t.Name, l.TeamWidth)
	parts := []string{
		posCell,
		padCell(teamName, l.TeamWidth, alignLeft),
	}
	if l.ShowPlayers {
		parts = append(parts, padCell(truncate(t.Players, l.PlayersWidth), l.PlayersWidth, alignLeft))
	}
	threshold := float64(m.quiz.Config.QuestionsPerRound)
	for _, r := range l.VisibleRounds {
		v, ok := t.Score(r)
		cell := ""
		if ok {
			s := score.Format(v)
			if v >= threshold {
				s += "*"
			}
			cell = s
		}
		cellStr := padCell(cell, l.RoundWidth, alignRight)
		if ok && v >= threshold {
			cellStr = styles.Perfect.Render(cellStr)
		}
		parts = append(parts, cellStr)
		if slices.Contains(l.VisibleCheckpts, r) {
			parts = append(parts, padCell(score.Format(quiz.Checkpoint(t, r)), l.CheckptWidth, alignRight))
		}
	}
	parts = append(parts, padCell(score.Format(t.Total()), l.TotalWidth, alignRight))

	line := strings.Join(parts, " | ")
	if focused {
		line = styles.RowFocus.Render(padLine(line, l.Width))
	} else {
		line = padLine(line, l.Width)
	}
	return line
}

// positionStyle returns the lipgloss style for the given position. Gold
// for 1, silver for 2, bronze for 3; otherwise a neutral default.
func positionStyle(pos int) lipgloss.Style {
	switch pos {
	case 1:
		return styles.Gold
	case 2:
		return styles.Silver
	case 3:
		return styles.Bronze
	default:
		return lipgloss.NewStyle()
	}
}

// formatAvg formats a row-average value; returns empty string when ok is
// false so blank-round cells stay blank.
func formatAvg(v float64, ok bool) string {
	if !ok {
		return ""
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// windowRange returns a half-open [start, end) range of row indices that
// (a) fits in `height` rows and (b) includes `cursor`.
func windowRange(height, total, cursor int) (int, int) {
	if total <= height {
		return 0, total
	}
	start := max(cursor-height/2, 0)
	end := start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

// alignment constants for padCell.
const (
	alignLeft  = 0
	alignRight = 1
)

func padCell(s string, width, align int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w >= width {
		return truncate(s, width)
	}
	pad := strings.Repeat(" ", width-w)
	if align == alignRight {
		return pad + s
	}
	return s + pad
}

// truncate returns s clipped to at most `width` visible columns, with a
// trailing '~' to indicate truncation when clipping occurs.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "~"
	}
	// Approximate: drop runes from the end until it fits, then replace the
	// last character with '~'.
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "~"
}
