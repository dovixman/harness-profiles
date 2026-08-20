## ADDED Requirements

### Requirement: Polished visual language
The TUI SHALL use a consistent visual language with icons, status badges, borders, colors, contextual help, and distinct success, warning, error, active, and destructive states.

#### Scenario: User opens dashboard
- **WHEN** the user launches the interactive interface
- **THEN** the system displays a styled dashboard with a title, harness list, active profile indicators, available keybindings, and visual status markers.

#### Scenario: Operation completes
- **WHEN** an interactive operation succeeds or fails
- **THEN** the TUI displays a visually distinct result state with an icon, short summary, and next available actions.

### Requirement: Animated operation feedback
The TUI SHALL show animated progress feedback for filesystem or configuration mutations that may take noticeable time.

#### Scenario: Mutation starts
- **WHEN** the user confirms an add, update, delete, switch, adopt, clone, or profile delete operation
- **THEN** the TUI shows a spinner or equivalent animated progress state until the operation returns.

### Requirement: Searchable selection and autocomplete
The TUI SHALL provide searchable selection for existing harnesses and profiles, and autocomplete known harness or profile values where users must choose an existing item.

#### Scenario: Select existing profile
- **WHEN** the user starts a switch, clone, or delete-profile flow for a harness with profiles
- **THEN** the TUI lets the user search or navigate available profiles instead of typing the full profile name from memory.

#### Scenario: Select existing harness
- **WHEN** the user starts an update or delete-harness flow without specifying a harness
- **THEN** the TUI lets the user search or navigate configured harnesses.

### Requirement: Guided forms
The TUI SHALL provide guided forms with labels, placeholders, validation feedback, and submit/cancel controls for operations requiring user input.

#### Scenario: Add harness interactively
- **WHEN** the user starts the add-harness flow
- **THEN** the TUI prompts for harness ID, label, link path, restart hint, initial profile, and external symlink handling with validation before submission.

### Requirement: Destructive and filesystem previews
The TUI SHALL display a preview before any operation that deletes files, moves directories, clones profile content, imports external symlinks, registers external symlinks, or repoints managed symlinks.

#### Scenario: Delete profile preview
- **WHEN** the user prepares to delete a profile interactively
- **THEN** the TUI shows the profile name, resolved profile path, active-profile status, and exact deletion consequence before confirmation.

### Requirement: Multiselect where multiple choices are useful
The TUI SHALL support multiselect controls for flows that ask the user to choose zero or more related options.

#### Scenario: Add harness options
- **WHEN** the user adds a harness interactively and the managed path already exists or is an external symlink
- **THEN** the TUI presents applicable handling options as selectable choices rather than requiring memorized flags.

## MODIFIED Requirements

### Requirement: TUI entrypoint
The system SHALL provide `hp tui` as an explicit alias for the interactive terminal UI, and the same interactive terminal UI SHALL be available from human-facing command flows.

#### Scenario: Launch TUI
- **WHEN** the user runs `hp tui`
- **THEN** the system opens an interactive terminal interface showing configured harnesses.

### Requirement: Guided mutations
The TUI SHALL provide guided, styled, validated forms for add, update, delete, switch, adopt, clone, and profile delete operations.

#### Scenario: Switch profile from TUI
- **WHEN** the user selects a profile switch action and confirms
- **THEN** the TUI invokes the same switch use case as the CLI command and displays the result.

#### Scenario: Clone profile from TUI
- **WHEN** the user selects a profile clone action
- **THEN** the TUI lets the user select the source profile, enter the target profile, choose symlink handling, preview the copy operation, and confirm before invoking the clone use case.

### Requirement: Destructive previews
The TUI SHALL show exact affected paths and consequences before destructive operations.

#### Scenario: Delete harness from TUI
- **WHEN** the user chooses to delete a harness
- **THEN** the TUI displays the config entry, managed profiles directory, managed root path, and chosen root handling mode before asking for confirmation.

#### Scenario: Delete profile from TUI
- **WHEN** the user chooses to delete a profile
- **THEN** the TUI displays the profile path and refuses or warns clearly when the selected profile is active.
