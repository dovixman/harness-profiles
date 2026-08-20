## Context

The initial prototype established the desired API shape and safety semantics. The Go implementation preserves those semantics while making operations testable with temporary filesystems and explicit use case boundaries.

## Goals / Non-Goals

**Goals:**
- Implement all core non-TUI harness and profile commands.
- Refuse unsafe overwrites unless a command explicitly supports a confirmed migration path.
- Detect active profiles by resolving the managed link target against known profile directories.
- Keep destructive operations guarded by typed confirmation prompts.

**Non-Goals:**
- Implement Bubble Tea views.
- Automatically migrate legacy `~/.claude-profiles` or `~/.opencode-profiles` stores.
- Add daemon/process restart behavior; restart hints remain informational.

## Decisions

- Keep `hp <harness> <command>` for profile operations and global `hp <command>` for registry operations.
- Treat `hp delete <harness>` and `hp <harness> delete <profile>` as distinct commands based on argument position.
- Use copy by default when importing an external symlink target into the new store, so legacy profile stores are not silently destroyed.
- Use move when adopting a real root directory, because adoption converts the current runtime directory into the managed profile source of truth.
- Make `--yes` valid for profile deletion only. Harness deletion must keep interactive typed confirmations.

## Risks / Trade-offs

- Copying external symlink targets can leave old and new stores diverging -> mitigate by reporting the copied source and new target clearly.
- Deleting harnesses can remove large directories -> mitigate with summary output and typed confirmations.
- Updating a managed link path can leave stale old symlinks -> mitigate by creating the new symlink before removing the old one and asking before removal.
