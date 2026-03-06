# ARCHITECTURE.md

## Goal

Read IDEA.md 

## Locked stack for v1

- `charm.land/bubbletea/v2`
- `charm.land/bubbles/v2` (`textarea`, `viewport`, `help`, `key`)
- `charm.land/lipgloss/v2`
- `github.com/charmbracelet/glamour`
- `github.com/github/copilot-sdk/go` (behind internal adapter)

Dependency note:
- keep source imports on the upstream `charm.land/...` paths,
- pin the Windows input fix by replacing `charm.land/bubbletea/v2` in `go.mod`
  with `github.com/dcostap/bubbletea/v2`,
- let the Bubble Tea fork pull `github.com/dcostap/ultraviolet` transitively, so
  fresh clones do not need a machine-local `go.work` sibling-repo layout.

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

### Automated tests (primary)
- Focus on **Layer 1 only**:
  - pure unit tests for state reducers / update functions,
  - synthetic event sequences,
  - deterministic edge cases (tiny deltas, abort, empty response, long markdown, etc.).
- Keep tests table-driven and fast.
- Use standard `go test` and `go test -race`.

### Adapter safety checks (minimal)
- Keep a small contract test set to ensure `mockAdapter` and `copilotSDKAdapter`
  follow the same event contract.
- Avoid heavy UI automation unless there is a clear regression need.

### Manual UI validation (preferred for UX)
- Validate Bubble Tea rendering/interaction manually using fake scenarios.
- Prioritize rapid visual iteration over over-engineered UI test harnesses.

### Practical test tooling
- `go test ./...`
- `go test -race ./...`
- scenario-driven manual checks via mock streaming.

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

## Command palette scenario access (manual testing workflow)

Use the command palette as the primary test driver:
- `Load scenario: normal_markdown_stream`
- `Load scenario: tool_call_failure`
- `Load scenario: interrupted_stream`
- `Load scenario: slow_token_stream`

Optional palette actions:
- `List scenarios`
- `Set stream speed`
- `Random scenario (seeded)`

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
  - Mitigation: fast manual scenario runs + focused unit tests on state transitions.

## Final recommendation

Proceed in two phases:

1. **Phase A (offline-first):** ship robust UI on `mockAdapter`.
2. **Phase B (online):** plug in `copilotSDKAdapter` behind the same interface.

This is the safest path to fast iteration, low cost, and high confidence.

## Local developer commands (MVP scaffold)

- `go run ./cmd/copilot-tui`
- `go test ./...`
- `go test -race ./...` (requires `CGO_ENABLED=1` and a C compiler such as `gcc` on Windows)
