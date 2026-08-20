## ADDED Requirements

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
The system SHALL preserve non-interactive behavior for complete commands and for processes that are not attached to an interactive terminal.

#### Scenario: Complete command remains scriptable
- **WHEN** the user runs `hp <harness> switch <profile>`
- **THEN** the CLI performs the switch without launching the TUI.

#### Scenario: Missing argument in non-interactive process
- **WHEN** a non-interactive process runs a command with missing required arguments
- **THEN** the CLI prints usage or an error and exits non-zero instead of blocking for input.

## MODIFIED Requirements

### Requirement: Go CLI binary
The system SHALL provide a Go CLI binary named `hp` as the primary entrypoint for both scriptable commands and interactive terminal workflows.

#### Scenario: User runs help
- **WHEN** the user runs `hp --help`
- **THEN** the CLI prints available global commands and exits successfully.

#### Scenario: User runs interactive default
- **WHEN** the user runs `hp` without arguments in an interactive terminal
- **THEN** the CLI opens the interactive dashboard.
