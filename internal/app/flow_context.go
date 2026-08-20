package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dovixman/harness-profiles/internal/domain"
)

func NormalizeConfigRootPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = abs
	}
	return filepath.Clean(path), nil
}

func (s Service) Where() (Paths, error) {
	return s.Paths.Paths()
}

func (s Service) loadHarness(id string) (domain.Config, domain.Harness, int, error) {
	config, err := s.Repo.Load()
	if err != nil {
		return domain.Config{}, domain.Harness{}, 0, err
	}
	for i, harness := range config.Harnesses {
		if harness.ID == id {
			return config, harness, i, nil
		}
	}
	return domain.Config{}, domain.Harness{}, 0, fmt.Errorf("unknown harness %q", id)
}

func findHarness(config domain.Config, id string) (domain.Harness, bool) {
	for _, harness := range config.Harnesses {
		if harness.ID == id {
			return harness, true
		}
	}
	return domain.Harness{}, false
}

func harnessRoot(config domain.Config, id string) string {
	return filepath.Join(config.HarnessesRoot, id)
}

func profilePath(config domain.Config, id, name string) string {
	return filepath.Join(harnessRoot(config, id), name)
}

func artifactPath(config domain.Config, id, profile, linkID string) string {
	return filepath.Join(profilePath(config, id, profile), linkID)
}

func artifactKindFromFileInfo(path string, info os.FileInfo) (domain.HarnessLinkKind, error) {
	if info.IsDir() {
		return domain.HarnessLinkKindDir, nil
	}
	if info.Mode().IsRegular() {
		return domain.HarnessLinkKindFile, nil
	}
	return "", fmt.Errorf("%s is neither a regular file nor directory", path)
}

func (s Service) profileArtifactPath(config domain.Config, harnessID, profile string, link domain.HarnessLink) (string, bool, error) {
	path := artifactPath(config, harnessID, profile, link.ID)
	if link.ID != domain.LegacyDefaultLinkID {
		return path, false, nil
	}
	if _, err := s.FS.Stat(path); err == nil {
		return path, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	legacy := profilePath(config, harnessID, profile)
	if _, err := s.FS.Stat(legacy); err == nil {
		return legacy, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	return path, false, nil
}

func (s Service) currentProfileForLink(config domain.Config, harnessID string, link domain.HarnessLink, legacyHarness bool) (ProfileLinkStatus, error) {
	status := ProfileLinkStatus{ID: strings.TrimSpace(link.ID), Kind: link.Kind, LinkPath: link.Path}
	info, err := s.FS.InspectLink(link.Path)
	if err != nil {
		return status, err
	}
	if !info.Exists {
		status.State = ProfileStateMissing
		return status, nil
	}
	if !info.IsSymlink {
		status.State = ProfileStateUnknown
		status.ArtifactPath = link.Path
		return status, nil
	}

	target, err := absLinkTarget(link.Path, info.Target)
	if err != nil {
		return status, err
	}
	status.Target = target

	root := harnessRoot(config, harnessID)
	if !isInside(root, target) {
		status.State = ProfileStateExternal
		return status, nil
	}

	rel, err := filepath.Rel(root, target)
	if err != nil {
		status.State = ProfileStateUnknown
		return status, nil
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) == 0 || parts[0] == "" || parts[0] == "." {
		status.State = ProfileStateUnknown
		return status, nil
	}
	profile := parts[0]
	status.Profile = profile
	legacyProfilePath := filepath.Join(harnessRoot(config, harnessID), profile)

	artifactPath, needsMigration, err := s.profileArtifactPath(config, harnessID, profile, link)
	if err != nil {
		return status, err
	}
	status.ArtifactPath = artifactPath
	status.NeedsMigration = needsMigration

	if !isSamePath(target, artifactPath) {
		if !legacyHarness || link.ID != domain.LegacyDefaultLinkID || link.Kind != domain.HarnessLinkKindDir || !isSamePath(target, legacyProfilePath) {
			status.State = ProfileStateUnknown
			return status, nil
		}
		artifactPath = legacyProfilePath
		needsMigration = true
		status.ArtifactPath = artifactPath
		status.NeedsMigration = needsMigration
	}

	targetInfo, err := s.FS.Stat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			status.State = ProfileStateMissing
			return status, nil
		}
		return status, err
	}
	kind, err := artifactKindFromFileInfo(target, targetInfo)
	if err != nil {
		status.State = ProfileStateUnknown
		return status, nil
	}
	if kind != link.Kind {
		status.State = ProfileStateUnknown
		return status, nil
	}

	status.State = ProfileStateActive
	status.ArtifactPath = artifactPath
	return status, nil
}

func (s Service) aggregateProfileState(statuses []ProfileLinkStatus) ProfileStatus {
	result := ProfileStatus{Links: statuses}
	if len(statuses) == 0 {
		return result
	}

	activeProfile := ""
	missingProfile := ""
	hasExternal := false
	hasMissing := false
	hasUnknown := false
	hasMismatch := false

	for _, status := range statuses {
		switch status.State {
		case ProfileStateActive:
			if activeProfile == "" {
				activeProfile = status.Profile
				result.Path = status.ArtifactPath
			} else if !strings.EqualFold(activeProfile, status.Profile) {
				hasMismatch = true
			}
		case ProfileStateExternal:
			hasExternal = true
			if result.Path == "" {
				result.Path = status.Target
			}
		case ProfileStateMissing:
			hasMissing = true
			if missingProfile == "" {
				missingProfile = status.Profile
			} else if !strings.EqualFold(missingProfile, status.Profile) {
				hasMismatch = true
			}
		default:
			hasUnknown = true
		}
	}

	if hasMissing {
		result.State = ProfileStateMissing
		if hasMismatch || hasExternal {
			result.State = ProfileStateMixed
			return result
		}
		result.Name = missingProfile
		return result
	}
	if hasUnknown {
		result.State = ProfileStateUnknown
		return result
	}
	if hasExternal {
		result.State = ProfileStateExternal
		result.External = true
		return result
	}
	if hasMismatch {
		result.State = ProfileStateMixed
		return result
	}
	if activeProfile != "" {
		result.State = ProfileStateActive
		result.Name = activeProfile
		result.Active = true
	}
	return result
}

func isSamePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	return a == b
}

func (s Service) profileNameExists(config domain.Config, id, name, except string) (bool, error) {
	entries, err := s.FS.ReadDir(harnessRoot(config, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entryName := entry.Name()
		if strings.EqualFold(entryName, except) {
			continue
		}
		if strings.EqualFold(entryName, name) {
			return true, nil
		}
	}
	return false, nil
}

func (s Service) currentProfilePerLink(config domain.Config, harness domain.Harness) ([]ProfileLinkStatus, error) {
	links := harness.LinksOrLegacy()
	statuses := make([]ProfileLinkStatus, 0, len(links))
	legacyHarness := len(harness.Links) == 0
	for _, link := range links {
		status, err := s.currentProfileForLink(config, harness.ID, link, legacyHarness)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s Service) currentProfile(config domain.Config, harness domain.Harness) (ProfileStatus, error) {
	statuses, err := s.currentProfilePerLink(config, harness)
	if err != nil {
		return ProfileStatus{}, err
	}
	return s.aggregateProfileState(statuses), nil
}

func absLinkTarget(linkPath, target string) (string, error) {
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(linkPath), target)
	}
	return filepath.Abs(target)
}

func isInside(root, path string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
