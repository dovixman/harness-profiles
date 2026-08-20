package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dovixman/harness-profiles/internal/adapters/configyaml"
	"github.com/dovixman/harness-profiles/internal/adapters/tui"
	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "where") || !strings.Contains(stdout.String(), "ls") {
		t.Fatalf("help output = %q, want command list", stdout.String())
	}
}

func TestWhereUsesTemporaryConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"where"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), filepath.Join(configHome, "config.json")) {
		t.Fatalf("where output = %q, want config path", stdout.String())
	}
	if !strings.Contains(stdout.String(), filepath.Join(configHome, "harnesses")) {
		t.Fatalf("where output = %q, want harnesses path", stdout.String())
	}
}

func TestListHarnesses(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	paths := configyaml.Paths{ConfigHome: configHome}
	repo := configyaml.NewRepository(paths)
	if err := repo.Save(domain.Config{
		HarnessesRoot: paths.HarnessesRoot(),
		Harnesses: []domain.Harness{{
			ID:       "opencode",
			Label:    "OpenCode",
			LinkPath: filepath.Join(configHome, "opencode", "agents"),
		}},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"ls"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "opencode\tOpenCode\t") {
		t.Fatalf("ls output = %q, want harness row", stdout.String())
	}
}

func TestListHarnessesDisplaysAllLinksForMultiLinkHarness(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	paths := configyaml.Paths{ConfigHome: configHome}
	repo := configyaml.NewRepository(paths)
	if err := repo.Save(domain.Config{
		HarnessesRoot: paths.HarnessesRoot(),
		Harnesses: []domain.Harness{{
			ID:    "claude",
			Label: "Claude",
			Links: []domain.HarnessLink{
				{ID: "root", Path: filepath.Join(configHome, "claude-root"), Kind: domain.HarnessLinkKindDir},
				{ID: "state", Path: filepath.Join(configHome, "claude-state"), Kind: domain.HarnessLinkKindFile},
			},
		}},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"ls"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	row := "claude\tClaude\t"
	if !strings.Contains(stdout.String(), row) {
		t.Fatalf("ls output = %q, want row containing %q", stdout.String(), row)
	}
	if !strings.Contains(stdout.String(), "root:dir:") || !strings.Contains(stdout.String(), "state:file:") {
		t.Fatalf("ls output = %q, want multi-link summary", stdout.String())
	}
}

func TestParseConfigRootLinksSupportsLegacyAndMultiSyntax(t *testing.T) {
	configHome := t.TempDir()
	legacyRoot := filepath.Join(configHome, "legacy")
	stateFile := filepath.Join(configHome, "state.json")
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	links, _, err := parseConfigRootLinks([]string{legacyRoot}, domain.Harness{}, true)
	if err != nil {
		t.Fatalf("parseConfigRootLinks(legacy) error = %v", err)
	}
	if got, want := len(links), 1; got != want {
		t.Fatalf("len(links) = %d, want %d", got, want)
	}
	if links[0].ID != domain.LegacyDefaultLinkID {
		t.Fatalf("links[0].ID = %q, want %q", links[0].ID, domain.LegacyDefaultLinkID)
	}
	if links[0].Kind != domain.HarnessLinkKindDir {
		t.Fatalf("links[0].Kind = %q, want %q", links[0].Kind, domain.HarnessLinkKindDir)
	}

	multi, _, err := parseConfigRootLinks([]string{"root=" + legacyRoot, "state=" + stateFile}, domain.Harness{}, true)
	if err != nil {
		t.Fatalf("parseConfigRootLinks(multi) error = %v", err)
	}
	if got, want := len(multi), 2; got != want {
		t.Fatalf("len(multi) = %d, want %d", got, want)
	}
	if multi[1].Kind != domain.HarnessLinkKindFile {
		t.Fatalf("multi[1].Kind = %q, want %q", multi[1].Kind, domain.HarnessLinkKindFile)
	}
}

func TestParseConfigRootLinksRejectsMixedOrDuplicateSpecs(t *testing.T) {
	base := t.TempDir()
	link := filepath.Join(base, "runtime")
	if err := os.MkdirAll(link, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := parseConfigRootLinks([]string{"/tmp/legacy", "root=" + link}, domain.Harness{}, true); err == nil {
		t.Fatal("parseConfigRootLinks mixed syntax error = nil, want error")
	}

	_, _, err := parseConfigRootLinks([]string{"root=" + link, "root=" + link}, domain.Harness{}, true)
	if err == nil {
		t.Fatal("parseConfigRootLinks duplicate id error = nil, want error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("parseConfigRootLinks duplicate id error = %v, want duplicate", err)
	}
}

func TestParseConfigRootLinksPreservesExistingKindForKnownIDs(t *testing.T) {
	base := t.TempDir()
	rootArtifactDir := filepath.Join(base, "root-artifact")
	if err := os.MkdirAll(rootArtifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(base, "state")
	if err := os.WriteFile(statePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	existing := domain.Harness{Links: []domain.HarnessLink{
		{ID: domain.LegacyDefaultLinkID, Kind: domain.HarnessLinkKindFile, Path: filepath.Join(base, "existing-root")},
		{ID: "state", Kind: domain.HarnessLinkKindDir, Path: filepath.Join(base, "existing-state-dir")},
	}}
	links, _, err := parseConfigRootLinks([]string{"root=unused-root", "state=" + rootArtifactDir, "notes=" + statePath}, existing, false)
	if err != nil {
		t.Fatalf("parseConfigRootLinks(existing) error = %v", err)
	}
	if links[0].Kind != domain.HarnessLinkKindFile {
		t.Fatalf("links[0].Kind = %q, want preserved kind %q", links[0].Kind, domain.HarnessLinkKindFile)
	}
	if links[1].Kind != domain.HarnessLinkKindDir {
		t.Fatalf("links[1].Kind = %q, want preserved kind %q", links[1].Kind, domain.HarnessLinkKindDir)
	}
	if links[2].Kind != domain.HarnessLinkKindFile {
		t.Fatalf("links[2].Kind = %q, want inferred file kind", links[2].Kind)
	}
}

func TestHarnessWhereDisplaysAllLinksForMultiLinkHarness(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	paths := configyaml.Paths{ConfigHome: configHome}
	repo := configyaml.NewRepository(paths)
	if err := repo.Save(domain.Config{
		HarnessesRoot: paths.HarnessesRoot(),
		Harnesses: []domain.Harness{{
			ID:    "claude",
			Label: "Claude",
			Links: []domain.HarnessLink{
				{ID: domain.LegacyDefaultLinkID, Path: filepath.Join(configHome, "claude-root"), Kind: domain.HarnessLinkKindDir},
				{ID: "state", Path: filepath.Join(configHome, "claude-state"), Kind: domain.HarnessLinkKindFile},
			},
		}},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"claude", "where"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "link:\troot\tdir\t") {
		t.Fatalf("where output = %q, want root link line", stdout.String())
	}
	if !strings.Contains(stdout.String(), "link:\tstate\tfile\t") {
		t.Fatalf("where output = %q, want state link line", stdout.String())
	}
}

func TestListMissingConfigIsEmpty(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"ls"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("ls output = %q, want empty output", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(configHome, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("config.json stat error = %v, want not exist", err)
	}
}

func TestNoArgsLaunchesTUIInInteractiveTerminal(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	called, restore := stubInteractiveTUI(t, true)
	defer restore()

	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got := *called; got != "dashboard" {
		t.Fatalf("stub output = %q, want dashboard", got)
	}
}

func TestIncompleteAddRoutesToGuidedFlowOnlyWhenInteractive(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	called, restore := stubInteractiveTUI(t, true)
	defer restore()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"add"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got := *called; got != string(tui.FlowAdd) {
		t.Fatalf("stub flow = %q, want add flow", got)
	}

	restore()
	_, restore = stubInteractiveTUI(t, false)
	defer restore()
	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"add"}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "usage: hp add") {
		t.Fatalf("Run() code = %d stderr = %q, want usage error", code, stderr.String())
	}
}

func TestCompleteSwitchRemainsScriptable(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	called, restore := stubInteractiveTUI(t, true)
	defer restore()
	paths := configyaml.Paths{ConfigHome: configHome}
	repo := configyaml.NewRepository(paths)
	harnessRoot := filepath.Join(paths.HarnessesRoot(), "claude")
	profilePath := filepath.Join(harnessRoot, "default")
	if err := os.MkdirAll(profilePath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	linkPath := filepath.Join(configHome, "claude-link")
	if err := os.Symlink(profilePath, linkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	if err := repo.Save(domain.Config{HarnessesRoot: paths.HarnessesRoot(), Harnesses: []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: linkPath}}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"claude", "switch", "default"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "switched claude to default") {
		t.Fatalf("stdout = %q, want scriptable switch output", stdout.String())
	}
	if *called != "" {
		t.Fatalf("called TUI = %q, did not expect TUI flow", *called)
	}
}

func TestScriptableHarnessLifecycleCommands(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	_, restore := stubInteractiveTUI(t, false)
	defer restore()
	root := filepath.Join(configHome, "tool-root")
	writeCLIFile(t, filepath.Join(root, "settings.json"), "{}")

	runCLI(t, []string{"add", "--label", "Tool", "--link", root, "--profile", "default", "tool"}, "tool\tTool")
	runCLI(t, []string{"tool", "ls"}, "default\t*")
	runCLI(t, []string{"tool", "current"}, "default")
	runCLI(t, []string{"tool", "where"}, "link: "+root)
	runCLI(t, []string{"tool", "where", "default"}, filepath.Join(configHome, "harnesses", "tool", "default"))
	runCLI(t, []string{"tool", "clone", "--materialize", "default", "work"}, "cloned default to work")
	runCLI(t, []string{"tool", "switch", "work"}, "switched tool to work")
	runCLI(t, []string{"tool", "delete", "--yes", "default"}, "deleted default")
	runCLI(t, []string{"update", "--label", "Tooling", "--restart", "Restart it", "tool"}, "tool\tTooling")
	runCLI(t, []string{"delete", "--mode", "keep-root", "tool"}, "deleted tool")
}

func TestScriptableProfileAdoptCommand(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	_, restore := stubInteractiveTUI(t, false)
	defer restore()
	root := filepath.Join(configHome, "tool-root")
	writeCLIFile(t, filepath.Join(root, "settings.json"), "{}")
	runCLI(t, []string{"add", "--label", "Tool", "--link", root, "--profile", "default", "tool"}, "tool\tTool")
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	writeCLIFile(t, filepath.Join(root, "external.json"), "{}")

	runCLI(t, []string{"tool", "adopt", "external"}, "adopted external")
}

func TestScriptableCommandsReportUsageAndErrors(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HP_CONFIG_HOME", configHome)
	_, restore := stubInteractiveTUI(t, false)
	defer restore()

	runCLIError(t, []string{"unknown"}, "unknown command")
	runCLIError(t, []string{"missing", "ls"}, "unknown harness")
	runCLIError(t, []string{"missing", "switch"}, "usage: hp <harness> switch")
	runCLIError(t, []string{"missing", "clone"}, "usage: hp <harness> clone")
	runCLIError(t, []string{"missing", "delete"}, "usage: hp <harness> delete")
	runCLIError(t, []string{"missing", "where", "one", "two"}, "usage: hp <harness> where")
}

func TestSplitAndParseLinkActionHelpers(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantRaw    string
		wantAction string
		want       app.HarnessLinkAction
	}{
		{name: "import", raw: "root|import", wantRaw: "root", wantAction: string(app.HarnessLinkActionImport), want: app.HarnessLinkActionImport},
		{name: "register", raw: "state|register", wantRaw: "state", wantAction: string(app.HarnessLinkActionRegister), want: app.HarnessLinkActionRegister},
		{name: "create", raw: "cache|create", wantRaw: "cache", wantAction: string(app.HarnessLinkActionCreate), want: app.HarnessLinkActionCreate},
		{name: "invalid", raw: "cache|unknown", wantRaw: "cache|unknown", wantAction: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRaw, got := splitLinkAction(tt.raw)
			if gotRaw != tt.wantRaw || got != tt.want {
				t.Fatalf("splitLinkAction(%q) = %q, %q; want %q, %q", tt.raw, gotRaw, got, tt.wantRaw, tt.want)
			}
			if tt.wantAction != "" && parseLinkAction(tt.wantAction) != tt.want {
				t.Fatalf("parseLinkAction(%q) = %q, want %q", tt.wantAction, parseLinkAction(tt.wantAction), tt.want)
			}
		})
	}
}

func TestInferLinkKindFromPath(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "dir")
	file := filepath.Join(base, "file.json")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want domain.HarnessLinkKind
	}{
		{name: "dir", path: dir, want: domain.HarnessLinkKindDir},
		{name: "file", path: file, want: domain.HarnessLinkKindFile},
		{name: "missing", path: filepath.Join(base, "missing"), want: domain.HarnessLinkKindDir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inferLinkKindFromPath(tt.path)
			if err != nil {
				t.Fatalf("inferLinkKindFromPath(%q) error = %v", tt.path, err)
			}
			if got != tt.want {
				t.Fatalf("inferLinkKindFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestValidateHarnessLinksAndLegacySingleDirectoryLink(t *testing.T) {
	valid := []domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: "/tmp/runtime", Kind: domain.HarnessLinkKindDir}}
	if err := validateHarnessLinks(valid); err != nil {
		t.Fatalf("validateHarnessLinks(valid) error = %v", err)
	}
	if !isLegacySingleDirectoryLink(valid) {
		t.Fatal("isLegacySingleDirectoryLink(valid) = false, want true")
	}

	if err := validateHarnessLinks([]domain.HarnessLink{{ID: "bad id", Path: "/tmp/runtime", Kind: domain.HarnessLinkKindDir}}); err == nil {
		t.Fatal("validateHarnessLinks(invalid) error = nil, want error")
	}
	if isLegacySingleDirectoryLink([]domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: "/tmp/runtime", Kind: domain.HarnessLinkKindFile}}) {
		t.Fatal("isLegacySingleDirectoryLink(file) = true, want false")
	}
}

func TestFormatHarnessLinksSummary(t *testing.T) {
	got := formatHarnessLinksSummary([]domain.HarnessLink{{ID: "root", Path: "/tmp/runtime", Kind: domain.HarnessLinkKindDir}, {ID: "state", Path: "/tmp/state.json", Kind: domain.HarnessLinkKindFile}})
	if !strings.Contains(got, "root:dir:/tmp/runtime") || !strings.Contains(got, "state:file:/tmp/state.json") {
		t.Fatalf("formatHarnessLinksSummary() = %q, want link summaries", got)
	}
}

func stubInteractiveTUI(t *testing.T, interactive bool) (*string, func()) {
	t.Helper()
	called := ""
	oldInteractive := interactiveTerminal
	oldRunTUI := runTUIProgram
	oldRunFlow := runTUIFlowProgram
	interactiveTerminal = func() bool { return interactive }
	runTUIProgram = func(tui.Service) error {
		called = "dashboard"
		return nil
	}
	runTUIFlowProgram = func(_ tui.Service, flow tui.Flow, _ string) error {
		called = string(flow)
		return nil
	}
	return &called, func() {
		interactiveTerminal = oldInteractive
		runTUIProgram = oldRunTUI
		runTUIFlowProgram = oldRunFlow
	}
}

func runCLI(t *testing.T, args []string, want string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run(%v) code = %d, want 0; stderr = %q", args, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), want) {
		t.Fatalf("Run(%v) stdout = %q, want %q", args, stdout.String(), want)
	}
	return stdout.String()
}

func runCLIError(t *testing.T, args []string, want string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Run(%v) code = 0, want error; stdout = %q", args, stdout.String())
	}
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("Run(%v) stderr = %q, want %q", args, stderr.String(), want)
	}
	return stderr.String()
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
