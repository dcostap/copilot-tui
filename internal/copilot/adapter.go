package copilot

import (
	"context"
	"time"
)

type EventType string

const (
	EventAssistantDelta EventType = "assistant_delta"
	EventReasoningDelta EventType = "reasoning_delta"
	EventToolStart      EventType = "tool_start"
	EventToolProgress   EventType = "tool_progress"
	EventToolComplete   EventType = "tool_complete"
	EventTurnComplete   EventType = "turn_complete"
	EventError          EventType = "error"
)

type Event struct {
	Type    EventType
	Text    string
	Tool    string
	Payload map[string]any
}

type Adapter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	SendPrompt(ctx context.Context, prompt string) error
	Abort(ctx context.Context) error
	Events() <-chan Event
}

type ScenarioController interface {
	Scenarios() []string
	CurrentScenario() string
	SetScenario(name string) error
	SetStreamDelay(delay time.Duration)
}
