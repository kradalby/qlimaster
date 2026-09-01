package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
)

// TestNewTeam_AddsTeam verifies pressing 'a', typing a name, Enter, Enter
// results in a new team.
func TestNewTeam_AddsTeam(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:        filepath.Join(dir, "quiz.hujson"),
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
		HistoryPath: filepath.Join(dir, "history.hujson"),
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
		Path:        filepath.Join(dir, "quiz.hujson"),
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
		HistoryPath: filepath.Join(dir, "history.hujson"),
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

// TestNewTeam_SelectsSuggestion is a regression test: when the user filters
// with a query and arrows onto a suggestion, Enter must save the highlighted
// team, not the raw typed query.
func TestNewTeam_SelectsSuggestion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:        filepath.Join(dir, "quiz.hujson"),
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
		HistoryPath: filepath.Join(dir, "history.hujson"),
	})
	require.NoError(t, err)

	m.history.Teams = []history.Entry{
		{Name: "Alpha", LastSeen: "2026-01-01", TimesSeen: 1},
		{Name: "Alligators", LastSeen: "2026-01-01", TimesSeen: 1},
		{Name: "Alpacas", LastSeen: "2026-01-01", TimesSeen: 1},
	}

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey("a"))

	// Filter to several matches, then arrow onto the second one.
	for _, r := range "al" {
		model, _ = model.Update(teaKey(string(r)))
	}

	mm, _ := model.(Model)
	suggestions := mm.newTeamSuggestions()
	require.GreaterOrEqual(t, len(suggestions), 2, "query must yield >=2 matches")

	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mm, _ = model.(Model)
	want := suggestions[mm.newTeam.suggestIdx].Name
	require.NotEqual(t, "al", want, "must have moved onto a real suggestion")

	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"}) // accept name
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"}) // accept players

	mm, _ = model.(Model)
	require.Len(t, mm.quiz.Teams, 1)
	assert.Equal(t, want, mm.quiz.Teams[0].Name, "saves the selected suggestion, not the query")
}

// TestNewTeam_EmptyNameErrors shows an inline error and does not advance.
func TestNewTeam_EmptyNameErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:        filepath.Join(dir, "quiz.hujson"),
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
		HistoryPath: filepath.Join(dir, "history.hujson"),
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

// TestNewTeam_TypedNameAdoptsHistoryCasing is a regression test for the
// garbage-name bug: history is case-insensitive but styling-preserving,
// so typing a known name in the wrong case must save the stored casing
// instead of forking a second, differently-cased entry.
func TestNewTeam_TypedNameAdoptsHistoryCasing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:        filepath.Join(dir, "quiz.hujson"),
		QuizConfig:  quiz.DefaultConfig(),
		QuizRoot:    dir,
		HistoryPath: filepath.Join(dir, "history.hujson"),
	})
	require.NoError(t, err)

	m.history.Teams = []history.Entry{
		{Name: "Gin Team", LastSeen: "2026-07-21", TimesSeen: 2},
	}

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey("a"))

	for _, r := range "gin team" {
		if r == ' ' {
			model, _ = model.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

			continue
		}

		model, _ = model.Update(teaKey(string(r)))
	}

	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"}) // accept name
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"}) // accept players

	mm, _ := model.(Model)
	require.Len(t, mm.quiz.Teams, 1)
	assert.Equal(t, "Gin Team", mm.quiz.Teams[0].Name)
	assert.Len(t, mm.history.Teams, 1, "no case-variant entry added to history")
}
