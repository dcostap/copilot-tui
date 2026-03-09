package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestDescribeInputMsgPaste(t *testing.T) {
	t.Parallel()

	got := describeInputMsg(tea.PasteMsg{Content: "first line\r\nsecond line"})

	for _, want := range []string{
		"PasteMsg",
		"runes=23",
		"newlines=1",
		`preview="first line`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}

func TestDescribeInputMsgKeyPress(t *testing.T) {
	t.Parallel()

	got := describeInputMsg(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl})

	for _, want := range []string{
		`KeyPressMsg`,
		`key="ctrl+v"`,
		`text=""`,
		`textRunes=0`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}
