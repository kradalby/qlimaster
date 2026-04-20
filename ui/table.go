package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/score"
)

// renderTable renders the full data table (header, rows, averages) using
// the precomputed layout. The outer frame is drawn by the caller; this
// function emits rows that already fit within Layout.Width.
func (m Model) renderTable(l Layout) string {
	header := m.renderHeaderRow(l)
	body := m.renderBodyRows(l)
	avg := m.renderAveragesRow(l)
	thinRule := m.renderThinRule(l)
	thickRule := m.renderThickRule(l)
	return lipgloss.JoinVertical(lipgloss.Left, header, thinRule, body, thickRule, avg)
}

// labelPosition, labelTeam ... return the header label text for the
// current breakpoint.
func (l Layout) labelPosition() string {
	if l.UseLongLabels {
		return "Position"
	}
	return "Pos"
}

func (l Layout) labelTeam() string { return "Team" }

func (l Layout) labelPlayers() string { return "Players" }

func (l Layout) labelRound(r int) string {
	if l.UseLongLabels {
		return "Round " + strconv.Itoa(r)
	}
	return "R" + strconv.Itoa(r)
}

func (l Layout) labelCheckpoint(r int) string {
	if l.UseLongLabels {
		return "Half R" + strconv.Itoa(r)
	}
	return "H" + strconv.Itoa(r)
}

func (l Layout) labelTotal() string { return "Total" }

func (m Model) renderHeaderRow(l Layout) string {
	parts := []string{
		padCell(l.labelPosition(), l.PosWidth, alignRight),
		padCell(l.labelTeam(), l.TeamWidth, alignLeft),
	}
	if l.ShowPlayers {
		parts = append(parts, padCell(l.labelPlayers(), l.PlayersWidth, alignLeft))
	}
	for _, r := range l.VisibleRounds {
		parts = append(parts, padCell(l.labelRound(r), l.RoundWidth, alignRight))
		if slices.Contains(l.VisibleCheckpts, r) {
			parts = append(parts, padCell(l.labelCheckpoint(r), l.CheckptWidth, alignRight))
		}
	}
	parts = append(parts, padCell(l.labelTotal(), l.TotalWidth, alignRight))

	inner := strings.Join(parts, " │ ")
	inner = " " + inner
	if l.RightPad > 0 {
		inner += strings.Repeat(" ", l.RightPad)
	}
	padded := padLine(inner, l.Width)
	return styles.TableHeader.Render(padded)
}

// renderThinRule draws a full-width single-line rule between the header
// and the body rows.
func (m Model) renderThinRule(l Layout) string {
	return padLine(styles.Separator.Render(strings.Repeat("─", l.Width)), l.Width)
}

// renderThickRule draws a full-width heavy rule above the averages row
// so the averages visually separate from the data body.
func (m Model) renderThickRule(l Layout) string {
	return padLine(styles.ThickRule.Render(strings.Repeat("━", l.Width)), l.Width)
}

func (m Model) renderBodyRows(l Layout) string {
	sorted := quiz.SortByRanking(m.quiz)
	ranking := quiz.Rank(m.quiz)

	if len(sorted) == 0 || l.TableHeight <= 0 {
		return padLine("", l.Width) // keep vertical space consistent
	}

	start, end := windowRange(l.TableHeight, len(sorted), m.rowCursor)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines,
			m.renderDataRow(l, sorted[i], ranking.PositionOf(sorted[i].ID), i == m.rowCursor, i-start))
	}
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

	line := strings.Join(parts, " │ ")
	line = " " + line
	if l.RightPad > 0 {
		line += strings.Repeat(" ", l.RightPad)
	}
	return padLine(styles.Averages.Render(line), l.Width)
}

func (m Model) renderDataRow(l Layout, t quiz.Team, position int, focused bool, visibleIdx int) string {
	posText := strconv.Itoa(position)
	if position == 1 && quiz.RoundComplete(m.quiz, m.quiz.Config.Rounds) {
		posText = "★ " + posText
	}
	posCell := padCell(posText, l.PosWidth, alignRight)
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
			cell = score.Format(v)
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

	line := strings.Join(parts, " │ ")
	line = " " + line
	if l.RightPad > 0 {
		line += strings.Repeat(" ", l.RightPad)
	}
	// Zebra stripe: alternate rows get a subtle dim background.
	line = padLine(line, l.Width)
	if focused {
		return styles.RowFocus.Render(line)
	}
	if visibleIdx%2 == 1 {
		return styles.RowZebra.Render(line)
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

// truncate returns s clipped to at most width visible columns, with a
// trailing ellipsis when clipping occurs.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
