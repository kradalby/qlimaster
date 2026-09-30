package ui

import (
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
)

// newSaveModel builds a Model with its quiz and history files in a fresh
// temp dir.
func newSaveModel(t *testing.T) (Model, string, string) {
	t.Helper()

	dir := t.TempDir()
	quizPath := filepath.Join(dir, "quiz.hujson")
	historyPath := filepath.Join(dir, "history.hujson")

	m, err := New(Config{Path: quizPath, HistoryPath: historyPath, QuizRoot: dir})
	require.NoError(t, err)

	return m, quizPath, historyPath
}

// assertOnDisk checks both files hold the named teams.
func assertOnDisk(t *testing.T, quizPath, historyPath string, names ...string) {
	t.Helper()

	q, err := store.Load(quizPath)
	require.NoError(t, err)

	got := make([]string, 0, len(q.Teams))
	for _, tm := range q.Teams {
		got = append(got, tm.Name)
	}

	assert.ElementsMatch(t, names, got, "quiz file")

	h, err := history.Load(historyPath)
	require.NoError(t, err)
	assert.ElementsMatch(t, names, h.Names(), "history file")
}

// TestSave_StaleSnapshotNeverWins runs the save commands of two changes
// newest-first, an order bubbletea's per-command goroutines allow, and
// expects the newest state to stay on disk.
//
//nolint:paralleltest // synctest bubbles are incompatible with t.Parallel
func TestSave_StaleSnapshotNeverWins(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, quizPath, historyPath := newSaveModel(t)

		m, first := m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
		_, second := m.apply(quiz.ChangeAddTeam{Name: "Beta"})

		drainBatch(second)
		drainBatch(first)

		assertOnDisk(t, quizPath, historyPath, "Alpha", "Beta")
	})
}

// TestSave_FlushWritesPendingChanges covers quitting before the save
// commands ran: Flush must persist the final model, and a stale command
// finishing afterwards must not overwrite it.
//
//nolint:paralleltest // synctest bubbles are incompatible with t.Parallel
func TestSave_FlushWritesPendingChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, quizPath, historyPath := newSaveModel(t)

		m, first := m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
		m, _ = m.apply(quiz.ChangeAddTeam{Name: "Beta"})

		require.NoError(t, m.Flush())
		assertOnDisk(t, quizPath, historyPath, "Alpha", "Beta")

		drainBatch(first)
		assertOnDisk(t, quizPath, historyPath, "Alpha", "Beta")
	})
}
