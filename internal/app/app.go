package app

import (
	"os"

	"github.com/dovixman/harness-profiles/internal/domain"
)

type ConfigRepository interface {
	Load() (domain.Config, error)
	Save(domain.Config) error
}

type Paths struct {
	ConfigPath    string
	HarnessesRoot string
}

type PathsProvider interface {
	Paths() (Paths, error)
}

type LinkInfo struct {
	Exists    bool
	IsSymlink bool
	Target    string
}

type FileSystem interface {
	InspectLink(string) (LinkInfo, error)
	ReadDir(string) ([]os.DirEntry, error)
	Stat(string) (os.FileInfo, error)
	Lstat(string) (os.FileInfo, error)
	MkdirAll(string, os.FileMode) error
	Rename(oldPath, newPath string) error
	ReplaceSymlink(path, target string) error
	RemoveSymlink(string) error
	CopyDir(src, dst string, materializeSymlinks bool) error
	CopyArtifact(src, dst string, materializeSymlinks bool) error
	MoveDir(src, dst string) error
	MoveArtifact(src, dst string) error
	DeletePath(string) error
	WriteFile(path string, contents []byte) error
}

type Service struct {
	Repo  ConfigRepository
	Paths PathsProvider
	FS    FileSystem
}

type AddHarnessOptions struct {
	ID              string
	Label           string
	LinkPath        string
	Links           []domain.HarnessLink
	LinkActions     map[string]HarnessLinkAction
	RestartHint     string
	InitialProfile  string
	ImportSymlink   bool
	RegisterSymlink bool
}

type UpdateHarnessOptions struct {
	ID          string
	Label       string
	LinkPath    string
	Links       []domain.HarnessLink
	LinkActions map[string]HarnessLinkAction
	RestartHint string
	RemoveOld   bool
}

type HarnessLinkAction string

const (
	HarnessLinkActionCreate   HarnessLinkAction = "create"
	HarnessLinkActionImport   HarnessLinkAction = "import"
	HarnessLinkActionRegister HarnessLinkAction = "register"
)

type DeleteHarnessOptions struct {
	ID             string
	Mode           string
	RestoreProfile string
	Confirm        string
}

type ProfileLinkState string

const (
	ProfileStateActive   ProfileLinkState = "active"
	ProfileStateExternal ProfileLinkState = "external"
	ProfileStateMissing  ProfileLinkState = "missing"
	ProfileStateMixed    ProfileLinkState = "mixed"
	ProfileStateUnknown  ProfileLinkState = "unknown"
)

type ProfileLinkStatus struct {
	ID             string
	LinkPath       string
	Kind           domain.HarnessLinkKind
	Target         string
	Profile        string
	State          ProfileLinkState
	ArtifactPath   string
	NeedsMigration bool
}

type ProfileStatus struct {
	Name     string
	Path     string
	Active   bool
	External bool
	State    ProfileLinkState
	Links    []ProfileLinkStatus
}
