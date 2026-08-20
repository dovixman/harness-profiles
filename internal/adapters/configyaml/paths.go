package configyaml

import (
	"os"
	"path/filepath"

	"github.com/dovixman/harness-profiles/internal/app"
)

const configHomeEnv = "HP_CONFIG_HOME"

type Paths struct {
	ConfigHome string
}

func DefaultPaths() (Paths, error) {
	if configHome := os.Getenv(configHomeEnv); configHome != "" {
		return Paths{ConfigHome: configHome}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	return Paths{ConfigHome: filepath.Join(home, ".config", "harness-profiles")}, nil
}

func (p Paths) ConfigPath() string {
	return filepath.Join(p.ConfigHome, "config.json")
}

func (p Paths) HarnessesRoot() string {
	return filepath.Join(p.ConfigHome, "harnesses")
}

func (p Paths) Paths() (app.Paths, error) {
	harnessesRoot := p.HarnessesRoot()
	config, err := NewRepository(p).Load()
	if err != nil {
		return app.Paths{}, err
	}
	if config.HarnessesRoot != "" {
		harnessesRoot = config.HarnessesRoot
	}

	return app.Paths{ConfigPath: p.ConfigPath(), HarnessesRoot: harnessesRoot}, nil
}
