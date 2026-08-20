package app

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dovixman/harness-profiles/internal/domain"
)

type memoryRepo struct {
	config domain.Config
}

type failSaveRepo struct {
	*memoryRepo
}

func (r failSaveRepo) Save(domain.Config) error {
	return errors.New("save failed")
}

type staticPaths struct {
	paths Paths
}

func (p staticPaths) Paths() (Paths, error) {
	return p.paths, nil
}

func (r *memoryRepo) Load() (domain.Config, error) {
	return r.config, nil
}

func (r *memoryRepo) Save(config domain.Config) error {
	r.config = config
	return nil
}

func newTestService(t *testing.T) (Service, *memoryRepo, string) {
	t.Helper()
	base := t.TempDir()
	repo := &memoryRepo{config: domain.Config{HarnessesRoot: filepath.Join(base, "harnesses")}}
	return Service{Repo: repo, FS: testFS{}}, repo, base
}

type testFS struct{}

func (testFS) InspectLink(path string) (LinkInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LinkInfo{}, nil
		}
		return LinkInfo{}, err
	}
	link := LinkInfo{Exists: true, IsSymlink: info.Mode()&os.ModeSymlink != 0}
	if link.IsSymlink {
		target, err := os.Readlink(path)
		if err != nil {
			return LinkInfo{}, err
		}
		link.Target = target
	}
	return link, nil
}

func (testFS) ReadDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }

func (testFS) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

func (testFS) Lstat(path string) (os.FileInfo, error) { return os.Lstat(path) }

func (testFS) MkdirAll(path string, mode os.FileMode) error { return os.MkdirAll(path, mode) }

func (testFS) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func (testFS) ReplaceSymlink(path, target string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Symlink(target, path)
}

func (testFS) RemoveSymlink(path string) error { return os.Remove(path) }

func (testFS) MoveDir(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

func (testFS) CopyArtifact(src, dst string, materializeSymlinks bool) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dst, rel)
			entryInfo, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if rel == "." || entry.IsDir() {
				return os.MkdirAll(target, entryInfo.Mode().Perm())
			}
			if entryInfo.Mode()&os.ModeSymlink != 0 && !materializeSymlinks {
				linkTarget, err := os.Readlink(path)
				if err != nil {
					return err
				}
				return os.Symlink(linkTarget, target)
			}
			return copyTestFile(path, target, entryInfo.Mode().Perm())
		})
	}
	if !info.Mode().IsRegular() {
		return os.ErrInvalid
	}
	return copyTestFile(src, dst, info.Mode().Perm())
}

func (testFS) WriteFile(path string, contents []byte) error {
	return os.WriteFile(path, contents, 0o644)
}

func (testFS) MoveArtifact(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.Rename(src, dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyTestFile(src, dst, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Remove(src)
}

func (testFS) DeletePath(path string) error { return os.RemoveAll(path) }

func (testFS) CopyDir(src, dst string, materializeSymlinks bool) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if rel == "." || entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 && !materializeSymlinks {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		return copyTestFile(path, target, info.Mode().Perm())
	})
}

func copyTestFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func TestNormalizeConfigRootPathExpandsHomeAndRelativePaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	expanded, err := NormalizeConfigRootPath("~/.config/opencode")
	if err != nil {
		t.Fatal(err)
	}
	if expanded != filepath.Join(home, ".config", "opencode") {
		t.Fatalf("expanded = %q, want home-expanded path", expanded)
	}

	relative, err := NormalizeConfigRootPath("relative/config")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(relative) || !strings.HasSuffix(relative, filepath.Join("relative", "config")) {
		t.Fatalf("relative = %q, want absolute path ending in relative/config", relative)
	}
}

func TestServiceQueriesReturnConfiguredState(t *testing.T) {
	service, repo, base := newTestService(t)
	service.Paths = staticPaths{paths: Paths{ConfigPath: filepath.Join(base, "config.json"), HarnessesRoot: repo.config.HarnessesRoot}}
	link := filepath.Join(base, "runtime")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatal(err)
	}

	paths, err := service.Where()
	if err != nil || paths.HarnessesRoot != repo.config.HarnessesRoot {
		t.Fatalf("Where() = %+v, %v", paths, err)
	}
	harnesses, err := service.ListHarnesses()
	if err != nil || len(harnesses) != 1 || harnesses[0].ID != "claude" {
		t.Fatalf("ListHarnesses() = %+v, %v", harnesses, err)
	}
	harness, err := service.InspectHarness("claude")
	if err != nil || harness.Label != "Claude" {
		t.Fatalf("InspectHarness() = %+v, %v", harness, err)
	}
	profiles, err := service.ListProfiles("claude")
	if err != nil || len(profiles) != 1 || profiles[0].Name != "default" || !profiles[0].Active {
		t.Fatalf("ListProfiles() = %+v, %v", profiles, err)
	}
	profilePath, err := service.ProfilePath("claude", "default")
	if err != nil || !strings.HasSuffix(profilePath, filepath.Join("claude", "default")) {
		t.Fatalf("ProfilePath() = %q, %v", profilePath, err)
	}
}

func TestAddHarnessMissingRootCreatesInitialProfileAndSymlink(t *testing.T) {
	service, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime")

	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	if len(repo.config.Harnesses) != 1 {
		t.Fatalf("harnesses = %d, want 1", len(repo.config.Harnesses))
	}
	profile := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root")
	if _, err := os.Stat(profile); err != nil {
		t.Fatalf("profile root stat error = %v", err)
	}
	assertSymlinkTarget(t, link, profile)
}

func TestAddHarnessStoresNormalizedLinkPath(t *testing.T) {
	service, repo, base := newTestService(t)
	t.Chdir(base)

	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: "relative-runtime", InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	if len(repo.config.Harnesses[0].Links) != 1 {
		t.Fatalf("links = %+v, want single migrated link", repo.config.Harnesses[0].Links)
	}
	got := repo.config.Harnesses[0].Links[0].Path
	if !filepath.IsAbs(got) || strings.HasSuffix(got, string(filepath.Separator)+"~") {
		t.Fatalf("link path = %q, want normalized absolute path", got)
	}
}

func TestHarnessesMustBeUniqueByIDLabelAndRootPath(t *testing.T) {
	service, _, base := newTestService(t)
	firstRoot := filepath.Join(base, "first")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: firstRoot, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	if _, err := service.AddHarness(AddHarnessOptions{ID: "CLAUDE", Label: "Other", LinkPath: filepath.Join(base, "other"), InitialProfile: "default"}); err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("AddHarness(duplicate id) error = %v, want duplicate id", err)
	}
	if _, err := service.AddHarness(AddHarnessOptions{ID: "opencode", Label: "claude", LinkPath: filepath.Join(base, "opencode"), InitialProfile: "default"}); err == nil || !strings.Contains(err.Error(), "label") {
		t.Fatalf("AddHarness(duplicate label) error = %v, want duplicate label", err)
	}
	if _, err := service.AddHarness(AddHarnessOptions{ID: "codex", Label: "Codex", LinkPath: firstRoot, InitialProfile: "default"}); err == nil || !strings.Contains(err.Error(), "root path") {
		t.Fatalf("AddHarness(duplicate root) error = %v, want duplicate root path", err)
	}
	secondRoot := filepath.Join(base, "second")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "codex", Label: "Codex", LinkPath: secondRoot, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness(second) error = %v", err)
	}
	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "codex", Label: "CLAUDE"}); err == nil || !strings.Contains(err.Error(), "label") {
		t.Fatalf("UpdateHarness(duplicate label) error = %v, want duplicate label", err)
	}
	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "codex", LinkPath: firstRoot}); err == nil || !strings.Contains(err.Error(), "root path") {
		t.Fatalf("UpdateHarness(duplicate root) error = %v, want duplicate root path", err)
	}
	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", Label: "Claude"}); err != nil {
		t.Fatalf("UpdateHarness(same label) error = %v", err)
	}
}

func TestAddHarnessRejectsInvalidInputsAndUnapprovedExternalSymlink(t *testing.T) {
	service, _, base := newTestService(t)
	if _, err := service.AddHarness(AddHarnessOptions{ID: "bad/name", LinkPath: filepath.Join(base, "root")}); err == nil {
		t.Fatal("AddHarness(invalid id) error = nil, want error")
	}
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude"}); err == nil || !strings.Contains(err.Error(), "link path") {
		t.Fatalf("AddHarness(missing link) error = %v, want link path error", err)
	}
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", LinkPath: filepath.Join(base, "missing")}); err == nil || !strings.Contains(err.Error(), "initial profile") {
		t.Fatalf("AddHarness(missing profile) error = %v, want initial profile error", err)
	}

	external := filepath.Join(base, "external")
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "runtime")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link}); err == nil || !strings.Contains(err.Error(), "external symlink") {
		t.Fatalf("AddHarness(external symlink) error = %v, want external symlink error", err)
	}
}

func TestAddHarnessRealRootMovesInitialProfile(t *testing.T) {
	service, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	mustWrite(t, filepath.Join(link, "settings.json"), "{}")

	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	target := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root")
	assertSymlinkTarget(t, link, target)
	if _, err := os.Stat(filepath.Join(target, "settings.json")); err != nil {
		t.Fatalf("moved profile file stat error = %v", err)
	}
}

func TestAddHarnessRollsBackWhenSymlinkReplacementFails(t *testing.T) {
	_, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime.json")
	mustWrite(t, link, "current")
	faulty := &faultyUpdateFS{testFS: testFS{}, failPath: link}
	addService := Service{Repo: repo, FS: faulty}

	if _, err := addService.AddHarness(AddHarnessOptions{
		ID:             "claude",
		Label:          "Claude",
		Links:          []domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: link, Kind: domain.HarnessLinkKindFile}},
		InitialProfile: "default",
	}); err == nil {
		t.Fatal("AddHarness() error = nil, want rollback failure")
	}
	if info, err := os.Lstat(link); err != nil {
		t.Fatalf("restored file lstat error = %v", err)
	} else if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("restored file is still a symlink")
	}
	if got, err := os.ReadFile(link); err != nil || string(got) != "current" {
		t.Fatalf("restored file contents = %q, %v, want current", string(got), err)
	}
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root")); !os.IsNotExist(err) {
		t.Fatalf("artifact stat error = %v, want rolled back removal", err)
	}
}

func TestAddHarnessManagedSymlinkRegisters(t *testing.T) {
	service, repo, base := newTestService(t)
	target := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	link := filepath.Join(base, "runtime")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	assertSymlinkTarget(t, link, target)
}

func TestAddHarnessExternalSymlinkImportCopiesTarget(t *testing.T) {
	service, repo, base := newTestService(t)
	external := filepath.Join(base, "external")
	mustWrite(t, filepath.Join(external, "settings.json"), "{}")
	link := filepath.Join(base, "runtime")
	if err := os.Symlink(external, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "imported", ImportSymlink: true}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	target := filepath.Join(repo.config.HarnessesRoot, "claude", "imported", "root")
	assertSymlinkTarget(t, link, target)
	if _, err := os.Stat(filepath.Join(target, "settings.json")); err != nil {
		t.Fatalf("imported file stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(external, "settings.json")); err != nil {
		t.Fatalf("external source stat error = %v", err)
	}
}

func TestAddHarnessExternalSymlinkCanBeRegisteredWithoutImport(t *testing.T) {
	service, repo, base := newTestService(t)
	external := filepath.Join(base, "external")
	mustWrite(t, filepath.Join(external, "settings.json"), "{}")
	link := filepath.Join(base, "runtime")
	if err := os.Symlink(external, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, RegisterSymlink: true}); err != nil {
		t.Fatalf("AddHarness(register external) error = %v", err)
	}

	assertSymlinkTarget(t, link, external)
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude")); err != nil {
		t.Fatalf("managed root stat error = %v", err)
	}
}

func TestAddHarnessSupportsMixedPerLinkImportAndRegisterActions(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")
	runtimeTarget := filepath.Join(base, "external-runtime")
	stateTarget := filepath.Join(base, "external-state.json")
	if err := os.MkdirAll(runtimeTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runtimeTarget, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, stateTarget, "external")
	if err := os.Symlink(runtimeTarget, runtime); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(stateTarget, state); err != nil {
		t.Fatal(err)
	}

	if _, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		LinkActions: map[string]HarnessLinkAction{
			strings.ToLower(domain.LegacyDefaultLinkID): HarnessLinkActionImport,
			"state": HarnessLinkActionRegister,
		},
		InitialProfile: "default",
	}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	assertSymlinkTarget(t, runtime, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"))
	assertSymlinkTarget(t, state, stateTarget)
}

func TestCurrentProfileReportsExternalSymlinkTarget(t *testing.T) {
	service, repo, base := newTestService(t)
	external := filepath.Join(base, "external")
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "runtime")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	repo.config.Harnesses = []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: link}}

	current, err := service.CurrentProfile("claude")
	if err != nil {
		t.Fatalf("CurrentProfile() error = %v", err)
	}
	if !current.External || current.Path != external || current.Name != "" {
		t.Fatalf("current = %+v, want external target", current)
	}
}

func TestCurrentProfileReadsLegacyProfileStoreLayoutAsActive(t *testing.T) {
	service, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	legacyProfile := filepath.Join(repo.config.HarnessesRoot, "claude", "default")
	if err := os.MkdirAll(legacyProfile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(legacyProfile, link); err != nil {
		t.Fatal(err)
	}
	repo.config.Harnesses = []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: link}}

	current, err := service.CurrentProfile("claude")
	if err != nil {
		t.Fatalf("CurrentProfile() error = %v", err)
	}
	if current.State != ProfileStateActive {
		t.Fatalf("current.State = %q, want %q", current.State, ProfileStateActive)
	}
	if current.Name != "default" || !current.Active {
		t.Fatalf("current = %+v, want default active", current)
	}
	if len(current.Links) != 1 {
		t.Fatalf("current.Links len = %d, want 1", len(current.Links))
	}
	if !current.Links[0].NeedsMigration {
		t.Fatalf("current link = %+v, want migration flag", current.Links[0])
	}
}

func TestCurrentProfileDetectsMixedProfileSelectionAcrossLinks(t *testing.T) {
	service, repo, base := newTestService(t)
	rootLink := filepath.Join(base, "runtime")
	fileLink := filepath.Join(base, "state")
	defaultState := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state")
	otherState := filepath.Join(repo.config.HarnessesRoot, "claude", "work", "state")
	if err := os.MkdirAll(filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(defaultState, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo.config.HarnessesRoot, "claude", "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherState, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"), rootLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(otherState, fileLink); err != nil {
		t.Fatal(err)
	}
	repo.config.Harnesses = []domain.Harness{{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: rootLink, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: fileLink, Kind: domain.HarnessLinkKindFile},
		},
	}}

	current, err := service.CurrentProfile("claude")
	if err != nil {
		t.Fatalf("CurrentProfile() error = %v", err)
	}
	if current.State != ProfileStateMixed {
		t.Fatalf("current.State = %q, want %q", current.State, ProfileStateMixed)
	}
	if current.Name != "" || current.Active {
		t.Fatalf("current = %+v, want mixed without name/active", current)
	}
}

func TestProfileSwitchAdoptCloneAndDelete(t *testing.T) {
	service, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	mustWrite(t, filepath.Join(link, "config.json"), "current")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if err := service.CloneProfile("claude", "default", "work", false); err != nil {
		t.Fatalf("CloneProfile() error = %v", err)
	}
	if _, err := service.SwitchProfile("claude", "work"); err != nil {
		t.Fatalf("SwitchProfile() error = %v", err)
	}
	current, err := service.CurrentProfile("claude")
	if err != nil {
		t.Fatalf("CurrentProfile() error = %v", err)
	}
	if current.Name != "work" {
		t.Fatalf("current = %q, want work", current.Name)
	}
	if err := service.DeleteProfile("claude", "work", true); err == nil || !strings.Contains(err.Error(), "active") {
		t.Fatalf("DeleteProfile(active) error = %v, want active refusal", err)
	}
	if _, err := service.SwitchProfile("claude", "default"); err != nil {
		t.Fatalf("SwitchProfile(default) error = %v", err)
	}
	if err := service.DeleteProfile("claude", "work", true); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude", "work")); !os.IsNotExist(err) {
		t.Fatalf("deleted profile stat error = %v, want not exist", err)
	}

	if err := os.Remove(link); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	mustWrite(t, filepath.Join(link, "config.json"), "adopt")
	if err := service.AdoptProfile("claude", "adopted"); err != nil {
		t.Fatalf("AdoptProfile() error = %v", err)
	}
	assertSymlinkTarget(t, link, filepath.Join(repo.config.HarnessesRoot, "claude", "adopted", "root"))
}

func TestAdoptProfileRollsBackWhenSymlinkReplacementFails(t *testing.T) {
	_, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime.json")
	mustWrite(t, link, "current")
	repo.config.Harnesses = []domain.Harness{{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: link, Kind: domain.HarnessLinkKindFile}},
	}}
	faulty := &faultyUpdateFS{testFS: testFS{}, failPath: link}
	adoptService := Service{Repo: repo, FS: faulty}

	if err := adoptService.AdoptProfile("claude", "adopted"); err == nil {
		t.Fatal("AdoptProfile() error = nil, want rollback failure")
	}
	if info, err := os.Lstat(link); err != nil {
		t.Fatalf("restored file lstat error = %v", err)
	} else if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("restored file is still a symlink")
	}
	if got, err := os.ReadFile(link); err != nil || string(got) != "current" {
		t.Fatalf("restored file contents = %q, %v, want current", string(got), err)
	}
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude", "adopted", "root")); !os.IsNotExist(err) {
		t.Fatalf("artifact stat error = %v, want rolled back removal", err)
	}
}

func TestCloneProfileMaterializeSymlink(t *testing.T) {
	service, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	mustWrite(t, filepath.Join(link, "real.txt"), "real")
	if err := os.Symlink("real.txt", filepath.Join(link, "linked.txt")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	if err := service.CloneProfile("claude", "default", "copy", true); err != nil {
		t.Fatalf("CloneProfile() error = %v", err)
	}
	info, err := os.Lstat(filepath.Join(repo.config.HarnessesRoot, "claude", "copy", "root", "linked.txt"))
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("materialized linked.txt is still a symlink")
	}
}

func TestCreateAndRenameProfile(t *testing.T) {
	service, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	mustWrite(t, filepath.Join(link, "config.json"), "{}")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if err := service.CreateProfile("claude", "empty"); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude", "empty", "root")); err != nil {
		t.Fatalf("created profile stat error = %v", err)
	}
	if err := service.RenameProfile("claude", "default", "renamed"); err != nil {
		t.Fatalf("RenameProfile() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude", "default")); !os.IsNotExist(err) {
		t.Fatalf("old profile stat error = %v, want not exist", err)
	}
	assertSymlinkTarget(t, link, filepath.Join(repo.config.HarnessesRoot, "claude", "renamed", "root"))
}

func TestCreateProfileUsesRootLayoutForLegacyHarness(t *testing.T) {
	service, repo, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	mustWrite(t, filepath.Join(link, "config.json"), "{}")
	repo.config.Harnesses = []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: link}}

	if err := service.CreateProfile("claude", "empty"); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude", "empty", "root")); err != nil {
		t.Fatalf("legacy profile root stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude", "empty", "root", "config.json")); !os.IsNotExist(err) {
		t.Fatalf("unexpected legacy config file stat error = %v, want empty root layout", err)
	}
}

func TestProfilesMustBeUniqueCaseInsensitive(t *testing.T) {
	service, _, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	mustWrite(t, filepath.Join(link, "config.json"), "{}")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}

	if err := service.CreateProfile("claude", "DEFAULT"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateProfile(duplicate) error = %v, want duplicate", err)
	}
	if err := service.CloneProfile("claude", "default", "Default", false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CloneProfile(duplicate) error = %v, want duplicate", err)
	}
	if err := service.CreateProfile("claude", "work"); err != nil {
		t.Fatalf("CreateProfile(work) error = %v", err)
	}
	if err := service.RenameProfile("claude", "work", "DEFAULT"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("RenameProfile(duplicate) error = %v, want duplicate", err)
	}
	if err := service.AdoptProfile("claude", "Default"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("AdoptProfile(duplicate) error = %v, want duplicate", err)
	}
}

func TestUpdateHarnessRootPathPersistsAfterSymlinkSucceeds(t *testing.T) {
	service, repo, base := newTestService(t)
	oldLink := filepath.Join(base, "old")
	mustWrite(t, filepath.Join(oldLink, "config.json"), "{}")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: oldLink, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	blocker := filepath.Join(base, "blocker")
	mustWrite(t, blocker, "file")

	_, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", LinkPath: filepath.Join(blocker, "new")})
	if err == nil {
		t.Fatal("UpdateHarness() error = nil, want symlink creation failure")
	}
	if len(repo.config.Harnesses[0].Links) != 1 || repo.config.Harnesses[0].Links[0].Path != oldLink {
		t.Fatalf("links = %+v, want unchanged %q", repo.config.Harnesses[0].Links, oldLink)
	}

	newLink := filepath.Join(base, "new")
	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", LinkPath: newLink, RemoveOld: true}); err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}
	if len(repo.config.Harnesses[0].Links) != 1 || repo.config.Harnesses[0].Links[0].Path != newLink {
		t.Fatalf("links = %+v, want migrated root link %q", repo.config.Harnesses[0].Links, newLink)
	}
	if _, err := os.Lstat(oldLink); !os.IsNotExist(err) {
		t.Fatalf("old link stat error = %v, want not exist", err)
	}
}

func TestUpdateHarnessMigratesLegacySingleLinkLayoutToRoot(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	legacyProfile := filepath.Join(repo.config.HarnessesRoot, "claude", "default")
	mustWrite(t, filepath.Join(legacyProfile, "config.json"), "current")
	if err := os.Symlink(legacyProfile, runtime); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	repo.config.Harnesses = []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: runtime}}

	updated, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", Label: "Claude Code"})
	if err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}
	if len(updated.Links) != 1 || updated.LinkPath != "" || updated.Links[0].ID != domain.LegacyDefaultLinkID {
		t.Fatalf("updated = %+v, want migrated root link", updated)
	}

	rootArtifact := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root")
	assertSymlinkTarget(t, runtime, rootArtifact)
	if _, err := os.Stat(filepath.Join(rootArtifact, "config.json")); err != nil {
		t.Fatalf("migrated root config stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyProfile, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("legacy profile file stat error = %v, want moved into root", err)
	}
}

func TestUpdateHarnessRejectsLegacyMigrationConflicts(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	legacyProfile := filepath.Join(repo.config.HarnessesRoot, "claude", "default")
	if err := os.MkdirAll(filepath.Join(legacyProfile, "root"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(legacyProfile, "config.json"), "legacy")
	if err := os.Symlink(legacyProfile, runtime); err != nil {
		t.Fatal(err)
	}
	repo.config.Harnesses = []domain.Harness{{ID: "claude", Label: "Claude", LinkPath: runtime}}

	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", Label: "Claude Code"}); err == nil || !strings.Contains(err.Error(), "repair required") {
		t.Fatalf("UpdateHarness() error = %v, want migration conflict", err)
	}
}

func TestAddHarnessSupportsExplicitMultipleLinks(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")

	harness, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		InitialProfile: "default",
	})
	if err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if len(harness.Links) != 2 || harness.LinkPath != "" {
		t.Fatalf("harness = %+v, want explicit links persisted", harness)
	}

	rootArtifact := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root")
	stateArtifact := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state")
	assertSymlinkTarget(t, runtime, rootArtifact)
	assertSymlinkTarget(t, state, stateArtifact)
	if info, err := os.Stat(stateArtifact); err != nil || info.IsDir() {
		t.Fatalf("state artifact = %v, %v, want file", info, err)
	}
}

func TestAddHarnessRejectsDuplicateNormalizedLinkPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "tilde", path: "~/runtime"},
		{name: "relative", path: filepath.Join("runtime", "..", "runtime")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _, base := newTestService(t)
			t.Setenv("HOME", base)
			t.Chdir(base)

			_, err := service.AddHarness(AddHarnessOptions{
				ID:    "claude",
				Label: "Claude",
				Links: []domain.HarnessLink{
					{ID: domain.LegacyDefaultLinkID, Path: filepath.Join(base, "runtime"), Kind: domain.HarnessLinkKindDir},
					{ID: "state", Path: tt.path, Kind: domain.HarnessLinkKindDir},
				},
				InitialProfile: "default",
			})
			if err == nil || (!strings.Contains(err.Error(), "duplicate harness link path") && !strings.Contains(err.Error(), "root path")) {
				t.Fatalf("AddHarness() error = %v, want duplicate path error", err)
			}
		})
	}
}

func TestUpdateHarnessSupportsExplicitMultipleLinks(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")
	if _, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		InitialProfile: "default",
	}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	updatedState := filepath.Join(base, "runtime-state.json")

	updated, err := service.UpdateHarness(UpdateHarnessOptions{
		ID:    "claude",
		Label: "Claude Code",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: updatedState, Kind: domain.HarnessLinkKindFile},
		},
		RemoveOld: true,
	})
	if err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}
	if updated.Label != "Claude Code" || len(updated.Links) != 2 {
		t.Fatalf("updated = %+v, want explicit links", updated)
	}
	assertSymlinkTarget(t, updatedState, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state"))
	if _, err := os.Lstat(state); !os.IsNotExist(err) {
		t.Fatalf("old state link stat error = %v, want not exist", err)
	}
}

func TestUpdateHarnessAddsMissingLinkArtifactsToEveryProfile(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: runtime, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if err := service.CreateProfile("claude", "work"); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	state := filepath.Join(base, "state.json")

	updated, err := service.UpdateHarness(UpdateHarnessOptions{
		ID: "claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
	})
	if err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}
	if len(updated.Links) != 2 {
		t.Fatalf("updated links = %+v, want two links", updated.Links)
	}
	defaultState := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state")
	workState := filepath.Join(repo.config.HarnessesRoot, "claude", "work", "state")
	assertSymlinkTarget(t, state, defaultState)
	for _, path := range []string{defaultState, workState} {
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("state artifact %q = %v, %v, want regular file", path, info, err)
		}
	}
}

func TestUpdateHarnessImportsNewLinkAndUnregistersRemovedLink(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: runtime, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if err := service.CreateProfile("claude", "work"); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	state := filepath.Join(base, "state.json")
	mustWrite(t, state, "current state")
	links := []domain.HarnessLink{
		{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
		{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
	}

	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", Links: links, LinkActions: map[string]HarnessLinkAction{"state": HarnessLinkActionImport}}); err != nil {
		t.Fatalf("UpdateHarness(add link) error = %v", err)
	}
	defaultState := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state")
	assertSymlinkTarget(t, state, defaultState)
	contents, err := os.ReadFile(defaultState)
	if err != nil || string(contents) != "current state" {
		t.Fatalf("imported state = %q, %v, want current state", contents, err)
	}

	updated, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", Links: links[:1], RemoveOld: true})
	if err != nil {
		t.Fatalf("UpdateHarness(remove link) error = %v", err)
	}
	if len(updated.Links) != 1 || updated.Links[0].ID != domain.LegacyDefaultLinkID {
		t.Fatalf("updated links = %+v, want only root", updated.Links)
	}
	if _, err := os.Lstat(state); !os.IsNotExist(err) {
		t.Fatalf("removed link stat error = %v, want missing", err)
	}
	for _, profile := range []string{"default", "work"} {
		artifact := filepath.Join(repo.config.HarnessesRoot, "claude", profile, "state")
		if _, err := os.Stat(artifact); err != nil {
			t.Fatalf("preserved artifact %q stat error = %v", artifact, err)
		}
	}
}

func TestUpdateHarnessRemovesRegisteredExternalLinkWithoutManagedCurrentProfile(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: runtime, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	external := filepath.Join(base, "external.json")
	state := filepath.Join(base, "state.json")
	mustWrite(t, external, "external")
	if err := os.Symlink(external, state); err != nil {
		t.Fatal(err)
	}
	links := []domain.HarnessLink{
		{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
		{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
	}
	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", Links: links, LinkActions: map[string]HarnessLinkAction{"state": HarnessLinkActionRegister}}); err != nil {
		t.Fatalf("UpdateHarness(register link) error = %v", err)
	}
	assertSymlinkTarget(t, state, external)

	if _, err := service.UpdateHarness(UpdateHarnessOptions{ID: "claude", Links: links[:1], RemoveOld: true}); err != nil {
		t.Fatalf("UpdateHarness(remove external link) error = %v", err)
	}
	if _, err := os.Lstat(state); !os.IsNotExist(err) {
		t.Fatalf("registered link stat error = %v, want removed", err)
	}
	if contents, err := os.ReadFile(external); err != nil || string(contents) != "external" {
		t.Fatalf("external target = %q, %v, want preserved", contents, err)
	}
	if len(repo.config.Harnesses[0].Links) != 1 {
		t.Fatalf("persisted links = %+v, want one", repo.config.Harnesses[0].Links)
	}
}

func TestUpdateHarnessRollsBackFilesystemWhenConfigSaveFails(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: runtime, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	newRuntime := filepath.Join(base, "new-runtime")
	failing := Service{Repo: failSaveRepo{memoryRepo: repo}, FS: testFS{}}

	_, err := failing.UpdateHarness(UpdateHarnessOptions{
		ID:        "claude",
		Links:     []domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: newRuntime, Kind: domain.HarnessLinkKindDir}},
		RemoveOld: true,
	})
	if err == nil || !strings.Contains(err.Error(), "save failed") {
		t.Fatalf("UpdateHarness() error = %v, want save failure", err)
	}
	assertSymlinkTarget(t, runtime, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"))
	if _, err := os.Lstat(newRuntime); !os.IsNotExist(err) {
		t.Fatalf("new runtime stat error = %v, want rollback removal", err)
	}
}

func TestUpdateHarnessRollsBackPartialMultiLinkReplacement(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")
	if _, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		InitialProfile: "default",
	}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	updatedState := filepath.Join(base, "runtime-state.json")
	faulty := &faultyUpdateFS{testFS: testFS{}, failPath: updatedState}
	updateService := Service{Repo: repo, FS: faulty}
	if _, err := updateService.UpdateHarness(UpdateHarnessOptions{
		ID: "claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: updatedState, Kind: domain.HarnessLinkKindFile},
		},
	}); err == nil {
		t.Fatal("UpdateHarness() error = nil, want rollback failure")
	}
	assertSymlinkTarget(t, runtime, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"))
	assertSymlinkTarget(t, state, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state"))
	if _, err := os.Lstat(updatedState); !os.IsNotExist(err) {
		t.Fatalf("updated state stat error = %v, want not exist", err)
	}
}

func TestSwitchProfileRollsBackPartialLinkReplacement(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")
	if _, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		InitialProfile: "default",
	}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if err := service.CreateProfile("claude", "work"); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	faulty := &faultySwitchFS{testFS: testFS{}, failPath: state}
	switchService := Service{Repo: repo, FS: faulty}
	if _, err := switchService.SwitchProfile("claude", "work"); err == nil {
		t.Fatal("SwitchProfile() error = nil, want rollback failure")
	}
	assertSymlinkTarget(t, runtime, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"))
	assertSymlinkTarget(t, state, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state"))
}

func TestSwitchProfilePreflightsAllDestinationPaths(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")
	if _, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		InitialProfile: "default",
	}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if err := service.CreateProfile("claude", "work"); err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	if err := os.Remove(state); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if err := os.WriteFile(state, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := service.SwitchProfile("claude", "work"); err == nil || !strings.Contains(err.Error(), "not a symlink") {
		t.Fatalf("SwitchProfile() error = %v, want preflight failure", err)
	}
	assertSymlinkTarget(t, runtime, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"))
}

func TestDeleteHarnessModes(t *testing.T) {
	t.Run("keep root", func(t *testing.T) {
		service, repo, base := newTestService(t)
		link := filepath.Join(base, "runtime")
		mustWrite(t, filepath.Join(link, "config.json"), "{}")
		if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
			t.Fatalf("AddHarness() error = %v", err)
		}
		if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "keep-root"}); err != nil {
			t.Fatalf("DeleteHarness() error = %v", err)
		}
		if _, err := os.Lstat(link); err != nil {
			t.Fatalf("kept root lstat error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude")); !os.IsNotExist(err) {
			t.Fatalf("harness root stat error = %v, want not exist", err)
		}
	})

	t.Run("restore", func(t *testing.T) {
		service, _, base := newTestService(t)
		link := filepath.Join(base, "runtime")
		mustWrite(t, filepath.Join(link, "config.json"), "{}")
		if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
			t.Fatalf("AddHarness() error = %v", err)
		}
		if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "restore", RestoreProfile: "default"}); err != nil {
			t.Fatalf("DeleteHarness() error = %v", err)
		}
		info, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("restored root lstat error = %v", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("restored root is still a symlink")
		}
	})

	t.Run("delete all", func(t *testing.T) {
		service, _, base := newTestService(t)
		link := filepath.Join(base, "runtime")
		mustWrite(t, filepath.Join(link, "config.json"), "{}")
		if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
			t.Fatalf("AddHarness() error = %v", err)
		}
		if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "delete-all"}); err == nil {
			t.Fatal("DeleteHarness(delete-all without confirm) error = nil, want error")
		}
		if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "delete-all", Confirm: "claude"}); err != nil {
			t.Fatalf("DeleteHarness(delete-all) error = %v", err)
		}
		if _, err := os.Lstat(link); !os.IsNotExist(err) {
			t.Fatalf("deleted link stat error = %v, want not exist", err)
		}
	})
}

func TestDeleteHarnessUsesEveryConfiguredLink(t *testing.T) {
	t.Run("restore", func(t *testing.T) {
		service, repo, base := newTestService(t)
		runtime := filepath.Join(base, "runtime")
		state := filepath.Join(base, "runtime.json")
		if _, err := service.AddHarness(AddHarnessOptions{
			ID:    "claude",
			Label: "Claude",
			Links: []domain.HarnessLink{
				{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
				{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
			},
			InitialProfile: "default",
		}); err != nil {
			t.Fatalf("AddHarness() error = %v", err)
		}
		if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "restore", RestoreProfile: "default"}); err != nil {
			t.Fatalf("DeleteHarness() error = %v", err)
		}
		if info, err := os.Lstat(runtime); err != nil {
			t.Fatalf("restored runtime lstat error = %v", err)
		} else if info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("restored runtime is still a symlink")
		}
		if info, err := os.Lstat(state); err != nil {
			t.Fatalf("restored state lstat error = %v", err)
		} else if info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("restored state is still a symlink")
		}
		if _, err := os.Stat(filepath.Join(repo.config.HarnessesRoot, "claude")); !os.IsNotExist(err) {
			t.Fatalf("harness root stat error = %v, want not exist", err)
		}
	})

	t.Run("delete all", func(t *testing.T) {
		service, _, base := newTestService(t)
		runtime := filepath.Join(base, "runtime")
		state := filepath.Join(base, "runtime.json")
		if _, err := service.AddHarness(AddHarnessOptions{
			ID:    "claude",
			Label: "Claude",
			Links: []domain.HarnessLink{
				{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
				{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
			},
			InitialProfile: "default",
		}); err != nil {
			t.Fatalf("AddHarness() error = %v", err)
		}
		if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "delete-all", Confirm: "claude"}); err != nil {
			t.Fatalf("DeleteHarness() error = %v", err)
		}
		if _, err := os.Lstat(runtime); !os.IsNotExist(err) {
			t.Fatalf("runtime stat error = %v, want not exist", err)
		}
		if _, err := os.Lstat(state); !os.IsNotExist(err) {
			t.Fatalf("state stat error = %v, want not exist", err)
		}
	})
}

func TestDeleteHarnessRestoreRollsBackPartialFailureAcrossLinks(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")
	if _, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		InitialProfile: "default",
	}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	faulty := &faultyDeleteFS{testFS: testFS{}, failPath: state}
	deleteService := Service{Repo: repo, FS: faulty}
	if err := deleteService.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "restore", RestoreProfile: "default"}); err == nil {
		t.Fatal("DeleteHarness() error = nil, want rollback failure")
	}
	assertSymlinkTarget(t, runtime, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "root"))
	assertSymlinkTarget(t, state, filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state"))
}

func TestDeleteHarnessRestorePreflightsArtifactKindsBeforeMutation(t *testing.T) {
	service, repo, base := newTestService(t)
	runtime := filepath.Join(base, "runtime")
	state := filepath.Join(base, "runtime.json")
	if _, err := service.AddHarness(AddHarnessOptions{
		ID:    "claude",
		Label: "Claude",
		Links: []domain.HarnessLink{
			{ID: domain.LegacyDefaultLinkID, Path: runtime, Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: state, Kind: domain.HarnessLinkKindFile},
		},
		InitialProfile: "default",
	}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	corrupted := filepath.Join(repo.config.HarnessesRoot, "claude", "default", "state")
	if err := os.RemoveAll(corrupted); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(corrupted, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "restore", RestoreProfile: "default"}); err == nil {
		t.Fatal("DeleteHarness() error = nil, want artifact kind validation failure")
	}
	if info, err := os.Lstat(runtime); err != nil {
		t.Fatalf("runtime lstat error = %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("runtime link changed before preflight failure")
	}
	if info, err := os.Lstat(state); err != nil {
		t.Fatalf("state lstat error = %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("state link changed before preflight failure")
	}
}

type faultySwitchFS struct {
	testFS
	failPath string
}

func (f *faultySwitchFS) ReplaceSymlink(path, target string) error {
	if path == f.failPath {
		return errors.New("forced switch failure")
	}
	return f.testFS.ReplaceSymlink(path, target)
}

type faultyUpdateFS struct {
	testFS
	failPath string
}

func (f *faultyUpdateFS) ReplaceSymlink(path, target string) error {
	if path == f.failPath {
		return errors.New("forced update failure")
	}
	return f.testFS.ReplaceSymlink(path, target)
}

type faultyDeleteFS struct {
	testFS
	failPath string
}

func (f *faultyDeleteFS) CopyArtifact(src, dst string, materializeSymlinks bool) error {
	if dst == f.failPath {
		return errors.New("forced delete failure")
	}
	return f.testFS.CopyArtifact(src, dst, materializeSymlinks)
}

func TestDeleteHarnessRejectsInvalidModeAndRestoreProfile(t *testing.T) {
	service, _, base := newTestService(t)
	link := filepath.Join(base, "runtime")
	mustWrite(t, filepath.Join(link, "config.json"), "{}")
	if _, err := service.AddHarness(AddHarnessOptions{ID: "claude", Label: "Claude", LinkPath: link, InitialProfile: "default"}); err != nil {
		t.Fatalf("AddHarness() error = %v", err)
	}
	if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "bad"}); err == nil || !strings.Contains(err.Error(), "delete mode") {
		t.Fatalf("DeleteHarness(bad mode) error = %v, want mode error", err)
	}
	if err := service.DeleteHarness(DeleteHarnessOptions{ID: "claude", Mode: "restore", RestoreProfile: "bad/name"}); err == nil || !strings.Contains(err.Error(), "path separators") {
		t.Fatalf("DeleteHarness(bad restore profile) error = %v, want profile validation error", err)
	}
}

func TestCurrentProfileAndAggregateStateHelpers(t *testing.T) {
	t.Run("current profile for link covers every state", func(t *testing.T) {
		service, _, base := newTestService(t)
		config := domain.Config{HarnessesRoot: filepath.Join(base, "harnesses")}
		harnessID := "claude"
		root := filepath.Join(config.HarnessesRoot, harnessID)
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}

		linkTo := func(linkPath, target string) {
			t.Helper()
			rel, err := filepath.Rel(filepath.Dir(linkPath), target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(rel, linkPath); err != nil {
				t.Fatal(err)
			}
		}

		activeDir := filepath.Join(root, "work", domain.LegacyDefaultLinkID)
		if err := os.MkdirAll(activeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		activeDirLink := filepath.Join(base, "runtime")
		linkTo(activeDirLink, activeDir)
		status, err := service.currentProfileForLink(config, harnessID, domain.HarnessLink{ID: domain.LegacyDefaultLinkID, Path: activeDirLink, Kind: domain.HarnessLinkKindDir}, false)
		if err != nil {
			t.Fatalf("currentProfileForLink(dir) error = %v", err)
		}
		if status.State != ProfileStateActive || status.Profile != "work" || status.ArtifactPath != activeDir || status.NeedsMigration {
			t.Fatalf("currentProfileForLink(dir) = %+v, want active work", status)
		}

		activeFile := filepath.Join(root, "work", "state")
		mustWrite(t, activeFile, "payload")
		activeFileLink := filepath.Join(base, "state.json")
		linkTo(activeFileLink, activeFile)
		status, err = service.currentProfileForLink(config, harnessID, domain.HarnessLink{ID: "state", Path: activeFileLink, Kind: domain.HarnessLinkKindFile}, false)
		if err != nil {
			t.Fatalf("currentProfileForLink(file) error = %v", err)
		}
		if status.State != ProfileStateActive || status.Profile != "work" || status.ArtifactPath != activeFile {
			t.Fatalf("currentProfileForLink(file) = %+v, want active work", status)
		}

		missingStatus, err := service.currentProfileForLink(config, harnessID, domain.HarnessLink{ID: "missing", Path: filepath.Join(base, "missing"), Kind: domain.HarnessLinkKindDir}, false)
		if err != nil {
			t.Fatalf("currentProfileForLink(missing) error = %v", err)
		}
		if missingStatus.State != ProfileStateMissing {
			t.Fatalf("currentProfileForLink(missing) = %+v, want missing", missingStatus)
		}

		plain := filepath.Join(base, "plain")
		mustWrite(t, plain, "raw")
		status, err = service.currentProfileForLink(config, harnessID, domain.HarnessLink{ID: "plain", Path: plain, Kind: domain.HarnessLinkKindFile}, false)
		if err != nil {
			t.Fatalf("currentProfileForLink(plain) error = %v", err)
		}
		if status.State != ProfileStateUnknown || status.ArtifactPath != plain {
			t.Fatalf("currentProfileForLink(plain) = %+v, want unknown", status)
		}

		external := filepath.Join(base, "external")
		if err := os.MkdirAll(external, 0o755); err != nil {
			t.Fatal(err)
		}
		externalLink := filepath.Join(base, "external-link")
		linkTo(externalLink, external)
		status, err = service.currentProfileForLink(config, harnessID, domain.HarnessLink{ID: "external", Path: externalLink, Kind: domain.HarnessLinkKindDir}, false)
		if err != nil {
			t.Fatalf("currentProfileForLink(external) error = %v", err)
		}
		if status.State != ProfileStateExternal || status.Target != external {
			t.Fatalf("currentProfileForLink(external) = %+v, want external target", status)
		}

		mismatchLink := filepath.Join(base, "mismatch")
		linkTo(mismatchLink, activeFile)
		status, err = service.currentProfileForLink(config, harnessID, domain.HarnessLink{ID: "state", Path: mismatchLink, Kind: domain.HarnessLinkKindDir}, false)
		if err != nil {
			t.Fatalf("currentProfileForLink(mismatch) error = %v", err)
		}
		if status.State != ProfileStateUnknown {
			t.Fatalf("currentProfileForLink(mismatch) = %+v, want unknown", status)
		}

		legacyProfile := filepath.Join(root, "legacy")
		if err := os.MkdirAll(legacyProfile, 0o755); err != nil {
			t.Fatal(err)
		}
		legacyLink := filepath.Join(base, "legacy-link")
		linkTo(legacyLink, legacyProfile)
		status, err = service.currentProfileForLink(config, harnessID, domain.HarnessLink{ID: domain.LegacyDefaultLinkID, Path: legacyLink, Kind: domain.HarnessLinkKindDir}, true)
		if err != nil {
			t.Fatalf("currentProfileForLink(legacy) error = %v", err)
		}
		if status.State != ProfileStateActive || status.Profile != "legacy" || !status.NeedsMigration || status.ArtifactPath != legacyProfile {
			t.Fatalf("currentProfileForLink(legacy) = %+v, want active legacy migration", status)
		}
	})

	t.Run("aggregate profile state summarizes mixed and external combinations", func(t *testing.T) {
		service := Service{}
		cases := []struct {
			name  string
			input []ProfileLinkStatus
			want  ProfileStatus
		}{
			{name: "empty"},
			{name: "active", input: []ProfileLinkStatus{{State: ProfileStateActive, Profile: "work", ArtifactPath: "/tmp/work/root"}}, want: ProfileStatus{Name: "work", Path: "/tmp/work/root", Active: true, State: ProfileStateActive}},
			{name: "active mismatch", input: []ProfileLinkStatus{{State: ProfileStateActive, Profile: "work"}, {State: ProfileStateActive, Profile: "other"}}, want: ProfileStatus{State: ProfileStateMixed}},
			{name: "missing", input: []ProfileLinkStatus{{State: ProfileStateMissing, Profile: "work"}}, want: ProfileStatus{Name: "work", State: ProfileStateMissing}},
			{name: "missing mismatch", input: []ProfileLinkStatus{{State: ProfileStateMissing, Profile: "work"}, {State: ProfileStateMissing, Profile: "other"}}, want: ProfileStatus{State: ProfileStateMixed}},
			{name: "external", input: []ProfileLinkStatus{{State: ProfileStateExternal, Target: "/tmp/external"}}, want: ProfileStatus{Path: "/tmp/external", External: true, State: ProfileStateExternal}},
			{name: "unknown", input: []ProfileLinkStatus{{State: ProfileStateUnknown}}, want: ProfileStatus{State: ProfileStateUnknown}},
			{name: "missing and external", input: []ProfileLinkStatus{{State: ProfileStateMissing, Profile: "work"}, {State: ProfileStateExternal, Target: "/tmp/external"}}, want: ProfileStatus{Path: "/tmp/external", State: ProfileStateMixed}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := service.aggregateProfileState(tc.input)
				if got.State != tc.want.State || got.Name != tc.want.Name || got.Path != tc.want.Path || got.Active != tc.want.Active || got.External != tc.want.External {
					t.Fatalf("aggregateProfileState() = %+v, want %+v", got, tc.want)
				}
			})
		}
	})
}

func TestProfileArtifactHelpersCoverLegacyCompatibilityAndValidation(t *testing.T) {
	t.Run("profileArtifacts and profileArtifactsWithKinds handle canonical and legacy layouts", func(t *testing.T) {
		service, _, base := newTestService(t)
		config := domain.Config{HarnessesRoot: filepath.Join(base, "harnesses")}
		multiHarness := domain.Harness{ID: "claude", Label: "Claude", Links: []domain.HarnessLink{
			{ID: "agents", Path: filepath.Join(base, "agents"), Kind: domain.HarnessLinkKindDir},
			{ID: "state", Path: filepath.Join(base, "state.json"), Kind: domain.HarnessLinkKindFile},
		}}

		artifacts, err := service.profileArtifacts(config, multiHarness, "work")
		if err != nil {
			t.Fatalf("profileArtifacts() error = %v", err)
		}
		if len(artifacts) != 2 || artifacts[0].ArtifactPath != filepath.Join(config.HarnessesRoot, "claude", "work", "agents") || artifacts[1].ArtifactPath != filepath.Join(config.HarnessesRoot, "claude", "work", "state") {
			t.Fatalf("profileArtifacts() = %+v, want canonical artifact paths", artifacts)
		}

		if err := os.MkdirAll(artifacts[0].ArtifactPath, 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, artifacts[1].ArtifactPath, "payload")

		withKinds, err := service.profileArtifactsWithKinds(config, multiHarness, "work")
		if err != nil {
			t.Fatalf("profileArtifactsWithKinds() error = %v", err)
		}
		if len(withKinds) != 2 || withKinds[0].State != ProfileStateActive || withKinds[1].State != ProfileStateActive {
			t.Fatalf("profileArtifactsWithKinds() = %+v, want active artifacts", withKinds)
		}

		wrongKinds := multiHarness
		wrongKinds.Links = append([]domain.HarnessLink(nil), multiHarness.Links...)
		wrongKinds.Links[1].Kind = domain.HarnessLinkKindDir
		if _, err := service.profileArtifactsWithKinds(config, wrongKinds, "work"); err == nil || !strings.Contains(err.Error(), "expected dir") {
			t.Fatalf("profileArtifactsWithKinds(wrong kind) error = %v, want kind mismatch", err)
		}

		missingHarness := domain.Harness{ID: "claude", Label: "Claude", Links: []domain.HarnessLink{{ID: "agents", Path: filepath.Join(base, "missing"), Kind: domain.HarnessLinkKindDir}}}
		if _, err := service.profileArtifactsWithKinds(config, missingHarness, "missing"); err == nil || !strings.Contains(err.Error(), "missing") {
			t.Fatalf("profileArtifactsWithKinds(missing) error = %v, want missing artifact", err)
		}

		legacyProfile := filepath.Join(config.HarnessesRoot, "claude", "default")
		if err := os.MkdirAll(legacyProfile, 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(legacyProfile, "config.json"), "legacy")
		legacyHarness := domain.Harness{ID: "claude", Label: "Claude", LinkPath: filepath.Join(base, "runtime")}

		legacyArtifacts, err := service.profileArtifacts(config, legacyHarness, "default")
		if err != nil {
			t.Fatalf("profileArtifacts(legacy) error = %v", err)
		}
		if len(legacyArtifacts) != 1 || legacyArtifacts[0].ArtifactPath != legacyProfile || !legacyArtifacts[0].NeedsMigration || legacyArtifacts[0].ID != domain.LegacyDefaultLinkID {
			t.Fatalf("profileArtifacts(legacy) = %+v, want legacy profile compatibility", legacyArtifacts)
		}

		legacyKinds, err := service.profileArtifactsWithKinds(config, legacyHarness, "default")
		if err != nil {
			t.Fatalf("profileArtifactsWithKinds(legacy) error = %v", err)
		}
		if len(legacyKinds) != 1 || legacyKinds[0].State != ProfileStateActive || legacyKinds[0].ArtifactPath != legacyProfile {
			t.Fatalf("profileArtifactsWithKinds(legacy) = %+v, want active legacy artifact", legacyKinds)
		}
	})

	t.Run("ensureArtifactForLink creates and migrates managed artifacts", func(t *testing.T) {
		service, _, base := newTestService(t)

		createConfig := domain.Config{HarnessesRoot: filepath.Join(base, "create")}
		if err := service.ensureArtifactForLink(createConfig, "claude", "work", domain.HarnessLink{ID: "agents", Kind: domain.HarnessLinkKindDir}, false); err != nil {
			t.Fatalf("ensureArtifactForLink(dir) error = %v", err)
		}
		if info, err := os.Stat(filepath.Join(createConfig.HarnessesRoot, "claude", "work", "agents")); err != nil || !info.IsDir() {
			t.Fatalf("dir artifact stat = %v, %v, want directory", info, err)
		}
		if err := service.ensureArtifactForLink(createConfig, "claude", "work", domain.HarnessLink{ID: "state", Kind: domain.HarnessLinkKindFile}, false); err != nil {
			t.Fatalf("ensureArtifactForLink(file) error = %v", err)
		}
		if info, err := os.Stat(filepath.Join(createConfig.HarnessesRoot, "claude", "work", "state")); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("file artifact stat = %v, %v, want regular file", info, err)
		}

		migrateConfig := domain.Config{HarnessesRoot: filepath.Join(base, "migrate")}
		legacyProfile := filepath.Join(migrateConfig.HarnessesRoot, "claude", "default")
		if err := os.MkdirAll(legacyProfile, 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(legacyProfile, "config.json"), "legacy")
		if err := service.ensureArtifactForLink(migrateConfig, "claude", "default", domain.HarnessLink{ID: domain.LegacyDefaultLinkID, Kind: domain.HarnessLinkKindDir}, true); err != nil {
			t.Fatalf("ensureArtifactForLink(migrate) error = %v", err)
		}
		if _, err := os.Stat(filepath.Join(legacyProfile, domain.LegacyDefaultLinkID, "config.json")); err != nil {
			t.Fatalf("migrated artifact missing = %v", err)
		}
		if _, err := os.Stat(filepath.Join(legacyProfile, "config.json")); !os.IsNotExist(err) {
			t.Fatalf("legacy root file still present = %v, want removed", err)
		}

		conflictConfig := domain.Config{HarnessesRoot: filepath.Join(base, "conflict")}
		conflictLegacy := filepath.Join(conflictConfig.HarnessesRoot, "claude", "default")
		if err := os.MkdirAll(filepath.Join(conflictLegacy, domain.LegacyDefaultLinkID), 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(conflictLegacy, "config.json"), "legacy")
		if err := service.migrateLegacyProfileArtifact(conflictConfig, "claude", "default", filepath.Join(conflictLegacy, domain.LegacyDefaultLinkID)); err == nil || !strings.Contains(err.Error(), "repair required") {
			t.Fatalf("migrateLegacyProfileArtifact(conflict) error = %v, want repair-needed conflict", err)
		}
	})
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func assertSymlinkTarget(t *testing.T, link, target string) {
	t.Helper()
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if got != target {
		t.Fatalf("symlink target = %q, want %q", got, target)
	}
}
