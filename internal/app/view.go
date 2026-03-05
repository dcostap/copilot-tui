package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"copilot-tui/internal/ui"
)

func (m *model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Starting..."
	}

	timelinePanel := m.styles.Panel.
		Width(m.width).
		Render(m.viewport.View())

	inputPanel := m.styles.Panel.
		Width(m.width).
		Render(m.input.View())

	parts := []string{timelinePanel}
	if m.showPalette {
		parts = append(parts, m.renderPalette())
	}
	parts = append(parts, inputPanel, m.renderFooter())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *model) renderPalette() string {
	if len(m.paletteItems) == 0 {
		return m.styles.Palette.Width(m.width).Render("No palette commands available")
	}

	availableWidth := m.width - 2
	if availableWidth < 1 {
		availableWidth = 1
	}

	lines := make([]string, 0, len(m.paletteItems))
	for i, item := range m.paletteItems {
		prefix := "  "
		if i == m.paletteIndex {
			prefix = "> "
		}
		line := ui.Truncate(prefix+item, availableWidth)
		if i == m.paletteIndex {
			line = m.styles.PaletteSelected.Render(line)
		}
		lines = append(lines, line)
	}

	return m.styles.Palette.
		Width(m.width).
		Render(strings.Join(lines, "\n"))
}

func (m *model) renderFooter() string {
	mode := "chat"
	if m.showPalette {
		mode = "palette"
	}

	text := fmt.Sprintf(
		"mode:%s | scenario:%s | %s | Ctrl+P palette | Enter send | Shift+Enter newline | Ctrl+C quit",
		mode,
		m.currentScenario,
		m.status,
	)

	return m.styles.Footer.
		Width(m.width).
		Render(ui.Truncate(text, m.width))
}
