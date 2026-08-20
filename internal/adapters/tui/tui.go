package tui

import (
	"io"
	"os"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	accent             = lipgloss.Color("63")
	green              = lipgloss.Color("42")
	yellow             = lipgloss.Color("214")
	red                = lipgloss.Color("196")
	muted              = lipgloss.Color("245")
	titleStyle         = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	panelStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 2)
	rowStyle           = lipgloss.NewStyle().PaddingLeft(1)
	selectStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(accent).PaddingLeft(1).PaddingRight(1)
	helpStyle          = lipgloss.NewStyle().Foreground(muted)
	infoStyle          = lipgloss.NewStyle().Border(lipgloss.Border{Left: "│"}).BorderForeground(lipgloss.Color("39")).PaddingLeft(1).Foreground(lipgloss.Color("39"))
	headerStyle        = lipgloss.NewStyle().Border(lipgloss.Border{Left: "┃"}).BorderForeground(accent).PaddingLeft(1)
	keyStyle           = lipgloss.NewStyle().Foreground(muted)
	buttonStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(accent).Padding(0, 2)
	confirmButtonStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("22")).Background(lipgloss.Color("157")).Bold(true).Padding(0, 2)
	warnStyle          = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	dangerStyle        = lipgloss.NewStyle().Foreground(red).Bold(true)
	okStyle            = lipgloss.NewStyle().Foreground(green).Bold(true)
	errStyle           = lipgloss.NewStyle().Foreground(red).Bold(true)
)

func init() {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

func Run(service Service) error { return RunWithOptions(service, Options{}) }

func RunFlow(service Service, flow Flow, harnessID string) error {
	return RunWithOptions(service, Options{Flow: flow, HarnessID: harnessID})
}

func RunWithOptions(service Service, opts Options) error {
	_, err := tea.NewProgram(NewModelWithOptions(service, opts), tea.WithAltScreen()).Run()
	return err
}

func RunWithIO(service Service, input io.Reader, output io.Writer) error {
	_, err := tea.NewProgram(NewModel(service), tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen()).Run()
	return err
}

func NewModel(service Service) Model { return NewModelWithOptions(service, Options{}) }

func NewModelWithOptions(service Service, opts Options) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = okStyle
	m := Model{service: service, screen: screenHarnesses, width: 88, height: 28, spin: sp}
	m.loadHarnesses()
	m.applyInitialFlow(opts)
	return m
}

func (m *Model) applyInitialFlow(opts Options) {
	if opts.Flow == "" || opts.Flow == FlowDashboard {
		return
	}
	if opts.HarnessID != "" {
		for i, h := range m.harnesses {
			if h.ID == opts.HarnessID {
				m.menu = i
				m.loadDetail()
				break
			}
		}
	}
	switch opts.Flow {
	case FlowAdd:
		m.startForm(opAdd)
	case FlowUpdate:
		m.startHarnessSelection(opUpdate)
	case FlowDeleteHarness:
		m.startHarnessSelection(opDeleteHarness)
	case FlowSwitch:
		m.startProfileSelection(opSwitch)
	case FlowAdopt:
		m.startForm(opAdopt)
	case FlowClone:
		m.startProfileSelection(opClone)
	case FlowDeleteProfile:
		m.startProfileSelection(opDeleteProfile)
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case opResultMsg:
		if msg.err != nil {
			m.resultErr(msg.err)
		} else {
			m.message = msg.message
			m.screen = screenResult
		}
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "?" && m.screen != screenProgress && m.screen != screenDetail {
		m.previous = m.screen
		m.screen = screenHelp
		return m, nil
	}
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	switch m.screen {
	case screenHarnesses:
		return m.updateHarnesses(msg)
	case screenDetail:
		return m.updateDetail(msg), nil
	case screenForm:
		return m.updateForm(msg)
	case screenAddLinks:
		return m.updateAddLinks(msg)
	case screenProfileLink:
		return m.updateProfileLink(msg)
	case screenConfirm:
		return m.updateConfirm(msg)
	case screenResult:
		return m.updateResult(key), nil
	case screenHelp:
		return m.updateHelp(key), nil
	}
	return m, nil
}

func (m Model) updateResult(key string) Model {
	if key == keyEnter || key == keyEsc || key == keyBackspace {
		m.err = nil
		m.message = ""
		m.returnFromResult()
	}
	return m
}

func (m Model) updateHelp(key string) Model {
	if key == keyEsc || key == keyBackspace || key == keyEnter || key == "?" {
		m.screen = m.previous
		if m.screen == screenHelp {
			m.screen = screenHarnesses
		}
	}
	return m
}

func (m *Model) returnFromResult() {
	if m.shouldReturnToProfiles() {
		m.loadDetail()
		return
	}
	m.loadHarnesses()
	m.screen = screenHarnesses
}

func (m Model) shouldReturnToProfiles() bool {
	if m.harness.ID == "" {
		return false
	}
	switch m.op {
	case opSwitch, opAdopt, opCreateProfile, opRenameProfile, opClone, opDeleteProfile:
		return true
	case opAddLink, opUpdateLink, opDeleteLink:
		return true
	}
	return false
}

func (m Model) View() string {
	if m.tooSmall() {
		return m.viewTooSmall()
	}
	switch m.screen {
	case screenHarnesses:
		return m.viewHarnesses()
	case screenDetail:
		return m.viewDetail()
	case screenForm:
		return m.viewForm()
	case screenAddLinks:
		return m.viewAddLinks()
	case screenProfileLink:
		return m.viewProfileLink()
	case screenConfirm:
		return m.viewConfirm()
	case screenProgress:
		return m.viewProgress()
	case screenResult:
		return m.viewResult()
	case screenHelp:
		return m.viewHelp()
	}
	return ""
}

func (m *Model) loadHarnesses() {
	harnesses, err := m.service.ListHarnesses()
	if err != nil {
		m.err = err
		return
	}
	sort.Slice(harnesses, func(i, j int) bool { return harnesses[i].ID < harnesses[j].ID })
	m.harnesses = harnesses
	m.menu = clamp(m.menu, len(m.dashboardItems()))
}

func (m *Model) loadDetail() {
	idxs := m.filteredHarnessIndexes()
	if len(idxs) == 0 {
		return
	}
	harness, err := m.service.InspectHarness(m.harnesses[idxs[clamp(m.menu, len(idxs))]].ID)
	if err != nil {
		m.resultErr(err)
		return
	}
	profiles, err := m.service.ListProfiles(harness.ID)
	if err != nil {
		m.resultErr(err)
		return
	}
	current, _ := m.service.CurrentProfile(harness.ID)
	paths, _ := m.service.Where()
	m.harness = harness
	m.profiles = profiles
	m.current = current
	m.paths = paths
	m.profile = clamp(m.profile, len(m.filteredProfileIndexes()))
	m.detailMenu = clamp(m.detailMenu, len(m.detailItems()))
	m.screen = screenDetail
}

func (m *Model) resultErr(err error) {
	m.err = err
	m.screen = screenResult
}
