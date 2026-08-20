## 1. Domain and Config Model

- [x] 1.1 Add `HarnessLink` domain type with id, path, and `dir`/`file` kind validation.
- [x] 1.2 Add canonical harness link helpers so legacy `LinkPath` is treated as one directory link.
- [x] 1.3 Update config validation to reject missing links, invalid link ids/kinds, and duplicate normalized link paths across all harnesses.
- [x] 1.4 Update config repository load/save to read legacy `link_path` and persist new `links` entries.
- [x] 1.5 Add domain and config repository tests for legacy single-link and new multi-link configs.

## 2. Filesystem Port and Adapter

- [x] 2.1 Extend app filesystem needs to support regular file copy/move and artifact kind checks.
- [x] 2.2 Implement safe file operations in `internal/adapters/fsops` without touching non-temp user paths in tests.
- [x] 2.3 Add preflight helpers for replacing multiple symlinks only when all destinations are safe.
- [x] 2.4 Add fsops tests for file copy, file move, kind mismatch, and unsafe overwrite refusal.

## 3. Harness Lifecycle Flows

- [x] 3.1 Update add-harness flow to create/import/register one profile artifact per configured link.
- [x] 3.2 Update harness update flow to add/change/remove managed links and persist only after filesystem success.
- [x] 3.3 Update delete-harness restore, keep-root, and delete-all modes to handle every managed link path.
- [x] 3.4 Add app tests for adding missing dir+file links, importing existing artifacts, updating link paths, and delete modes.

## 4. Profile Lifecycle Flows

- [x] 4.1 Update current-profile detection to report active, external, missing, or mixed state across all links.
- [x] 4.2 Update switch-profile to preflight all target artifacts and repoint every managed symlink.
- [x] 4.3 Update adopt-profile to move every real configured path into the selected profile before linking.
- [x] 4.4 Update create, rename, clone, and delete profile flows for multi-link profile artifact layout.
- [x] 4.5 Add app tests for switch preflight failure, mixed current state, active delete refusal, clone, and rename.

## 5. CLI and TUI UX

- [x] 5.1 Preserve existing single-link CLI commands and output for compatible usage.
- [x] 5.2 Add scriptable multi-link inspection output for `hp ls` and `hp <harness> where`.
- [x] 5.3 Add or confirm CLI syntax for creating/updating multiple links and cover it with executable tests.
- [x] 5.4 Update TUI harness detail, forms, and previews to show all managed links and link kinds.
- [x] 5.5 Update destructive TUI previews to list every affected file, directory, and symlink.
- [x] 5.6 Add TUI tests for rendering multi-link harness details and previews.
- [x] 5.7 Add guided profile-detail actions to add, update, and delete managed links.

## 6. Documentation and Validation

- [x] 6.1 Update README config examples and command examples for multi-link harnesses.
- [x] 6.2 Document Claude Code example with `~/.claude` resources plus `~/.claude.json` as separate managed links.
- [x] 6.3 Run `gofmt` on changed Go files.
- [x] 6.4 Run `go test ./internal/adapters/tui` after TUI changes.
- [x] 6.5 Run `go test ./internal/adapters/cli -run 'TestExecutable' -count=1` after CLI/executable changes.
- [x] 6.6 Run `go test ./...`.
- [x] 6.7 Run `golangci-lint run --output.text.path=stdout`.
