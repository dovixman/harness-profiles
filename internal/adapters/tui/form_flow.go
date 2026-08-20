package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func (m Model) updateForm(key tea.KeyMsg) (Model, tea.Cmd) {
	totalFocus := len(m.fields) + len(m.options) + 1
	switch key.String() {
	case keyEsc:
		m.screen = screenDetail
		if m.op == opAdd || m.harness.ID == "" {
			m.screen = screenHarnesses
		}
		return m, nil
	case "tab", keyShiftTab, "up", keyDown, keyEnter:
		return m.moveFormFocus(key, totalFocus)
	case " ":
		if m.focusedOption() >= 0 {
			m.checkFocusedOption()
		}
		return m, nil
	}
	if m.field >= len(m.fields) {
		return m, nil
	}
	return m.updateFocusedField(key)
}

func (m Model) moveFormFocus(key tea.KeyMsg, totalFocus int) (Model, tea.Cmd) {
	if key.String() == keyEnter && m.canSubmitFocusedFormControl() {
		return m.submitForm(), nil
	}
	if key.String() == "up" || key.String() == keyShiftTab {
		m.field--
	} else {
		m.field++
	}
	m.field = wrap(m.field, totalFocus)
	if idx := m.focusedOption(); idx >= 0 {
		m.option = idx
		m.checkFocusedOption()
	}
	m.focusField()
	return m, nil
}

func (m Model) canSubmitFocusedFormControl() bool {
	return m.field == len(m.fields) || m.focusedOption() >= 0
}

func (m Model) updateFocusedField(key tea.KeyMsg) (Model, tea.Cmd) {
	before := m.fields[m.field].Input.Value()
	var cmd tea.Cmd
	m.fields[m.field].Input, cmd = m.fields[m.field].Input.Update(key)
	if m.op == opAdd && len(m.fields) == 3 && m.field == 1 && before != m.fields[m.field].Input.Value() {
		m.syncDefaultRestartHint(before)
	}
	return m, cmd
}

func (m *Model) syncDefaultRestartHint(previousLabel string) {
	if len(m.fields) < 3 {
		return
	}
	previousDefault := defaultRestartHint(previousLabel)
	currentRestart := strings.TrimSpace(m.fields[2].Input.Value())
	if currentRestart == "" || currentRestart == previousDefault {
		m.fields[2].Input.SetValue(defaultRestartHint(m.fields[1].Input.Value()))
	}
}

func (m *Model) focusOption(idx int) {
	m.option = idx
	m.field = len(m.fields) + 1 + idx
	m.focusField()
}

func (m *Model) checkFocusedOption() {
	if m.focusedOption() < 0 {
		return
	}
	for i := range m.options {
		m.options[i].Checked = i == m.option
	}
}

func (m Model) focusedOption() int {
	idx := m.field - len(m.fields) - 1
	if idx < 0 || idx >= len(m.options) {
		return -1
	}
	return idx
}

func (m *Model) startForm(op operation) {
	m.op = op
	m.field = 0
	m.option = 0
	m.options = nil
	m.fields = nil
	m.cloneStep = false
	m.cloneActive = false
	m.deleteMode = ""
	if op == opAdd {
		m.addDraft = addHarnessDraft{}
	}
	for _, spec := range fieldSpecs(op) {
		input := textinput.New()
		input.Placeholder = spec.Hint
		input.Prompt = ""
		m.fields = append(m.fields, formField{Label: spec.Label, Hint: spec.Hint, Input: input})
	}
	m.configureFormDefaults(op)
	m.focusField()
	m.screen = screenForm
}

func (m *Model) configureFormDefaults(op operation) {
	if op == opAdd {
		m.fields[2].Input.SetValue(defaultRestartHint(""))
	}
	if op == opClone {
		m.options = []optionItem{{Label: "Preserve symlinks", Value: "preserve", Checked: true}, {Label: "Materialize symlinks", Value: "materialize"}}
	}
	if op == opRenameProfile {
		if name := m.highlightedProfileName(); name != "" {
			m.fields[0].Input.SetValue(name)
		}
	}
	if op == opUpdate {
		m.fields[0].Input.SetValue(m.harness.Label)
		m.fields[1].Input.SetValue(m.harness.RestartHint)
	}
	if op == opDeleteHarness {
		if m.hasExplicitLinks() {
			m.options = []optionItem{{Label: "Keep managed links", Value: deleteModeKeepRoot, Checked: true}, {Label: "Restore profile into managed links", Value: deleteModeRestore}, {Label: "Delete managed links", Value: deleteModeDeleteAll}}
		} else {
			m.options = []optionItem{{Label: "Keep root symlink", Value: deleteModeKeepRoot, Checked: true}, {Label: "Restore profile into root", Value: deleteModeRestore}, {Label: "Delete managed root", Value: deleteModeDeleteAll}}
		}
		m.field = len(m.fields) + 1
	}
}

func (m *Model) focusField() {
	for i := range m.fields {
		if i == m.field {
			m.fields[i].Input.Focus()
		} else {
			m.fields[i].Input.Blur()
		}
	}
}

type fieldSpec struct{ Label, Hint string }

func fieldSpecs(op operation) []fieldSpec {
	switch op {
	case opAdd:
		return []fieldSpec{{"Harness ID", "opencode"}, {"Label", "OpenCode"}, {"Restart hint", "Restart <Label> so it re-reads config from the new path"}}
	case opUpdate:
		return []fieldSpec{{"Label", "Display name"}, {"Restart hint", "Optional restart note"}}
	case opDeleteHarness:
		return nil
	case opAdopt:
		return []fieldSpec{{"New profile name", "work"}}
	case opCreateProfile:
		return []fieldSpec{{"Profile name", "work"}}
	case opRenameProfile:
		return []fieldSpec{{"New profile name", "work"}}
	case opClone:
		return []fieldSpec{{"Target profile", "experiment"}}
	}
	return nil
}

func field(m *Model, idx int) string {
	if idx >= len(m.fields) {
		return ""
	}
	return strings.TrimSpace(m.fields[idx].Input.Value())
}

func (m Model) addOptions() app.AddHarnessOptions {
	draft := m.addDraft
	profile := ""
	if draft.Branch == addBranchMissing || draft.Branch == addBranchDirectory || draft.Branch == addBranchFile || draft.Branch == addBranchSymlink {
		profile = draft.ProfileName
		if len(m.fields) == 1 {
			profile = field(&m, 0)
		}
	}
	options := app.AddHarnessOptions{ID: draft.ID, Label: draft.Label, LinkPath: draft.ConfigPath, Links: append([]domain.HarnessLink(nil), draft.Links...), LinkActions: draft.LinkActions, RestartHint: draft.RestartHint, InitialProfile: profile, ImportSymlink: draft.Branch == addBranchSymlink}
	if len(draft.Links) > 0 {
		options.LinkPath = ""
	}
	return options
}

func (m Model) updateOptions() (app.UpdateHarnessOptions, error) {
	return app.UpdateHarnessOptions{ID: m.harness.ID, Label: field(&m, 0), RestartHint: field(&m, 1)}, nil
}

func defaultRestartHint(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "<Label>"
	}
	return fmt.Sprintf("Restart %s so it re-reads config from the new path", label)
}
