# WIP paste-handling status

This file is a handoff note for the current paste-handling work. The goal is to make this TUI behave like Codex CLI for multiline paste, large paste collapse, and editing of collapsed paste placeholders.

## Current repo state

- Latest pre-note WIP commit: `674b249` (`WIP harden composer paste handling`)
- Branch: `main`
- Validation that passed before this note:
  - `GOWORK=off go test ./...`
  - `go build ./...`

## What was learned from Codex CLI

The key lesson from `openai/codex` is that paste is handled as a **composer problem**, not an app-level shortcut problem.

Codex has two paste paths that both end in the same composer-owned integration path:

1. **Real explicit paste**
   - `crossterm` surfaces `Event::Paste(...)`
   - that becomes one bulk composer paste operation
   - newlines are normalized
   - large pasted content can be replaced by a placeholder/token and expanded later on submit

2. **Fallback terminal paste**
   - for terminals that replay paste as many normal key events, Codex uses a composer-local `PasteBurst` state machine
   - that state machine:
     - briefly holds the first fast char to avoid flicker
     - groups fast plain chars into one buffered paste
     - treats Enter as newline during burst context
     - keeps a short Enter-suppression window after burst activity
     - supports retro-capture for cases where text was initially inserted normally, then later reclassified as paste-like

Codex also has a second important structural idea we have **not** finished porting:

- **large-paste placeholders are atomic UI elements**
  - they render as one token
  - they are deleted atomically
  - they expand to real content on submit

That atomic token model is still the main missing piece in this repo.

## What has already been changed here

### Earlier stabilizations

- Owned composer fork added under `internal/composer`
- Selection behavior added and tested
- Modifier-only key events no longer delete selection
- `Ctrl+Left` start-of-buffer infinite loop fixed
- `Shift+Home`, `Shift+End`, `Ctrl+Shift+Home`, `Ctrl+Shift+End` selection behavior added
- App smoke tests extended to raw VT + raw Win32 input paths

### Paste-related work already completed

- Removed the earlier app-level paste heuristics after they proved unreliable in the real terminal
- Kept the straightforward `tea.PasteMsg` forwarding path in `internal/app/update.go`
- Added a first composer-local paste-burst implementation in:
  - `internal/composer/paste_burst.go`
  - `internal/composer/paste_burst_test.go`
- The app now only:
  - forwards explicit `tea.PasteMsg`
  - schedules micro-flush ticks while the composer burst detector is active
  - asks the composer whether Enter should become newline during burst context

### Most recent bug solved

The fresh runtime trace showed that live pasted Win32 bursts include **modifier-only key events**, for example bare shift presses/releases such as `leftshift`.

That mattered because the first composer burst implementation was flushing its paste context on any non-plain key, so a modifier-only event could break the active burst and allow the next pasted Enter to submit.

That specific issue was fixed by hardening the composer so **modifier-only keypresses no longer break burst grouping**.

Relevant files:

- `internal/composer/textarea.go`
- `internal/composer/paste_burst.go`
- `internal/composer/paste_burst_test.go`

## What is still broken

The user still reports that live paste is **still buggy** and can still submit unexpectedly.

So the current state is:

- we have improved the composer-side fallback burst handling
- we have fixed one real live-trace bug (modifier-only events breaking burst grouping)
- but the overall live terminal behavior still does **not** match Codex yet

## Fresh live-trace findings

The current trace artifact is here:

- `C:\Users\Dario Costa\.copilot\session-state\23a93493-509d-4f61-bce5-03861da6255a\files\tea-trace.log`

Important findings from the fresh trace:

- still **no bracketed paste markers** (`\x1b[200~` / `\x1b[201~`)
- paste is still coming in as **huge raw Win32 key-event replay blocks**
- Enter appears as ordinary key events such as:
  - `\x1b[13;28;13;1;0;1_`
- the trace now also clearly shows **modifier-only events** in the stream, for example:
  - `\x1b[16;42;0;1;16;1_`
  - which Bubble Tea decodes as `leftshift`

This confirms that the real terminal path is still ordinary key replay, not a true paste event, and that the fallback burst logic must survive noisy modifier traffic inside the replay stream.

## The main remaining engineering gaps

### 1. The burst detector still needs to be hardened further

Even after the modifier-only fix, the current detector is still a simplified version of what Codex does.

Likely next improvements:

- move closer to the full Codex `PasteBurst` behavior
- preserve burst context more robustly across noisy replay streams
- consider the richer Codex paths:
  - consecutive plain-char counting
  - `on_plain_char_no_hold`-style handling for non-ASCII / IME / special text paths
  - retro-capture of recently inserted text when a stream becomes clearly paste-like

### 2. Atomic paste-token model is still missing

This repo still does **not** have the Codex-style atomic placeholder/token layer for large pastes.

That means these requested behaviors are still unfinished:

- auto collapse to a placeholder like `[Pasted N lines]`
- inverse placeholder styling
- deleting any character in the placeholder deletes the whole token
- submit-time expansion back into full text

### 3. The fallback path still is not instant enough

Codex’s UX bar is:

- paste appears as one bulk operation
- multiline paste never accidentally submits
- large paste can collapse into one placeholder

This repo is still between stages:

- better than raw character replay
- but not yet at a fully Codex-like bulk-paste / token-aware model

## Recommended next steps

1. **Keep working in `internal/composer`, not in app-level shortcuts**
   - the real fix direction is still composer-owned

2. **Continue hardening the composer-local burst detector**
   - especially around noisy Win32 replay streams
   - use the live trace as the truth source

3. **Implement atomic paste tokens next**
   - this is the biggest remaining structural step
   - it is also needed for:
     - `[Pasted N lines]`
     - inverse styling
     - atomic deletion
     - submit-time expansion

4. **Retest against real trace-driven behavior after each slice**
   - synthetic tests are useful, but the live terminal trace has already proven more complicated than the synthetic Win32 tests

## Repo files most relevant to resume from

- `internal/composer/paste_burst.go`
- `internal/composer/paste_burst_test.go`
- `internal/composer/textarea.go`
- `internal/composer/textarea_test.go`
- `internal/app/update.go`
- `internal/app/model.go`
- `internal/app/messages.go`
- `internal/app/program_smoke_test.go`

## Current task status summary

- `composer-paste-ownership`: done
- `composer-paste-burst-detector`: in progress
- `composer-paste-token-model`: pending
- `composer-paste-token-rendering`: pending
- `composer-paste-regressions`: pending

## Short version

What worked:

- moving fallback paste logic into the composer was the right direction
- the modifier-only live-trace bug was real and got fixed

What did not finish:

- live paste still sometimes submits
- Codex-style atomic paste tokens are not implemented yet

If resuming later, the best next move is:

- continue hardening the composer burst detector using the fresh Win32 trace, then
- build the atomic paste-token layer
