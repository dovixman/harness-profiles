## Context

The CLI must remain scriptable, but workflows like adding a harness over an existing symlink or deleting a harness require context-heavy choices. Bubble Tea is a good fit for guided terminal flows without making the core app depend on terminal libraries.

## Goals / Non-Goals

**Goals:**
- Add `hp tui` as the explicit interactive entrypoint.
- Let `hp` without arguments optionally open the TUI only if configured that way later; default behavior remains help/current CLI output.
- Keep Bubble Tea models in an adapter package.
- Reuse application use cases for all state changes.

**Non-Goals:**
- Replace normal CLI commands.
- Add a graphical desktop UI.
- Implement process restarts.

## Decisions

- Use Bubble Tea for state machine and rendering.
- Use Bubbles for list, text input, viewport, and confirmation prompts where useful.
- Use Lip Gloss for styling but keep output readable in plain terminals.
- Model navigation as high-level screens: harness list, harness detail, profile picker, operation form, confirmation, result.
- Every destructive TUI action must show the exact paths affected before confirmation.

## Risks / Trade-offs

- TUI can obscure scriptability if overused -> mitigate by making `hp tui` explicit and keeping CLI commands first-class.
- Terminal styling can become noisy -> mitigate with minimal styles and plain-text fallbacks.
- UI state machines can get tangled -> mitigate by keeping each operation form small and delegating mutations to app use cases.
