package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/quiz"
)

// configField enumerates the form fields in the Config overlay.
type configField int

const (
	configFieldRounds configField = iota
	configFieldQuestions
	configFieldCheckpoints
	configFieldCount
)

// configState is the ephemeral state for ModeConfig.
type configState struct {
	focus       configField
	rounds      string
	questions   string
	checkpoints string
}

// startConfig seeds the overlay with the current config values and opens
// the mode.
func (m Model) startConfig() Model {
	m.mode = ModeConfig
	m.configEdit = configState{
		focus:       configFieldRounds,
		rounds:      strconv.Itoa(m.quiz.Config.Rounds),
		questions:   strconv.Itoa(m.quiz.Config.QuestionsPerRound),
		checkpoints: joinInts(m.quiz.Config.Checkpoints),
	}
	m.errMsg = ""
	return m
}

// handleConfigKey dispatches keys while ModeConfig is active.
func (m Model) handleConfigKey(k, text string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Escape, k):
		m.mode = ModeNormal
		m.configEdit = configState{}
		m.errMsg = ""
		return m, nil
	case matches(km.Tab, k):
		m.configEdit.focus = (m.configEdit.focus + 1) % configFieldCount
		return m, nil
	case k == "shift+tab":
		m.configEdit.focus = (m.configEdit.focus + configFieldCount - 1) % configFieldCount
		return m, nil
	case matches(km.Enter, k):
		return m.submitConfig()
	case k == keyBackspace:
		return m.configDelete(), nil
	}
	if text != "" {
		return m.configAppend(text), nil
	}
	return m, nil
}

func (m Model) configAppend(k string) Model {
	switch m.configEdit.focus {
	case configFieldRounds:
		m.configEdit.rounds += k
	case configFieldQuestions:
		m.configEdit.questions += k
	case configFieldCheckpoints:
		m.configEdit.checkpoints += k
	}
	return m
}

func (m Model) configDelete() Model {
	switch m.configEdit.focus {
	case configFieldRounds:
		if m.configEdit.rounds != "" {
			m.configEdit.rounds = m.configEdit.rounds[:len(m.configEdit.rounds)-1]
		}
	case configFieldQuestions:
		if m.configEdit.questions != "" {
			m.configEdit.questions = m.configEdit.questions[:len(m.configEdit.questions)-1]
		}
	case configFieldCheckpoints:
		if m.configEdit.checkpoints != "" {
			m.configEdit.checkpoints = m.configEdit.checkpoints[:len(m.configEdit.checkpoints)-1]
		}
	}
	return m
}

func (m Model) submitConfig() (tea.Model, tea.Cmd) {
	rounds, err := strconv.Atoi(strings.TrimSpace(m.configEdit.rounds))
	if err != nil || rounds < 1 {
		m.errMsg = "rounds must be a positive integer"
		return m, nil
	}
	questions, err := strconv.Atoi(strings.TrimSpace(m.configEdit.questions))
	if err != nil || questions < 1 {
		m.errMsg = "questions must be a positive integer"
		return m, nil
	}
	cps, err := parseIntList(m.configEdit.checkpoints)
	if err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	newCfg := quiz.Config{
		Rounds:            rounds,
		QuestionsPerRound: questions,
		Checkpoints:       cps,
	}
	if err := newCfg.Validate(); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	m2, cmd := m.apply(quiz.ChangeSetConfig{Config: newCfg})
	if m2.errMsg != "" {
		return m2, cmd
	}
	m2.mode = ModeNormal
	m2.configEdit = configState{}
	return m2, cmd
}

// parseIntList parses "a,b,c" into a sorted unique slice, rejecting
// non-integers.
func parseIntList(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, errBadIntValue(p)
		}
		out = append(out, v)
	}
	return out, nil
}

// errBadIntValue returns an error describing a non-integer in a comma
// list.
func errBadIntValue(s string) error {
	return badIntError{value: s}
}

type badIntError struct {
	value string
}

func (e badIntError) Error() string { return "not an integer: " + e.value }

// joinInts renders []int as "a,b,c".
func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, v := range xs {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

// renderConfig draws the Config overlay.
func (m Model) renderConfig() string {
	title := styles.OverlayTitle.Render("Config")
	fields := []struct {
		label string
		value string
		field configField
	}{
		{"Rounds:             ", m.configEdit.rounds, configFieldRounds},
		{"Questions per round:", m.configEdit.questions, configFieldQuestions},
		{"Checkpoints:        ", m.configEdit.checkpoints, configFieldCheckpoints},
	}
	lines := []string{title, ""}
	for _, f := range fields {
		cursor := " "
		if m.configEdit.focus == f.field {
			cursor = styles.FuzzyArrow.Render(">")
		}
		lines = append(lines, cursor+" "+f.label+" ["+f.value+"]")
	}
	if m.errMsg != "" {
		lines = append(lines, "", styles.Error.Render("! err: "+m.errMsg))
	}
	lines = append(lines, "", "Tab/Shift+Tab field | Enter save | Esc cancel")
	return styles.OverlayBorder.Padding(1, 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
