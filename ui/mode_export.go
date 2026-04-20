package ui

import (
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/kradalby/qlimaster/export"
)

// exportTo writes the quiz in the given format(s) next to the quiz file.
// format is "csv", "xlsx" or "both". Errors are surfaced as inline status
// messages; the overlay is dismissed on success.
func (m Model) exportTo(format string) (tea.Model, tea.Cmd) {
	dir := filepath.Dir(m.path)
	base := strings.TrimSuffix(filepath.Base(m.path), filepath.Ext(m.path))
	var paths []string
	if format == "csv" || format == "both" {
		if err := export.CSVFile(filepath.Join(dir, base+".csv"), m.quiz); err != nil {
			m.status = "export failed: " + err.Error()
			return m, clearStatusCmd(2 * time.Second)
		}
		paths = append(paths, base+".csv")
	}
	if format == "xlsx" || format == "both" {
		if err := export.XLSX(filepath.Join(dir, base+".xlsx"), m.quiz); err != nil {
			m.status = "export failed: " + err.Error()
			return m, clearStatusCmd(2 * time.Second)
		}
		paths = append(paths, base+".xlsx")
	}
	m.status = "[ok] exported " + strings.Join(paths, ", ")
	m.statusExpiry = time.Now().Add(2 * time.Second)
	m.mode = ModeNormal
	return m, clearStatusCmd(2 * time.Second)
}

// renderExport draws the centered export-menu overlay.
func (m Model) renderExport() string {
	title := styles.OverlayTitle.Render("Export")
	lines := []string{
		title,
		"",
		" [c] CSV",
		" [x] XLSX",
		" [b] Both",
		"",
		" Esc cancel",
	}
	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return styles.OverlayBorder.Padding(1, 2).Render(body)
}

// renderHelp draws the centered help overlay listing every keybind.
func (m Model) renderHelp() string {
	title := styles.OverlayTitle.Render("qlimaster Help")
	lines := []string{
		title,
		"",
		"Modes",
		"  e  enter-score     i  edit-score     a  add team",
		"  :  config          E  export         ?  help",
		"  q  quit            Ctrl+C  force quit",
		"",
		"Navigation (normal / edit)",
		"  hjkl / arrows    g/G first/last row    0/$ first/last col",
		"",
		"Enter-score flow",
		"  round# -> Enter -> fuzzy team -> Enter -> score -> Enter -> next",
		"",
		"Score shortcuts",
		"  1 -> 1    1. -> 1.5    , -> 0.5    1,5 -> 1.5",
		"",
		"Press ? or Esc to dismiss",
	}
	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return styles.OverlayBorder.Padding(1, 2).Render(body)
}
