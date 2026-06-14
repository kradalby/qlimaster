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

// TestConfig_UpdatesQuiz walks through the config flow and confirms the
// quiz config is mutated.
func TestConfig_UpdatesQuiz(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	mm, _ := model.(Model)
	require.Equal(t, ModeConfig, mm.mode)

	// Select all of the rounds field and replace with "6".
	for range 2 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}

	model, _ = model.Update(teaKey("6"))

	// Tab to questions, keep.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	// Tab past max points, keep.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	// Tab past round max points, keep.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	// Tab to checkpoints, replace with "3,6".
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	for range 3 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}

	model, _ = model.Update(teaKey("3"))
	model, _ = model.Update(teaKey(","))
	model, _ = model.Update(teaKey("6"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ = model.(Model)
	assert.Equal(t, ModeNormal, mm.mode)
	assert.Equal(t, 6, mm.quiz.Config.Rounds)
	assert.Equal(t, []int{3, 6}, mm.quiz.Config.Checkpoints)
}

// TestConfig_RejectsInvalid shows inline error on unparseable input.
func TestConfig_RejectsInvalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})

	model, _ = model.Update(teaKey(":"))
	for range 2 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}
	// "x" is not a digit and is now silently dropped by configAppend's
	// character filter. Submitting with an empty Rounds field must still
	// surface the inline error.
	model, _ = model.Update(teaKey("x"))
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, ModeConfig, mm.mode)
	assert.NotEmpty(t, mm.errMsg)
}

// TestConfig_IgnoresEscapeSequenceNoise is a regression test for the
// "new quiz opens with garbage Rounds value" bug. Stray escape
// sequences (e.g. a cursor position report leaking through the key
// path) arriving in tea.KeyPressMsg.Text must not be appended to the
// numeric form fields - not even the digits inside them.
func TestConfig_IgnoresEscapeSequenceNoise(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	mm, _ := model.(Model)
	require.Equal(t, ModeConfig, mm.mode)

	// Simulate the exact payload from the reported bug: ESC plus CSI
	// introducer plus a cursor-position-report tail, all arriving in a
	// single Text blob. The embedded digits (7,5,1) must not leak
	// through - the whole blob is noise and must be rejected wholesale.
	noise := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "\x1b[75;1R"}
	model, _ = model.Update(noise)
	// A raw-control-char variant (DEL + other noise) - also rejected.
	raw := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "\x7f[1;2R"}
	model, _ = model.Update(raw)
	// Plain-text garbage with embedded digits contains no control
	// characters, so it reaches configAppend. There the per-field
	// digit filter keeps the "3" and drops "abc" and "def".
	mixed := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "abc3def"}
	model, _ = model.Update(mixed)

	mm, _ = model.(Model)
	// Seed "8" plus the lone "3" that survived the mixed payload.
	// Nothing from the escape-sequence blobs landed in the field.
	assert.Equal(t, "83", mm.configEdit.rounds)
	assert.NotContains(t, mm.configEdit.rounds, "7")
	assert.NotContains(t, mm.configEdit.rounds, "5")
	assert.NotContains(t, mm.configEdit.rounds, "1")
	assert.NotContains(t, mm.configEdit.rounds, "[")
	assert.NotContains(t, mm.configEdit.rounds, "R")
	assert.NotContains(t, mm.configEdit.rounds, "\x1b")
}

// TestConfig_IgnoresSplitEscapeSequence reproduces the harder variant of
// the garbage-Rounds bug: a terminal report (cursor position / DECRQM mode
// report) racing the Config overlay open arrives split across reads, so the
// decoder emits the CSI prefix as a uv.UnknownEvent and the parameter and
// terminator bytes as individual KeyPressMsgs. The leading ESC and '[' are
// already gone, so the per-keystroke control-char filter cannot recognise
// the leak - only the in-progress-escape signal from the UnknownEvent can.
// Without the swallow, the bare digits ("2", "6") corrupt the focused
// Rounds field, producing values like "800000000000020262".
func TestConfig_IgnoresSplitEscapeSequence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	mm, _ := model.(Model)
	require.Equal(t, ModeConfig, mm.mode)

	// "\x1b[20;26R" split as the decoder delivers it: the prefix lands as
	// an incomplete-CSI UnknownEvent, the tail as bare key presses.
	model, _ = model.Update(uv.UnknownEvent("\x1b[20"))
	for _, r := range []rune{';', '2', '6', 'R'} {
		model, _ = model.Update(teaKey(string(r)))
	}

	mm, _ = model.(Model)
	assert.Equal(t, ModeConfig, mm.mode, "config must stay open")
	assert.Equal(t, "8", mm.configEdit.rounds, "no leaked digits in Rounds")

	// After the terminator the swallow clears, so real typing resumes.
	model, _ = model.Update(teaKey("5"))
	mm, _ = model.(Model)
	assert.Equal(t, "85", mm.configEdit.rounds)
}

// TestConfig_CheckpointsAllowsCommaAndSpace confirms the Checkpoints
// field accepts the separator characters it needs while still
// filtering out non-digit noise.
func TestConfig_CheckpointsAllowsCommaAndSpace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	// Tab past Rounds, Questions, Max points and Round max points to land
	// on Checkpoints.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	// Erase the default "4,8".
	for range 3 {
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	}
	// A mix of legal (digits, comma, space) and illegal (letters)
	// characters with NO control codes - the whole blob reaches
	// configAppend and the per-field filter strips just the letter.
	payload := tea.KeyPressMsg{Code: tea.KeyExtended, Text: "2, 5,x9"}
	model, _ = model.Update(payload)
	mm, _ := model.(Model)
	assert.Equal(t, "2, 5,9", mm.configEdit.checkpoints)
}

func TestParseRoundMaxPoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    map[string]int
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"whitespace only", "   ", nil, false},
		{"single", "3:11", map[string]int{"3": 11}, false},
		{"multiple", "3:11,7:12", map[string]int{"3": 11, "7": 12}, false},
		{"whitespace around", " 3 : 11 , 7:12 ", map[string]int{"3": 11, "7": 12}, false},
		{"trailing comma", "3:11,", map[string]int{"3": 11}, false},
		{"duplicate last wins", "3:11,3:12", map[string]int{"3": 12}, false},
		{"missing colon", "3-11", nil, true},
		{"non-int round", "x:11", nil, true},
		{"non-int value", "3:y", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseRoundMaxPoints(tc.in)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatRoundMaxPoints(t *testing.T) {
	t.Parallel()

	assert.Empty(t, formatRoundMaxPoints(nil))
	// Keys are emitted in ascending numeric order regardless of map order.
	assert.Equal(t, "1:10,2:20", formatRoundMaxPoints(map[string]int{"2": 20, "1": 10}))

	// Format then parse round-trips to the original map.
	in := map[string]int{"3": 11, "7": 12}
	got, err := parseRoundMaxPoints(formatRoundMaxPoints(in))
	require.NoError(t, err)
	assert.Equal(t, in, got)
}

// TestConfig_RoundMaxPointsRoundtrip types a per-round override into the
// config overlay and confirms it lands on the quiz config.
func TestConfig_RoundMaxPointsRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := New(Config{
		Path:       filepath.Join(dir, "quiz.hujson"),
		QuizConfig: quiz.DefaultConfig(),
		QuizRoot:   dir,
	})
	require.NoError(t, err)

	var model tea.Model = m

	model, _ = model.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	model, _ = model.Update(teaKey(":"))
	// Tab past Rounds, Questions and Max points to land on Round max points.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab, Text: "\t"})

	for _, k := range []string{"3", ":", "1", "1"} {
		model, _ = model.Update(teaKey(k))
	}

	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\n"})
	mm, _ := model.(Model)
	assert.Equal(t, ModeNormal, mm.mode)
	assert.Equal(t, 11, mm.quiz.Config.RoundMaxPoints["3"])
}
