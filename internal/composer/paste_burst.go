package composer

import (
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

var pasteNewlineNormalizer = strings.NewReplacer("\r\n", "\n", "\r", "\n")

type pasteDecision uint8

const (
	pasteDecisionNone pasteDecision = iota
	pasteDecisionBuffered
)

type pasteFlushResult struct {
	text string
	ok   bool
}

type pasteKeyPressResult struct {
	tracked  bool
	consumed bool
	flushed  bool
}

type pasteBurst struct {
	lastPlainCharTime time.Time
	burstWindowUntil  time.Time
	buffer            []rune
	active            bool
	hasBurstWindow    bool
	suppressEnter     bool
	visibleRunLength  int
}

func RecommendedPasteFlushDelay() time.Duration {
	return pasteBurstActiveIdleTimeout() + time.Millisecond
}

func pasteBurstCharInterval() time.Duration {
	if runtime.GOOS == "windows" {
		return 30 * time.Millisecond
	}
	return 8 * time.Millisecond
}

func pasteBurstActiveIdleTimeout() time.Duration {
	if runtime.GOOS == "windows" {
		return 220 * time.Millisecond
	}
	return 8 * time.Millisecond
}

func pasteBurstActivationRunes() int {
	if runtime.GOOS == "windows" {
		return 12
	}
	return 6
}

func pasteBurstNewlineRunes() int {
	if runtime.GOOS == "windows" {
		return 8
	}
	return 4
}

func pasteEnterSuppressWindow() time.Duration {
	if runtime.GOOS == "windows" {
		return 350 * time.Millisecond
	}
	return 120 * time.Millisecond
}

func (b *pasteBurst) isActive() bool {
	return b.active || len(b.buffer) > 0
}

func (b *pasteBurst) onPlainChar(ch rune, now time.Time) pasteDecision {
	b.expireWindowIfNeeded(now)

	if b.active {
		b.appendCharToBuffer(ch, now)
		return pasteDecisionBuffered
	}

	if b.lastPlainCharTime.IsZero() || now.Sub(b.lastPlainCharTime) > pasteBurstCharInterval() {
		b.visibleRunLength = 1
		b.lastPlainCharTime = now
		return pasteDecisionNone
	}

	b.visibleRunLength++
	b.lastPlainCharTime = now
	if b.visibleRunLength >= pasteBurstActivationRunes() {
		b.active = true
		b.extendWindow(now)
	}

	return pasteDecisionNone
}

func (b *pasteBurst) appendCharToBuffer(ch rune, now time.Time) {
	b.buffer = append(b.buffer, ch)
	b.lastPlainCharTime = now
	b.extendWindow(now)
}

func (b *pasteBurst) appendNewlineIfActive(now time.Time) bool {
	b.expireWindowIfNeeded(now)
	if !b.active && !b.hasBurstWindow {
		if b.lastPlainCharTime.IsZero() || now.Sub(b.lastPlainCharTime) > pasteBurstCharInterval() || b.visibleRunLength < pasteBurstNewlineRunes() {
			return false
		}
	}
	b.active = true
	b.buffer = append(b.buffer, '\n')
	b.lastPlainCharTime = now
	b.extendWindow(now)
	return true
}

func (b *pasteBurst) newlineShouldInsertInsteadOfSubmit(now time.Time) bool {
	return b.suppressEnter && b.hasBurstWindow && !now.After(b.burstWindowUntil)
}

func (b *pasteBurst) extendWindow(now time.Time) {
	b.burstWindowUntil = now.Add(pasteEnterSuppressWindow())
	b.hasBurstWindow = true
	b.suppressEnter = true
}

func (b *pasteBurst) expireWindowIfNeeded(now time.Time) {
	if !b.hasBurstWindow || !now.After(b.burstWindowUntil) {
		return
	}
	b.hasBurstWindow = false
	b.suppressEnter = false
}

func (b *pasteBurst) nextFlushDelay(now time.Time) (time.Duration, bool) {
	if b.active || len(b.buffer) > 0 {
		return remainingPasteDelay(pasteBurstActiveIdleTimeout(), b.lastPlainCharTime, now), true
	}
	return 0, false
}

func remainingPasteDelay(window time.Duration, start, now time.Time) time.Duration {
	if start.IsZero() {
		return time.Millisecond
	}
	delay := window - now.Sub(start)
	if delay < 0 {
		delay = 0
	}
	return delay + time.Millisecond
}

func (b *pasteBurst) flushIfDue(now time.Time) pasteFlushResult {
	b.expireWindowIfNeeded(now)

	if b.active || len(b.buffer) > 0 {
		if b.lastPlainCharTime.IsZero() || now.Sub(b.lastPlainCharTime) <= pasteBurstActiveIdleTimeout() {
			return pasteFlushResult{}
		}
		b.active = false
		b.visibleRunLength = 0
		b.lastPlainCharTime = time.Time{}
		if len(b.buffer) == 0 {
			return pasteFlushResult{}
		}
		text := string(b.buffer)
		b.buffer = b.buffer[:0]
		return pasteFlushResult{text: text, ok: text != ""}
	}

	if !b.lastPlainCharTime.IsZero() && now.Sub(b.lastPlainCharTime) > pasteBurstCharInterval() {
		b.lastPlainCharTime = time.Time{}
		b.visibleRunLength = 0
	}
	return pasteFlushResult{}
}

func (b *pasteBurst) flushBeforeModifiedInput() pasteFlushResult {
	if len(b.buffer) > 0 {
		text := string(b.buffer)
		b.clearAfterExplicitPaste()
		return pasteFlushResult{text: text, ok: text != ""}
	}

	b.clearAfterExplicitPaste()
	return pasteFlushResult{}
}

func (b *pasteBurst) clearAfterExplicitPaste() {
	b.lastPlainCharTime = time.Time{}
	b.burstWindowUntil = time.Time{}
	b.buffer = b.buffer[:0]
	b.active = false
	b.hasBurstWindow = false
	b.suppressEnter = false
	b.visibleRunLength = 0
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

func (m *Model) PasteBurstFlushDelay() (time.Duration, bool) {
	if !m.pasteBurstEnabled {
		return 0, false
	}
	return m.pasteBurst.nextFlushDelay(time.Now())
}

func (m *Model) handlePlainKeyPress(msg tea.KeyPressMsg) pasteKeyPressResult {
	if !m.pasteBurstEnabled {
		return pasteKeyPressResult{}
	}

	ch, ok := trackedPasteRune(msg)
	if !ok {
		return pasteKeyPressResult{}
	}

	now := time.Now()
	flushed := m.applyPasteBurstFlush(m.pasteBurst.flushIfDue(now))

	switch m.pasteBurst.onPlainChar(ch, now) {
	case pasteDecisionBuffered:
		return pasteKeyPressResult{tracked: true, consumed: true, flushed: flushed}
	default:
		return pasteKeyPressResult{tracked: true, flushed: flushed}
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
