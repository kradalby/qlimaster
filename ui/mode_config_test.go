package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_UpdatesQuiz walks through the config flow and confirms the
// quiz config is mutated.
func TestConfig_UpdatesQuiz(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	mm, _ := model.(Model)
	require.Equal(t, ModeConfig, mm.mode)

	// Select all of the rounds field and replace with "6".
	for range 2 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}
	model, _ = model.Update(teaKey("6"))

	// Tab to questions, keep.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	// Tab to checkpoints, replace with "3,6".
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	for range 3 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}
	model, _ = model.Update(teaKey("3"))
	model, _ = model.Update(teaKey(","))
	model, _ = model.Update(teaKey("6"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ = model.(Model)
	assert.Equal(t, ModeNormal, mm.mode)
	assert.Equal(t, 6, mm.quiz.Config.Rounds)
	assert.Equal(t, []int{3, 6}, mm.quiz.Config.Checkpoints)
}

// TestConfig_RejectsInvalid shows inline error on unparseable input.
func TestConfig_RejectsInvalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	for range 2 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}
	model, _ = model.Update(teaKey("x")) // not a number
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, ModeConfig, mm.mode)
	assert.NotEmpty(t, mm.errMsg)
}
