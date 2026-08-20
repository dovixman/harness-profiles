## Why

After the Go CLI foundation exists, the tool needs complete harness and profile behavior. This change implements the scriptable core operations before adding an interactive TUI.

## What Changes

- Add global harness registry commands: `hp add`, `hp update <harness>`, `hp delete <harness>`.
- Add per-harness profile commands: `hp <harness> ls`, `current`, `switch`, `adopt`, `clone`, `delete`, and `where`.
- Manage symlinks safely with explicit confirmation for destructive operations.
- Preserve the new profile layout under `~/.config/harness-profiles/profiles/<harness>/<profile>`.
- Keep commands usable in scripts without requiring the TUI.

## Capabilities

### New Capabilities
- `harness-registry`: Defines adding, updating, deleting, listing, and inspecting harnesses.
- `profile-management`: Defines profile listing, current detection, switching, adoption, cloning, and profile deletion.
- `safe-filesystem-operations`: Defines safety rules for symlinks, real directories, confirmations, and rollback behavior.

### Modified Capabilities

## Impact

- Adds filesystem adapter behavior for symlinks, directory copy, directory move, and deletion.
- Adds application use cases for harness and profile lifecycle operations.
- Adds command coverage for the complete non-interactive workflow.
