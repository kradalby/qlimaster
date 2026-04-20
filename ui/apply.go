package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
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
		for _, t := range newQuiz.Teams {
			if quiz.Rank(newQuiz).PositionOf(t.ID) == 1 {
				winnerID = t.ID
				break
			}
		}
		if winnerID != "" {
			cmds = append(cmds, winnerGlow(winnerID))
		}
	}
	_ = res.ReRanked
	return m, tea.Batch(cmds...)
}
