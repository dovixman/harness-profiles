package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

func fitMiddle(value string, maxWidth int) string {
	if maxWidth <= 0 || lipgloss.Width(value) <= maxWidth {
		return value
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	runes := []rune(value)
	keep := maxWidth - 3
	front := keep / 2
	back := keep - front
	if front+back >= len(runes) {
		return value
	}
	return string(runes[:front]) + "..." + string(runes[len(runes)-back:])
}

func fitTail(value string, maxWidth int) string {
	if maxWidth <= 0 || lipgloss.Width(value) <= maxWidth {
		return value
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	runes := []rune(value)
	if len(runes) <= maxWidth {
		return value
	}
	return string(runes[:maxWidth-3]) + "..."
}

func wrapFieldValue(value string, width int) string {
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(value)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= 1 {
		return wrapped
	}
	indent := strings.Repeat(" ", 25)
	return strings.Join(lines, "\n"+indent)
}

func clamp(value, length int) int {
	if length <= 0 {
		return 0
	}
	return min(max(value, 0), length-1)
}

func wrap(value, length int) int {
	if length <= 0 {
		return 0
	}
	if value < 0 {
		return length - 1
	}
	if value >= length {
		return 0
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func lastRune(value string) string {
	_, size := utf8.DecodeLastRuneInString(value)
	if size > 0 {
		return value[len(value)-size:]
	}
	return ""
}
