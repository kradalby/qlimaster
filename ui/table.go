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
		focusedCell := NoCell
		if m.mode == ModeEditScore && i == m.rowCursor {
			focusedCell = m.focusedCell
		}
		lines = append(lines,
			m.renderDataRow(l, sorted[i], ranking.PositionOf(sorted[i].ID),
				i == m.rowCursor, i-start, focusedCell))
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

func (m Model) renderDataRow(l Layout, t quiz.Team, position int, focused bool, visibleIdx int, focusCell Cell) string {
	// Build one styled cell per addressable column, in order. The
	// per-cell styling (perfect-round fill, focus highlight, editing
	// buffer) is composed here so the later row-level styling (zebra /
	// row focus) doesn't clobber per-cell backgrounds.
	threshold := float64(m.quiz.Config.QuestionsPerRound)
	cells := make([]string, 0, 8)

	// Position cell.
	posText := strconv.Itoa(position)
	if position == 1 && quiz.RoundComplete(m.quiz, m.quiz.Config.Rounds) {
		posText = "★ " + posText
	}
	posCell := positionStyle(position).Render(padCell(posText, l.PosWidth, alignRight))
	cells = append(cells, decorateFocus(posCell, l.PosWidth, alignRight, focusCell,
		Cell{Kind: CellPosition}, m.edit, false))

	// Team name.
	teamRaw := padCell(truncate(t.Name, l.TeamWidth), l.TeamWidth, alignLeft)
	cells = append(cells, decorateFocus(teamRaw, l.TeamWidth, alignLeft, focusCell,
		Cell{Kind: CellTeam}, m.edit, true))

	if l.ShowPlayers {
		playersRaw := padCell(truncate(t.Players, l.PlayersWidth), l.PlayersWidth, alignLeft)
		cells = append(cells, decorateFocus(playersRaw, l.PlayersWidth, alignLeft, focusCell,
			Cell{Kind: CellPlayers}, m.edit, true))
	}

	for _, r := range l.VisibleRounds {
		v, ok := t.Score(r)
		text := ""
		if ok {
			text = score.Format(v)
		}
		raw := padCell(text, l.RoundWidth, alignRight)
		perfect := ok && v >= threshold
		roundCell := Cell{Kind: CellRound, Round: r}
		if perfect && !focusCell.Equal(roundCell) {
			raw = styles.Perfect.Render(raw)
		}
		cells = append(cells, decorateFocus(raw, l.RoundWidth, alignRight, focusCell,
			roundCell, m.edit, true))

		if slices.Contains(l.VisibleCheckpts, r) {
			cpRaw := padCell(score.Format(quiz.Checkpoint(t, r)), l.CheckptWidth, alignRight)
			cells = append(cells, decorateFocus(cpRaw, l.CheckptWidth, alignRight, focusCell,
				Cell{Kind: CellCheckpoint, Round: r}, m.edit, false))
		}
	}

	totalRaw := padCell(score.Format(t.Total()), l.TotalWidth, alignRight)
	cells = append(cells, decorateFocus(totalRaw, l.TotalWidth, alignRight, focusCell,
		Cell{Kind: CellTotal}, m.edit, false))

	line := " " + strings.Join(cells, " │ ")
	if l.RightPad > 0 {
		line += strings.Repeat(" ", l.RightPad)
	}
	line = padLine(line, l.Width)

	switch {
	case m.mode == ModeEditScore && focused:
		// Row-level highlight is dropped in Edit mode to keep per-cell
		// focus legible.
		return line
	case focused:
		return styles.RowFocus.Render(line)
	case visibleIdx%2 == 1:
		return styles.RowZebra.Render(line)
	default:
		return line
	}
}

// decorateFocus returns the given pre-padded cell string, optionally
// replaced by the focus/edit highlight when the cell matches focusCell.
// The editing buffer is rendered in-place when edit.editing is true and
// the cell is editable.
func decorateFocus(raw string, width, align int, focusCell, myCell Cell, es editState, editable bool) string {
	if !focusCell.Equal(myCell) {
		return raw
	}
	if es.editing && editable {
		text := es.input + "│"
		padded := padCell(text, width, align)
		return styles.CellEditing.Render(padded)
	}
	return styles.CellFocus.Render(raw)
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
