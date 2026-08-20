package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dovixman/harness-profiles/internal/app"
)

func (m *Model) addPreview() string {
	draft := m.addDraft
	profile := ""
	if draft.ProfileName != "" {
		profile = draft.ProfileName
	} else if len(m.fields) == 1 {
		profile = field(m, 0)
	}
	profileStore := filepath.Join(m.paths.HarnessesRoot, draft.ID)
	var b strings.Builder
	fmt.Fprintf(&b, "Add harness %s?\n\n", draft.ID)
	fmt.Fprintf(&b, "Harness ID: %s\n", draft.ID)
	fmt.Fprintf(&b, "Label: %s\n", draft.Label)
	if len(draft.Links) > 0 {
		b.WriteString("Managed links:\n")
		for _, line := range draftLinkPreviewLines(draft.Links) {
			fmt.Fprintf(&b, "  - %s\n", line)
		}
	} else {
		fmt.Fprintf(&b, "Config root: %s\n", draft.ConfigPath)
	}
	fmt.Fprintf(&b, "Detected state: %s\n", draft.Branch)
	if draft.Branch == addBranchSymlink {
		fmt.Fprintf(&b, "Symlink target: %s\n", draft.SourcePath)
	}
	fmt.Fprintf(&b, "Harness store: %s\n", profileStore)
	if draft.Branch == addBranchMissing {
		profilePath := filepath.Join(profileStore, profile)
		fmt.Fprintf(&b, "Profile name: %s\n", profile)
		fmt.Fprintf(&b, "Profile path: %s\n", profilePath)
		fmt.Fprintf(&b, "Create profile: %s\n", profilePath)
		for _, link := range draftLinkPaths(draft) {
			fmt.Fprintf(&b, "Create symlink: %s -> %s\n", link, profilePath)
		}
	} else {
		profilePath := filepath.Join(profileStore, profile)
		fmt.Fprintf(&b, "Profile name: %s\n", profile)
		fmt.Fprintf(&b, "Profile path: %s\n", profilePath)
		if len(draft.Links) > 0 {
			for _, link := range draftLinkPreviewLines(draft.Links) {
				fmt.Fprintf(&b, "Copy from / replace: %s\n", link)
			}
		} else {
			copyFrom := draft.ConfigPath
			if draft.Branch == addBranchSymlink {
				copyFrom = draft.SourcePath
			}
			fmt.Fprintf(&b, "Copy from: %s\n", copyFrom)
			fmt.Fprintf(&b, "Copy to: %s\n", profilePath)
			fmt.Fprintf(&b, "Replace symlink: %s -> %s\n", draft.ConfigPath, profilePath)
		}
	}
	fmt.Fprintf(&b, "After switch message: %s", draft.RestartHint)
	return b.String()
}

func (m *Model) deleteHarnessPreview() string {
	if !m.hasExplicitLinks() {
		return fmt.Sprintf("Delete harness %s?\n\nConfig entry: id=%s label=%s link_path=%s\nManaged profiles directory: %s\nManaged root path: %s\nRoot handling mode: %s\nRestore profile: %s\nConsequence: removes config entry and managed profile store; delete-all also deletes the linked root.", m.harness.ID, m.harness.ID, m.harness.Label, m.harness.LinkPath, filepath.Join(m.paths.HarnessesRoot, m.harness.ID), m.harness.LinkPath, m.deleteMode, m.deleteRestoreProfile())
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Delete harness %s?\n\n", m.harness.ID)
	fmt.Fprintf(&b, "Config entry: id=%s label=%s\n", m.harness.ID, m.harness.Label)
	fmt.Fprintf(&b, "Managed profiles directory: %s\n", filepath.Join(m.paths.HarnessesRoot, m.harness.ID))
	fmt.Fprintf(&b, "Managed links:\n")
	for _, line := range m.managedLinkLines() {
		fmt.Fprintf(&b, "  - %s\n", line)
	}
	fmt.Fprintf(&b, "Root handling mode: %s\n", m.deleteMode)
	fmt.Fprintf(&b, "Restore profile: %s\n", m.deleteRestoreProfile())
	b.WriteString("Consequence: removes config entry and managed profile store; delete-all also deletes every managed link path.")
	return b.String()
}

func (m *Model) updatePreview() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Update harness %s?\n\n", m.harness.ID)
	fmt.Fprintf(&b, "Config entry: id=%s label=%s\n", m.harness.ID, m.harness.Label)
	b.WriteString("Managed links:\n")
	for _, line := range m.managedLinkLines() {
		fmt.Fprintf(&b, "  - %s\n", line)
	}
	fmt.Fprintf(&b, "New label: %s\n", field(m, 0))
	fmt.Fprintf(&b, "Restart hint: %s\n", field(m, 1))
	b.WriteString("Consequence: updates harness metadata. Managed links are unchanged.")
	return b.String()
}

func (m *Model) adoptPreview() string {
	if !m.hasExplicitLinks() {
		return fmt.Sprintf("Adopt current root as profile %s?\n\nHarness: %s\nCurrent root: %s\nTarget profile path: %s\nConsequence: moves the real directory into the managed profile store and replaces it with a symlink.", field(m, 0), m.harness.ID, m.harness.LinkPath, filepath.Join(m.paths.HarnessesRoot, m.harness.ID, field(m, 0)))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Adopt current links as profile %s?\n\n", field(m, 0))
	fmt.Fprintf(&b, "Harness: %s\n", m.harness.ID)
	fmt.Fprintf(&b, "Managed links:\n")
	for _, line := range m.managedLinkLines() {
		fmt.Fprintf(&b, "  - %s\n", line)
	}
	fmt.Fprintf(&b, "Target profile path: %s\n", filepath.Join(m.paths.HarnessesRoot, m.harness.ID, field(m, 0)))
	fmt.Fprintf(&b, "Managed artifacts:\n")
	for _, line := range m.managedArtifactLines(field(m, 0)) {
		fmt.Fprintf(&b, "  - %s\n", line)
	}
	b.WriteString("Consequence: moves every configured path into the managed profile store and replaces each one with a symlink.")
	return b.String()
}

func (m Model) createProfilePreview() string {
	name := field(&m, 0)
	if !m.hasExplicitLinks() {
		return fmt.Sprintf("Create profile %s?\n\nHarness: %s\nProfile path: %s\nConsequence: creates an empty managed profile directory.", name, m.harness.ID, filepath.Join(m.paths.HarnessesRoot, m.harness.ID, name))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Create profile %s?\n\n", name)
	fmt.Fprintf(&b, "Harness: %s\n", m.harness.ID)
	fmt.Fprintf(&b, "Profile path: %s\n", filepath.Join(m.paths.HarnessesRoot, m.harness.ID, name))
	fmt.Fprintf(&b, "Managed artifacts:\n")
	for _, line := range m.managedArtifactLines(name) {
		fmt.Fprintf(&b, "  - %s\n", line)
	}
	b.WriteString("Consequence: creates one empty managed artifact per configured link.")
	return b.String()
}

func (m Model) renameProfilePreview() string {
	oldName := m.highlightedProfileName()
	newName := field(&m, 0)
	if !m.hasExplicitLinks() {
		return fmt.Sprintf("Update profile %s to %s?\n\nHarness: %s\nOld path: %s\nNew path: %s\nConsequence: renames the managed profile directory; if it is active, the config root symlink is repointed.", oldName, newName, m.harness.ID, filepath.Join(m.paths.HarnessesRoot, m.harness.ID, oldName), filepath.Join(m.paths.HarnessesRoot, m.harness.ID, newName))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Update profile %s to %s?\n\n", oldName, newName)
	fmt.Fprintf(&b, "Harness: %s\n", m.harness.ID)
	fmt.Fprintf(&b, "Old path: %s\n", filepath.Join(m.paths.HarnessesRoot, m.harness.ID, oldName))
	fmt.Fprintf(&b, "New path: %s\n", filepath.Join(m.paths.HarnessesRoot, m.harness.ID, newName))
	fmt.Fprintf(&b, "Managed artifacts:\n")
	for _, link := range m.managedLinks() {
		fmt.Fprintf(&b, "  - %s (%s): %s -> %s\n", link.ID, link.Kind, m.profileArtifactPath(link, oldName), m.profileArtifactPath(link, newName))
	}
	b.WriteString("Consequence: renames the managed profile directory; if it is active, every managed symlink is repointed.")
	return b.String()
}

func (m *Model) clonePreview() string {
	source := m.highlightedProfileName()
	target := field(m, 0)
	activation := "keep current active profile"
	if m.cloneActive {
		activation = "switch to cloned profile"
	}
	if !m.hasExplicitLinks() {
		return fmt.Sprintf("Clone profile %s to %s?\n\nHarness: %s\nSource path: %s\nTarget path: %s\nSymlink handling: %s\nAfter clone: %s\nConsequence: copies profile content into the new target profile.", source, target, m.harness.ID, m.highlightedProfilePath(), filepath.Join(m.paths.HarnessesRoot, m.harness.ID, target), m.selectedOption(), activation)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Clone profile %s to %s?\n\n", source, target)
	fmt.Fprintf(&b, "Harness: %s\n", m.harness.ID)
	fmt.Fprintf(&b, "Source path: %s\n", filepath.Join(m.paths.HarnessesRoot, m.harness.ID, source))
	fmt.Fprintf(&b, "Target path: %s\n", filepath.Join(m.paths.HarnessesRoot, m.harness.ID, target))
	fmt.Fprintf(&b, "Source artifacts:\n")
	for _, link := range m.managedLinks() {
		fmt.Fprintf(&b, "  - %s (%s): %s\n", link.ID, link.Kind, m.profileArtifactPath(link, source))
	}
	fmt.Fprintf(&b, "Target artifacts:\n")
	for _, link := range m.managedLinks() {
		fmt.Fprintf(&b, "  - %s (%s): %s\n", link.ID, link.Kind, m.profileArtifactPath(link, target))
	}
	fmt.Fprintf(&b, "Symlink handling: %s\n", m.selectedOption())
	fmt.Fprintf(&b, "After clone: %s\n", activation)
	b.WriteString("Consequence: copies every managed link artifact into the new target profile.")
	return b.String()
}

func (m Model) cloneActivationPrompt() string {
	target := field(&m, 0)
	return fmt.Sprintf("Set cloned profile %s as active?\n\nCurrent active profile: %s\nNew profile: %s\nChoose whether to switch after cloning.", target, m.currentProfileLabel(), target)
}

func (m Model) currentProfileLabel() string {
	switch m.current.State {
	case app.ProfileStateActive:
		if m.current.Name != "" {
			return m.current.Name
		}
		return "active"
	case app.ProfileStateExternal:
		if m.current.Path != "" {
			return "external: " + m.current.Path
		}
		return "external"
	case app.ProfileStateMixed:
		return "mixed"
	case app.ProfileStateMissing:
		return "missing"
	case app.ProfileStateUnknown:
		return "unknown"
	}
	if m.current.External {
		return "external"
	}
	if m.current.Name == "" {
		return "none"
	}
	return m.current.Name
}

func (m Model) switchProfilePreview(profile app.ProfileStatus) string {
	if !m.hasExplicitLinks() {
		return fmt.Sprintf("Switch %s to profile %s?\n\nAffected link: %s\nTarget profile path: %s", m.harness.ID, profile.Name, m.harness.LinkPath, profile.Path)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Switch %s to profile %s?\n\n", m.harness.ID, profile.Name)
	fmt.Fprintf(&b, "Managed links:\n")
	for _, link := range m.managedLinks() {
		fmt.Fprintf(&b, "  - %s (%s): %s -> %s\n", link.ID, link.Kind, link.Path, m.profileArtifactPath(link, profile.Name))
	}
	b.WriteString("Consequence: repoints every managed link to the selected profile artifact.")
	return b.String()
}
