package tui

import (
	"fmt"
	"strings"
)

const itemKindNoop = "noop"

type menuViewSection struct {
	Title string
	Items []menuViewItem
}

type menuViewItem struct {
	Icon        string
	Label       string
	Description string
	Kind        string
}

func (m Model) renderMenuSections(b *strings.Builder, sections []menuViewSection, selected int, marker string) {
	itemIndex := 0
	for _, section := range sections {
		b.WriteString(helpStyle.Render(section.Title) + "\n")
		for _, item := range section.Items {
			line := fmt.Sprintf("%s %s", item.Icon, item.Label)
			selectable := item.Kind != itemKindNoop
			if item.Description != "" {
				description := fitMiddle(item.Description, m.descriptionWidth())
				if selectable && itemIndex == selected {
					line += "  " + description
				} else {
					line += "  " + helpStyle.Render(description)
				}
			}
			if selectable && itemIndex == selected {
				b.WriteString(selectStyle.Render(marker+" "+line) + "\n")
			} else {
				b.WriteString(rowStyle.Render("  "+line) + "\n")
			}
			if selectable {
				itemIndex++
			}
		}
		b.WriteString("\n")
	}
}
