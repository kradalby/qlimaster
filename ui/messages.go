package ui

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
)

// savedMsg is emitted by the async save command. Err is non-nil when the
// save failed; the UI will display it inline rather than silently losing
// state.
type savedMsg struct {
	When time.Time
	Err  error
}

// clearStatusMsg is scheduled by toast-style status updates so they fade
// out after a short delay rather than lingering forever.
type clearStatusMsg struct{}

// saveCmd returns a tea.Cmd that persists snapshot seq of the quiz and
// emits savedMsg.
func saveCmd(s *saver[quiz.Quiz], seq uint64, q quiz.Quiz) tea.Cmd {
	return func() tea.Msg {
		return savedMsg{When: time.Now(), Err: s.save(seq, q)}
	}
}

// clearStatusCmd schedules clearStatusMsg after d.
func clearStatusCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// historySavedMsg is emitted by the async history-save command. An error
// is recorded so the UI can surface it as a toast, but history save
// failures never abort the quiz save itself.
type historySavedMsg struct {
	When time.Time
	Err  error
}

// historySaveCmd persists snapshot seq of the team-name history
// asynchronously.
func historySaveCmd(s *saver[history.History], seq uint64, h history.History) tea.Cmd {
	return func() tea.Msg {
		return historySavedMsg{When: time.Now(), Err: s.save(seq, h)}
	}
}

// onSaved folds a savedMsg into the model, producing a status toast.
func (m Model) onSaved(msg savedMsg) Model {
	if msg.Err != nil {
		m.status = "save failed: " + msg.Err.Error()

		m.errMsg = msg.Err.Error()
		if errors.Is(msg.Err, store.ErrNotFound) {
			m.errMsg = "quiz file not found"
		}
	} else {
		m.status = "[ok] saved"
	}

	m.statusExpiry = msg.When.Add(1200 * time.Millisecond)

	return m
}
