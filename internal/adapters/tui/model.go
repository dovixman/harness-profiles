package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/dovixman/harness-profiles/internal/app"
	"github.com/dovixman/harness-profiles/internal/domain"
)

type Service interface {
	Where() (app.Paths, error)
	ListHarnesses() ([]domain.Harness, error)
	InspectHarness(string) (domain.Harness, error)
	ListProfiles(string) ([]app.ProfileStatus, error)
	CurrentProfile(string) (app.ProfileStatus, error)
	AddHarness(app.AddHarnessOptions) (domain.Harness, error)
	UpdateHarness(app.UpdateHarnessOptions) (domain.Harness, error)
	DeleteHarness(app.DeleteHarnessOptions) error
	SwitchProfile(string, string) (domain.Harness, error)
	AdoptProfile(string, string) error
	CreateProfile(string, string) error
	RenameProfile(string, string, string) error
	CloneProfile(string, string, string, bool) error
	DeleteProfile(string, string, bool) error
}

type Flow string

const (
	FlowDashboard     Flow = "dashboard"
	FlowAdd           Flow = "add"
	FlowUpdate        Flow = "update"
	FlowDeleteHarness Flow = "delete-harness"
	FlowSwitch        Flow = "switch"
	FlowAdopt         Flow = "adopt"
	FlowClone         Flow = "clone"
	FlowDeleteProfile Flow = "delete-profile"
)

type Options struct {
	Flow      Flow
	HarnessID string
}

type screen int

const (
	screenHarnesses screen = iota
	screenDetail
	screenForm
	screenAddLinks
	screenProfileLink
	screenConfirm
	screenProgress
	screenResult
	screenHelp
)

type operation int

const (
	opAdd operation = iota
	opUpdate
	opDeleteHarness
	opSwitch
	opAdopt
	opCreateProfile
	opRenameProfile
	opClone
	opDeleteProfile
	opAddLink
	opUpdateLink
	opDeleteLink
)

type formField struct {
	Label string
	Hint  string
	Input textinput.Model
}

type opResultMsg struct {
	message string
	err     error
}

type Model struct {
	service     Service
	screen      screen
	previous    screen
	width       int
	height      int
	harnesses   []domain.Harness
	profiles    []app.ProfileStatus
	menu        int
	profile     int
	detailMenu  int
	link        int
	current     app.ProfileStatus
	paths       app.Paths
	harness     domain.Harness
	op          operation
	fields      []formField
	field       int
	addDraft    addHarnessDraft
	hQuery      string
	pQuery      string
	hDraft      string
	pDraft      string
	hSearching  bool
	pSearching  bool
	confirm     string
	confirmBtn  int
	cloneStep   bool
	cloneActive bool
	deleteMode  string
	message     string
	err         error
	busy        string
	spin        spinner.Model
	option      int
	options     []optionItem
	linkKind    domain.HarnessLinkKind
	linkAction  app.HarnessLinkAction
}

type optionItem struct {
	Label   string
	Value   string
	Checked bool
}

type dashboardItem struct {
	Icon        string
	Label       string
	Description string
	Kind        string
	Harness     int
}

type dashboardSection struct {
	Title string
	Items []dashboardItem
}

type detailItem struct {
	Icon        string
	Label       string
	Description string
	Kind        string
	Profile     int
	Link        int
}

type detailSection struct {
	Title string
	Items []detailItem
}

type addHarnessDraft struct {
	ID             string
	Label          string
	ConfigPath     string
	Links          []domain.HarnessLink
	LinkActions    map[string]app.HarnessLinkAction
	RestartHint    string
	Branch         string
	SourcePath     string
	ProfileName    string
	ImportApproved bool
}
