package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnterScoreFlow walks through round -> pick -> score -> back to pick.
func TestEnterScoreFlow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	// Seed two teams.
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Beta"})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})

	// Press 'e' to open EnterScore.
	model, _ = model.Update(teaKey("e"))
	mm, _ := model.(Model)
	assert.Equal(t, ModeEnterScore, mm.mode)
	assert.Equal(t, enterStepRound, mm.enter.step)

	// Type round "1" then Enter.
	model, _ = model.Update(teaKey("1"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ = model.(Model)
	assert.Equal(t, enterStepPick, mm.enter.step)
	assert.Equal(t, 1, mm.enter.round)

	// Pick the highlighted team (Alpha).
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ = model.(Model)
	assert.Equal(t, enterStepScore, mm.enter.step)

	// Type "5" and Enter -> score recorded.
	model, _ = model.Update(teaKey("5"))
	model, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	_ = cmd
	mm, _ = model.(Model)
	// One team still without a round 1 score, so we should be back at pick.
	assert.Equal(t, ModeEnterScore, mm.mode)
	assert.Equal(t, enterStepPick, mm.enter.step)

	// Pick next and score.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	model, _ = model.Update(teaKey("3"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ = model.(Model)
	// All teams scored -> mode returns to Normal.
	assert.Equal(t, ModeNormal, mm.mode)

	// Both teams should have scores for round 1.
	for _, team := range mm.quiz.Teams {
		v, ok := team.Score(1)
		assert.True(t, ok)
		assert.Positive(t, v)
	}
}

// TestEnterScore_InvalidRound shows the inline error and does not advance.
func TestEnterScore_InvalidRound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	model, _ = model.Update(teaKey("e"))
	model, _ = model.Update(teaKey("9"))
	model, _ = model.Update(teaKey("9"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, enterStepRound, mm.enter.step)
	assert.NotEmpty(t, mm.errMsg)
}

// TestEnterScore_EscapeUnwinds confirms Esc rolls back one step at a time.
func TestEnterScore_EscapeUnwinds(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	model, _ = model.Update(teaKey("e"))
	model, _ = model.Update(teaKey("1"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, enterStepPick, mm.enter.step)

	esc := tea.KeyPressMsg{Code: tea.KeyEscape, Text: ""}
	model, _ = model.Update(esc)
	mm, _ = model.(Model)
	assert.Equal(t, enterStepRound, mm.enter.step)

	model, _ = model.Update(esc)
	mm, _ = model.(Model)
	assert.Equal(t, ModeNormal, mm.mode)
}
