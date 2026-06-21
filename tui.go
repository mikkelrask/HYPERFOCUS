package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type editFileMsg struct {
	path string
}

type editorDoneMsg struct {
	err error
}

// ── Screen identifiers ───────────────────────────────────────────────────

type screen int

const (
	screenList screen = iota
	screenCreateName
	screenCreateType
	screenCreateFramework
	screenCreating
	screenCreateDone
	screenAdopt
)

// ── Messages ─────────────────────────────────────────────────────────────

type createResultMsg struct {
	project Project
	err     error
}

// ── Model ────────────────────────────────────────────────────────────────

type model struct {
	screen screen

	// Project registry
	projects []Project
	filtered []Project
	cursor   int

	// Search
	searchInput textinput.Model

	// Create wizard
	createName string
	createType ProjectType
	createFW   Framework
	nameInput  textinput.Model
	typeCursor int
	fwCursor   int

	// Running scaffolding
	creating       bool
	createResult   error
	createdProject Project
	spinner        spinner.Model

	// Adopt
	pathInput textinput.Model

	// Terminal
	width, height int
	ready         bool

	// Post-TUI action
	openProject string
}

var (
	typeOptions = []ProjectType{TypePython, TypeJSTS}
	fwOptions   = []Framework{
		FrameworkNextJS,
		FrameworkViteReact,
		FrameworkViteVue,
		FrameworkViteSvelte,
		FrameworkAstro,
		FrameworkPlainNPM,
	}
)

// ── Constructor ──────────────────────────────────────────────────────────

func newModel() model {
	si := textinput.New()
	si.Placeholder = "Search projects…"
	si.Prompt = ""
	si.Width = 40
	si.CharLimit = 80
	si.Focus()

	ni := textinput.New()
	ni.Placeholder = "my-project"
	ni.Prompt = "Name: "
	ni.Width = 40
	ni.CharLimit = 64

	pi := textinput.New()
	pi.Placeholder = "~/path/to/repo  or  /absolute/path"
	pi.Prompt = "Path: "
	pi.Width = 50
	pi.CharLimit = 256

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	projects, _ := loadProjects()
	if projects == nil {
		projects = []Project{}
	}

	return model{
		screen:      screenList,
		projects:    projects,
		filtered:    projects,
		cursor:      0,
		searchInput: si,
		nameInput:   ni,
		pathInput:   pi,
		spinner:     s,
		createType:  TypePython,
		typeCursor:  0,
		fwCursor:    0,
	}
}

// ── Init ─────────────────────────────────────────────────────────────────

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// ── Filter ───────────────────────────────────────────────────────────────

func (m *model) filterProjects() {
	q := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	if q == "" {
		m.filtered = m.projects
	} else {
		m.filtered = nil
		for _, p := range m.projects {
			if strings.Contains(strings.ToLower(p.Name), q) {
				m.filtered = append(m.filtered, p)
			}
		}
	}
	// Clamp cursor
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// ── Update ───────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case createResultMsg:
		m.creating = false
		m.createResult = msg.err
		m.createdProject = msg.project
		if msg.err == nil {
			projects, _ := loadProjects()
			m.projects = projects
			m.filtered = projects
		}
		m.screen = screenCreateDone
		return m, nil

	case editFileMsg:
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		cmd := exec.Command(editor, msg.path)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return editorDoneMsg{err: err}
		})

	case editorDoneMsg:
		return m, nil

	default:
		// Forward to sub-components (BlinkMsg, spinner TickMsg, etc.)
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)
		m.pathInput, cmd = m.pathInput.Update(msg)
		cmds = append(cmds, cmd)
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
}

// ── Key dispatch ─────────────────────────────────────────────────────────

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenList:
		return m.handleListKey(msg)
	case screenCreateName:
		return m.handleCreateNameKey(msg)
	case screenCreateType:
		return m.handleCreateTypeKey(msg)
	case screenCreateFramework:
		return m.handleCreateFrameworkKey(msg)
	case screenCreating:
		return m.handleCreatingKey(msg)
	case screenCreateDone:
		return m.handleCreateDoneKey(msg)
	case screenAdopt:
		return m.handleAdoptKey(msg)
	}
	return m, nil
}

// ── Mouse ────────────────────────────────────────────────────────────────

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	switch m.screen {
	case screenList:
		return m.handleListMouse(msg)
	}
	return m, nil
}

func (m model) handleListMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	y := msg.Y
	// ── Layout (approximate):
	//   0:  "" (padding top, 1 row)
	//   1:  "🗃️ HYPERFOCUS" (header)
	//   2:  "Search: …" (search)
	//   3:  "" (blank after search)
	//   4+: items (3 rows each: name, desc, blank)
	itemStart := 4
	itemH := 3 // name + desc + blank
	idx := (y - itemStart) / itemH

	if idx >= 0 && idx < len(m.filtered) {
		m.cursor = idx
		// Open on click
		return m.openSelectedProject()
	}
	return m, nil
}

// ── Screen: List ─────────────────────────────────────────────────────────

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return m, nil
	case "enter":
		if len(m.filtered) > 0 {
			return m.openSelectedProject()
		}
		return m, nil
	case "ctrl+n":
		m.screen = screenCreateName
		m.searchInput.Blur()
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		return m, nil
	case "ctrl+a":
		m.screen = screenAdopt
		m.searchInput.Blur()
		m.pathInput.SetValue("")
		m.pathInput.Focus()
		return m, nil
	case "ctrl+e":
		if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
			return m, m.editLaunchScript()
		}
		return m, nil
	case "ctrl+q", "ctrl+c":
		return m.cancelProject()
	case "esc":
		if m.searchInput.Value() != "" {
			m.searchInput.SetValue("")
			m.filterProjects()
			return m, nil
		}
		return m.cancelProject()
	}

	// All other keys go to the search input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.filterProjects()
	return m, cmd
}

func (m model) openSelectedProject() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return m, nil
	}
	m.openProject = m.filtered[m.cursor].Name
	return m, tea.Quit
}

func (m model) cancelProject() (tea.Model, tea.Cmd) {
	m.openProject = ""
	return m, tea.Quit
}

func (m model) editLaunchScript() tea.Cmd {
	project := m.filtered[m.cursor]
	return func() tea.Msg {
		return editFileMsg{path: projectLaunchScript(project.Name)}
	}
}

// ── Screen: Create Name ──────────────────────────────────────────────────

func (m model) handleCreateNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return m, nil
		}
		// Validate: alphanumeric, hyphens, underscores, dots only
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
				return m, nil // invalid character, ignore
			}
		}
		m.createName = name
		m.screen = screenCreateType
		m.typeCursor = 0
		m.nameInput.Blur()
		return m, nil
	case "esc":
		m.screen = screenList
		m.nameInput.Blur()
		m.searchInput.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// ── Screen: Create Type ──────────────────────────────────────────────────

func (m model) handleCreateTypeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.typeCursor > 0 {
			m.typeCursor--
		}
		return m, nil
	case "down", "j":
		if m.typeCursor < len(typeOptions)-1 {
			m.typeCursor++
		}
		return m, nil
	case "enter":
		m.createType = typeOptions[m.typeCursor]
		if m.createType == TypeJSTS {
			m.screen = screenCreateFramework
			m.fwCursor = 0
			return m, nil
		}
		// Python → create directly
		return m.startCreate()
	case "esc":
		m.screen = screenCreateName
		m.nameInput.Focus()
		return m, nil
	}
	return m, nil
}

// ── Screen: Create Framework ─────────────────────────────────────────────

func (m model) handleCreateFrameworkKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.fwCursor > 0 {
			m.fwCursor--
		}
		return m, nil
	case "down", "j":
		if m.fwCursor < len(fwOptions)-1 {
			m.fwCursor++
		}
		return m, nil
	case "enter":
		m.createFW = fwOptions[m.fwCursor]
		return m.startCreate()
	case "esc":
		m.screen = screenCreateType
		return m, nil
	}
	return m, nil
}

// ── Start scaffolding ────────────────────────────────────────────────────

func (m model) startCreate() (tea.Model, tea.Cmd) {
	repos := expandPath("~/Repos")
	project := Project{
		Name:      m.createName,
		Type:      m.createType,
		Framework: m.createFW,
		Path:      filepath.Join(repos, m.createName),
		CreatedAt: time.Now(),
	}

	m.screen = screenCreating
	m.creating = true
	m.createResult = nil
	m.createdProject = Project{}

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := scaffoldProject(project)
			if err != nil {
				// Clean up the project from registry if it was saved mid-scaffold
				return createResultMsg{project: project, err: err}
			}
			// Save to registry only on success
			projects, loadErr := loadProjects()
			if loadErr != nil {
				return createResultMsg{project: project, err: loadErr}
			}
			projects = append(projects, project)
			if saveErr := saveProjects(projects); saveErr != nil {
				return createResultMsg{project: project, err: saveErr}
			}
			return createResultMsg{project: project, err: nil}
		},
	)
}

// ── Screen: Creating (spinner) ───────────────────────────────────────────

func (m model) handleCreatingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ignore input while creating; Esc cancels watching but not the process
	if msg.String() == "esc" {
		// Just ignore Esc during creation — can't cancel safely
		return m, nil
	}
	return m, nil
}

// ── Screen: Create Done ──────────────────────────────────────────────────

func (m model) handleCreateDoneKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.createResult == nil && m.createdProject.Name != "" {
			m.openProject = m.createdProject.Name
			return m, tea.Quit
		}
		return m, nil
	case "esc":
		m.screen = screenList
		m.searchInput.Focus()
		// Refresh projects
		projects, _ := loadProjects()
		m.projects = projects
		m.filterProjects()
		return m, nil
	}
	return m, nil
}

// ── Screen: Adopt ────────────────────────────────────────────────────────

func (m model) handleAdoptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			return m, nil
		}
		// Run adopt synchronously (fast operation)
		if err := adoptProject(path); err != nil {
			// Adopt failed; just go back to list
			m.screen = screenList
			m.pathInput.Blur()
			m.searchInput.Focus()
			projects, _ := loadProjects()
			m.projects = projects
			m.filterProjects()
			return m, nil
		}
		m.screen = screenList
		m.pathInput.Blur()
		m.searchInput.Focus()
		projects, _ := loadProjects()
		m.projects = projects
		m.filtered = projects
		m.cursor = 0
		return m, nil
	case "esc":
		m.screen = screenList
		m.pathInput.Blur()
		m.searchInput.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}
