package ui

import (
	"path/filepath"
	"testing"
	"testing/synctest"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
)

// TestApply_LiveHistoryUpdate adds a new team and asserts that a history
// save command is emitted and that, when executed, it writes the name
// to the history file.
//
//nolint:paralleltest // synctest bubbles are incompatible with t.Parallel
func TestApply_LiveHistoryUpdate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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
	})
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
//
//nolint:paralleltest // synctest bubbles are incompatible with t.Parallel
func TestApply_ReopenQuizDoesNotRebump(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
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
	})
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

// saveQuizIn writes a quiz holding the named teams to dir/quiz.hujson.
func saveQuizIn(t *testing.T, dir string, names ...string) {
	t.Helper()

	q := quiz.New(quiz.DefaultConfig())

	for _, n := range names {
		var err error

		q, _, err = quiz.Apply(q, quiz.ChangeAddTeam{Name: n})
		require.NoError(t, err)
	}

	require.NoError(t, store.Save(filepath.Join(dir, "quiz.hujson"), q))
}

// TestHistory_TimesSeenStableAcrossSessions runs successive sessions in
// sibling quiz folders and expects every team, each played once, to show
// once rather than growing with each session.
func TestHistory_TimesSeenStableAcrossSessions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	historyPath := filepath.Join(root, "history.hujson")
	saveQuizIn(t, filepath.Join(root, "2026-01-01"), "Alpha")

	session := func(day string) Model {
		t.Helper()

		m, err := New(Config{
			Path:        filepath.Join(root, day, "quiz.hujson"),
			HistoryPath: historyPath,
			QuizRoot:    root,
		})
		require.NoError(t, err)

		return m
	}

	for _, day := range []string{"2026-02-01", "2026-02-08", "2026-02-15"} {
		m, _ := session(day).apply(quiz.ChangeAddTeam{Name: "Team " + day})
		require.NoError(t, m.Flush())
	}

	m := session("2026-03-01")
	require.Len(t, m.history.Teams, 4)

	for _, e := range m.history.Teams {
		assert.Equal(t, 1, e.TimesSeen, e.Name)
	}
}

// TestNew_DefaultQuizRootScansSiblings confirms that without an explicit
// QuizRoot the scan covers the quiz folder's siblings, as --quiz-root
// documents.
func TestNew_DefaultQuizRootScansSiblings(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	saveQuizIn(t, filepath.Join(root, "2026-01-01"), "Alpha")

	m, err := New(Config{Path: filepath.Join(root, "2026-02-01", "quiz.hujson")})
	require.NoError(t, err)
	assert.Contains(t, m.history.Names(), "Alpha")
}
