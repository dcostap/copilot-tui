package copilot

import (
	"context"
	"testing"
	"time"
)

func TestMockAdapterDeterministicScenario(t *testing.T) {
	adapter := NewMockAdapter()
	adapter.SetStreamDelay(0)

	if err := adapter.SetScenario("normal_markdown_stream"); err != nil {
		t.Fatalf("set scenario failed: %v", err)
	}
	if err := adapter.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Stop(stopCtx)
	})

	if err := adapter.SendPrompt(context.Background(), "hello"); err != nil {
		t.Fatalf("first send failed: %v", err)
	}
	first := collectUntilTurnComplete(t, adapter.Events())

	if err := adapter.SendPrompt(context.Background(), "hello"); err != nil {
		t.Fatalf("second send failed: %v", err)
	}
	second := collectUntilTurnComplete(t, adapter.Events())

	if len(first) != len(second) {
		t.Fatalf("event length mismatch: %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i].Type != second[i].Type || first[i].Tool != second[i].Tool || first[i].Text != second[i].Text {
			t.Fatalf("event[%d] mismatch: %#v vs %#v", i, first[i], second[i])
		}
	}
}

func TestMockAdapterSetScenarioValidation(t *testing.T) {
	adapter := NewMockAdapter()
	if err := adapter.SetScenario("does_not_exist"); err == nil {
		t.Fatal("expected error for unknown scenario")
	}
}

func collectUntilTurnComplete(t *testing.T, events <-chan Event) []Event {
	t.Helper()

	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	collected := make([]Event, 0, 8)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatal("event channel closed unexpectedly")
			}
			collected = append(collected, ev)
			if ev.Type == EventTurnComplete {
				return collected
			}
		case <-timeout.C:
			t.Fatal("timed out waiting for turn_complete")
		}
	}
}
