package ui

import (
	"errors"
	"maps"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/quiz"
)

// Config-cell validation errors.
var (
	errBadRounds    = errors.New("rounds must be a positive integer")
	errBadQuestions = errors.New("questions must be a positive integer")
	errBadMaxPoints = errors.New("max points must be a positive integer")
)

// configCellKind classifies the addressable cells in the Config form.
type configCellKind int

const (
	cfgRounds configCellKind = iota
	cfgQuestions
	cfgCheckpoints
	cfgRoundMax // per-round max points; uses the Round field
)

// configCell identifies one navigable cell in the Config form. Round is
// only meaningful for cfgRoundMax cells.
type configCell struct {
	Kind  configCellKind
	Round int
}

// configState is the ephemeral state for ModeConfig. All values are read
// straight from m.quiz.Config; the only mutable state here is the focus and
// the in-place edit buffer, mirroring the score table's edit mode.
type configState struct {
	focus   configCell
	editing bool
	input   string
}

// startConfig opens the Config form with the cursor on Rounds.
func (m Model) startConfig() Model {
	m.mode = ModeConfig
	m.configEdit = configState{focus: configCell{Kind: cfgRounds}}
	m.errMsg = ""

	return m
}

// handleConfigKey dispatches keys while ModeConfig is active, splitting on
// whether a cell is currently being edited.
func (m Model) handleConfigKey(k, text string, km KeyMap) (tea.Model, tea.Cmd) {
	if m.configEdit.editing {
		return m.configEditKey(k, text, km)
	}

	return m.configNavKey(k, text, km)
}

// configNavKey handles navigation while no cell is being edited. Pressing
// Enter edits the focused cell starting from its current value; typing a
// legal character starts an edit that replaces the value (spreadsheet style).
func (m Model) configNavKey(k, text string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Escape, k):
		m.mode = ModeNormal
		m.configEdit = configState{}
		m.errMsg = ""

		return m, nil
	case matches(km.Enter, k):
		return m.beginConfigEdit(), nil
	case matches(km.Delete, k):
		return m.resetConfigCell()
	}

	if t := filterRunes(sanitizeText(text), m.configCellFilter()); t != "" {
		m.configEdit.editing = true
		m.configEdit.input = t
		m.errMsg = ""

		return m, nil
	}

	return m.moveConfigFocus(k, km), nil
}

// configEditKey handles keys while a cell's text buffer is active. Mirrors
// editCellKey (mode_edit.go): Esc cancels, Ctrl+U clears, Enter commits,
// everything else appends the legal subset of the typed text.
func (m Model) configEditKey(k, text string, km KeyMap) (tea.Model, tea.Cmd) {
	switch {
	case matches(km.Escape, k):
		m.configEdit.editing = false
		m.configEdit.input = ""
		m.errMsg = ""
	case matches(km.Clear, k):
		m.configEdit.input = ""
	case k == keyBackspace:
		if m.configEdit.input != "" {
			m.configEdit.input = m.configEdit.input[:len(m.configEdit.input)-1]
		}
	case matches(km.Enter, k):
		return m.commitConfigEdit()
	default:
		if text := sanitizeText(text); text != "" {
			m.configEdit.input += filterRunes(text, m.configCellFilter())
		}
	}

	return m, nil
}

// configCellFilter returns the legal-rune predicate for the focused cell:
// the checkpoints list also accepts comma and space, everything else is
// digits only.
func (m Model) configCellFilter() func(rune) bool {
	if m.configEdit.focus.Kind == cfgCheckpoints {
		return isCheckpointChar
	}

	return isDigit
}

// beginConfigEdit seeds the edit buffer with the focused cell's current
// value and switches to editing.
func (m Model) beginConfigEdit() Model {
	m.configEdit.editing = true
	m.configEdit.input = m.configCellValue(m.configEdit.focus)
	m.errMsg = ""

	return m
}

// commitConfigEdit parses the buffer, derives a new Config, and applies it.
// Validation failures keep the cell in edit state with an inline error.
func (m Model) commitConfigEdit() (tea.Model, tea.Cmd) {
	cfg, err := m.configWithEdit()
	if err != nil {
		m.errMsg = err.Error()

		return m, nil
	}

	m2, cmd := m.apply(quiz.ChangeSetConfig{Config: cfg})
	if m2.errMsg != "" {
		return m2, cmd
	}

	m2.configEdit.editing = false
	m2.configEdit.input = ""

	return m2.clampConfigFocus(), cmd
}

// resetConfigCell removes a per-round override (back to the questions-per-round
// default). Ignored on non-round cells.
func (m Model) resetConfigCell() (tea.Model, tea.Cmd) {
	if m.configEdit.focus.Kind != cfgRoundMax {
		return m, nil
	}

	cfg := m.configBase()
	delete(cfg.RoundMaxPoints, strconv.Itoa(m.configEdit.focus.Round))

	return m.apply(quiz.ChangeSetConfig{Config: cfg})
}

// configWithEdit returns a new Config reflecting the committed value of the
// focused cell.
func (m Model) configWithEdit() (quiz.Config, error) {
	cfg := m.configBase()

	switch m.configEdit.focus.Kind {
	case cfgRounds:
		v, ok := positiveInt(m.configEdit.input)
		if !ok {
			return cfg, errBadRounds
		}

		cfg.Rounds = v
		pruneToRounds(&cfg, v)
	case cfgQuestions:
		v, ok := positiveInt(m.configEdit.input)
		if !ok {
			return cfg, errBadQuestions
		}

		cfg.QuestionsPerRound = v
	case cfgCheckpoints:
		cps, err := parseIntList(m.configEdit.input)
		if err != nil {
			return cfg, err
		}

		cfg.Checkpoints = cps
	case cfgRoundMax:
		v, ok := positiveInt(m.configEdit.input)
		if !ok {
			return cfg, errBadMaxPoints
		}

		setRoundMax(&cfg, m.configEdit.focus.Round, v)
	}

	return cfg, nil
}

// configBase returns a deep copy of the current config with the legacy global
// MaxPoints dropped, so the per-round grid always baselines at questions per
// round. Rounds that need a different cap carry an explicit override.
func (m Model) configBase() quiz.Config {
	cfg := m.quiz.Config
	cfg.Checkpoints = append([]int(nil), m.quiz.Config.Checkpoints...)
	cfg.RoundMaxPoints = maps.Clone(m.quiz.Config.RoundMaxPoints)
	cfg.MaxPoints = 0

	return cfg
}

// setRoundMax records a per-round override, or drops it when the value equals
// the questions-per-round default (so only genuine overrides persist).
func setRoundMax(cfg *quiz.Config, round, v int) {
	key := strconv.Itoa(round)
	if v == cfg.QuestionsPerRound {
		delete(cfg.RoundMaxPoints, key)

		return
	}

	if cfg.RoundMaxPoints == nil {
		cfg.RoundMaxPoints = map[string]int{}
	}

	cfg.RoundMaxPoints[key] = v
}

// pruneToRounds drops overrides and checkpoints that fall outside a reduced
// round range so Config.Validate accepts the result.
func pruneToRounds(cfg *quiz.Config, rounds int) {
	for k := range cfg.RoundMaxPoints {
		if r, err := strconv.Atoi(k); err != nil || r > rounds {
			delete(cfg.RoundMaxPoints, k)
		}
	}

	kept := cfg.Checkpoints[:0]
	for _, cp := range cfg.Checkpoints {
		if cp <= rounds {
			kept = append(kept, cp)
		}
	}

	cfg.Checkpoints = kept
}

// positiveInt parses s as an integer >= 1, reporting ok=false otherwise.
func positiveInt(s string) (int, bool) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 1 {
		return 0, false
	}

	return v, true
}

// clampConfigFocus moves focus off a round cell that no longer exists after a
// rounds reduction.
func (m Model) clampConfigFocus() Model {
	if m.configEdit.focus.Kind == cfgRoundMax && m.configEdit.focus.Round > m.quiz.Config.Rounds {
		m.configEdit.focus = configCell{Kind: cfgRounds}
	}

	return m
}

// configRows lays the form out as a grid of navigable cells: one row each for
// Rounds, Questions and Checkpoints, then the per-round max-points boxes
// chunked to fit the viewport width.
func (m Model) configRows() [][]configCell {
	rows := [][]configCell{
		{{Kind: cfgRounds}},
		{{Kind: cfgQuestions}},
		{{Kind: cfgCheckpoints}},
	}

	perRow := m.roundCellsPerRow()

	var cur []configCell

	for r := 1; r <= m.quiz.Config.Rounds; r++ {
		cur = append(cur, configCell{Kind: cfgRoundMax, Round: r})
		if len(cur) == perRow {
			rows = append(rows, cur)
			cur = nil
		}
	}

	if len(cur) > 0 {
		rows = append(rows, cur)
	}

	return rows
}

// roundCellsPerRow returns how many per-round boxes fit on one line.
func (m Model) roundCellsPerRow() int {
	const cellW = 6 // value cell plus a separating space

	avail := m.width - 8

	n := avail / cellW
	if n < 1 {
		return 1
	}

	return n
}

// configFocusPos returns the row/column of the focused cell, defaulting to
// the first cell when the focus is not found.
func configFocusPos(rows [][]configCell, focus configCell) (int, int) {
	for r, row := range rows {
		for c, cell := range row {
			if cell == focus {
				return r, c
			}
		}
	}

	return 0, 0
}

// moveConfigFocus applies arrow / hjkl / g / G / 0 / $ navigation across the
// 2-D cell grid, clamping the column onto shorter rows.
func (m Model) moveConfigFocus(k string, km KeyMap) Model {
	rows := m.configRows()
	r, c := configFocusPos(rows, m.configEdit.focus)

	switch {
	case matches(km.Up, k):
		r = max(r-1, 0)
	case matches(km.Down, k):
		r = min(r+1, len(rows)-1)
	case matches(km.Left, k):
		c = max(c-1, 0)
	case matches(km.Right, k):
		c = min(c+1, len(rows[r])-1)
	case matches(km.Top, k):
		r = 0
	case matches(km.Bottom, k):
		r = len(rows) - 1
	case matches(km.First, k):
		c = 0
	case matches(km.Last, k):
		c = len(rows[r]) - 1
	}

	c = min(c, len(rows[r])-1)
	m.configEdit.focus = rows[r][c]

	return m
}

// configCellValue returns the display string for a cell, reading from the
// current config.
func (m Model) configCellValue(cell configCell) string {
	cfg := m.quiz.Config

	switch cell.Kind {
	case cfgRounds:
		return strconv.Itoa(cfg.Rounds)
	case cfgQuestions:
		return strconv.Itoa(cfg.QuestionsPerRound)
	case cfgCheckpoints:
		return joinInts(cfg.Checkpoints)
	case cfgRoundMax:
		if pts, ok := cfg.RoundMaxPoints[strconv.Itoa(cell.Round)]; ok {
			return strconv.Itoa(pts)
		}

		return strconv.Itoa(cfg.QuestionsPerRound)
	default:
		return ""
	}
}

// isRoundOverride reports whether a round has an explicit per-round override
// (and so should get the highlight in the grid).
func (m Model) isRoundOverride(round int) bool {
	_, ok := m.quiz.Config.RoundMaxPoints[strconv.Itoa(round)]

	return ok
}

// parseIntList parses "a,b,c" into a slice, rejecting non-integers.
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

// errBadIntValue returns an error describing a non-integer in a comma list.
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

// renderConfig draws the Config form: the three scalar cells, then the live
// per-round max-points grid.
func (m Model) renderConfig() string {
	lines := []string{styles.OverlayTitle.Render("Config"), ""}

	lines = append(
		lines,
		m.renderConfigScalar("Rounds", configCell{Kind: cfgRounds}),
		m.renderConfigScalar("Questions per round", configCell{Kind: cfgQuestions}),
		m.renderConfigScalar("Checkpoints", configCell{Kind: cfgCheckpoints}),
		"",
		styles.Dimmed.Render("  Max points per round  (default = questions per round)"),
	)

	rows := m.configRows()
	for _, row := range rows[3:] {
		labels := make([]string, 0, len(row))
		vals := make([]string, 0, len(row))

		for _, cell := range row {
			labels = append(labels, styles.Dimmed.Render(padCell("R"+strconv.Itoa(cell.Round), 5, alignRight)))
			vals = append(vals, m.renderConfigCell(cell, 5, alignRight))
		}

		lines = append(lines, "  "+strings.Join(labels, " "), "  "+strings.Join(vals, " "))
	}

	if m.errMsg != "" {
		lines = append(lines, "", styles.Error.Render("! err: "+m.errMsg))
	}

	hint := "  ↑↓←→ move · type or Enter to edit · x reset · Esc exit"
	if m.configEdit.editing {
		hint = "  type value · Enter commit · Esc cancel"
	}

	lines = append(lines, "", styles.Dimmed.Render(hint))

	return styles.OverlayBorder.Padding(1, 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// renderConfigScalar renders a labelled single-value row.
func (m Model) renderConfigScalar(label string, cell configCell) string {
	return "  " + padCell(label, 20, alignLeft) + " " + m.renderConfigCell(cell, 10, alignLeft)
}

// renderConfigCell renders one cell with the focus/editing/override styling.
func (m Model) renderConfigCell(cell configCell, width, align int) string {
	focused := m.configEdit.focus == cell

	if focused && m.configEdit.editing {
		return styles.CellEditing.Render(padCell(m.configEdit.input+"│", width, align))
	}

	padded := padCell(m.configCellValue(cell), width, align)

	switch {
	case focused:
		return styles.CellFocus.Render(padded)
	case cell.Kind == cfgRoundMax && m.isRoundOverride(cell.Round):
		return styles.Perfect.Render(padded)
	default:
		return padded
	}
}
