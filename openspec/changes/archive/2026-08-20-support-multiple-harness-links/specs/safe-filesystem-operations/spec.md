## ADDED Requirements

### Requirement: File artifact operations
The system SHALL safely copy, move, restore, and delete regular file artifacts as first-class managed profile content.

#### Scenario: Copy file artifact
- **WHEN** a managed file link is imported or restored
- **THEN** the system copies the regular file without treating it as a directory.

#### Scenario: File kind mismatch
- **WHEN** a configured file link path resolves to a directory or a configured directory link path resolves to a file
- **THEN** the system fails with a validation error before mutation.

## MODIFIED Requirements

### Requirement: Refuse unsafe overwrite
The system SHALL refuse to overwrite real directories or files during switch operations for any managed link.

#### Scenario: Switch encounters real root
- **WHEN** any managed root path exists and is not a symlink
- **THEN** the switch operation fails before changing managed links and suggests adoption instead.

#### Scenario: Multi-link switch preflight fails
- **WHEN** any managed link cannot be safely replaced or its target artifact is invalid
- **THEN** the switch operation fails before changing any managed link.

### Requirement: Explicit destructive confirmation
The system SHALL require typed confirmation for operations that can delete harness stores or managed root paths, including every path in a multi-link harness.

#### Scenario: Harness delete all
- **WHEN** harness deletion would delete one or more managed root paths
- **THEN** the system requires a dedicated typed confirmation before deleting the managed profile store or any managed root path.

### Requirement: Config update ordering
The system SHALL persist config changes only after required filesystem operations succeed across all affected managed links.

#### Scenario: Root path update fails
- **WHEN** creating any new symlink fails during a root path update
- **THEN** the existing config remains unchanged.

#### Scenario: Multi-link update partially fails
- **WHEN** one link mutation fails after another link mutation succeeded
- **THEN** the system reports the partial failure and does not persist the updated config.

### Requirement: Testable filesystem behavior
The system SHALL support filesystem tests using temporary directories without touching real user config paths.

#### Scenario: Core operation test
- **WHEN** a test configures a temporary home and harnesses root
- **THEN** profile operations run entirely inside that temporary tree.

#### Scenario: Multi-link executable test
- **WHEN** an executable test configures a harness with directory and file links under temporary paths
- **THEN** switching, deleting, and restoring profiles do not read or mutate the user's real configured harnesses.
