package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "tab", keyShiftTab, "up", keyDown, "k", "j", "left", "right", "h", "l":
		if key.String() == keyShiftTab || key.String() == "up" || key.String() == "k" || key.String() == "left" || key.String() == "h" {
			m.confirmBtn = wrap(m.confirmBtn-1, 2)
		} else {
			m.confirmBtn = wrap(m.confirmBtn+1, 2)
		}
		return m, nil
	case keyEnter:
		if m.cloneStep {
			return m.chooseCloneActivation(m.confirmBtn == 0)
		}
		if m.confirmBtn == 0 {
			return m.confirmOperation()
		}
		return m.cancelConfirm(), nil
	case "y", "Y":
		if m.cloneStep {
			return m.chooseCloneActivation(true)
		}
		return m.confirmOperation()
	case "n", "N":
		if m.cloneStep {
			return m.chooseCloneActivation(false)
		}
		return m.cancelConfirm(), nil
	case keyEsc, keyBackspace:
		return m.cancelConfirm(), nil
	}
	return m, nil
}

func (m Model) chooseCloneActivation(active bool) (tea.Model, tea.Cmd) {
	m.cloneActive = active
	m.cloneStep = false
	m.confirmBtn = 0
	m.confirm = m.clonePreview()
	return m, nil
}

func (m Model) confirmOperation() (tea.Model, tea.Cmd) {
	if m.op == opAdd && m.addNeedsImportPlan() && !m.addDraft.ImportApproved {
		m.addDraft.ImportApproved = true
		m.startAddProfileNameForm()
		return m, nil
	}
	m.screen = screenProgress
	m.busy = operationProgress(m.op)
	return m, tea.Batch(m.spin.Tick, m.executeCmd())
}

func (m Model) cancelConfirm() Model {
	m.screen = screenDetail
	if m.op == opAdd || m.harness.ID == "" {
		m.screen = screenHarnesses
	}
	return m
}
