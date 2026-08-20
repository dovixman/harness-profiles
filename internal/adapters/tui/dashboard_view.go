package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewHarnesses() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("▣ Harness Profiles") + "\n")
	b.WriteString(helpStyle.Render("Dashboard menu. Press / to filter harnesses; enter opens harness.") + "\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("Harnesses: %d/%d", len(m.filteredHarnessIndexes()), len(m.harnesses))) + "\n\n")
	if m.hSearching || m.hQuery != "" {
		fmt.Fprintf(&b, "⌕ search: %s\n\n", m.hSearchText())
	}
	if m.err != nil {
		b.WriteString(errStyle.Render("✕ "+m.err.Error()) + "\n")
	}
	m.renderMenuSections(&b, dashboardMenuSections(m.dashboardSections()), m.menu, "▸")
	return m.panel(b.String(), m.footer(m.dashboardFooterHints()...))
}

func dashboardMenuSections(sections []dashboardSection) []menuViewSection {
	views := make([]menuViewSection, 0, len(sections))
	for _, section := range sections {
		items := make([]menuViewItem, 0, len(section.Items))
		for _, item := range section.Items {
			items = append(items, menuViewItem{Icon: item.Icon, Label: item.Label, Description: item.Description, Kind: item.Kind})
		}
		views = append(views, menuViewSection{Title: section.Title, Items: items})
	}
	return views
}

func (m Model) dashboardFooterHints() []string {
	if m.hSearching {
		return []string{"type filter", "backspace delete", "enter apply", "esc cancel", "ctrl+c quit"}
	}
	if m.hQuery != "" {
		return []string{"↑/↓ move", "/ edit search", "enter open", "u update", "d delete", "esc clear"}
	}
	return []string{"↑/↓ move", "/ search", "enter open", "u update", "d delete", "ctrl+c quit"}
}

func (m Model) hSearchText() string {
	if m.hSearching {
		return m.hDraft
	}
	return m.hQuery
}

func (m Model) dashboardItems() []dashboardItem {
	var items []dashboardItem
	for _, section := range m.dashboardSections() {
		for _, item := range section.Items {
			if item.Kind != itemKindNoop {
				items = append(items, item)
			}
		}
	}
	return items
}

func (m Model) dashboardSections() []dashboardSection {
	harnesses := []dashboardItem{}
	idxs := m.filteredHarnessIndexes()
	if len(idxs) > 0 {
		for row, idx := range idxs {
			h := m.harnesses[idx]
			label := strings.TrimSpace(h.Label)
			if label == "" {
				label = h.ID
			}
			harnesses = append(harnesses, dashboardItem{Icon: "◇", Label: label, Description: harnessLinkSummary(h, m.descriptionWidth()), Kind: itemKindHarness, Harness: row})
		}
	} else if m.hQuery != "" {
		harnesses = append(harnesses, dashboardItem{Icon: "⚠", Label: "No harnesses match", Description: "clear the search or add one", Kind: itemKindNoop})
	} else {
		harnesses = append(harnesses, dashboardItem{Icon: "◇", Label: "No harnesses yet", Description: "start by adding one", Kind: itemKindNoop})
	}
	actions := []dashboardItem{
		{Icon: "✚", Label: "Add harness", Description: "create a new managed harness", Kind: itemKindAdd},
		{Icon: "✕", Label: "Quit", Description: "exit hp", Kind: itemKindQuit},
	}
	return []dashboardSection{{Title: "Harnesses", Items: harnesses}, {Title: "Actions", Items: actions}}
}
