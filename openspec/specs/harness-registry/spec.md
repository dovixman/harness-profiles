# harness-registry Specification

## Purpose
Define registration, update, deletion, listing, and inspection behavior for harness registry entries.
## Requirements
### Requirement: Add harness
The system SHALL allow users to register a harness with id, label, one or more managed links, and restart hint.

#### Scenario: Add harness without existing root
- **WHEN** the user adds a single-link harness whose config root path does not exist
- **THEN** the system asks for an initial profile name, creates that managed profile artifact, persists the harness, and creates the config root symlink to the profile artifact.

#### Scenario: Add harness with missing multi-link paths
- **WHEN** the user adds a harness whose configured link paths do not exist
- **THEN** the system asks for an initial profile name, creates one managed profile artifact per link using each configured kind, persists the harness, and creates every configured symlink.

#### Scenario: Normalize config root path
- **WHEN** the user provides `~`, `~/...`, or a relative config root path for any managed link
- **THEN** the system expands it to an absolute path before inspecting the filesystem or persisting the harness.

#### Scenario: Add harness with real root directory
- **WHEN** a configured directory link path exists as a real directory
- **THEN** the system asks for explicit confirmation and an initial profile name before copying that directory into the managed profile artifact and replacing the path with a symlink.

#### Scenario: Add harness with real root file
- **WHEN** a configured file link path exists as a real file
- **THEN** the system asks for explicit confirmation and an initial profile name before copying that file into the managed profile artifact and replacing the path with a symlink.

#### Scenario: Add harness with external symlink
- **WHEN** a configured link path is a symlink pointing outside the managed profile store
- **THEN** the system allows the user to import the target into a new managed profile artifact and repoint the symlink to that artifact.

### Requirement: Update harness
The system SHALL allow users to update label, managed links, and restart hint for an existing harness.

#### Scenario: Update managed link path
- **WHEN** a managed link path changes and the current profile is known
- **THEN** the system creates a new symlink to the current profile artifact for that link before optionally removing the old symlink.

#### Scenario: Update multiple managed links
- **WHEN** one or more managed links are added, removed, or changed
- **THEN** the system validates link uniqueness, preflights affected filesystem paths, and persists the updated harness only after required filesystem operations succeed.

### Requirement: Delete harness
The system SHALL allow users to delete a harness registry entry and choose how to handle profiles and every managed link path.

#### Scenario: Restore profile during harness deletion
- **WHEN** the user deletes a harness and chooses restore
- **THEN** each managed link symlink is removed and the selected profile artifact is copied back to that link path before the managed profiles directory is removed.

#### Scenario: Keep root during harness deletion
- **WHEN** the user deletes a harness and chooses keep-root
- **THEN** the registry entry and managed profiles directory are removed while every managed link path is left untouched.

#### Scenario: Delete all during harness deletion
- **WHEN** the user deletes a harness and chooses delete-all
- **THEN** the system requires typed confirmation before deleting the managed profiles directory and every managed link path.
