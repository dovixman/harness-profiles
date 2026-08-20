## 1. Project Bootstrap

- [x] 1.1 Initialize the Go module and choose the module path.
- [x] 1.2 Create `cmd/hp/main.go` as the binary entrypoint.
- [x] 1.3 Add initial `README.md` with CLI purpose and default paths.

## 2. Architecture Skeleton

- [x] 2.1 Create `internal/domain` with harness and config value types.
- [x] 2.2 Create `internal/app` with use case interfaces for read-only commands.
- [x] 2.3 Create adapter packages for CLI parsing and YAML config persistence.

## 3. Config Foundation

- [x] 3.1 Add YAML config structs and load/save behavior.
- [x] 3.2 Add validation for harness ids, labels, link paths, and profiles root.
- [x] 3.3 Add unit tests for config loading, missing config, and validation errors.

## 4. Read-Only Commands

- [x] 4.1 Implement `hp --help` and `hp where`.
- [x] 4.2 Implement `hp ls` against the YAML config repository.
- [x] 4.3 Add command tests using a temporary config home.
