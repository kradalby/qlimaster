package ui

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModel_Init creates a Model against a fresh quiz file and sends the
// initial window-size message; the rendered view must be non-empty.
func TestModel_Init(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	v := model.View()
	assert.NotEmpty(t, v.Content)
}

// TestModel_Apply exercises the quiz.Apply wrapper and the save command.
func TestModel_Apply(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	m, cmd := m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	require.NotNil(t, cmd)
	// Run the save command synchronously to ensure it doesn't error.
	msg := cmd()
	// cmd is tea.Batch(saveCmd, clearStatusCmd) - resolve the BatchMsg.
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, sub := range batch {
			_ = sub()
		}
	}
	assert.Len(t, m.quiz.Teams, 1)
	assert.Equal(t, "Alpha", m.quiz.Teams[0].Name)
}

// TestModel_HelpToggle verifies pressing '?' opens and dismisses the help
// overlay.
func TestModel_HelpToggle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Press '?': should enter help mode.
	model, _ = model.Update(teaKey("?"))
	mm, ok := model.(Model)
	require.True(t, ok)
	assert.Equal(t, ModeHelp, mm.mode)

	// Press '?' again: back to Normal.
	model, _ = model.Update(teaKey("?"))
	mm, ok = model.(Model)
	require.True(t, ok)
	assert.Equal(t, ModeNormal, mm.mode)
}

// TestModel_SavedMsg exercises the toast clearing path.
func TestModel_SavedMsg(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m
	model, _ = model.Update(savedMsg{When: time.Now(), Err: nil})
	mm, ok := model.(Model)
	require.True(t, ok)
	assert.Contains(t, mm.status, "saved")
}

// teaKey builds a tea.KeyPressMsg from a single-rune string.
func teaKey(s string) tea.KeyPressMsg {
	runes := []rune(s)
	return tea.KeyPressMsg{Code: runes[0], Text: s}
}
