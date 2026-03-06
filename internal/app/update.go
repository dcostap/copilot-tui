package app

import (
	"context"

	tea "charm.land/bubbletea/v2"
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

	case tea.KeyPressMsg:
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

		case "shift+enter", "ctrl+j":
			m.input.InsertString("\n")
			return m, nil

		case "ctrl+s":
			return m, m.submitPrompt()

		case "enter":
			return m, m.submitPrompt()

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}
