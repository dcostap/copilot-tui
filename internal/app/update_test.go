package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"copilot-tui/internal/copilot"
)

func TestApplyLayoutUsesContentDrivenInputHeight(t *testing.T) {
	t.Parallel()

	m := newModel(copilot.NewMockAdapter())
	m.width = 80
	m.height = 24

	m.applyLayout()
	if got := m.input.Height(); got != 1 {
		t.Fatalf("expected empty input height 1, got %d", got)
	}

	m.input.SetValue("one\ntwo\nthree")
	m.applyLayout()
	if got := m.input.Height(); got != 3 {
		t.Fatalf("expected multiline input height 3, got %d", got)
	}

	m.input.SetValue("one")
	m.applyLayout()
	if got := m.input.Height(); got != 1 {
		t.Fatalf("expected input height to shrink back to 1, got %d", got)
	}
}

func TestMouseWheelScrollsTimelineOnly(t *testing.T) {
	t.Parallel()

	m := newModel(copilot.NewMockAdapter())
	m.width = 40
	m.height = 8
	for i := 0; i < 20; i++ {
		m.state.AddUserPrompt(strings.Repeat("line ", 8))
	}
	m.input.SetValue("one\ntwo\nthree")
	m.renderNow()

	beforeTimeline := m.viewport.YOffset()
	if beforeTimeline == 0 {
		t.Fatal("expected timeline to start scrollable at the bottom")
	}
	beforeInput := m.input.ScrollYOffset()

	next, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	updated, ok := next.(*model)
	if !ok {
		t.Fatalf("expected *model, got %T", next)
	}

	if got := updated.viewport.YOffset(); got >= beforeTimeline {
		t.Fatalf("expected mouse wheel to scroll the timeline up, before=%d after=%d", beforeTimeline, got)
	}
	if got := updated.input.ScrollYOffset(); got != beforeInput {
		t.Fatalf("expected mouse wheel to leave input scroll unchanged, before=%d after=%d", beforeInput, got)
	}
}

func TestKeyboardEnhancementsSwitchNewlineHint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  tea.KeyboardEnhancementsMsg
		want string
	}{
		{
			name: "fallback newline hint",
			msg:  tea.KeyboardEnhancementsMsg{},
			want: "Shift+Enter newline",
		},
		{
			name: "shift enter newline hint",
			msg:  tea.KeyboardEnhancementsMsg{Flags: 1},
			want: "Shift+Enter newline",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(copilot.NewMockAdapter())
			m.width = 200

			next, _ := m.Update(tc.msg)
			updated, ok := next.(*model)
			if !ok {
				t.Fatalf("expected *model, got %T", next)
			}

			if got := updated.renderFooter(); !strings.Contains(got, tc.want) {
				t.Fatalf("expected footer to contain %q, got %q", tc.want, got)
			}
		})
	}
}
