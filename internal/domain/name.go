package domain

import (
	"fmt"
	"strings"
	"unicode"
)

func validateName(label, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", label)
	}
	if strings.ContainsAny(value, `/\\`) {
		return fmt.Errorf("%s %q must not contain path separators", label, value)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%s %q is reserved", label, value)
	}
	for _, r := range value {
		if unicode.IsSpace(r) {
			return fmt.Errorf("%s %q must not contain whitespace", label, value)
		}
	}
	return nil
}
