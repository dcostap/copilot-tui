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

	p.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	p.Send(tea.KeyPressMsg{Code: 'h', Text: "h"})
	p.Send(tea.KeyPressMsg{Code: 'i', Text: "i"})
	p.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	select {
	case result := <-results:
		if result.err != nil {
			t.Fatalf("program exited with error: %v", result.err)
		}

		finalModel, ok := result.model.(*model)
		if !ok {
			t.Fatalf("expected *model, got %T", result.model)
		}

		if got, want := finalModel.input.Value(), "hi"; got != want {
			t.Fatalf("expected typed input %q, got %q", want, got)
		}

	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatal("program did not process startup, typing, and quit before timeout")
		}
		t.Fatalf("program context ended unexpectedly: %v", ctx.Err())
	}
}
