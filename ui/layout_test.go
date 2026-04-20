package ui_test

import (
	"testing"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/ui"
	"github.com/stretchr/testify/assert"
)

func TestCompute_Breakpoints(t *testing.T) {
	t.Parallel()

	cfg := quiz.DefaultConfig()
	tests := []struct {
		width int
		want  ui.Breakpoint
	}{
		{200, ui.BreakpointFull},
		{140, ui.BreakpointFull},
		{120, ui.BreakpointNoPlayers},
		{100, ui.BreakpointNoPlayers},
		{90, ui.BreakpointMinimalCheckpoints},
		{80, ui.BreakpointMinimalCheckpoints},
		{60, ui.BreakpointCompact},
		{40, ui.BreakpointCompact},
	}
	for _, tc := range tests {
		l := ui.Compute(tc.width, 40, cfg, 5)
		assert.Equalf(t, tc.want, l.Breakpoint, "width=%d", tc.width)
	}
}

func TestCompute_ShowPlayers(t *testing.T) {
	t.Parallel()

	cfg := quiz.DefaultConfig()
	l := ui.Compute(150, 40, cfg, 0)
	assert.True(t, l.ShowPlayers)

	l = ui.Compute(110, 40, cfg, 0)
	assert.False(t, l.ShowPlayers)
}

func TestCompute_VisibleRounds(t *testing.T) {
	t.Parallel()

	cfg := quiz.Config{Rounds: 8, QuestionsPerRound: 10, Checkpoints: []int{4, 8}}
	l := ui.Compute(150, 40, cfg, 0)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, l.VisibleRounds)

	// Compact view shows the current round only.
	l = ui.Compute(60, 40, cfg, 5)
	assert.Equal(t, []int{5}, l.VisibleRounds)

	// Compact view with no round entered defaults to round 1.
	l = ui.Compute(60, 40, cfg, 0)
	assert.Equal(t, []int{1}, l.VisibleRounds)
}

func TestCompute_MinimalCheckpoints(t *testing.T) {
	t.Parallel()

	cfg := quiz.Config{Rounds: 8, QuestionsPerRound: 10, Checkpoints: []int{4, 8}}
	// Narrow with only round 2 entered: no checkpoint visible yet.
	l := ui.Compute(85, 40, cfg, 2)
	assert.Empty(t, l.VisibleCheckpts)

	// Round 5 entered: checkpoint 4 should be shown alone.
	l = ui.Compute(85, 40, cfg, 5)
	assert.Equal(t, []int{4}, l.VisibleCheckpts)

	// Round 8 entered: checkpoint 8 (highest <= 8) shown.
	l = ui.Compute(85, 40, cfg, 8)
	assert.Equal(t, []int{8}, l.VisibleCheckpts)
}

func TestCompute_TableHeight(t *testing.T) {
	t.Parallel()

	cfg := quiz.DefaultConfig()
	l := ui.Compute(150, 30, cfg, 0)
	// height - topBanner(3) - bottomBanner(3) - tableChrome(4)
	assert.Equal(t, 30-3-3-4, l.TableHeight)

	// Very short viewport clamps to zero.
	l = ui.Compute(150, 5, cfg, 0)
	assert.Equal(t, 0, l.TableHeight)
}

func TestCompute_TeamWidthCapped(t *testing.T) {
	t.Parallel()

	cfg := quiz.DefaultConfig()
	// On a very wide terminal, surplus must go to RightPad, not TeamWidth.
	l := ui.Compute(300, 40, cfg, 0)
	assert.LessOrEqual(t, l.TeamWidth, 28)
	assert.Positive(t, l.RightPad)
}

func TestCompute_TeamWidthNonNegative(t *testing.T) {
	t.Parallel()

	cfg := quiz.DefaultConfig()
	for _, w := range []int{40, 60, 80, 100, 140, 200} {
		l := ui.Compute(w, 40, cfg, 0)
		assert.GreaterOrEqual(t, l.TeamWidth, 8, "w=%d", w)
	}
}
