package quiz_test

import (
	"testing"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApply_AddTeam(t *testing.T) {
	t.Parallel()

	q := quiz.New(quiz.DefaultConfig())
	q2, res, err := quiz.Apply(q, quiz.ChangeAddTeam{Name: "Alpha"})
	require.NoError(t, err)
	require.True(t, res.Mutated)
	require.Len(t, q2.Teams, 1)
	assert.Equal(t, "Alpha", q2.Teams[0].Name)
	assert.NotEmpty(t, q2.Teams[0].ID)
	assert.True(t, res.ReRanked)
	// Input quiz must not have been mutated.
	assert.Empty(t, q.Teams)
}

func TestApply_AddTeamDuplicate(t *testing.T) {
	t.Parallel()

	q := quiz.New(quiz.DefaultConfig())
	q, _, err := quiz.Apply(q, quiz.ChangeAddTeam{Name: "Alpha"})
	require.NoError(t, err)

	_, _, err = quiz.Apply(q, quiz.ChangeAddTeam{Name: "alpha"}) // case-insensitive
	require.ErrorIs(t, err, quiz.ErrDuplicateTeam)
}

func TestApply_AddTeamEmptyName(t *testing.T) {
	t.Parallel()

	q := quiz.New(quiz.DefaultConfig())
	_, _, err := quiz.Apply(q, quiz.ChangeAddTeam{Name: "   "})
	require.ErrorIs(t, err, quiz.ErrEmptyTeamName)
}

func TestApply_SetScore(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a")
	teamID := q.Teams[0].ID

	q2, res, err := quiz.Apply(q, quiz.ChangeSetScore{TeamID: teamID, Round: 1, Score: 5})
	require.NoError(t, err)
	require.True(t, res.Mutated)

	v, ok := q2.Teams[0].Score(1)
	require.True(t, ok)
	assert.InDelta(t, 5.0, v, 1e-9)
	// Single team, round 1 complete after setting score: should flag.
	assert.Equal(t, 1, res.RoundJustCompleted)
	assert.True(t, res.ReRanked)
}

func TestApply_SetScoreUnknownTeam(t *testing.T) {
	t.Parallel()

	q := quiz.New(quiz.DefaultConfig())
	_, _, err := quiz.Apply(q, quiz.ChangeSetScore{TeamID: "nope", Round: 1, Score: 1})
	require.ErrorIs(t, err, quiz.ErrUnknownTeam)
}

func TestApply_SetScoreInvalidRound(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a")
	_, _, err := quiz.Apply(q, quiz.ChangeSetScore{TeamID: q.Teams[0].ID, Round: 0, Score: 1})
	require.ErrorIs(t, err, quiz.ErrInvalidRound)
	_, _, err = quiz.Apply(q, quiz.ChangeSetScore{TeamID: q.Teams[0].ID, Round: 99, Score: 1})
	require.ErrorIs(t, err, quiz.ErrInvalidRound)
}

func TestApply_ClearScore(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a")
	q, _, err := quiz.Apply(q, quiz.ChangeSetScore{TeamID: q.Teams[0].ID, Round: 1, Score: 5})
	require.NoError(t, err)
	q, _, err = quiz.Apply(q, quiz.ChangeClearScore{TeamID: q.Teams[0].ID, Round: 1})
	require.NoError(t, err)
	_, ok := q.Teams[0].Score(1)
	assert.False(t, ok)
}

func TestApply_PerfectRoundDetection(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a", "b")
	// Give team a a perfect round; team b a normal one.
	q, res, err := quiz.Apply(q, quiz.ChangeSetScore{TeamID: q.Teams[0].ID, Round: 1, Score: 10})
	require.NoError(t, err)
	require.Len(t, res.NewPerfectRounds, 1)
	assert.Equal(t, q.Teams[0].ID, res.NewPerfectRounds[0].TeamID)
	assert.Equal(t, 1, res.NewPerfectRounds[0].Round)

	q, res, err = quiz.Apply(q, quiz.ChangeSetScore{TeamID: q.Teams[1].ID, Round: 1, Score: 7})
	require.NoError(t, err)
	// No NEW perfect rounds this call.
	assert.Empty(t, res.NewPerfectRounds)
	// Round 1 fully scored -> just completed now.
	assert.Equal(t, 1, res.RoundJustCompleted)
	_ = q
}

func TestApply_DeleteTeam(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a", "b", "c")
	q, res, err := quiz.Apply(q, quiz.ChangeDeleteTeam{TeamID: q.Teams[1].ID})
	require.NoError(t, err)
	assert.True(t, res.Mutated)
	assert.True(t, res.ReRanked)
	require.Len(t, q.Teams, 2)
}

func TestApply_RenameTeam(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "Alpha", "Beta")
	q2, _, err := quiz.Apply(q, quiz.ChangeRenameTeam{TeamID: q.Teams[0].ID, Name: "Gamma"})
	require.NoError(t, err)
	assert.Equal(t, "Gamma", q2.Teams[0].Name)

	// Renaming to a name used by another team fails.
	_, _, err = quiz.Apply(q, quiz.ChangeRenameTeam{TeamID: q.Teams[0].ID, Name: "beta"})
	require.ErrorIs(t, err, quiz.ErrDuplicateTeam)
}

func TestApply_SetPlayers(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a")
	q2, _, err := quiz.Apply(q, quiz.ChangeSetPlayers{TeamID: q.Teams[0].ID, Players: "  alice, bob  "})
	require.NoError(t, err)
	assert.Equal(t, "alice, bob", q2.Teams[0].Players)
}

func TestApply_SetConfig(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a")
	q, _, err := quiz.Apply(q, quiz.ChangeSetScore{TeamID: q.Teams[0].ID, Round: 7, Score: 3})
	require.NoError(t, err)

	newCfg := quiz.Config{Rounds: 6, QuestionsPerRound: 10, Checkpoints: []int{3, 6}}
	q2, res, err := quiz.Apply(q, quiz.ChangeSetConfig{Config: newCfg})
	require.NoError(t, err)
	assert.True(t, res.ReRanked)
	assert.Equal(t, 6, q2.Config.Rounds)

	// Round 7 score should be dropped since it's out of range.
	_, ok := q2.Teams[0].Score(7)
	assert.False(t, ok)
}

func TestApply_SetConfigInvalid(t *testing.T) {
	t.Parallel()

	q := quiz.New(quiz.DefaultConfig())
	_, _, err := quiz.Apply(q, quiz.ChangeSetConfig{Config: quiz.Config{Rounds: 0}})
	require.ErrorIs(t, err, quiz.ErrInvalidConfig)
}

func TestApply_WinnerDecided(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a", "b")
	// Fill every round for both teams with a small 2-round quiz.
	q, _, _ = quiz.Apply(q, quiz.ChangeSetConfig{Config: quiz.Config{
		Rounds:            2,
		QuestionsPerRound: 10,
		Checkpoints:       []int{2},
	}})

	aid, bid := q.Teams[0].ID, q.Teams[1].ID
	q, _, _ = quiz.Apply(q, quiz.ChangeSetScore{TeamID: aid, Round: 1, Score: 5})
	q, _, _ = quiz.Apply(q, quiz.ChangeSetScore{TeamID: bid, Round: 1, Score: 5})
	q, _, _ = quiz.Apply(q, quiz.ChangeSetScore{TeamID: aid, Round: 2, Score: 5})
	_, res, _ := quiz.Apply(q, quiz.ChangeSetScore{TeamID: bid, Round: 2, Score: 7})

	assert.True(t, res.WinnerDecided)

	// If totals tied, no winner.
	q = withTeams(t, "a", "b")
	q, _, _ = quiz.Apply(q, quiz.ChangeSetConfig{Config: quiz.Config{
		Rounds: 1, QuestionsPerRound: 10, Checkpoints: []int{1},
	}})
	aid = q.Teams[0].ID
	bid = q.Teams[1].ID
	q, _, _ = quiz.Apply(q, quiz.ChangeSetScore{TeamID: aid, Round: 1, Score: 5})
	_, res2, _ := quiz.Apply(q, quiz.ChangeSetScore{TeamID: bid, Round: 1, Score: 5})
	assert.False(t, res2.WinnerDecided)
}

func TestApply_InputNotMutated(t *testing.T) {
	t.Parallel()

	q := withTeams(t, "a")
	orig := q.Teams[0].ID

	_, _, err := quiz.Apply(q, quiz.ChangeSetScore{TeamID: orig, Round: 1, Score: 5})
	require.NoError(t, err)
	// Original quiz's team must still have an empty scores map.
	_, ok := q.Teams[0].Score(1)
	assert.False(t, ok)
}

// withTeams builds a quiz with the default config and the supplied team
// names added via Apply so IDs are assigned by the normal path.
func withTeams(t *testing.T, names ...string) quiz.Quiz {
	t.Helper()
	q := quiz.New(quiz.DefaultConfig())
	for _, n := range names {
		var err error
		q, _, err = quiz.Apply(q, quiz.ChangeAddTeam{Name: n})
		require.NoError(t, err)
	}
	return q
}
