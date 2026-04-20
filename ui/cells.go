package ui

// CellKind classifies the addressable cells in a data row. It is the
// enum used by edit-mode navigation and by the renderer to decide which
// cell (if any) should carry the cursor highlight.
type CellKind int

const (
	// CellNone means "no cell has focus"; used by Normal mode.
	CellNone CellKind = iota
	// CellPosition is the read-only ranking position column.
	CellPosition
	// CellTeam is the team name (editable).
	CellTeam
	// CellPlayers is the free-text players column (editable when visible).
	CellPlayers
	// CellRound refers to a particular round's score (editable).
	CellRound
	// CellCheckpoint is a cumulative total column (read-only).
	CellCheckpoint
	// CellTotal is the grand total column (read-only).
	CellTotal
)

// Cell identifies a single cell in a data row. Round and Checkpoint use
// the Round field to carry the round number.
type Cell struct {
	Kind  CellKind
	Round int // valid for CellRound and CellCheckpoint
}

// NoCell is the zero value of Cell and means "no focus".
var NoCell = Cell{Kind: CellNone}

// IsEditable reports whether cells of this kind can be edited (written
// to via quiz.Apply).
func (c Cell) IsEditable() bool {
	switch c.Kind {
	case CellTeam, CellPlayers, CellRound:
		return true
	default:
		return false
	}
}

// Equal reports equality of two cells.
func (c Cell) Equal(o Cell) bool {
	return c.Kind == o.Kind && c.Round == o.Round
}

// AddressableCells returns the left-to-right sequence of cell
// identifiers visible in the given layout. Arrow-key navigation in Edit
// mode walks exactly this sequence, so invisible columns (e.g. Players
// in narrow breakpoints) are automatically skipped.
func AddressableCells(l Layout) []Cell {
	out := make([]Cell, 0, 4+len(l.VisibleRounds)+len(l.VisibleCheckpts))
	out = append(out, Cell{Kind: CellPosition}, Cell{Kind: CellTeam})
	if l.ShowPlayers {
		out = append(out, Cell{Kind: CellPlayers})
	}
	visibleCheckpt := map[int]bool{}
	for _, c := range l.VisibleCheckpts {
		visibleCheckpt[c] = true
	}
	for _, r := range l.VisibleRounds {
		out = append(out, Cell{Kind: CellRound, Round: r})
		if visibleCheckpt[r] {
			out = append(out, Cell{Kind: CellCheckpoint, Round: r})
		}
	}
	out = append(out, Cell{Kind: CellTotal})
	return out
}

// IndexOf returns the position of c in the slice, or -1 if absent.
func cellIndexOf(cells []Cell, c Cell) int {
	for i, x := range cells {
		if x.Equal(c) {
			return i
		}
	}
	return -1
}
