package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"copilot-tui/internal/copilot"
)

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
			want: "Ctrl+J newline",
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
