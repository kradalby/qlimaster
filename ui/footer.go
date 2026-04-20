package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// footerHint is a (key, label) pair displayed in the bottom keymap bar.
type footerHint struct {
	Key   string
	Label string
}

// renderFooter returns the two-line bottom bar:
//
//	[MODE]   <status / toast>
//	 key label | key label | ...
func renderFooter(width int, mode Mode, status string, hints []footerHint) string {
	if width <= 0 {
		return ""
	}
	badge := styles.ModeBadge.Render(mode.Label())
	statusStr := styles.Toast.Render(status)

	firstLine := badge
	if statusStr != "" {
		firstLine += " " + statusStr
	}
	firstLine = padLine(firstLine, width)
	firstLine = styles.BottomBarBase.Render(firstLine)

	second := renderHints(hints)
	secondLine := padLine(" "+second, width)
	secondLine = styles.BottomBarBase.Render(secondLine)

	return lipgloss.JoinVertical(lipgloss.Left, firstLine, secondLine)
}

func renderHints(hints []footerHint) string {
	if len(hints) == 0 {
		return ""
	}
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, styles.HintKey.Render(h.Key)+" "+styles.Hint.Render(h.Label))
	}
	sep := styles.Hint.Render(" | ")
	return strings.Join(parts, sep)
}

// padLine right-pads s (measured after ANSI stripping) to exactly width
// columns.
func padLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
