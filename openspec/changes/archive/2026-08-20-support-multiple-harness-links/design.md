## Context

`hp` currently models each harness as one managed config root path (`link_path`) that is replaced with a symlink to `harnesses/<harness>/<profile>`. This fits tools whose profile state lives under one directory, but some tools distribute profile state across multiple directories and files. A single wrapper or home-directory override therefore cannot replace filesystem profile switching consistently across supported harnesses.

The repo already has a hexagonal boundary: CLI/TUI call `app.Service`, filesystem mutation goes through the app `FileSystem` port, and persisted config is loaded through `configyaml`. The design should preserve that boundary while removing the single-link assumption.

## Goals / Non-Goals

**Goals:**

- Represent one harness as one or more managed filesystem links.
- Support both directory and file link targets.
- Keep existing single-link configs working without manual migration.
- Keep switch/delete/restore/adopt/import operations safe when multiple paths are involved.
- Make CLI and TUI show the full set of affected paths.
- Keep filesystem behavior testable with temp `HP_CONFIG_HOME` and temp managed roots.

**Non-Goals:**

- Do not replace symlink switching with wrappers, aliases, shims, or environment-variable launchers.
- Do not add tool-specific Claude/Codex/Pi/OpenCode logic to the domain model.
- Do not support nested profile names or arbitrary path templates inside profile directories.
- Do not guarantee a fully atomic filesystem transaction across multiple OS paths; preflight and best-effort rollback are sufficient for this change.

## Decisions

### Decision: model links explicitly with stable IDs

Add a domain type similar to:

```go
type HarnessLink struct {
    ID   string
    Path string
    Kind string // "dir" or "file"
}
```

`domain.Harness` keeps `LinkPath` only as a compatibility field while exposing canonical links through methods or normalized construction. New configs persist `links`; legacy configs with `link_path` load as one link with a reserved/default ID such as `root` and kind `dir`.

Alternatives considered:

- Store only `[]string`: rejected because file vs directory behavior needs to be explicit and UI needs stable labels.
- Make each path a separate harness: rejected because profiles must switch as one logical unit.

### Decision: profile layout stores one artifact per link ID

For a harness `claude`, profile `work`, and links `agents` and `state`, targets live under:

```text
harnesses/claude/work/agents
harnesses/claude/work/state
```

Switching profile repoints each configured link path to its corresponding artifact path. This keeps clone/delete/rename operations profile-directory based while allowing heterogeneous artifacts inside the profile.

Alternatives considered:

- One profile directory per link under `harnesses/<harness>/<link>/<profile>`: rejected because profile-level clone/delete/list operations become scattered.
- Preserve original basename inside profile: rejected because basenames can collide and are less stable than configured link IDs.

### Decision: validate all links before mutating where possible

Multi-link operations perform preflight checks for every affected link before creating/removing/repointing symlinks. Preflight verifies profile artifacts exist with the expected kind, destination paths are missing or symlinks when replacement is required, and destructive confirmations cover every affected path.

If an operation fails after mutation begins, the service returns a clear partial-failure error, restores known previous symlink targets where possible, and avoids persisting config changes. Config persistence happens only after filesystem success.

Alternatives considered:

- Mutate links sequentially without preflight: rejected because it makes mixed profile state much more likely.
- Implement a full transaction log for rollback: deferred because the app is local-first and OS filesystem operations cannot be made truly atomic across paths.

### Decision: current profile can be active, external, missing, or mixed

Current-profile detection inspects every configured link. If all links point to artifacts under the same managed profile, the profile is active. If one or more links point outside the managed store, the state is external. If links point to different profiles, are missing, or have unexpected shapes, the state is mixed/unknown and profile deletion must not treat any candidate as safely inactive unless the active profile is unambiguous.

Alternatives considered:

- Use only the first link as the current-profile source of truth: rejected because it can hide split-brain Claude state.
- Refuse to show current status unless all links are perfect: rejected because external/mixed diagnostics are useful to recover.

### Decision: CLI uses repeatable link arguments

Scriptable commands remain compatible while supporting explicit multi-link input:

- Existing `hp add <id> --link <path> --profile <name>` creates a single directory link with the reserved ID `root`.
- Repeated `--link <id>=<path>` arguments create multiple managed links.
- `hp ls` and `hp <harness> where` must expose all links for multi-link harnesses without breaking single-link usage.
- The TUI provides guided multi-link input and previews every affected path.

### Decision: managed links are edited from profile detail

Managed links remain shared harness slots because every profile must expose the same artifact layout, but the TUI manages them alongside profiles. Adding a link creates an artifact for every profile and points its filesystem path at the active profile artifact. Updating changes the managed path while preserving the stable link ID and artifact kind.

Removing a link unregisters its symlink and config entry but leaves artifacts named after that link ID inside existing profiles. This avoids irreversible data loss and allows manual recovery or re-adding the same slot later.

## Risks / Trade-offs

- [Risk] Multi-link switch can leave partial symlink state if the OS fails after preflight. → Mitigation: preflight all links, persist config only after success, and return explicit partial-failure diagnostics; add best-effort rollback where previous targets are known.
- [Risk] File links make restore/delete operations more destructive than the current directory-only model. → Mitigation: require typed harness ID confirmation for delete-all and show every affected file and directory in previews.
- [Risk] Legacy `link_path` migrations could surprise users if the saved config changes shape. → Mitigation: load legacy configs compatibly and only write the new shape after a normal save/update; document the migration.
- [Risk] CLI repeated-link syntax may be awkward. → Mitigation: keep single-link syntax unchanged and accept explicit repeated `--link <id>=<path>` arguments.
- [Risk] Mixed current states complicate active-profile deletion checks. → Mitigation: refuse active deletion only when unambiguous and warn/report mixed states clearly so users can switch or repair first.

## Migration Plan

1. Add the new domain link model and config read support for both `link_path` and `links`.
2. Update app helpers so all flows consume canonical link collections.
3. Add filesystem support for file artifacts and kind checks.
4. Update profile operations to operate over every link with preflight.
5. Update CLI/TUI/docs to show multiple links and keep single-link usage compatible.
6. Add executable and service tests for legacy config, multi-dir links, dir+file links, mixed current state, and destructive modes.

Rollback strategy: because legacy configs still load, users can manually restore a single `link_path` config if needed. Filesystem rollback restores known previous targets where possible, and failed operations do not persist new config state.
