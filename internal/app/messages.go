package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"copilot-tui/internal/copilot"
)

type adapterEventMsg struct {
	event copilot.Event
}

type renderTickMsg struct{}
type inputFlushTickMsg struct{}

func waitForAdapterEvent(events <-chan copilot.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return nil
		}
		return adapterEventMsg{event: ev}
	}
}

func renderTickCmd(delay time.Duration) tea.Cmd {
	if delay < 0 {
		delay = 0
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return renderTickMsg{}
	})
}

func inputFlushTickCmd(delay time.Duration) tea.Cmd {
	if delay < 0 {
		delay = 0
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return inputFlushTickMsg{}
	})
}
