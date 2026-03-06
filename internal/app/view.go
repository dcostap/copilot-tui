package app

import (
	"fmt"
	"strings"

	"copilot-tui/internal/ui"
)

func (m *model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Starting..."
	}

	parts := []string{
		m.viewport.View(),
		m.renderSeparator(),
	}
	if m.showPalette {
		parts = append(parts, m.renderPalette(), m.renderSeparator())
	}
	parts = append(parts, m.input.View(), m.renderFooter())
	return strings.Join(parts, "\n")
}

func (m *model) renderPalette() string {
	if len(m.paletteItems) == 0 {
		return m.styles.Palette.Render("• no palette commands available")
	}

	availableWidth := m.width
	if availableWidth < 1 {
		availableWidth = 1
	}

	lines := make([]string, 0, len(m.paletteItems)+1)
	lines = append(lines, m.styles.Meta.Render("Command palette"))
	for i, item := range m.paletteItems {
		prefix := "• "
		if i == m.paletteIndex {
			prefix = "› "
		}
		line := ui.Truncate(prefix+item, availableWidth)
		if i == m.paletteIndex {
			line = m.styles.PaletteSelected.Render(line)
		} else {
			line = m.styles.Palette.Render(line)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m *model) renderFooter() string {
	mode := "chat"
	if m.showPalette {
		mode = "palette"
	}

	text := fmt.Sprintf(
		"mode:%s · scenario:%s · %s · Ctrl+P palette · Enter send · Shift+Enter/Ctrl+J newline · Ctrl+C quit",
		mode,
		m.currentScenario,
		m.status,
	)

	return m.styles.Footer.Render(ui.Truncate(text, m.width))
}

func (m *model) renderSeparator() string {
	width := m.width
	if width < 1 {
		width = 1
	}
	return m.styles.Meta.Render(strings.Repeat("─", width))
}
