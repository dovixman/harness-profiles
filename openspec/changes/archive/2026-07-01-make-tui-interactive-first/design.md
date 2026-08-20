## Context

`hp` currently has scriptable commands plus an explicit `hp tui` screen. The TUI reuses application services, which is good, but the user experience is too bare: plain text lists, limited discoverability, no motion feedback, no autocomplete, and command flows that fall back to usage errors instead of helping the user complete the operation.

The next iteration should treat Bubble Tea as the primary human interface while keeping complete flag-based command invocations stable for automation.

## Goals / Non-Goals

**Goals:**

- Make `hp` without arguments open a polished interactive dashboard.
- Make incomplete human commands open guided flows instead of failing with terse usage output.
- Add visual affordances: icons, colors, borders, badges, progress spinners, contextual help, and clear success/error states.
- Add practical widgets: searchable harness/profile selection, autocomplete for known values, guided forms, and multiselect for destructive or batch-like choices where relevant.
- Preserve non-interactive behavior when commands include all required arguments.
- Keep business logic in `internal/app`; TUI code remains an adapter.

**Non-Goals:**

- Implement fish migration flows.
- Add persistent UI preferences or theme configuration.
- Replace existing application service APIs unless required by the UI.
- Build a full terminal framework abstraction beyond what this product needs now.

## Decisions

### Decision: Interactive-first command routing

`hp` with no arguments SHALL launch the dashboard. Commands with missing required operands SHALL launch the corresponding guided flow when running in an interactive terminal. Fully specified commands SHALL keep their current scriptable behavior.

Alternative considered: always require `hp tui`. Rejected because it keeps the product split between an attractive path and a plain path, which is exactly the problem.

### Decision: Use Bubble components instead of hand-rolled plain strings

The TUI SHALL use Bubble Tea components for lists, text inputs, viewport/help/spinner where useful, plus Lip Gloss styling. Searchable lists and autocomplete can be implemented with `bubbles/list` filtering and focused text inputs rather than inventing a separate widget system.

Alternative considered: keep the current minimal model and only add colors. Rejected because the missing pieces are interaction quality and discoverability, not just paint.

### Decision: One TUI package with operation-specific models

Keep `internal/adapters/tui` as the terminal adapter, but split complex screens into focused model helpers or files if needed: dashboard, harness detail, forms, confirm previews, result/progress. This avoids leaking UI concerns into `internal/app`.

Alternative considered: create separate packages per command. Rejected for now because the workflows share selection, forms, styles, and service calls.

### Decision: Preview before mutation

Every destructive or filesystem-changing flow SHALL show a preview before execution. The preview should include affected harness ID, link path, profile paths, root handling mode, and whether the operation will delete, move, clone, or repoint symlinks.

Alternative considered: rely on existing service errors. Rejected because a polished TUI must make consequences obvious before execution.

## Risks / Trade-offs

- Terminal detection can be flaky in tests -> keep an explicit fallback path and test TUI routing through injectable stdin/stdout where possible.
- More dependencies increase surface area -> prefer Charm packages already aligned with Bubble Tea.
- Animated UI is harder to snapshot-test -> test model state transitions and command routing rather than brittle full-frame snapshots.
- Interactive-first can annoy scripts -> preserve non-interactive behavior for complete invocations and avoid prompting when stdin/stdout is not a terminal.
