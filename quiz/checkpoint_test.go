package quiz_test

import (
	"testing"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
)

func TestCheckpoint(t *testing.T) {
	t.Parallel()

	team := quiz.Team{
		Scores: map[string]float64{"1": 5, "2": 2.5, "3": 1, "5": 4},
	}
	assert.InDelta(t, 0.0, quiz.Checkpoint(team, 0), 1e-9)
	assert.InDelta(t, 5.0, quiz.Checkpoint(team, 1), 1e-9)
	assert.InDelta(t, 7.5, quiz.Checkpoint(team, 2), 1e-9)
	assert.InDelta(t, 8.5, quiz.Checkpoint(team, 3), 1e-9)
	// Gap at round 4 doesn't break accumulation.
	assert.InDelta(t, 12.5, quiz.Checkpoint(team, 5), 1e-9)
}

func TestRoundComplete(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{
		Config: quiz.DefaultConfig(),
		Teams: []quiz.Team{
			{ID: "a", Scores: map[string]float64{"1": 5}},
			{ID: "b", Scores: map[string]float64{"1": 3}},
		},
	}
	assert.True(t, quiz.RoundComplete(q, 1))
	assert.False(t, quiz.RoundComplete(q, 2))

	// No teams -> never complete.
	empty := quiz.Quiz{Config: quiz.DefaultConfig()}
	assert.False(t, quiz.RoundComplete(empty, 1))
}

func TestRoundAverage(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{
		Teams: []quiz.Team{
			{Scores: map[string]float64{"1": 6}},
			{Scores: map[string]float64{"1": 8}},
			{Scores: map[string]float64{"1": 4}},
		},
	}
	v, ok := quiz.RoundAverage(q, 1)
	assert.True(t, ok)
	assert.InDelta(t, 6.0, v, 1e-9)

	_, ok = quiz.RoundAverage(q, 2)
	assert.False(t, ok)
}

func TestTotalAverage(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{
		Teams: []quiz.Team{
			{Scores: map[string]float64{"1": 5, "2": 5}},
			{Scores: map[string]float64{"1": 3, "2": 3}},
		},
	}
	v, ok := quiz.TotalAverage(q)
	assert.True(t, ok)
	assert.InDelta(t, 8.0, v, 1e-9)

	_, ok = quiz.TotalAverage(quiz.Quiz{})
	assert.False(t, ok)
}
