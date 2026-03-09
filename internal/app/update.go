package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.tracer.LogMsg(msg)

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

	case inputFlushTickMsg:
		m.inputFlushTick = false
		m.input.FlushPasteBurstIfDue()
		return m, m.scheduleInputFlushTick(nil)

	case tea.KeyboardEnhancementsMsg:
		m.useShiftEnter = msg.SupportsKeyDisambiguation()
		m.renderNow()
		return m, nil

	case tea.PasteMsg:
		if m.showPalette {
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, m.scheduleInputFlushTick(cmd)

	case tea.KeyPressMsg:
		if m.showPalette {
			m.input.FlushPasteBurstBeforeExternalInput()
			return m, m.updatePaletteKeys(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			m.input.FlushPasteBurstBeforeExternalInput()
			_ = m.adapter.Stop(context.Background())
			return m, tea.Quit

		case "ctrl+p":
			m.input.FlushPasteBurstBeforeExternalInput()
			m.showPalette = true
			m.paletteIndex = 0
			m.rebuildPalette()
			return m, nil

		case "shift+enter", "ctrl+j":
			m.input.FlushPasteBurstBeforeExternalInput()
			m.input.InsertString("\n")
			return m, m.scheduleInputFlushTick(nil)

		case "ctrl+s":
			m.input.FlushPasteBurstBeforeExternalInput()
			return m, m.submitPrompt()

		case "enter":
			if m.input.HandlePasteBurstEnter() {
				m.tracer.LogNote("enter suppressed by paste-burst handling")
				return m, m.scheduleInputFlushTick(nil)
			}
			m.input.FlushPasteBurstBeforeExternalInput()
			return m, m.submitPrompt()

		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, m.scheduleInputFlushTick(cmd)
		}
	}

	return m, nil
}

func (m *model) scheduleInputFlushTick(cmd tea.Cmd) tea.Cmd {
	delay, ok := m.input.PasteBurstFlushDelay()
	if !ok {
		m.inputFlushTick = false
		return cmd
	}
	if m.inputFlushTick {
		return cmd
	}

	m.inputFlushTick = true
	m.tracer.LogNote("schedule input flush tick delay=%s", delay.Truncate(time.Microsecond))
	tick := inputFlushTickCmd(delay)
	if cmd == nil {
		return tick
	}
	return tea.Batch(cmd, tick)
}
