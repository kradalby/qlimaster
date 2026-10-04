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
		m.width, pal.BgHeader,
	))

	tied := readOutTiedTeams(worstFirst, team)
	card := readOutCard(m.quiz, team, position, tied, isWinner, m.width, m.height-6)
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
// total, alphabetical descending for ties).
func readOutOrder(q quiz.Quiz) []quiz.Team {
	best := quiz.SortByRanking(q) // best-first

	out := make([]quiz.Team, len(best))
	for i, t := range best {
		out[len(best)-1-i] = t
	}

	return out
}

// readOutTiedTeams returns the teams sharing t's total, in readout order.
// The exact == comparison matches quiz.Rank's tie grouping.
func readOutTiedTeams(order []quiz.Team, t quiz.Team) []quiz.Team {
	var tied []quiz.Team

	total := t.Total()

	for _, other := range order {
		if other.Total() == total {
			tied = append(tied, other)
		}
	}

	return tied
}

// readOutCard shows every tied team, with the selected team's stats below.
func readOutCard(q quiz.Quiz, t quiz.Team, position int, tied []quiz.Team, isWinner bool, width, height int) string {
	titleStyle := styles.OverlayTitle
	if isWinner {
		titleStyle = styles.Gold.Bold(true)
	}

	shared := len(tied) > 1

	label := "Position " + strconv.Itoa(position)
	if isWinner {
		label = "POSITION " + strconv.Itoa(position)
	}

	if shared {
		label = "Shared position " + strconv.Itoa(position) + " · " + strconv.Itoa(len(tied)) + " teams"
	}

	if isWinner {
		label = "★ " + label + " ★"
	}

	posLine := titleStyle.Render(label)

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot)
	if isWinner {
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(pal.Gold)
	}

	nameLine := nameStyle.Render(t.Name)
	totalLine := lipgloss.NewStyle().Foreground(pal.FgBody).Render(
		score.Format(t.Total()) + " points",
	)

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

	if shared {
		roster, selected := renderReadOutRoster(tied, t.ID, nameStyle)
		lines = []string{
			"", posLine, totalLine + " each", "", roster, "",
			strings.Repeat("─", 48), "",
			"Stats for " + nameLine + " · " + strconv.Itoa(selected) + " / " + strconv.Itoa(len(tied)),
			"", roundsBlock,
		}
	}

	if checkpointsLine != "" {
		lines = append(lines, "", checkpointsLine)
	}

	if isWinner {
		winner := "W  I  N  N  E  R"
		if shared {
			winner += "  S"
		}

		lines = append(lines, "",
			lipgloss.NewStyle().Bold(true).Foreground(pal.Gold).Render(winner))
	}

	lines = append(lines, "")

	border := styles.OverlayBorder
	if isWinner {
		border = styles.OverlayBorder.BorderForeground(pal.Gold)
	}

	return fitReadOutCard(lines, border, shared, width, height)
}

func fitReadOutCard(lines []string, border lipgloss.Style, shared bool, width, height int) string {
	// Bound long names to the terminal and remove spacer lines when a
	// shared roster would otherwise push the stats below the viewport.
	bodyStyle := lipgloss.NewStyle()
	if shared {
		bodyStyle = bodyStyle.Width(max(1, min(78, width-10))).Align(lipgloss.Center)
	}

	card := border.Padding(1, 4).Render(bodyStyle.Render(lipgloss.JoinVertical(lipgloss.Center, lines...)))
	if shared && lipgloss.Height(card) > height {
		compact := make([]string, 0, len(lines))
		for _, line := range lines {
			if line != "" {
				compact = append(compact, line)
			}
		}

		card = border.Padding(0, 4).Render(bodyStyle.Render(lipgloss.JoinVertical(lipgloss.Center, compact...)))
	}

	return card
}

// renderReadOutRoster uses team IDs so duplicate names still select one row.
func renderReadOutRoster(tied []quiz.Team, selectedID string, nameStyle lipgloss.Style) (string, int) {
	rows := make([]string, 0, len(tied))
	selected := 0

	for i, team := range tied {
		row := "  " + team.Name
		if team.ID == selectedID {
			selected = i + 1
			row = nameStyle.Background(pal.BgSelect).Render("▶ " + team.Name)
		}

		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...), selected
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

// renderCheckpointsLine adds subtotal checkpoints and the final total at
// the bottom of the card. Checkpoints that land on the last round are
// skipped because the Final column already shows the same value.
func renderCheckpointsLine(q quiz.Quiz, t quiz.Team) string {
	cps := filterNonFinalCheckpoints(q.Config.Checkpoints, q.Config.Rounds)

	parts := make([]string, 0, len(cps)+1)
	for _, cp := range cps {
		parts = append(parts,
			"Score after R"+strconv.Itoa(cp)+"  "+score.Format(quiz.Checkpoint(t, cp)))
	}

	parts = append(parts, "Total  "+score.Format(t.Total()))

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
