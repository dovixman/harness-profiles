package tui

import (
	"fmt"
	"strings"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func (m Model) viewProfileLink() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(operationTitle(m.op)) + "\n")
	if m.op == opUpdateLink {
		if link, ok := m.selectedProfileLink(); ok {
			fmt.Fprintf(&b, "%s %s (%s)\n", helpStyle.Render("Managed link:"), link.ID, link.Kind)
		}
	}
	if m.err != nil {
		b.WriteString("\n" + errStyle.Render("✕ "+m.err.Error()) + "\n")
	}
	b.WriteString("\n")
	for index, field := range m.fields {
		b.WriteString(m.formFieldLine(index, field) + "\n")
	}
	if m.op == opAddLink {
		base := len(m.fields)
		b.WriteString("\n" + helpStyle.Render("Artifact type") + "\n")
		b.WriteString(m.profileLinkChoice(base, m.linkKind == domain.HarnessLinkKindDir, "Directory") + "\n")
		b.WriteString(m.profileLinkChoice(base+1, m.linkKind == domain.HarnessLinkKindFile, "File") + "\n")
		b.WriteString("\n" + helpStyle.Render("Existing symlink") + "\n")
		b.WriteString(m.profileLinkChoice(base+2, m.linkAction == app.HarnessLinkActionImport, "Import target into active profile") + "\n")
		b.WriteString(m.profileLinkChoice(base+3, m.linkAction == app.HarnessLinkActionRegister, "Keep target external") + "\n")
	}
	b.WriteString("\n" + m.profileLinkButton(m.profileLinkFocusCount()-1, "Review changes"))
	return m.panel(b.String(), m.footer("tab/↑/↓ move", "space select", "enter activate", "esc cancel", "? help"))
}

func (m Model) profileLinkChoice(focus int, selected bool, label string) string {
	marker := "○"
	if selected {
		marker = "●"
	}
	line := fmt.Sprintf("%s %s", marker, label)
	if m.field == focus {
		return selectStyle.Render("› " + line)
	}
	if selected {
		return okStyle.Render("  " + line)
	}
	return "  " + line
}

func (m Model) profileLinkButton(focus int, label string) string {
	if m.field == focus {
		return "  " + selectStyle.Render("› "+label)
	}
	return "  " + buttonStyle.Render(label)
}

func (m Model) profileLinkPreview(link domain.HarnessLink) string {
	if m.op == opUpdateLink {
		old, _ := m.selectedProfileLink()
		return fmt.Sprintf("Update managed link %s?\n\nHarness: %s\nKind: %s\nOld path: %s\nNew path: %s\nConsequence: repoints the new path to the active profile artifact and removes the old symlink.", link.ID, m.harness.ID, link.Kind, old.Path, link.Path)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Add managed link %s?\n\n", link.ID)
	fmt.Fprintf(&b, "Harness: %s\nKind: %s\nPath: %s\n", m.harness.ID, link.Kind, link.Path)
	fmt.Fprintf(&b, "Existing symlink action: %s\n", m.linkAction)
	b.WriteString("Profile artifacts:\n")
	for _, profile := range m.profiles {
		fmt.Fprintf(&b, "  - %s\n", m.profileArtifactPath(link, profile.Name))
	}
	b.WriteString("Consequence: creates one artifact per profile and manages the configured path from the active profile.")
	return b.String()
}

func (m Model) deleteProfileLinkPreview(link domain.HarnessLink) string {
	return fmt.Sprintf("Remove managed link %s?\n\nHarness: %s\nKind: %s\nManaged path: %s\nConsequence: removes this symlink and its config entry. Existing profile artifacts named %s are preserved.", link.ID, m.harness.ID, link.Kind, link.Path, link.ID)
}
