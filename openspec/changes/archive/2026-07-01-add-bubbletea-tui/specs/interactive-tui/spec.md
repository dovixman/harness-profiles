## ADDED Requirements

### Requirement: TUI entrypoint
The system SHALL provide an explicit `hp tui` command that launches the interactive terminal UI.

#### Scenario: Launch TUI
- **WHEN** the user runs `hp tui`
- **THEN** the system opens an interactive terminal interface showing configured harnesses.

### Requirement: Harness navigation
The TUI SHALL allow users to browse configured harnesses and inspect their profiles.

#### Scenario: Select harness
- **WHEN** the user selects a harness in the TUI
- **THEN** the TUI shows the harness link path, profiles root, current profile, and available actions.

### Requirement: Guided mutations
The TUI SHALL provide guided forms for add, update, delete, switch, adopt, and clone operations.

#### Scenario: Switch profile from TUI
- **WHEN** the user selects a profile switch action and confirms
- **THEN** the TUI invokes the same switch use case as the CLI command and displays the result.

### Requirement: Destructive previews
The TUI SHALL show exact affected paths before destructive operations.

#### Scenario: Delete harness from TUI
- **WHEN** the user chooses to delete a harness
- **THEN** the TUI displays the config entry, managed profiles directory, managed root path, and chosen root handling mode before asking for confirmation.

### Requirement: Non-duplicated business logic
The TUI SHALL use application use cases for state changes rather than implementing filesystem operations directly.

#### Scenario: TUI operation executes
- **WHEN** a TUI action performs a mutation
- **THEN** the mutation passes through the same application service used by the CLI command.
