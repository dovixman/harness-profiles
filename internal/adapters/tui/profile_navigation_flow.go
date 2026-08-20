package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateDetail(key tea.KeyMsg) Model {
	if m.pSearching {
		return m.updateProfileSearch(key)
	}
	switch key.String() {
	case keyEsc:
		if m.pQuery != "" {
			m.pQuery = ""
			m.profile = clamp(m.profile, len(m.filteredProfileIndexes()))
			m.detailMenu = clamp(m.detailMenu, len(m.detailItems()))
			return m
		}
		m.screen = screenHarnesses
	case "up", "k":
		m.detailMenu = max(0, m.detailMenu-1)
	case keyDown, "j":
		m.detailMenu = min(len(m.detailItems())-1, m.detailMenu+1)
	case "home":
		m.detailMenu = 0
	case "end", "G":
		m.detailMenu = max(0, len(m.detailItems())-1)
	case "/":
		m.pSearching = true
		m.pDraft = m.pQuery
		m.detailMenu = 0
	case keyEnter:
		m.activateDetailItem()
	case "s":
		m.startHighlightedProfileOperation(opSwitch)
	case "u":
		m.startHighlightedDetailUpdate()
	case "c":
		m.startHighlightedProfileOperation(opClone)
	case "d", keyBackspace, "delete":
		m.startHighlightedDetailDelete()
	}
	return m
}

func (m Model) updateProfileSearch(key tea.KeyMsg) Model {
	switch key.String() {
	case keyEsc:
		m.pSearching = false
		m.pDraft = m.pQuery
		m.profile = clamp(m.profile, len(m.filteredProfileIndexes()))
		m.detailMenu = clamp(m.detailMenu, len(m.detailItems()))
	case keyEnter:
		m.pQuery = m.pDraft
		m.pSearching = false
	case keyBackspace:
		if m.pDraft != "" {
			m.pDraft = strings.TrimSuffix(m.pDraft, lastRune(m.pDraft))
			m.profile = clamp(m.profile, len(m.filteredProfileIndexes()))
			m.detailMenu = clamp(m.detailMenu, len(m.detailItems()))
		}
	default:
		if len(key.Runes) > 0 {
			m.pDraft += string(key.Runes)
			m.profile = 0
			m.detailMenu = 0
		}
	}
	return m
}

func (m *Model) activateDetailItem() {
	items := m.detailItems()
	if len(items) == 0 {
		return
	}
	item := items[clamp(m.detailMenu, len(items))]
	switch item.Kind {
	case itemKindLink:
		m.link = item.Link
		m.startProfileLinkForm(opUpdateLink)
	case "profile":
		m.profile = item.Profile
		m.startProfileConfirm(opSwitch)
	case "create-profile":
		m.startForm(opCreateProfile)
	case "add-link":
		m.startProfileLinkForm(opAddLink)
	case "update-harness":
		m.startForm(opUpdate)
	case "delete-harness":
		m.startForm(opDeleteHarness)
	case "back":
		m.pQuery = ""
		m.screen = screenHarnesses
	}
}

func (m *Model) startHighlightedDetailUpdate() {
	item, ok := m.highlightedDetailItem()
	if !ok {
		return
	}
	if item.Kind == itemKindLink {
		m.link = item.Link
		m.startProfileLinkForm(opUpdateLink)
		return
	}
	if item.Kind == "profile" {
		m.startHighlightedProfileOperation(opRenameProfile)
	}
}

func (m *Model) startHighlightedDetailDelete() {
	item, ok := m.highlightedDetailItem()
	if !ok {
		return
	}
	if item.Kind == itemKindLink {
		m.link = item.Link
		m.startDeleteProfileLink()
		return
	}
	if item.Kind == "profile" {
		m.startHighlightedProfileOperation(opDeleteProfile)
	}
}

func (m Model) highlightedDetailItem() (detailItem, bool) {
	items := m.detailItems()
	if len(items) == 0 {
		return detailItem{}, false
	}
	return items[clamp(m.detailMenu, len(items))], true
}

func (m *Model) startHighlightedProfileOperation(op operation) {
	items := m.detailItems()
	if len(items) == 0 {
		return
	}
	item := items[clamp(m.detailMenu, len(items))]
	if item.Kind != "profile" {
		return
	}
	m.profile = item.Profile
	m.startProfileSelection(op)
}
