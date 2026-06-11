package ui

import (
	"strings"
	"unicode"
)

// isArrowUp and isArrowDown check for the actual arrow keys and reject
// the hjkl aliases. Text-input fields (score prompts, team-name
// pickers, config form fields) must accept letters verbatim instead of
// treating "k" as "up" or "j" as "down".
func isArrowUp(k string) bool   { return k == "up" }
func isArrowDown(k string) bool { return k == "down" }

// sanitizeText returns text unchanged if it contains only printable
// characters, or "" if any control character (ESC, newline, tab, C0,
// etc.) is present.
//
// Text-input handlers use this as a defensive all-or-nothing filter
// against terminal escape-sequence noise - cursor position reports,
// background/foreground color replies, OSC responses, and so on -
// that can leak into [tea.KeyPressMsg].Text on startup or after
// terminal capability queries and otherwise land verbatim in an input
// buffer. Dropping only the control bytes is not enough: the digits
// and brackets embedded in e.g. "\x1b[75;1R" would still slip through
// and corrupt the form. Normal single-character typing never contains
// a control rune in Text (Enter and Tab are caught by their keybind
// before any fallthrough to text-append), so rejecting the whole blob
// when one is seen is safe.
func sanitizeText(text string) string {
	if text == "" {
		return ""
	}

	if strings.ContainsFunc(text, unicode.IsControl) {
		return ""
	}

	return text
}

// filterRunes returns text with only the runes for which keep returns
// true. Used by restricted input fields (e.g. numeric config fields)
// that want to silently discard illegal characters rather than store
// them and fail at submit time.
func filterRunes(text string, keep func(rune) bool) string {
	if text == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(text))

	for _, r := range text {
		if keep(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// isIncompleteCSI reports whether s is the prefix of a CSI escape sequence
// that has not yet reached its final byte - i.e. ESC '[' followed only by
// parameter/intermediate bytes (0x20-0x3F). Such a fragment is the decoder's
// way of telling us a terminal report was split mid-sequence and its tail is
// about to arrive as stray key presses.
func isIncompleteCSI(s string) bool {
	if len(s) < 2 || s[0] != 0x1b || s[1] != '[' {
		return false
	}

	last := s[len(s)-1]

	return last < 0x40
}

// handleSwallow consumes one trailing byte of a split CSI sequence. It
// returns the updated model and whether the key was swallowed. Parameter and
// intermediate bytes (0x20-0x3F) keep the swallow armed and are dropped; the
// first final byte (0x40-0x7E) terminates the sequence, clears the swallow,
// and is dropped. Anything else (a control key such as Enter, or a real
// keystroke) means the sequence ended early: the swallow is released and the
// key is passed back to normal handling.
func (m Model) handleSwallow(code rune) (Model, bool) {
	switch {
	case code >= 0x40 && code <= 0x7e:
		m.csiSwallow = false

		return m, true
	case code >= 0x20 && code <= 0x3f:
		return m, true
	default:
		m.csiSwallow = false

		return m, false
	}
}

// isDigit reports whether r is an ASCII decimal digit.
func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// isCheckpointChar reports whether r is legal in the Checkpoints
// comma-separated int-list field: digits, comma, or whitespace.
func isCheckpointChar(r rune) bool {
	return isDigit(r) || r == ',' || r == ' '
}
