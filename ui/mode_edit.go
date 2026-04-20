package ui

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/score"
)

// editState holds ephemeral state for the EditScore mode.
//
// editCol is the logical column within the editable set:
//
//	0    = Team name (free text)
//	1    = Players (free text; omitted when the column is hidden, but
//	       still navigable via keyboard so narrow viewports don't lose
//	       access)
//	2..  = one entry per configured round, 1-based (so editCol=2 means
//	       round 1, editCol=3 means round 2, ...)
type editState struct {
	editing bool
	input   string
	// dPending is true after the first 'd' of a 'dd' delete-team combo.
	dPending bool
}

// startEditScore enters EditScore mode with the cursor at the current
// row/column selection.
func (m Model) startEditScore() Model {
	m.mode = ModeEditScore
	m.edit = editState{}
	m.errMsg = ""
	if m.editCol == 0 && len(m.quiz.Teams) > 0 {
		m.editCol = editColTeam
	}
	return m
}

// editCol indices -- constants to make dispatch readable.
const (
	editColTeam    = 0
	editColPlayers = 1
	editColRound1  = 2
)

// focusedTeamID returns the team ID under the cursor in edit mode, or
// empty when no team row exists.
func (m Model) focusedTeamID() string {
	sorted := quiz.SortByRanking(m.quiz)
	if m.rowCursor < 0 || m.rowCursor >= len(sorted) {
		return ""
	}
	return sorted[m.rowCursor].ID
}

// focusedRound returns the round number addressed by the current edit
// cursor, or 0 if the cursor is not on a round column.
func (m Model) focusedRound() int {
	col := m.editCol - editColRound1
	if col < 0 {
		return 0
	}
	r := col + 1
	if r < 1 || r > m.quiz.Config.Rounds {
		return 0
	}
	return r
}

// handleEditKey handles input in ModeEditScore.
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
		return m, nil
	}
	if matches(km.Enter, k) {
		return m.beginEdit(), nil
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

// editNavMove handles the arrow / hjkl / g / G / 0 / $ navigation keys.
func (m Model) editNavMove(k string, km KeyMap) Model {
	switch {
	case matches(km.Up, k):
		if m.rowCursor > 0 {
			m.rowCursor--
		}
	case matches(km.Down, k):
		if m.rowCursor < len(m.quiz.Teams)-1 {
			m.rowCursor++
		}
	case matches(km.Left, k):
		if m.editCol > 0 {
			m.editCol--
		}
	case matches(km.Right, k):
		maxCol := editColRound1 + m.quiz.Config.Rounds - 1
		if m.editCol < maxCol {
			m.editCol++
		}
	case matches(km.First, k):
		m.editCol = 0
	case matches(km.Last, k):
		m.editCol = editColRound1 + m.quiz.Config.Rounds - 1
	case matches(km.Top, k):
		m.rowCursor = 0
	case matches(km.Bottom, k):
		m.rowCursor = max(len(m.quiz.Teams)-1, 0)
	}
	return m
}

// beginEdit enters edit mode for the focused cell and seeds the input
// buffer with the current value so the user can append or retype.
func (m Model) beginEdit() Model {
	teamID := m.focusedTeamID()
	if teamID == "" {
		return m
	}
	team := m.quiz.FindTeam(teamID)
	if team == nil {
		return m
	}
	m.edit.editing = true
	m.errMsg = ""
	switch m.editCol {
	case editColTeam:
		m.edit.input = team.Name
	case editColPlayers:
		m.edit.input = team.Players
	default:
		if r := m.focusedRound(); r > 0 {
			if v, ok := team.Score(r); ok {
				m.edit.input = score.Format(v)
			} else {
				m.edit.input = ""
			}
		}
	}
	return m
}

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

func (m Model) commitEdit() (tea.Model, tea.Cmd) {
	teamID := m.focusedTeamID()
	if teamID == "" {
		m.edit.editing = false
		return m, nil
	}
	switch m.editCol {
	case editColTeam:
		return m.finishEdit(quiz.ChangeRenameTeam{TeamID: teamID, Name: m.edit.input})
	case editColPlayers:
		return m.finishEdit(quiz.ChangeSetPlayers{TeamID: teamID, Players: m.edit.input})
	default:
		r := m.focusedRound()
		if r == 0 {
			m.edit.editing = false
			return m, nil
		}
		v, err := score.Parse(m.edit.input, float64(m.quiz.Config.QuestionsPerRound))
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		return m.finishEdit(quiz.ChangeSetScore{TeamID: teamID, Round: r, Score: v})
	}
}

// finishEdit applies c through m.apply and exits the single-cell edit.
func (m Model) finishEdit(c quiz.Change) (tea.Model, tea.Cmd) {
	m2, cmd := m.apply(c)
	if m2.errMsg != "" {
		return m2, cmd
	}
	m2.edit.editing = false
	m2.edit.input = ""
	return m2, cmd
}

// clearFocused clears the focused score cell via ChangeClearScore.
func (m Model) clearFocused() (tea.Model, tea.Cmd) {
	teamID := m.focusedTeamID()
	if teamID == "" || m.focusedRound() == 0 {
		return m, nil
	}
	return m.apply(quiz.ChangeClearScore{TeamID: teamID, Round: m.focusedRound()})
}

// deleteFocusedTeam deletes the focused team after the 'dd' sequence.
// A confirmation prompt could be added later; for v1 the intent is
// explicit enough.
func (m Model) deleteFocusedTeam() (tea.Model, tea.Cmd) {
	teamID := m.focusedTeamID()
	if teamID == "" {
		return m, nil
	}
	m2, cmd := m.apply(quiz.ChangeDeleteTeam{TeamID: teamID})
	// Keep the cursor in bounds after deletion.
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
	if m.edit.editing {
		input := "[" + m.edit.input + "_]"
		switch m.editCol {
		case editColTeam:
			return "editing team " + input
		case editColPlayers:
			return "editing players " + input
		default:
			return "R" + strconv.Itoa(m.focusedRound()) + " " + team.Name + " " + input
		}
	}
	switch m.editCol {
	case editColTeam:
		return "team name"
	case editColPlayers:
		return "players"
	default:
		return "R" + strconv.Itoa(m.focusedRound()) + " " + team.Name
	}
}
