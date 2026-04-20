package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApply_LiveHistoryUpdate adds a new team and asserts that a history
// save command is emitted and that, when executed, it writes the name
// to the history file.
func TestApply_LiveHistoryUpdate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.hujson")
	m, err := New(Config{
		Path:        filepath.Join(dir, "quiz.hujson"),
		HistoryPath: historyPath,
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
	})
	require.NoError(t, err)

	_, cmd := m.apply(quiz.ChangeAddTeam{Name: "The rookies"})
	require.NotNil(t, cmd)

	// Drain the batch; one of the sub-commands is the history save.
	drainBatch(cmd)

	// Expect the history file to exist and contain the team name.
	h, err := history.Load(historyPath)
	require.NoError(t, err)
	var found bool
	for _, e := range h.Teams {
		if e.Name == "The rookies" {
			found = true
			assert.Equal(t, 1, e.TimesSeen)
		}
	}
	assert.True(t, found, "expected 'The rookies' in saved history file")
}

// TestApply_SessionDedupe confirms that repeated mutations on the same
// team do not repeatedly bump TimesSeen in the history.
func TestApply_SessionDedupe(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.hujson")
	m, err := New(Config{
		Path:        filepath.Join(dir, "quiz.hujson"),
		HistoryPath: historyPath,
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
	})
	require.NoError(t, err)

	// Add a team once.
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	// Persist the first save synchronously so we can read back.
	require.NoError(t, history.Save(historyPath, m.history))

	teamID := m.quiz.Teams[0].ID
	// Mutate scores repeatedly.
	m, _ = m.apply(quiz.ChangeSetScore{TeamID: teamID, Round: 1, Score: 5})
	m, _ = m.apply(quiz.ChangeSetScore{TeamID: teamID, Round: 2, Score: 7})
	m, _ = m.apply(quiz.ChangeSetScore{TeamID: teamID, Round: 3, Score: 9})
	require.NoError(t, history.Save(historyPath, m.history))

	h, err := history.Load(historyPath)
	require.NoError(t, err)
	require.Len(t, h.Teams, 1)
	assert.Equal(t, 1, h.Teams[0].TimesSeen,
		"TimesSeen must bump once per session regardless of score edits")
}

// TestApply_ReopenQuizDoesNotRebump confirms that loading a quiz file
// that already contains teams does not cause those names to be recorded
// again on the next mutation.
func TestApply_ReopenQuizDoesNotRebump(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	historyPath := filepath.Join(dir, "history.hujson")
	quizPath := filepath.Join(dir, "quiz.hujson")

	// Session 1: create quiz, add team. Persist both files synchronously
	// so session 2 can reopen them.
	m, err := New(Config{
		Path:        quizPath,
		HistoryPath: historyPath,
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
	})
	require.NoError(t, err)
	m, cmd := m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	drainBatch(cmd)
	// Also explicitly persist in case the async save race left any
	// remaining state unsaved.
	require.NoError(t, history.Save(historyPath, m.history))

	// Session 2: reopen. The constructor seeds sessionRecordedNames
	// with Alpha, so a score edit must not re-bump TimesSeen.
	m2, err := New(Config{
		Path:        quizPath,
		HistoryPath: historyPath,
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
	})
	require.NoError(t, err)
	require.Len(t, m2.quiz.Teams, 1)
	teamID := m2.quiz.Teams[0].ID
	m2, cmd2 := m2.apply(quiz.ChangeSetScore{TeamID: teamID, Round: 1, Score: 5})
	drainBatch(cmd2)
	require.NoError(t, history.Save(historyPath, m2.history))

	h, err := history.Load(historyPath)
	require.NoError(t, err)
	require.Len(t, h.Teams, 1)
	// Still 1; reopening must not rebump TimesSeen.
	assert.Equal(t, 1, h.Teams[0].TimesSeen)
}

// drainBatch runs a tea.Cmd and, if it produced a BatchMsg, runs every
// sub-command inside it. Used by tests to force async saves to complete.
func drainBatch(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			if sub != nil {
				_ = sub()
			}
		}
	}
}
