# Technical Documentation

## Purpose

`hp` manages local developer-tool configuration roots as named harnesses with one or more switchable managed links. Each managed link is replaced with a symlink to the active profile artifact for that link, so external tools keep reading their normal config paths while `hp` controls which profile backs them.

## Architecture

The code follows a ports-and-adapters shape:

- `cmd/hp` is only the process entry point.
- `internal/adapters/cli` parses scriptable commands and delegates use cases to `app.Service`.
- `internal/adapters/tui` owns Bubble Tea state, forms, navigation, previews, and result screens.
- `internal/app` sequences use cases and coordinates config plus filesystem ports.
- `internal/domain` contains validation and domain invariants.
- `internal/adapters/configyaml` persists local JSON config and resolves `HP_CONFIG_HOME` paths.
- `internal/adapters/fsops` implements filesystem operations behind the app `FileSystem` port.

Adapters should call `app.Service`; they should not duplicate domain rules or mutate persisted config directly.

## Core Model

- Harness: stable ID, user label, one or more managed links, optional restart hint.
- Managed link: stable link ID, normalized filesystem path, and artifact kind (`dir` or `file`).
- Profile: a directory under `<harnesses_root>/<harness-id>/<profile-name>`, with managed artifacts stored per link under `<profile-name>/<link-id>`.
- Active profile: the profile whose artifacts back all managed links.
- External profile: one or more managed links point outside the managed harness root.

## Filesystem Safety

Filesystem mutation is centralized behind `app.FileSystem` and implemented by `fsops.OS`.

Important rules:

- Replacing a managed link refuses to overwrite a real file or directory.
- Removing a managed link symlink is a no-op for missing paths and an error for real paths.
- Adding an existing real directory or file copies it into the managed profile store before replacing the link with a symlink.
- Adding an external symlink requires explicit import or registration behavior for that link.
- `delete-all` requires confirmation with the harness ID and covers every managed link path.

## CLI And TUI Behavior

The CLI and TUI share behavior through `app.Service`.

- The primary CLI form for multiple links uses repeated `--link <id>=<path>`.
- Legacy `--link <path>` remains compatible for a single link and is normalized to `root`.

- Fully specified CLI commands stay scriptable and do not launch the TUI.
- In an interactive terminal, incomplete mutation commands open the matching guided TUI flow.
- In non-interactive mode, incomplete commands return usage errors for deterministic automation.

The TUI is split by flow/view files to keep update logic, rendering, previews, and execution dispatch localized.

## Testing And Validation

Tests must isolate runtime state with `HP_CONFIG_HOME` or temp directories. They must not read or mutate the user's real harness configuration.

Required validation before merge:

```sh
gofmt -w $(git ls-files '*.go')
make test
go test ./internal/adapters/cli -run 'TestExecutable' -count=1
golangci-lint run --output.text.path=stdout
```

`make test` runs the complete suite and fails unless global statement coverage is strictly greater than 80.0%.
