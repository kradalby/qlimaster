package ui

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"
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

// saveCmd returns a tea.Cmd that persists q to path and emits savedMsg.
func saveCmd(path string, q quiz.Quiz) tea.Cmd {
	return func() tea.Msg {
		err := store.Save(path, q)
		return savedMsg{When: time.Now(), Err: err}
	}
}

// clearStatusCmd schedules clearStatusMsg after d.
func clearStatusCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
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
