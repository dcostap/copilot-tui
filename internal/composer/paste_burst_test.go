package composer

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestTextareaTypingShowsImmediatelyBeforeBurstActivation(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	textarea, _ = textarea.Update(keyPress('a'))

	if got := textarea.Value(); got != "a" {
		t.Fatalf("expected typed value %q, got %q", "a", got)
	}
	if textarea.FlushPasteBurstBeforeExternalInput() {
		t.Fatal("did not expect visible typed input to require a flush")
	}
}

func TestPasteBurstStartsBufferingAfterRapidThreshold(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	typed := strings.Repeat("a", pasteBurstActivationRunes()+1)
	for _, ch := range []rune(typed) {
		textarea, _ = textarea.Update(keyPress(ch))
	}

	if got := textarea.Value(); got != strings.Repeat("a", pasteBurstActivationRunes()) {
		t.Fatalf("expected rapid burst to keep only the activation prefix visible, got %q", got)
	}
	if !textarea.FlushPasteBurstBeforeExternalInput() {
		t.Fatal("expected buffered tail to flush")
	}
	if got := textarea.Value(); got != typed {
		t.Fatalf("expected flushed burst value %q, got %q", typed, got)
	}
}

func TestTextareaHandlePasteBurstEnterBuffersNewline(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	seed := strings.Repeat("a", pasteBurstActivationRunes())
	for _, ch := range []rune(seed) {
		textarea, _ = textarea.Update(keyPress(ch))
	}

	if !textarea.HandlePasteBurstEnter() {
		t.Fatal("expected enter to be consumed by the paste burst")
	}
	if !textarea.FlushPasteBurstBeforeExternalInput() {
		t.Fatal("expected buffered paste burst to flush")
	}
	if got := textarea.Value(); got != seed+"\n" {
		t.Fatalf("expected flushed multiline value %q, got %q", seed+"\n", got)
	}
}

func TestTextareaModifierOnlyKeyDoesNotBreakPasteBurst(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	seed := strings.Repeat("a", pasteBurstActivationRunes())
	for _, ch := range []rune(seed) {
		textarea, _ = textarea.Update(keyPress(ch))
	}

	textarea, _ = textarea.Update(tea.KeyPressMsg{Code: tea.KeyLeftShift, Mod: tea.ModShift})
	if !textarea.HandlePasteBurstEnter() {
		t.Fatal("expected modifier-only key to preserve paste burst state")
	}
	if !textarea.FlushPasteBurstBeforeExternalInput() {
		t.Fatal("expected buffered paste burst to flush")
	}
	if got := textarea.Value(); got != seed+"\n" {
		t.Fatalf("expected flushed multiline value %q, got %q", seed+"\n", got)
	}
}

func TestPasteBurstKeepsEnterBufferedAcrossReplayGap(t *testing.T) {
	textarea := newTextArea()
	textarea.SetPasteBurstEnabled(true)
	seed := strings.Repeat("a", pasteBurstActivationRunes())
	textarea.SetValue(seed)

	t0 := time.Unix(0, 0)
	textarea.pasteBurst.active = true
	textarea.pasteBurst.hasBurstWindow = true
	textarea.pasteBurst.suppressEnter = true
	textarea.pasteBurst.burstWindowUntil = t0.Add(pasteEnterSuppressWindow())
	textarea.pasteBurst.lastPlainCharTime = t0.Add(2 * time.Millisecond)

	enterAt := t0.Add(pasteBurstActiveIdleTimeout() + 40*time.Millisecond)
	if !textarea.handlePasteBurstEnterAt(enterAt) {
		t.Fatal("expected enter inside the replay window to stay buffered")
	}
	if !textarea.flushPasteBurstIfDueAt(enterAt.Add(pasteBurstActiveIdleTimeout() + 2*time.Millisecond)) {
		t.Fatal("expected buffered replay text to flush")
	}
	if got := textarea.Value(); got != seed+"\n" {
		t.Fatalf("expected buffered multiline replay %q, got %q", seed+"\n", got)
	}
}

func TestPasteBurstFlushDelayMatchesState(t *testing.T) {
	var burst pasteBurst
	t0 := time.Unix(0, 0)

	burst.onPlainChar('a', t0)
	delay, ok := burst.nextFlushDelay(t0)
	if ok || delay != 0 {
		t.Fatalf("expected normal typing state not to schedule a flush, got ok=%v delay=%v", ok, delay)
	}

	for i := 1; i < pasteBurstActivationRunes(); i++ {
		burst.onPlainChar('a', t0.Add(time.Duration(i)*time.Millisecond))
	}
	delay, ok = burst.nextFlushDelay(t0.Add(time.Duration(pasteBurstActivationRunes()-1) * time.Millisecond))
	if !ok {
		t.Fatal("expected active burst to request a flush delay")
	}
	if delay < pasteBurstActiveIdleTimeout()-time.Millisecond || delay > pasteBurstActiveIdleTimeout()+2*time.Millisecond {
		t.Fatalf("expected active-burst delay near %v, got %v", pasteBurstActiveIdleTimeout(), delay)
	}
}
