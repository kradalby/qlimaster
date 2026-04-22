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
	// "x" is not a digit and is now silently dropped by configAppend's
	// character filter. Submitting with an empty Rounds field must still
	// surface the inline error.
	model, _ = model.Update(teaKey("x"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, ModeConfig, mm.mode)
	assert.NotEmpty(t, mm.errMsg)
}

// TestConfig_IgnoresEscapeSequenceNoise is a regression test for the
// "new quiz opens with garbage Rounds value" bug. Stray escape
// sequences (e.g. a cursor position report leaking through the key
// path) arriving in tea.KeyPressMsg.Text must not be appended to the
// numeric form fields - not even the digits inside them.
func TestConfig_IgnoresEscapeSequenceNoise(t *testing.T) {
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

	// Simulate the exact payload from the reported bug: ESC plus CSI
	// introducer plus a cursor-position-report tail, all arriving in a
	// single Text blob. The embedded digits (7,5,1) must not leak
	// through - the whole blob is noise and must be rejected wholesale.
	noise := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "\x1b[75;1R"}
	model, _ = model.Update(noise)
	// A raw-control-char variant (DEL + other noise) - also rejected.
	raw := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "\x7f[1;2R"}
	model, _ = model.Update(raw)
	// Plain-text garbage with embedded digits contains no control
	// characters, so it reaches configAppend. There the per-field
	// digit filter keeps the "3" and drops "abc" and "def".
	mixed := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "abc3def"}
	model, _ = model.Update(mixed)

	mm, _ = model.(Model)
	// Seed "8" plus the lone "3" that survived the mixed payload.
	// Nothing from the escape-sequence blobs landed in the field.
	assert.Equal(t, "83", mm.configEdit.rounds)
	assert.NotContains(t, mm.configEdit.rounds, "7")
	assert.NotContains(t, mm.configEdit.rounds, "5")
	assert.NotContains(t, mm.configEdit.rounds, "1")
	assert.NotContains(t, mm.configEdit.rounds, "[")
	assert.NotContains(t, mm.configEdit.rounds, "R")
	assert.NotContains(t, mm.configEdit.rounds, "\x1b")
}

// TestConfig_CheckpointsAllowsCommaAndSpace confirms the Checkpoints
// field accepts the separator characters it needs while still
// filtering out non-digit noise.
func TestConfig_CheckpointsAllowsCommaAndSpace(t *testing.T) {
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
	// Tab past Rounds and Questions to land on Checkpoints.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	// Erase the default "4,8".
	for range 3 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}
	// A mix of legal (digits, comma, space) and illegal (letters)
	// characters with NO control codes - the whole blob reaches
	// configAppend and the per-field filter strips just the letter.
	payload := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "2, 5,x9"}
	model, _ = model.Update(payload)
	mm, _ := model.(Model)
	assert.Equal(t, "2, 5,9", mm.configEdit.checkpoints)
}
