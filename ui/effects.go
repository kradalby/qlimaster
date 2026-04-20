package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/quiz"
)

// perfectFlashTickMsg is sent by the perfect-round flash animation.
type perfectFlashTickMsg struct {
	Ref   quiz.PerfectRef
	Phase int
	Until time.Time
}

// winnerGlowTickMsg is sent by the winner glow animation.
type winnerGlowTickMsg struct {
	TeamID string
	Phase  int
	Until  time.Time
}

// flashPerfect schedules a 3-phase pulse at the given (team, round)
// coordinate. A later phase will consume these ticks to alter rendering;
// for now the scheduler is wired and the ticks are no-ops.
func flashPerfect(ref quiz.PerfectRef) tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return perfectFlashTickMsg{
			Ref:   ref,
			Phase: 0,
			Until: t.Add(450 * time.Millisecond),
		}
	})
}

// winnerGlow schedules a one-shot pink->gold->pink pulse on the winning
// team when the final round completes.
func winnerGlow(teamID string) tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return winnerGlowTickMsg{
			TeamID: teamID,
			Phase:  0,
			Until:  t.Add(2 * time.Second),
		}
	})
}

// onPerfectTick advances the perfect-round flash state and returns a
// follow-up Cmd when more phases remain.
func (m Model) onPerfectTick(msg perfectFlashTickMsg) (Model, tea.Cmd) {
	if msg.Phase >= 2 {
		return m, nil
	}
	next := msg
	next.Phase++
	return m, tea.Tick(150*time.Millisecond, func(_ time.Time) tea.Msg { return next })
}

// onWinnerTick advances the winner-glow state and returns a follow-up.
func (m Model) onWinnerTick(msg winnerGlowTickMsg) (Model, tea.Cmd) {
	if time.Now().After(msg.Until) {
		return m, nil
	}
	next := msg
	next.Phase++
	return m, tea.Tick(200*time.Millisecond, func(_ time.Time) tea.Msg { return next })
}
