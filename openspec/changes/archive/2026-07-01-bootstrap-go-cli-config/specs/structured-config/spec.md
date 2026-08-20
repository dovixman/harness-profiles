## ADDED Requirements

### Requirement: YAML configuration file
The system SHALL persist harness registry configuration in `config.yaml` by default.

#### Scenario: Config exists
- **WHEN** the CLI starts and `config.yaml` exists
- **THEN** it loads configured harnesses from that file.

#### Scenario: Config is missing
- **WHEN** the CLI starts and `config.yaml` is missing
- **THEN** read-only commands handle the empty registry without creating or modifying symlinks.

### Requirement: Harness config fields
Each harness config entry SHALL include an id, label, managed link path, and optional restart hint.

#### Scenario: Harness is listed
- **WHEN** the user runs `hp ls`
- **THEN** each configured harness is shown with its id, label, and managed link path.

### Requirement: Config validation
The system SHALL reject invalid harness ids and missing link paths before persisting configuration.

#### Scenario: Invalid harness id
- **WHEN** the user attempts to persist a harness id containing whitespace or `/`
- **THEN** the operation fails with a validation error.
