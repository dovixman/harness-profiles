## Why

The initial harness profile manager prototype is difficult to test and package portably. A Go CLI gives the project a portable, testable foundation before adding richer profile operations and a TUI.

## What Changes

- Create a Go module for a new `hp` CLI.
- Use a hexagonal architecture with domain, application ports, and adapters separated.
- Replace `config.fish` with a structured `config.yaml` file.
- Define default filesystem locations under `~/.config/harness-profiles/`.
- Add basic non-destructive commands for help, version, config inspection, and harness listing.

## Capabilities

### New Capabilities
- `cli-foundation`: Defines the Go CLI bootstrap, package layout, command entrypoint, and default locations.
- `structured-config`: Defines YAML-based configuration loading, validation, and persistence.

### Modified Capabilities

## Impact

- Adds Go module files and source tree under the project root.
- Adds dependencies for CLI parsing and YAML handling.
- Establishes the config format consumed by later changes.
