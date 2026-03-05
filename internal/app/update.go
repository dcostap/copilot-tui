package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyLayout()
		m.renderNow()
		return m, nil

	case adapterEventMsg:
		return m, tea.Batch(waitForAdapterEvent(m.events), m.handleAdapterEvent(msg.event))

	case renderTickMsg:
		m.renderScheduled = false
		if m.pendingRender {
			m.pendingRender = false
			m.renderNow()
		}
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.showPalette {
			return m, m.updatePaletteKeys(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			_ = m.adapter.Stop(context.Background())
			return m, tea.Quit

		case "ctrl+p":
			m.showPalette = true
			m.paletteIndex = 0
			m.rebuildPalette()
			return m, nil

		case "enter":
			return m, m.submitPrompt()

		case "shift+enter":
			m.input.InsertString("\n")
			return m, nil

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}
