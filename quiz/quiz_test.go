package quiz_test

import (
	"testing"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     quiz.Config
		wantErr bool
	}{
		{"default ok", quiz.DefaultConfig(), false},
		{"zero rounds", quiz.Config{Rounds: 0, QuestionsPerRound: 10, Checkpoints: []int{}}, true},
		{"zero questions", quiz.Config{Rounds: 8, QuestionsPerRound: 0, Checkpoints: []int{}}, true},
		{"too many rounds", quiz.Config{Rounds: 100, QuestionsPerRound: 10}, true},
		{"checkpoint too high", quiz.Config{Rounds: 8, QuestionsPerRound: 10, Checkpoints: []int{9}}, true},
		{"checkpoint zero", quiz.Config{Rounds: 8, QuestionsPerRound: 10, Checkpoints: []int{0, 4}}, true},
		{"checkpoints unsorted", quiz.Config{Rounds: 8, QuestionsPerRound: 10, Checkpoints: []int{8, 4}}, true},
		{"checkpoints duplicate", quiz.Config{Rounds: 8, QuestionsPerRound: 10, Checkpoints: []int{4, 4}}, true},
		{"no checkpoints ok", quiz.Config{Rounds: 8, QuestionsPerRound: 10, Checkpoints: nil}, false},
		{"max points zero ok (legacy)", quiz.Config{Rounds: 8, QuestionsPerRound: 10, MaxPoints: 0}, false},
		{"max points positive ok", quiz.Config{Rounds: 8, QuestionsPerRound: 10, MaxPoints: 20}, false},
		{"max points negative", quiz.Config{Rounds: 8, QuestionsPerRound: 10, MaxPoints: -1}, true},
		{"max points too high", quiz.Config{Rounds: 8, QuestionsPerRound: 10, MaxPoints: 1001}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_MaxScore(t *testing.T) {
	t.Parallel()

	// Explicit MaxPoints wins.
	assert.InDelta(t, 20.0, quiz.Config{QuestionsPerRound: 10, MaxPoints: 20}.MaxScore(), 1e-9)
	// Zero MaxPoints falls back to QuestionsPerRound (legacy files).
	assert.InDelta(t, 10.0, quiz.Config{QuestionsPerRound: 10, MaxPoints: 0}.MaxScore(), 1e-9)
}

func TestNew(t *testing.T) {
	t.Parallel()

	q := quiz.New(quiz.DefaultConfig())
	assert.Equal(t, 1, q.Version)
	assert.Empty(t, q.Teams)
	assert.Equal(t, 8, q.Config.Rounds)
}

func TestTeam_TotalAndScore(t *testing.T) {
	t.Parallel()

	team := quiz.Team{
		ID:     "t1",
		Name:   "a",
		Scores: map[string]float64{"1": 5, "2": 2.5, "3": 0},
	}
	v, ok := team.Score(1)
	require.True(t, ok)
	assert.InDelta(t, 5.0, v, 1e-9)

	v, ok = team.Score(3)
	require.True(t, ok)
	assert.InDelta(t, 0.0, v, 1e-9)

	_, ok = team.Score(4)
	assert.False(t, ok)

	assert.InDelta(t, 7.5, team.Total(), 1e-9)
}

func TestFindTeam(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{
		Teams: []quiz.Team{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}},
	}
	require.NotNil(t, q.FindTeam("a"))
	assert.Nil(t, q.FindTeam("nope"))
}

func TestHasTeamNamed_CaseInsensitive(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{Teams: []quiz.Team{{ID: "a", Name: "Dark Horse"}}}
	assert.True(t, q.HasTeamNamed("dark horse"))
	assert.True(t, q.HasTeamNamed("DARK HORSE"))
	assert.False(t, q.HasTeamNamed("Dark"))
}
