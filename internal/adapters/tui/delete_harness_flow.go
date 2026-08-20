package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
)

func (m Model) submitDeleteHarnessForm() Model {
	if m.deleteMode == "" {
		return m.submitDeleteHarnessMode()
	}
	if err := m.validateDeleteHarnessDetails(); err != nil {
		m.err = err
		return m
	}
	m.err = nil
	m.confirm = m.deleteHarnessPreview()
	m.screen = screenConfirm
	return m
}

func (m Model) submitDeleteHarnessMode() Model {
	m.deleteMode = m.selectedOption()
	m.confirmBtn = 0
	switch m.deleteMode {
	case deleteModeKeepRoot:
		m.confirm = m.deleteHarnessPreview()
		m.screen = screenConfirm
	case deleteModeRestore:
		m.startDeleteRestoreForm()
	case deleteModeDeleteAll:
		m.startDeleteAllForm()
	default:
		m.err = fmt.Errorf("delete mode is required")
	}
	return m
}

func (m *Model) startDeleteRestoreForm() {
	m.fields = nil
	m.options = make([]optionItem, 0, len(m.profiles))
	for i, profile := range m.profiles {
		m.options = append(m.options, optionItem{Label: profile.Name, Value: profile.Name, Checked: i == 0})
	}
	m.field = len(m.fields) + 1
	m.option = 0
	m.err = nil
	m.screen = screenForm
}

func (m *Model) startDeleteAllForm() {
	input := textinput.New()
	input.Placeholder = m.deleteAllConfirmationText()
	input.Prompt = ""
	m.fields = []formField{{Label: "Typed confirmation", Hint: "Type " + m.deleteAllConfirmationText() + " to confirm", Input: input}}
	m.options = nil
	m.field = 0
	m.err = nil
	m.focusField()
	m.screen = screenForm
}

func (m Model) validateDeleteHarnessDetails() error {
	switch m.deleteMode {
	case deleteModeRestore:
		if m.selectedOption() == "" {
			return fmt.Errorf("restore profile is required")
		}
	case deleteModeDeleteAll:
		if field(&m, 0) != m.deleteAllConfirmationText() {
			return fmt.Errorf("type %s to confirm delete-all mode", m.deleteAllConfirmationText())
		}
	}
	return nil
}

func (m Model) deleteAllConfirmationText() string {
	return m.harness.ID
}

func (m Model) deleteRestoreProfile() string {
	if m.deleteMode == deleteModeRestore {
		return m.selectedOption()
	}
	return ""
}

func (m Model) deleteConfirmation() string {
	if m.deleteMode == deleteModeDeleteAll {
		return field(&m, 0)
	}
	return ""
}
