package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	HarnessesRoot string
	Harnesses     []Harness
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HarnessesRoot) == "" {
		return fmt.Errorf("harnesses root is required")
	}

	seenIDs := map[string]struct{}{}
	seenLabels := map[string]struct{}{}
	seenLinks := map[string]struct{}{}
	for _, harness := range c.Harnesses {
		if err := harness.Validate(); err != nil {
			return err
		}
		idKey := strings.ToLower(harness.ID)
		if _, ok := seenIDs[idKey]; ok {
			return fmt.Errorf("duplicate harness id %q", harness.ID)
		}
		seenIDs[idKey] = struct{}{}
		labelKey := strings.ToLower(strings.TrimSpace(harness.Label))
		if _, ok := seenLabels[labelKey]; ok {
			return fmt.Errorf("duplicate harness label %q", harness.Label)
		}
		seenLabels[labelKey] = struct{}{}
		for _, link := range harness.LinksOrLegacy() {
			linkKey := canonicalLinkPath(link.Path)
			if _, ok := seenLinks[linkKey]; ok {
				return fmt.Errorf("duplicate harness root path %q", link.Path)
			}
			seenLinks[linkKey] = struct{}{}
		}
	}

	return nil
}

func canonicalLinkPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if path == "~" {
				path = home
			} else {
				path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
			}
		}
	}
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	return filepath.Clean(path)
}
