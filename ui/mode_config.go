package ui

import (
	"slices"
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
	configFieldPoints
	configFieldRoundPoints
	configFieldCheckpoints
	configFieldCount
)

// configState is the ephemeral state for ModeConfig.
type configState struct {
	focus       configField
	rounds      string
	questions   string
	points      string
	checkpoints string
	roundPoints string
}

// startConfig seeds the overlay with the current config values and opens
// the mode.
func (m Model) startConfig() Model {
	m.mode = ModeConfig
	m.configEdit = configState{
		focus:       configFieldRounds,
		rounds:      strconv.Itoa(m.quiz.Config.Rounds),
		questions:   strconv.Itoa(m.quiz.Config.QuestionsPerRound),
		points:      strconv.Itoa(int(m.quiz.Config.MaxScore())),
		roundPoints: formatRoundMaxPoints(m.quiz.Config.RoundMaxPoints),
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

	if text := sanitizeText(text); text != "" {
		return m.configAppend(text), nil
	}

	return m, nil
}

// configAppend extends the focused field with the legal subset of k.
// Rounds and Questions accept digits only; Checkpoints also accepts
// comma and whitespace. Anything else (letters, punctuation, and -
// critically - control characters from leaked terminal escape
// sequences like cursor position reports) is silently dropped so the
// user never sees bytes like "[75;1R" land in the form.
func (m Model) configAppend(k string) Model {
	k = stripTerminalNoise(k)
	if k == "" {
		return m
	}

	switch m.configEdit.focus {
	case configFieldRounds:
		m.configEdit.rounds += filterRunes(k, isDigit)
	case configFieldQuestions:
		m.configEdit.questions += filterRunes(k, isDigit)
	case configFieldPoints:
		m.configEdit.points += filterRunes(k, isDigit)
	case configFieldCheckpoints:
		m.configEdit.checkpoints += filterRunes(k, isCheckpointChar)
	case configFieldRoundPoints:
		m.configEdit.roundPoints += filterRunes(k, isRoundMaxPointChar)
	default:
		// configFieldCount is a sentinel, never a real focus value.
	}

	return m
}

func stripTerminalNoise(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); i++ {
		if s[i] == '[' {
			j := i + 1
			for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == ';') {
				j++
			}
			if j < len(s) && ((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				i = j
				continue
			}
		}

		if s[i] >= 32 && s[i] != 127 {
			b.WriteByte(s[i])
		}
	}

	return b.String()
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
	case configFieldPoints:
		if m.configEdit.points != "" {
			m.configEdit.points = m.configEdit.points[:len(m.configEdit.points)-1]
		}
	case configFieldCheckpoints:
		if m.configEdit.checkpoints != "" {
			m.configEdit.checkpoints = m.configEdit.checkpoints[:len(m.configEdit.checkpoints)-1]
		}
	case configFieldRoundPoints:
		if m.configEdit.roundPoints != "" {
			m.configEdit.roundPoints = m.configEdit.roundPoints[:len(m.configEdit.roundPoints)-1]
		}
	default:
		// configFieldCount is a sentinel, never a real focus value.
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

	points, err := strconv.Atoi(strings.TrimSpace(m.configEdit.points))
	if err != nil || points < 1 {
		m.errMsg = "max points must be a positive integer"

		return m, nil
	}

	cps, err := parseIntList(m.configEdit.checkpoints)
	if err != nil {
		m.errMsg = err.Error()

		return m, nil
	}

	roundMaxPoints, err := parseRoundMaxPoints(m.configEdit.roundPoints)
	if err != nil {
		m.errMsg = err.Error()
		return m, nil
	}

	newCfg := quiz.Config{
		Rounds:            rounds,
		QuestionsPerRound: questions,
		MaxPoints:         points,
		RoundMaxPoints:    roundMaxPoints,
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

func isRoundMaxPointChar(r rune) bool {
	return isDigit(r) || r == ':' || r == ',' || r == ' '
}

func parseRoundMaxPoints(s string) (map[string]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	out := map[string]int{}
	parts := strings.Split(s, ",")

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		pair := strings.Split(p, ":")
		if len(pair) != 2 {
			return nil, badIntError{value: p}
		}

		round, err := strconv.Atoi(strings.TrimSpace(pair[0]))
		if err != nil {
			return nil, badIntError{value: pair[0]}
		}

		max, err := strconv.Atoi(strings.TrimSpace(pair[1]))
		if err != nil {
			return nil, badIntError{value: pair[1]}
		}

		out[strconv.Itoa(round)] = max
	}

	return out, nil
}

func formatRoundMaxPoints(xs map[string]int) string {
	if len(xs) == 0 {
		return ""
	}

	keys := make([]int, 0, len(xs))
	for k := range xs {
		v, err := strconv.Atoi(k)
		if err == nil {
			keys = append(keys, v)
		}
	}

	slices.Sort(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts,
			strconv.Itoa(k)+":"+strconv.Itoa(xs[strconv.Itoa(k)]))
	}

	return strings.Join(parts, ",")
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
		{"Max points/round:   ", m.configEdit.points, configFieldPoints},
		{"Round max points:  ", m.configEdit.roundPoints, configFieldRoundPoints},
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
