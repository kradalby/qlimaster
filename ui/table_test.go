package ui

import (
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kradalby/qlimaster/quiz"
)

// TestDecorateFocus_HighlightSurvivesRowBackground guards the edit-mode
// focus marker: the CellFocus highlight must be painted over the plain
// cell text, not the row-background-painted version, otherwise the row's
// zebra/focus background clobbers the marker and the user can't see which
// cell they are about to edit.
//
//nolint:paralleltest // mutates the global lipgloss color profile
func TestDecorateFocus_HighlightSurvivesRowBackground(t *testing.T) {
	// Force a color profile so lipgloss emits ANSI off-TTY; the global
	// default renderer is shared, so don't run this in parallel.
	lipgloss.SetColorProfile(termenv.TrueColor)

	plain := padCell("5", 4, alignRight)
	rowPainted := styles.RowFocus.Render(plain)

	cell := Cell{Kind: CellRound, Round: 1}
	got := decorateFocus(rowPainted, plain, 4, alignRight, cell, cell, editState{}, true)

	assert.Equal(t, styles.CellFocus.Render(plain), got,
		"focus highlight must wrap the plain text")
	assert.NotEqual(t, styles.CellFocus.Render(rowPainted), got,
		"focus highlight must not wrap the row-painted text (background would clobber it)")
}

// TestRenderTable_EditFocusMarkerVisible is the end-to-end guard: render a
// real table in edit mode and confirm the focused score cell carries the
// CellFocus highlight over its plain text, proving the call sites pass the
// unpainted string through decorateFocus.
//
//nolint:paralleltest // mutates the global lipgloss color profile
func TestRenderTable_EditFocusMarkerVisible(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.Config{Rounds: 3, QuestionsPerRound: 10},
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	teamID := m.quiz.Teams[0].ID
	m, _ = m.apply(quiz.ChangeSetScore{TeamID: teamID, Round: 1, Score: 5})

	m.width, m.height = 140, 30
	m.mode = ModeEditScore
	m.rowCursor = 0
	m.focusedCell = Cell{Kind: CellRound, Round: 1}

	l := Compute(m.width, m.height, m.quiz.Config, m.lastEntered)
	out := m.renderTable(l)

	want := styles.CellFocus.Render(padCell("5", l.RoundWidth, alignRight))
	assert.Contains(t, out, want, "focused score cell must show the CellFocus marker")
}

// TestAddressableCells verifies the cell sequence in the Full breakpoint
// includes Position, Team, Players, Round 1..N (with checkpoints
// interleaved after matching round numbers), and Total. Checkpoints that
// land on the final round are filtered out because they duplicate Total.
func TestAddressableCells_Full(t *testing.T) {
	t.Parallel()

	l := Compute(160, 40,
		quiz.Config{Rounds: 3, QuestionsPerRound: 10, Checkpoints: []int{2, 3}}, 0)
	cells := AddressableCells(l)

	kinds := make([]CellKind, len(cells))
	for i, c := range cells {
		kinds[i] = c.Kind
	}
	// The checkpoint at round 3 is dropped because it equals the total.
	want := []CellKind{
		CellPosition, CellTeam, CellPlayers,
		CellRound, CellRound, CellCheckpoint, CellRound,
		CellTotal,
	}
	assert.Equal(t, want, kinds)
}

// TestAddressableCells_NoPlayersBreakpoint drops the Players cell when
// the layout hides that column.
func TestAddressableCells_NoPlayersBreakpoint(t *testing.T) {
	t.Parallel()

	l := Compute(110, 40,
		quiz.Config{Rounds: 2, QuestionsPerRound: 10, Checkpoints: nil}, 0)

	cells := AddressableCells(l)
	for _, c := range cells {
		assert.NotEqual(t, CellPlayers, c.Kind)
	}
}

// TestTableRendersTeamName is a smoke test: a team with scores renders
// its name somewhere in the table output.
func TestTableRendersTeamName(t *testing.T) {
	t.Parallel()

	m := Model{
		width: 160, height: 30,
		quiz: quiz.Quiz{
			Version: 1,
			Config:  quiz.Config{Rounds: 2, QuestionsPerRound: 10, Checkpoints: []int{2}},
			Teams: []quiz.Team{
				{ID: "t1", Name: "The rookies", Scores: map[string]float64{"1": 10, "2": 5}},
			},
		},
	}
	l := Compute(m.width, m.height, m.quiz.Config, 2)
	out := m.renderTable(l)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "The rookies")
}

// TestPerfectScoreRespectsRoundCap confirms the perfect-round highlight is
// judged against each round's own cap, so a bonus round (higher
// RoundMaxPoints) only lights up at its true maximum.
func TestPerfectScoreRespectsRoundCap(t *testing.T) {
	t.Parallel()

	m := Model{
		quiz: quiz.Quiz{
			Config: quiz.Config{
				Rounds: 3, QuestionsPerRound: 10,
				RoundMaxPoints: map[string]int{"3": 11},
			},
		},
	}

	// Normal round (cap = quiz-wide MaxScore 10) is perfect at 10.
	assert.True(t, m.perfectScore(10, true, 1))
	// Round 3 is capped at 11: a 10 is below the cap, so not perfect...
	assert.False(t, m.perfectScore(10, true, 3))
	// ...but a true 11 hits the cap.
	assert.True(t, m.perfectScore(11, true, 3))
	// A blank cell (ok=false) is never perfect.
	assert.False(t, m.perfectScore(0, false, 1))
}
