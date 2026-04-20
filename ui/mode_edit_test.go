package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEditScore_NavigateAndEdit sets up a two-team quiz, enters EditScore,
// navigates to round 1 for the first team, types 5, and confirms the
// score lands in the model.
func TestEditScore_NavigateAndEdit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.Config{Rounds: 3, QuestionsPerRound: 10, Checkpoints: []int{3}},
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Beta"})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})

	// Press 'i' to enter edit mode.
	model, _ = model.Update(teaKey("i"))
	mm, ok := model.(Model)
	require.True(t, ok)
	require.Equal(t, ModeEditScore, mm.mode)

	// Move right twice so editCol points at round 1 (editColRound1).
	model, _ = model.Update(teaKey("l"))
	model, _ = model.Update(teaKey("l"))
	mm, ok = model.(Model)
	require.True(t, ok)
	assert.Equal(t, editColRound1, mm.editCol)
	assert.Equal(t, 1, mm.focusedRound())

	// Enter edit cell, type "5", commit.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	model, _ = model.Update(teaKey("5"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, ok = model.(Model)
	require.True(t, ok)
	assert.False(t, mm.edit.editing)

	// Confirm a score was recorded (for one of the teams -- the first
	// sorted, which by name is Alpha).
	var found bool
	for _, team := range mm.quiz.Teams {
		if v, ok := team.Score(1); ok && v == 5 {
			found = true
		}
	}
	assert.True(t, found, "expected a round 1 score of 5 to be recorded")
}

// TestEditScore_ClearCell clears a previously recorded score via 'x'.
func TestEditScore_ClearCell(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.Config{Rounds: 3, QuestionsPerRound: 10},
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	teamID := m.quiz.Teams[0].ID
	m, _ = m.apply(quiz.ChangeSetScore{TeamID: teamID, Round: 1, Score: 5})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey("i"))
	model, _ = model.Update(teaKey("l"))
	model, _ = model.Update(teaKey("l"))
	model, _ = model.Update(teaKey("x"))
	mm, _ := model.(Model)

	_, ok := mm.quiz.FindTeam(teamID).Score(1)
	assert.False(t, ok)
}

// TestEditScore_InvalidScoreStaysInEdit keeps the flow in editing state
// with an inline error.
func TestEditScore_InvalidScoreStaysInEdit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.Config{Rounds: 3, QuestionsPerRound: 10},
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey("i"))
	model, _ = model.Update(teaKey("l"))
	model, _ = model.Update(teaKey("l"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	// 11 is out of range for questions=10.
	model, _ = model.Update(teaKey("1"))
	model, _ = model.Update(teaKey("1"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.True(t, mm.edit.editing, "should stay in editing state")
	assert.NotEmpty(t, mm.errMsg)
}

// TestEditScore_DeleteTeam requires two 'd' presses to remove a team.
func TestEditScore_DeleteTeam(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Beta"})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey("i"))
	// One 'd' alone should arm but not delete.
	model, _ = model.Update(teaKey("d"))
	mm, _ := model.(Model)
	assert.Len(t, mm.quiz.Teams, 2)
	// A non-'d' key cancels the combo.
	model, _ = model.Update(teaKey("j"))
	mm, _ = model.(Model)
	assert.Len(t, mm.quiz.Teams, 2)
	// Two 'd's delete.
	model, _ = model.Update(teaKey("d"))
	model, _ = model.Update(teaKey("d"))
	mm, _ = model.(Model)
	assert.Len(t, mm.quiz.Teams, 1)
}
