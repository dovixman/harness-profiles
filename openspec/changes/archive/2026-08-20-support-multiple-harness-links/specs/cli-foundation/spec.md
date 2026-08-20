## MODIFIED Requirements

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
- **THEN** the CLI registers a single directory link harness without requiring new multi-link flags.

#### Scenario: Multi-link harness paths are inspectable
- **WHEN** the user runs `hp <harness> where` for a multi-link harness
- **THEN** the CLI prints all managed link ids, kinds, and paths in a scriptable format.
