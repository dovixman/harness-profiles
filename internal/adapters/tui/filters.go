package tui

import "strings"

func (m Model) filteredHarnessIndexes() []int {
	var idxs []int
	q := strings.ToLower(strings.TrimSpace(m.hQuery))
	if m.hSearching {
		q = strings.ToLower(strings.TrimSpace(m.hDraft))
	}
	for i, h := range m.harnesses {
		searchable := strings.ToLower(h.ID + " " + h.Label)
		for _, link := range h.LinksOrLegacy() {
			searchable += " " + strings.ToLower(strings.TrimSpace(link.ID)+" "+string(link.Kind))
		}
		if q == "" || strings.Contains(searchable, q) {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func (m Model) filteredProfileIndexes() []int {
	var idxs []int
	q := strings.ToLower(strings.TrimSpace(m.pQuery))
	if m.pSearching {
		q = strings.ToLower(strings.TrimSpace(m.pDraft))
	}
	for i, p := range m.profiles {
		if q == "" || strings.Contains(strings.ToLower(p.Name+" "+p.Path), q) {
			idxs = append(idxs, i)
		}
	}
	return idxs
}
