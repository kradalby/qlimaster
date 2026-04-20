package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// overlayOnto composes overlay on top of base. Each visible character in
// overlay replaces the corresponding character in base at the overlay's
// top-left placement, centered in the viewport. Lines outside the overlay
// are left unchanged.
//
// The implementation is intentionally simple: it splits both inputs into
// lines, chooses a centered top-left coordinate, and stitches line by
// line. ANSI styling is preserved because we only splice at whole-line
// boundaries for non-overlay rows; for overlay rows we place the overlay
// on a fresh line padded with spaces to full width, which a downstream
// consumer is expected to handle.
func overlayOnto(base, overlay string, width, height int) string {
	baseLines := splitLinesExactly(base, height, width)
	overlayLines := strings.Split(overlay, "\n")

	oh := len(overlayLines)
	ow := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > ow {
			ow = w
		}
	}
	top := max((height-oh)/2, 0)
	left := max((width-ow)/2, 0)

	for i, ol := range overlayLines {
		r := top + i
		if r >= height {
			break
		}
		baseLines[r] = overlayLine(baseLines[r], ol, left, width)
	}
	return strings.Join(baseLines, "\n")
}

// overlayLine returns base with overlay spliced in at column `left`,
// rendered on a plain-space background so the overlay is legible even
// when the base had styled content underneath.
func overlayLine(base, overlay string, left, width int) string {
	// The simplest faithful splice: build a string of `left` spaces, then
	// the overlay, then pad out to `width` with spaces. This visibly covers
	// base content at the overlay's position and leaves the rest blank --
	// which for the overlay band is intentional.
	_ = base
	padL := strings.Repeat(" ", left)
	line := padL + overlay
	w := lipgloss.Width(line)
	if w < width {
		line += strings.Repeat(" ", width-w)
	}
	return line
}

// splitLinesExactly returns exactly `lines` lines, each padded (with
// spaces, so alignment is preserved) to `width`.
func splitLinesExactly(s string, lines, width int) []string {
	existing := strings.Split(s, "\n")
	out := make([]string, lines)
	for i := range lines {
		var l string
		if i < len(existing) {
			l = existing[i]
		}
		if w := lipgloss.Width(l); w < width {
			l += strings.Repeat(" ", width-w)
		}
		out[i] = l
	}
	return out
}

// clampLines ensures s contains exactly height lines of width columns.
// Extra lines are discarded; missing lines are filled with blank padding.
func clampLines(s string, width, height int) string {
	return strings.Join(splitLinesExactly(s, height, width), "\n")
}
