package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"tailscale.com/util/set"
)

// apply routes a quiz.Change through quiz.Apply, persists the result, and
// schedules any UI side-effects (save toast, sort animation, perfect-round
// flash, winner glow). This is the single UI entry point for state
// mutations; every keybind that changes state ends here.
func (m Model) apply(c quiz.Change) (Model, tea.Cmd) {
	newQuiz, res, err := quiz.Apply(m.quiz, c)
	if err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	m.errMsg = ""
	m.quiz = newQuiz
	m.lastEntered = computeLastEntered(newQuiz)

	cmds := []tea.Cmd{saveCmd(m.path, newQuiz)}
	if res.Mutated {
		cmds = append(cmds, clearStatusCmd(1200*time.Millisecond))
	}
	for _, p := range res.NewPerfectRounds {
		cmds = append(cmds, flashPerfect(p))
	}
	if res.WinnerDecided {
		winnerID := ""
		ranking := quiz.Rank(newQuiz)
		for _, t := range newQuiz.Teams {
			if ranking.PositionOf(t.ID) == 1 {
				winnerID = t.ID
				break
			}
		}
		if winnerID != "" {
			cmds = append(cmds, winnerGlow(winnerID))
		}
	}
	_ = res.ReRanked

	// Live-update the global team-name history whenever a fresh name
	// appears. This keeps the fuzzy add-team suggestions useful across
	// sessions without requiring a manual `qlimaster history rebuild`.
	var historyCmd tea.Cmd
	m, historyCmd = m.maybeRecordNewNames()
	if historyCmd != nil {
		cmds = append(cmds, historyCmd)
	}

	return m, tea.Batch(cmds...)
}

// maybeRecordNewNames checks the current quiz for any team name not yet
// tracked in this session's sessionRecordedNames set. For each fresh
// name it updates the in-memory history and schedules an async save.
// The session set prevents double-bumping TimesSeen when the same team
// is mutated repeatedly (score edits, rename, etc.) within one run.
func (m Model) maybeRecordNewNames() (Model, tea.Cmd) {
	if m.historyPath == "" {
		return m, nil
	}
	if m.sessionRecordedNames == nil {
		m.sessionRecordedNames = set.Set[string]{}
	}

	fresh := make([]string, 0)
	for _, t := range m.quiz.Teams {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if m.sessionRecordedNames.Contains(key) {
			continue
		}
		m.sessionRecordedNames.Add(key)
		fresh = append(fresh, name)
	}
	if len(fresh) == 0 {
		return m, nil
	}
	m.history = history.RecordNames(m.history, fresh, time.Now())
	return m, historySaveCmd(m.historyPath, m.history)
}
