package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func TestTextFittingHelpers(t *testing.T) {
	if got := fitMiddle("abcdefghijklmnopqrstuvwxyz", 10); got != "abc...wxyz" {
		t.Fatalf("fitMiddle() = %q", got)
	}
	if got := fitMiddle("abcdef", 3); got != "..." {
		t.Fatalf("fitMiddle(small) = %q", got)
	}
	if got := fitTail("abcdefghijklmnopqrstuvwxyz", 8); got != "abcde..." {
		t.Fatalf("fitTail() = %q", got)
	}
	if got := fitTail("abcdef", 2); got != ".." {
		t.Fatalf("fitTail(small) = %q", got)
	}
	if got := lastRune("abcñ"); got != "ñ" {
		t.Fatalf("lastRune() = %q", got)
	}
	if got := lastRune(""); got != "" {
		t.Fatalf("lastRune(empty) = %q", got)
	}
}

func TestWrapAndClampHelpers(t *testing.T) {
	if got := clamp(-1, 3); got != 0 {
		t.Fatalf("clamp(low) = %d", got)
	}
	if got := clamp(9, 3); got != 2 {
		t.Fatalf("clamp(high) = %d", got)
	}
	if got := wrap(-1, 3); got != 2 {
		t.Fatalf("wrap(low) = %d", got)
	}
	if got := wrap(3, 3); got != 0 {
		t.Fatalf("wrap(high) = %d", got)
	}
	if got := wrap(1, 3); got != 1 {
		t.Fatalf("wrap(in-range) = %d", got)
	}
}

func TestWrapFieldValueIndentsContinuationLines(t *testing.T) {
	got := wrapFieldValue("one two three four five", 8)
	if !strings.Contains(got, "\n                         ") {
		t.Fatalf("wrapFieldValue() = %q, want indented continuation", got)
	}
}

func TestOperationLabelsCoverKnownOperations(t *testing.T) {
	for _, op := range []operation{opUpdate, opDeleteHarness, opSwitch, opRenameProfile, opClone, opDeleteProfile, opAddLink, opUpdateLink, opDeleteLink} {
		if operationTitle(op) == "" {
			t.Fatalf("operationTitle(%v) is empty", op)
		}
		if operationProgress(op) == "" {
			t.Fatalf("operationProgress(%v) is empty", op)
		}
	}
}

func TestSmallViewHelpersCoverSearchAndLabels(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.pSearching = true
	m.pDraft = "wo"
	if got := m.pSearchText(); got != "wo" {
		t.Fatalf("pSearchText(searching) = %q", got)
	}
	m.pSearching = false
	m.pQuery = "default"
	if got := m.pSearchText(); got != "default" {
		t.Fatalf("pSearchText(query) = %q", got)
	}
	if hints := strings.Join(m.profileFooterHints(), " "); !strings.Contains(hints, "edit search") || !strings.Contains(hints, "esc clear") {
		t.Fatalf("profileFooterHints() = %v", m.profileFooterHints())
	}
	for _, op := range []operation{opUpdate, opDeleteHarness, opSwitch, opRenameProfile, opClone, opDeleteProfile} {
		m.op = op
		if m.formActionLabel() == "" {
			t.Fatalf("formActionLabel(%v) is empty", op)
		}
	}
}

func TestPreviewHelpersCoverOperations(t *testing.T) {
	m := NewModelWithOptions(&mockService{}, Options{})
	m.paths.HarnessesRoot = "/tmp/harnesses"
	m.harness.ID = "claude"
	m.harness.Label = "Claude"
	m.harness.LinkPath = "/tmp/root"
	m.harness.Links = []domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: "/tmp/root", Kind: domain.HarnessLinkKindDir}, {ID: "state", Path: "/tmp/root.json", Kind: domain.HarnessLinkKindFile}}
	m.fields = []formField{fieldWithValue("work"), fieldWithValue("new"), fieldWithValue("restart")}
	m.addDraft = addHarnessDraft{ID: "claude", Label: "Claude", ConfigPath: "/tmp/root", Branch: addBranchMissing, RestartHint: "restart"}
	m.profiles = []app.ProfileStatus{{Name: "default", Path: "/tmp/harnesses/claude/default"}}
	m.profile = 0
	m.options = []optionItem{{Value: "preserve", Checked: true}}

	previews := []string{
		m.addPreview(),
		m.deleteHarnessPreview(),
		m.updatePreview(),
		m.adoptPreview(),
		m.createProfilePreview(),
		m.renameProfilePreview(),
		m.clonePreview(),
		m.cloneActivationPrompt(),
	}
	for _, preview := range previews {
		if strings.TrimSpace(preview) == "" {
			t.Fatal("preview is empty")
		}
	}
}

func TestInitialFlowOptionsStartExpectedScreens(t *testing.T) {
	tests := []struct {
		name string
		flow Flow
		want screen
	}{
		{name: "add", flow: FlowAdd, want: screenForm},
		{name: "update", flow: FlowUpdate, want: screenForm},
		{name: "delete harness", flow: FlowDeleteHarness, want: screenForm},
		{name: "switch", flow: FlowSwitch, want: screenConfirm},
		{name: "adopt", flow: FlowAdopt, want: screenForm},
		{name: "clone", flow: FlowClone, want: screenForm},
		{name: "delete profile", flow: FlowDeleteProfile, want: screenResult},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModelWithOptions(newMockService(), Options{Flow: tt.flow, HarnessID: "claude"})
			if m.screen != tt.want {
				t.Fatalf("screen = %v, want %v", m.screen, tt.want)
			}
		})
	}
}

func TestAddFormLabelTypingSyncsDefaultRestartHint(t *testing.T) {
	m := NewModel(newMockService())
	m.startForm(opAdd)
	m.field = 1
	m.focusField()

	updated, _ := m.Update(key("C"))
	m = updated.(Model)
	if got := m.fields[2].Input.Value(); !strings.Contains(got, "C") {
		t.Fatalf("restart hint = %q, want typed label", got)
	}
}

func TestFormValidationErrorsKeepUserOnForm(t *testing.T) {
	m := NewModel(newMockService())
	m.startForm(opAdd)

	m = m.submitForm()
	if m.screen != screenForm || m.err == nil || !strings.Contains(m.err.Error(), "required") {
		t.Fatalf("screen = %v err = %v, want add validation error", m.screen, m.err)
	}

	m.startForm(opClone)
	m.profiles = nil
	m = m.submitForm()
	if m.err == nil || !strings.Contains(m.err.Error(), "selected source") {
		t.Fatalf("err = %v, want clone validation error", m.err)
	}
}

func TestResultReturnRoutesByOperation(t *testing.T) {
	m := NewModel(newMockService())
	m.loadDetail()
	m.op = opSwitch
	m.screen = screenResult
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenDetail {
		t.Fatalf("screen = %v, want detail for profile operation result", m.screen)
	}

	m.op = opAdd
	m.screen = screenResult
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.screen != screenHarnesses {
		t.Fatalf("screen = %v, want harness dashboard for harness operation result", m.screen)
	}
}

func fieldWithValue(value string) formField {
	input := textinput.New()
	input.SetValue(value)
	field := formField{Input: input}
	return field
}
