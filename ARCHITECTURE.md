# ARCHITECTURE.md

## Goal

Build a Go TUI Copilot client with:
- prompt input at the bottom,
- streaming conversation at the top,
- clean separation between UI logic and provider (Copilot SDK) logic.

For now, image features are explicitly out of scope.

## Locked stack for v1

- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/bubbles` (`textarea`, `viewport`, `help`, `key`)
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/glamour`
- `github.com/github/copilot-sdk/go` (behind internal adapter)

Optional later:
- `github.com/yuin/goldmark` (advanced markdown control),
- `github.com/alecthomas/chroma/v2` (custom code highlighting),
- `github.com/knadh/koanf/v2` (config layering),
- `github.com/spf13/cobra` (larger CLI command surface).

## Core architecture decision: adapter boundary

The UI and app state must not depend directly on Copilot SDK types.

Instead, define an internal provider contract in `internal/copilot/adapter.go`, for example:

```go
type Adapter interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    SendPrompt(ctx context.Context, prompt string) error
    Abort(ctx context.Context) error
    Events() <-chan Event
}
```

With internal event types like:

```go
type EventType string

const (
    EventAssistantDelta   EventType = "assistant_delta"
    EventReasoningDelta   EventType = "reasoning_delta"
    EventToolStart        EventType = "tool_start"
    EventToolComplete     EventType = "tool_complete"
    EventTurnComplete     EventType = "turn_complete"
    EventError            EventType = "error"
)

type Event struct {
    Type    EventType
    Text    string
    Tool    string
    Payload map[string]any
}
```

Implementations:
- `copilotSDKAdapter` (real provider),
- `mockAdapter` (deterministic fake streaming for UI/dev/tests).

This lets us test UI behavior deeply without making real Copilot calls or spending credits.

## UI-first development plan (no Copilot dependency initially)

1. Build the full TUI using `mockAdapter` only.
2. Stream deterministic fake content (lorem, markdown blocks, tool events, errors, retries).
3. Validate:
   - input ergonomics (`textarea` behavior),
   - scroll behavior (`viewport`),
   - message grouping,
   - reasoning/tool event rendering,
   - markdown rendering quality.
4. Integrate real `copilotSDKAdapter` only after UI behavior is stable.

## Testing strategy (good + simple)

### Layer 1: Pure unit tests (fast, deterministic)
- Test state reducers / update functions with synthetic events.
- Test edge cases:
  - many tiny deltas,
  - out-of-order completion events,
  - abort during stream,
  - empty response,
  - very long markdown blocks.

### Layer 2: Model-level Bubble Tea tests
- Drive `Update(msg)` with scripted message sequences.
- Assert model state transitions and key invariants:
  - current input buffer,
  - selected pane/scroll position,
  - transcript entry count,
  - stream-in-progress flags.

### Layer 3: Golden rendering tests (regression safety)
- Render `View()` for representative model states.
- Compare against golden files (stable snapshots).
- Great for catching accidental layout/style regressions.

### Layer 4: Adapter contract tests
- Same test suite must pass for `mockAdapter` and `copilotSDKAdapter`.
- Ensures provider swap does not change app behavior.

### Practical test tooling
- Start with standard `go test` only.
- Add `-race` in CI early.
- Add table-driven tests for event sequences.
- Keep tests deterministic (fixed timers/seeded pseudo-random streams).

## Mock streaming design (essential)

`mockAdapter` should support scenario presets:
- `normal_markdown_stream`
- `reasoning_then_answer`
- `tool_call_success`
- `tool_call_failure`
- `slow_token_stream`
- `interrupted_stream`

Each scenario emits timed events over the same channel contract as real adapter.
This gives realistic UX testing while remaining fully offline.

## Markdown/input specifics

- Main prompt: `bubbles/textarea` (not `textinput`).
- Enter submits when not in a modal; Shift+Enter inserts newline.
- For markdown:
  - keep raw text buffer per message,
  - throttle re-render during deltas,
  - always do final full re-render at turn completion.

## Risks and mitigations

- **Copilot SDK technical preview changes**
  - Mitigation: adapter boundary + version pinning.

- **Streaming markdown edge cases**
  - Mitigation: buffered rendering + deterministic scenario tests.

- **UI regressions as features grow**
  - Mitigation: golden rendering tests + model-state invariants.

## Final recommendation

Proceed in two phases:

1. **Phase A (offline-first):** ship robust UI on `mockAdapter`.
2. **Phase B (online):** plug in `copilotSDKAdapter` behind the same interface.

This is the safest path to fast iteration, low cost, and high confidence.

## Implementation roadmap (practical, iterative)

### Milestone 0 - Project skeleton
- Create package layout:
  - `cmd/copilot-tui/`
  - `internal/app/` (model/update/view)
  - `internal/copilot/` (adapter contracts + impls)
  - `internal/render/` (markdown/tool block rendering)
  - `internal/testutil/` (event fixtures, test helpers)
- Add basic app bootstrap with a static placeholder UI.
- Exit criteria: app starts, handles resize, exits cleanly.

### Milestone 1 - Input + transcript shell (offline)
- Implement `textarea` prompt input and `viewport` transcript pane.
- Add submit/newline behavior and basic keymap/help.
- Add message list model and scrolling behavior.
- Exit criteria: manual prompt submission produces local echo in transcript.

### Milestone 2 - Mock streaming adapter
- Implement `mockAdapter` and scenario emitter.
- Stream synthetic events into app state:
  - message deltas,
  - reasoning deltas,
  - tool start/complete,
  - terminal errors.
- Exit criteria: realistic streaming conversation works without Copilot.

### Milestone 3 - Rendering quality + markdown
- Integrate Glamour rendering for assistant content.
- Add throttled incremental rerender + final-pass rerender.
- Style conversation blocks and tool timeline entries with Lip Gloss.
- Exit criteria: stable markdown rendering under long streaming outputs.

### Milestone 4 - Regression test baseline
- Add table-driven model tests for event/state transitions.
- Add golden tests for `View()` output on representative states.
- Add race checks (`go test -race`) in CI/local scripts.
- Exit criteria: deterministic tests catch regressions in state and layout.

### Milestone 5 - Real Copilot adapter integration
- Implement `copilotSDKAdapter` behind the same `Adapter` interface.
- Map SDK events into internal event model.
- Keep mock + real adapter contract tests identical.
- Exit criteria: switch provider via config flag/env without UI code changes.

### Milestone 6 - Hardening and polish
- Better error surfaces and reconnect/abort behavior.
- Persist/restore local transcript history if needed.
- Optional config layer and CLI command expansion.
- Exit criteria: stable daily-driver workflow.

