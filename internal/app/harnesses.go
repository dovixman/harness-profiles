package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dovixman/harness-profiles/internal/domain"
)

type linkRollbackState struct {
	path       string
	previous   LinkInfo
	removePath bool
}

type restoreRollbackState struct {
	path     string
	previous LinkInfo
}

type adoptRollbackState struct {
	source string
	target string
}

func (s Service) ListHarnesses() ([]domain.Harness, error) {
	config, err := s.Repo.Load()
	if err != nil {
		return nil, err
	}
	return config.Harnesses, nil
}

func (s Service) AddHarness(opts AddHarnessOptions) (domain.Harness, error) {
	if err := domain.ValidateHarnessID(opts.ID); err != nil {
		return domain.Harness{}, err
	}
	label := strings.TrimSpace(opts.Label)
	if label == "" {
		label = opts.ID
	}
	links, err := harnessLinksForAdd(opts)
	if err != nil {
		return domain.Harness{}, err
	}
	harness := domain.Harness{ID: opts.ID, Label: label, Links: links, RestartHint: opts.RestartHint}
	if err := harness.Validate(); err != nil {
		return domain.Harness{}, err
	}

	config, err := s.Repo.Load()
	if err != nil {
		return domain.Harness{}, err
	}
	if _, ok := findHarness(config, opts.ID); ok {
		return domain.Harness{}, fmt.Errorf("harness %q already exists", opts.ID)
	}
	if err := validateUniqueHarness(config, harness, ""); err != nil {
		return domain.Harness{}, err
	}
	root := harnessRoot(config, opts.ID)
	if err := s.FS.MkdirAll(root, 0o755); err != nil {
		return domain.Harness{}, err
	}

	if err := s.addHarnessRoot(config, harness, opts); err != nil {
		return domain.Harness{}, err
	}
	config.Harnesses = append(config.Harnesses, harness)
	if err := s.Repo.Save(config); err != nil {
		return domain.Harness{}, err
	}
	return harness, nil
}

func (s Service) addHarnessRoot(config domain.Config, harness domain.Harness, opts AddHarnessOptions) error {
	return s.addHarnessLinks(config, harness, opts)
}

func (s Service) addHarnessLinks(config domain.Config, harness domain.Harness, opts AddHarnessOptions) error {
	var rollbacks []func() error
	for _, link := range harness.Links {
		undo, err := s.addHarnessLink(config, harness.ID, opts.InitialProfile, link, opts)
		if err != nil {
			for i := len(rollbacks) - 1; i >= 0; i-- {
				_ = rollbacks[i]()
			}
			return err
		}
		if undo != nil {
			rollbacks = append(rollbacks, undo)
		}
	}
	return nil
}

func (s Service) addHarnessLink(config domain.Config, harnessID, profile string, link domain.HarnessLink, opts AddHarnessOptions) (func() error, error) {
	root := harnessRoot(config, harnessID)
	artifact := artifactPath(config, harnessID, profile, link.ID)
	info, err := s.FS.InspectLink(link.Path)
	if err != nil {
		return nil, err
	}
	if !info.Exists {
		if err := domain.ValidateProfileName(profile); err != nil {
			return nil, fmt.Errorf("initial profile required for missing config root: %w", err)
		}
		if err := s.createManagedArtifact(link, artifact); err != nil {
			return nil, err
		}
		if err := s.FS.ReplaceSymlink(link.Path, artifact); err != nil {
			_ = s.FS.DeletePath(artifact)
			return nil, err
		}
		return func() error {
			_ = s.FS.DeletePath(link.Path)
			return s.FS.DeletePath(artifact)
		}, nil
	}
	if info.IsSymlink {
		absTarget, err := absLinkTarget(link.Path, info.Target)
		if err != nil {
			return nil, err
		}
		if isInside(root, absTarget) {
			return nil, nil
		}
		action := linkActionFor(opts, link.ID)
		switch action {
		case HarnessLinkActionRegister:
			return nil, nil
		case HarnessLinkActionCreate:
			if opts.RegisterSymlink {
				return nil, nil
			}
			if opts.ImportSymlink {
			} else {
				return nil, fmt.Errorf("%s is an external symlink; use import or register mode", link.Path)
			}
		case HarnessLinkActionImport:
			// handled below
		}
		if err := domain.ValidateProfileName(profile); err != nil {
			return nil, fmt.Errorf("initial profile required for imported symlink: %w", err)
		}
		targetInfo, err := s.FS.Stat(absTarget)
		if err != nil {
			return nil, err
		}
		targetKind, err := artifactKindFromFileInfo(absTarget, targetInfo)
		if err != nil {
			return nil, err
		}
		if targetKind != link.Kind {
			return nil, fmt.Errorf("%s is a %s but expected a %s", absTarget, targetKind, link.Kind)
		}
		if err := s.FS.CopyArtifact(absTarget, artifact, false); err != nil {
			return nil, err
		}
		if err := s.FS.ReplaceSymlink(link.Path, artifact); err != nil {
			_ = s.FS.DeletePath(artifact)
			return nil, err
		}
		return func() error {
			_ = s.FS.DeletePath(link.Path)
			_ = s.FS.DeletePath(artifact)
			return s.FS.ReplaceSymlink(link.Path, absTarget)
		}, nil
	}
	if err := domain.ValidateProfileName(profile); err != nil {
		return nil, fmt.Errorf("initial profile required for existing root: %w", err)
	}
	undo, err := s.copyConfiguredLink(link.Path, artifact, link.Kind)
	if err != nil {
		return nil, err
	}
	return undo, nil
}

func (s Service) createManagedArtifact(link domain.HarnessLink, artifact string) error {
	if err := s.FS.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		return err
	}
	switch link.Kind {
	case domain.HarnessLinkKindDir:
		return s.FS.MkdirAll(artifact, 0o755)
	case domain.HarnessLinkKindFile:
		return s.FS.WriteFile(artifact, nil)
	default:
		return fmt.Errorf("link kind %q is invalid", link.Kind)
	}
}

func (s Service) copyConfiguredLink(src, dst string, kind domain.HarnessLinkKind) (func() error, error) {
	sourceInfo, err := s.FS.Stat(src)
	if err != nil {
		return nil, err
	}
	sourceKind, err := artifactKindFromFileInfo(src, sourceInfo)
	if err != nil {
		return nil, err
	}
	if sourceKind != kind {
		return nil, fmt.Errorf("%s is a %s but expected a %s", src, sourceKind, kind)
	}
	if err := s.FS.CopyArtifact(src, dst, false); err != nil {
		return nil, err
	}
	if err := s.FS.DeletePath(src); err != nil {
		_ = s.FS.DeletePath(dst)
		return nil, err
	}
	if err := s.FS.ReplaceSymlink(src, dst); err != nil {
		if rollbackErr := s.FS.MoveArtifact(dst, src); rollbackErr != nil {
			return nil, fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}
	return func() error {
		if err := s.FS.DeletePath(src); err != nil {
			return err
		}
		return s.FS.MoveArtifact(dst, src)
	}, nil
}

func linkActionFor(opts AddHarnessOptions, linkID string) HarnessLinkAction {
	if opts.LinkActions != nil {
		if action, ok := opts.LinkActions[strings.ToLower(strings.TrimSpace(linkID))]; ok {
			return action
		}
	}
	if opts.ImportSymlink {
		return HarnessLinkActionImport
	}
	if opts.RegisterSymlink {
		return HarnessLinkActionRegister
	}
	return HarnessLinkActionCreate
}

func harnessLinksForAdd(opts AddHarnessOptions) ([]domain.HarnessLink, error) {
	if len(opts.Links) > 0 {
		return normalizeHarnessLinks(opts.Links)
	}
	linkPath, err := NormalizeConfigRootPath(opts.LinkPath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(linkPath) == "" {
		return nil, fmt.Errorf("link path is required")
	}
	return normalizeHarnessLinks([]domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: linkPath, Kind: domain.HarnessLinkKindDir}})
}

func harnessLinksForUpdate(harness domain.Harness, opts UpdateHarnessOptions) ([]domain.HarnessLink, bool, error) {
	if len(opts.Links) > 0 {
		links, err := normalizeHarnessLinks(opts.Links)
		return links, true, err
	}
	if linkPath := strings.TrimSpace(opts.LinkPath); linkPath != "" {
		normalized, err := NormalizeConfigRootPath(linkPath)
		if err != nil {
			return nil, false, err
		}
		links, err := normalizeHarnessLinks([]domain.HarnessLink{{ID: domain.LegacyDefaultLinkID, Path: normalized, Kind: domain.HarnessLinkKindDir}})
		return links, true, err
	}
	if len(harness.Links) == 0 {
		links, err := normalizeHarnessLinks(harness.LinksOrLegacy())
		return links, true, err
	}
	return nil, false, nil
}

func normalizeHarnessLinks(links []domain.HarnessLink) ([]domain.HarnessLink, error) {
	if len(links) == 0 {
		return nil, nil
	}
	normalized := make([]domain.HarnessLink, len(links))
	for i, link := range links {
		path, err := NormalizeConfigRootPath(link.Path)
		if err != nil {
			return nil, fmt.Errorf("link %q: %w", link.ID, err)
		}
		normalized[i] = domain.HarnessLink{ID: strings.TrimSpace(link.ID), Path: path, Kind: link.Kind}
	}
	return normalized, nil
}

func normalizedLinkMap(links []domain.HarnessLink) map[string]domain.HarnessLink {
	result := make(map[string]domain.HarnessLink, len(links))
	for _, link := range links {
		result[strings.ToLower(strings.TrimSpace(link.ID))] = link
	}
	return result
}

func (s Service) UpdateHarness(opts UpdateHarnessOptions) (domain.Harness, error) {
	config, harness, idx, err := s.loadHarness(opts.ID)
	if err != nil {
		return domain.Harness{}, err
	}
	updated := harness
	if strings.TrimSpace(opts.Label) != "" {
		updated.Label = opts.Label
	}
	if opts.RestartHint != "" {
		updated.RestartHint = opts.RestartHint
	}
	links, changed, err := harnessLinksForUpdate(harness, opts)
	if err != nil {
		return domain.Harness{}, err
	}
	if changed {
		updated.Links = links
		updated.LinkPath = ""
	}
	if err := updated.Validate(); err != nil {
		return domain.Harness{}, err
	}
	if err := validateUniqueHarness(config, updated, harness.ID); err != nil {
		return domain.Harness{}, err
	}
	if changed {
		rollback, err := s.updateHarnessLinks(config, harness, updated.Links, opts)
		if err != nil {
			return domain.Harness{}, err
		}
		if err := updated.Validate(); err != nil {
			_ = rollback()
			return domain.Harness{}, err
		}
		config.Harnesses[idx] = updated
		if err := s.Repo.Save(config); err != nil {
			if rollbackErr := rollback(); rollbackErr != nil {
				return domain.Harness{}, fmt.Errorf("%w; filesystem rollback failed: %v", err, rollbackErr)
			}
			return domain.Harness{}, err
		}
		return updated, nil
	}
	config.Harnesses[idx] = updated
	if err := s.Repo.Save(config); err != nil {
		return domain.Harness{}, err
	}
	return updated, nil
}

func validateUniqueHarness(config domain.Config, candidate domain.Harness, currentID string) error {
	candidateLinks := candidate.LinksOrLegacy()
	for _, harness := range config.Harnesses {
		if currentID != "" && harness.ID == currentID {
			continue
		}
		if strings.EqualFold(harness.ID, candidate.ID) {
			return fmt.Errorf("harness id %q already exists", candidate.ID)
		}
		if strings.EqualFold(strings.TrimSpace(harness.Label), strings.TrimSpace(candidate.Label)) {
			return fmt.Errorf("harness label %q already exists", candidate.Label)
		}
		existingLinks := harness.LinksOrLegacy()
		for _, existing := range existingLinks {
			existingPath, err := NormalizeConfigRootPath(existing.Path)
			if err != nil {
				return err
			}
			for _, candidateLink := range candidateLinks {
				candidatePath, err := NormalizeConfigRootPath(candidateLink.Path)
				if err != nil {
					return err
				}
				if existingPath == candidatePath {
					return fmt.Errorf("harness root path %q already exists", candidateLink.Path)
				}
			}
		}
	}
	return nil
}

func (s Service) updateHarnessLinks(config domain.Config, harness domain.Harness, links []domain.HarnessLink, opts UpdateHarnessOptions) (func() error, error) {
	if len(links) == 0 {
		return noRollback, nil
	}
	oldLinks := normalizedLinkMap(harness.LinksOrLegacy())
	newLinks := normalizedLinkMap(links)
	if len(harness.Links) > 0 && !linksRequireProfileUpdate(oldLinks, newLinks) {
		if !opts.RemoveOld {
			return noRollback, nil
		}
		return s.removeObsoleteHarnessLinks(oldLinks, newLinks)
	}
	current, err := s.currentProfile(config, harness)
	if err != nil {
		return nil, err
	}
	if current.State != ProfileStateActive || current.Name == "" || current.External {
		return nil, fmt.Errorf("current profile is not managed")
	}
	var rollbacks []func() error
	for _, link := range links {
		old, exists := oldLinks[normalizeLinkID(link.ID)]
		if exists {
			undo, err := s.updateExistingHarnessLink(config, harness, current.Name, old, link)
			if err != nil {
				_ = rollbackAll(rollbacks)
				return nil, err
			}
			rollbacks = appendRollback(rollbacks, undo)
			continue
		}
		undo, err := s.addUpdatedHarnessLink(config, harness, current.Name, link, opts)
		if err != nil {
			_ = rollbackAll(rollbacks)
			return nil, err
		}
		rollbacks = appendRollback(rollbacks, undo)
	}
	if opts.RemoveOld {
		undo, err := s.removeObsoleteHarnessLinks(oldLinks, newLinks)
		if err != nil {
			_ = rollbackAll(rollbacks)
			return nil, err
		}
		rollbacks = appendRollback(rollbacks, undo)
	}
	return func() error { return rollbackAll(rollbacks) }, nil
}

func linksRequireProfileUpdate(oldLinks, newLinks map[string]domain.HarnessLink) bool {
	for id, updated := range newLinks {
		old, exists := oldLinks[id]
		if !exists || old.Path != updated.Path || old.Kind != updated.Kind || old.ID != updated.ID {
			return true
		}
	}
	return false
}

func (s Service) updateExistingHarnessLink(config domain.Config, harness domain.Harness, profile string, old, updated domain.HarnessLink) (func() error, error) {
	if old.Kind != updated.Kind {
		return nil, fmt.Errorf("link %q kind cannot change while profile artifacts exist", old.ID)
	}
	if err := s.ensureArtifactForLink(config, harness.ID, profile, updated, len(harness.Links) == 0); err != nil {
		return nil, err
	}
	if err := s.ensureSafeLinkReplacement(updated.Path); err != nil {
		return nil, err
	}
	return s.replaceUpdatedLink(updated.Path, artifactPath(config, harness.ID, profile, updated.ID))
}

func (s Service) addUpdatedHarnessLink(config domain.Config, harness domain.Harness, profile string, link domain.HarnessLink, opts UpdateHarnessOptions) (func() error, error) {
	before, err := s.missingProfileArtifacts(config, harness.ID, link)
	if err != nil {
		return nil, err
	}
	previous, err := s.FS.InspectLink(link.Path)
	if err != nil {
		return nil, err
	}
	addOpts := AddHarnessOptions{InitialProfile: profile, Links: []domain.HarnessLink{link}, LinkActions: opts.LinkActions}
	undoLink, err := s.addHarnessLink(config, harness.ID, profile, link, addOpts)
	if err != nil {
		return nil, err
	}
	if err := s.ensureArtifactForLink(config, harness.ID, profile, link, false); err != nil {
		if undoLink != nil {
			_ = undoLink()
		}
		_ = s.deleteArtifacts(before)
		return nil, err
	}
	if undoLink == nil && previous.IsSymlink {
		previousTarget, err := absLinkTarget(link.Path, previous.Target)
		if err != nil {
			_ = s.deleteArtifacts(before)
			return nil, err
		}
		target := artifactPath(config, harness.ID, profile, link.ID)
		if isInside(harnessRoot(config, harness.ID), previousTarget) && filepath.Clean(previousTarget) != filepath.Clean(target) {
			undoLink, err = s.replaceUpdatedLink(link.Path, target)
			if err != nil {
				_ = s.deleteArtifacts(before)
				return nil, err
			}
		}
	}
	undoArtifacts := func() error { return s.deleteArtifacts(before) }
	return func() error {
		if undoLink != nil {
			if err := undoLink(); err != nil {
				return err
			}
		}
		return undoArtifacts()
	}, nil
}

func (s Service) replaceUpdatedLink(path, target string) (func() error, error) {
	previous, err := s.FS.InspectLink(path)
	if err != nil {
		return nil, err
	}
	if err := s.FS.ReplaceSymlink(path, target); err != nil {
		return nil, err
	}
	return func() error { return s.rollbackUpdatedLink(linkRollbackState{path: path, previous: previous}) }, nil
}

func (s Service) removeObsoleteHarnessLinks(oldLinks, newLinks map[string]domain.HarnessLink) (func() error, error) {
	newPaths := make(map[string]struct{}, len(newLinks))
	for _, link := range newLinks {
		newPaths[link.Path] = struct{}{}
	}
	var removed []linkRollbackState
	for id, old := range oldLinks {
		updated, retained := newLinks[id]
		if retained && updated.Path == old.Path {
			continue
		}
		if _, reused := newPaths[old.Path]; reused {
			continue
		}
		info, err := s.FS.InspectLink(old.Path)
		if err != nil {
			_ = rollbackLinkStates(s, removed)
			return nil, err
		}
		if !info.Exists {
			continue
		}
		if err := s.FS.RemoveSymlink(old.Path); err != nil {
			_ = rollbackLinkStates(s, removed)
			return nil, err
		}
		removed = append(removed, linkRollbackState{path: old.Path, previous: info, removePath: true})
	}
	return func() error { return rollbackLinkStates(s, removed) }, nil
}

func normalizeLinkID(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func appendRollback(rollbacks []func() error, rollback func() error) []func() error {
	if rollback == nil {
		return rollbacks
	}
	return append(rollbacks, rollback)
}

func noRollback() error { return nil }

func rollbackAll(rollbacks []func() error) error {
	var first error
	for i := len(rollbacks) - 1; i >= 0; i-- {
		if err := rollbacks[i](); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func rollbackLinkStates(s Service, states []linkRollbackState) error {
	var first error
	for i := len(states) - 1; i >= 0; i-- {
		if err := s.rollbackUpdatedLink(states[i]); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s Service) missingProfileArtifacts(config domain.Config, harnessID string, link domain.HarnessLink) ([]string, error) {
	entries, err := s.FS.ReadDir(harnessRoot(config, harnessID))
	if err != nil {
		return nil, err
	}
	missing := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		artifact := artifactPath(config, harnessID, entry.Name(), link.ID)
		if _, err := s.FS.Lstat(artifact); errors.Is(err, os.ErrNotExist) {
			missing = append(missing, artifact)
		} else if err != nil {
			return nil, err
		}
	}
	return missing, nil
}

func (s Service) deleteArtifacts(paths []string) error {
	var first error
	for _, path := range paths {
		if err := s.FS.DeletePath(path); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s Service) rollbackUpdatedLink(change linkRollbackState) error {
	if change.removePath {
		if change.previous.Exists && change.previous.IsSymlink {
			return s.FS.ReplaceSymlink(change.path, change.previous.Target)
		}
		return nil
	}
	if err := s.FS.DeletePath(change.path); err != nil {
		return err
	}
	if change.previous.Exists && change.previous.IsSymlink {
		return s.FS.ReplaceSymlink(change.path, change.previous.Target)
	}
	return nil
}

func (s Service) ensureArtifactForLink(config domain.Config, harnessID, profile string, link domain.HarnessLink, legacyLayout bool) error {
	profiles, err := s.FS.ReadDir(harnessRoot(config, harnessID))
	if err != nil {
		artifact := artifactPath(config, harnessID, profile, link.ID)
		if _, statErr := s.FS.Lstat(artifact); statErr == nil {
			return s.ensureArtifactKind(artifact, link.Kind)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		if legacyLayout && link.ID == domain.LegacyDefaultLinkID && link.Kind == domain.HarnessLinkKindDir {
			return s.migrateLegacyProfileArtifact(config, harnessID, profile, artifact)
		}
		return s.createManagedArtifact(link, artifact)
	}
	createdCurrent := false
	for _, entry := range profiles {
		if !entry.IsDir() {
			continue
		}
		artifact := artifactPath(config, harnessID, entry.Name(), link.ID)
		if _, statErr := s.FS.Lstat(artifact); statErr == nil {
			if err := s.ensureArtifactKind(artifact, link.Kind); err != nil {
				return err
			}
			if legacyLayout && link.ID == domain.LegacyDefaultLinkID && link.Kind == domain.HarnessLinkKindDir {
				if err := s.assertLegacyMigrationSafe(config, harnessID, entry.Name(), artifact); err != nil {
					return err
				}
			}
			if entry.Name() == profile {
				createdCurrent = true
			}
			continue
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		if legacyLayout && link.ID == domain.LegacyDefaultLinkID && link.Kind == domain.HarnessLinkKindDir {
			if err := s.migrateLegacyProfileArtifact(config, harnessID, entry.Name(), artifact); err != nil {
				return err
			}
			if entry.Name() == profile {
				createdCurrent = true
			}
			continue
		}
		if err := s.createManagedArtifact(link, artifact); err != nil {
			return err
		}
		if entry.Name() == profile {
			createdCurrent = true
		}
	}
	if createdCurrent {
		return nil
	}
	artifact := artifactPath(config, harnessID, profile, link.ID)
	if _, statErr := s.FS.Lstat(artifact); statErr == nil {
		return s.ensureArtifactKind(artifact, link.Kind)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	if legacyLayout && link.ID == domain.LegacyDefaultLinkID && link.Kind == domain.HarnessLinkKindDir {
		return s.migrateLegacyProfileArtifact(config, harnessID, profile, artifact)
	}
	return s.createManagedArtifact(link, artifact)
}

func (s Service) ensureArtifactKind(path string, expected domain.HarnessLinkKind) error {
	info, err := s.FS.Stat(path)
	if err != nil {
		return err
	}
	actual, err := artifactKindFromFileInfo(path, info)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("%s is a %s but expected a %s", path, actual, expected)
	}
	return nil
}

func (s Service) assertLegacyMigrationSafe(config domain.Config, harnessID, profile, artifact string) error {
	legacyProfile := profilePath(config, harnessID, profile)
	entries, err := s.FS.ReadDir(legacyProfile)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.Name()) != domain.LegacyDefaultLinkID {
			return fmt.Errorf("legacy profile %q contains direct contents beside %q; repair required", legacyProfile, artifact)
		}
	}
	return nil
}

func (s Service) migrateLegacyProfileArtifact(config domain.Config, harnessID, profile, artifact string) error {
	legacyProfile := profilePath(config, harnessID, profile)
	if _, err := s.FS.Lstat(artifact); err == nil {
		entries, legacyErr := s.FS.ReadDir(legacyProfile)
		if legacyErr != nil {
			if errors.Is(legacyErr, os.ErrNotExist) {
				return nil
			}
			return legacyErr
		}
		for _, entry := range entries {
			if strings.TrimSpace(entry.Name()) != domain.LegacyDefaultLinkID {
				return fmt.Errorf("legacy profile %q contains direct contents beside %q; repair required", legacyProfile, artifact)
			}
		}
		return nil
	}
	if _, err := s.FS.Lstat(legacyProfile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s.FS.MkdirAll(artifact, 0o755)
		}
		return err
	}
	entries, err := s.FS.ReadDir(legacyProfile)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.Name()) == domain.LegacyDefaultLinkID {
			return fmt.Errorf("legacy profile %q already contains %q; repair required", legacyProfile, artifact)
		}
	}
	if err := s.FS.MkdirAll(artifact, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		if err := s.FS.MoveArtifact(filepath.Join(legacyProfile, entry.Name()), filepath.Join(artifact, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) ensureSafeLinkReplacement(path string) error {
	info, err := s.FS.InspectLink(path)
	if err != nil {
		return err
	}
	if info.Exists && !info.IsSymlink {
		return fmt.Errorf("%s exists and is not a symlink", path)
	}
	return nil
}

func (s Service) DeleteHarness(opts DeleteHarnessOptions) error {
	config, harness, idx, err := s.loadHarness(opts.ID)
	if err != nil {
		return err
	}
	if err := s.deleteHarnessRoot(config, harness, opts); err != nil {
		return err
	}
	config.Harnesses = append(config.Harnesses[:idx], config.Harnesses[idx+1:]...)
	return s.Repo.Save(config)
}

func (s Service) deleteHarnessRoot(config domain.Config, harness domain.Harness, opts DeleteHarnessOptions) error {
	mode, err := domain.ParseDeleteMode(opts.Mode)
	if err != nil {
		return err
	}
	links := harness.LinksOrLegacy()
	switch mode {
	case domain.DeleteModeKeepRoot:
	case domain.DeleteModeRestore:
		if err := domain.ValidateProfileName(opts.RestoreProfile); err != nil {
			return err
		}
		targets, err := s.profileArtifactsWithKinds(config, harness, opts.RestoreProfile)
		if err != nil {
			return err
		}
		var restored []restoreRollbackState
		for _, target := range targets {
			previous, err := s.FS.InspectLink(target.LinkPath)
			if err != nil {
				for i := len(restored) - 1; i >= 0; i-- {
					_ = s.restoreDeletedLink(restored[i])
				}
				return err
			}
			if err := s.FS.RemoveSymlink(target.LinkPath); err != nil {
				for i := len(restored) - 1; i >= 0; i-- {
					_ = s.restoreDeletedLink(restored[i])
				}
				return err
			}
			if err := s.FS.CopyArtifact(target.ArtifactPath, target.LinkPath, false); err != nil {
				_ = s.restoreDeletedLink(restoreRollbackState{path: target.LinkPath, previous: previous})
				for i := len(restored) - 1; i >= 0; i-- {
					_ = s.restoreDeletedLink(restored[i])
				}
				return err
			}
			restored = append(restored, restoreRollbackState{path: target.LinkPath, previous: previous})
		}
	case domain.DeleteModeDeleteAll:
		if opts.Confirm != harness.ID {
			return fmt.Errorf("delete-all requires --confirm %s", harness.ID)
		}
		deleted := make([]string, 0, len(links))
		for _, link := range links {
			if err := s.FS.DeletePath(link.Path); err != nil {
				if len(deleted) > 0 {
					return fmt.Errorf("delete-all partially deleted %s before failure at %s: %w", strings.Join(deleted, ", "), link.Path, err)
				}
				return err
			}
			deleted = append(deleted, link.Path)
		}
	}
	return s.FS.DeletePath(harnessRoot(config, harness.ID))
}

func (s Service) restoreDeletedLink(change restoreRollbackState) error {
	if err := s.FS.DeletePath(change.path); err != nil {
		return err
	}
	if change.previous.Exists && change.previous.IsSymlink {
		return s.FS.ReplaceSymlink(change.path, change.previous.Target)
	}
	return nil
}

func (s Service) InspectHarness(id string) (domain.Harness, error) {
	_, harness, _, err := s.loadHarness(id)
	return harness, err
}
