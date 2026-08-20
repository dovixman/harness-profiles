package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dovixman/harness-profiles/internal/app"
)

func (m Model) executeCmd() tea.Cmd {
	return func() tea.Msg {
		var err error
		switch m.op {
		case opAdd:
			_, err = m.service.AddHarness(m.addOptions())
			m.message = "✓ harness added"
		case opUpdate:
			var opts app.UpdateHarnessOptions
			opts, err = m.updateOptions()
			if err == nil {
				_, err = m.service.UpdateHarness(opts)
			}
			m.message = "✓ harness updated"
		case opDeleteHarness:
			err = m.service.DeleteHarness(app.DeleteHarnessOptions{ID: m.harness.ID, Mode: m.deleteMode, RestoreProfile: m.deleteRestoreProfile(), Confirm: m.deleteConfirmation()})
			m.message = "✓ harness deleted"
		case opSwitch:
			idxs := m.filteredProfileIndexes()
			_, err = m.service.SwitchProfile(m.harness.ID, m.profiles[idxs[m.profile]].Name)
			m.message = "✓ profile switched"
		case opAdopt:
			err = m.service.AdoptProfile(m.harness.ID, field(&m, 0))
			m.message = "✓ profile adopted"
		case opCreateProfile:
			err = m.service.CreateProfile(m.harness.ID, field(&m, 0))
			m.message = "✓ profile created"
		case opRenameProfile:
			err = m.service.RenameProfile(m.harness.ID, m.highlightedProfileName(), field(&m, 0))
			m.message = "✓ profile updated"
		case opClone:
			err = m.service.CloneProfile(m.harness.ID, m.highlightedProfileName(), field(&m, 0), m.selectedOption() == "materialize")
			if err == nil && m.cloneActive {
				_, err = m.service.SwitchProfile(m.harness.ID, field(&m, 0))
			}
			m.message = "✓ profile cloned"
			if m.cloneActive {
				m.message = "✓ profile cloned and switched"
			}
		case opDeleteProfile:
			idxs := m.filteredProfileIndexes()
			err = m.service.DeleteProfile(m.harness.ID, m.profiles[idxs[m.profile]].Name, true)
			m.message = "✓ profile deleted"
		case opAddLink, opUpdateLink, opDeleteLink:
			_, err = m.service.UpdateHarness(m.profileLinkUpdateOptions())
			m.message = "✓ managed link updated"
			if m.op == opAddLink {
				m.message = "✓ managed link added"
			}
			if m.op == opDeleteLink {
				m.message = "✓ managed link removed"
			}
		}
		if err != nil {
			return opResultMsg{err: err}
		}
		return opResultMsg{message: m.message}
	}
}

func (m *Model) execute() {
	cmd := m.executeCmd()
	if msg := cmd(); msg != nil {
		if res, ok := msg.(opResultMsg); ok {
			if res.err != nil {
				m.resultErr(res.err)
				return
			}
			m.message = res.message
		}
	}
	m.screen = screenResult
}

func (m Model) selectedOption() string {
	for _, opt := range m.options {
		if opt.Checked {
			return opt.Value
		}
	}
	return ""
}
