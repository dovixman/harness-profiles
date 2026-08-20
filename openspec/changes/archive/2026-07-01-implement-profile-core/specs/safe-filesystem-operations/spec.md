## ADDED Requirements

### Requirement: Refuse unsafe overwrite
The system SHALL refuse to overwrite real directories or files during switch operations.

#### Scenario: Switch encounters real root
- **WHEN** the managed root path exists and is not a symlink
- **THEN** the switch operation fails and suggests adoption instead.

### Requirement: Explicit destructive confirmation
The system SHALL require typed confirmation for operations that can delete harness stores or managed root paths.

#### Scenario: Harness delete all
- **WHEN** harness deletion would delete the managed root path
- **THEN** the system requires a dedicated typed confirmation before deletion.

### Requirement: Config update ordering
The system SHALL persist config changes only after required filesystem operations succeed.

#### Scenario: Root path update fails
- **WHEN** creating the new symlink fails during a root path update
- **THEN** the existing config remains unchanged.

### Requirement: Testable filesystem behavior
The system SHALL support filesystem tests using temporary directories without touching real user config paths.

#### Scenario: Core operation test
- **WHEN** a test configures a temporary home and profiles root
- **THEN** profile operations run entirely inside that temporary tree.
