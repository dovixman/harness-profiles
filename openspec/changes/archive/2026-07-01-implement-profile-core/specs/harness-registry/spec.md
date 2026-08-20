## ADDED Requirements

### Requirement: Add harness
The system SHALL allow users to register a harness with id, label, managed link path, and restart hint.

#### Scenario: Add harness without existing root
- **WHEN** the user adds a harness whose managed link path does not exist
- **THEN** the harness is persisted and its profile root directory is created.

#### Scenario: Add harness with real root directory
- **WHEN** the managed link path exists as a real directory
- **THEN** the system asks for explicit confirmation and an initial profile name before moving that directory into the managed profile store and replacing it with a symlink.

#### Scenario: Add harness with external symlink
- **WHEN** the managed link path is a symlink pointing outside the managed profile store
- **THEN** the system allows the user to either import the target into a new managed profile or register the external target as-is.

### Requirement: Update harness
The system SHALL allow users to update label, managed link path, and restart hint for an existing harness.

#### Scenario: Update managed link path
- **WHEN** the managed link path changes and the current profile is known
- **THEN** the system creates a new symlink to the current profile before optionally removing the old symlink.

### Requirement: Delete harness
The system SHALL allow users to delete a harness registry entry and choose how to handle profiles and the managed root path.

#### Scenario: Restore profile during harness deletion
- **WHEN** the user deletes a harness and chooses restore
- **THEN** the chosen profile is copied back to the managed root path as a real directory before the managed profiles directory is removed.

#### Scenario: Keep root during harness deletion
- **WHEN** the user deletes a harness and chooses keep-root
- **THEN** the registry entry and managed profiles directory are removed while the managed root path is left untouched.

#### Scenario: Delete all during harness deletion
- **WHEN** the user deletes a harness and chooses delete-all
- **THEN** the system requires typed confirmation before deleting both managed profiles and the managed root path.
