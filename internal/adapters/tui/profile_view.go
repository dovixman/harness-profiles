package tui

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (m Model) viewDetail() string {
	var b strings.Builder
	b.WriteString(m.detailHeader() + "\n\n")
	b.WriteString(titleStyle.Render("Profiles & managed links") + "\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("Profiles: %d/%d", len(m.filteredProfileIndexes()), len(m.profiles))) + "\n")
	if m.pSearching || m.pQuery != "" {
		fmt.Fprintf(&b, "⌕ search: %s\n", m.pSearchText())
	}
	m.renderMenuSections(&b, detailMenuSections(m.detailSections()), m.detailMenu, "›")
	return m.panel(b.String(), m.footer(m.profileFooterHints()...))
}

func detailMenuSections(sections []detailSection) []menuViewSection {
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

func (m Model) profileFooterHints() []string {
	if m.pSearching {
		return []string{"type filter", "backspace delete", "enter apply", "esc cancel", "ctrl+c quit"}
	}
	item, _ := m.highlightedDetailItem()
	if m.pQuery != "" {
		if item.Kind == itemKindLink {
			return []string{"↑/↓ move", "/ edit search", "enter/u update link", "d remove link", "esc clear"}
		}
		return []string{"↑/↓ move", "/ edit search", "enter/s switch", "u update", "c clone", "d delete", "esc clear"}
	}
	if item.Kind == itemKindLink {
		return []string{"↑/↓ move", "/ search", "enter/u update link", "d remove link", "esc back"}
	}
	if item.Kind != "profile" {
		return []string{"↑/↓ move", "/ search", "enter activate", "esc back"}
	}
	return []string{"↑/↓ move", "/ search", "enter/s switch", "u update", "c clone", "d delete", "esc back"}
}

func (m Model) pSearchText() string {
	if m.pSearching {
		return m.pDraft
	}
	return m.pQuery
}

func (m Model) detailHeader() string {
	var b strings.Builder
	store := filepath.Join(m.paths.HarnessesRoot, m.harness.ID)
	b.WriteString(titleStyle.Render("◆ "+m.harness.ID) + "\n")
	fmt.Fprintf(&b, "%s %s\n", keyStyle.Render("Managed links"), fitMiddle(m.managedLinkSummary(m.descriptionWidth()+10), m.descriptionWidth()+10))
	fmt.Fprintf(&b, "%s %s\n", keyStyle.Render("Profile store"), fitMiddle(store, m.descriptionWidth()+10))
	fmt.Fprintf(&b, "%s %s", keyStyle.Render("Current"), m.currentStateLine(m.descriptionWidth()+10))
	return headerStyle.Render(b.String())
}

func (m Model) detailItems() []detailItem {
	var items []detailItem
	for _, section := range m.detailSections() {
		for _, item := range section.Items {
			if item.Kind != itemKindNoop {
				items = append(items, item)
			}
		}
	}
	return items
}

func (m Model) detailSections() []detailSection {
	links := m.managedLinkItems()
	profiles := []detailItem{}
	idxs := m.filteredProfileIndexes()
	if len(idxs) > 0 {
		for row, idx := range idxs {
			p := m.profiles[idx]
			icon := "○"
			label := p.Name
			if p.Active {
				icon = "●"
				label += "  active"
			}
			profiles = append(profiles, detailItem{Icon: icon, Label: label, Description: p.Path, Kind: "profile", Profile: row})
		}
	} else if m.pQuery != "" {
		profiles = append(profiles, detailItem{Icon: "⚠", Label: "No profiles match", Description: "clear the search or create one", Kind: itemKindNoop})
	} else {
		profiles = append(profiles, detailItem{Icon: "◇", Label: "No profiles yet", Description: "create or clone one", Kind: itemKindNoop})
	}
	actions := []detailItem{
		{Icon: "✚", Label: "Add managed link", Description: "add a path to every profile", Kind: "add-link"},
		{Icon: "✚", Label: "Create profile", Description: "new empty managed profile", Kind: "create-profile"},
	}
	sections := []detailSection{}
	if len(links) > 0 {
		sections = append(sections, detailSection{Title: "Managed links", Items: links})
	}
	sections = append(sections, detailSection{Title: "Profiles", Items: profiles}, detailSection{Title: "Actions", Items: actions})
	return sections
}
