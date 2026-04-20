package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/score"
)

// readOutState holds the ephemeral state for ModeReadOut.
type readOutState struct {
	// idx is the index into the worst-to-best ordering. 0 means the
	// lowest-ranked team (the one announced first).
	idx int
}

// startReadOut opens the presentation mode at the worst-ranked team.
func (m Model) startReadOut() Model {
	m.mode = ModeReadOut
	m.readOut = readOutState{idx: 0}
	m.errMsg = ""
	return m
}

// handleReadOutKey advances/rewinds through teams, or exits.
func (m Model) handleReadOutKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	total := len(m.quiz.Teams)
	if total == 0 {
		if matches(km.Escape, k) {
			m.mode = ModeNormal
		}
		return m, nil
	}
	switch {
	case matches(km.Escape, k):
		m.mode = ModeNormal
	case matches(km.Enter, k), k == "space", k == " ", isArrowDown(k), matches(km.Down, k):
		if m.readOut.idx < total-1 {
			m.readOut.idx++
		}
	case isArrowUp(k), matches(km.Up, k):
		if m.readOut.idx > 0 {
			m.readOut.idx--
		}
	case matches(km.Top, k):
		m.readOut.idx = 0
	case matches(km.Bottom, k):
		m.readOut.idx = total - 1
	}
	return m, nil
}

// renderReadOut draws the centered presentation card for the current
// team, over the full viewport. The table underneath is not drawn; the
// read-out mode takes over the whole screen so the host can project the
// terminal without table clutter.
func (m Model) renderReadOut() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	worstFirst := readOutOrder(m.quiz)
	if len(worstFirst) == 0 {
		return m.renderReadOutEmpty()
	}
	if m.readOut.idx < 0 {
		m.readOut.idx = 0
	}
	if m.readOut.idx >= len(worstFirst) {
		m.readOut.idx = len(worstFirst) - 1
	}
	team := worstFirst[m.readOut.idx]
	position := quiz.Rank(m.quiz).PositionOf(team.ID)
	isWinner := position == 1 && quiz.RoundComplete(m.quiz, m.quiz.Config.Rounds)

	title := styles.TopBarBase.Render(centerInWidth(
		styles.AppName.Render(" READ OUT  ·  "+
			strconv.Itoa(m.readOut.idx+1)+" / "+strconv.Itoa(len(worstFirst))+" "),
		m.width, pal.BgHeader))

	card := readOutCard(m.quiz, team, position, isWinner)
	cardPlaced := placeCenter(card, m.width, m.height-6)

	hints := []footerHint{
		{"Space / ↓", "next"},
		{"↑", "previous"},
		{"g", "first"},
		{"G", "last"},
		{"Esc", "exit"},
	}
	footer := renderFooter(m.width, ModeReadOut, statusForReadOut(m), hints)

	return lipgloss.JoinVertical(lipgloss.Left, title, cardPlaced, footer)
}

func (m Model) renderReadOutEmpty() string {
	msg := styles.OverlayTitle.Render("No teams yet")
	body := placeCenter(msg, m.width, m.height)
	return body
}

func statusForReadOut(m Model) string {
	return "team " + strconv.Itoa(m.readOut.idx+1) + " / " + strconv.Itoa(len(m.quiz.Teams))
}

// readOutOrder returns the teams sorted worst-to-best (ascending by
// total, alphabetical ascending for ties).
func readOutOrder(q quiz.Quiz) []quiz.Team {
	best := quiz.SortByRanking(q) // best-first
	out := make([]quiz.Team, len(best))
	for i, t := range best {
		out[len(best)-1-i] = t
	}
	return out
}

// readOutCard builds the centered card for one team.
func readOutCard(q quiz.Quiz, t quiz.Team, position int, isWinner bool) string {
	titleStyle := styles.OverlayTitle
	if isWinner {
		titleStyle = styles.Gold.Bold(true)
	}
	posLine := titleStyle.Render("Position " + strconv.Itoa(position))
	if isWinner {
		posLine = titleStyle.Render("★  POSITION  " + strconv.Itoa(position) + "  ★")
	}

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot)
	if isWinner {
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(pal.Gold)
	}
	nameLine := nameStyle.Render(t.Name)
	totalLine := lipgloss.NewStyle().Foreground(pal.FgBody).Render(
		score.Format(t.Total()) + " points")

	roundsBlock := renderRoundsTwoColumn(q, t)

	checkpointsLine := renderCheckpointsLine(q, t)

	lines := []string{
		"",
		posLine,
		"",
		nameLine,
		totalLine,
		"",
		strings.Repeat("─", 48),
		"",
		roundsBlock,
	}
	if checkpointsLine != "" {
		lines = append(lines, "", checkpointsLine)
	}
	if isWinner {
		lines = append(lines, "",
			lipgloss.NewStyle().Bold(true).Foreground(pal.Gold).Render("W  I  N  N  E  R"))
	}
	lines = append(lines, "")

	body := lipgloss.JoinVertical(lipgloss.Center, lines...)
	border := styles.OverlayBorder
	if isWinner {
		border = styles.OverlayBorder.BorderForeground(pal.Gold)
	}
	return border.Padding(1, 4).Render(body)
}

// renderRoundsTwoColumn renders the per-round scores in two side-by-side
// columns so the card fits in a reasonable height even for long quizzes.
func renderRoundsTwoColumn(q quiz.Quiz, t quiz.Team) string {
	rounds := q.Config.Rounds
	half := (rounds + 1) / 2

	left := make([]string, 0, half)
	right := make([]string, 0, rounds-half)
	for r := 1; r <= rounds; r++ {
		label := "Round " + strconv.Itoa(r)
		val := "—"
		if v, ok := t.Score(r); ok {
			val = score.Format(v)
		}
		line := label + "   " + val
		if r <= half {
			left = append(left, line)
		} else {
			right = append(right, line)
		}
	}
	// Join line-by-line with a gap.
	maxLines := max(len(left), len(right))
	for len(left) < maxLines {
		left = append(left, "")
	}
	for len(right) < maxLines {
		right = append(right, "")
	}
	rows := make([]string, maxLines)
	for i := range maxLines {
		rows[i] = padCell(left[i], 20, alignLeft) + "    " + padCell(right[i], 20, alignLeft)
	}
	return strings.Join(rows, "\n")
}

// renderCheckpointsLine adds halftime and final subtotals at the bottom
// of the card.
func renderCheckpointsLine(q quiz.Quiz, t quiz.Team) string {
	if len(q.Config.Checkpoints) == 0 {
		return ""
	}
	parts := make([]string, 0, len(q.Config.Checkpoints)+1)
	for _, cp := range q.Config.Checkpoints {
		parts = append(parts,
			"After R"+strconv.Itoa(cp)+"  "+score.Format(quiz.Checkpoint(t, cp)))
	}
	parts = append(parts, "Final  "+score.Format(t.Total()))
	return styles.Dimmed.Render(strings.Join(parts, "     "))
}

// placeCenter places s in a rectangle of (width, height), centered.
func placeCenter(s string, width, height int) string {
	if height <= 0 {
		return ""
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, s)
}

// keep package tea imported so goimports does not drop it.
var _ = tea.Quit
