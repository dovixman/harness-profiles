# cli-foundation Specification

## Purpose
Define the `hp` command-line entrypoint, its interactive and scriptable behavior, package boundaries, and default storage locations.
## Requirements
### Requirement: Go CLI binary
The system SHALL provide a Go CLI binary named `hp` as the primary entrypoint for both scriptable commands and interactive terminal workflows.

#### Scenario: User runs help
- **WHEN** the user runs `hp --help`
- **THEN** the CLI prints available global commands and exits successfully.

#### Scenario: User runs interactive default
- **WHEN** the user runs `hp` without arguments in an interactive terminal
- **THEN** the CLI opens the interactive dashboard.

### Requirement: Interactive default entrypoint
The system SHALL launch the interactive dashboard when a human user runs `hp` without arguments.

#### Scenario: User runs hp without arguments in a terminal
- **WHEN** the user runs `hp` without arguments from an interactive terminal
- **THEN** the CLI launches the interactive dashboard.

### Requirement: Guided incomplete commands
The system SHALL launch guided interactive flows for mutation commands when required arguments are omitted and the process is attached to an interactive terminal.

#### Scenario: User runs add without arguments
- **WHEN** the user runs `hp add` from an interactive terminal
- **THEN** the CLI launches the add-harness guided flow instead of printing a usage error.

#### Scenario: User runs switch without a profile
- **WHEN** the user runs `hp <harness> switch` from an interactive terminal
- **THEN** the CLI launches a guided profile switch flow for that harness.

### Requirement: Scriptable command preservation
The system SHALL preserve non-interactive behavior for complete commands and for processes that are not attached to an interactive terminal, including commands for single-link and multi-link harnesses.

#### Scenario: Complete command remains scriptable
- **WHEN** the user runs `hp <harness> switch <profile>`
- **THEN** the CLI performs the switch for every managed link without launching the TUI.

#### Scenario: Missing argument in non-interactive process
- **WHEN** a non-interactive process runs a command with missing required arguments
- **THEN** the CLI prints usage or an error and exits non-zero instead of blocking for input.

#### Scenario: Single-link add command remains compatible
- **WHEN** the user runs the existing `hp add <harness> --link <path> --profile <profile>` command
- **THEN** the CLI registers a single link, infers the kind of an existing file or directory, and defaults a missing path to a directory link.

#### Scenario: Multi-link harness paths are inspectable
- **WHEN** the user runs `hp <harness> where` for a multi-link harness
- **THEN** the CLI prints all managed link ids, kinds, and paths in a scriptable format.

### Requirement: Hexagonal package layout
The implementation SHALL separate domain, application use cases, adapters, and executable entrypoint into distinct packages.

#### Scenario: Developer locates domain logic
- **WHEN** a developer opens the repository
- **THEN** core entities and rules are available under an internal domain package without depending on terminal UI packages.

### Requirement: Default storage locations
The system SHALL use `~/.config/harness-profiles/` as the default home for configuration and managed profiles.

#### Scenario: User asks for paths
- **WHEN** the user runs `hp where`
- **THEN** the CLI prints the config path and harnesses root path.
