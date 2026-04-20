package ui

import (
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/score"
)

// editState holds the ephemeral UI state for ModeEditScore.
type editState struct {
	// editing is true while a cell's text input is active.
	editing bool
	// input is the buffered text for the current edit.
	input string
	// dPending is true after the first 'd' of a 'dd' delete-team combo.
	dPending bool
}

// startEditScore enters edit mode with the cursor on Team (unless the
// cursor already points at a valid cell in the current layout, in which
// case keep it).
func (m Model) startEditScore() Model {
	m.mode = ModeEditScore
	m.edit = editState{}
	m.errMsg = ""
	layout := Compute(m.width, m.height, m.quiz.Config, m.lastEntered)
	cells := AddressableCells(layout)
	if cellIndexOf(cells, m.focusedCell) < 0 {
		m.focusedCell = Cell{Kind: CellTeam}
	}
	return m
}

// handleEditKey dispatches input in ModeEditScore.
func (m Model) handleEditKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	if m.edit.editing {
		return m.editCellKey(k, km)
	}
	return m.editNavKey(k, km)
}

func (m Model) editNavKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	// Two-key 'dd' to delete team.
	if m.edit.dPending {
		m.edit.dPending = false
		if k == "d" {
			return m.deleteFocusedTeam()
		}
	}
	if matches(km.Escape, k) {
		m.mode = ModeNormal
		m.focusedCell = NoCell
		return m, nil
	}
	if matches(km.Enter, k) {
		return m.beginEdit()
	}
	if matches(km.Delete, k) {
		return m.clearFocused()
	}
	if k == "d" {
		m.edit.dPending = true
		return m, nil
	}
	return m.editNavMove(k, km), nil
}

// editNavMove handles arrow / hjkl / g / G / 0 / $ navigation. Left/Right
// walks the addressable-cell sequence (so invisible columns are
// automatically skipped). Up/Down moves between rows.
func (m Model) editNavMove(k string, km KeyMap) Model {
	layout := Compute(m.width, m.height, m.quiz.Config, m.lastEntered)
	cells := AddressableCells(layout)
	idx := cellIndexOf(cells, m.focusedCell)
	if idx < 0 && len(cells) > 0 {
		m.focusedCell = cells[0]
		idx = 0
	}
	m = m.moveRow(k, km)
	m = m.moveCell(k, km, cells, idx)
	return m
}

// moveRow applies row-cursor key handling (up/down/top/bottom).
func (m Model) moveRow(k string, km KeyMap) Model {
	switch {
	case matches(km.Up, k):
		if m.rowCursor > 0 {
			m.rowCursor--
		}
	case matches(km.Down, k):
		if m.rowCursor < len(m.quiz.Teams)-1 {
			m.rowCursor++
		}
	case matches(km.Top, k):
		m.rowCursor = 0
	case matches(km.Bottom, k):
		m.rowCursor = max(len(m.quiz.Teams)-1, 0)
	}
	return m
}

// moveCell applies cell-cursor key handling (left/right/first/last).
func (m Model) moveCell(k string, km KeyMap, cells []Cell, idx int) Model {
	switch {
	case matches(km.Left, k):
		if idx > 0 {
			m.focusedCell = cells[idx-1]
		}
	case matches(km.Right, k):
		if idx < len(cells)-1 {
			m.focusedCell = cells[idx+1]
		}
	case matches(km.First, k):
		if len(cells) > 0 {
			m.focusedCell = cells[0]
		}
	case matches(km.Last, k):
		if len(cells) > 0 {
			m.focusedCell = cells[len(cells)-1]
		}
	}
	return m
}

// beginEdit attempts to start editing the focused cell. Read-only cells
// show a 1.2s toast instead.
func (m Model) beginEdit() (tea.Model, tea.Cmd) {
	if !m.focusedCell.IsEditable() {
		m.status = "read-only cell"
		m.statusExpiry = time.Now().Add(1200 * time.Millisecond)
		return m, clearStatusCmd(1200 * time.Millisecond)
	}
	teamID := m.focusedTeamID()
	if teamID == "" {
		return m, nil
	}
	team := m.quiz.FindTeam(teamID)
	if team == nil {
		return m, nil
	}
	m.edit.editing = true
	m.errMsg = ""
	switch m.focusedCell.Kind {
	case CellTeam:
		m.edit.input = team.Name
	case CellPlayers:
		m.edit.input = team.Players
	case CellRound:
		if v, ok := team.Score(m.focusedCell.Round); ok {
			m.edit.input = score.Format(v)
		} else {
			m.edit.input = ""
		}
	}
	return m, nil
}

// focusedTeamID returns the team ID under the row cursor, from the
// currently-sorted order.
func (m Model) focusedTeamID() string {
	sorted := quiz.SortByRanking(m.quiz)
	if m.rowCursor < 0 || m.rowCursor >= len(sorted) {
		return ""
	}
	return sorted[m.rowCursor].ID
}

// editCellKey handles keys while actively editing a cell.
func (m Model) editCellKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Escape, k):
		m.edit.editing = false
		m.edit.input = ""
		m.errMsg = ""
	case matches(km.Clear, k):
		m.edit.input = ""
	case k == keyBackspace:
		if m.edit.input != "" {
			m.edit.input = m.edit.input[:len(m.edit.input)-1]
		}
	case matches(km.Enter, k):
		return m.commitEdit()
	default:
		if len(k) == 1 && k[0] >= ' ' {
			m.edit.input += k
		}
	}
	return m, nil
}

// commitEdit applies the pending change through Model.apply, clears the
// editing state, and keeps the cursor on the same cell. The user can
// then move with arrow keys to the next cell.
func (m Model) commitEdit() (tea.Model, tea.Cmd) {
	teamID := m.focusedTeamID()
	if teamID == "" {
		m.edit.editing = false
		return m, nil
	}
	var change quiz.Change
	switch m.focusedCell.Kind {
	case CellTeam:
		change = quiz.ChangeRenameTeam{TeamID: teamID, Name: m.edit.input}
	case CellPlayers:
		change = quiz.ChangeSetPlayers{TeamID: teamID, Players: m.edit.input}
	case CellRound:
		v, err := score.Parse(m.edit.input, float64(m.quiz.Config.QuestionsPerRound))
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		change = quiz.ChangeSetScore{TeamID: teamID, Round: m.focusedCell.Round, Score: v}
	default:
		m.edit.editing = false
		return m, nil
	}
	m2, cmd := m.apply(change)
	if m2.errMsg != "" {
		return m2, cmd
	}
	m2.edit.editing = false
	m2.edit.input = ""
	// Intentionally leave m2.focusedCell where it was so the cursor
	// stays put; arrow keys afterwards move to the next cell.
	return m2, cmd
}

// clearFocused clears the focused round cell via ChangeClearScore.
// Ignored on non-round cells.
func (m Model) clearFocused() (tea.Model, tea.Cmd) {
	if m.focusedCell.Kind != CellRound {
		return m, nil
	}
	teamID := m.focusedTeamID()
	if teamID == "" {
		return m, nil
	}
	return m.apply(quiz.ChangeClearScore{TeamID: teamID, Round: m.focusedCell.Round})
}

// deleteFocusedTeam deletes the focused team via ChangeDeleteTeam after
// the 'dd' sequence.
func (m Model) deleteFocusedTeam() (tea.Model, tea.Cmd) {
	teamID := m.focusedTeamID()
	if teamID == "" {
		return m, nil
	}
	m2, cmd := m.apply(quiz.ChangeDeleteTeam{TeamID: teamID})
	if m2.rowCursor >= len(m2.quiz.Teams) {
		m2.rowCursor = max(len(m2.quiz.Teams)-1, 0)
	}
	return m2, cmd
}

// renderEditStatus returns the right-side status string shown in the
// footer while in edit mode.
func (m Model) renderEditStatus() string {
	teamID := m.focusedTeamID()
	if teamID == "" {
		return "no team"
	}
	team := m.quiz.FindTeam(teamID)
	if team == nil {
		return ""
	}
	label := cellLabel(m.focusedCell, team)
	if m.edit.editing {
		return "editing " + label + " · [" + m.edit.input + "│]"
	}
	return label
}

// cellLabel returns a human-readable description of the focused cell.
func cellLabel(c Cell, team *quiz.Team) string {
	switch c.Kind {
	case CellPosition:
		return "position · " + team.Name
	case CellTeam:
		return "team name · " + team.Name
	case CellPlayers:
		return "players · " + team.Name
	case CellRound:
		return "round " + strconv.Itoa(c.Round) + " · " + team.Name
	case CellCheckpoint:
		return "halftime R" + strconv.Itoa(c.Round) + " · " + team.Name
	case CellTotal:
		return "total · " + team.Name
	default:
		return team.Name
	}
}
