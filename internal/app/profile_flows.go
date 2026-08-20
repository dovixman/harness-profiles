package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dovixman/harness-profiles/internal/domain"
)

func (s Service) profileArtifacts(config domain.Config, harness domain.Harness, profile string) ([]ProfileLinkStatus, error) {
	links := harness.LinksOrLegacy()
	legacySingleLink := len(harness.Links) == 0
	statuses := make([]ProfileLinkStatus, 0, len(links))
	for _, link := range links {
		artifact := ProfileLinkStatus{
			ID:       strings.TrimSpace(link.ID),
			Kind:     link.Kind,
			LinkPath: link.Path,
			Profile:  profile,
		}
		if legacySingleLink && link.ID == domain.LegacyDefaultLinkID && link.Kind == domain.HarnessLinkKindDir {
			artifactPath, needsMigration, err := s.profileArtifactPath(config, harness.ID, profile, link)
			if err != nil {
				return nil, err
			}
			artifact.ArtifactPath = artifactPath
			artifact.NeedsMigration = needsMigration
			statuses = append(statuses, artifact)
			continue
		}
		artifactPath, needsMigration, err := s.profileArtifactPath(config, harness.ID, profile, link)
		if err != nil {
			return nil, err
		}
		artifact.ArtifactPath = artifactPath
		artifact.NeedsMigration = needsMigration
		statuses = append(statuses, artifact)
	}
	return statuses, nil
}

func (s Service) profileArtifactsWithKinds(config domain.Config, harness domain.Harness, profile string) ([]ProfileLinkStatus, error) {
	artifacts, err := s.profileArtifacts(config, harness, profile)
	if err != nil {
		return nil, err
	}
	for i, artifact := range artifacts {
		info, err := s.FS.Stat(artifact.ArtifactPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("profile %q %q artifact is missing: %w", profile, artifact.ID, err)
			}
			return nil, fmt.Errorf("profile %q is missing %s artifact: %w", profile, artifact.ID, err)
		}
		kind, err := artifactKindFromFileInfo(artifact.ArtifactPath, info)
		if err != nil {
			return nil, fmt.Errorf("profile %q artifact %q is invalid: %w", profile, artifact.ArtifactPath, err)
		}
		if kind != artifact.Kind {
			return nil, fmt.Errorf("profile %q link %q is a %s artifact but expected %s", profile, artifact.ID, kind, artifact.Kind)
		}
		artifacts[i].State = ProfileStateActive
	}
	return artifacts, nil
}

func (s Service) ListProfiles(id string) ([]ProfileStatus, error) {
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return nil, err
	}
	current, _ := s.currentProfile(config, harness)
	entries, err := s.FS.ReadDir(harnessRoot(config, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var profiles []ProfileStatus
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			profiles = append(profiles, ProfileStatus{Name: name, Path: profilePath(config, id, name), Active: current.State == ProfileStateActive && current.Name == name})
		}
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return profiles, nil
}

func (s Service) CurrentProfile(id string) (ProfileStatus, error) {
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return ProfileStatus{}, err
	}
	return s.currentProfile(config, harness)
}

func (s Service) SwitchProfile(id, name string) (domain.Harness, error) {
	if err := domain.ValidateProfileName(name); err != nil {
		return domain.Harness{}, err
	}
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return domain.Harness{}, err
	}
	targets, err := s.profileArtifactsWithKinds(config, harness, name)
	if err != nil {
		return domain.Harness{}, err
	}
	preflight, err := s.preflightSwitchDestinations(targets)
	if err != nil {
		return domain.Harness{}, err
	}
	replaced := make([]switchRollback, 0, len(targets))
	for _, target := range targets {
		if err := s.FS.ReplaceSymlink(target.LinkPath, target.ArtifactPath); err != nil {
			if rollbackErr := s.rollbackSwitchTargets(replaced); rollbackErr != nil {
				return domain.Harness{}, fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
			}
			return domain.Harness{}, err
		}
		replaced = append(replaced, switchRollback{path: target.LinkPath, hadSymlink: preflight[target.LinkPath].Exists && preflight[target.LinkPath].IsSymlink, target: preflight[target.LinkPath].Target})
	}
	return harness, nil
}

type switchRollback struct {
	path       string
	hadSymlink bool
	target     string
}

func (s Service) preflightSwitchDestinations(targets []ProfileLinkStatus) (map[string]LinkInfo, error) {
	infoByPath := make(map[string]LinkInfo, len(targets))
	for _, target := range targets {
		info, err := s.FS.InspectLink(target.LinkPath)
		if err != nil {
			return nil, err
		}
		if info.Exists && !info.IsSymlink {
			return nil, fmt.Errorf("%s exists and is not a symlink", target.LinkPath)
		}
		infoByPath[target.LinkPath] = info
	}
	return infoByPath, nil
}

func (s Service) rollbackSwitchTargets(changed []switchRollback) error {
	var firstErr error
	for i := len(changed) - 1; i >= 0; i-- {
		r := changed[i]
		var err error
		if r.hadSymlink {
			err = s.FS.ReplaceSymlink(r.path, r.target)
		} else {
			err = s.FS.DeletePath(r.path)
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s Service) AdoptProfile(id, name string) error {
	if err := domain.ValidateProfileName(name); err != nil {
		return err
	}
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return err
	}
	if exists, err := s.profileNameExists(config, id, name, ""); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("profile %q already exists", name)
	}
	artifacts, err := s.profileArtifacts(config, harness, name)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		link, err := s.FS.InspectLink(artifact.LinkPath)
		if err != nil {
			return err
		}
		if !link.Exists || link.IsSymlink {
			return fmt.Errorf("%s is not a real %s", artifact.LinkPath, artifact.Kind)
		}
		sourceInfo, err := s.FS.Stat(artifact.LinkPath)
		if err != nil {
			return err
		}
		sourceKind, err := artifactKindFromFileInfo(artifact.LinkPath, sourceInfo)
		if err != nil {
			return fmt.Errorf("%s is neither a regular file nor directory", artifact.LinkPath)
		}
		if sourceKind != artifact.Kind {
			return fmt.Errorf("%s is a %s but expected a %s", artifact.LinkPath, sourceKind, artifact.Kind)
		}
	}
	var moved []adoptRollbackState
	for _, artifact := range artifacts {
		if err := s.FS.MoveArtifact(artifact.LinkPath, artifact.ArtifactPath); err != nil {
			for i := len(moved) - 1; i >= 0; i-- {
				_ = s.rollbackAdoptedLink(moved[i])
			}
			return err
		}
		if err := s.FS.ReplaceSymlink(artifact.LinkPath, artifact.ArtifactPath); err != nil {
			_ = s.rollbackAdoptedLink(adoptRollbackState{source: artifact.LinkPath, target: artifact.ArtifactPath})
			for i := len(moved) - 1; i >= 0; i-- {
				_ = s.rollbackAdoptedLink(moved[i])
			}
			return err
		}
		moved = append(moved, adoptRollbackState{source: artifact.LinkPath, target: artifact.ArtifactPath})
	}
	return nil
}

func (s Service) rollbackAdoptedLink(change adoptRollbackState) error {
	if err := s.FS.DeletePath(change.source); err != nil {
		return err
	}
	return s.FS.MoveArtifact(change.target, change.source)
}

func (s Service) CreateProfile(id, name string) error {
	if err := domain.ValidateProfileName(name); err != nil {
		return err
	}
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return err
	}
	if exists, err := s.profileNameExists(config, id, name, ""); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("profile %q already exists", name)
	}
	target := profilePath(config, id, name)
	if _, err := s.FS.Lstat(target); err == nil {
		return fmt.Errorf("profile %q already exists", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(harness.Links) == 0 {
		if err := s.FS.MkdirAll(target, 0o755); err != nil {
			return err
		}
		return s.FS.MkdirAll(filepath.Join(target, domain.LegacyDefaultLinkID), 0o755)
	}
	artifacts, err := s.profileArtifacts(config, harness, name)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if _, err := s.FS.Lstat(artifact.ArtifactPath); err == nil {
			return fmt.Errorf("profile %q already exists", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := s.FS.MkdirAll(target, 0o755); err != nil {
		return err
	}
	for _, artifact := range artifacts {
		switch artifact.Kind {
		case domain.HarnessLinkKindDir:
			if err := s.FS.MkdirAll(artifact.ArtifactPath, 0o755); err != nil {
				return err
			}
		case domain.HarnessLinkKindFile:
			if err := s.FS.WriteFile(artifact.ArtifactPath, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s Service) RenameProfile(id, oldName, newName string) error {
	if err := domain.ValidateProfileName(oldName); err != nil {
		return err
	}
	if err := domain.ValidateProfileName(newName); err != nil {
		return err
	}
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return err
	}
	if exists, err := s.profileNameExists(config, id, newName, oldName); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("profile %q already exists", newName)
	}
	oldPath := profilePath(config, id, oldName)
	newPath := profilePath(config, id, newName)
	if _, err := s.FS.Lstat(newPath); err == nil {
		return fmt.Errorf("profile %q already exists", newName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	info, err := s.FS.Stat(oldPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("profile %q is not a directory", oldName)
	}
	currentLinks, err := s.currentProfilePerLink(config, harness)
	if err != nil {
		return err
	}
	if err := s.FS.Rename(oldPath, newPath); err != nil {
		return err
	}
	for _, link := range currentLinks {
		if link.State != ProfileStateActive || !strings.EqualFold(link.Profile, oldName) {
			continue
		}
		linkInfo := domain.HarnessLink{ID: link.ID, Kind: link.Kind, Path: link.LinkPath}
		target, _, err := s.profileArtifactPath(config, harness.ID, newName, linkInfo)
		if err != nil {
			return err
		}
		if err := s.FS.ReplaceSymlink(link.LinkPath, target); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) CloneProfile(id, src, dst string, materialize bool) error {
	if err := domain.ValidateProfileName(src); err != nil {
		return err
	}
	if err := domain.ValidateProfileName(dst); err != nil {
		return err
	}
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return err
	}
	if exists, err := s.profileNameExists(config, id, dst, ""); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("profile %q already exists", dst)
	}
	source, err := s.profileArtifactsWithKinds(config, harness, src)
	if err != nil {
		return err
	}
	targets, err := s.profileArtifacts(config, harness, dst)
	if err != nil {
		return err
	}
	if len(source) != len(targets) {
		return fmt.Errorf("profile %q has unexpected artifact layout", src)
	}
	for i := range source {
		if err := s.FS.CopyArtifact(source[i].ArtifactPath, targets[i].ArtifactPath, materialize); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) DeleteProfile(id, name string, yes bool) error {
	if !yes {
		return fmt.Errorf("profile deletion requires --yes")
	}
	if err := domain.ValidateProfileName(name); err != nil {
		return err
	}
	config, harness, _, err := s.loadHarness(id)
	if err != nil {
		return err
	}
	current, err := s.currentProfile(config, harness)
	if err != nil {
		return err
	}
	if current.State == ProfileStateActive && strings.EqualFold(current.Name, name) {
		return fmt.Errorf("cannot delete active profile %q; switch first", name)
	}
	if current.State == ProfileStateActive {
		if strings.EqualFold(current.Name, name) {
			return fmt.Errorf("cannot delete active profile %q; switch first", name)
		}
	}
	if current.Name != "" && strings.EqualFold(current.Name, name) {
		return fmt.Errorf("cannot delete active profile %q; switch first", name)
	}
	if current.State != ProfileStateActive {
		return fmt.Errorf("cannot delete profile %q while current state is %q; repair or switch first", name, current.State)
	}
	return s.FS.DeletePath(profilePath(config, id, name))
}

func (s Service) ProfilePath(id, name string) (string, error) {
	if err := domain.ValidateProfileName(name); err != nil {
		return "", err
	}
	config, _, _, err := s.loadHarness(id)
	if err != nil {
		return "", err
	}
	return profilePath(config, id, name), nil
}
