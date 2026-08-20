## 1. Interactive Routing

- [x] 1.1 Make `hp` without arguments launch the interactive dashboard when stdin/stdout are terminals, while keeping help available through `hp help`, `-h`, and `--help`.
- [x] 1.2 Route incomplete mutation commands (`hp add`, `hp update`, `hp delete`, `hp <harness> switch`, `adopt`, `clone`, `delete`) to operation-specific TUI flows in interactive terminals.
- [x] 1.3 Preserve current non-interactive behavior for fully specified commands and missing-argument invocations in non-terminal contexts.

## 2. TUI Visual System

- [x] 2.1 Introduce reusable TUI styles for titles, panels, selected rows, muted help, active badges, warnings, destructive actions, success, and errors.
- [x] 2.2 Upgrade the dashboard and harness detail screens with icons, borders, visual status badges, current-profile markers, and contextual key help.
- [x] 2.3 Add result and progress states with success/error icons and spinner-based animation for confirmed mutations.

## 3. Guided Widgets and Flows

- [x] 3.1 Replace plain profile/harness choice flows with searchable list selection for existing harnesses and profiles.
- [x] 3.2 Add guided forms with labels, placeholders, validation feedback, and submit/cancel controls for add and update harness flows.
- [x] 3.3 Add guided switch, adopt, clone, and profile delete flows using profile selection/autocomplete where applicable.
- [x] 3.4 Add multiselect or selectable option controls for add-harness symlink/import handling and clone symlink materialization choices where applicable.

## 4. Previews and Safety

- [x] 4.1 Add preview screens before filesystem-changing operations showing affected harness, link path, profile paths, and operation consequences.
- [x] 4.2 Ensure destructive operations require explicit confirmation and clearly refuse active-profile deletion.
- [x] 4.3 Keep all mutations routed through `internal/app` service methods without duplicating filesystem business logic in the TUI adapter.

## 5. Documentation and Tests

- [x] 5.1 Update README examples to describe interactive-first usage, command fallbacks, keybindings, and non-interactive scripting behavior.
- [x] 5.2 Add or update CLI tests for interactive routing versus non-interactive usage behavior.
- [x] 5.3 Add or update TUI model tests for dashboard rendering, selection/search flows, form validation, previews, result states, and operation dispatch.
- [x] 5.4 Run `go test ./...`, `go build ./cmd/hp`, and `openspec validate make-tui-interactive-first --strict`.
