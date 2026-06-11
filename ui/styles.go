package ui

import "github.com/charmbracelet/lipgloss"

// palette groups the qlimaster colour tokens. Every token uses
// lipgloss.AdaptiveColor so the UI looks correct on both light and dark
// terminals. The palette is pink-first by design.
type palette struct {
	PinkHot     lipgloss.AdaptiveColor
	PinkNeon    lipgloss.AdaptiveColor
	PinkSoft    lipgloss.AdaptiveColor
	PinkDim     lipgloss.AdaptiveColor
	Purple      lipgloss.AdaptiveColor
	Yellow      lipgloss.AdaptiveColor
	Green       lipgloss.AdaptiveColor
	Red         lipgloss.AdaptiveColor
	Gold        lipgloss.AdaptiveColor
	Silver      lipgloss.AdaptiveColor
	Bronze      lipgloss.AdaptiveColor
	FgMuted     lipgloss.AdaptiveColor
	FgBody      lipgloss.AdaptiveColor
	BgHeader    lipgloss.AdaptiveColor
	BgFooter    lipgloss.AdaptiveColor
	BgSelect    lipgloss.AdaptiveColor
	BgZebra     lipgloss.AdaptiveColor
	BgAvg       lipgloss.AdaptiveColor
	BgHeaderRow lipgloss.AdaptiveColor
}

// pal is the shared palette instance.
var pal = palette{
	PinkHot:     lipgloss.AdaptiveColor{Light: "#d6156e", Dark: "#ff5fd7"},
	PinkNeon:    lipgloss.AdaptiveColor{Light: "#c8006a", Dark: "#ff00aa"},
	PinkSoft:    lipgloss.AdaptiveColor{Light: "#ff6bb5", Dark: "#ffafd7"},
	PinkDim:     lipgloss.AdaptiveColor{Light: "#a8385e", Dark: "#d75f87"},
	Purple:      lipgloss.AdaptiveColor{Light: "#7a2fcf", Dark: "#af5fff"},
	Yellow:      lipgloss.AdaptiveColor{Light: "#b88900", Dark: "#ffd75f"},
	Green:       lipgloss.AdaptiveColor{Light: "#138020", Dark: "#5fd787"},
	Red:         lipgloss.AdaptiveColor{Light: "#c20000", Dark: "#ff5f5f"},
	Gold:        lipgloss.AdaptiveColor{Light: "#b8860b", Dark: "#ffd700"},
	Silver:      lipgloss.AdaptiveColor{Light: "#707070", Dark: "#c0c0c0"},
	Bronze:      lipgloss.AdaptiveColor{Light: "#8a5020", Dark: "#cd7f32"},
	FgMuted:     lipgloss.AdaptiveColor{Light: "#6a6a6a", Dark: "#8a8a8a"},
	FgBody:      lipgloss.AdaptiveColor{Light: "#2a1a22", Dark: "#f5e6ef"},
	BgHeader:    lipgloss.AdaptiveColor{Light: "#ffe6f0", Dark: "#2a0a1e"},
	BgFooter:    lipgloss.AdaptiveColor{Light: "#ffd1e6", Dark: "#1a0612"},
	BgSelect:    lipgloss.AdaptiveColor{Light: "#ffb4d6", Dark: "#5a1a3c"},
	BgZebra:     lipgloss.AdaptiveColor{Light: "#fff5f9", Dark: "#1a0a14"},
	BgAvg:       lipgloss.AdaptiveColor{Light: "#ffe6f0", Dark: "#3a0a24"},
	BgHeaderRow: lipgloss.AdaptiveColor{Light: "#ffd1e6", Dark: "#2a0a1e"},
}

// style bundles the lipgloss styles derived from the palette. Precomputed
// once so rendering hot paths do not re-allocate style values.
type style struct {
	TopBarBase    lipgloss.Style
	AppName       lipgloss.Style
	Stats         lipgloss.Style
	DateRight     lipgloss.Style
	BottomBarBase lipgloss.Style
	ModeBadge     lipgloss.Style
	Hint          lipgloss.Style
	HintKey       lipgloss.Style
	TableHeader   lipgloss.Style
	Separator     lipgloss.Style
	ThickRule     lipgloss.Style
	RowFocus      lipgloss.Style
	RowZebra      lipgloss.Style
	CellFocus     lipgloss.Style
	CellEditing   lipgloss.Style
	Averages      lipgloss.Style
	AveragesBand  lipgloss.Style
	Gold          lipgloss.Style
	Silver        lipgloss.Style
	Bronze        lipgloss.Style
	Perfect       lipgloss.Style
	Error         lipgloss.Style
	Toast         lipgloss.Style
	OverlayBorder lipgloss.Style
	OverlayTitle  lipgloss.Style
	FuzzyArrow    lipgloss.Style
	FuzzyMatch    lipgloss.Style
	Dimmed        lipgloss.Style
}

// styles is the shared, lazily-built style bundle. It is safe to reuse
// across Model instances in tests since lipgloss styles are immutable
// after construction.
var styles = buildStyles()

func buildStyles() style {
	return style{
		TopBarBase:    lipgloss.NewStyle().Background(pal.BgHeader).Foreground(pal.FgBody),
		AppName:       lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot).Background(pal.BgHeader),
		Stats:         lipgloss.NewStyle().Foreground(pal.FgMuted).Background(pal.BgHeader),
		DateRight:     lipgloss.NewStyle().Italic(true).Foreground(pal.Purple).Background(pal.BgHeader),
		BottomBarBase: lipgloss.NewStyle().Background(pal.BgFooter).Foreground(pal.FgBody),
		ModeBadge:     lipgloss.NewStyle().Bold(true).Foreground(pal.FgBody).Background(pal.PinkSoft).Padding(0, 1),
		Hint:          lipgloss.NewStyle().Foreground(pal.PinkDim).Background(pal.BgFooter),
		HintKey:       lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot).Background(pal.BgFooter),
		TableHeader:   lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot).Background(pal.BgHeaderRow),
		Separator:     lipgloss.NewStyle().Foreground(pal.PinkDim),
		ThickRule:     lipgloss.NewStyle().Foreground(pal.PinkHot),
		RowFocus:      lipgloss.NewStyle().Background(pal.BgSelect).Foreground(pal.FgBody),
		RowZebra:      lipgloss.NewStyle().Background(pal.BgZebra).Foreground(pal.FgBody),
		CellFocus:     lipgloss.NewStyle().Bold(true).Background(pal.Yellow).Foreground(lipgloss.Color("#000000")),
		CellEditing:   lipgloss.NewStyle().Bold(true).Background(pal.PinkHot).Foreground(lipgloss.Color("#000000")),
		Averages:      lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot).Background(pal.BgAvg),
		AveragesBand:  lipgloss.NewStyle().Background(pal.BgAvg).Foreground(pal.FgBody),
		Gold:          lipgloss.NewStyle().Bold(true).Foreground(pal.Gold),
		Silver:        lipgloss.NewStyle().Bold(true).Foreground(pal.Silver),
		Bronze:        lipgloss.NewStyle().Bold(true).Foreground(pal.Bronze),
		Perfect:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")).Background(pal.PinkNeon),
		Error:         lipgloss.NewStyle().Bold(true).Foreground(pal.Red),
		Toast:         lipgloss.NewStyle().Bold(true).Foreground(pal.Green).Background(pal.BgFooter),
		OverlayBorder: lipgloss.NewStyle().BorderStyle(lipgloss.ThickBorder()).BorderForeground(pal.PinkDim),
		OverlayTitle:  lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot),
		FuzzyArrow:    lipgloss.NewStyle().Bold(true).Foreground(pal.PinkHot),
		FuzzyMatch:    lipgloss.NewStyle().Bold(true).Foreground(pal.Yellow),
		Dimmed:        lipgloss.NewStyle().Foreground(pal.FgMuted),
	}
}
