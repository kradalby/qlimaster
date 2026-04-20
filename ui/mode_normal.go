package ui

import (
	tea "charm.land/bubbletea/v2"
)

// handleKey dispatches a key press to the appropriate mode handler.
// k is the Bubble Tea string form of the key (used for binding matching
// like "enter", "esc", "ctrl+c"). text is msg.Text, the raw character(s)
// actually typed, used when populating text-input buffers. The space
// key for example has k=="space" but text==" ", and we want the literal
// space in buffers.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	km := DefaultKeyMap()
	k := msg.String()
	text := msg.Text

	// Global unconditional bindings.
	if matches(km.ForceQuit, k) {
		return m, tea.Quit
	}

	// Help overlay intercepts everything.
	if m.mode == ModeHelp {
		if matches(km.ToggleHelp, k) || matches(km.Escape, k) {
			m.mode = ModeNormal
		}
		return m, nil
	}

	// Export overlay intercepts its own keys.
	if m.mode == ModeExport {
		return m.handleExportKey(k, km)
	}

	switch m.mode {
	case ModeNormal:
		return m.handleNormalKey(k, km)
	case ModeEnterScore:
		return m.handleEnterKey(k, text, km)
	case ModeEditScore:
		return m.handleEditKey(k, text, km)
	case ModeNewTeam:
		return m.handleNewTeamKey(k, text, km)
	case ModeConfig:
		return m.handleConfigKey(k, text, km)
	case ModeReadOut:
		return m.handleReadOutKey(k, km)
	default:
		if matches(km.Escape, k) {
			m.mode = ModeNormal
		}
		return m, nil
	}
}

func (m Model) handleNormalKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Quit, k):
		return m, tea.Quit
	case matches(km.ToggleHelp, k):
		m.mode = ModeHelp
	case matches(km.Export, k):
		m.mode = ModeExport
	case matches(km.EnterScore, k):
		return m.startEnterScore(), nil
	case matches(km.EditScore, k):
		return m.startEditScore(), nil
	case matches(km.AddTeam, k):
		return m.startNewTeam(), nil
	case matches(km.Config, k):
		return m.startConfig(), nil
	case matches(km.ReadOut, k):
		return m.startReadOut(), nil
	case matches(km.Up, k):
		if m.rowCursor > 0 {
			m.rowCursor--
		}
	case matches(km.Down, k):
		if m.rowCursor < len(m.quiz.Teams)-1 {
			m.rowCursor++
		}
	case matches(km.Top, k):
		m.rowCursor = 0
	case matches(km.Bottom, k):
		m.rowCursor = max(len(m.quiz.Teams)-1, 0)
	}
	return m, nil
}

func (m Model) handleExportKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Escape, k):
		m.mode = ModeNormal
	case matches(km.ExportCSV, k):
		return m.exportTo("csv")
	case matches(km.ExportXLSX, k):
		return m.exportTo("xlsx")
	case matches(km.ExportBoth, k):
		return m.exportTo("both")
	}
	return m, nil
}
