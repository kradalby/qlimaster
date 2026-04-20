package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
