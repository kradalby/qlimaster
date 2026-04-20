package history_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "history.hujson")
	h := history.History{
		Version: 1,
		Teams: []history.Entry{
			{Name: "Alpha", LastSeen: "2026-04-14", TimesSeen: 3},
			{Name: "Beta", LastSeen: "2025-11-01", TimesSeen: 1},
		},
	}
	require.NoError(t, history.Save(path, h))

	loaded, err := history.Load(path)
	require.NoError(t, err)
	require.Len(t, loaded.Teams, 2)
	assert.Equal(t, "Alpha", loaded.Teams[0].Name)
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	t.Parallel()

	h, err := history.Load(filepath.Join(t.TempDir(), "nope.hujson"))
	require.NoError(t, err)
	assert.Equal(t, 1, h.Version)
	assert.Empty(t, h.Teams)
}

func TestMerge_Dedup(t *testing.T) {
	t.Parallel()

	a := history.History{
		Teams: []history.Entry{
			{Name: "Alpha", LastSeen: "2026-04-14", TimesSeen: 2},
			{Name: "Beta", LastSeen: "2025-11-01", TimesSeen: 1},
		},
	}
	b := history.History{
		Teams: []history.Entry{
			{Name: "alpha", LastSeen: "2025-02-10", TimesSeen: 1}, // dup
			{Name: "Gamma", LastSeen: "2026-01-01", TimesSeen: 2},
		},
	}
	m := history.Merge(a, b)
	// Alpha + alpha -> one entry with TimesSeen=3.
	for _, e := range m.Teams {
		if e.Name == "Alpha" {
			assert.Equal(t, 3, e.TimesSeen)
			assert.Equal(t, "2026-04-14", e.LastSeen)
		}
	}
	// Sorted most-recent first.
	assert.Equal(t, "Alpha", m.Teams[0].Name)
}

func TestRecordNames_DedupesAndBumpsOnce(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 4, 14, 19, 0, 0, 0, time.UTC)
	h := history.RecordNames(history.History{Version: 1},
		[]string{"Alpha", "alpha", "  Beta ", "", "Alpha"},
		date)
	require.Len(t, h.Teams, 2)

	byName := map[string]history.Entry{}
	for _, e := range h.Teams {
		byName[e.Name] = e
	}
	// Alpha was named twice with the same casing and once as "alpha",
	// but RecordNames is one-session-one-bump, so TimesSeen=1.
	assert.Equal(t, 1, byName["Alpha"].TimesSeen)
	assert.Equal(t, 1, byName["Beta"].TimesSeen)
	assert.Equal(t, "2026-04-14", byName["Alpha"].LastSeen)
}

func TestRecordNames_MergesWithExisting(t *testing.T) {
	t.Parallel()

	older := history.History{
		Version: 1,
		Teams: []history.Entry{
			{Name: "Alpha", LastSeen: "2025-11-01", TimesSeen: 2},
		},
	}
	date := time.Date(2026, 4, 14, 19, 0, 0, 0, time.UTC)
	h := history.RecordNames(older, []string{"alpha", "Beta"}, date)

	// Merge preserves the most-recently-seen casing; "alpha" (this
	// session) wins over "Alpha" (older entry).
	byKey := map[string]history.Entry{}
	for _, e := range h.Teams {
		byKey[strings.ToLower(e.Name)] = e
	}
	// Alpha: pre-existing 2 + this session 1 = 3; LastSeen advances.
	assert.Equal(t, 3, byKey["alpha"].TimesSeen)
	assert.Equal(t, "2026-04-14", byKey["alpha"].LastSeen)
	assert.Equal(t, 1, byKey["beta"].TimesSeen)
}

func TestRecordQuiz(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{
		Teams: []quiz.Team{
			{Name: "Alpha"},
			{Name: "Beta"},
			{Name: ""}, // skipped
		},
	}
	date := time.Date(2026, 4, 14, 19, 0, 0, 0, time.UTC)
	h := history.RecordQuiz(history.History{}, q, date)
	require.Len(t, h.Teams, 2)

	// Record same quiz again -> times_seen increments.
	h = history.RecordQuiz(h, q, date)
	for _, e := range h.Teams {
		assert.Equal(t, 2, e.TimesSeen)
	}
}

func TestScan_SiblingFolders(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Folder 1: 2026-04-14-quiz
	writeQuiz(t, filepath.Join(root, "2026-04-14-quiz", "quiz.hujson"), []string{"Alpha", "Beta"})
	// Folder 2: 2025-11-01
	writeQuiz(t, filepath.Join(root, "2025-11-01", "quiz.hujson"), []string{"Alpha", "Gamma"})
	// Folder 3: misc; should fall back to mtime-date
	writeQuiz(t, filepath.Join(root, "misc-folder", "quiz.hujson"), []string{"Delta"})
	// Plain file in root is ignored.
	require.NoError(t, os.WriteFile(filepath.Join(root, "note.txt"), []byte("hi"), 0o600))

	h, err := history.Scan(root)
	require.NoError(t, err)

	names := map[string]history.Entry{}
	for _, e := range h.Teams {
		names[e.Name] = e
	}
	assert.Contains(t, names, "Alpha")
	assert.Contains(t, names, "Beta")
	assert.Contains(t, names, "Gamma")
	assert.Contains(t, names, "Delta")
	// Alpha appeared in two quizzes.
	assert.Equal(t, 2, names["Alpha"].TimesSeen)
}

func writeQuiz(t *testing.T, path string, teamNames []string) {
	t.Helper()
	q := quiz.New(quiz.DefaultConfig())
	for _, n := range teamNames {
		q.Teams = append(q.Teams, quiz.Team{ID: n, Name: n, Scores: map[string]float64{}})
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, store.Save(path, q))
}
