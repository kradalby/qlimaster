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

// renderFooter returns the three-line bottom banner:
//
//	line 1: thin rule
//	line 2: mode badge + status/toast on solid footer band
//	line 3: contextual hotkey helper on the same band
//
// All three lines are exactly width columns wide.
func renderFooter(width int, mode Mode, status string, hints []footerHint) string {
	if width <= 0 {
		return ""
	}
	rule := styles.Separator.Render(strings.Repeat("─", width))
	band := modeStatusBand(modeBadge(mode), status, width)
	hintsLine := hintsBand(hints, width)
	return lipgloss.JoinVertical(lipgloss.Left, rule, band, hintsLine)
}

func modeBadge(mode Mode) string {
	return styles.ModeBadge.Render(" " + mode.Label() + " ")
}

// modeStatusBand composes the mode-badge-plus-status line on the footer
// background, padded to full width.
func modeStatusBand(badge, status string, width int) string {
	left := badge
	if status != "" {
		left += "  " + styles.Toast.Render(status)
	}
	lw := lipgloss.Width(left)
	if lw < width {
		left += lipgloss.NewStyle().Background(pal.BgFooter).Render(strings.Repeat(" ", width-lw))
	}
	return styles.BottomBarBase.Render(left)
}

// hintsBand renders the keymap helper line, padded to full width on the
// footer background.
func hintsBand(hints []footerHint, width int) string {
	body := "  " + renderHints(hints)
	w := lipgloss.Width(body)
	if w < width {
		body += lipgloss.NewStyle().Background(pal.BgFooter).Render(strings.Repeat(" ", width-w))
	}
	return styles.BottomBarBase.Render(body)
}

func renderHints(hints []footerHint) string {
	if len(hints) == 0 {
		return ""
	}
	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		parts = append(parts, styles.HintKey.Render(h.Key)+" "+styles.Hint.Render(h.Label))
	}
	sep := styles.Hint.Render("  ·  ")
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
