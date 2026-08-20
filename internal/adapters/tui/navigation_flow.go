package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateHarnesses(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.hSearching {
		return m.updateHarnessSearch(key), nil
	}
	switch key.String() {
	case "up", "k", "h":
		m.menu = max(0, m.menu-1)
	case keyDown, "j", "l":
		m.menu = min(len(m.dashboardItems())-1, m.menu+1)
	case "home":
		m.menu = 0
	case "end", "G":
		m.menu = max(0, len(m.dashboardItems())-1)
	case "tab", keyShiftTab:
		m.toggleDashboardSection()
	case "r":
		m.loadHarnesses()
	case "/":
		m.hSearching = true
		m.hDraft = m.hQuery
		m.menu = 0
	case keyEsc:
		if m.hQuery != "" {
			m.hQuery = ""
			m.menu = clamp(m.menu, len(m.dashboardItems()))
		}
	case keyEnter:
		return m.activateDashboardItem()
	case "u":
		m.startHighlightedHarnessForm(opUpdate)
	case "d", keyBackspace, "delete":
		m.startHighlightedHarnessForm(opDeleteHarness)
	}
	return m, nil
}

func (m Model) updateHarnessSearch(key tea.KeyMsg) Model {
	switch key.String() {
	case keyEsc:
		m.hSearching = false
		m.hDraft = m.hQuery
		m.menu = clamp(m.menu, len(m.dashboardItems()))
	case keyEnter:
		m.hQuery = m.hDraft
		m.hSearching = false
	case keyBackspace:
		if m.hDraft != "" {
			m.hDraft = strings.TrimSuffix(m.hDraft, lastRune(m.hDraft))
			m.menu = clamp(m.menu, len(m.dashboardItems()))
		}
	default:
		if len(key.Runes) > 0 {
			m.hDraft += string(key.Runes)
			m.menu = 0
		}
	}
	return m
}

func (m *Model) toggleDashboardSection() {
	items := m.dashboardItems()
	if len(items) == 0 {
		return
	}
	current := items[clamp(m.menu, len(items))]
	if current.Kind == itemKindHarness {
		for i, item := range items {
			if item.Kind != itemKindHarness {
				m.menu = i
				return
			}
		}
		return
	}
	m.menu = 0
}

func (m Model) activateDashboardItem() (Model, tea.Cmd) {
	items := m.dashboardItems()
	if len(items) == 0 {
		return m, nil
	}
	item := items[clamp(m.menu, len(items))]
	switch item.Kind {
	case itemKindAdd:
		m.startForm(opAdd)
	case itemKindHarness:
		m.menu = item.Harness
		m.loadDetail()
	case itemKindQuit:
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) startHighlightedHarnessForm(op operation) {
	items := m.dashboardItems()
	if len(items) == 0 {
		return
	}
	item := items[clamp(m.menu, len(items))]
	if item.Kind != itemKindHarness {
		return
	}
	m.menu = item.Harness
	m.startHarnessSelection(op)
}
