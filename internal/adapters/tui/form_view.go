package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewForm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("✎ "+operationTitle(m.op)) + "\n")
	if subtitle := m.formSubtitle(); subtitle != "" {
		b.WriteString(helpStyle.Render(subtitle) + "\n\n")
	}
	if context := m.formContext(); context != "" {
		b.WriteString(context + "\n")
	}
	if m.err != nil {
		b.WriteString(errStyle.Render("✕ "+m.err.Error()) + "\n\n")
	}
	if len(m.fields) > 0 {
		b.WriteString(helpStyle.Render("Fields") + "\n")
		for i, f := range m.fields {
			b.WriteString(m.formFieldLine(i, f) + "\n")
		}
		b.WriteString("\n")
	}
	if m.shouldShowOptionsBeforeButton() {
		b.WriteString(m.formOptions())
		b.WriteString("\n" + m.formPrimaryButton() + "\n")
	} else {
		b.WriteString(m.formPrimaryButton() + "\n")
		b.WriteString(m.formOptions())
	}
	return m.panel(b.String(), m.footer(m.formFooterHints()...))
}

func (m Model) shouldShowOptionsBeforeButton() bool {
	return m.op == opDeleteHarness && len(m.options) > 0
}

func (m Model) formFieldLine(i int, f formField) string {
	label := f.Label + requiredMark(m.op, i)
	value := formFieldValue(f, m.formValueWidth())
	if i == m.field {
		return selectStyle.Render("› "+fmt.Sprintf("%-22s", label)) + " " + wrapFieldValue(f.Input.View(), m.formValueWidth())
	}
	return fmt.Sprintf("  %-22s %s", label, wrapFieldValue(value, m.formValueWidth()))
}

func (m Model) formPrimaryButton() string {
	if m.field == len(m.fields) {
		return "  " + selectStyle.Render("› Next")
	}
	return "  " + buttonStyle.Render("Next")
}

func (m Model) formOptions() string {
	if len(m.options) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n" + titleStyle.Render(m.formOptionsTitle()) + " " + helpStyle.Render("move onto an option to choose it") + "\n")
	for i, opt := range m.options {
		b.WriteString(m.formOptionLine(i, opt) + "\n")
	}
	return b.String()
}

func (m Model) formOptionsTitle() string {
	if m.op == opDeleteHarness && m.deleteMode == deleteModeRestore {
		return "Choose profile to restore"
	}
	if m.op == opDeleteHarness && m.deleteMode == "" {
		if m.hasExplicitLinks() {
			return "Choose managed link mode"
		}
		return "Choose delete mode"
	}
	return "Options"
}

func (m Model) formOptionLine(i int, opt optionItem) string {
	box := "○"
	if opt.Checked {
		box = "●"
	}
	line := fmt.Sprintf("%s %s", box, opt.Label)
	if m.field == len(m.fields)+1+i {
		return selectStyle.Render("› " + line)
	}
	if opt.Checked {
		return okStyle.Render("  " + line)
	}
	return "  " + line
}

func (m Model) formFooterHints() []string {
	enterHint := "enter next"
	if m.op != opAdd && m.field == len(m.fields)-1 {
		enterHint = "enter preview"
	}
	if len(m.options) > 0 {
		return []string{"tab/down choose", "shift+tab/up prev", enterHint, "esc cancel", "? help"}
	}
	return []string{"tab/down next", "shift+tab/up prev", enterHint, "esc cancel", "? help"}
}

func (m Model) formSubtitle() string {
	if m.op == opDeleteHarness {
		return m.deleteHarnessSubtitle()
	}
	if m.op == opAdd {
		if len(m.fields) == 1 && m.addNeedsProfile() {
			return "Step 3/3: name the initial profile"
		}
		return "Step 1/3: describe the harness"
	}
	return ""
}

func (m Model) deleteHarnessSubtitle() string {
	switch m.deleteMode {
	case deleteModeRestore:
		if m.hasExplicitLinks() {
			return "Step 2/2: choose which profile to restore into every managed link"
		}
		return "Step 2/2: choose which profile to restore into the config root"
	case deleteModeDeleteAll:
		if m.hasExplicitLinks() {
			return "Step 2/2: Type " + m.deleteAllConfirmationText() + " to confirm deleting the managed links"
		}
		return "Step 2/2: Type " + m.deleteAllConfirmationText() + " to confirm deleting the managed root"
	default:
		if m.hasExplicitLinks() {
			return "Step 1/2: choose what should happen to the managed links"
		}
		return "Step 1/2: choose what should happen to the config root"
	}
}

func (m Model) formContext() string {
	action := m.formActionLabel()
	if action == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(infoRow("Action", action, m.descriptionWidth()+10))
	if m.op == opUpdate || m.op == opDeleteHarness {
		b.WriteString(infoRow("Harness", m.harness.ID, m.descriptionWidth()+10))
		b.WriteString(infoRow("Managed links", m.managedLinkSummary(m.descriptionWidth()+10), m.descriptionWidth()+10))
		if m.op == opDeleteHarness && m.deleteMode != "" {
			b.WriteString(infoRow("Mode", m.deleteMode, m.descriptionWidth()+10))
		}
		return infoStyle.Render(strings.TrimRight(b.String(), "\n")) + "\n"
	}
	name := m.highlightedProfileName()
	if name == "" {
		return ""
	}
	b.WriteString(infoRow("Profile", name, m.descriptionWidth()+10))
	if path := m.highlightedProfilePath(); path != "" {
		b.WriteString(infoRow("Path", path, m.descriptionWidth()+10))
	}
	return infoStyle.Render(strings.TrimRight(b.String(), "\n")) + "\n"
}

func (m Model) formActionLabel() string {
	switch m.op {
	case opUpdate:
		return "Update selected harness"
	case opDeleteHarness:
		return "Delete selected harness"
	case opSwitch:
		return "Switch selected profile"
	case opRenameProfile:
		return "Update selected profile"
	case opClone:
		return "Clone selected profile"
	case opDeleteProfile:
		return "Delete selected profile"
	}
	return ""
}

func (m Model) addImportContext() string {
	draft := m.addDraft
	var b strings.Builder
	if len(draft.Links) == 0 {
		b.WriteString(okStyle.Render("ⓘ Detected config root") + "\n")
		b.WriteString(infoRow("Config root", draft.ConfigPath, m.descriptionWidth()))
		switch draft.Branch {
		case addBranchDirectory:
			fmt.Fprintf(&b, "%-10s %s\n", "State", okStyle.Render("existing directory"))
			b.WriteString(infoRow("Action", "copy this directory into a new managed profile", m.descriptionWidth()))
		case addBranchFile:
			fmt.Fprintf(&b, "%-10s %s\n", "State", okStyle.Render("existing file"))
			b.WriteString(infoRow("Action", "copy this file into a new managed profile", m.descriptionWidth()))
		case addBranchSymlink:
			fmt.Fprintf(&b, "%-10s %s\n", "State", warnStyle.Render("symlink"))
			b.WriteString(infoRow("Symlink", draft.SourcePath, m.descriptionWidth()))
			b.WriteString(infoRow("Action", "copy the symlink target into a new managed profile", m.descriptionWidth()))
		default:
			fmt.Fprintf(&b, "%-10s %s\n", "State", draft.Branch)
		}
	} else {
		b.WriteString(okStyle.Render("ⓘ Detected managed links") + "\n")
		for _, line := range draftLinkPreviewLines(draft.Links) {
			b.WriteString(line + "\n")
		}
	}
	b.WriteString(helpStyle.Render("Next: choose a profile name, then preview symlink replacement.") + "\n")
	return infoStyle.Render(strings.TrimRight(b.String(), "\n")) + "\n"
}

func (m Model) addImportPlanPreview() string {
	return strings.TrimSpace(m.addImportContext()) + "\n\nContinue with this import plan?"
}

func infoRow(label, value string, width int) string {
	return fmt.Sprintf("%-10s %s\n", label, wrapInfoValue(value, width))
}

func wrapInfoValue(value string, width int) string {
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(value)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= 1 {
		return wrapped
	}
	return strings.Join(lines, "\n"+strings.Repeat(" ", 11))
}

func formFieldValue(field formField, width int) string {
	value := strings.TrimSpace(field.Input.Value())
	if value != "" {
		return value
	}
	if field.Hint == "" {
		return helpStyle.Render("<empty>")
	}
	return helpStyle.Render(field.Hint)
}

func requiredMark(op operation, idx int) string {
	switch op {
	case opAdd:
		if idx == 0 || idx == 1 {
			return " *"
		}
	case opAdopt:
		if idx == 0 {
			return " *"
		}
	case opClone:
		if idx == 0 || idx == 1 {
			return " *"
		}
	}
	return ""
}
