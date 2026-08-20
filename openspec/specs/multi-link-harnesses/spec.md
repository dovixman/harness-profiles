# multi-link-harnesses Specification

## Purpose
Define how one harness coordinates multiple directory and file links through a shared profile lifecycle.
## Requirements
### Requirement: Harness links
A harness SHALL be able to define one or more managed links, each with a stable link id, normalized filesystem path, and artifact kind of `dir` or `file`.

#### Scenario: Harness has multiple links
- **WHEN** a harness is configured with multiple links
- **THEN** the system treats all links as part of the same harness and profile lifecycle.

#### Scenario: Link ids are stable profile artifact names
- **WHEN** the system stores profile artifacts for a link
- **THEN** it uses the link id as the artifact name under the managed profile directory.

### Requirement: Profile artifact layout
The system SHALL store each managed link artifact under `harnesses/<harness>/<profile>/<link-id>`.

#### Scenario: Directory and file artifacts share one profile
- **WHEN** a profile contains a directory link `agents` and a file link `state`
- **THEN** the managed artifacts are stored at `harnesses/<harness>/<profile>/agents` and `harnesses/<harness>/<profile>/state`.

### Requirement: Link kind behavior
The system SHALL preserve the configured link kind when creating, importing, restoring, switching, cloning, or adopting profile artifacts.

#### Scenario: Directory link target
- **WHEN** the configured link kind is `dir`
- **THEN** the profile artifact MUST be a directory before the link can be switched to that profile.

#### Scenario: File link target
- **WHEN** the configured link kind is `file`
- **THEN** the profile artifact MUST be a regular file before the link can be switched to that profile.

### Requirement: Legacy single-link compatibility
The system SHALL load existing harness entries with `link_path` as single-link harnesses without requiring manual config migration.

#### Scenario: Existing config uses link_path
- **WHEN** the config file contains a harness with `link_path` and no `links`
- **THEN** the system exposes that harness as having one directory link to the normalized `link_path`.

#### Scenario: New multi-link config exists
- **WHEN** the config file contains a harness with `links`
- **THEN** the system uses the configured link ids, paths, and kinds instead of deriving links from `link_path`.
