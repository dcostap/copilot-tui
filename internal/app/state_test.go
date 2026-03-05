package app

import (
	"testing"

	"copilot-tui/internal/copilot"
)

func TestConversationStateAssistantStream(t *testing.T) {
	state := NewConversationState()

	state.AddUserPrompt("hello")
	state.ApplyEvent(copilot.Event{Type: copilot.EventAssistantDelta, Text: "Hi"})
	state.ApplyEvent(copilot.Event{Type: copilot.EventAssistantDelta, Text: " there"})
	state.ApplyEvent(copilot.Event{Type: copilot.EventTurnComplete})

	if len(state.Items) != 2 {
		t.Fatalf("expected 2 timeline items, got %d", len(state.Items))
	}
	if state.Items[0].Kind != TimelineUser || state.Items[0].Text != "hello" {
		t.Fatalf("unexpected user item: %#v", state.Items[0])
	}
	if state.Items[1].Kind != TimelineAssistant || state.Items[1].Text != "Hi there" {
		t.Fatalf("unexpected assistant item: %#v", state.Items[1])
	}
}

func TestConversationStateReasoningAndToolEvents(t *testing.T) {
	state := NewConversationState()

	state.ApplyEvent(copilot.Event{Type: copilot.EventReasoningDelta, Text: "Thinking"})
	state.ApplyEvent(copilot.Event{Type: copilot.EventReasoningDelta, Text: " more"})
	state.ApplyEvent(copilot.Event{Type: copilot.EventToolStart, Tool: "search", Text: "starting"})
	state.ApplyEvent(copilot.Event{Type: copilot.EventToolComplete, Tool: "search", Text: "done"})
	state.ApplyEvent(copilot.Event{Type: copilot.EventError, Text: "problem"})

	if len(state.Items) != 4 {
		t.Fatalf("expected 4 timeline items, got %d", len(state.Items))
	}

	if state.Items[0].Kind != TimelineReasoning || state.Items[0].Text != "Thinking more" {
		t.Fatalf("unexpected reasoning item: %#v", state.Items[0])
	}
	if state.Items[1].Kind != TimelineTool || state.Items[1].Tool != "search" || state.Items[1].Text != "starting" {
		t.Fatalf("unexpected tool start item: %#v", state.Items[1])
	}
	if state.Items[2].Kind != TimelineTool || state.Items[2].Tool != "search" || state.Items[2].Text != "done" {
		t.Fatalf("unexpected tool complete item: %#v", state.Items[2])
	}
	if state.Items[3].Kind != TimelineError || state.Items[3].Text != "problem" {
		t.Fatalf("unexpected error item: %#v", state.Items[3])
	}
}
