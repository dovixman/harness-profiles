## Why

Claude Code keeps relevant global state across more than one filesystem location: the managed config directory can be redirected, but `~/.claude.json` remains a separate file. The current `hp` model can only manage one symlinked config root per harness, so switching a Claude profile can leave mixed state across profiles.

## What Changes

- Allow a harness to define multiple managed links instead of a single config root path.
- Support managed links that point to either directories or files.
- Switch, restore, delete, adopt, import, and current-profile detection operate across all links in a harness.
- Preserve backward compatibility with existing `link_path` configs by treating them as a single legacy link.
- Update CLI, TUI, previews, and docs to display and manage multiple affected paths.
- Add safety checks so multi-link operations preflight all links before mutating user filesystem state where possible.
- Non-goal: replace symlink switching with wrappers or environment-variable launchers.

## Capabilities

### New Capabilities
- `multi-link-harnesses`: Defines multi-link harness behavior, link identity, link kind, profile storage layout, and migration from single-link harnesses.

### Modified Capabilities
- `harness-registry`: Harness add, update, delete, list, and inspect behavior changes from one config root path to one or more managed links.
- `profile-management`: Current profile detection, switch, adopt, clone, rename, and delete behavior must account for every managed link in a harness.
- `safe-filesystem-operations`: Filesystem safety rules must cover multi-link preflight, file links, partial failure handling, and destructive previews over multiple paths.
- `structured-config`: Config format must persist multiple links while loading existing `link_path` entries compatibly.
- `interactive-tui`: TUI detail screens, forms, and previews must present multiple managed links and their consequences.
- `cli-foundation`: Scriptable output and command flags must remain usable when a harness has multiple links.

## Impact

- Domain model: replace the single-path assumption with a collection of harness links while keeping legacy compatibility.
- App service: update add/update/delete/profile flows to inspect and mutate all links through filesystem ports.
- Filesystem adapter: add safe file copy/move/restore behavior alongside existing directory support.
- Config adapter: read old `link_path`, write new multi-link shape, and validate uniqueness across all link paths.
- CLI/TUI/docs/tests: update user-facing commands, previews, status output, and executable tests for multi-link harnesses.
