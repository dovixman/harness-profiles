## Purpose

Provide an interactive-first terminal UI for guided harness and profile management while keeping scriptable CLI commands first-class.
## Requirements
### Requirement: TUI entrypoint
The system SHALL provide `hp tui` as an explicit alias for the interactive terminal UI, and the same interactive terminal UI SHALL be available from human-facing command flows.

#### Scenario: Launch TUI
- **WHEN** the user runs `hp tui`
- **THEN** the system opens an interactive terminal interface showing configured harnesses.

### Requirement: Harness navigation
The TUI SHALL allow users to browse configured harnesses and inspect their profiles and all managed links.

#### Scenario: Select harness
- **WHEN** the user selects a harness in the TUI
- **THEN** the TUI shows the harness link paths, harness store path, current profile state, and available actions.

### Requirement: Guided mutations
The TUI SHALL provide guided, styled, validated forms for add, update, delete, switch, clone, create-profile, update-profile, and profile delete operations, including multi-link harness inputs and previews where applicable.

#### Scenario: Switch profile from TUI
- **WHEN** the user selects a profile switch action and confirms
- **THEN** the TUI invokes the same switch use case as the CLI command and displays the result for all managed links.

#### Scenario: Clone profile from TUI
- **WHEN** the user selects a profile clone action
- **THEN** the TUI lets the user select the source profile, enter the target profile, choose symlink handling, preview the copy operation, and confirm before invoking the clone use case.

### Requirement: Destructive previews
The TUI SHALL show exact affected paths and consequences before destructive operations across every managed link.

#### Scenario: Delete harness from TUI
- **WHEN** the user chooses to delete a harness
- **THEN** the TUI displays the config entry, managed profiles directory, every managed link path, and chosen root handling mode before asking for confirmation.

#### Scenario: Delete profile from TUI
- **WHEN** the user chooses to delete a profile
- **THEN** the TUI displays the profile path and refuses or warns clearly when the selected profile is active.

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
- **WHEN** the user confirms an add, update, delete, switch, clone, create-profile, update-profile, or profile delete operation
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
The TUI SHALL provide guided forms with labels, placeholders, validation feedback, and submit/cancel controls for operations requiring user input, including adding one or more managed links.

#### Scenario: Add harness interactively
- **WHEN** the user starts the add-harness flow
- **THEN** the TUI prompts for harness ID, label, one or more managed link paths with kinds, and restart hint before inspecting the configured paths.

#### Scenario: Add harness with existing config root
- **WHEN** any configured link path exists as a real file, real directory, or symlink
- **THEN** the TUI asks for a new profile name before showing the final confirmation preview.

#### Scenario: Manage links from profile detail
- **WHEN** the user opens a harness profile detail screen
- **THEN** the TUI offers guided add, update, and remove actions for managed links without requiring the harness metadata form.

#### Scenario: Remove managed link
- **WHEN** the user confirms removal of a managed link from profile detail
- **THEN** the TUI removes its config entry and symlink while clearly stating that existing per-profile artifacts are preserved.

### Requirement: Destructive and filesystem previews
The TUI SHALL display a preview before any operation that deletes files, moves directories or files, clones profile content, imports external symlinks, registers external symlinks, or repoints managed symlinks.

#### Scenario: Delete profile preview
- **WHEN** the user prepares to delete a profile interactively
- **THEN** the TUI shows the profile name, resolved profile path, active-profile status across all managed links, and exact deletion consequence before confirmation.

#### Scenario: Multi-link switch preview
- **WHEN** the user prepares to switch a multi-link harness interactively
- **THEN** the TUI shows every link path and target profile artifact that will be repointed.

### Requirement: Selectable options where choices are useful
The TUI SHALL support selectable option controls for flows that ask the user to choose among related options.

#### Scenario: Clone profile options
- **WHEN** the user clones a profile interactively
- **THEN** the TUI lets the user select symlink handling from focused options using keyboard navigation.

### Requirement: Non-duplicated business logic
The TUI SHALL use application use cases for state changes rather than implementing filesystem operations directly.

#### Scenario: TUI operation executes
- **WHEN** a TUI action performs a mutation
- **THEN** the mutation passes through the same application service used by the CLI command.
