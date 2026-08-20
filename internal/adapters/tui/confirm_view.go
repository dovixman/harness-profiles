package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewConfirm() string {
	title := "△ Preview"
	body := renderPreviewBody(m.confirm)
	if m.op == opDeleteHarness || m.op == opDeleteProfile || m.op == opDeleteLink {
		title = "⚠ Destructive preview"
		body = renderDestructivePreviewBody(m.confirm)
	}
	header := titleStyle.Render(title)
	if subtitle := m.confirmSubtitle(); subtitle != "" {
		header += "\n" + helpStyle.Render(subtitle)
	}
	return m.panel(header+"\n\n"+body+"\n\n"+m.confirmButtons(), m.footer(m.confirmFooterHints()...))
}

func (m Model) confirmButtons() string {
	primary := "Confirm"
	if m.op == opAdd && m.addNeedsProfile() && !m.addDraft.ImportApproved {
		primary = "Next"
	}
	secondary := "Cancel"
	if m.cloneStep {
		primary = "Set active"
		secondary = "Keep current"
	}
	buttons := []string{primary, secondary}
	for i, label := range buttons {
		if i == m.confirmBtn {
			buttons[i] = confirmButtonStyle.Render("› " + label)
		} else {
			buttons[i] = buttonStyle.Render(label)
		}
	}
	return "  " + strings.Join(buttons, "  ")
}

func (m Model) confirmFooterHints() []string {
	if m.cloneStep {
		return []string{"tab buttons", "enter choose", "y active", "n keep current", "esc cancel", "? help"}
	}
	if m.op == opAdd && m.addNeedsProfile() && !m.addDraft.ImportApproved {
		return []string{"tab buttons", "enter next", "y next", "n cancel", "esc back", "? help"}
	}
	return []string{"tab buttons", "enter confirm", "y confirm", "n cancel", "esc back", "? help"}
}

func (m Model) confirmSubtitle() string {
	if m.cloneStep {
		return "Choose whether to switch after clone"
	}
	if m.op != opAdd {
		return ""
	}
	if m.addNeedsProfile() && !m.addDraft.ImportApproved {
		return "Step 2/3: review detected config root"
	}
	if m.addNeedsProfile() {
		return "Final confirmation"
	}
	return "Step 2/2: confirm changes"
}

func renderPreviewBody(value string) string {
	var b strings.Builder
	for _, line := range strings.Split(value, "\n") {
		b.WriteString(renderPreviewLine(line))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderDestructivePreviewBody(value string) string {
	var b strings.Builder
	firstContent := true
	for _, line := range strings.Split(value, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			b.WriteString("\n")
			continue
		}
		if firstContent || strings.HasPrefix(trimmed, "Consequence:") {
			b.WriteString(dangerStyle.Render(line))
		} else {
			b.WriteString(renderPreviewLine(line))
		}
		firstContent = false
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderPreviewLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		label := trimmed[:idx+1]
		rest := strings.TrimSpace(trimmed[idx+1:])
		value := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Render(rest)
		if rest == "" {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Render(label)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Render(label) + " " + value
	}
	return line
}
