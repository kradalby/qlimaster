package store_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "quiz.hujson")

	q := quiz.Quiz{
		Version: 1,
		Created: time.Date(2026, 4, 14, 19, 0, 0, 0, time.UTC),
		Config:  quiz.DefaultConfig(),
		Teams: []quiz.Team{
			{
				ID:      "t_a",
				Name:    "Alpha",
				Players: "alice, bob",
				Scores:  map[string]float64{"1": 5, "2": 2.5},
			},
		},
	}

	require.NoError(t, store.Save(path, q))

	loaded, err := store.Load(path)
	require.NoError(t, err)
	assert.Equal(t, q.Version, loaded.Version)
	assert.Equal(t, q.Config, loaded.Config)
	assert.Equal(t, "Alpha", loaded.Teams[0].Name)
	assert.InDelta(t, 5.0, loaded.Teams[0].Scores["1"], 1e-9)
	assert.InDelta(t, 2.5, loaded.Teams[0].Scores["2"], 1e-9)
}

func TestLoadMissing(t *testing.T) {
	t.Parallel()

	_, err := store.Load(filepath.Join(t.TempDir(), "nope.hujson"))
	require.Error(t, err)
	require.ErrorIs(t, err, store.ErrNotFound)
}

func TestLoadWithComments(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
	  // Quiz on 2026-04-14.
	  "version": 1,
	  "created": "2026-04-14T19:00:00Z",
	  "config": {
	    "rounds": 8,
	    "questions_per_round": 10,
	    "checkpoints": [4, 8],
	  },
	  "teams": [
	    {
	      "id": "t1",
	      "name": "Alpha",
	      "players": "",
	      "scores": { "1": 5 },
	    },
	  ],
	}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "quiz.hujson")
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	q, err := store.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 8, q.Config.Rounds)
	assert.Equal(t, []int{4, 8}, q.Config.Checkpoints)
	assert.Equal(t, "Alpha", q.Teams[0].Name)
}

func TestSavePreservesTopComment(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "quiz.hujson")

	// Seed a file with a leading comment.
	seed := []byte(`// qlimaster quiz state. Edit by hand carefully.
{
  "version": 1,
  "created": "2026-04-14T19:00:00Z",
  "config": { "rounds": 8, "questions_per_round": 10, "checkpoints": [4, 8] },
  "teams": []
}
`)
	require.NoError(t, os.WriteFile(path, seed, 0o600))

	q, err := store.Load(path)
	require.NoError(t, err)
	// Mutate and save.
	q.Teams = append(q.Teams, quiz.Team{
		ID:     "t_x",
		Name:   "Gamma",
		Scores: map[string]float64{},
	})
	require.NoError(t, store.Save(path, q))

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(after), "qlimaster quiz state")
	assert.Contains(t, string(after), "Gamma")
}

func TestSaveAtomicWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "quiz.hujson")

	q := quiz.New(quiz.DefaultConfig())
	require.NoError(t, store.Save(path, q))

	// No leftover .tmp files.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp", "leftover temp file: %s", e.Name())
	}
}

// TestSaveMapKeyOrderIsDeterministic locks in the sorted map-key output that
// the on-disk quiz.hujson format depends on.
//
// Team.Scores and Config.RoundMaxPoints are Go maps, and map iteration order
// is randomised. encoding/json (v1) sorts map keys when marshalling, so a
// saved file is byte-stable across saves and produces clean diffs in the
// user's version control -- which is the whole point of the
// comment-preserving save path.
//
// Go 1.27 reimplemented encoding/json on top of encoding/json/v2, and v2 does
// NOT sort map keys. Migrating this package to json/v2 without opting into
// json.Deterministic would silently reorder every score on every save. This
// test is the tripwire for that.
func TestSaveMapKeyOrderIsDeterministic(t *testing.T) {
	t.Parallel()

	cfg := quiz.DefaultConfig()
	cfg.RoundMaxPoints = map[string]int{"8": 12, "3": 11, "6": 9, "1": 20, "2": 15}

	q := quiz.Quiz{
		Version: 1,
		Created: time.Date(2026, 4, 14, 19, 0, 0, 0, time.UTC),
		Config:  cfg,
		Teams: []quiz.Team{{
			ID:      "t_a",
			Name:    "Alpha",
			Players: "alice, bob",
			Scores: map[string]float64{
				"1": 4, "2": 2, "3": 6, "4": 8, "5": 5,
				"6": 10, "7": 3, "8": 9,
			},
		}},
	}

	var first string

	// Repeat enough times that randomised map iteration would show up.
	for i := range 20 {
		path := filepath.Join(t.TempDir(), "quiz.hujson")
		require.NoError(t, store.Save(path, q))

		raw, err := os.ReadFile(path)
		require.NoError(t, err)

		if i == 0 {
			first = string(raw)

			continue
		}

		require.Equal(t, first, string(raw),
			"saved bytes must be identical across saves; map keys are no longer sorted")
	}

	// Spell out the property the byte-comparison relies on: keys ascend.
	scoresAt := strings.Index(first, `"scores"`)
	require.Positive(t, scoresAt, "expected a scores object in the saved file")

	positions := make([]int, 0, 8)

	for r := 1; r <= 8; r++ {
		key := `"` + strconv.Itoa(r) + `"`

		p := strings.Index(first[scoresAt:], key)
		require.Positive(t, p, "round key %s missing from scores object", key)

		positions = append(positions, p)
	}

	assert.IsIncreasing(t, positions, "score map keys must be written in sorted order")
}
