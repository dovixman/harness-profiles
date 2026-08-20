## ADDED Requirements

### Requirement: List profiles
The system SHALL list profile directories for a configured harness.

#### Scenario: Profiles exist
- **WHEN** the user runs `hp <harness> ls`
- **THEN** the system lists profile names and marks the active managed profile when known.

### Requirement: Detect current profile
The system SHALL resolve the managed link target and map it to a known profile when possible.

#### Scenario: Current target is managed
- **WHEN** the managed link points to `profiles/<harness>/<profile>`
- **THEN** `hp <harness> current` shows the profile name as active.

#### Scenario: Current target is external
- **WHEN** the managed link points outside the harness profile directory
- **THEN** `hp <harness> current` reports the raw external target.

### Requirement: Switch profile
The system SHALL repoint the managed symlink to a selected existing profile.

#### Scenario: Switch to existing profile
- **WHEN** the user runs `hp <harness> switch <profile>` and the profile exists
- **THEN** the managed link points to that profile and the restart hint is printed.

### Requirement: Adopt real root
The system SHALL convert a real managed root directory into a profile.

#### Scenario: Adopt existing root
- **WHEN** the user runs `hp <harness> adopt <profile>` and the managed root is a real directory
- **THEN** the directory is moved into the profile store and replaced by a symlink.

### Requirement: Clone profile
The system SHALL clone an existing profile to a new profile name.

#### Scenario: Clone preserving symlinks
- **WHEN** the user clones a profile without `--materialize`
- **THEN** symlinks inside the profile are copied as symlinks.

#### Scenario: Clone materializing symlinks
- **WHEN** the user clones a profile with `--materialize`
- **THEN** symlinks inside the profile are dereferenced and copied as real content.

### Requirement: Delete profile
The system SHALL delete non-active profiles with confirmation.

#### Scenario: Delete active profile refused
- **WHEN** the user tries to delete the active profile
- **THEN** the system refuses and asks the user to switch first.
