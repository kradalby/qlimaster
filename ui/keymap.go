package ui

import "slices"

// KeyMap is the central list of every keybind in qlimaster. Keeping the
// mapping here in one place makes it easy to audit the user-facing
// surface and to update the help overlay.
//
// Each entry names the key (as returned by tea.KeyPressMsg.String()) and a
// short label describing the action. The actual dispatch happens in
// mode-specific Update functions that consult this map by name.
type KeyMap struct {
	// Global bindings active in every mode where it makes sense.
	Quit       []string
	ForceQuit  []string
	ToggleHelp []string
	Redraw     []string

	// Mode entry.
	EnterScore []string
	EditScore  []string
	AddTeam    []string
	Config     []string
	Export     []string
	ReadOut    []string
	ForceSort  []string
	Refresh    []string

	// Movement.
	Up     []string
	Down   []string
	Left   []string
	Right  []string
	Top    []string
	Bottom []string
	First  []string
	Last   []string

	// Common controls.
	Enter  []string
	Escape []string
	Tab    []string
	Back   []string
	Clear  []string

	// Edit mode extras.
	Delete     []string
	DeleteTeam []string // 'dd'

	// Export menu.
	ExportCSV  []string
	ExportXLSX []string
	ExportBoth []string
}

// DefaultKeyMap returns the project-wide keymap.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:       []string{"q"},
		ForceQuit:  []string{"ctrl+c"},
		ToggleHelp: []string{"?"},
		Redraw:     []string{"ctrl+l"},

		EnterScore: []string{"e"},
		EditScore:  []string{"i"},
		AddTeam:    []string{"a"},
		Config:     []string{":"},
		Export:     []string{"E"},
		ReadOut:    []string{"R"},
		ForceSort:  []string{"s"},
		Refresh:    []string{"r"},

		Up:     []string{"up", "k"},
		Down:   []string{"down", "j"},
		Left:   []string{"left", "h"},
		Right:  []string{"right", "l"},
		Top:    []string{"g"},
		Bottom: []string{"G"},
		First:  []string{"0"},
		Last:   []string{"$"},

		Enter:  []string{"enter", "\n", "\r"},
		Escape: []string{"esc", "escape"},
		Tab:    []string{"tab", "\t"},
		Back:   []string{"ctrl+r"},
		Clear:  []string{"ctrl+u"},

		Delete:     []string{"x", "delete"},
		DeleteTeam: []string{"dd"},

		ExportCSV:  []string{"c"},
		ExportXLSX: []string{"x"},
		ExportBoth: []string{"b"},
	}
}

// matches reports whether s matches any entry in keys.
func matches(keys []string, s string) bool {
	return slices.Contains(keys, s)
}
