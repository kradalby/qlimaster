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

	ShowPlayers     bool
	VisibleRounds   []int // round numbers, in left-to-right order
	VisibleCheckpts []int // checkpoint round numbers kept

	// Column widths.
	PosWidth     int
	TeamWidth    int
	PlayersWidth int
	RoundWidth   int
	CheckptWidth int
	TotalWidth   int
}

// Compute chooses a responsive layout for the given viewport size and
// configuration. lastEnteredRound is used to select which single
// checkpoint to show in the minimal-checkpoint breakpoint; pass 0 if no
// round has been entered yet.
func Compute(width, height int, cfg quiz.Config, lastEnteredRound int) Layout {
	const (
		topBarLines    = 1
		bottomBarLines = 2
		// Separator lines above and below data rows inside the table area.
		tableChrome = 4
	)

	layout := Layout{
		Width:        width,
		Height:       height,
		TableHeight:  max(height-topBarLines-bottomBarLines-tableChrome, 0),
		PosWidth:     4,
		TotalWidth:   7,
		RoundWidth:   4,
		CheckptWidth: 4,
		PlayersWidth: 12,
	}
	layout.Breakpoint = classify(width)

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
	case BreakpointMinimalCheckpoints:
		layout.ShowPlayers = false
		layout.PlayersWidth = 0
		layout.VisibleRounds = allRounds(cfg.Rounds)
		layout.VisibleCheckpts = minimalCheckpoints(cfg.Checkpoints, lastEnteredRound)
	case BreakpointCompact:
		layout.ShowPlayers = false
		layout.PlayersWidth = 0
		layout.VisibleRounds = compactRound(cfg.Rounds, lastEnteredRound)
		layout.VisibleCheckpts = nil
	}

	layout.TeamWidth = computeTeamWidth(layout)
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

// computeTeamWidth fills the remaining horizontal space with the team
// column. It assumes a 3-char padding per column separator (" | "), which
// matches the table renderer's joiner.
func computeTeamWidth(l Layout) int {
	const sep = 3 // " | "
	used := l.PosWidth + sep + sep + l.TotalWidth
	if l.ShowPlayers {
		used += l.PlayersWidth + sep
	}
	used += (l.RoundWidth + sep) * len(l.VisibleRounds)
	used += (l.CheckptWidth + sep) * len(l.VisibleCheckpts)
	return max(l.Width-used, 8)
}
