## ADDED Requirements

### Requirement: Go CLI binary
The system SHALL provide a Go CLI binary named `hp` as the primary entrypoint.

#### Scenario: User runs help
- **WHEN** the user runs `hp --help`
- **THEN** the CLI prints available global commands and exits successfully.

### Requirement: Hexagonal package layout
The implementation SHALL separate domain, application use cases, adapters, and executable entrypoint into distinct packages.

#### Scenario: Developer locates domain logic
- **WHEN** a developer opens the repository
- **THEN** core entities and rules are available under an internal domain package without depending on terminal UI packages.

### Requirement: Default storage locations
The system SHALL use `~/.config/harness-profiles/` as the default home for configuration and managed profiles.

#### Scenario: User asks for paths
- **WHEN** the user runs `hp where`
- **THEN** the CLI prints the config path and profiles root path.
