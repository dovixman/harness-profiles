## 1. Application Use Cases

- [x] 1.1 Implement harness add, update, delete, list, and inspect use cases.
- [x] 1.2 Implement profile list, current, switch, adopt, clone, delete, and inspect use cases.
- [x] 1.3 Add validation for harness names, profile names, and path inputs.

## 2. Filesystem Adapter

- [x] 2.1 Implement symlink inspection, creation, replacement, and removal helpers.
- [x] 2.2 Implement directory copy with preserve-symlink and materialize modes.
- [x] 2.3 Implement safe directory move and delete helpers with explicit error reporting.

## 3. CLI Commands

- [x] 3.1 Wire global commands: `add`, `update`, `delete`, `ls`, and `where`.
- [x] 3.2 Wire harness commands: `ls`, `current`, `switch`, `adopt`, `clone`, `delete`, and `where`.
- [x] 3.3 Implement command aliases matching the Fish prototype where useful.

## 4. Safety and Tests

- [x] 4.1 Add tests for adding harnesses across missing root, real root, managed symlink, and external symlink cases.
- [x] 4.2 Add tests for profile switch, adopt, clone, and delete behavior.
- [x] 4.3 Add tests for update root path ordering and harness delete modes.
