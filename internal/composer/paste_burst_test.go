package composer

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestPasteBurstFirstCharFlushesAsTyped(t *testing.T) {
	var burst pasteBurst
	t0 := time.Unix(0, 0)

	if decision := burst.onPlainChar('a', t0); decision != pasteDecisionHeld {
		t.Fatalf("expected first char to be held, got %v", decision)
	}

	flushed := burst.flushIfDue(t0.Add(RecommendedPasteFlushDelay() + time.Millisecond))
	if !flushed.ok || flushed.text != "a" {
		t.Fatalf("expected typed flush %q, got %#v", "a", flushed)
	}
}

func TestPasteBurstFastCharsFlushAsBulkText(t *testing.T) {
	var burst pasteBurst
	t0 := time.Unix(0, 0)

	burst.onPlainChar('a', t0)
	if decision := burst.onPlainChar('b', t0.Add(time.Millisecond)); decision != pasteDecisionBuffered {
		t.Fatalf("expected second char to enter buffered mode, got %v", decision)
	}

	flushed := burst.flushIfDue(t0.Add(pasteBurstActiveIdleTimeout() + 2*time.Millisecond))
	if !flushed.ok || flushed.text != "ab" {
		t.Fatalf("expected buffered flush %q, got %#v", "ab", flushed)
	}
}

func TestTextareaFlushesPendingPasteBeforeExternalInput(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	textarea, _ = textarea.Update(keyPress('a'))

	if got := textarea.Value(); got != "" {
		t.Fatalf("expected pending char to remain hidden, got %q", got)
	}
	if !textarea.FlushPasteBurstBeforeExternalInput() {
		t.Fatal("expected pending char to flush")
	}
	if got := textarea.Value(); got != "a" {
		t.Fatalf("expected flushed value %q, got %q", "a", got)
	}
}

func TestTextareaHandlePasteBurstEnterBuffersNewline(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	for _, ch := range []rune("abc") {
		textarea, _ = textarea.Update(keyPress(ch))
	}

	if !textarea.HandlePasteBurstEnter() {
		t.Fatal("expected enter to be consumed by the paste burst")
	}
	if !textarea.FlushPasteBurstBeforeExternalInput() {
		t.Fatal("expected buffered paste burst to flush")
	}
	if got := textarea.Value(); got != "abc\n" {
		t.Fatalf("expected flushed multiline value %q, got %q", "abc\n", got)
	}
}

func TestTextareaModifierOnlyKeyDoesNotBreakPasteBurst(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	for _, ch := range []rune("abc") {
		textarea, _ = textarea.Update(keyPress(ch))
	}

	textarea, _ = textarea.Update(tea.KeyPressMsg{Code: tea.KeyLeftShift, Mod: tea.ModShift})
	if !textarea.HandlePasteBurstEnter() {
		t.Fatal("expected modifier-only key to preserve paste burst state")
	}
	if !textarea.FlushPasteBurstBeforeExternalInput() {
		t.Fatal("expected buffered paste burst to flush")
	}
	if got := textarea.Value(); got != "abc\n" {
		t.Fatalf("expected flushed multiline value %q, got %q", "abc\n", got)
	}
}
