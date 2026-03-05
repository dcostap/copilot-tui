# IDEA.md

## Vision. Essentials. 

Build a custom terminal app that feels like a simplified Copilot CLI:
- Prompt/input box at the bottom. Multi-line. Basic text manipulation. Auto text wrapping.
- Streaming conversation output at the top. Auto rendering markdown.
- Full chat buffer visible and scrollable. Mouse input. Clickable links.
- Clear rendering of normal text, reasoning/thinking text, and tool activity.
- The app is a UI wrapper around the Copilot SDK, not a reimplementation of the agent engine.
- Focus on simplicity: all features are hand tailored according to the needs and workflow of the single user. It's an app for personal use. It's not meant to be used by others.
- Windows-only for now; don't worry about support for other platforms.

## Core User Experience

### Layout
- **Top/Center pane:** conversation timeline (user + assistant + system/tool events).
- **Bottom pane:** editable prompt box (multi-line input, queue message on Enter, insert newline in your prompt on Shift+Enter, etc).
- **Status/footer line:** mode, model, token/usage hints, connection state, etc.

Very clean looking, very uninstrusive, like CODEX CLI.

### Streaming behavior
- Assistant text appears incrementally as chunks arrive.
- Reasoning/thinking chunks are visually distinct from final answer text.
- Tool events are shown as structured timeline items:
  - tool start,
  - tool status/progress,
  - tool completion (success/failure + summary).

### Conversation buffer
- Keep full session history in memory (with optional compaction marker display).
- Support scrolling, jump to latest, and search in transcript.

## What the app should represent

The UI should map Copilot SDK events into readable blocks. Check up-to-date Copilot SDK docs for details.

## Commands

One single, global, simple entry point for all commands, customizations, options, etc: a command palette triggered with Ctrl+P.
