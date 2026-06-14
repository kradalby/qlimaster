package ui

import (
	"testing"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
