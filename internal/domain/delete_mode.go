package domain

import "fmt"

type DeleteMode string

const (
	DeleteModeKeepRoot  DeleteMode = "keep-root"
	DeleteModeRestore   DeleteMode = "restore"
	DeleteModeDeleteAll DeleteMode = "delete-all"
)

func ParseDeleteMode(value string) (DeleteMode, error) {
	mode := DeleteMode(value)
	if err := mode.Validate(); err != nil {
		return "", err
	}
	return mode, nil
}

func (m DeleteMode) Validate() error {
	switch m {
	case DeleteModeKeepRoot, DeleteModeRestore, DeleteModeDeleteAll:
		return nil
	default:
		return fmt.Errorf("delete mode must be restore, keep-root, or delete-all")
	}
}
