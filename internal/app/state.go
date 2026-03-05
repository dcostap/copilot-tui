package app

import "copilot-tui/internal/copilot"

type TimelineKind string

const (
	TimelineUser      TimelineKind = "user"
	TimelineAssistant TimelineKind = "assistant"
	TimelineReasoning TimelineKind = "reasoning"
	TimelineTool      TimelineKind = "tool"
	TimelineError     TimelineKind = "error"
)

type TimelineItem struct {
	Kind TimelineKind
	Tool string
	Text string
}

type ConversationState struct {
	Items           []TimelineItem
	activeAssistant int
	activeReasoning int
}

func NewConversationState() ConversationState {
	return ConversationState{
		Items:           []TimelineItem{},
		activeAssistant: -1,
		activeReasoning: -1,
	}
}

func (s *ConversationState) AddUserPrompt(prompt string) {
	s.clearActive()
	s.Items = append(s.Items, TimelineItem{
		Kind: TimelineUser,
		Text: prompt,
	})
}

func (s *ConversationState) ApplyEvent(ev copilot.Event) {
	switch ev.Type {
	case copilot.EventAssistantDelta:
		s.activeReasoning = -1
		idx := s.ensureActive(TimelineAssistant, &s.activeAssistant)
		s.Items[idx].Text += ev.Text

	case copilot.EventReasoningDelta:
		s.activeAssistant = -1
		idx := s.ensureActive(TimelineReasoning, &s.activeReasoning)
		s.Items[idx].Text += ev.Text

	case copilot.EventToolStart, copilot.EventToolProgress, copilot.EventToolComplete:
		s.clearActive()
		s.Items = append(s.Items, TimelineItem{
			Kind: TimelineTool,
			Tool: ev.Tool,
			Text: ev.Text,
		})

	case copilot.EventError:
		s.clearActive()
		s.Items = append(s.Items, TimelineItem{
			Kind: TimelineError,
			Text: ev.Text,
		})

	case copilot.EventTurnComplete:
		s.clearActive()
	}
}

func (s *ConversationState) ensureActive(kind TimelineKind, current *int) int {
	if *current >= 0 && *current < len(s.Items) && s.Items[*current].Kind == kind {
		return *current
	}

	s.Items = append(s.Items, TimelineItem{
		Kind: kind,
	})
	*current = len(s.Items) - 1
	return *current
}

func (s *ConversationState) clearActive() {
	s.activeAssistant = -1
	s.activeReasoning = -1
}
