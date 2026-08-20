# structured-config Specification

## Purpose
Define JSON persistence, default paths, validation, and backward-compatible managed-link configuration.
## Requirements
### Requirement: JSON configuration file
The system SHALL persist harness registry configuration in `$HOME/.config/harness-profiles/config.json` by default.

#### Scenario: Config exists
- **WHEN** the CLI starts and `config.json` exists
- **THEN** it loads configured harnesses from that file.

#### Scenario: Config is missing
- **WHEN** the CLI starts and `config.json` is missing
- **THEN** read-only commands handle the empty registry without creating or modifying symlinks.

#### Scenario: Default config home
- **WHEN** no `HP_CONFIG_HOME` override is set
- **THEN** the system uses `$HOME/.config/harness-profiles` for config and profiles.

### Requirement: Harness config fields
Each harness config entry SHALL include an id, label, one or more managed links persisted as `links`, and optional restart hint, while existing `link_path` entries remain readable for compatibility.

#### Scenario: Harness is listed
- **WHEN** the user runs `hp ls`
- **THEN** each configured harness is shown with its id, label, and managed link path summary.

#### Scenario: Legacy harness config is loaded
- **WHEN** a harness entry contains `link_path` and no `links`
- **THEN** the system loads it as a single directory link without requiring the user to edit the config file.

#### Scenario: Multi-link harness config is saved
- **WHEN** a harness has more than one managed link or any file link
- **THEN** the config file persists those links with link id, path, and kind.

### Requirement: Config validation
The system SHALL reject invalid harness ids, missing managed links, invalid link ids, missing link paths, invalid link kinds, and duplicate managed link paths before persisting configuration.

#### Scenario: Invalid harness id
- **WHEN** the user attempts to persist a harness id containing whitespace or `/`
- **THEN** the operation fails with a validation error.

#### Scenario: Duplicate managed link path
- **WHEN** two harnesses or two links are configured with the same normalized managed link path
- **THEN** the operation fails with a validation error.

#### Scenario: Missing links
- **WHEN** a harness has neither `links` nor legacy `link_path`
- **THEN** the operation fails with a validation error.
