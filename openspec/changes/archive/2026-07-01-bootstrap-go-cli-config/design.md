## Context

The initial prototype proved the workflow but mixed command parsing, filesystem behavior, and persistence. The Go implementation preserves the validated API shape while making the core logic testable and portable.

## Goals / Non-Goals

**Goals:**
- Build a small Go CLI binary named `hp`.
- Keep domain logic independent from terminal UI and filesystem details.
- Store configuration as YAML at `~/.config/harness-profiles/config.yaml` by default.
- Store profiles under `~/.config/harness-profiles/profiles/<harness>/<profile>`.
- Make config paths overrideable for tests and future packaging.

**Non-Goals:**
- Implement symlink-changing profile operations in this change.
- Implement Bubble Tea screens in this change.
- Migrate existing Fish config in this change.

## Decisions

- Use Go as the implementation language because it produces a single portable binary and has strong filesystem testing support.
- Use hexagonal architecture:
  - `internal/domain` for entities such as `Harness`, `Profile`, and config value objects.
  - `internal/app` for use cases and ports.
  - `internal/adapters/configyaml` for YAML persistence.
  - `internal/adapters/fs` for later filesystem operations.
  - `internal/adapters/cli` for command parsing.
  - `cmd/hp` for the executable entrypoint.
- Use YAML for human-editable config. JSON can be supported later as import/export, but YAML is the canonical on-disk format.
- Keep the command surface scriptable: TUI will be an adapter on top, not the only way to operate the tool.

## Risks / Trade-offs

- YAML introduces a dependency and indentation-sensitive config -> mitigate with validation and clear error messages.
- Hexagonal structure adds folders early -> mitigate by keeping interfaces minimal and only introducing ports used by real use cases.
- Keeping `hp` as the binary name may conflict with local aliases -> document the command and avoid shell-level aliases.
