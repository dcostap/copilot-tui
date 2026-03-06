package app

import "charm.land/lipgloss/v2"

type styleSet struct {
	Footer          lipgloss.Style
	Meta            lipgloss.Style
	Palette         lipgloss.Style
	PaletteSelected lipgloss.Style
	UserPrefix      lipgloss.Style
	ReasoningPrefix lipgloss.Style
	ToolPrefix      lipgloss.Style
	ErrorPrefix     lipgloss.Style
}

func newStyles() styleSet {
	return styleSet{
		Footer: lipgloss.NewStyle().
			Faint(true).
			Foreground(lipgloss.Color("244")),
		Meta: lipgloss.NewStyle().
			Faint(true).
			Foreground(lipgloss.Color("242")),
		Palette: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		PaletteSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true),
		UserPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true),
		ReasoningPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")),
		ToolPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Bold(true),
		ErrorPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
	}
}
