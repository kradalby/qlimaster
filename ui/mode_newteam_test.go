package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTeam_AddsTeam verifies pressing 'a', typing a name, Enter, Enter
// results in a new team.
func TestNewTeam_AddsTeam(t *testing.T) {
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
	model, _ = model.Update(teaKey("a"))

	for _, r := range "Alpha" {
		model, _ = model.Update(teaKey(string(r)))
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	// Players step: just Enter.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, ModeNormal, mm.mode)
	require.Len(t, mm.quiz.Teams, 1)
	assert.Equal(t, "Alpha", mm.quiz.Teams[0].Name)
}

// TestNewTeam_NameWithSpacesPreserved verifies the space key lands in
// the name buffer verbatim and the resulting team keeps its internal
// whitespace through add + render.
func TestNewTeam_NameWithSpacesPreserved(t *testing.T) {
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
	model, _ = model.Update(teaKey("a"))

	// Type "The rookies" including the space. tea.KeyPressMsg for the
	// space key reports String()="space" and Text=" "; the input path
	// must consult Text.
	for _, r := range "The rookies" {
		if r == ' ' {
			model, _ = model.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
			continue
		}
		model, _ = model.Update(teaKey(string(r)))
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	require.Len(t, mm.quiz.Teams, 1)
	assert.Equal(t, "The rookies", mm.quiz.Teams[0].Name)
}

// TestNewTeam_EmptyNameErrors shows an inline error and does not advance.
func TestNewTeam_EmptyNameErrors(t *testing.T) {
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
	model, _ = model.Update(teaKey("a"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, ModeNewTeam, mm.mode)
	assert.NotEmpty(t, mm.errMsg)
}
