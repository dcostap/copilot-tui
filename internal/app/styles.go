package app

import "github.com/charmbracelet/lipgloss"

type styleSet struct {
	Panel           lipgloss.Style
	Footer          lipgloss.Style
	Palette         lipgloss.Style
	PaletteSelected lipgloss.Style
	UserPrefix      lipgloss.Style
	ReasoningPrefix lipgloss.Style
	ToolPrefix      lipgloss.Style
	ErrorPrefix     lipgloss.Style
}

func newStyles() styleSet {
	return styleSet{
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()),
		Footer: lipgloss.NewStyle().
			Faint(true),
		Palette: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Foreground(lipgloss.Color("212")),
		PaletteSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true),
		UserPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		ReasoningPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
		ToolPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true),
		ErrorPrefix: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
	}
}
