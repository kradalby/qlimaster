package ui

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/fuzzy"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/score"
)

// keyBackspace is the literal string reported by tea.KeyPressMsg for the
// backspace key. Pulled out as a constant so the handlers can reference it
// without repetition.
const keyBackspace = "backspace"

// enterStep is the sub-state within ModeEnterScore.
type enterStep int

const (
	// enterStepRound asks the host for the round number.
	enterStepRound enterStep = iota
	// enterStepPick lets the host fuzzy-search the team to score.
	enterStepPick
	// enterStepScore reads the score for the picked team.
	enterStepScore
)

// enterState holds the ephemeral UI state for the EnterScore flow.
type enterState struct {
	step      enterStep
	round     int
	query     string
	pickIndex int
	pickID    string
	input     string
	skipped   map[string]struct{} // team IDs skipped this round
}

// startEnterScore resets and opens the EnterScore flow.
func (m Model) startEnterScore() Model {
	m.mode = ModeEnterScore
	m.enter = enterState{
		step:    enterStepRound,
		skipped: map[string]struct{}{},
	}
	m.errMsg = ""
	return m
}

// handleEnterKey dispatches a key press while ModeEnterScore is active.
func (m Model) handleEnterKey(k, text string, km KeyMap) (tea.Model, tea.Cmd) {
	if matches(km.Escape, k) {
		return m.backOneStep(), nil
	}
	switch m.enter.step {
	case enterStepRound:
		return m.enterRoundKey(k, km), nil
	case enterStepPick:
		return m.enterPickKey(k, text, km)
	case enterStepScore:
		return m.enterScoreKey(k, text, km)
	}
	return m, nil
}

// backOneStep unwinds the current enter sub-state.
func (m Model) backOneStep() Model {
	switch m.enter.step {
	case enterStepRound:
		m.mode = ModeNormal
	case enterStepPick:
		m.enter.step = enterStepRound
		m.enter.query = ""
		m.enter.pickIndex = 0
	case enterStepScore:
		m.enter.step = enterStepPick
		m.enter.input = ""
		m.errMsg = ""
	}
	return m
}

func (m Model) enterRoundKey(k string, km KeyMap) Model {
	switch {
	case matches(km.Clear, k):
		m.enter.input = ""
		m.errMsg = ""
	case k == keyBackspace:
		if m.enter.input != "" {
			m.enter.input = m.enter.input[:len(m.enter.input)-1]
		}
	case matches(km.Enter, k):
		r, err := strconv.Atoi(strings.TrimSpace(m.enter.input))
		if err != nil || r < 1 || r > m.quiz.Config.Rounds {
			m.errMsg = "round must be 1.." + strconv.Itoa(m.quiz.Config.Rounds)
			return m
		}
		m.enter.round = r
		m.enter.step = enterStepPick
		m.enter.input = ""
		m.enter.query = ""
		m.enter.pickIndex = 0
		m.errMsg = ""
	default:
		if len(k) == 1 && k[0] >= '0' && k[0] <= '9' {
			m.enter.input += k
			m.errMsg = ""
		}
	}
	return m
}

func (m Model) enterPickKey(k, text string, km KeyMap) (tea.Model, tea.Cmd) {
	candidates := m.pickCandidates()
	switch {
	case isArrowUp(k) || k == "ctrl+p":
		if m.enter.pickIndex > 0 {
			m.enter.pickIndex--
		}
	case isArrowDown(k) || k == "ctrl+n":
		if m.enter.pickIndex < len(candidates)-1 {
			m.enter.pickIndex++
		}
	case matches(km.Tab, k):
		if len(candidates) > 0 {
			m.enter.skipped[candidates[m.enter.pickIndex].ID] = struct{}{}
			m.enter.pickIndex = 0
			m.enter.query = ""
		}
	case matches(km.Enter, k):
		if len(candidates) == 0 {
			m.errMsg = "no team to pick"
			return m, nil
		}
		m.enter.pickID = candidates[m.enter.pickIndex].ID
		m.enter.step = enterStepScore
		m.enter.input = ""
		m.errMsg = ""
	case k == keyBackspace:
		if m.enter.query != "" {
			m.enter.query = m.enter.query[:len(m.enter.query)-1]
			m.enter.pickIndex = 0
		}
	default:
		if text := sanitizeText(text); text != "" {
			m.enter.query += text
			m.enter.pickIndex = 0
		}
	}
	return m, nil
}

func (m Model) enterScoreKey(k, text string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Clear, k):
		m.enter.input = ""
		m.errMsg = ""
	case matches(km.Back, k):
		// Ctrl+R: change round without leaving the flow.
		m.enter.step = enterStepRound
		m.enter.input = strconv.Itoa(m.enter.round)
		return m, nil
	case k == keyBackspace:
		if m.enter.input != "" {
			m.enter.input = m.enter.input[:len(m.enter.input)-1]
		}
	case matches(km.Enter, k):
		v, err := score.Parse(m.enter.input, float64(m.quiz.Config.QuestionsPerRound))
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		teamID := m.enter.pickID
		round := m.enter.round
		m, cmd := m.apply(quiz.ChangeSetScore{TeamID: teamID, Round: round, Score: v})
		if m.errMsg != "" {
			return m, cmd
		}
		// After saving, advance to the next team without a score for this
		// round. If none remain, drop back to Normal.
		if len(m.remainingForRound(round)) == 0 {
			m.mode = ModeNormal
			m.enter = enterState{}
			return m, cmd
		}
		m.enter.step = enterStepPick
		m.enter.input = ""
		m.enter.query = ""
		m.enter.pickIndex = 0
		return m, cmd
	default:
		if isScoreChar(text) {
			m.enter.input += text
			m.errMsg = ""
		}
	}
	return m, nil
}

// pickCandidates returns the teams eligible for selection at the current
// pick step: fuzzy-filtered by query, excluding teams already scored and
// those explicitly skipped this round.
func (m Model) pickCandidates() []quiz.Team {
	pool := m.remainingForRound(m.enter.round)
	if m.enter.query == "" {
		return pool
	}
	names := make([]string, len(pool))
	for i, t := range pool {
		names[i] = t.Name
	}
	matches := fuzzy.Do(m.enter.query, names)
	out := make([]quiz.Team, 0, len(matches))
	for _, mt := range matches {
		for _, t := range pool {
			if t.Name == mt.Item {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

// remainingForRound returns teams without a recorded score for the given
// round and not currently skipped in the enter state.
func (m Model) remainingForRound(round int) []quiz.Team {
	out := make([]quiz.Team, 0, len(m.quiz.Teams))
	for _, t := range m.quiz.Teams {
		if _, done := t.Score(round); done {
			continue
		}
		if _, skipped := m.enter.skipped[t.ID]; skipped {
			continue
		}
		out = append(out, t)
	}
	return out
}

// scoredForRound returns teams that already have a score for the given
// round, paired with the recorded score for display under the picker.
func (m Model) scoredForRound(round int) []scoredEntry {
	out := make([]scoredEntry, 0, len(m.quiz.Teams))
	for _, t := range m.quiz.Teams {
		if v, ok := t.Score(round); ok {
			out = append(out, scoredEntry{Name: t.Name, Score: v})
		}
	}
	return out
}

type scoredEntry struct {
	Name  string
	Score float64
}

// isScoreChar reports whether k is a legal character in a score input.
// Digits, '.', ',' and space are accepted; everything else is ignored.
func isScoreChar(k string) bool {
	if len(k) != 1 {
		return false
	}
	c := k[0]
	return c >= '0' && c <= '9' || c == '.' || c == ',' || c == ' '
}

// renderEnter draws the EnterScore overlay for the current sub-step.
func (m Model) renderEnter() string {
	switch m.enter.step {
	case enterStepRound:
		return m.renderEnterRound()
	case enterStepPick:
		return m.renderEnterPick()
	case enterStepScore:
		return m.renderEnterScore()
	}
	return ""
}

func (m Model) renderEnterRound() string {
	title := styles.OverlayTitle.Render("Enter scores")
	input := "Round number: " + m.enter.input + "_"
	errLine := ""
	if m.errMsg != "" {
		errLine = "\n" + styles.Error.Render("! err: "+m.errMsg)
	}
	body := lipgloss.JoinVertical(lipgloss.Left, title, "", input+errLine, "", "Enter next | Esc cancel")
	return styles.OverlayBorder.Padding(1, 2).Render(body)
}

func (m Model) renderEnterPick() string {
	cands := m.pickCandidates()
	scored := m.scoredForRound(m.enter.round)

	title := styles.OverlayTitle.Render(
		"Round " + strconv.Itoa(m.enter.round) + " - pick team (" +
			strconv.Itoa(len(cands)) + " remaining)",
	)
	query := "> " + m.enter.query + "_"

	candLines := make([]string, 0, len(cands))
	for i, t := range cands {
		prefix := "  "
		if i == m.enter.pickIndex {
			prefix = styles.FuzzyArrow.Render("> ")
		}
		candLines = append(candLines, prefix+highlightName(t.Name, m.enter.query))
	}
	if len(candLines) == 0 {
		candLines = append(candLines, styles.Dimmed.Render("  (no matches)"))
	}

	scoredBlock := []string{}
	if len(scored) > 0 {
		scoredBlock = append(scoredBlock, "", styles.Dimmed.Render("-- already scored this round --"))
		for _, s := range scored {
			scoredBlock = append(scoredBlock, styles.Dimmed.Render(
				"  "+s.Name+"  "+score.Format(s.Score),
			))
		}
	}

	lines := make([]string, 0, 4+len(candLines)+len(scoredBlock)+2)
	lines = append(lines, title, "", query, strings.Repeat("-", 40))
	lines = append(lines, candLines...)
	lines = append(lines, scoredBlock...)
	lines = append(lines, "", "Enter select | Tab skip | Esc back")
	return styles.OverlayBorder.Padding(1, 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderEnterScore() string {
	var teamName string
	for _, t := range m.quiz.Teams {
		if t.ID == m.enter.pickID {
			teamName = t.Name
			break
		}
	}
	title := styles.OverlayTitle.Render("Round " + strconv.Itoa(m.enter.round) + " | " + teamName)
	input := "Score: " + m.enter.input + "_"
	errLine := ""
	if m.errMsg != "" {
		errLine = "\n" + styles.Error.Render("! err: "+m.errMsg)
	}
	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		input+errLine,
		"",
		styles.Dimmed.Render("digit = whole, . or , = +0.5"),
		"Enter save & next | Ctrl+U clear | Ctrl+R change round | Esc back",
	)
	return styles.OverlayBorder.Padding(1, 2).Render(body)
}

// highlightName renders name with matched query characters highlighted
// (fuzzy positions). Case-insensitive match.
func highlightName(name, query string) string {
	if query == "" {
		return name
	}
	positions := fuzzy.Do(query, []string{name})
	if len(positions) == 0 || len(positions[0].Positions) == 0 {
		return name
	}
	var sb strings.Builder
	seen := map[int]struct{}{}
	for _, p := range positions[0].Positions {
		seen[p] = struct{}{}
	}
	for i, r := range name {
		if _, ok := seen[i]; ok {
			sb.WriteString(styles.FuzzyMatch.Render(string(r)))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

var _ = slices.Contains[[]int] // keep slices imported for future use
