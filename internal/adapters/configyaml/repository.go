package configyaml

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/dovixman/harness-profiles/internal/domain"
)

type Repository struct {
	paths Paths
}

type fileConfig struct {
	HarnessesRoot string        `json:"harnesses_root"`
	Harnesses     []fileHarness `json:"harnesses"`
}

type fileHarness struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	LinkPath    string            `json:"link_path,omitempty"`
	Links       []fileHarnessLink `json:"links,omitempty"`
	RestartHint string            `json:"restart_hint,omitempty"`
}

type fileHarnessLink struct {
	ID   string                 `json:"id"`
	Path string                 `json:"path"`
	Kind domain.HarnessLinkKind `json:"kind"`
}

func NewRepository(paths Paths) Repository {
	return Repository{paths: paths}
}

func (r Repository) Load() (domain.Config, error) {
	data, err := os.ReadFile(r.paths.ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.Config{HarnessesRoot: r.paths.HarnessesRoot()}, nil
		}
		return domain.Config{}, err
	}

	var file fileConfig
	if err := json.Unmarshal(data, &file); err != nil {
		return domain.Config{}, err
	}

	config := fromFile(file, r.paths.HarnessesRoot())
	if err := config.Validate(); err != nil {
		return domain.Config{}, err
	}

	return config, nil
}

func (r Repository) Save(config domain.Config) error {
	if config.HarnessesRoot == "" {
		config.HarnessesRoot = r.paths.HarnessesRoot()
	}
	if err := config.Validate(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(toFile(config), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(r.paths.ConfigPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.paths.ConfigPath(), data, 0o644)
}

func fromFile(file fileConfig, defaultHarnessesRoot string) domain.Config {
	config := domain.Config{HarnessesRoot: file.HarnessesRoot}
	if config.HarnessesRoot == "" {
		config.HarnessesRoot = defaultHarnessesRoot
	}

	for _, harness := range file.Harnesses {
		var links []domain.HarnessLink
		if len(harness.Links) > 0 {
			links = make([]domain.HarnessLink, len(harness.Links))
			for i, link := range harness.Links {
				links[i] = domain.HarnessLink{ID: link.ID, Path: link.Path, Kind: link.Kind}
			}
		}
		config.Harnesses = append(config.Harnesses, domain.Harness{
			ID:          harness.ID,
			Label:       harness.Label,
			LinkPath:    harness.LinkPath,
			Links:       links,
			RestartHint: harness.RestartHint,
		})
	}

	return config
}

func toFile(config domain.Config) fileConfig {
	file := fileConfig{HarnessesRoot: config.HarnessesRoot}
	for _, harness := range config.Harnesses {
		var links []fileHarnessLink
		if len(harness.Links) > 0 {
			links = make([]fileHarnessLink, len(harness.Links))
			for i, link := range harness.Links {
				links[i] = fileHarnessLink{ID: link.ID, Path: link.Path, Kind: link.Kind}
			}
		}
		linkPath := ""
		if len(harness.Links) == 0 {
			linkPath = harness.LinkPath
		}
		file.Harnesses = append(file.Harnesses, fileHarness{
			ID:          harness.ID,
			Label:       harness.Label,
			LinkPath:    linkPath,
			Links:       links,
			RestartHint: harness.RestartHint,
		})
	}
	return file
}
