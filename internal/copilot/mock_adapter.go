package copilot

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type scenarioStep struct {
	DelayTicks int
	Event      Event
}

type MockAdapter struct {
	mu           sync.RWMutex
	events       chan Event
	scenarios    map[string][]scenarioStep
	scenarioName string
	tickDelay    time.Duration
	runCancel    context.CancelFunc
	runWG        sync.WaitGroup
	closed       bool
}

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		events:       make(chan Event, 128),
		scenarios:    defaultScenarios(),
		scenarioName: "normal_markdown_stream",
		tickDelay:    40 * time.Millisecond,
	}
}

func (m *MockAdapter) Start(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return errors.New("mock adapter is stopped")
	}
	return nil
}

func (m *MockAdapter) Stop(ctx context.Context) error {
	if err := m.Abort(ctx); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		m.runWG.Wait()
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.events)
	return nil
}

func (m *MockAdapter) SendPrompt(ctx context.Context, prompt string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("mock adapter is stopped")
	}

	steps, ok := m.scenarios[m.scenarioName]
	if !ok {
		return fmt.Errorf("unknown scenario %q", m.scenarioName)
	}

	if m.runCancel != nil {
		m.runCancel()
	}

	runCtx, cancel := context.WithCancel(ctx)
	m.runCancel = cancel
	delay := m.tickDelay
	copied := append([]scenarioStep(nil), steps...)

	m.runWG.Add(1)
	go m.runScenario(runCtx, copied, delay, prompt)

	return nil
}

func (m *MockAdapter) Abort(ctx context.Context) error {
	m.mu.Lock()
	cancel := m.runCancel
	m.runCancel = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

func (m *MockAdapter) Events() <-chan Event {
	return m.events
}

func (m *MockAdapter) Scenarios() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.scenarios))
	for name := range m.scenarios {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *MockAdapter) CurrentScenario() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scenarioName
}

func (m *MockAdapter) SetScenario(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.scenarios[name]; !ok {
		return fmt.Errorf("scenario %q does not exist", name)
	}
	m.scenarioName = name
	return nil
}

func (m *MockAdapter) SetStreamDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if delay < 0 {
		return
	}
	m.tickDelay = delay
}

func (m *MockAdapter) runScenario(ctx context.Context, steps []scenarioStep, delay time.Duration, prompt string) {
	defer m.runWG.Done()

	for _, step := range steps {
		wait := time.Duration(step.DelayTicks) * delay
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}

		ev := step.Event
		ev.Text = strings.ReplaceAll(ev.Text, "{{prompt}}", prompt)

		select {
		case m.events <- ev:
		case <-ctx.Done():
			return
		}
	}
}

func defaultScenarios() map[string][]scenarioStep {
	return map[string][]scenarioStep{
		"normal_markdown_stream": {
			{DelayTicks: 1, Event: Event{Type: EventAssistantDelta, Text: "# Mock answer"}},
			{DelayTicks: 1, Event: Event{Type: EventAssistantDelta, Text: "\n\nYou said: `{{prompt}}`."}},
			{DelayTicks: 1, Event: Event{Type: EventAssistantDelta, Text: "\n\nThis is a **streamed markdown** response."}},
			{DelayTicks: 1, Event: Event{Type: EventTurnComplete}},
		},
		"reasoning_then_answer": {
			{DelayTicks: 1, Event: Event{Type: EventReasoningDelta, Text: "Thinking through the prompt structure..."}},
			{DelayTicks: 1, Event: Event{Type: EventReasoningDelta, Text: " mapping it to a concise answer."}},
			{DelayTicks: 1, Event: Event{Type: EventAssistantDelta, Text: "Here is the final response for `{{prompt}}`."}},
			{DelayTicks: 1, Event: Event{Type: EventTurnComplete}},
		},
		"tool_call_success": {
			{DelayTicks: 1, Event: Event{Type: EventToolStart, Tool: "search", Text: "Starting repository search"}},
			{DelayTicks: 1, Event: Event{Type: EventToolProgress, Tool: "search", Text: "Found 3 matching files"}},
			{DelayTicks: 1, Event: Event{Type: EventToolComplete, Tool: "search", Text: "Search completed successfully"}},
			{DelayTicks: 1, Event: Event{Type: EventAssistantDelta, Text: "I found relevant files and summarized the results."}},
			{DelayTicks: 1, Event: Event{Type: EventTurnComplete}},
		},
		"tool_call_failure": {
			{DelayTicks: 1, Event: Event{Type: EventToolStart, Tool: "build", Text: "Starting build command"}},
			{DelayTicks: 1, Event: Event{Type: EventToolComplete, Tool: "build", Text: "Build failed with exit code 1"}},
			{DelayTicks: 1, Event: Event{Type: EventError, Text: "Tool build failed: missing dependency"}},
			{DelayTicks: 1, Event: Event{Type: EventTurnComplete}},
		},
		"slow_token_stream": {
			{DelayTicks: 2, Event: Event{Type: EventAssistantDelta, Text: "This"}},
			{DelayTicks: 2, Event: Event{Type: EventAssistantDelta, Text: " is"}},
			{DelayTicks: 2, Event: Event{Type: EventAssistantDelta, Text: " a"}},
			{DelayTicks: 2, Event: Event{Type: EventAssistantDelta, Text: " slow"}},
			{DelayTicks: 2, Event: Event{Type: EventAssistantDelta, Text: " token"}},
			{DelayTicks: 2, Event: Event{Type: EventAssistantDelta, Text: " stream."}},
			{DelayTicks: 1, Event: Event{Type: EventTurnComplete}},
		},
		"interrupted_stream": {
			{DelayTicks: 1, Event: Event{Type: EventAssistantDelta, Text: "Starting response..."}},
			{DelayTicks: 1, Event: Event{Type: EventError, Text: "Stream interrupted before completion"}},
			{DelayTicks: 1, Event: Event{Type: EventTurnComplete}},
		},
	}
}
