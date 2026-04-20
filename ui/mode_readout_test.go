package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadOut_EnterAndAdvance verifies 'R' enters the mode at the
// worst-ranked team and Space advances toward the best.
func TestReadOut_EnterAndAdvance(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.Config{Rounds: 1, QuestionsPerRound: 10},
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Alpha"})
	m, _ = m.apply(quiz.ChangeAddTeam{Name: "Beta"})
	// Give totals so ordering is unambiguous: Alpha=3, Beta=7.
	m, _ = m.apply(quiz.ChangeSetScore{TeamID: m.quiz.Teams[0].ID, Round: 1, Score: 3})
	m, _ = m.apply(quiz.ChangeSetScore{TeamID: m.quiz.Teams[1].ID, Round: 1, Score: 7})

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	model, _ = model.Update(teaKey("R"))
	mm, _ := model.(Model)
	require.Equal(t, ModeReadOut, mm.mode)

	// worst-first order: Alpha (3) then Beta (7).
	worstFirst := readOutOrder(mm.quiz)
	require.Len(t, worstFirst, 2)
	assert.Equal(t, "Alpha", worstFirst[0].Name)
	assert.Equal(t, "Beta", worstFirst[1].Name)

	// Start at idx 0.
	assert.Equal(t, 0, mm.readOut.idx)

	// Space advances.
	model, _ = model.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	mm, _ = model.(Model)
	assert.Equal(t, 1, mm.readOut.idx)

	// Already at last; Space stays.
	model, _ = model.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	mm, _ = model.(Model)
	assert.Equal(t, 1, mm.readOut.idx)

	// Up goes back.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Text: ""})
	mm, _ = model.(Model)
	assert.Equal(t, 0, mm.readOut.idx)

	// Esc exits.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Text: ""})
	mm, _ = model.(Model)
	assert.Equal(t, ModeNormal, mm.mode)
}

// TestReadOutOrder confirms the worst-to-best ordering.
func TestReadOutOrder(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{
		Version: 1,
		Config:  quiz.Config{Rounds: 1, QuestionsPerRound: 10},
		Teams: []quiz.Team{
			{ID: "a", Name: "Alpha", Scores: map[string]float64{"1": 3}},
			{ID: "b", Name: "Beta", Scores: map[string]float64{"1": 7}},
			{ID: "c", Name: "Gamma", Scores: map[string]float64{"1": 5}},
		},
	}
	order := readOutOrder(q)
	require.Len(t, order, 3)
	assert.Equal(t, "Alpha", order[0].Name)
	assert.Equal(t, "Gamma", order[1].Name)
	assert.Equal(t, "Beta", order[2].Name)
}
