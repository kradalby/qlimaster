package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/fuzzy"
	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
)

// newTeamStep tracks the sub-state of the add-team overlay.
type newTeamStep int

const (
	// newTeamStepName reads the team name (with fuzzy suggestions from
	// the history file).
	newTeamStepName newTeamStep = iota
	// newTeamStepPlayers reads the optional free-text players string.
	newTeamStepPlayers
)

// newTeamState holds ephemeral state for the add-team flow.
type newTeamState struct {
	step       newTeamStep
	name       string
	players    string
	suggestIdx int
}

// startNewTeam opens the add-team overlay.
func (m Model) startNewTeam() Model {
	m.mode = ModeNewTeam
	m.newTeam = newTeamState{}
	m.errMsg = ""
	return m
}

// handleNewTeamKey dispatches keys while ModeNewTeam is active.
func (m Model) handleNewTeamKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	if matches(km.Escape, k) {
		m.mode = ModeNormal
		m.newTeam = newTeamState{}
		return m, nil
	}
	switch m.newTeam.step {
	case newTeamStepName:
		return m.newTeamNameKey(k, km)
	case newTeamStepPlayers:
		return m.newTeamPlayersKey(k, km)
	}
	return m, nil
}

func (m Model) newTeamNameKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	suggestions := m.newTeamSuggestions()
	switch {
	case matches(km.Up, k):
		if m.newTeam.suggestIdx > 0 {
			m.newTeam.suggestIdx--
		}
		return m, nil
	case matches(km.Down, k):
		if m.newTeam.suggestIdx < len(suggestions)-1 {
			m.newTeam.suggestIdx++
		}
		return m, nil
	case matches(km.Enter, k):
		return m.newTeamAcceptName(suggestions), nil
	case k == keyBackspace:
		if m.newTeam.name != "" {
			m.newTeam.name = m.newTeam.name[:len(m.newTeam.name)-1]
			m.newTeam.suggestIdx = 0
		}
		return m, nil
	}
	if len(k) == 1 && k[0] >= ' ' {
		m.newTeam.name += k
		m.newTeam.suggestIdx = 0
	}
	return m, nil
}

// newTeamAcceptName confirms the name field and advances to the players
// step. When the typed query is empty, the highlighted suggestion wins;
// otherwise the typed text is preferred, promoted to the casing of any
// suggestion that matches case-insensitively.
func (m Model) newTeamAcceptName(suggestions []history.Entry) Model {
	name := strings.TrimSpace(m.newTeam.name)
	if name == "" && len(suggestions) > 0 {
		name = suggestions[m.newTeam.suggestIdx].Name
	}
	if name == "" {
		m.errMsg = "team name required"
		return m
	}
	if idx := m.newTeam.suggestIdx; idx >= 0 && idx < len(suggestions) &&
		strings.EqualFold(suggestions[idx].Name, name) {
		name = suggestions[idx].Name
	}
	m.newTeam.name = name
	m.newTeam.step = newTeamStepPlayers
	m.errMsg = ""
	return m
}

func (m Model) newTeamPlayersKey(k string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Enter, k):
		// Commit via apply.
		cmd := m.newTeam
		m, tcmd := m.apply(teamAddChange(cmd))
		if m.errMsg != "" {
			return m, tcmd
		}
		m.mode = ModeNormal
		m.newTeam = newTeamState{}
		return m, tcmd
	case k == keyBackspace:
		if m.newTeam.players != "" {
			m.newTeam.players = m.newTeam.players[:len(m.newTeam.players)-1]
		}
	default:
		if len(k) == 1 && k[0] >= ' ' {
			m.newTeam.players += k
		}
	}
	return m, nil
}

// newTeamSuggestions returns history entries filtered by the current
// typed name, in fuzzy-ranked order.
func (m Model) newTeamSuggestions() []history.Entry {
	entries := m.history.Teams
	if m.newTeam.name == "" {
		// Empty query: return in the history's natural sort (most recent
		// first) but cap at a reasonable display size.
		limit := min(len(entries), 8)
		return append([]history.Entry(nil), entries[:limit]...)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	matches := fuzzy.Do(m.newTeam.name, names)
	out := make([]history.Entry, 0, min(len(matches), 8))
	for i, mt := range matches {
		if i >= 8 {
			break
		}
		for _, e := range entries {
			if e.Name == mt.Item {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// teamAddChange constructs the quiz.Change for the accumulated add-team
// state. Returned by value so callers can pass it through Model.apply.
func teamAddChange(s newTeamState) quiz.Change {
	return quiz.ChangeAddTeam{
		Name:    strings.TrimSpace(s.name),
		Players: strings.TrimSpace(s.players),
	}
}

// renderNewTeam draws the add-team overlay.
func (m Model) renderNewTeam() string {
	if m.newTeam.step == newTeamStepPlayers {
		return m.renderNewTeamPlayers()
	}
	return m.renderNewTeamName()
}

func (m Model) renderNewTeamName() string {
	title := styles.OverlayTitle.Render("Add team")
	input := "Name: " + m.newTeam.name + "_"
	suggestions := m.newTeamSuggestions()
	lines := []string{title, "", input, strings.Repeat("-", 48)}
	if len(suggestions) == 0 {
		lines = append(lines, styles.Dimmed.Render("  (no suggestions)"))
	}
	for i, s := range suggestions {
		prefix := "  "
		if i == m.newTeam.suggestIdx {
			prefix = styles.FuzzyArrow.Render("> ")
		}
		suffix := styles.Dimmed.Render("  last: " + s.LastSeen + "  (" + strconv.Itoa(s.TimesSeen) + "x)")
		lines = append(lines, prefix+highlightName(s.Name, m.newTeam.name)+suffix)
	}
	if m.errMsg != "" {
		lines = append(lines, "", styles.Error.Render("! err: "+m.errMsg))
	}
	lines = append(lines, "", "Enter accept | Esc cancel")
	return styles.OverlayBorder.Padding(1, 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) renderNewTeamPlayers() string {
	title := styles.OverlayTitle.Render("Add team: " + m.newTeam.name)
	input := "Players (optional): " + m.newTeam.players + "_"
	lines := []string{title, "", input, "", "Enter confirm | Esc cancel"}
	return styles.OverlayBorder.Padding(1, 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
