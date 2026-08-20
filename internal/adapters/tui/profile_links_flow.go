package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func (m *Model) startProfileLinkForm(op operation) {
	m.op = op
	m.err = nil
	m.field = 0
	m.linkKind = domain.HarnessLinkKindDir
	m.linkAction = app.HarnessLinkActionImport
	if op == opAddLink {
		m.fields = []formField{newFormField("Link ID", "state"), newFormField("Path", "~/.config/tool.json")}
	} else {
		link, ok := m.selectedProfileLink()
		if !ok {
			m.resultErr(fmt.Errorf("no managed link selected"))
			return
		}
		m.fields = []formField{newFormField("Path", link.Path)}
		m.fields[0].Input.SetValue(link.Path)
		m.linkKind = link.Kind
	}
	m.focusField()
	m.screen = screenProfileLink
}

func (m *Model) startDeleteProfileLink() {
	link, ok := m.selectedProfileLink()
	if !ok {
		m.resultErr(fmt.Errorf("no managed link selected"))
		return
	}
	if len(m.managedLinks()) == 1 {
		m.resultErr(fmt.Errorf("cannot remove the only managed link"))
		return
	}
	m.op = opDeleteLink
	m.confirmBtn = 0
	m.confirm = m.deleteProfileLinkPreview(link)
	m.screen = screenConfirm
}

func (m Model) updateProfileLink(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.String() == keyEsc {
		m.err = nil
		m.screen = screenDetail
		return m, nil
	}
	total := m.profileLinkFocusCount()
	switch key.String() {
	case "tab", keyDown:
		m.field = wrap(m.field+1, total)
		m.focusField()
		return m, nil
	case keyShiftTab, "up":
		m.field = wrap(m.field-1, total)
		m.focusField()
		return m, nil
	case " ":
		m.selectProfileLinkOption()
		return m, nil
	case keyEnter:
		if m.field == total-1 {
			return m.submitProfileLink(), nil
		}
		m.selectProfileLinkOption()
		m.field = wrap(m.field+1, total)
		m.focusField()
		return m, nil
	}
	if m.field >= len(m.fields) {
		return m, nil
	}
	var cmd tea.Cmd
	m.fields[m.field].Input, cmd = m.fields[m.field].Input.Update(key)
	m.err = nil
	return m, cmd
}

func (m Model) profileLinkFocusCount() int {
	if m.op == opAddLink {
		return len(m.fields) + 5
	}
	return len(m.fields) + 1
}

func (m *Model) selectProfileLinkOption() {
	if m.op != opAddLink {
		return
	}
	switch m.field - len(m.fields) {
	case 0:
		m.linkKind = domain.HarnessLinkKindDir
	case 1:
		m.linkKind = domain.HarnessLinkKindFile
	case 2:
		m.linkAction = app.HarnessLinkActionImport
	case 3:
		m.linkAction = app.HarnessLinkActionRegister
	}
}

func (m Model) submitProfileLink() Model {
	link, err := m.profileLinkDraft()
	if err != nil {
		m.err = err
		return m
	}
	if err := m.validateProfileLinkDraft(link); err != nil {
		m.err = err
		return m
	}
	m.err = nil
	m.confirmBtn = 0
	m.confirm = m.profileLinkPreview(link)
	m.screen = screenConfirm
	return m
}

func (m Model) profileLinkDraft() (domain.HarnessLink, error) {
	if m.op == opAddLink {
		path, err := app.NormalizeConfigRootPath(field(&m, 1))
		if err != nil {
			return domain.HarnessLink{}, err
		}
		link := domain.HarnessLink{ID: field(&m, 0), Path: path, Kind: m.linkKind}
		return link, link.Validate()
	}
	selected, ok := m.selectedProfileLink()
	if !ok {
		return domain.HarnessLink{}, fmt.Errorf("no managed link selected")
	}
	path, err := app.NormalizeConfigRootPath(field(&m, 0))
	if err != nil {
		return domain.HarnessLink{}, err
	}
	selected.Path = path
	return selected, selected.Validate()
}

func (m Model) validateProfileLinkDraft(candidate domain.HarnessLink) error {
	for index, existing := range m.managedLinks() {
		if m.op == opUpdateLink && index == m.link {
			continue
		}
		if strings.EqualFold(existing.ID, candidate.ID) {
			return fmt.Errorf("link ID %q is already managed", candidate.ID)
		}
		if existing.Path == candidate.Path {
			return fmt.Errorf("path %q is already managed", candidate.Path)
		}
	}
	return nil
}

func (m Model) profileLinkUpdateOptions() app.UpdateHarnessOptions {
	links := append([]domain.HarnessLink(nil), m.managedLinks()...)
	opts := app.UpdateHarnessOptions{ID: m.harness.ID}
	switch m.op {
	case opAddLink:
		link, _ := m.profileLinkDraft()
		links = append(links, link)
		opts.LinkActions = map[string]app.HarnessLinkAction{strings.ToLower(link.ID): m.linkAction}
	case opUpdateLink:
		link, _ := m.profileLinkDraft()
		links[m.link] = link
		opts.RemoveOld = true
	case opDeleteLink:
		links = append(links[:m.link:m.link], links[m.link+1:]...)
		opts.RemoveOld = true
	}
	opts.Links = links
	return opts
}

func (m Model) selectedProfileLink() (domain.HarnessLink, bool) {
	links := m.managedLinks()
	if m.link < 0 || m.link >= len(links) {
		return domain.HarnessLink{}, false
	}
	return links[m.link], true
}
