package tui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

type mockService struct {
	harnesses []domain.Harness
	profiles  []app.ProfileStatus
	current   app.ProfileStatus
	paths     app.Paths
	switched  string
	created   string
	renamed   [2]string
	added     app.AddHarnessOptions
	deleted   app.DeleteHarnessOptions
	updated   app.UpdateHarnessOptions
}

func TestManagedLinkStatusRenderingCoversRecoveryStates(t *testing.T) {
	statuses := []struct {
		state app.ProfileLinkState
		data  app.ProfileLinkStatus
		want  string
	}{
		{state: app.ProfileStateActive, data: app.ProfileLinkStatus{State: app.ProfileStateActive}, want: "active"},
		{state: app.ProfileStateExternal, data: app.ProfileLinkStatus{State: app.ProfileStateExternal}, want: "external"},
		{state: app.ProfileStateMissing, data: app.ProfileLinkStatus{State: app.ProfileStateMissing, Profile: "work"}, want: "profile work"},
		{state: app.ProfileStateMixed, data: app.ProfileLinkStatus{State: app.ProfileStateMixed}, want: "mixed"},
		{state: app.ProfileStateUnknown, data: app.ProfileLinkStatus{State: app.ProfileStateUnknown}, want: "unknown"},
		{data: app.ProfileLinkStatus{LinkPath: "/tmp/root"}, want: "/tmp/root"},
	}
	for _, tt := range statuses {
		if icon := profileLinkStateIcon(tt.state); icon == "" {
			t.Fatalf("state %q rendered an empty icon", tt.state)
		}
		if description := profileLinkDescription(tt.data, 40); !strings.Contains(description, tt.want) {
			t.Fatalf("state %q description = %q, want %q", tt.state, description, tt.want)
		}
	}

	m := NewModel(newMockService())
	states := []struct {
		status app.ProfileStatus
		want   string
	}{
		{status: app.ProfileStatus{State: app.ProfileStateActive}, want: "active"},
		{status: app.ProfileStatus{State: app.ProfileStateExternal}, want: "external"},
		{status: app.ProfileStatus{State: app.ProfileStateMixed}, want: "mixed"},
		{status: app.ProfileStatus{State: app.ProfileStateMissing}, want: "missing"},
		{status: app.ProfileStatus{State: app.ProfileStateUnknown}, want: "unknown"},
		{status: app.ProfileStatus{External: true, Path: "/tmp/external"}, want: "/tmp/external"},
		{status: app.ProfileStatus{Name: "legacy"}, want: "legacy"},
		{status: app.ProfileStatus{}, want: "No active managed profile"},
	}
	for _, tt := range states {
		m.current = tt.status
		if line := m.currentStateLine(40); !strings.Contains(line, tt.want) {
			t.Fatalf("status = %+v line = %q, want %q", tt.status, line, tt.want)
		}
	}
}

func TestAddLinkEditorKeyboardNavigationAndPartialDraft(t *testing.T) {
	m := NewModel(newMockService())
	m.startForm(opAdd)
	m.fields[0].Input.SetValue("claude")
	m.fields[1].Input.SetValue("Claude")
	m = m.submitForm()

	updated, _ := m.Update(key("tab"))
	m = updated.(Model)
	if m.field != addLinkPathField {
		t.Fatalf("field = %d, want path after tab", m.field)
	}
	updated, _ = m.Update(key(keyShiftTab))
	m = updated.(Model)
	if m.field != addLinkIDField {
		t.Fatalf("field = %d, want link ID after shift+tab", m.field)
	}
	updated, _ = m.Update(key(keyDown))
	m = updated.(Model)
	updated, _ = m.Update(key("up"))
	m = updated.(Model)
	if m.field != addLinkIDField {
		t.Fatalf("field = %d, want up/down navigation to round-trip", m.field)
	}
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.field != addLinkPathField {
		t.Fatalf("field = %d, want Enter to advance from link ID", m.field)
	}
	updated, _ = m.Update(key("x"))
	m = updated.(Model)
	if !strings.HasSuffix(m.fields[1].Input.Value(), "x") {
		t.Fatalf("path = %q, want typed input", m.fields[1].Input.Value())
	}
	m.fields[1].Input.SetValue("")

	m.field = addLinkDirOption
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.linkKind != domain.HarnessLinkKindDir || m.field != addLinkFileOption {
		t.Fatalf("kind = %q field = %d, want selected directory and next control", m.linkKind, m.field)
	}
	m.field = addLinkImportOption
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.linkAction != app.HarnessLinkActionImport || m.field != addLinkRegisterOption {
		t.Fatalf("action = %q field = %d, want selected import and next control", m.linkAction, m.field)
	}

	m.fields[0].Input.SetValue("root")
	m.fields[1].Input.SetValue(filepath.Join(t.TempDir(), "claude"))
	m = m.addDraftLink()
	m.fields[0].Input.SetValue("state")
	m.field = addLinksContinueButton
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil || !strings.Contains(m.err.Error(), "current link") {
		t.Fatalf("err = %v, want partial-link guidance", m.err)
	}
}

func TestLegacyAddImportContextExplainsDetectedArtifact(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		sourcePath string
		want       string
	}{
		{name: "directory", branch: addBranchDirectory, want: "existing directory"},
		{name: "file", branch: addBranchFile, want: "existing file"},
		{name: "symlink", branch: addBranchSymlink, sourcePath: "/tmp/target", want: "copy the symlink target"},
		{name: "missing", branch: addBranchMissing, want: addBranchMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(newMockService())
			m.width = 80
			m.addDraft = addHarnessDraft{ConfigPath: "/tmp/root", Branch: tt.branch, SourcePath: tt.sourcePath}

			if got := m.addImportContext(); !strings.Contains(got, tt.want) {
				t.Fatalf("context = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddLinkEditorSupportsKeyboardSelectionAndBack(t *testing.T) {
	m := NewModel(newMockService())
	m.startForm(opAdd)
	m.fields[0].Input.SetValue("claude")
	m.fields[1].Input.SetValue("Claude")
	m.fields[2].Input.SetValue("Restart Claude")
	m = m.submitForm()

	m.field = addLinkFileOption
	updated, _ := m.Update(key(" "))
	m = updated.(Model)
	if m.linkKind != domain.HarnessLinkKindFile {
		t.Fatalf("link kind = %q, want file", m.linkKind)
	}
	m.field = addLinkRegisterOption
	updated, _ = m.Update(key(" "))
	m = updated.(Model)
	if m.linkAction != app.HarnessLinkActionRegister {
		t.Fatalf("link action = %q, want register", m.linkAction)
	}

	m.fields[1].Input.SetValue(filepath.Join(t.TempDir(), "state.json"))
	m.field = addLinkButton
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if len(m.addDraft.Links) != 1 || m.addDraft.Links[0].Kind != domain.HarnessLinkKindFile {
		t.Fatalf("draft = %+v, want one guided file link", m.addDraft)
	}

	updated, _ = m.Update(key(keyEsc))
	m = updated.(Model)
	if m.screen != screenForm || field(&m, 0) != "claude" || len(m.addDraft.Links) != 1 {
		t.Fatalf("screen = %v draft = %+v, want preserved details and links", m.screen, m.addDraft)
	}
	m = m.submitForm()
	if m.screen != screenAddLinks || len(m.addDraft.Links) != 1 {
		t.Fatalf("screen = %v draft = %+v, want to resume link editing", m.screen, m.addDraft)
	}
}

func TestAddLinkEditorExplainsInvalidAndDuplicateLinks(t *testing.T) {
	m := NewModel(newMockService())
	m.startForm(opAdd)
	m.fields[0].Input.SetValue("claude")
	m.fields[1].Input.SetValue("Claude")
	m = m.submitForm()

	m.field = addLinksContinueButton
	updated, _ := m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil || !strings.Contains(m.err.Error(), "at least one") {
		t.Fatalf("err = %v, want missing-link guidance", m.err)
	}

	m.field = addLinkButton
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil {
		t.Fatal("want an error for an empty link path")
	}

	path := filepath.Join(t.TempDir(), "claude")
	m.fields[1].Input.SetValue(path)
	m.field = addLinkButton
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if len(m.addDraft.Links) != 1 {
		t.Fatalf("draft = %+v, want one link", m.addDraft)
	}

	m.fields[0].Input.SetValue("root")
	m.fields[1].Input.SetValue(filepath.Join(t.TempDir(), "other"))
	m.field = addLinkButton
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil || !strings.Contains(m.err.Error(), "already added") {
		t.Fatalf("err = %v, want duplicate ID guidance", m.err)
	}

	m.fields[0].Input.SetValue("other")
	m.fields[1].Input.SetValue(path)
	m.field = addLinkButton
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil || !strings.Contains(m.err.Error(), "already managed") {
		t.Fatalf("err = %v, want duplicate path guidance", m.err)
	}

	m.field = addLinkRemoveButton
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if len(m.addDraft.Links) != 0 || m.err != nil {
		t.Fatalf("draft = %+v err = %v, want last link removed", m.addDraft, m.err)
	}
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil || !strings.Contains(m.err.Error(), "no managed links") {
		t.Fatalf("err = %v, want empty-list removal guidance", m.err)
	}
}

func TestProfileFooterSpellsOutActionsAndWraps(t *testing.T) {
	m := NewModel(newMockService())
	m.width = 60
	m.loadDetail()

	view := m.View()
	for _, action := range []string{"enter/u update link", "d remove link"} {
		if !strings.Contains(view, action) {
			t.Fatalf("view = %q, want explicit %q footer action", view, action)
		}
	}
	if strings.Contains(view, "c clone") {
		t.Fatalf("view = %q, should not show profile-only actions for a selected link", view)
	}
	if footer := m.footer(m.profileFooterHints()...); strings.Count(footer, "\n") > 1 {
		t.Fatalf("footer = %q, want explicit actions in at most two lines", footer)
	}
}

func (s *mockService) Where() (app.Paths, error) { return s.paths, nil }
func (s *mockService) ListHarnesses() ([]domain.Harness, error) {
	return append([]domain.Harness(nil), s.harnesses...), nil
}
func (s *mockService) InspectHarness(id string) (domain.Harness, error) {
	for _, harness := range s.harnesses {
		if harness.ID == id {
			return harness, nil
		}
	}
	return domain.Harness{}, errors.New("missing harness")
}
func (s *mockService) ListProfiles(string) ([]app.ProfileStatus, error) {
	return append([]app.ProfileStatus(nil), s.profiles...), nil
}
func (s *mockService) CurrentProfile(string) (app.ProfileStatus, error) { return s.current, nil }
func (s *mockService) AddHarness(opts app.AddHarnessOptions) (domain.Harness, error) {
	s.added = opts
	return domain.Harness{ID: opts.ID, Label: opts.Label, LinkPath: opts.LinkPath, Links: append([]domain.HarnessLink(nil), opts.Links...)}, nil
}
func (s *mockService) UpdateHarness(opts app.UpdateHarnessOptions) (domain.Harness, error) {
	s.updated = opts
	return domain.Harness{ID: opts.ID, Label: opts.Label, LinkPath: opts.LinkPath, Links: append([]domain.HarnessLink(nil), opts.Links...)}, nil
}
func (s *mockService) DeleteHarness(opts app.DeleteHarnessOptions) error {
	s.deleted = opts
	return nil
}
func (s *mockService) SwitchProfile(_ string, name string) (domain.Harness, error) {
	s.switched = name
	return domain.Harness{}, nil
}
func (s *mockService) AdoptProfile(string, string) error         { return nil }
func (s *mockService) CreateProfile(_ string, name string) error { s.created = name; return nil }
func (s *mockService) RenameProfile(_, oldName, newName string) error {
	s.renamed = [2]string{oldName, newName}
	return nil
}
func (s *mockService) CloneProfile(string, string, string, bool) error { return nil }
func (s *mockService) DeleteProfile(string, string, bool) error        { return nil }

func TestHarnessSelectionLoadsDetail(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.menu = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	if got.screen != screenDetail {
		t.Fatalf("screen = %v, want detail", got.screen)
	}
	view := got.View()
	if !strings.Contains(view, "Current") || !strings.Contains(view, "default  active") {
		t.Fatalf("detail view = %q, want current profile marker", view)
	}
	if !strings.Contains(view, "Actions") || !strings.Contains(view, "Create profile") || strings.Contains(view, "Update profile") || strings.Contains(view, "Delete profile") || strings.Contains(view, "Switch profile") || strings.Contains(view, "Clone profile") {
		t.Fatalf("detail view = %q, want create-only profile action menu", view)
	}
	if strings.Contains(view, "Update harness") || strings.Contains(view, "Delete harness") {
		t.Fatalf("detail view = %q, should not include harness actions", view)
	}
}

func TestDetailShowsManagedLinksForMultiLinkHarness(t *testing.T) {
	service := newMockService()
	service.harnesses[0].Links = []domain.HarnessLink{
		{ID: domain.LegacyDefaultLinkID, Path: "/tmp/claude", Kind: domain.HarnessLinkKindDir},
		{ID: "state", Path: "/tmp/claude.json", Kind: domain.HarnessLinkKindFile},
	}
	service.current = app.ProfileStatus{
		State: app.ProfileStateMixed,
		Links: []app.ProfileLinkStatus{
			{ID: domain.LegacyDefaultLinkID, Kind: domain.HarnessLinkKindDir, LinkPath: "/tmp/claude", State: app.ProfileStateActive, Profile: "default", ArtifactPath: "/tmp/harnesses/claude/default/root"},
			{ID: "state", Kind: domain.HarnessLinkKindFile, LinkPath: "/tmp/claude.json", State: app.ProfileStateExternal, Target: "/tmp/shared/claude.json"},
		},
	}
	m := NewModel(service)
	m.loadDetail()

	view := m.View()
	if !strings.Contains(view, "Managed links") || !strings.Contains(view, "root (dir)") || !strings.Contains(view, "state (file)") || !strings.Contains(view, "active") || !strings.Contains(view, "external") {
		t.Fatalf("detail view = %q, want multi-link details and link states", view)
	}
}

func TestResultScreenShowsManagedLinksForMultiLinkHarness(t *testing.T) {
	service := newMockService()
	service.harnesses[0].Links = []domain.HarnessLink{
		{ID: domain.LegacyDefaultLinkID, Path: "/tmp/claude", Kind: domain.HarnessLinkKindDir},
		{ID: "state", Path: "/tmp/claude.json", Kind: domain.HarnessLinkKindFile},
	}
	m := NewModel(service)
	m.loadDetail()
	m.message = "✓ profile switched"
	m.screen = screenResult

	view := m.View()
	if !strings.Contains(view, "Managed links") || !strings.Contains(view, "root (dir)") || !strings.Contains(view, "claude.json") {
		t.Fatalf("result view = %q, want managed-link summary", view)
	}
}

func TestHelpScreenOpensAndReturns(t *testing.T) {
	m := NewModel(newMockService())

	updated, _ := m.Update(key("?"))
	m = updated.(Model)
	if m.screen != screenHelp || !strings.Contains(m.View(), "? Help") {
		t.Fatalf("screen = %v view = %q, want help", m.screen, m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.screen != screenHarnesses {
		t.Fatalf("screen = %v, want dashboard after help", m.screen)
	}
}

func TestSmallTerminalShowsFloorMessage(t *testing.T) {
	m := NewModel(newMockService())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 12})
	m = updated.(Model)

	if !strings.Contains(m.View(), "Terminal too small") || !strings.Contains(m.View(), "Minimum: 60x18") {
		t.Fatalf("view = %q, want terminal floor message", m.View())
	}
}

func TestEmptyDashboardRendersSelectableMenu(t *testing.T) {
	service := newMockService()
	service.harnesses = nil
	m := NewModel(service)
	view := m.View()

	if !strings.Contains(view, "▸ ✚ Add harness") || !strings.Contains(view, "◇ No harnesses yet") || strings.Contains(view, "Press a to add one") {
		t.Fatalf("dashboard view = %q, want selectable menu empty state", view)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenForm || m.op != opAdd {
		t.Fatalf("screen = %v op = %v, want add form from selected menu item", m.screen, m.op)
	}
}

func TestDashboardQuitActionExits(t *testing.T) {
	service := newMockService()
	service.harnesses = nil
	m := NewModel(service)
	m.menu = 1

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil, want quit command")
	}
}

func TestSwitchProfileConfirmationUsesService(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.profile = 1
	m.detailMenu = 2

	updated, _ := m.Update(key("s"))
	m = updated.(Model)
	if m.screen != screenConfirm || !strings.Contains(m.confirm, "work") {
		t.Fatalf("confirm = %q screen = %v, want switch confirmation", m.confirm, m.screen)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.screen != screenProgress {
		t.Fatalf("screen = %v cmd nil = %v, want progress command", m.screen, cmd == nil)
	}
	m.execute()
	if service.switched != "work" {
		t.Fatalf("switched = %q, want work", service.switched)
	}
	if m.screen != screenResult || !strings.Contains(m.View(), "profile switched") {
		t.Fatalf("result view = %q, want success", m.View())
	}
}

func TestEnterSwitchesHighlightedProfile(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.detailMenu = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenConfirm || !strings.Contains(m.confirm, "work") {
		t.Fatalf("screen = %v confirm = %q, want enter to switch focused profile", m.screen, m.confirm)
	}
}

func TestDetailMenuCreatesProfile(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.detailMenu = 4

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenForm || m.op != opCreateProfile || !strings.Contains(m.View(), "Create Profile") {
		t.Fatalf("screen = %v op = %v view = %q, want create profile form", m.screen, m.op, m.View())
	}
	m.fields[0].Input.SetValue("scratch")
	m = m.submitForm()
	if m.screen != screenConfirm || !strings.Contains(m.confirm, "Create profile scratch") {
		t.Fatalf("screen = %v confirm = %q, want create profile preview", m.screen, m.confirm)
	}
	m.execute()
	if service.created != "scratch" || m.screen != screenResult {
		t.Fatalf("created = %q screen = %v, want executed create profile", service.created, m.screen)
	}
}

func TestProfileDetailAddsManagedLink(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.detailMenu = 3

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenProfileLink || m.op != opAddLink || !strings.Contains(m.View(), "Add Managed Link") {
		t.Fatalf("screen = %v op = %v view = %q, want add-link form", m.screen, m.op, m.View())
	}
	path := filepath.Join(t.TempDir(), "state.json")
	m.fields[0].Input.SetValue("state")
	m.fields[1].Input.SetValue(path)
	m.linkKind = domain.HarnessLinkKindFile
	m.linkAction = app.HarnessLinkActionImport
	m.field = m.profileLinkFocusCount() - 1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenConfirm || !strings.Contains(m.confirm, "Add managed link state") || !strings.Contains(m.confirm, "default/state") || !strings.Contains(m.confirm, "work/state") {
		t.Fatalf("screen = %v confirm = %q, want profile-aware add preview", m.screen, m.confirm)
	}
	m.execute()
	if len(service.updated.Links) != 2 || service.updated.Links[1].ID != "state" || service.updated.LinkActions["state"] != app.HarnessLinkActionImport {
		t.Fatalf("updated opts = %+v, want added state link", service.updated)
	}
}

func TestProfileDetailUpdatesManagedLinkPath(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.detailMenu = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenProfileLink || m.op != opUpdateLink {
		t.Fatalf("screen = %v op = %v, want update-link form", m.screen, m.op)
	}
	path := filepath.Join(t.TempDir(), "new-root")
	m.fields[0].Input.SetValue(path)
	m.field = m.profileLinkFocusCount() - 1
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !strings.Contains(m.confirm, "Old path: /tmp/claude") || !strings.Contains(m.confirm, "New path: "+path) {
		t.Fatalf("confirm = %q, want old and new paths", m.confirm)
	}
	m.execute()
	if len(service.updated.Links) != 1 || service.updated.Links[0].Path != path || !service.updated.RemoveOld {
		t.Fatalf("updated opts = %+v, want repointed root link", service.updated)
	}
}

func TestProfileDetailRemovesManagedLinkAndPreservesArtifacts(t *testing.T) {
	service := newMockService()
	service.harnesses[0].LinkPath = ""
	service.harnesses[0].Links = []domain.HarnessLink{
		{ID: "root", Path: "/tmp/claude", Kind: domain.HarnessLinkKindDir},
		{ID: "state", Path: "/tmp/claude.json", Kind: domain.HarnessLinkKindFile},
	}
	m := NewModel(service)
	m.loadDetail()
	m.detailMenu = 1

	updated, _ := m.Update(key("d"))
	m = updated.(Model)
	if m.screen != screenConfirm || m.op != opDeleteLink || !strings.Contains(m.confirm, "Existing profile artifacts named state are preserved") {
		t.Fatalf("screen = %v op = %v confirm = %q, want safe delete-link preview", m.screen, m.op, m.confirm)
	}
	m.execute()
	if len(service.updated.Links) != 1 || service.updated.Links[0].ID != "root" || !service.updated.RemoveOld {
		t.Fatalf("updated opts = %+v, want state link removed", service.updated)
	}
}

func TestProfileLinkFormSupportsKeyboardOptionsValidationAndCancel(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.startProfileLinkForm(opAddLink)

	for _, selection := range []struct {
		focus int
		check func(Model) bool
	}{
		{focus: 2, check: func(m Model) bool { return m.linkKind == domain.HarnessLinkKindDir }},
		{focus: 3, check: func(m Model) bool { return m.linkKind == domain.HarnessLinkKindFile }},
		{focus: 4, check: func(m Model) bool { return m.linkAction == app.HarnessLinkActionImport }},
		{focus: 5, check: func(m Model) bool { return m.linkAction == app.HarnessLinkActionRegister }},
	} {
		m.field = selection.focus
		updated, _ := m.Update(key(" "))
		m = updated.(Model)
		if !selection.check(m) {
			t.Fatalf("focus %d did not select expected link option", selection.focus)
		}
	}

	m.field = m.profileLinkFocusCount() - 1
	updated, _ := m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil || m.screen != screenProfileLink {
		t.Fatalf("err = %v screen = %v, want required-field validation", m.err, m.screen)
	}
	m.fields[0].Input.SetValue("root")
	m.fields[1].Input.SetValue(filepath.Join(t.TempDir(), "other"))
	updated, _ = m.Update(key(keyEnter))
	m = updated.(Model)
	if m.err == nil || !strings.Contains(m.err.Error(), "already managed") {
		t.Fatalf("err = %v, want duplicate-link validation", m.err)
	}

	updated, _ = m.Update(key(keyEsc))
	m = updated.(Model)
	if m.screen != screenDetail || m.err != nil {
		t.Fatalf("screen = %v err = %v, want cancel back to detail", m.screen, m.err)
	}
}

func TestProfileLinkActionsRejectMissingSelectionAndOnlyLinkRemoval(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.link = 99
	m.startProfileLinkForm(opUpdateLink)
	if m.screen != screenResult || m.err == nil || !strings.Contains(m.err.Error(), "no managed link selected") {
		t.Fatalf("screen = %v err = %v, want missing selection error", m.screen, m.err)
	}

	m.loadDetail()
	m.link = 0
	m.startDeleteProfileLink()
	if m.screen != screenResult || m.err == nil || !strings.Contains(m.err.Error(), "only managed link") {
		t.Fatalf("screen = %v err = %v, want last-link refusal", m.screen, m.err)
	}
}

func TestProfileFormsUseSelectedProfileInsteadOfOpenSourceField(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.profile = 1

	m.startForm(opClone)
	cloneView := m.View()
	if strings.Contains(cloneView, "Source profile") || !strings.Contains(cloneView, "Target profile") || !strings.Contains(cloneView, "Clone selected profile") || !strings.Contains(cloneView, "work") {
		t.Fatalf("clone form = %q, want selected source profile and target field only", cloneView)
	}
	m.fields[0].Input.SetValue("copy")
	m = m.submitForm()
	if !m.cloneStep || !strings.Contains(m.confirm, "Set cloned profile copy as active") {
		t.Fatalf("confirm = %q cloneStep = %v, want clone activation choice", m.confirm, m.cloneStep)
	}
	updated, _ := m.Update(key("n"))
	m = updated.(Model)
	if !strings.Contains(m.confirm, "Clone profile work to copy") {
		t.Fatalf("confirm = %q, want selected source profile in clone preview", m.confirm)
	}

	m.startForm(opRenameProfile)
	renameView := m.View()
	if strings.Contains(renameView, "Current profile") || !strings.Contains(renameView, "New profile name") || !strings.Contains(renameView, "Update selected profile") || !strings.Contains(renameView, "work") {
		t.Fatalf("rename form = %q, want selected current profile and new-name field only", renameView)
	}
}

func startGuidedAdd(m Model, id, label, path string, kind domain.HarnessLinkKind, action app.HarnessLinkAction) Model {
	m.startForm(opAdd)
	m.fields[0].Input.SetValue(id)
	m.fields[1].Input.SetValue(label)
	m.fields[2].Input.SetValue("Restart " + label + " so it re-reads config from the new path")
	m = m.submitForm()
	m.fields[0].Input.SetValue("root")
	m.fields[1].Input.SetValue(path)
	m.linkKind = kind
	m.linkAction = action
	m = m.addDraftLink()
	return m.continueAddHarness()
}

func TestAddHarnessMissingConfigPathAsksForInitialProfile(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m = startGuidedAdd(m, "opencode", "OpenCode", filepath.Join(t.TempDir(), "missing-config"), domain.HarnessLinkKindDir, app.HarnessLinkActionImport)

	if m.screen != screenForm || m.addDraft.Branch != "missing" || !strings.Contains(m.View(), "Step 3/3: name the initial profile") || !strings.Contains(m.View(), "New profile name") {
		t.Fatalf("screen = %v draft = %+v view = %q, want initial profile form", m.screen, m.addDraft, m.View())
	}
	m.fields[0].Input.SetValue("default")
	m = m.submitForm()
	if strings.Contains(m.View(), "\x1b[31m") || strings.Contains(m.View(), "Destructive preview") {
		t.Fatalf("view = %q, add preview should not be styled as destructive", m.View())
	}
	m.execute()
	if service.added.ID != "opencode" || len(service.added.Links) != 1 || service.added.LinkPath != "" || service.added.InitialProfile != "default" || service.added.ImportSymlink {
		t.Fatalf("add opts = %+v, want missing-path add with initial profile", service.added)
	}
}

func TestAddHarnessPreviewUsesNormalizedConfigRootPath(t *testing.T) {
	m := NewModel(newMockService())
	m = startGuidedAdd(m, "opencode", "OpenCode", "relative-opencode-config", domain.HarnessLinkKindDir, app.HarnessLinkActionImport)

	if !filepath.IsAbs(m.addDraft.ConfigPath) || strings.Contains(m.confirm, "Config root path: relative-opencode-config") {
		t.Fatalf("draft = %+v confirm = %q, want normalized absolute path in preview", m.addDraft, m.confirm)
	}
}

func TestAddHarnessFormIsCompact(t *testing.T) {
	m := NewModel(newMockService())
	m.startForm(opAdd)
	view := m.View()

	if strings.Contains(view, "placeholder:") || !strings.Contains(view, "Harness ID *") || !strings.Contains(view, "Label *") || !strings.Contains(view, "Step 1/3: describe the harness") || strings.Contains(view, "Config root path") || strings.Contains(view, "Focus") {
		t.Fatalf("form view = %q, want compact form without duplicated placeholders", view)
	}
	if strings.Contains(view, "space select") {
		t.Fatalf("form view = %q, should not show option shortcut when no options exist", view)
	}
	if !strings.Contains(view, "enter") || !strings.Contains(view, "next") || strings.Contains(view, "preview") {
		t.Fatalf("form view = %q, want enter next before final field", view)
	}
	if !strings.Contains(view, "Next") {
		t.Fatalf("form view = %q, want visible Next button", view)
	}
	if !strings.Contains(view, "from the") || !strings.Contains(view, "new path") {
		t.Fatalf("form view = %q, should wrap long field values instead of truncating them", view)
	}
	m.field = len(m.fields) - 1
	m.focusField()
	if view = m.View(); !strings.Contains(view, "enter next") || !strings.Contains(view, "Next") || strings.Contains(view, "enter preview") {
		t.Fatalf("form view = %q, want enter next hint and visible Next button on final field", view)
	}
}

func TestFormButtonIsFocusableAndAdvances(t *testing.T) {
	m := NewModel(newMockService())
	m.startForm(opAdd)
	m.fields[0].Input.SetValue("opencode")
	m.fields[1].Input.SetValue("OpenCode")
	m.fields[2].Input.SetValue("Restart OpenCode so it re-reads config from the new path")

	for range len(m.fields) {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	if m.field != len(m.fields) || !strings.Contains(m.View(), "› Next") {
		t.Fatalf("field = %d view = %q, want focused Next button", m.field, m.View())
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenAddLinks || !strings.Contains(m.View(), "Step 2/3: add every path") {
		t.Fatalf("screen = %v view = %q, want Next button to open managed links", m.screen, m.View())
	}
}

func TestAddHarnessDirectoryPathAsksForProfileName(t *testing.T) {
	m := NewModel(newMockService())
	root := filepath.Join(t.TempDir(), "opencode")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	m = startGuidedAdd(m, "opencode", "OpenCode", root, domain.HarnessLinkKindDir, app.HarnessLinkActionImport)

	if m.screen != screenConfirm || m.addDraft.Branch != "directory" {
		t.Fatalf("screen = %v draft = %+v view = %q, want import plan preview", m.screen, m.addDraft, m.View())
	}
	if !strings.Contains(m.View(), "Detected managed links") || !strings.Contains(m.View(), "root (dir)") {
		t.Fatalf("view = %q, want detected directory context", m.View())
	}
	if !strings.Contains(m.View(), "y") || !strings.Contains(m.View(), "next") || strings.Contains(m.View(), "confirm") {
		t.Fatalf("view = %q, want intermediate import step footer", m.View())
	}
	if !strings.Contains(m.View(), "› Next") || !strings.Contains(m.View(), "Cancel") {
		t.Fatalf("view = %q, want focused Next and Cancel buttons", m.View())
	}
	updated, _ := m.Update(key("y"))
	m = updated.(Model)
	if m.screen != screenForm || len(m.fields) != 1 || !strings.Contains(m.View(), "New profile name") || !strings.Contains(m.View(), "Step 3/3: name the initial profile") {
		t.Fatalf("screen = %v fields = %d view = %q, want profile-name form after approving plan", m.screen, len(m.fields), m.View())
	}
}

func TestConfirmButtonsAreFocusableAndCancelable(t *testing.T) {
	m := NewModel(newMockService())
	m = startGuidedAdd(m, "opencode", "OpenCode", filepath.Join(t.TempDir(), "missing-config"), domain.HarnessLinkKindDir, app.HarnessLinkActionImport)
	if m.screen != screenForm || !strings.Contains(m.View(), "Step 3/3: name the initial profile") {
		t.Fatalf("screen = %v view = %q, want next form step", m.screen, m.View())
	}
	m.fields[0].Input.SetValue("default")
	m.field = len(m.fields)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenConfirm || !strings.Contains(m.View(), "› Confirm") || !strings.Contains(m.View(), "Cancel") {
		t.Fatalf("screen = %v view = %q, want confirm buttons", m.screen, m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if !strings.Contains(m.View(), "› Cancel") {
		t.Fatalf("view = %q, want focused Cancel button", m.View())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenHarnesses {
		t.Fatalf("screen = %v, want cancel back to harnesses", m.screen)
	}
}

func TestAddHarnessSymlinkPathImportsTargetProfile(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	m = startGuidedAdd(m, "opencode", "OpenCode", link, domain.HarnessLinkKindDir, app.HarnessLinkActionImport)
	if m.addDraft.ConfigPath != link || m.addDraft.SourcePath != target || !strings.Contains(m.View(), "Detected managed links") || !strings.Contains(m.View(), "root (dir)") {
		t.Fatalf("view = %q, want symlink import context", m.View())
	}
	updated, _ := m.Update(key("y"))
	m = updated.(Model)
	m.fields[0].Input.SetValue("imported")
	m = m.submitForm()
	if strings.Contains(m.confirm, "Source copied") || strings.Contains(m.confirm, "Managed profile") || !strings.Contains(m.confirm, "Managed links:") || !strings.Contains(m.confirm, "Symlink target:") || !strings.Contains(m.confirm, "Copy from / replace:") {
		t.Fatalf("confirm = %q, want clear multi-link import plan without duplicate source/profile fields", m.confirm)
	}
	m.execute()

	if service.added.InitialProfile != "imported" || !service.added.ImportSymlink || service.added.RegisterSymlink {
		t.Fatalf("add opts = %+v, want symlink import into named profile", service.added)
	}
}

func TestAddHarnessAcceptsFilePathsAndFileSymlinks(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(string) string
		wantImport bool
	}{
		{
			name: "regular file",
			prepare: func(base string) string {
				path := filepath.Join(base, "runtime.json")
				if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
		},
		{
			name: "symlink to file",
			prepare: func(base string) string {
				target := filepath.Join(base, "target.json")
				if err := os.WriteFile(target, []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(base, "runtime.json")
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
				return link
			},
			wantImport: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newMockService()
			m := NewModel(service)
			base := t.TempDir()
			path := tt.prepare(base)
			m = startGuidedAdd(m, "opencode", "OpenCode", path, domain.HarnessLinkKindFile, app.HarnessLinkActionImport)
			if m.screen != screenConfirm {
				t.Fatalf("screen = %v, want confirmation step", m.screen)
			}

			updated, _ := m.Update(key("y"))
			m = updated.(Model)
			if m.screen != screenForm || len(m.fields) != 1 {
				t.Fatalf("screen = %v fields = %d, want profile name step", m.screen, len(m.fields))
			}
			m.fields[0].Input.SetValue("default")
			m = m.submitForm()
			m.execute()

			if len(service.added.Links) != 1 || service.added.Links[0].Kind != domain.HarnessLinkKindFile || service.added.Links[0].Path != path || service.added.LinkPath != "" || service.added.InitialProfile != "default" || service.added.ImportSymlink != tt.wantImport {
				t.Fatalf("add opts = %+v, want file link add", service.added)
			}
		})
	}
}

func TestAddHarnessAcceptsMultiLinkInput(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.startForm(opAdd)
	m.fields[0].Input.SetValue("claude")
	m.fields[1].Input.SetValue("Claude")
	m.fields[2].Input.SetValue("Restart Claude so it re-reads config from the new path")
	m = m.submitForm()
	if m.screen != screenAddLinks || !strings.Contains(m.View(), "Managed links (0)") || !strings.Contains(m.View(), "Link ID") || !strings.Contains(m.View(), "Directory") || !strings.Contains(m.View(), "File") || !strings.Contains(m.View(), "Add link") || !strings.Contains(m.View(), "Continue") || strings.Contains(m.View(), "id|kind|path") {
		t.Fatalf("view = %q, want a guided managed-link editor", m.View())
	}
	m.fields[0].Input.SetValue("root")
	m.fields[1].Input.SetValue(filepath.Join(t.TempDir(), "claude"))
	m.linkKind = domain.HarnessLinkKindDir
	m.field = addLinkButton
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !strings.Contains(m.View(), "Managed links (1)") || !strings.Contains(m.View(), "root (dir)") || !strings.Contains(m.View(), "Remove last link") {
		t.Fatalf("view = %q, want the added link and clear editing actions", m.View())
	}
	m.fields[0].Input.SetValue("state")
	m.fields[1].Input.SetValue(filepath.Join(t.TempDir(), "claude.json"))
	m.linkKind = domain.HarnessLinkKindFile
	m.linkAction = app.HarnessLinkActionRegister
	m.field = addLinkButton
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	m.field = addLinksContinueButton
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if len(m.addDraft.Links) != 2 || m.addDraft.Links[1].Kind != domain.HarnessLinkKindFile {
		t.Fatalf("draft = %+v, want parsed multi-link input", m.addDraft)
	}
	if m.screen != screenForm || len(m.fields) != 1 {
		t.Fatalf("screen = %v fields = %d, want initial profile step for multi-link add", m.screen, len(m.fields))
	}
	m.fields[0].Input.SetValue("default")
	m = m.submitForm()
	if !strings.Contains(m.confirm, "Managed links") || !strings.Contains(m.confirm, "root (dir)") || !strings.Contains(m.confirm, "state (file)") {
		t.Fatalf("confirm = %q, want multi-link preview", m.confirm)
	}
	m.execute()
	if len(service.added.Links) != 2 || service.added.LinkPath != "" || service.added.LinkActions["state"] != app.HarnessLinkActionRegister {
		t.Fatalf("added opts = %+v, want multi-link add options", service.added)
	}
}

func TestUpdateHarnessFormOnlyUpdatesMetadata(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.startForm(opUpdate)
	if len(m.fields) != 2 || strings.Contains(m.View(), "Managed link path") || strings.Contains(m.View(), "id|kind|path") {
		t.Fatalf("fields = %d view = %q, want metadata-only harness update", len(m.fields), m.View())
	}
	m.fields[0].Input.SetValue("Claude Code")
	m.fields[1].Input.SetValue("Restart Claude")

	m = m.submitForm()
	if !strings.Contains(m.confirm, "Managed links:") || !strings.Contains(m.confirm, "Managed links are unchanged") {
		t.Fatalf("confirm = %q, want unchanged-link metadata preview", m.confirm)
	}
	m.execute()
	if service.updated.Label != "Claude Code" || service.updated.RestartHint != "Restart Claude" || len(service.updated.Links) != 0 || service.updated.LinkPath != "" {
		t.Fatalf("updated opts = %+v, want metadata-only update options", service.updated)
	}
}

func TestFormOptionsAreReachedWithDownAndSpace(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.startForm(opClone)

	for range 3 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	updated, _ := m.Update(key(" "))
	m = updated.(Model)

	if m.focusedOption() != 1 || !m.options[1].Checked || m.options[0].Checked {
		t.Fatalf("field = %d option = %d options = %+v, want second option selected via vertical navigation", m.field, m.option, m.options)
	}
}

func TestFormEnumOptionsChangeWithVerticalNavigation(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.startForm(opDeleteHarness)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.selectedOption() != "restore" {
		t.Fatalf("selected option = %q, want restore after vertical navigation", m.selectedOption())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.selectedOption() != "keep-root" {
		t.Fatalf("selected option = %q, want keep-root after vertical navigation", m.selectedOption())
	}
}

func TestDeleteHarnessKeepRootGoesStraightToPreview(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.startForm(opDeleteHarness)

	m = m.submitForm()

	if m.screen != screenConfirm || m.deleteMode != "keep-root" || !strings.Contains(m.confirm, "Root handling mode: keep-root") {
		t.Fatalf("screen = %v mode = %q confirm = %q, want keep-root preview", m.screen, m.deleteMode, m.confirm)
	}
}

func TestDeleteHarnessRestoreUsesProfileOptions(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.startForm(opDeleteHarness)
	m.focusOption(1)
	m.checkFocusedOption()

	m = m.submitForm()
	view := m.View()

	if m.screen != screenForm || m.deleteMode != "restore" || len(m.fields) != 0 || len(m.options) != 2 || !strings.Contains(view, "Choose profile to restore") || strings.Contains(view, "Restore profile") {
		t.Fatalf("screen = %v mode = %q fields = %d options = %+v view = %q, want restore profile picker", m.screen, m.deleteMode, len(m.fields), m.options, view)
	}

	m.focusOption(1)
	m.checkFocusedOption()
	m = m.submitForm()
	m.execute()

	if serviceDeleted := m.service.(*mockService).deleted; serviceDeleted.Mode != "restore" || serviceDeleted.RestoreProfile != "work" {
		t.Fatalf("delete opts = %+v, want selected restore profile", serviceDeleted)
	}
}

func TestDeleteHarnessDeleteAllShowsRequiredConfirmationText(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.startForm(opDeleteHarness)
	m.focusOption(2)
	m.checkFocusedOption()

	m = m.submitForm()
	view := m.View()

	if m.screen != screenForm || m.deleteMode != "delete-all" || len(m.fields) != 1 || !strings.Contains(view, "Type claude to confirm") {
		t.Fatalf("screen = %v mode = %q fields = %+v view = %q, want explicit delete-all confirmation", m.screen, m.deleteMode, m.fields, view)
	}

	m.fields[0].Input.SetValue("claude")
	m = m.submitForm()
	m.execute()

	if serviceDeleted := m.service.(*mockService).deleted; serviceDeleted.Mode != "delete-all" || serviceDeleted.Confirm != "claude" {
		t.Fatalf("delete opts = %+v, want delete-all confirmation", serviceDeleted)
	}
}

func TestDeleteHarnessDeleteAllConfirmationUsesHarnessID(t *testing.T) {
	service := newMockService()
	service.harnesses[0].Label = "Demo 3"
	m := NewModel(service)
	m.loadDetail()
	m.startForm(opDeleteHarness)
	m.focusOption(2)
	m.checkFocusedOption()

	m = m.submitForm()

	if !strings.Contains(m.View(), "Type claude to confirm") || m.fields[0].Input.Placeholder != "claude" || strings.Contains(m.View(), "Type Demo 3 to confirm") {
		t.Fatalf("view = %q placeholder = %q, want harness ID confirmation", m.View(), m.fields[0].Input.Placeholder)
	}
}

func TestDeleteHarnessPreviewAndExecution(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.startForm(opDeleteHarness)
	m.focusOption(2)
	m.checkFocusedOption()
	m = m.submitForm()
	m.fields[0].Input.SetValue("claude")

	m = m.submitForm()
	if !strings.Contains(m.confirm, "Config entry") || !strings.Contains(m.confirm, "Managed profiles directory") || !strings.Contains(m.confirm, "Root handling mode: delete-all") {
		t.Fatalf("confirm = %q, want destructive preview", m.confirm)
	}
	m.execute()
	if service.deleted.Mode != "delete-all" || service.deleted.Confirm != "claude" {
		t.Fatalf("delete opts = %+v, want submitted options", service.deleted)
	}
}

func TestMultiLinkPreviewsListEveryManagedLink(t *testing.T) {
	service := newMockService()
	service.harnesses[0].Links = []domain.HarnessLink{
		{ID: domain.LegacyDefaultLinkID, Path: "/tmp/claude", Kind: domain.HarnessLinkKindDir},
		{ID: "state", Path: "/tmp/claude.json", Kind: domain.HarnessLinkKindFile},
	}
	m := NewModel(service)
	m.loadDetail()
	m.profile = 0

	m.startForm(opSwitch)
	if got := m.switchProfilePreview(app.ProfileStatus{Name: "work", Path: "/tmp/harnesses/claude/work"}); !strings.Contains(got, "root (dir): /tmp/claude ->") || !strings.Contains(got, "state (file): /tmp/claude.json ->") {
		t.Fatalf("switch preview = %q, want every managed link repointed", got)
	}

	m.startForm(opDeleteHarness)
	m.deleteMode = deleteModeDeleteAll
	if got := m.deleteHarnessPreview(); !strings.Contains(got, "Managed links") || !strings.Contains(got, "root (dir): /tmp/claude") || !strings.Contains(got, "state (file): /tmp/claude.json") {
		t.Fatalf("delete preview = %q, want every managed link listed", got)
	}

	m.startForm(opAdopt)
	m.fields[0].Input.SetValue("work")
	if got := m.adoptPreview(); !strings.Contains(got, "Adopt current links") || !strings.Contains(got, "Managed artifacts") || !strings.Contains(got, "root (dir):") || !strings.Contains(got, "state (file):") {
		t.Fatalf("adopt preview = %q, want multi-link artifact list", got)
	}

	m.startForm(opCreateProfile)
	m.fields[0].Input.SetValue("work")
	if got := m.createProfilePreview(); !strings.Contains(got, "Managed artifacts") || !strings.Contains(got, "root (dir):") || !strings.Contains(got, "state (file):") {
		t.Fatalf("create preview = %q, want artifact list", got)
	}

	m.startForm(opRenameProfile)
	m.fields[0].Input.SetValue("renamed")
	if got := m.renameProfilePreview(); !strings.Contains(got, "every managed symlink") || !strings.Contains(got, "Managed artifacts") {
		t.Fatalf("rename preview = %q, want multi-link repointing", got)
	}

	m.startForm(opClone)
	m.fields[0].Input.SetValue("copy")
	if got := m.clonePreview(); !strings.Contains(got, "every managed link artifact") || !strings.Contains(got, "Source artifacts") || !strings.Contains(got, "Target artifacts") {
		t.Fatalf("clone preview = %q, want multi-link clone details", got)
	}
}

func TestDestructivePreviewOnlyHighlightsNameAndConsequence(t *testing.T) {
	body := renderDestructivePreviewBody("Delete harness demo4?\n\nConfig entry: id=demo4 label=Demo4\nManaged root path: /tmp/demo4\nConsequence: removes config entry")

	if !strings.Contains(body, "Delete harness demo4?") || !strings.Contains(body, "Config entry:") || !strings.Contains(body, "id=demo4 label=Demo4") || !strings.Contains(body, "Consequence: removes config entry") {
		t.Fatalf("body = %q, want destructive preview content preserved", body)
	}
}

func TestDashboardSupportsSearchAndPolishedMarkers(t *testing.T) {
	service := newMockService()
	service.harnesses = append(service.harnesses, domain.Harness{ID: "opencode", Label: "OpenCode", LinkPath: "/tmp/opencode"})
	m := NewModel(service)

	updated, _ := m.Update(key("/"))
	m = updated.(Model)
	updated, _ = m.Update(key("o"))
	m = updated.(Model)
	view := m.View()

	if !strings.Contains(view, "⌕ search: o") || !strings.Contains(view, "▸") || !strings.Contains(view, "Harness Profiles") || !strings.Contains(view, "enter") || !strings.Contains(view, "apply") || !strings.Contains(view, "esc") || !strings.Contains(view, "cancel") {
		t.Fatalf("dashboard view = %q, want search, marker, and title", view)
	}
	if !strings.Contains(view, "OpenCode") || strings.Contains(view, "label: OpenCode") || strings.Contains(view, "id: opencode") || strings.Contains(view, "\x1b[48;5;42") {
		t.Fatalf("dashboard view = %q, harness row should show label only", view)
	}
}

func TestDashboardSearchMatchesOnlyHarnessIDAndLabel(t *testing.T) {
	service := newMockService()
	service.harnesses = []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: "/tmp/path-only-match"}}
	m := NewModel(service)

	m.hQuery = "path-only"
	if got := m.filteredHarnessIndexes(); len(got) != 0 {
		t.Fatalf("filtered indexes = %v, want no match from link path", got)
	}

	m.hQuery = "cla"
	if got := m.filteredHarnessIndexes(); len(got) != 1 || got[0] != 0 {
		t.Fatalf("filtered indexes = %v, want match from harness id", got)
	}
}

func TestDashboardShowsManagedLinkSummaryForMultiLinkHarness(t *testing.T) {
	service := newMockService()
	service.harnesses[0].Links = []domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: "/tmp/claude", Kind: domain.HarnessLinkKindDir}, {ID: "state", Path: "/tmp/claude.json", Kind: domain.HarnessLinkKindFile}}
	m := NewModel(service)
	view := m.View()

	if !strings.Contains(view, "root (dir)") || !strings.Contains(view, "claude.json") {
		t.Fatalf("dashboard view = %q, want multi-link summary", view)
	}
}

func TestDashboardFilteredSelectionLoadsFilteredHarness(t *testing.T) {
	service := newMockService()
	service.harnesses = append(service.harnesses, domain.Harness{ID: "opencode", Label: "OpenCode", LinkPath: "/tmp/opencode"})
	m := NewModel(service)

	m.hSearching = true
	m.hDraft = "open"
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenHarnesses || m.hSearching || m.hQuery != "open" {
		t.Fatalf("screen = %v searching = %v query = %q, want filtered dashboard after enter", m.screen, m.hSearching, m.hQuery)
	}
	if !strings.Contains(m.View(), "d") || !strings.Contains(m.View(), "delete") || !strings.Contains(m.View(), "esc") || !strings.Contains(m.View(), "clear") || !strings.Contains(m.View(), "/") || !strings.Contains(m.View(), "edit search") || strings.Contains(m.View(), "apply") {
		t.Fatalf("view = %q, want normal dashboard footer after applying filter", m.View())
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.screen != screenDetail || m.harness.ID != "opencode" {
		t.Fatalf("screen = %v harness = %+v, want opencode detail from filtered dashboard", m.screen, m.harness)
	}
}

func TestDashboardEscClearsAppliedSearchFilter(t *testing.T) {
	service := newMockService()
	service.harnesses = append(service.harnesses, domain.Harness{ID: "opencode", Label: "OpenCode", LinkPath: "/tmp/opencode"})
	m := NewModel(service)
	m.hQuery = "open"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.hSearching || m.hQuery != "" || len(m.filteredHarnessIndexes()) != 2 {
		t.Fatalf("searching = %v query = %q indexes = %v, want cleared dashboard filter", m.hSearching, m.hQuery, m.filteredHarnessIndexes())
	}
}

func TestDashboardEscCancelsSearchModeWithoutClearingAppliedFilter(t *testing.T) {
	service := newMockService()
	service.harnesses = append(service.harnesses, domain.Harness{ID: "opencode", Label: "OpenCode", LinkPath: "/tmp/opencode"})
	m := NewModel(service)
	m.hQuery = "cla"
	m.hSearching = true
	m.hDraft = "open"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.hSearching || m.hQuery != "cla" || len(m.filteredHarnessIndexes()) != 1 || !strings.Contains(m.View(), "esc clear") {
		t.Fatalf("searching = %v query = %q indexes = %v view = %q, want cancelled search with previous filter kept", m.hSearching, m.hQuery, m.filteredHarnessIndexes(), m.View())
	}
}

func TestProfileSearchEnterKeepsFilterAndEscClearsIt(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.pSearching = true
	m.pDraft = "work"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenDetail || m.pSearching || m.pQuery != "work" || len(m.filteredProfileIndexes()) != 1 {
		t.Fatalf("screen = %v searching = %v query = %q indexes = %v, want profile filter kept after enter", m.screen, m.pSearching, m.pQuery, m.filteredProfileIndexes())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.pSearching || m.pQuery != "" || len(m.filteredProfileIndexes()) != 2 {
		t.Fatalf("searching = %v query = %q indexes = %v, want cleared profile filter", m.pSearching, m.pQuery, m.filteredProfileIndexes())
	}
}

func TestDashboardHarnessActionsUseSelectedRowContext(t *testing.T) {
	service := newMockService()
	service.harnesses = append(service.harnesses, domain.Harness{ID: "opencode", Label: "OpenCode", LinkPath: "/tmp/opencode"})
	m := NewModel(service)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if !strings.Contains(m.View(), "OpenCode") || strings.Contains(m.View(), "opencode  label") || strings.Contains(m.View(), "selected") {
		t.Fatalf("view = %q, want selected-row opencode without persistent selection text", m.View())
	}
	updated, _ = m.Update(key("d"))
	m = updated.(Model)

	if m.screen != screenForm || m.op != opDeleteHarness || !strings.Contains(m.View(), "Delete selected harness") || !strings.Contains(m.View(), "opencode") || !strings.Contains(m.View(), "/tmp/opencode") {
		t.Fatalf("screen = %v op = %v view = %q, want delete form with selected harness context", m.screen, m.op, m.View())
	}
}

func TestDashboardTabJumpsBetweenHarnessesAndActions(t *testing.T) {
	service := newMockService()
	service.harnesses = append(service.harnesses, domain.Harness{ID: "opencode", Label: "OpenCode", LinkPath: "/tmp/opencode"})
	m := NewModel(service)
	m.menu = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	items := m.dashboardItems()
	if items[m.menu].Kind != "add" {
		t.Fatalf("menu = %d item = %+v, want first action", m.menu, items[m.menu])
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if items[m.menu].Kind != "harness" {
		t.Fatalf("menu = %d item = %+v, want harness section", m.menu, items[m.menu])
	}
}

func TestDashboardTypingLetterDoesNotFilterOutsideSearchMode(t *testing.T) {
	m := NewModel(newMockService())

	updated, _ := m.Update(key("a"))
	m = updated.(Model)

	if m.screen != screenHarnesses || m.hQuery != "" {
		t.Fatalf("screen = %v query = %q, want no dashboard filter outside search mode", m.screen, m.hQuery)
	}
}

func TestClonePreviewAndProgressDispatch(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.startForm(opClone)
	m.fields[0].Input.SetValue("experiment")
	m.options[0].Checked = false
	m.options[1].Checked = true

	m = m.submitForm()
	if m.screen != screenConfirm || !strings.Contains(m.confirm, "Set cloned profile experiment as active") {
		t.Fatalf("confirm = %q screen = %v, want clone activation choice", m.confirm, m.screen)
	}
	updated, _ := m.Update(key("y"))
	m = updated.(Model)
	if !strings.Contains(m.confirm, "Materialize") && !strings.Contains(m.confirm, "materialize") || !strings.Contains(m.confirm, "switch to cloned profile") {
		t.Fatalf("confirm = %q, want clone materialize preview with activation", m.confirm)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if m.screen != screenProgress || cmd == nil || !strings.Contains(m.View(), "Cloning profile") {
		t.Fatalf("screen = %v cmd nil = %v view = %q, want progress", m.screen, cmd == nil, m.View())
	}
}

func TestProfileOperationResultReturnsToProfilesDashboard(t *testing.T) {
	service := newMockService()
	m := NewModel(service)
	m.loadDetail()
	m.profile = 1
	m.startForm(opClone)
	m.fields[0].Input.SetValue("experiment")
	m = m.submitForm()
	updated, _ := m.Update(key("n"))
	m = updated.(Model)
	m.execute()
	if m.screen != screenResult {
		t.Fatalf("screen = %v, want result", m.screen)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenDetail || m.harness.ID != "claude" {
		t.Fatalf("screen = %v harness = %q, want profiles dashboard for claude", m.screen, m.harness.ID)
	}
}

func TestDeleteActiveProfileRefusedBeforeConfirm(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.profile = 0

	m.startProfileConfirm(opDeleteProfile)

	if m.screen != screenResult || m.err == nil || !strings.Contains(m.err.Error(), "cannot delete active profile") {
		t.Fatalf("screen = %v err = %v, want active deletion refusal", m.screen, m.err)
	}
}

func TestRunWithIOSmoke(t *testing.T) {
	input := bytes.NewBufferString("\x03")
	var output bytes.Buffer
	if err := RunWithIO(newMockService(), input, &output); err != nil {
		t.Fatalf("RunWithIO() error = %v", err)
	}
}

func newMockService() *mockService {
	return &mockService{
		harnesses: []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: "/tmp/claude"}},
		profiles: []app.ProfileStatus{
			{Name: "default", Path: "/tmp/profiles/claude/default", Active: true},
			{Name: "work", Path: "/tmp/profiles/claude/work"},
		},
		current: app.ProfileStatus{Name: "default", Path: "/tmp/profiles/claude/default"},
		paths:   app.Paths{ConfigPath: "/tmp/config.json", HarnessesRoot: "/tmp/harnesses"},
	}
}

func key(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}
