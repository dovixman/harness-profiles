## Why

The current TUI is technically interactive, but the experience is too plain: it lacks visual hierarchy, iconography, motion feedback, searchable selection, and guided command flows. The CLI should feel like a polished terminal product rather than a static list with keybindings.

## What Changes

- Make the interactive interface the default product experience for humans while preserving scriptable CLI commands for automation.
- Upgrade TUI screens with consistent visual language: icons, status badges, borders, colors, help text, progress spinners, and success/error states.
- Replace freeform-only flows with guided widgets: searchable selects, autocomplete, confirmation previews, and multiselect where users choose from existing harnesses or profiles.
- Route relevant commands through interactive flows when required arguments are omitted, such as `hp add`, `hp update`, `hp delete`, `hp <harness> switch`, `clone`, `adopt`, and `delete`.
- Keep non-interactive commands available when all required arguments are provided, so scripts and tests remain stable.
- Add automated tests around interactive command routing and TUI model behavior.

## Capabilities

### New Capabilities

### Modified Capabilities

- `interactive-tui`: Require a polished interactive terminal experience with visual affordances, animated feedback, searchable choices, autocomplete, guided forms, and destructive previews.
- `cli-foundation`: Change command behavior so human-facing commands can launch guided interactive flows when required arguments are missing, while complete flag-based invocations remain scriptable.

## Impact

- Affected code: `internal/adapters/cli`, `internal/adapters/tui`, tests, and README usage documentation.
- Dependencies: likely additional Charm Bubble packages such as list/spinner/key/help or equivalent Bubble Tea components.
- User behavior: `hp` and incomplete mutation commands become interactive entrypoints instead of only printing help/usage errors.
- Compatibility: complete non-interactive command invocations continue to work for automation.
