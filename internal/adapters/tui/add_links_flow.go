package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

const (
	addLinkIDField = iota
	addLinkPathField
	addLinkDirOption
	addLinkFileOption
	addLinkImportOption
	addLinkRegisterOption
	addLinkButton
	addLinkRemoveButton
	addLinksContinueButton
	addLinksFocusCount
)

func (m *Model) startAddLinksForm() {
	m.fields = []formField{
		newFormField("Link ID", "root"),
		newFormField("Path", "~/.config/tool"),
	}
	m.fields[0].Input.SetValue(domain.LegacyDefaultLinkID)
	m.field = addLinkIDField
	m.linkKind = domain.HarnessLinkKindDir
	m.linkAction = app.HarnessLinkActionImport
	m.focusField()
	m.screen = screenAddLinks
}

func newFormField(label, hint string) formField {
	input := textinput.New()
	input.Placeholder = hint
	input.Prompt = ""
	return formField{Label: label, Hint: hint, Input: input}
}

func (m Model) updateAddLinks(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.String() {
	case keyEsc:
		m.returnToAddDetails()
		return m, nil
	case "tab", keyDown:
		m.field = wrap(m.field+1, addLinksFocusCount)
		m.focusField()
		return m, nil
	case keyShiftTab, "up":
		m.field = wrap(m.field-1, addLinksFocusCount)
		m.focusField()
		return m, nil
	case " ":
		m.selectAddLinkOption()
		return m, nil
	case keyEnter:
		return m.activateAddLinkControl(), nil
	}
	if m.field >= len(m.fields) {
		return m, nil
	}
	before := m.fields[m.field].Input.Value()
	var cmd tea.Cmd
	m.fields[m.field].Input, cmd = m.fields[m.field].Input.Update(key)
	if m.field == addLinkPathField && before != m.fields[m.field].Input.Value() {
		m.err = nil
	}
	return m, cmd
}

func (m *Model) returnToAddDetails() {
	draft := m.addDraft
	m.startForm(opAdd)
	m.addDraft = draft
	m.fields[0].Input.SetValue(draft.ID)
	m.fields[1].Input.SetValue(draft.Label)
	m.fields[2].Input.SetValue(draft.RestartHint)
}

func (m *Model) selectAddLinkOption() {
	switch m.field {
	case addLinkDirOption:
		m.linkKind = domain.HarnessLinkKindDir
	case addLinkFileOption:
		m.linkKind = domain.HarnessLinkKindFile
	case addLinkImportOption:
		m.linkAction = app.HarnessLinkActionImport
	case addLinkRegisterOption:
		m.linkAction = app.HarnessLinkActionRegister
	}
}

func (m Model) activateAddLinkControl() Model {
	switch m.field {
	case addLinkIDField, addLinkPathField:
		m.field++
		m.focusField()
	case addLinkDirOption, addLinkFileOption, addLinkImportOption, addLinkRegisterOption:
		m.selectAddLinkOption()
		m.field++
		m.focusField()
	case addLinkButton:
		return m.addDraftLink()
	case addLinkRemoveButton:
		return m.removeLastDraftLink()
	case addLinksContinueButton:
		return m.continueAddHarness()
	}
	return m
}

func (m Model) removeLastDraftLink() Model {
	if len(m.addDraft.Links) == 0 {
		m.err = fmt.Errorf("there are no managed links to remove")
		return m
	}
	removed := m.addDraft.Links[len(m.addDraft.Links)-1]
	m.addDraft.Links = m.addDraft.Links[:len(m.addDraft.Links)-1]
	delete(m.addDraft.LinkActions, strings.ToLower(removed.ID))
	m.err = nil
	return m
}

func (m Model) addDraftLink() Model {
	id := field(&m, addLinkIDField)
	path, err := app.NormalizeConfigRootPath(field(&m, addLinkPathField))
	if err != nil {
		m.err = err
		return m
	}
	link := domain.HarnessLink{ID: id, Path: path, Kind: m.linkKind}
	if err := link.Validate(); err != nil {
		m.err = err
		return m
	}
	for _, existing := range m.addDraft.Links {
		if strings.EqualFold(existing.ID, link.ID) {
			m.err = fmt.Errorf("link ID %q is already added", link.ID)
			return m
		}
		if existing.Path == link.Path {
			m.err = fmt.Errorf("path %q is already managed", link.Path)
			return m
		}
	}
	branch, _, detectedKind, err := inspectAddConfigPath(link.Path)
	if err != nil {
		m.err = err
		return m
	}
	if branch != addBranchMissing {
		link.Kind = detectedKind
	}
	m.addDraft.Links = append(m.addDraft.Links, link)
	m.addDraft.LinkActions[strings.ToLower(link.ID)] = m.linkAction
	m.fields[0] = newFormField("Link ID", "state")
	m.fields[1] = newFormField("Path", "~/.config/tool.json")
	m.field = addLinkIDField
	m.linkKind = domain.HarnessLinkKindDir
	m.linkAction = app.HarnessLinkActionImport
	m.err = nil
	m.focusField()
	return m
}

func (m Model) continueAddHarness() Model {
	if len(m.addDraft.Links) == 0 {
		m.err = fmt.Errorf("add at least one managed link")
		return m
	}
	if field(&m, addLinkIDField) != "" || field(&m, addLinkPathField) != "" {
		m.err = fmt.Errorf("add the current link or clear its fields before continuing")
		return m
	}
	m.addDraft.ConfigPath = m.addDraft.Links[0].Path
	branch, sourcePath, _, err := inspectAddManagedLinks(m.addDraft.Links, false)
	if err != nil {
		m.err = err
		return m
	}
	m.addDraft.Branch = branch
	m.addDraft.SourcePath = sourcePath
	if err := m.loadPathsForAdd(); err != nil {
		m.err = err
		return m
	}
	if branch == addBranchMissing {
		m.addDraft.ImportApproved = true
		m.startAddProfileNameForm()
		return m
	}
	m.confirm = m.addImportPlanPreview()
	m.screen = screenConfirm
	return m
}
