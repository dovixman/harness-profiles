package tui

import "fmt"

func (m *Model) startHarnessSelection(op operation) {
	if len(m.harnesses) == 0 {
		m.resultErr(fmt.Errorf("no harnesses configured"))
		return
	}
	m.op = op
	if m.loadHighlightedHarnessContext() {
		m.startForm(op)
	}
}

func (m *Model) loadHighlightedHarnessContext() bool {
	idxs := m.filteredHarnessIndexes()
	if len(idxs) == 0 {
		m.resultErr(fmt.Errorf("no harness selected"))
		return false
	}
	harness, err := m.service.InspectHarness(m.harnesses[idxs[clamp(m.menu, len(idxs))]].ID)
	if err != nil {
		m.resultErr(err)
		return false
	}
	profiles, err := m.service.ListProfiles(harness.ID)
	if err != nil {
		m.resultErr(err)
		return false
	}
	current, _ := m.service.CurrentProfile(harness.ID)
	paths, _ := m.service.Where()
	m.harness = harness
	m.profiles = profiles
	m.current = current
	m.paths = paths
	return true
}

func (m *Model) startProfileSelection(op operation) {
	if m.harness.ID == "" {
		m.loadDetail()
	}
	idxs := m.filteredProfileIndexes()
	if len(idxs) == 0 && op != opAdopt {
		m.resultErr(fmt.Errorf("no profiles available for %s", m.harness.ID))
		return
	}
	m.op = op
	switch op {
	case opSwitch, opDeleteProfile:
		m.startProfileConfirm(op)
	case opClone, opRenameProfile:
		m.startForm(op)
	}
}

func (m *Model) startProfileConfirm(op operation) {
	idxs := m.filteredProfileIndexes()
	if len(idxs) == 0 {
		m.resultErr(fmt.Errorf("no profile selected"))
		return
	}
	m.op = op
	profile := m.profiles[idxs[m.profile]]
	if op == opDeleteProfile && profile.Active {
		m.resultErr(fmt.Errorf("cannot delete active profile %q; switch first", profile.Name))
		return
	}
	switch op {
	case opSwitch:
		m.confirm = m.switchProfilePreview(profile)
	case opDeleteProfile:
		m.confirm = fmt.Sprintf("Delete profile %s?\n\nHarness: %s\nProfile path: %s\nConsequence: permanently deletes this managed profile directory.", profile.Name, m.harness.ID, profile.Path)
	}
	m.screen = screenConfirm
}

func (m Model) highlightedProfileName() string {
	idxs := m.filteredProfileIndexes()
	if len(idxs) == 0 {
		return ""
	}
	return m.profiles[idxs[clamp(m.profile, len(idxs))]].Name
}

func (m Model) highlightedProfilePath() string {
	idxs := m.filteredProfileIndexes()
	if len(idxs) == 0 {
		return ""
	}
	return m.profiles[idxs[clamp(m.profile, len(idxs))]].Path
}
