package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

const (
	inputTraceEnv         = "COPILOT_TUI_INPUT_TRACE"
	defaultInputTraceFile = "copilot-tui-input-trace.log"
	tracePreviewLimit     = 80
)

type inputTracer struct {
	mu sync.Mutex

	file    *os.File
	path    string
	started time.Time
	last    time.Time
	seq     int
}

func newInputTracerFromEnv() (*inputTracer, error) {
	raw := strings.TrimSpace(os.Getenv(inputTraceEnv))
	if raw == "" {
		return nil, nil
	}

	path := raw
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		path = defaultInputTraceFile
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve input trace path: %w", err)
	}

	file, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("create input trace file %s: %w", absPath, err)
	}

	now := time.Now()
	tracer := &inputTracer{
		file:    file,
		path:    absPath,
		started: now,
		last:    now,
	}

	if _, err := fmt.Fprintf(file, "# copilot-tui input trace started %s\n", now.Format(time.RFC3339Nano)); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("write input trace header: %w", err)
	}

	fmt.Fprintf(os.Stderr, "copilot-tui input trace: %s\n", absPath)
	return tracer, nil
}

func (t *inputTracer) LogMsg(msg tea.Msg) {
	if t == nil {
		return
	}
	t.write(describeInputMsg(msg))
}

func (t *inputTracer) LogNote(format string, args ...any) {
	if t == nil {
		return
	}
	t.write(fmt.Sprintf(format, args...))
}

func (t *inputTracer) write(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.seq++
	delta := now.Sub(t.last).Truncate(time.Microsecond)
	t.last = now

	if _, err := fmt.Fprintf(t.file, "%05d +%s %s\n", t.seq, delta, line); err != nil {
		fmt.Fprintf(os.Stderr, "copilot-tui input trace write failed: %v\n", err)
	}
}

func describeInputMsg(msg tea.Msg) string {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		return fmt.Sprintf(
			"PasteMsg len=%d runes=%d newlines=%d preview=%q",
			len(msg.Content),
			utf8.RuneCountInString(msg.Content),
			countTraceNewlines(msg.Content),
			tracePreview(msg.Content),
		)
	case tea.KeyPressMsg:
		return fmt.Sprintf(
			"KeyPressMsg key=%q code=%s shifted=%s base=%s mod=%v text=%q textRunes=%d repeat=%t",
			msg.String(),
			traceRune(msg.Code),
			traceRune(msg.ShiftedCode),
			traceRune(msg.BaseCode),
			msg.Mod,
			msg.Text,
			utf8.RuneCountInString(msg.Text),
			msg.IsRepeat,
		)
	default:
		return fmt.Sprintf("%T", msg)
	}
}

func traceRune(r rune) string {
	if r == 0 {
		return "0"
	}
	if r > utf8.MaxRune {
		return fmt.Sprintf("%d", r)
	}
	return fmt.Sprintf("%q/%U", r, r)
}

func tracePreview(s string) string {
	runes := []rune(s)
	if len(runes) <= tracePreviewLimit {
		return s
	}
	return string(runes[:tracePreviewLimit]) + "..."
}

func countTraceNewlines(s string) int {
	n := strings.Count(s, "\n")
	withoutCRLF := strings.ReplaceAll(s, "\r\n", "")
	return n + strings.Count(withoutCRLF, "\r")
}
