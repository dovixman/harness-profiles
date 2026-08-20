package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func (m Model) submitForm() Model {
	if m.op == opDeleteHarness {
		return m.submitDeleteHarnessForm()
	}
	if err := m.validateForm(); err != nil {
		m.err = err
		return m
	}
	m.err = nil
	m.confirmBtn = 0
	if m.op == opAdd {
		return m.submitAddForm()
	}
	switch m.op {
	case opUpdate:
		m.confirm = m.updatePreview()
	case opAdopt:
		m.confirm = m.adoptPreview()
	case opCreateProfile:
		m.confirm = m.createProfilePreview()
	case opRenameProfile:
		m.confirm = m.renameProfilePreview()
	case opClone:
		m.cloneStep = true
		m.confirm = m.cloneActivationPrompt()
	default:
		m.confirm = "Confirm operation?"
	}
	m.screen = screenConfirm
	return m
}

func (m Model) submitAddForm() Model {
	if len(m.fields) == 1 {
		m.addDraft.ProfileName = field(&m, 0)
		if err := m.loadPathsForAdd(); err != nil {
			m.err = err
			return m
		}
		m.confirm = m.addPreview()
		m.screen = screenConfirm
		return m
	}
	links := m.addDraft.Links
	actions := m.addDraft.LinkActions
	if actions == nil {
		actions = map[string]app.HarnessLinkAction{}
	}
	m.addDraft = addHarnessDraft{
		ID:          field(&m, 0),
		Label:       field(&m, 1),
		RestartHint: field(&m, 2),
		Links:       links,
		LinkActions: actions,
	}
	m.startAddLinksForm()
	return m
}

func (m *Model) startAddProfileNameForm() {
	m.fields = []formField{}
	input := textinput.New()
	input.Placeholder = "default"
	input.Prompt = ""
	m.fields = append(m.fields, formField{Label: "New profile name", Hint: "default", Input: input})
	m.field = 0
	m.options = nil
	m.focusField()
	m.screen = screenForm
}

func (m Model) addNeedsProfile() bool {
	return m.addDraft.Branch == addBranchMissing || m.addDraft.Branch == addBranchDirectory || m.addDraft.Branch == addBranchFile || m.addDraft.Branch == addBranchSymlink
}

func (m Model) addNeedsImportPlan() bool {
	return m.addDraft.Branch == addBranchDirectory || m.addDraft.Branch == addBranchFile || m.addDraft.Branch == addBranchSymlink
}

func (m *Model) loadPathsForAdd() error {
	paths, err := m.service.Where()
	if err != nil {
		return err
	}
	m.paths = paths
	return nil
}

func (m Model) validateForm() error {
	switch m.op {
	case opAdd:
		if len(m.fields) == 1 {
			if field(&m, 0) == "" {
				return fmt.Errorf("profile name is required")
			}
			return nil
		}
		if field(&m, 0) == "" || field(&m, 1) == "" {
			return fmt.Errorf("harness ID and label are required")
		}
	case opAdopt, opCreateProfile:
		if field(&m, 0) == "" {
			return fmt.Errorf("profile name is required")
		}
	case opRenameProfile:
		if m.highlightedProfileName() == "" || field(&m, 0) == "" {
			return fmt.Errorf("selected and new profile names are required")
		}
	case opClone:
		if m.highlightedProfileName() == "" || field(&m, 0) == "" {
			return fmt.Errorf("selected source and target profiles are required")
		}
	}
	return nil
}

func inspectAddConfigPath(path string) (branch string, sourcePath string, kind domain.HarnessLinkKind, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return addBranchMissing, "", domain.HarnessLinkKindDir, nil
		}
		return "", "", "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", "", "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		targetInfo, err := os.Stat(target)
		if err != nil {
			return "", "", "", err
		}
		if targetInfo.IsDir() {
			return addBranchSymlink, target, domain.HarnessLinkKindDir, nil
		}
		if targetInfo.Mode().IsRegular() {
			return addBranchSymlink, target, domain.HarnessLinkKindFile, nil
		}
		return "", "", "", fmt.Errorf("symlink target %s is neither a regular file nor directory", target)
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() {
			return addBranchFile, path, domain.HarnessLinkKindFile, nil
		}
		return "", "", "", fmt.Errorf("config root path must be a directory, file, or symlink")
	}
	return addBranchDirectory, path, domain.HarnessLinkKindDir, nil
}

func inspectAddManagedLinks(links []domain.HarnessLink, legacySingle bool) (branch string, sourcePath string, kind domain.HarnessLinkKind, err error) {
	if len(links) == 0 {
		return addBranchMissing, "", domain.HarnessLinkKindDir, nil
	}
	if legacySingle && len(links) == 1 {
		return inspectAddConfigPath(links[0].Path)
	}
	branch = addBranchMissing
	for _, link := range links {
		b, src, linkKind, err := inspectAddConfigPath(link.Path)
		if err != nil {
			return "", "", "", err
		}
		if b == addBranchSymlink {
			return addBranchSymlink, src, linkKind, nil
		}
		if b == addBranchFile {
			branch = addBranchFile
			kind = linkKind
			if sourcePath == "" {
				sourcePath = src
			}
			continue
		}
		if b == addBranchDirectory {
			branch = addBranchDirectory
			if sourcePath == "" {
				sourcePath = src
			}
		}
	}
	if branch == addBranchMissing {
		kind = domain.HarnessLinkKindDir
	}
	return branch, sourcePath, kind, nil
}
