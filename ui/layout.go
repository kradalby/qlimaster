// Package ui implements the qlimaster full-screen Bubble Tea model.
//
// The package is structured around a single top-level [Model] that
// dispatches input messages by the current [Mode]. All state mutations
// flow through the Model's apply method which calls [quiz.Apply], so the
// ui package never mutates a [quiz.Quiz] directly.
package ui

import (
	"github.com/kradalby/qlimaster/quiz"
)

// Breakpoint describes the horizontal density class of the UI. Narrower
// widths progressively hide columns rather than wrapping or truncating
// content unpredictably.
type Breakpoint int

const (
	// BreakpointFull shows the Players column, every round, and every
	// checkpoint column. Requires at least 140 columns.
	BreakpointFull Breakpoint = iota
	// BreakpointNoPlayers hides the Players column but keeps every round
	// and every checkpoint. Suitable from 100..139 columns.
	BreakpointNoPlayers
	// BreakpointMinimalCheckpoints hides all but the highest configured
	// checkpoint that is <= the last entered round. Suitable from 80..99
	// columns.
	BreakpointMinimalCheckpoints
	// BreakpointCompact shows only Pos, Team, the current round and the
	// Total. Round columns can be scrolled via h/l in this mode.
	BreakpointCompact
)

// Layout describes the concrete widths and visibility of table columns
// for a given viewport size and quiz configuration. It is a pure value
// produced by [Compute] and consumed by the table renderer; no ANSI or
// lipgloss state lives in here.
type Layout struct {
	Width  int
	Height int

	// TableHeight is the number of lines available for the main data
	// table. The header row, averages row and the two surrounding
	// separators are drawn inside this budget.
	TableHeight int

	Breakpoint Breakpoint

	// UseLongLabels is true at the Full breakpoint where header labels
	// spell out "Position", "Round 1", "Halftime R4" instead of
	// abbreviating.
	UseLongLabels bool

	ShowPlayers     bool
	VisibleRounds   []int // round numbers, in left-to-right order
	VisibleCheckpts []int // checkpoint round numbers kept

	// Column widths. These are the visible character widths of each cell
	// between separators.
	PosWidth     int
	TeamWidth    int
	PlayersWidth int
	RoundWidth   int
	CheckptWidth int
	TotalWidth   int

	// RightPad is a trailing blank column at the right edge of the
	// table. Computed to absorb surplus horizontal space when the team
	// column has reached its cap, so no single column blooms to fill the
	// whole viewport.
	RightPad int
}

// teamWidthCap returns the maximum width given to the Team column for a
// given breakpoint. Past this cap, surplus width becomes trailing
// padding. Team names longer than the cap are truncated with an ellipsis
// on render.
func teamWidthCap(bp Breakpoint) int {
	switch bp {
	case BreakpointFull:
		return 28
	case BreakpointNoPlayers:
		return 24
	case BreakpointMinimalCheckpoints:
		return 20
	case BreakpointCompact:
		return 16
	default:
		return 20
	}
}

// Compute chooses a responsive layout for the given viewport size and
// configuration. lastEnteredRound is used to select which single
// checkpoint to show in the minimal-checkpoint breakpoint; pass 0 if no
// round has been entered yet.
func Compute(width, height int, cfg quiz.Config, lastEnteredRound int) Layout {
	const (
		topBarLines    = 3 // thick banner
		bottomBarLines = 3 // thick banner
		tableChrome    = 4 // table top rule, header row, rule above avg, avg row
	)

	layout := Layout{
		Width:        width,
		Height:       height,
		TableHeight:  max(height-topBarLines-bottomBarLines-tableChrome, 0),
		PosWidth:     10,
		TotalWidth:   7,
		RoundWidth:   7,
		CheckptWidth: 8,
		PlayersWidth: 14,
	}
	layout.Breakpoint = classify(width)
	layout.UseLongLabels = layout.Breakpoint == BreakpointFull

	switch layout.Breakpoint {
	case BreakpointFull:
		layout.ShowPlayers = true
		layout.VisibleRounds = allRounds(cfg.Rounds)
		layout.VisibleCheckpts = append([]int(nil), cfg.Checkpoints...)
	case BreakpointNoPlayers:
		layout.ShowPlayers = false
		layout.PlayersWidth = 0
		layout.VisibleRounds = allRounds(cfg.Rounds)
		layout.VisibleCheckpts = append([]int(nil), cfg.Checkpoints...)
		layout.PosWidth = 5
		layout.RoundWidth = 5
		layout.CheckptWidth = 5
	case BreakpointMinimalCheckpoints:
		layout.ShowPlayers = false
		layout.PlayersWidth = 0
		layout.VisibleRounds = allRounds(cfg.Rounds)
		layout.VisibleCheckpts = minimalCheckpoints(cfg.Checkpoints, lastEnteredRound)
		layout.PosWidth = 4
		layout.RoundWidth = 4
		layout.CheckptWidth = 5
		layout.TotalWidth = 6
	case BreakpointCompact:
		layout.ShowPlayers = false
		layout.PlayersWidth = 0
		layout.VisibleRounds = compactRound(cfg.Rounds, lastEnteredRound)
		layout.VisibleCheckpts = nil
		layout.PosWidth = 4
		layout.RoundWidth = 5
		layout.TotalWidth = 6
	}

	layout.TeamWidth, layout.RightPad = computeTeamAndPad(layout)
	return layout
}

// classify chooses a breakpoint based on viewport width.
func classify(width int) Breakpoint {
	switch {
	case width >= 140:
		return BreakpointFull
	case width >= 100:
		return BreakpointNoPlayers
	case width >= 80:
		return BreakpointMinimalCheckpoints
	default:
		return BreakpointCompact
	}
}

func allRounds(n int) []int {
	out := make([]int, n)
	for i := range n {
		out[i] = i + 1
	}
	return out
}

// minimalCheckpoints keeps at most one checkpoint: the highest one at or
// below lastEntered. If no checkpoints qualify (e.g. the quiz hasn't
// advanced to a checkpoint yet), returns nil.
func minimalCheckpoints(all []int, lastEntered int) []int {
	best := 0
	for _, cp := range all {
		if cp <= lastEntered && cp > best {
			best = cp
		}
	}
	if best == 0 {
		return nil
	}
	return []int{best}
}

// compactRound returns the current round (lastEntered clamped to [1, rounds])
// as a single-element list.
func compactRound(rounds, lastEntered int) []int {
	r := min(max(lastEntered, 1), rounds)
	return []int{r}
}

// computeTeamAndPad returns the Team column width and any trailing
// right-pad. Team width is bounded by teamWidthCap; surplus becomes
// right padding so a wide terminal does not leave the Team column
// bloomed.
func computeTeamAndPad(l Layout) (int, int) {
	const frame = 1 // left + right border columns contribute 2 total
	// Separator between cells. The renderer joins with " │ " (3 cols).
	const sep = 3

	// Fixed columns always present: Pos, Team, Total (Team width TBD).
	used := frame*2 + l.PosWidth + sep + sep + l.TotalWidth
	if l.ShowPlayers {
		used += l.PlayersWidth + sep
	}
	used += (l.RoundWidth + sep) * len(l.VisibleRounds)
	used += (l.CheckptWidth + sep) * len(l.VisibleCheckpts)

	remaining := l.Width - used
	if remaining < 10 {
		return max(remaining, 8), 0
	}
	maxTeam := teamWidthCap(l.Breakpoint)
	if remaining <= maxTeam {
		return remaining, 0
	}
	return maxTeam, remaining - maxTeam
}
