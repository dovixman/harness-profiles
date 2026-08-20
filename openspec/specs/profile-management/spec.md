# profile-management Specification

## Purpose
Define profile lifecycle behavior within a configured harness.
## Requirements
### Requirement: List profiles
The system SHALL list profile directories for a configured harness and mark the active managed profile only when all managed links agree on the same active profile.

#### Scenario: Profiles exist
- **WHEN** the user runs `hp <harness> ls`
- **THEN** the system lists profile names and marks the active managed profile when known across all managed links.

#### Scenario: Current links are mixed
- **WHEN** managed links point to different profiles or include unknown targets
- **THEN** the system lists profiles without incorrectly marking a single active profile.

### Requirement: Detect current profile
The system SHALL resolve every managed link target and map the harness to a known profile only when all links consistently point to artifacts for the same profile.

#### Scenario: Current target is managed
- **WHEN** every managed link points to `harnesses/<harness>/<profile>/<link-id>` for the same profile
- **THEN** `hp <harness> current` shows the profile name as active.

#### Scenario: Current target is external
- **WHEN** one or more managed links point outside the harness profile directory
- **THEN** `hp <harness> current` reports the external state and the raw external target paths.

#### Scenario: Current targets are mixed
- **WHEN** managed links point to different profiles, missing paths, or unexpected artifacts
- **THEN** `hp <harness> current` reports a mixed or unknown state instead of naming one active profile.

### Requirement: Switch profile
The system SHALL repoint every managed symlink for a harness to the selected existing profile artifacts.

#### Scenario: Switch to existing profile
- **WHEN** the user runs `hp <harness> switch <profile>` and the profile exists with all required artifacts
- **THEN** every managed link points to that profile's corresponding artifact and the restart hint is printed.

#### Scenario: Switch target missing artifact
- **WHEN** the selected profile does not contain an artifact for any configured link
- **THEN** the switch operation fails before changing managed links.

### Requirement: Adopt real root
The system SHALL convert real managed root files or directories into profile artifacts for the configured harness links.

#### Scenario: Adopt existing root
- **WHEN** the user runs `hp <harness> adopt <profile>` and each managed link path is a real file or directory matching its configured kind
- **THEN** each path is moved into the profile store under its link id and replaced by a symlink.

#### Scenario: Adopt blocked by symlink or missing path
- **WHEN** any managed link path is missing or already a symlink
- **THEN** adoption fails before moving any managed link path.

### Requirement: Create profile
The system SHALL create an empty managed profile directory for a configured harness with placeholder artifacts for each configured link kind.

#### Scenario: Create empty profile
- **WHEN** the user creates a profile for a harness from the TUI profile actions
- **THEN** the system creates `harnesses/<harness>/<profile>/` and one empty artifact per configured link without switching to it.

### Requirement: Update profile
The system SHALL rename managed profile directories for a configured harness and keep all active managed links pointing at the renamed profile artifacts.

#### Scenario: Rename active profile
- **WHEN** the user renames the active managed profile
- **THEN** the profile directory is renamed and every managed config root symlink points to the renamed profile's corresponding artifact.

### Requirement: Clone profile
The system SHALL clone an existing profile with all configured link artifacts to a new profile name.

#### Scenario: Clone preserving symlinks
- **WHEN** the user clones a profile without `--materialize`
- **THEN** symlinks inside all profile artifacts are copied as symlinks.

#### Scenario: Clone materializing symlinks
- **WHEN** the user clones a profile with `--materialize`
- **THEN** symlinks inside all profile artifacts are dereferenced and copied as real content.

### Requirement: Delete profile
The system SHALL delete non-active profiles with confirmation and SHALL refuse deletion when the profile is active across the harness links.

#### Scenario: Delete active profile refused
- **WHEN** the user tries to delete the active profile
- **THEN** the system refuses and asks the user to switch first.

#### Scenario: Delete profile while current state is mixed
- **WHEN** the current harness links are mixed or unknown
- **THEN** the system refuses to infer safety from a single link and explains that the user must repair or switch first.
