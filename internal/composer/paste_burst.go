package composer

import (
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const pasteEnterSuppressWindow = 120 * time.Millisecond
const pasteBurstMinRunes = 3

var pasteNewlineNormalizer = strings.NewReplacer("\r\n", "\n", "\r", "\n")

type pasteDecision uint8

const (
	pasteDecisionNone pasteDecision = iota
	pasteDecisionHeld
	pasteDecisionBuffered
)

type pasteFlushResult struct {
	text string
	ok   bool
}

type pasteBurst struct {
	lastPlainCharTime  time.Time
	burstWindowUntil   time.Time
	pendingFirstChar   rune
	pendingFirstCharAt time.Time
	buffer             []rune
	active             bool
	hasBurstWindow     bool
	hasPendingFirst    bool
	suppressEnter      bool
}

func RecommendedPasteFlushDelay() time.Duration {
	return pasteBurstCharInterval() + time.Millisecond
}

func pasteBurstCharInterval() time.Duration {
	if runtime.GOOS == "windows" {
		return 30 * time.Millisecond
	}
	return 8 * time.Millisecond
}

func pasteBurstActiveIdleTimeout() time.Duration {
	if runtime.GOOS == "windows" {
		return 60 * time.Millisecond
	}
	return 8 * time.Millisecond
}

func (b *pasteBurst) isActive() bool {
	return b.active || len(b.buffer) > 0 || b.hasPendingFirst
}

func (b *pasteBurst) onPlainChar(ch rune, now time.Time) pasteDecision {
	if b.active || len(b.buffer) > 0 {
		b.appendCharToBuffer(ch, now)
		return pasteDecisionBuffered
	}

	if b.hasPendingFirst && now.Sub(b.pendingFirstCharAt) <= pasteBurstCharInterval() {
		b.active = true
		b.buffer = append(b.buffer, b.pendingFirstChar)
		b.hasPendingFirst = false
		b.appendCharToBuffer(ch, now)
		return pasteDecisionBuffered
	}

	b.pendingFirstChar = ch
	b.pendingFirstCharAt = now
	b.lastPlainCharTime = now
	b.hasPendingFirst = true
	return pasteDecisionHeld
}

func (b *pasteBurst) appendCharToBuffer(ch rune, now time.Time) {
	b.buffer = append(b.buffer, ch)
	b.lastPlainCharTime = now
	if len(b.buffer) >= pasteBurstMinRunes {
		b.extendWindow(now)
	}
}

func (b *pasteBurst) appendNewlineIfActive(now time.Time) bool {
	if (!b.active && len(b.buffer) == 0) || len(b.buffer) < pasteBurstMinRunes {
		return false
	}
	b.buffer = append(b.buffer, '\n')
	b.lastPlainCharTime = now
	b.extendWindow(now)
	return true
}

func (b *pasteBurst) newlineShouldInsertInsteadOfSubmit(now time.Time) bool {
	return b.suppressEnter && b.hasBurstWindow && !now.After(b.burstWindowUntil)
}

func (b *pasteBurst) extendWindow(now time.Time) {
	b.burstWindowUntil = now.Add(pasteEnterSuppressWindow)
	b.hasBurstWindow = true
	b.suppressEnter = true
}

func (b *pasteBurst) flushIfDue(now time.Time) pasteFlushResult {
	if b.active || len(b.buffer) > 0 {
		if b.lastPlainCharTime.IsZero() || now.Sub(b.lastPlainCharTime) <= pasteBurstActiveIdleTimeout() {
			return pasteFlushResult{}
		}
		text := string(b.buffer)
		b.buffer = b.buffer[:0]
		b.active = false
		b.lastPlainCharTime = time.Time{}
		return pasteFlushResult{text: text, ok: text != ""}
	}

	if !b.hasPendingFirst || now.Sub(b.pendingFirstCharAt) <= pasteBurstCharInterval() {
		return pasteFlushResult{}
	}

	text := string(b.pendingFirstChar)
	b.pendingFirstChar = 0
	b.pendingFirstCharAt = time.Time{}
	b.hasPendingFirst = false
	b.lastPlainCharTime = time.Time{}
	return pasteFlushResult{text: text, ok: text != ""}
}

func (b *pasteBurst) flushBeforeModifiedInput() pasteFlushResult {
	if len(b.buffer) > 0 {
		text := string(b.buffer)
		b.clearAfterExplicitPaste()
		return pasteFlushResult{text: text, ok: text != ""}
	}

	if !b.hasPendingFirst {
		b.clearAfterExplicitPaste()
		return pasteFlushResult{}
	}

	text := string(b.pendingFirstChar)
	b.clearAfterExplicitPaste()
	return pasteFlushResult{text: text, ok: text != ""}
}

func (b *pasteBurst) clearAfterExplicitPaste() {
	b.lastPlainCharTime = time.Time{}
	b.burstWindowUntil = time.Time{}
	b.pendingFirstChar = 0
	b.pendingFirstCharAt = time.Time{}
	b.buffer = b.buffer[:0]
	b.active = false
	b.hasBurstWindow = false
	b.hasPendingFirst = false
	b.suppressEnter = false
}

func (m *Model) IsPasteBurstActive() bool {
	if !m.pasteBurstEnabled {
		return false
	}
	return m.pasteBurst.isActive()
}

func (m *Model) FlushPasteBurstIfDue() bool {
	if !m.pasteBurstEnabled {
		return false
	}
	return m.flushPasteBurstIfDueAt(time.Now())
}

func (m *Model) flushPasteBurstIfDueAt(now time.Time) bool {
	return m.applyPasteBurstFlush(m.pasteBurst.flushIfDue(now))
}

func (m *Model) FlushPasteBurstBeforeExternalInput() bool {
	if !m.pasteBurstEnabled {
		return false
	}
	return m.applyPasteBurstFlush(m.pasteBurst.flushBeforeModifiedInput())
}

func (m *Model) HandlePasteBurstEnter() bool {
	if !m.pasteBurstEnabled {
		return false
	}
	return m.handlePasteBurstEnterAt(time.Now())
}

func (m *Model) handlePasteBurstEnterAt(now time.Time) bool {
	if m.pasteBurst.appendNewlineIfActive(now) {
		return true
	}
	if !m.pasteBurst.newlineShouldInsertInsteadOfSubmit(now) {
		return false
	}
	m.insertRunesFromUserInput([]rune{'\n'})
	m.pasteBurst.extendWindow(now)
	return true
}

func (m *Model) handlePlainKeyPress(msg tea.KeyPressMsg) bool {
	if !m.pasteBurstEnabled {
		return false
	}

	ch, ok := trackedPasteRune(msg)
	if !ok {
		return false
	}

	now := time.Now()
	m.applyPasteBurstFlush(m.pasteBurst.flushIfDue(now))

	switch m.pasteBurst.onPlainChar(ch, now) {
	case pasteDecisionHeld, pasteDecisionBuffered:
		return true
	default:
		return false
	}
}

func (m *Model) handleExplicitPaste(content string) {
	m.insertRunesFromUserInput([]rune(pasteNewlineNormalizer.Replace(content)))
	if m.pasteBurstEnabled {
		m.pasteBurst.clearAfterExplicitPaste()
	}
}

func (m *Model) applyPasteBurstFlush(result pasteFlushResult) bool {
	if !result.ok {
		return false
	}
	m.insertRunesFromUserInput([]rune(result.text))
	return true
}

func trackedPasteRune(msg tea.KeyPressMsg) (rune, bool) {
	if len(msg.Text) == 0 || msg.Mod&^tea.ModShift != 0 {
		return 0, false
	}

	runes := []rune(msg.Text)
	if len(runes) != 1 {
		return 0, false
	}

	return runes[0], true
}

func isModifierOnlyKeyPress(msg tea.KeyPressMsg) bool {
	if len(msg.Text) != 0 {
		return false
	}

	switch msg.Code {
	case tea.KeyLeftShift, tea.KeyRightShift,
		tea.KeyLeftCtrl, tea.KeyRightCtrl,
		tea.KeyLeftAlt, tea.KeyRightAlt,
		tea.KeyLeftSuper, tea.KeyRightSuper,
		tea.KeyLeftHyper, tea.KeyRightHyper,
		tea.KeyLeftMeta, tea.KeyRightMeta,
		tea.KeyIsoLevel3Shift, tea.KeyIsoLevel5Shift:
		return true
	default:
		return false
	}
}

func (m *Model) SetPasteBurstEnabled(enabled bool) {
	m.pasteBurstEnabled = enabled
	if !enabled {
		m.pasteBurst.clearAfterExplicitPaste()
	}
}
