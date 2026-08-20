package configyaml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dovixman/harness-profiles/internal/domain"
)

func TestLoadMissingConfigReturnsEmptyRegistry(t *testing.T) {
	paths := Paths{ConfigHome: t.TempDir()}
	repo := NewRepository(paths)

	config, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.HarnessesRoot != paths.HarnessesRoot() {
		t.Fatalf("HarnessesRoot = %q, want %q", config.HarnessesRoot, paths.HarnessesRoot())
	}
	if len(config.Harnesses) != 0 {
		t.Fatalf("Harnesses length = %d, want 0", len(config.Harnesses))
	}
}

func TestPathsReturnsConfigAndHarnessRoots(t *testing.T) {
	paths := Paths{ConfigHome: t.TempDir()}
	got, err := paths.Paths()
	if err != nil {
		t.Fatalf("Paths() = %v", err)
	}
	if got.ConfigPath != paths.ConfigPath() || got.HarnessesRoot != paths.HarnessesRoot() {
		t.Fatalf("Paths() = %+v, want config and harness roots", got)
	}
}

func TestDefaultPathsUseHomeDotConfig(t *testing.T) {
	t.Setenv(configHomeEnv, "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	paths, err := DefaultPaths()
	if err != nil {
		t.Fatal(err)
	}

	wantHome := filepath.Join(home, ".config", "harness-profiles")
	if paths.ConfigHome != wantHome || paths.ConfigPath() != filepath.Join(wantHome, "config.json") || paths.HarnessesRoot() != filepath.Join(wantHome, "harnesses") {
		t.Fatalf("paths = %+v config = %q harnesses = %q, want home .config paths", paths, paths.ConfigPath(), paths.HarnessesRoot())
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	paths := Paths{ConfigHome: t.TempDir()}
	repo := NewRepository(paths)
	want := domain.Config{
		HarnessesRoot: filepath.Join(paths.ConfigHome, "harnesses"),
		Harnesses: []domain.Harness{{
			ID:          "claude",
			Label:       "Claude Code",
			LinkPath:    filepath.Join(paths.ConfigHome, "claude", "agents"),
			RestartHint: "restart app",
		}},
	}

	if err := repo.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.HarnessesRoot != want.HarnessesRoot {
		t.Fatalf("HarnessesRoot = %q, want %q", got.HarnessesRoot, want.HarnessesRoot)
	}
	if len(got.Harnesses) != 1 || !reflect.DeepEqual(got.Harnesses[0], want.Harnesses[0]) {
		t.Fatalf("Harnesses = %#v, want %#v", got.Harnesses, want.Harnesses)
	}
}

func TestLoadRespectsLinksOverLegacyLinkPath(t *testing.T) {
	paths := Paths{ConfigHome: t.TempDir()}
	repo := NewRepository(paths)
	legacyPath := filepath.Join(paths.ConfigHome, "legacy", "root")

	raw, err := json.Marshal(map[string]any{
		"harnesses_root": filepath.Join(paths.ConfigHome, "harnesses"),
		"harnesses": []map[string]any{{
			"id":        "claude",
			"label":     "Claude",
			"link_path": legacyPath,
			"links": []map[string]any{{
				"id":   "root",
				"path": filepath.Join(paths.ConfigHome, "agents"),
				"kind": "dir",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if err := os.WriteFile(paths.ConfigPath(), raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded.Harnesses) != 1 {
		t.Fatalf("loaded harness count = %d, want 1", len(loaded.Harnesses))
	}
	if got := loaded.Harnesses[0].LinkPath; got != legacyPath {
		t.Fatalf("LinkPath = %q, want %q", got, legacyPath)
	}
	if len(loaded.Harnesses[0].Links) != 1 {
		t.Fatalf("len(Links) = %d, want 1", len(loaded.Harnesses[0].Links))
	}
	if got := loaded.Harnesses[0].Links[0].ID; got != "root" {
		t.Fatalf("Links[0].ID = %q, want %q", got, "root")
	}
}

func TestSaveWritesLinksField(t *testing.T) {
	paths := Paths{ConfigHome: t.TempDir()}
	repo := NewRepository(paths)
	want := domain.Config{
		HarnessesRoot: filepath.Join(paths.ConfigHome, "harnesses"),
		Harnesses: []domain.Harness{{
			ID:    "claude",
			Label: "Claude",
			Links: []domain.HarnessLink{{
				ID:   "state",
				Path: filepath.Join(paths.ConfigHome, "state"),
				Kind: domain.HarnessLinkKindFile,
			}},
			RestartHint: "restart app",
		}},
	}

	if err := repo.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(paths.ConfigPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), `"links"`) {
		t.Fatal("saved config does not contain links field")
	}
	if strings.Contains(string(raw), `"link_path"`) {
		t.Fatal("saved config should not include link_path when links are present")
	}

	loaded, err := repo.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Harnesses) != 1 || loaded.Harnesses[0].Links[0].Kind != domain.HarnessLinkKindFile {
		t.Fatalf("Loaded harness = %#v", loaded.Harnesses[0])
	}
}

func TestSaveRejectsInvalidConfig(t *testing.T) {
	paths := Paths{ConfigHome: t.TempDir()}
	repo := NewRepository(paths)

	err := repo.Save(domain.Config{
		HarnessesRoot: paths.HarnessesRoot(),
		Harnesses: []domain.Harness{{
			ID:       "bad id",
			Label:    "Bad",
			LinkPath: filepath.Join(paths.ConfigHome, "bad"),
		}},
	})
	if err == nil {
		t.Fatal("Save() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "whitespace") {
		t.Fatalf("Save() error = %q, want whitespace validation", err)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	paths := Paths{ConfigHome: t.TempDir()}
	repo := NewRepository(paths)
	if err := os.MkdirAll(paths.ConfigHome, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data := []byte(`{"harnesses_root":"` + paths.HarnessesRoot() + `","harnesses":[{"id":"claude","label":"Claude Code"}]}`)
	if err := os.WriteFile(paths.ConfigPath(), data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := repo.Load(); err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}
