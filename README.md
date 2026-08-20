# harness-profiles

`hp` is a small Go CLI for managing harness profile configuration.

The human-facing default is interactive: running `hp` in a terminal opens a guided Bubble Tea TUI. Complete commands remain scriptable for automation, and non-interactive missing-argument invocations still print usage/errors instead of blocking for input.

## Installation

The installer downloads the latest release for macOS or Linux, verifies its SHA-256 checksum, and installs `hp` in `~/.local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/dovixman/harness-profiles/main/install.sh | sh
```

Set `HP_INSTALL_DIR` to install it elsewhere:

```sh
curl -fsSL https://raw.githubusercontent.com/dovixman/harness-profiles/main/install.sh | HP_INSTALL_DIR=/usr/local/bin sh
```

Set `HP_VERSION` to install a specific release:

```sh
curl -fsSL https://raw.githubusercontent.com/dovixman/harness-profiles/main/install.sh | HP_VERSION=v1.0.0 sh
```

Developers with Go 1.24.2 or newer can also install from source:

```sh
go install github.com/dovixman/harness-profiles/cmd/hp@latest
```

### Release Process

Merges to `main` update a Release Please pull request based on Conventional Commits. Merging that release pull request creates the next semantic-version tag and GitHub Release, then GoReleaser uploads the platform binaries and checksums.

- `fix:` increments the patch version.
- `feat:` increments the minor version.
- `feat!:` or a `BREAKING CHANGE` footer increments the major version.

## Default Paths

- Config home: `~/.config/harness-profiles/`
- Config file: `~/.config/harness-profiles/config.json`
- Harnesses root: `~/.config/harness-profiles/harnesses/`

Set `HP_CONFIG_HOME` to override the config home for tests or local experimentation.

## Technical Documentation

See [docs/technical.md](docs/technical.md) for architecture, filesystem safety rules, CLI/TUI boundaries, and validation guidance.

## Commands

```sh
hp
hp --help
hp where
hp ls
hp tui
hp add claude --link root=~/.claude/agents --link state=~/.claude.json --profile default
hp claude switch work
```

## Interactive TUI

Run `hp` or `hp tui` to browse harnesses and profiles interactively. In an interactive terminal, incomplete mutation commands open the matching guided flow:

```sh
hp add
hp update
hp delete
hp claude switch
hp claude clone
hp claude delete
```

The TUI includes searchable harness/profile lists, styled panels, active-profile badges, guided forms, path-aware add flows, mutation previews, spinner progress, and clear success/error result screens.

When adding a harness, each managed link is a filesystem path that `hp` will switch between profiles. For example, Claude Code might manage both `~/.claude/agents` and `~/.claude.json` as separate links.

Each link path is normalized before inspection and storage: `~` expands to your home directory, relative paths become absolute paths, and `config.json` stores each link entry with normalized `id`, `path`, and `kind`. New configs persist explicit `links`; legacy `link_path` entries still load as a single `root` link.

The guided add flow inspects each managed link before confirming:

- The TUI collects links one at a time with explicit ID, path, `Directory`/`File`, and existing-symlink choices. Added links remain visible before continuing to the profile step.
- If a path does not exist, `hp` asks for an initial profile name, creates `harnesses/<harness>/<profile>/<link-id>` artifacts, and creates each managed symlink.
- If a path is a real directory or file, `hp` asks for a profile name, copies the artifact into the matching profile link artifact, and replaces the managed link with a symlink.
- If a path is a symlink, `hp` asks whether to import or register it for that specific link.

Managed links can also be changed from harness profile detail. Select a link to update its path, choose `Add managed link` to add a directory or file artifact to every profile, or press `d` to unregister the selected link. Unregistering removes the managed symlink and config entry but preserves existing profile artifacts with that link ID.

Keyboard shortcuts:

- `j`/`down` and `k`/`up`: move the cursor.
- Type on harness/profile screens: filter the searchable list.
- `backspace`: edit the current search filter.
- `space`: select the focused option in guided forms.
- `enter`: open the focused harness/profile/action, or submit the last form field.
- Harness detail shows `Managed links`, `Profiles`, and `Actions` sections. `enter` or `u` updates a selected link, while `d` unregisters it.
- `tab`/`shift+tab` or `up`/`down`: move through guided form fields and options.
- In guided forms, `space` selects the focused option.
- `y`/`n`: confirm or cancel previewed operations.
- `esc`/`backspace`: go back or cancel.
- `ctrl+c`: exit immediately; the dashboard also has a selectable `Quit` action.

## Scriptable Behavior

Fully specified commands do not launch the TUI:

```sh
hp add claude --link root=~/.claude/agents --link state=~/.claude.json --profile default
hp update claude --label "Claude Code"
hp claude switch work
hp claude clone default experiment --materialize
hp claude delete experiment --yes
```

When stdin/stdout are not terminals, incomplete commands keep CLI behavior and exit non-zero with usage output. This keeps shell scripts and tests deterministic.

## Config Format

```json
{
  "harnesses_root": "/Users/me/.config/harness-profiles/harnesses",
  "harnesses": [
    {
      "id": "claude",
      "label": "Claude Code",
      "links": [
        {
          "id": "root",
          "path": "/Users/me/.claude/agents",
          "kind": "dir"
        },
        {
          "id": "state",
          "path": "/Users/me/.claude.json",
          "kind": "file"
        }
      ],
      "restart_hint": "Restart Claude Code after switching profiles."
    }
  ]
}
```

Legacy configs with `link_path` still load as a single `root` link.
