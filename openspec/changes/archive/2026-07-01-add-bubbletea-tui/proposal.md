## Why

The profile manager has several interactive workflows with risky filesystem choices. A Bubble Tea TUI can make those flows clearer and safer while preserving the scriptable CLI for automation.

## What Changes

- Add a Bubble Tea-powered TUI mode for browsing harnesses and profiles.
- Provide interactive flows for adding, updating, deleting, switching, adopting, and cloning.
- Use existing application use cases instead of duplicating business logic in UI code.
- Display confirmations, previews, and warnings before destructive operations.

## Capabilities

### New Capabilities
- `interactive-tui`: Defines TUI entrypoints, navigation, screens, keybindings, and confirmation UX.

### Modified Capabilities

## Impact

- Adds Bubble Tea and companion terminal UI dependencies.
- Adds a terminal adapter package.
- Does not change the YAML config format or core command semantics.
