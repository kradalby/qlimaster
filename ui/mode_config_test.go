package ui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// config-form test key events.
var (
	kEnter = tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"}
	kEsc   = tea.KeyPressMsg{Code: tea.KeyEscape}
	kBack  = tea.KeyPressMsg{Code: tea.KeyBackspace}
	kDown  = tea.KeyPressMsg{Code: tea.KeyDown}
	kRight = tea.KeyPressMsg{Code: tea.KeyRight}
)

// openConfig builds a Model from cfg and opens the config form.
func openConfig(t *testing.T, cfg quiz.Config) tea.Model {
	t.Helper()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: cfg,
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	mm, _ := model.(Model)
	require.Equal(t, ModeConfig, mm.mode)

	return model
}

// clearCell sends enough backspaces to empty the active edit buffer.
func clearCell(model tea.Model) tea.Model {
	for range 6 {
		model, _ = model.Update(kBack)
	}

	return model
}

// typeStr sends each rune of s as a key press.
func typeStr(model tea.Model, s string) tea.Model {
	for _, r := range s {
		model, _ = model.Update(teaKey(string(r)))
	}

	return model
}

// TestConfig_UpdatesQuiz drives the spreadsheet flow: edit Rounds, then
// Checkpoints, each committed in place.
func TestConfig_UpdatesQuiz(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.DefaultConfig())

	// Edit Rounds (focused on open) 8 -> 6.
	model, _ = model.Update(kEnter)
	model = clearCell(model)
	model = typeStr(model, "6")
	model, _ = model.Update(kEnter)

	mm, _ := model.(Model)
	assert.Equal(t, ModeConfig, mm.mode, "commit keeps the form open")
	assert.Equal(t, 6, mm.quiz.Config.Rounds)

	// Down twice to Checkpoints, replace with "3,6".
	model, _ = model.Update(kDown)
	model, _ = model.Update(kDown)
	model, _ = model.Update(kEnter)
	model = clearCell(model)
	model = typeStr(model, "3,6")
	model, _ = model.Update(kEnter)

	mm, _ = model.(Model)
	assert.Equal(t, []int{3, 6}, mm.quiz.Config.Checkpoints)

	// Esc exits.
	model, _ = model.Update(kEsc)
	mm, _ = model.(Model)
	assert.Equal(t, ModeNormal, mm.mode)
}

// TestConfig_RejectsInvalid shows an inline error and keeps the cell open
// when the committed value is unparseable.
func TestConfig_RejectsInvalid(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.DefaultConfig())

	model, _ = model.Update(kEnter)
	model = clearCell(model) // empty Rounds buffer
	model, _ = model.Update(kEnter)

	mm, _ := model.(Model)
	assert.Equal(t, ModeConfig, mm.mode)
	assert.True(t, mm.configEdit.editing, "cell stays in edit on error")
	assert.NotEmpty(t, mm.errMsg)
}

// TestConfig_IgnoresEscapeSequenceNoise is a regression test: stray escape
// sequences arriving in tea.KeyPressMsg.Text must never reach a numeric edit
// buffer, not even the digits inside them.
func TestConfig_IgnoresEscapeSequenceNoise(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.DefaultConfig())

	// Begin editing Rounds; the buffer starts at the current value "8".
	model, _ = model.Update(kEnter)

	// A cursor-position-report blob: the whole thing is control-bearing
	// noise and must be rejected wholesale.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyExtended, Text: "\x1b[75;1R"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyExtended, Text: "\x7f[1;2R"})
	// Plain garbage with no control chars reaches the buffer; the digit
	// filter keeps "3" and drops the letters.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyExtended, Text: "abc3def"})

	mm, _ := model.(Model)
	assert.Equal(t, "83", mm.configEdit.input)
	assert.NotContains(t, mm.configEdit.input, "7")
	assert.NotContains(t, mm.configEdit.input, "[")
	assert.NotContains(t, mm.configEdit.input, "R")
}

// TestConfig_IgnoresSplitEscapeSequence reproduces the split-report variant:
// the CSI prefix arrives as a uv.UnknownEvent and the tail as bare key
// presses, which must be swallowed rather than landing in the buffer.
func TestConfig_IgnoresSplitEscapeSequence(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.DefaultConfig())

	model, _ = model.Update(kEnter) // edit Rounds, buffer "8"

	model, _ = model.Update(uv.UnknownEvent("\x1b[20"))
	for _, r := range []rune{';', '2', '6', 'R'} {
		model, _ = model.Update(teaKey(string(r)))
	}

	mm, _ := model.(Model)
	assert.Equal(t, "8", mm.configEdit.input, "no leaked digits")

	// After the terminator real typing resumes.
	model, _ = model.Update(teaKey("5"))
	mm, _ = model.(Model)
	assert.Equal(t, "85", mm.configEdit.input)
}

// TestConfig_CheckpointsAllowsCommaAndSpace confirms the Checkpoints cell
// accepts its separator characters while filtering letters.
func TestConfig_CheckpointsAllowsCommaAndSpace(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.DefaultConfig())

	// Down twice (Rounds -> Questions -> Checkpoints), then edit.
	model, _ = model.Update(kDown)
	model, _ = model.Update(kDown)
	model, _ = model.Update(kEnter)
	model = clearCell(model)

	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyExtended, Text: "2, 5,x9"})
	mm, _ := model.(Model)
	assert.Equal(t, "2, 5,9", mm.configEdit.input)
}

// TestConfigRows_GridTracksRounds confirms the navigable grid grows with the
// round count and chunks to fit the width.
func TestConfigRows_GridTracksRounds(t *testing.T) {
	t.Parallel()

	m := Model{width: 140, quiz: quiz.Quiz{Config: quiz.Config{Rounds: 8, QuestionsPerRound: 10}}}

	rows := m.configRows()
	require.GreaterOrEqual(t, len(rows), 4) // 3 scalar rows + at least one grid row

	count := 0
	for _, row := range rows[3:] {
		count += len(row)
	}

	assert.Equal(t, 8, count, "one grid cell per round")

	// A narrow viewport wraps the same cells across several rows.
	m.width = 30
	rows = m.configRows()

	count = 0
	for _, row := range rows[3:] {
		count += len(row)
	}

	assert.Equal(t, 8, count)
	assert.Greater(t, len(rows[3:]), 1, "narrow width wraps the grid")
}

// TestConfig_RoundOverrideViaCells edits a per-round box and confirms the
// override is stored, then cleared when set back to the default.
func TestConfig_RoundOverrideViaCells(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.Config{Rounds: 4, QuestionsPerRound: 10})

	// Down 3 to the first grid row, Right 2 to R3.
	model, _ = model.Update(kDown)
	model, _ = model.Update(kDown)
	model, _ = model.Update(kDown)
	model, _ = model.Update(kRight)
	model, _ = model.Update(kRight)

	// Edit R3 -> 11.
	model, _ = model.Update(kEnter)
	model = clearCell(model)
	model = typeStr(model, "11")
	model, _ = model.Update(kEnter)

	mm, _ := model.(Model)
	assert.Equal(t, 11, mm.quiz.Config.RoundMaxPoints["3"])

	// Edit R3 back to the default (10) -> override removed.
	model, _ = model.Update(kEnter)
	model = clearCell(model)
	model = typeStr(model, "10")
	model, _ = model.Update(kEnter)

	mm, _ = model.(Model)
	_, ok := mm.quiz.Config.RoundMaxPoints["3"]
	assert.False(t, ok, "value equal to default drops the override")
}

// TestConfig_RoundsDecreaseDropsOverrides confirms shrinking the round count
// prunes overrides and checkpoints that fall out of range.
func TestConfig_RoundsDecreaseDropsOverrides(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.Config{
		Rounds: 8, QuestionsPerRound: 10,
		RoundMaxPoints: map[string]int{"6": 11},
		Checkpoints:    []int{4, 8},
	})

	model, _ = model.Update(kEnter) // edit Rounds
	model = clearCell(model)
	model = typeStr(model, "3")
	model, _ = model.Update(kEnter)

	mm, _ := model.(Model)
	assert.Equal(t, 3, mm.quiz.Config.Rounds)
	_, ok := mm.quiz.Config.RoundMaxPoints["6"]
	assert.False(t, ok, "override beyond new round count dropped")
	assert.Empty(t, mm.quiz.Config.Checkpoints, "checkpoints beyond new round count dropped")
}

// TestConfig_LegacyMaxPointsDropped confirms the legacy global MaxPoints is
// dropped: the grid baselines at questions per round, and only the edited
// round gets an explicit override.
func TestConfig_LegacyMaxPointsDropped(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.Config{Rounds: 3, QuestionsPerRound: 10, MaxPoints: 20})

	mm, _ := model.(Model)
	// Before any edit the grid shows questions per round, not the old 20.
	assert.Equal(t, "10", mm.configCellValue(configCell{Kind: cfgRoundMax, Round: 1}))

	// Down 3 to R1, Right to R2, edit R2 -> 15.
	model, _ = model.Update(kDown)
	model, _ = model.Update(kDown)
	model, _ = model.Update(kDown)
	model, _ = model.Update(kRight)
	model = typeStr(model, "15") // type-to-edit, no Enter needed first
	model, _ = model.Update(kEnter)

	mm, _ = model.(Model)
	assert.Equal(t, 0, mm.quiz.Config.MaxPoints, "legacy MaxPoints dropped")
	assert.Equal(t, 15, mm.quiz.Config.RoundMaxPoints["2"])
	_, ok := mm.quiz.Config.RoundMaxPoints["1"]
	assert.False(t, ok, "untouched rounds keep the questions-per-round baseline")
}

// TestConfig_TypeToEdit confirms typing a digit on a focused cell starts the
// edit immediately, replacing the value.
func TestConfig_TypeToEdit(t *testing.T) {
	t.Parallel()

	model := openConfig(t, quiz.DefaultConfig())

	// Rounds focused on open; just type without pressing Enter first.
	model = typeStr(model, "5")
	mm, _ := model.(Model)
	require.True(t, mm.configEdit.editing, "typing starts the edit")
	assert.Equal(t, "5", mm.configEdit.input, "typed digit replaces the value")

	model, _ = model.Update(kEnter)
	mm, _ = model.(Model)
	assert.Equal(t, 5, mm.quiz.Config.Rounds)
}

// TestConfig_RenderShowsGrid is a smoke test for the form rendering.
func TestConfig_RenderShowsGrid(t *testing.T) {
	t.Parallel()

	m := Model{
		width: 140, height: 30, mode: ModeConfig,
		configEdit: configState{focus: configCell{Kind: cfgRounds}},
		quiz:       quiz.Quiz{Config: quiz.Config{Rounds: 4, QuestionsPerRound: 10}},
	}

	out := m.renderConfig()
	require.NotEmpty(t, out)
	assert.Contains(t, out, "Max points per round")
	assert.Contains(t, out, "R4")
}
