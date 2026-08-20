package tui

import (
	"fmt"
	"strings"

	"github.com/dovixman/harness-profiles/internal/domain"
)

func draftLinkPaths(draft addHarnessDraft) []string {
	if len(draft.Links) == 0 {
		if strings.TrimSpace(draft.ConfigPath) == "" {
			return nil
		}
		return []string{draft.ConfigPath}
	}
	paths := make([]string, 0, len(draft.Links))
	for _, link := range draft.Links {
		paths = append(paths, link.Path)
	}
	return paths
}

func draftLinkPreviewLines(links []domain.HarnessLink) []string {
	lines := make([]string, 0, len(links))
	for _, link := range links {
		lines = append(lines, fmt.Sprintf("%s (%s): %s", link.ID, link.Kind, link.Path))
	}
	return lines
}
