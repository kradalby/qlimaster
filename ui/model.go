package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
	"tailscale.com/util/set"
)

// Mode is the UI's current top-level state. Each value dispatches key
// messages to its own update function.
type Mode int

const (
	// ModeNormal is the default spreadsheet view.
	ModeNormal Mode = iota
	// ModeEnterScore is the fast path for entering a round's scores.
	ModeEnterScore
	// ModeEditScore allows arrow navigation and single-cell editing.
	ModeEditScore
	// ModeNewTeam is the add-team flow with fuzzy suggestions from history.
	ModeNewTeam
	// ModeConfig is the edit-rounds/questions/checkpoints form.
	ModeConfig
	// ModeExport shows the export-menu overlay.
	ModeExport
	// ModeHelp overlays the keymap documentation.
	ModeHelp
	// ModeReadOut is the presentation mode: full-screen card-per-team
	// walk-through from worst to best, for reading scores aloud at the
	// end of a quiz.
	ModeReadOut
)

// Label returns the bracketed badge string shown in the bottom bar.
func (m Mode) Label() string {
	switch m {
	case ModeNormal:
		return "[NORMAL]"
	case ModeEnterScore:
		return "[ENTER-SCORE]"
	case ModeEditScore:
		return "[EDIT]"
	case ModeNewTeam:
		return "[NEW-TEAM]"
	case ModeConfig:
		return "[CONFIG]"
	case ModeExport:
		return "[EXPORT]"
	case ModeHelp:
		return "[HELP]"
	case ModeReadOut:
		return "[READ-OUT]"
	default:
		return "[?]"
	}
}

// Model is the Bubble Tea model for the full qlimaster TUI.
//
// The zero value is not usable; construct via [New].
type Model struct {
	// width and height come from tea.WindowSizeMsg.
	width, height int

	quiz    quiz.Quiz
	path    string // absolute path to quiz.hujson
	history history.History
	// historyPath is the resolved absolute path to history.hujson; used
	// for live history updates on team-name mutations.
	historyPath string
	// sessionRecordedNames tracks which team names (lower-cased) have
	// already been persisted to the history file during this session.
	// Ensures each team bumps TimesSeen at most once per run, even if
	// the name is later edited.
	sessionRecordedNames set.Set[string]

	mode Mode

	// rowCursor is the currently selected row in Normal and Edit modes.
	rowCursor int

	// enter is the ephemeral state for the EnterScore flow.
	enter enterState

	// focusedCell is the cell under the cursor in Edit mode. NoCell in
	// other modes.
	focusedCell Cell
	// edit is the ephemeral state for the EditScore flow.
	edit editState

	// newTeam is the ephemeral state for the NewTeam flow.
	newTeam newTeamState

	// configEdit is the ephemeral state for the Config mode.
	configEdit configState

	// readOut is the ephemeral state for the ReadOut mode.
	readOut readOutState

	// Status/toast line shown in the footer. Cleared by a timer.
	status       string
	statusExpiry time.Time

	// errMsg is displayed inline for validation failures.
	errMsg string

	// lastEntered is the highest round number for which at least one team
	// has a recorded score. Used by the responsive layout.
	lastEntered int
}

// Config holds the parameters New accepts. Any zero-valued field falls back
// to a sensible default so callers can pass just the fields they care
// about.
type Config struct {
	// Path is the location of quiz.hujson. Required.
	Path string
	// HistoryPath is the location of the team history file. Defaults to
	// the XDG path when empty.
	HistoryPath string
	// QuizRoot is the folder scanned for sibling quizzes to build the
	// fuzzy-name history. Defaults to filepath.Dir(Config.Path).
	QuizRoot string
	// QuizConfig is the quiz structure to create when Path does not yet
	// exist. Ignored when Path exists and is parseable.
	QuizConfig quiz.Config
}

// New constructs a Model by loading or creating the quiz file at cfg.Path
// and merging the history file with a live scan of cfg.QuizRoot.
func New(cfg Config) (Model, error) {
	if cfg.Path == "" {
		return Model{}, errors.New("ui: Config.Path is required")
	}
	if cfg.QuizConfig.Rounds == 0 {
		cfg.QuizConfig = quiz.DefaultConfig()
	}
	if cfg.QuizRoot == "" {
		cfg.QuizRoot = filepath.Dir(cfg.Path)
	}

	q, err := loadOrCreate(cfg.Path, cfg.QuizConfig)
	if err != nil {
		return Model{}, err
	}

	historyPath := cfg.HistoryPath
	if historyPath == "" {
		historyPath = history.DefaultPath(cfg.QuizRoot)
	}
	hist, err := loadHistory(historyPath, cfg.QuizRoot)
	if err != nil {
		// History is best-effort; we still start the UI.
		hist = history.History{Version: 1}
	}

	// Seed the session-recorded set with every team already present
	// in the loaded quiz, so reopening a quiz file does not re-bump
	// TimesSeen on existing names.
	recorded := set.Set[string]{}
	for _, t := range q.Teams {
		if key := strings.ToLower(strings.TrimSpace(t.Name)); key != "" {
			recorded.Add(key)
		}
	}

	return Model{
		quiz:                 q,
		path:                 cfg.Path,
		history:              hist,
		historyPath:          historyPath,
		sessionRecordedNames: recorded,
		mode:                 ModeNormal,
		lastEntered:          computeLastEntered(q),
	}, nil
}

// loadOrCreate reads quiz.hujson at path or, when it does not exist yet,
// creates and persists a fresh quiz with the supplied config.
func loadOrCreate(path string, cfg quiz.Config) (quiz.Quiz, error) {
	q, err := store.Load(path)
	if err == nil {
		return q, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return quiz.Quiz{}, fmt.Errorf("load quiz: %w", err)
	}
	fresh := quiz.New(cfg)
	if err := store.Save(path, fresh); err != nil {
		return quiz.Quiz{}, fmt.Errorf("save initial quiz: %w", err)
	}
	return fresh, nil
}

// loadHistory combines the persisted history file with a live scan of
// sibling quiz folders under quizRoot.
func loadHistory(historyPath, quizRoot string) (history.History, error) {
	if historyPath == "" {
		historyPath = history.DefaultPath(quizRoot)
	}
	persisted, err := history.Load(historyPath)
	if err != nil {
		return history.History{}, fmt.Errorf("load history: %w", err)
	}
	scanned, err := history.Scan(quizRoot)
	if err != nil {
		return persisted, nil //nolint:nilerr // scan failure is non-fatal
	}
	return history.Merge(persisted, scanned), nil
}

// computeLastEntered returns the highest round number for which any team
// has a recorded score.
func computeLastEntered(q quiz.Quiz) int {
	last := 0
	for _, t := range q.Teams {
		for r := 1; r <= q.Config.Rounds; r++ {
			if _, ok := t.Score(r); ok && r > last {
				last = r
			}
		}
	}
	return last
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case savedMsg:
		return m.onSaved(msg), nil
	case historySavedMsg:
		// History saves are best-effort; only a failure is worth
		// surfacing, and even then the quiz.hujson save is already
		// persisted so we do not need to block the UI.
		if msg.Err != nil {
			m.status = "history save failed: " + msg.Err.Error()
			m.statusExpiry = msg.When.Add(1500 * time.Millisecond)
			return m, clearStatusCmd(1500 * time.Millisecond)
		}
		return m, nil
	case clearStatusMsg:
		if !time.Now().Before(m.statusExpiry) {
			m.status = ""
		}
		return m, nil
	case perfectFlashTickMsg:
		m2, cmd := m.onPerfectTick(msg)
		return m2, cmd
	case winnerGlowTickMsg:
		m2, cmd := m.onWinnerTick(msg)
		return m2, cmd
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render composes the full-screen view for the current viewport size and
// mode. Overlays are stacked on top of the normal view when active.
func (m Model) render() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	base := m.renderBase()
	// Overlays are drawn by mode-specific renderers; Normal is the default.
	switch m.mode {
	case ModeHelp:
		return overlayOnto(base, m.renderHelp(), m.width, m.height)
	case ModeExport:
		return overlayOnto(base, m.renderExport(), m.width, m.height)
	case ModeEnterScore:
		return overlayOnto(base, m.renderEnter(), m.width, m.height)
	case ModeNewTeam:
		return overlayOnto(base, m.renderNewTeam(), m.width, m.height)
	case ModeConfig:
		return overlayOnto(base, m.renderConfig(), m.width, m.height)
	case ModeReadOut:
		return m.renderReadOut()
	default:
		return base
	}
}

// renderBase draws the header + table + footer frame that is shared
// across all modes.
func (m Model) renderBase() string {
	layout := Compute(m.width, m.height, m.quiz.Config, m.lastEntered)
	header := renderHeader(m.width, m.quiz.Config, len(m.quiz.Teams), dateLabel(m.quiz))
	table := m.renderTable(layout)
	footer := renderFooter(m.width, m.mode, m.footerStatus(), m.hints())
	return joinVerticalLines(m.width, m.height, header, table, footer)
}

// footerStatus returns the text shown next to the mode badge in the
// footer. It prefers mode-specific state (the edit cursor location, for
// example), falling back to the generic status toast.
func (m Model) footerStatus() string {
	if m.errMsg != "" {
		return styles.Error.Render("! err: " + m.errMsg)
	}
	if m.mode == ModeEditScore {
		return m.renderEditStatus()
	}
	return m.status
}

// hints returns the contextual keybind helper shown in the footer for the
// current mode.
func (m Model) hints() []footerHint {
	switch m.mode {
	case ModeNormal:
		return []footerHint{
			{"e", "enter"},
			{"i", "edit"},
			{"a", "add"},
			{"R", "read-out"},
			{"E", "export"},
			{":", "config"},
			{"?", "help"},
			{"q", "quit"},
		}
	case ModeReadOut:
		return []footerHint{{"Space/↓", "next"}, {"↑", "prev"}, {"Esc", "exit"}}
	case ModeExport:
		return []footerHint{{"c", "CSV"}, {"x", "XLSX"}, {"b", "both"}, {"Esc", "cancel"}}
	case ModeHelp:
		return []footerHint{{"?", "dismiss"}}
	case ModeEnterScore:
		return []footerHint{{"Enter", "next"}, {"Tab", "skip"}, {"Esc", "back"}}
	case ModeEditScore:
		if m.edit.editing {
			return []footerHint{{"Enter", "save"}, {"Esc", "cancel"}, {"Ctrl+U", "clear"}}
		}
		return []footerHint{
			{"hjkl", "move"},
			{"Enter", "edit"},
			{"x", "clear"},
			{"dd", "delete"},
			{"Esc", "normal"},
		}
	case ModeNewTeam:
		return []footerHint{{"Enter", "accept"}, {"Up/Down", "select"}, {"Esc", "cancel"}}
	case ModeConfig:
		return []footerHint{{"Tab", "next"}, {"Enter", "save"}, {"Esc", "cancel"}}
	default:
		return []footerHint{{"Esc", "back"}}
	}
}

// dateLabel derives a short date string for the header's right region.
func dateLabel(q quiz.Quiz) string {
	if !q.Created.IsZero() {
		return q.Created.Format("Mon 2006-01-02")
	}
	return time.Now().Format("Mon 2006-01-02")
}

// joinVerticalLines glues header, table, footer together so the total
// occupies exactly height lines and width columns.
func joinVerticalLines(width, height int, header, table, footer string) string {
	// The table is produced already padded to the computed TableHeight+chrome
	// lines. We concat with newlines, then pad or truncate to height.
	result := header + "\n" + table + "\n" + footer
	return clampLines(result, width, height)
}
