package app

import (
	"context"
	"errors"
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
