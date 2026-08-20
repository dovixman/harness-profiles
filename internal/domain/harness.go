package domain

import (
	"fmt"
	"strings"
)

type Harness struct {
	ID          string
	Label       string
	LinkPath    string
	Links       []HarnessLink
	RestartHint string
}

const LegacyDefaultLinkID = "root"

type HarnessLinkKind string

const (
	HarnessLinkKindDir  HarnessLinkKind = "dir"
	HarnessLinkKindFile HarnessLinkKind = "file"
)

type HarnessLink struct {
	ID   string
	Path string
	Kind HarnessLinkKind
}

func (h Harness) LinksOrLegacy() []HarnessLink {
	if len(h.Links) > 0 {
		links := make([]HarnessLink, len(h.Links))
		copy(links, h.Links)
		return links
	}
	if strings.TrimSpace(h.LinkPath) == "" {
		return nil
	}
	return []HarnessLink{{ID: LegacyDefaultLinkID, Path: h.LinkPath, Kind: HarnessLinkKindDir}}
}

func ValidateHarnessLinkID(id string) error {
	return validateName("link id", id)
}

func ValidateHarnessLinkKind(kind HarnessLinkKind) error {
	switch kind {
	case HarnessLinkKindDir, HarnessLinkKindFile:
		return nil
	default:
		return fmt.Errorf("link kind %q is invalid", kind)
	}
}

func (l HarnessLink) Validate() error {
	if err := ValidateHarnessLinkID(l.ID); err != nil {
		return err
	}
	if strings.TrimSpace(l.Path) == "" {
		return fmt.Errorf("link path is required")
	}
	if err := ValidateHarnessLinkKind(l.Kind); err != nil {
		return err
	}
	return nil
}

func (h Harness) Validate() error {
	if err := ValidateHarnessID(h.ID); err != nil {
		return err
	}
	if strings.TrimSpace(h.Label) == "" {
		return fmt.Errorf("harness %q label is required", h.ID)
	}
	if len(h.Links) > 0 {
		seenIDs := map[string]struct{}{}
		seenPaths := map[string]struct{}{}
		for _, link := range h.Links {
			if err := link.Validate(); err != nil {
				return fmt.Errorf("harness %q link %q: %w", h.ID, link.ID, err)
			}
			idKey := strings.ToLower(strings.TrimSpace(link.ID))
			if _, ok := seenIDs[idKey]; ok {
				return fmt.Errorf("duplicate harness link id %q", link.ID)
			}
			seenIDs[idKey] = struct{}{}
			pathKey := canonicalLinkPath(link.Path)
			if _, ok := seenPaths[pathKey]; ok {
				return fmt.Errorf("duplicate harness link path %q", link.Path)
			}
			seenPaths[pathKey] = struct{}{}
		}
		return nil
	}
	if strings.TrimSpace(h.LinkPath) == "" {
		return fmt.Errorf("harness %q link path is required", h.ID)
	}
	return nil

}

func ValidateHarnessID(id string) error {
	return validateName("harness id", id)
}
