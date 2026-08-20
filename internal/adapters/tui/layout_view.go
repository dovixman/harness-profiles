package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewProgress() string {
	return m.panel(fmt.Sprintf("%s %s\n\n%s", m.spin.View(), m.busy, helpStyle.Render("Applying through internal/app services...")), m.footer("working", "ctrl+c cancel"))
}

func (m Model) viewResult() string {
	if m.err != nil {
		return m.panel(errStyle.Render("✕ Error")+"\n\n"+m.err.Error(), m.footer("enter dashboard", "ctrl+c quit", "? help"))
	}
	content := okStyle.Render(m.message)
	if m.hasExplicitLinks() {
		content += "\n\n" + helpStyle.Render("Managed links") + "\n" + strings.Join(m.managedLinkLines(), "\n")
	}
	return m.panel(content, m.footer("enter dashboard", "ctrl+c quit", "? help"))
}

func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("? Help") + "\n\n")
	b.WriteString("Navigation\n")
	b.WriteString(helpStyle.Render("  arrows move  ·  enter opens/runs selected item  ·  esc back/cancel  ·  ctrl+c quit") + "\n\n")
	b.WriteString("Dashboard\n")
	b.WriteString(helpStyle.Render("  / filters harnesses  ·  esc exits search  ·  enter opens harness/action  ·  u updates selected harness  ·  d/backspace/delete deletes it") + "\n\n")
	b.WriteString("Harness detail\n")
	b.WriteString(helpStyle.Render("  / filters profiles  ·  enter opens the selected link, profile, or action  ·  u updates  ·  c clones profiles  ·  d/backspace/delete removes") + "\n\n")
	b.WriteString("Forms and previews\n")
	b.WriteString(helpStyle.Render("  tab/shift+tab move fields  ·  space choose option  ·  y confirm  ·  n cancel  ·  previews list every managed link"))
	return m.panel(b.String(), m.footer("esc back", "enter back", "ctrl+c quit"))
}

func (m Model) viewTooSmall() string {
	content := titleStyle.Render("▣ Harness Profiles") + "\n\n" + warnStyle.Render("⚠ Terminal too small") + "\n\n" + helpStyle.Render(fmt.Sprintf("Current: %dx%d. Minimum: 60x18. Widen the pane or use a larger terminal.", m.width, m.height))
	return panelStyle.Render(content) + "\n"
}

func (m Model) panel(content, footer string) string {
	width := m.width - 4
	if width < 56 {
		width = 56
	}
	if footer != "" {
		content = content + "\n\n" + m.footerSeparator() + "\n" + footer
	}
	return panelStyle.Width(width).Render(content) + "\n"
}

func (m Model) footer(parts ...string) string {
	width := m.footerWidth()
	lines := make([]string, 0, 2)
	line := ""
	for _, part := range parts {
		candidate := part
		if line != "" {
			candidate = line + " · " + part
		}
		if line != "" && lipgloss.Width(candidate) > width {
			lines = append(lines, line)
			line = part
			continue
		}
		line = candidate
	}
	if line != "" {
		lines = append(lines, line)
	}
	return helpStyle.Faint(true).Render(strings.Join(lines, "\n"))
}

func (m Model) footerSeparator() string {
	width := m.width - 12
	if width < 32 {
		width = 32
	}
	return helpStyle.Faint(true).Render(strings.Repeat("─", width))
}

func (m Model) footerWidth() int {
	width := m.width - 12
	if width < 32 {
		return 32
	}
	return width
}

func (m Model) formValueWidth() int {
	width := m.width - 42
	if width < 18 {
		return 18
	}
	if width > 56 {
		return 56
	}
	return width
}

func (m Model) tooSmall() bool {
	return m.width > 0 && m.height > 0 && (m.width < 60 || m.height < 18)
}

func (m Model) descriptionWidth() int {
	width := m.width - 38
	if width < 22 {
		return 22
	}
	if width > 80 {
		return 80
	}
	return width
}
