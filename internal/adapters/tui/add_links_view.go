package tui

import (
	"fmt"
	"strings"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func (m Model) viewAddLinks() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("✎ Add harness") + "\n")
	b.WriteString(helpStyle.Render("Step 2/3: add every path that should switch with this profile") + "\n\n")
	b.WriteString(m.addedLinksView())
	if m.err != nil {
		b.WriteString(errStyle.Render("✕ "+m.err.Error()) + "\n\n")
	}
	b.WriteString(helpStyle.Render("New managed link") + "\n")
	for i, field := range m.fields {
		b.WriteString(m.formFieldLine(i, field) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("Type for a new or missing path") + "\n")
	b.WriteString(m.addLinkChoice(addLinkDirOption, m.linkKind == domain.HarnessLinkKindDir, "Directory") + "\n")
	b.WriteString(m.addLinkChoice(addLinkFileOption, m.linkKind == domain.HarnessLinkKindFile, "File") + "\n")
	b.WriteString("\n" + helpStyle.Render("If the path is an existing symlink") + "\n")
	b.WriteString(m.addLinkChoice(addLinkImportOption, m.linkAction == app.HarnessLinkActionImport, "Import target into this profile") + "\n")
	b.WriteString(m.addLinkChoice(addLinkRegisterOption, m.linkAction == app.HarnessLinkActionRegister, "Keep target external") + "\n\n")
	b.WriteString(m.addLinkButton(addLinkButton, "Add link") + "\n")
	b.WriteString(m.addLinkButton(addLinkRemoveButton, "Remove last link") + "\n")
	b.WriteString(m.addLinkButton(addLinksContinueButton, "Continue") + "\n")
	return m.panel(b.String(), m.footer("tab/↑/↓ move", "space select", "enter activate", "esc previous step", "? help"))
}

func (m Model) addedLinksView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Managed links (%d)", len(m.addDraft.Links))) + "\n")
	if len(m.addDraft.Links) == 0 {
		b.WriteString(helpStyle.Render("  No links yet. Add at least one directory or file.") + "\n\n")
		return b.String()
	}
	for _, link := range m.addDraft.Links {
		fmt.Fprintf(&b, "  %s  %s\n", okStyle.Render(fmt.Sprintf("%s (%s)", link.ID, link.Kind)), fitMiddle(link.Path, m.descriptionWidth()))
	}
	b.WriteString("\n")
	return b.String()
}

func (m Model) addLinkChoice(focus int, selected bool, label string) string {
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

func (m Model) addLinkButton(focus int, label string) string {
	if m.field == focus {
		return "  " + selectStyle.Render("› "+label)
	}
	return "  " + buttonStyle.Render(label)
}
