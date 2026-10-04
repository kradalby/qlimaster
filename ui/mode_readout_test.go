package ui

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kradalby/qlimaster/quiz"
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

// TestReadOutTiedTeams confirms ties include the team itself and preserve
// the order in which the stats are read out.
func TestReadOutTiedTeams(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{
		Version: 1,
		Config:  quiz.Config{Rounds: 1, QuestionsPerRound: 10},
		Teams: []quiz.Team{
			{ID: "a", Name: "Alpha", Scores: map[string]float64{"1": 5}},
			{ID: "b", Name: "Beta", Scores: map[string]float64{"1": 5}},
			{ID: "c", Name: "Gamma", Scores: map[string]float64{"1": 7}},
		},
	}

	order := readOutOrder(q)
	for _, team := range q.Teams[:2] {
		tied := readOutTiedTeams(order, team)
		require.Len(t, tied, 2)
		assert.Equal(t, "Beta", tied[0].Name)
		assert.Equal(t, "Alpha", tied[1].Name)
	}

	tied := readOutTiedTeams(order, q.Teams[2])
	require.Len(t, tied, 1)
	assert.Equal(t, "Gamma", tied[0].Name)
}

// Each tied team's card must retain the whole roster while selecting only
// that team's round scores and checkpoints.
func TestReadOut_SharedRosterNavigation(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{Config: quiz.Config{Rounds: 2, QuestionsPerRound: 10, Checkpoints: []int{1, 2}}, Teams: []quiz.Team{
		{ID: "a", Name: "Alpha", Scores: map[string]float64{"1": 5, "2": 5}},
		{ID: "b", Name: "Beta", Scores: map[string]float64{"1": 4, "2": 6}},
		{ID: "c", Name: "Gamma", Scores: map[string]float64{"1": 9, "2": 1}},
		{ID: "d", Name: "Delta", Scores: map[string]float64{"1": 8, "2": 8}},
		{ID: "e", Name: "Echo", Scores: map[string]float64{"1": 2, "2": 2}},
	}}
	m := Model{quiz: q, width: 140, height: 40}.startReadOut()

	for i, tc := range []struct {
		name, stats, round1, round2, checkpoint string
	}{
		{"Gamma", "Stats for Gamma · 1 / 3", "Round 1   9", "Round 2   1", "Score after R1  9"},
		{"Beta", "Stats for Beta · 2 / 3", "Round 1   4", "Round 2   6", "Score after R1  4"},
		{"Alpha", "Stats for Alpha · 3 / 3", "Round 1   5", "Round 2   5", "Score after R1  5"},
	} {
		model, _ := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
		m, _ = model.(Model)
		require.Equal(t, i+1, m.readOut.idx)

		view := ansi.Strip(m.renderReadOut())
		assert.Contains(t, view, "Shared position 2")
		assert.Contains(t, view, "10 points each")
		assert.Contains(t, view, "Gamma")
		assert.Contains(t, view, "Beta")
		assert.Contains(t, view, "Alpha")
		assert.NotContains(t, view, "Delta")
		assert.NotContains(t, view, "Echo")
		assert.Contains(t, view, "▶ "+tc.name)
		assert.Equal(t, 1, strings.Count(view, "▶"))
		assert.Contains(t, view, tc.stats)
		assert.Contains(t, view, tc.round1)
		assert.Contains(t, view, tc.round2)
		assert.Contains(t, view, tc.checkpoint)
		assert.Equal(t, 1, strings.Count(view, "Round 1"))
		assert.Equal(t, 1, strings.Count(view, "Round 2"))
		assert.Less(t, strings.Index(view, "Gamma"), strings.Index(view, "Beta"))
		assert.Less(t, strings.Index(view, "Beta"), strings.Index(view, "Alpha"))
	}

	model, _ := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m, _ = model.(Model)
	view := ansi.Strip(m.renderReadOut())
	assert.Contains(t, view, "Delta")
	assert.NotContains(t, view, "Stats for")
	assert.NotContains(t, view, "Alpha")

	model, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m, _ = model.(Model)
	assert.Contains(t, ansi.Strip(m.renderReadOut()), "Stats for Alpha · 3 / 3")

	model, _ = m.Update(teaKey("g"))
	m, _ = model.(Model)
	assert.Contains(t, ansi.Strip(m.renderReadOut()), "Echo")
	model, _ = m.Update(teaKey("G"))
	m, _ = model.(Model)
	assert.Contains(t, ansi.Strip(m.renderReadOut()), "Delta")
}

func TestReadOut_SharedWinners(t *testing.T) {
	t.Parallel()

	for _, complete := range []bool{false, true} {
		t.Run(strconv.FormatBool(complete), func(t *testing.T) {
			t.Parallel()

			q := quiz.Quiz{Config: quiz.Config{Rounds: 2, QuestionsPerRound: 10}, Teams: []quiz.Team{
				{ID: "a", Name: "Alpha", Scores: map[string]float64{"1": 5}},
				{ID: "b", Name: "Beta", Scores: map[string]float64{"1": 5}},
			}}
			if complete {
				q.Teams[0].Scores["2"] = 0
				q.Teams[1].Scores["2"] = 0
			}

			for idx, name := range []string{"Beta", "Alpha"} {
				m := Model{quiz: q, width: 140, height: 40, readOut: readOutState{idx: idx}}
				view := ansi.Strip(m.renderReadOut())
				assert.Contains(t, view, "Shared position 1")
				assert.Contains(t, view, "▶ "+name)

				if complete {
					assert.Contains(t, view, "W  I  N  N  E  R  S")
				} else {
					assert.NotContains(t, view, "W  I  N  N  E  R")
					assert.Contains(t, view, "Round 2   —")
				}
			}
		})
	}
}

func TestReadOut_SharedRosterUsesTeamID(t *testing.T) {
	t.Parallel()

	q := quiz.Quiz{Config: quiz.Config{Rounds: 1, QuestionsPerRound: 10}, Teams: []quiz.Team{
		{ID: "a", Name: "Same name", Scores: map[string]float64{"1": 5.5}},
		{ID: "b", Name: "Same name", Scores: map[string]float64{"1": 5.5}},
	}}
	for idx, progress := range []string{"1 / 2", "2 / 2"} {
		m := Model{quiz: q, width: 140, height: 40, readOut: readOutState{idx: idx}}
		view := ansi.Strip(m.renderReadOut())
		assert.Equal(t, 1, strings.Count(view, "▶"))
		assert.Contains(t, view, "Stats for Same name · "+progress)
		assert.Contains(t, view, "5,5 points each")
	}
}

func TestReadOut_SharedRosterFitsViewport(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		width, height int
		names         []string
	}{
		{"small terminal", 80, 24, []string{"Alpha", "Beta", "Gamma"}},
		{"large tie", 100, 40, []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta"}},
		{"long names", 80, 40, []string{
			"The Remarkably Long Named Thursday Quiz Collective", "Beta and the Masters of Overthinking", "Alpha",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			q := quiz.Quiz{Config: quiz.DefaultConfig()}
			for i, name := range tc.names {
				q.Teams = append(q.Teams, quiz.Team{ID: strconv.Itoa(i), Name: name, Scores: map[string]float64{"1": 5}})
			}

			for idx := range q.Teams {
				m := Model{quiz: q, width: tc.width, height: tc.height, readOut: readOutState{idx: idx}}

				view := ansi.Strip(m.renderReadOut())
				for line := range strings.SplitSeq(view, "\n") {
					assert.LessOrEqual(t, ansi.StringWidth(line), tc.width)
				}

				assert.LessOrEqual(t, strings.Count(view, "\n")+1, tc.height)

				for _, name := range tc.names {
					assert.Contains(t, strings.Join(strings.Fields(view), ""), strings.Join(strings.Fields(name), ""))
				}

				assert.Equal(t, 1, strings.Count(view, "▶"))
				assert.Contains(t, view, "Round 8")
				assert.Contains(t, view, "Stats for")
			}
		})
	}
}
