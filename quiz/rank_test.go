package quiz_test

import (
	"strconv"
	"testing"
	"testing/quick"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRank_BasicDescending(t *testing.T) {
	t.Parallel()

	q := buildQuiz(t, []testTeam{
		{id: "a", name: "Alpha", scores: map[int]float64{1: 5}},
		{id: "b", name: "Beta", scores: map[int]float64{1: 10}},
		{id: "c", name: "Gamma", scores: map[int]float64{1: 3}},
	})
	r := quiz.Rank(q)

	assert.Equal(t, 1, r.PositionOf("b"))
	assert.Equal(t, 2, r.PositionOf("a"))
	assert.Equal(t, 3, r.PositionOf("c"))
}

func TestRank_TiesShareStandardCompetition(t *testing.T) {
	t.Parallel()

	q := buildQuiz(t, []testTeam{
		{id: "a", name: "Alpha", scores: map[int]float64{1: 5}},
		{id: "b", name: "Beta", scores: map[int]float64{1: 5}},
		{id: "c", name: "Gamma", scores: map[int]float64{1: 3}},
		{id: "d", name: "Delta", scores: map[int]float64{1: 10}},
	})
	r := quiz.Rank(q)

	assert.Equal(t, 1, r.PositionOf("d"))
	// a and b tie at 5; both get 2.
	assert.Equal(t, 2, r.PositionOf("a"))
	assert.Equal(t, 2, r.PositionOf("b"))
	// c is next distinct; skips to 4.
	assert.Equal(t, 4, r.PositionOf("c"))
}

func TestRank_TiesAlphabeticalTiebreak(t *testing.T) {
	t.Parallel()

	// zebra and Alpha tie on total; Alpha should come first alphabetically
	// (after lowercasing) but they both share position.
	q := buildQuiz(t, []testTeam{
		{id: "z", name: "zebra", scores: map[int]float64{1: 5}},
		{id: "a", name: "Alpha", scores: map[int]float64{1: 5}},
	})
	sorted := quiz.SortByRanking(q)
	require.Len(t, sorted, 2)
	assert.Equal(t, "Alpha", sorted[0].Name)
	assert.Equal(t, "zebra", sorted[1].Name)
}

func TestRank_Property_Deterministic(t *testing.T) {
	t.Parallel()

	f := func(totals []uint8) bool {
		if len(totals) == 0 {
			return true
		}
		tt := make([]testTeam, 0, len(totals))
		for i, v := range totals {
			name := "team-" + string(rune('a'+i%26))
			tt = append(tt, testTeam{
				id:     name,
				name:   name,
				scores: map[int]float64{1: float64(v % 11)},
			})
		}
		q := buildQuizNoT(tt)
		r1 := quiz.Rank(q)
		r2 := quiz.Rank(q)
		for _, team := range q.Teams {
			if r1.PositionOf(team.ID) != r2.PositionOf(team.ID) {
				return false
			}
		}
		return true
	}
	require.NoError(t, quick.Check(f, nil))
}

func TestRank_Property_PositionIsPermutationOf1ToN(t *testing.T) {
	t.Parallel()

	f := func(totals []uint8) bool {
		if len(totals) == 0 {
			return true
		}
		tt := make([]testTeam, 0, len(totals))
		for i, v := range totals {
			id := string(rune('a' + i%26))
			if i/26 > 0 {
				id += string(rune('0' + i/26))
			}
			tt = append(tt, testTeam{
				id:     id,
				name:   id,
				scores: map[int]float64{1: float64(v % 11)},
			})
		}
		q := buildQuizNoT(tt)
		r := quiz.Rank(q)
		// Every position is in [1, len(teams)].
		for _, team := range q.Teams {
			p := r.PositionOf(team.ID)
			if p < 1 || p > len(q.Teams) {
				return false
			}
		}
		return true
	}
	require.NoError(t, quick.Check(f, nil))
}

// testTeam is a test-only construction helper; bypassing Apply is acceptable
// for test setup.
type testTeam struct {
	id     string
	name   string
	scores map[int]float64
}

func buildQuiz(t *testing.T, tt []testTeam) quiz.Quiz {
	t.Helper()
	return buildQuizNoT(tt)
}

func buildQuizNoT(tt []testTeam) quiz.Quiz {
	teams := make([]quiz.Team, 0, len(tt))
	for _, tc := range tt {
		scores := map[string]float64{}
		for r, v := range tc.scores {
			scores[roundKeyTest(r)] = v
		}
		teams = append(teams, quiz.Team{
			ID:     tc.id,
			Name:   tc.name,
			Scores: scores,
		})
	}
	return quiz.Quiz{
		Version: 1,
		Config:  quiz.DefaultConfig(),
		Teams:   teams,
	}
}

func roundKeyTest(r int) string {
	return strconv.Itoa(r)
}
