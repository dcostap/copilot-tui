package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"copilot-tui/internal/copilot"
)

func TestProgramStartupTypingAndQuit(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'h', Text: "h"},
		tea.KeyPressMsg{Code: 'i', Text: "i"},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got, want := finalModel.input.Value(), "hi"; got != want {
		t.Fatalf("expected typed input %q, got %q", want, got)
	}
}

func TestProgramSubmitAndQuitWhileStreaming(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'h', Text: "h"},
		tea.KeyPressMsg{Code: 'i', Text: "i"},
		tea.KeyPressMsg{Code: tea.KeyEnter},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got := finalModel.input.Value(); got != "" {
		t.Fatalf("expected input to be cleared after submit, got %q", got)
	}
	if len(finalModel.state.Items) == 0 || finalModel.state.Items[0].Kind != TimelineUser || finalModel.state.Items[0].Text != "hi" {
		t.Fatalf("expected submitted user prompt to be recorded, got %#v", finalModel.state.Items)
	}
}

func TestProgramShiftEnterInsertsNewlineWithoutSubmitting(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'h', Text: "h"},
		tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift},
		tea.KeyPressMsg{Code: 'i', Text: "i"},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got, want := finalModel.input.Value(), "h\ni"; got != want {
		t.Fatalf("expected shift+enter to preserve %q, got %q", want, got)
	}
	if len(finalModel.state.Items) != 0 {
		t.Fatalf("expected shift+enter not to submit, got timeline %#v", finalModel.state.Items)
	}
}

func TestProgramCtrlJDoesNotInsertNewline(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'h', Text: "h"},
		tea.KeyPressMsg{Code: 'i', Text: "i"},
		tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got, want := finalModel.input.Value(), "hi"; got != want {
		t.Fatalf("expected ctrl+j to leave %q unchanged, got %q", want, got)
	}
	if len(finalModel.state.Items) != 0 {
		t.Fatalf("expected ctrl+j not to submit, got timeline %#v", finalModel.state.Items)
	}
}

func TestProgramSelectionReplaceAndQuit(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: 'n', Text: "n"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: ' ', Text: " "},
		tea.KeyPressMsg{Code: 't', Text: "t"},
		tea.KeyPressMsg{Code: 'w', Text: "w"},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: ' ', Text: " "},
		tea.KeyPressMsg{Code: 't', Text: "t"},
		tea.KeyPressMsg{Code: 'h', Text: "h"},
		tea.KeyPressMsg{Code: 'r', Text: "r"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl | tea.ModShift},
		tea.KeyPressMsg{Code: 'X', Text: "X"},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got, want := finalModel.input.Value(), "one two X"; got != want {
		t.Fatalf("expected selection replacement %q, got %q", want, got)
	}
}

func TestProgramSelectionMotionIsNonDestructive(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: 'n', Text: "n"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: ' ', Text: " "},
		tea.KeyPressMsg{Code: 't', Text: "t"},
		tea.KeyPressMsg{Code: 'w', Text: "w"},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: ' ', Text: " "},
		tea.KeyPressMsg{Code: 't', Text: "t"},
		tea.KeyPressMsg{Code: 'h', Text: "h"},
		tea.KeyPressMsg{Code: 'r', Text: "r"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl | tea.ModShift},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got, want := finalModel.input.Value(), "one two three"; got != want {
		t.Fatalf("expected selection motion to preserve %q, got %q", want, got)
	}
}

func TestProgramCtrlShiftHomeReplacesToInputStart(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: 'n', Text: "n"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift},
		tea.KeyPressMsg{Code: 't', Text: "t"},
		tea.KeyPressMsg{Code: 'w', Text: "w"},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl | tea.ModShift},
		tea.KeyPressMsg{Code: 'X', Text: "X"},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got, want := finalModel.input.Value(), "X"; got != want {
		t.Fatalf("expected ctrl+shift+home replacement %q, got %q", want, got)
	}
}

func TestProgramCtrlShiftEndReplacesToInputEnd(t *testing.T) {
	t.Parallel()

	finalModel := runProgramScript(t,
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: 'n', Text: "n"},
		tea.KeyPressMsg{Code: 'e', Text: "e"},
		tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift},
		tea.KeyPressMsg{Code: 't', Text: "t"},
		tea.KeyPressMsg{Code: 'w', Text: "w"},
		tea.KeyPressMsg{Code: 'o', Text: "o"},
		tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModCtrl},
		tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModCtrl | tea.ModShift},
		tea.KeyPressMsg{Code: 'X', Text: "X"},
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
	)

	if got, want := finalModel.input.Value(), "X"; got != want {
		t.Fatalf("expected ctrl+shift+end replacement %q, got %q", want, got)
	}
}

func TestProgramRawInputTypingAndQuit(t *testing.T) {
	t.Parallel()

	finalModel := runProgramRawInput(t, []byte("hi\x03"))

	if got, want := finalModel.input.Value(), "hi"; got != want {
		t.Fatalf("expected typed input %q, got %q", want, got)
	}
}

func TestProgramRawInputSelectionReplace(t *testing.T) {
	t.Parallel()

	input := append([]byte("one two three"), []byte("\x1b[1;6DX\x03")...)
	finalModel := runProgramRawInput(t, input)

	if got, want := finalModel.input.Value(), "one two X"; got != want {
		t.Fatalf("expected raw-input selection replacement %q, got %q", want, got)
	}
}

func TestProgramRawInputWin32SelectionReplace(t *testing.T) {
	t.Parallel()

	input := append([]byte("one two three"), []byte(win32KeyPress(37, 0x0118)+"X\x03")...)
	finalModel := runProgramRawInput(t, input)

	if got, want := finalModel.input.Value(), "one two X"; got != want {
		t.Fatalf("expected win32 raw-input selection replacement %q, got %q", want, got)
	}
}

func TestProgramRawInputWin32HomeEndSelectionReplace(t *testing.T) {
	t.Parallel()

	input := append([]byte("abc"), []byte(win32KeyPress(36, 0x0110)+"X\x03")...)
	finalModel := runProgramRawInput(t, input)

	if got, want := finalModel.input.Value(), "X"; got != want {
		t.Fatalf("expected win32 shift+home replacement %q, got %q", want, got)
	}
}

func TestProgramRawInputWin32CtrlShiftHomeReplacesWholeInput(t *testing.T) {
	t.Parallel()

	input := []byte("one" + win32KeyPress(13, 0x0010) + "two" + win32KeyPress(36, 0x0118) + "X\x03")
	finalModel := runProgramRawInput(t, input)

	if got, want := finalModel.input.Value(), "X"; got != want {
		t.Fatalf("expected win32 ctrl+shift+home replacement %q, got %q", want, got)
	}
}

func TestProgramRawInputWin32CtrlShiftEndReplacesWholeInput(t *testing.T) {
	t.Parallel()

	input := []byte("one" + win32KeyPress(13, 0x0010) + "two" + win32KeyPress(36, 0x0008) + win32KeyPress(35, 0x0118) + "X\x03")
	finalModel := runProgramRawInput(t, input)

	if got, want := finalModel.input.Value(), "X"; got != want {
		t.Fatalf("expected win32 ctrl+shift+end replacement %q, got %q", want, got)
	}
}

func TestProgramRawInputWin32WordLeftAtStartDoesNotHang(t *testing.T) {
	t.Parallel()

	finalModel := runProgramRawInput(t, []byte(win32KeyPress(37, 0x0108)+"\x03"))

	if got := finalModel.input.Value(); got != "" {
		t.Fatalf("expected empty input to stay empty, got %q", got)
	}
}

func TestProgramRawBracketedPasteWithNewlinesDoesNotSubmit(t *testing.T) {
	t.Parallel()

	input := []byte("\x1b[200~first line\nsecond line\x1b[201~\x03")
	finalModel := runProgramRawInput(t, input)

	if got, want := finalModel.input.Value(), "first line\nsecond line"; got != want {
		t.Fatalf("expected bracketed paste to preserve %q, got %q", want, got)
	}
	if len(finalModel.state.Items) != 0 {
		t.Fatalf("expected paste not to submit, got timeline %#v", finalModel.state.Items)
	}
}

func TestProgramRawInputWin32PasteBurstNewlineDoesNotSubmit(t *testing.T) {
	t.Parallel()

	input := []byte("first line" + win32KeyPress(13, 0) + "second line\x03")
	finalModel := runProgramRawInput(t, input)

	if got, want := finalModel.input.Value(), "first line\nsecond line"; got != want {
		t.Fatalf("expected win32 paste burst text %q, got %q", want, got)
	}
	if len(finalModel.state.Items) != 0 {
		t.Fatalf("expected win32 paste burst not to submit, got timeline %#v", finalModel.state.Items)
	}
}

func runProgramScript(t *testing.T, msgs ...tea.Msg) *model {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	m := newModel(copilot.NewMockAdapter())
	p := tea.NewProgram(
		m,
		tea.WithContext(ctx),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
	)

	type runResult struct {
		model tea.Model
		err   error
	}

	results := make(chan runResult, 1)
	go func() {
		finalModel, err := p.Run()
		results <- runResult{model: finalModel, err: err}
	}()

	for _, msg := range msgs {
		p.Send(msg)
	}

	select {
	case result := <-results:
		if result.err != nil {
			t.Fatalf("program exited with error: %v", result.err)
		}

		finalModel, ok := result.model.(*model)
		if !ok {
			t.Fatalf("expected *model, got %T", result.model)
		}

		return finalModel
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatal("program did not process smoke-test script before timeout")
		}
		t.Fatalf("program context ended unexpectedly: %v", ctx.Err())
	}

	return nil
}

func runProgramRawInput(t *testing.T, input []byte) *model {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	m := newModel(copilot.NewMockAdapter())
	p := tea.NewProgram(
		m,
		tea.WithContext(ctx),
		tea.WithInput(bytes.NewReader(input)),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignalHandler(),
	)

	type runResult struct {
		model tea.Model
		err   error
	}

	results := make(chan runResult, 1)
	go func() {
		finalModel, err := p.Run()
		results <- runResult{model: finalModel, err: err}
	}()

	p.Send(tea.WindowSizeMsg{Width: 80, Height: 24})

	select {
	case result := <-results:
		if result.err != nil {
			t.Fatalf("program exited with error: %v", result.err)
		}

		finalModel, ok := result.model.(*model)
		if !ok {
			t.Fatalf("expected *model, got %T", result.model)
		}

		return finalModel
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatal("program did not process raw-input script before timeout")
		}
		t.Fatalf("program context ended unexpectedly: %v", ctx.Err())
	}

	return nil
}

func win32KeyPress(vk uint16, controlKeyState uint32) string {
	return fmt.Sprintf("\x1b[%d;0;0;1;%d;1_", vk, controlKeyState)
}
