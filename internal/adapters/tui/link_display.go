package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

func (m Model) hasExplicitLinks() bool {
	return len(m.harness.Links) > 0
}

func (m Model) managedLinks() []domain.HarnessLink {
	return m.harness.LinksOrLegacy()
}

func harnessLinkSummary(h domain.Harness, width int) string {
	links := h.LinksOrLegacy()
	if len(links) == 0 {
		return ""
	}
	parts := make([]string, 0, len(links))
	for _, link := range links {
		parts = append(parts, fmt.Sprintf("%s (%s) %s", link.ID, link.Kind, fitMiddle(link.Path, width)))
	}
	return strings.Join(parts, " · ")
}

func (m Model) managedLinkSummary(width int) string {
	links := m.managedLinks()
	if len(links) == 0 {
		return ""
	}
	parts := make([]string, 0, len(links))
	for _, link := range links {
		parts = append(parts, fmt.Sprintf("%s (%s) %s", link.ID, link.Kind, fitMiddle(link.Path, width)))
	}
	return strings.Join(parts, " · ")
}

func (m Model) managedLinkLines() []string {
	links := m.managedLinks()
	lines := make([]string, 0, len(links))
	for _, link := range links {
		lines = append(lines, fmt.Sprintf("%s (%s): %s", link.ID, link.Kind, link.Path))
	}
	return lines
}

func (m Model) managedArtifactLines(profile string) []string {
	links := m.managedLinks()
	lines := make([]string, 0, len(links))
	for _, link := range links {
		lines = append(lines, fmt.Sprintf("%s (%s): %s", link.ID, link.Kind, m.profileArtifactPath(link, profile)))
	}
	return lines
}

func (m Model) profileArtifactPath(link domain.HarnessLink, profile string) string {
	return filepath.Join(m.paths.HarnessesRoot, m.harness.ID, profile, link.ID)
}

func (m Model) managedLinkItems() []detailItem {
	links := m.managedLinks()
	if len(links) == 0 {
		return nil
	}
	statuses := make(map[string]app.ProfileLinkStatus, len(m.current.Links))
	for _, status := range m.current.Links {
		statuses[strings.ToLower(strings.TrimSpace(status.ID))] = status
	}
	items := make([]detailItem, 0, len(links))
	for index, link := range links {
		item := detailItem{
			Icon:        "○",
			Label:       fmt.Sprintf("%s (%s)", link.ID, link.Kind),
			Description: fitMiddle(link.Path, m.descriptionWidth()),
			Kind:        itemKindLink,
			Link:        index,
		}
		if status, ok := statuses[strings.ToLower(strings.TrimSpace(link.ID))]; ok {
			item.Icon = profileLinkStateIcon(status.State)
			item.Description = profileLinkDescription(status, m.descriptionWidth())
		}
		items = append(items, item)
	}
	return items
}

func profileLinkStateIcon(state app.ProfileLinkState) string {
	switch state {
	case app.ProfileStateActive:
		return "●"
	case app.ProfileStateExternal:
		return "↗"
	case app.ProfileStateMissing:
		return "⚠"
	case app.ProfileStateMixed:
		return "◐"
	case app.ProfileStateUnknown:
		return "?"
	default:
		return "○"
	}
}

func profileLinkDescription(status app.ProfileLinkStatus, width int) string {
	switch status.State {
	case app.ProfileStateActive:
		if status.ArtifactPath != "" {
			return "active · " + fitMiddle(status.ArtifactPath, width)
		}
		return "active"
	case app.ProfileStateExternal:
		if status.Target != "" {
			return "external · " + fitMiddle(status.Target, width)
		}
		return "external"
	case app.ProfileStateMissing:
		if status.Profile != "" {
			return "missing · profile " + status.Profile
		}
		return "missing"
	case app.ProfileStateMixed:
		return "mixed"
	case app.ProfileStateUnknown:
		return "unknown"
	default:
		if status.Target != "" {
			return fitMiddle(status.Target, width)
		}
		return fitMiddle(status.LinkPath, width)
	}
}

func (m Model) currentStateLine(width int) string {
	current := m.current
	switch current.State {
	case app.ProfileStateActive:
		if current.Name != "" {
			return okStyle.Render("● " + current.Name)
		}
		return okStyle.Render("● active")
	case app.ProfileStateExternal:
		if current.Path != "" {
			return warnStyle.Render("external") + " " + fitMiddle(current.Path, width)
		}
		return warnStyle.Render("external")
	case app.ProfileStateMixed:
		return warnStyle.Render("mixed")
	case app.ProfileStateMissing:
		return warnStyle.Render("missing")
	case app.ProfileStateUnknown:
		return warnStyle.Render("unknown")
	}
	if current.External {
		if current.Path != "" {
			return warnStyle.Render("external") + " " + fitMiddle(current.Path, width)
		}
		return warnStyle.Render("external")
	}
	if current.Name != "" {
		return okStyle.Render("● " + current.Name)
	}
	return helpStyle.Render("No active managed profile")
}
