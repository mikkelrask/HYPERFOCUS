package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	screenAdoptName
)

// ── Messages ─────────────────────────────────────────────────────────────

type createResultMsg struct {
	project Project
	err     error
}

// ── Model ────────────────────────────────────────────────────────────────

type model struct {
	screen screen

	// Config (languages & frameworks)
	config *Config

	// Project registry
	projects []Project
	filtered []Project
	cursor   int

	// Search
	searchInput textinput.Model

	// Create wizard
	createName string
	createType string // language name, e.g. "python"
	createFW   string // framework name, e.g. "vite-react"
	nameInput  textinput.Model
	typeCursor int
	fwCursor   int

	// Cached options (derived from config so we don't recompute every frame)
	typeOptions   []LanguageConfig
	fwOptions     []FrameworkConfig
	currentFwOpts []FrameworkConfig // frameworks for the currently selected language

	// Running scaffolding
	creating       bool
	createResult   error
	createdProject Project
	spinner        spinner.Model

	// Adopt
	pathInput         textinput.Model
	adoptPath         string // resolved absolute path, set after path entry
	adoptDetectedName string // directory name, shown as default
	adoptNameInput    textinput.Model
	adoptTabMatches   []string // path completion candidates
	adoptTabIdx       int      // index into adoptTabMatches

	// Terminal
	width, height int
	ready         bool

	// Post-TUI action
	openProject string
}

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

	ani := textinput.New()
	ani.Placeholder = "my-project"
	ani.Prompt = "Name: "
	ani.Width = 40
	ani.CharLimit = 64

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	projects, _ := loadProjects()
	if projects == nil {
		projects = []Project{}
	}

	cfg, _ := LoadConfig()
	typeOpts := cfg.Languages

	// Pre-populate fwOptions with first language's frameworks
	var fwOpts []FrameworkConfig
	if len(typeOpts) > 0 {
		fwOpts = typeOpts[0].Frameworks
	}

	return model{
		screen:          screenList,
		config:          cfg,
		projects:        projects,
		filtered:        projects,
		cursor:          0,
		searchInput:     si,
		nameInput:       ni,
		pathInput:       pi,
		adoptNameInput:  ani,
		adoptTabMatches: nil,
		adoptTabIdx:     0,
		spinner:         s,
		createType:      "",
		createFW:        "",
		typeCursor:      0,
		fwCursor:        0,
		typeOptions:     typeOpts,
		fwOptions:       fwOpts,
		currentFwOpts:   fwOpts,
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
		m.adoptNameInput, cmd = m.adoptNameInput.Update(msg)
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
	case screenAdoptName:
		return m.handleAdoptNameKey(msg)
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
		m.adoptTabMatches = nil
		m.adoptTabIdx = 0
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
		// Update framework options for the newly selected language
		lang := m.typeOptions[m.typeCursor]
		m.fwOptions = lang.Frameworks
		m.fwCursor = 0
		return m, nil
	case "down", "j":
		if m.typeCursor < len(m.typeOptions)-1 {
			m.typeCursor++
		}
		// Update framework options for the newly selected language
		lang := m.typeOptions[m.typeCursor]
		m.fwOptions = lang.Frameworks
		m.fwCursor = 0
		return m, nil
	case "enter":
		m.createType = m.typeOptions[m.typeCursor].Name
		m.createFW = ""
		// If the language has multiple frameworks, show the framework picker
		if len(m.typeOptions[m.typeCursor].Frameworks) > 1 {
			m.fwOptions = m.typeOptions[m.typeCursor].Frameworks
			m.fwCursor = 0
			m.screen = screenCreateFramework
			return m, nil
		}
		// Single framework (or none) → skip straight to creation
		if len(m.typeOptions[m.typeCursor].Frameworks) == 1 {
			m.createFW = m.typeOptions[m.typeCursor].Frameworks[0].Name
		}
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
		if m.fwCursor < len(m.fwOptions)-1 {
			m.fwCursor++
		}
		return m, nil
	case "enter":
		m.createFW = m.fwOptions[m.fwCursor].Name
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

	cfg := m.config
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := scaffoldProject(project, cfg)
			if err != nil {
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
	case "tab":
		return m.completeAdoptPath()
	case "enter":
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			return m, nil
		}
		// Resolve and validate path, then go to name screen
		resolved, name, err := resolveAdoptPath(path)
		if err != nil {
			// Show error by going back to list — simplified for now
			m.screen = screenList
			m.pathInput.Blur()
			m.searchInput.Focus()
			return m, nil
		}
		m.adoptPath = resolved
		m.adoptDetectedName = name
		m.adoptNameInput.SetValue("")
		m.adoptNameInput.Placeholder = name
		m.adoptNameInput.Focus()
		m.pathInput.Blur()
		m.screen = screenAdoptName
		return m, nil
	case "esc":
		m.screen = screenList
		m.pathInput.Blur()
		m.searchInput.Focus()
		m.adoptTabMatches = nil
		return m, nil
	}

	m.adoptTabMatches = nil // reset completion on any other key
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

// ── Path resolution ──────────────────────────────────────────────────────

func resolveAdoptPath(path string) (resolvedPath, name string, err error) {
	// Expand ~
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("not a directory")
	}

	return absPath, filepath.Base(absPath), nil
}

// ── Path completion ──────────────────────────────────────────────────────

func (m model) completeAdoptPath() (tea.Model, tea.Cmd) {
	val := m.pathInput.Value()
	if val == "" {
		return m, nil
	}

	// Normalise bare ~ to ~/
	if val == "~" {
		val = "~/"
	}

	// Expand ~ to home for filesystem ops
	searchPath := val
	if len(searchPath) > 0 && searchPath[0] == '~' {
		home, _ := os.UserHomeDir()
		searchPath = filepath.Join(home, searchPath[1:])
	}

	// Determine directory to list and filename prefix
	endsWithSep := strings.HasSuffix(val, "/") || strings.HasSuffix(val, "\\")
	var dir, prefix string
	if endsWithSep {
		dir = searchPath
		prefix = ""
	} else {
		dir = filepath.Dir(searchPath)
		prefix = filepath.Base(searchPath)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return m, nil
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		if prefix == "" || strings.HasPrefix(name, prefix) {
			if e.IsDir() {
				matches = append(matches, name+"/")
			} else {
				matches = append(matches, name)
			}
		}
	}
	// Sort so Tab gives a predictable result
	// (os.ReadDir order is filesystem-dependent)
	sort.Strings(matches)

	if len(matches) == 0 {
		return m, nil
	}

	// Use the first match (no cycling — type more chars to narrow)
	match := matches[0]

	// Build the completed path, preserving ~ prefix in display
	parent := filepath.Dir(val)
	var completed string
	switch {
	case endsWithSep:
		completed = val + match
	case parent == ".":
		completed = match
	case parent == "/":
		completed = "/" + match
	default:
		completed = parent + "/" + match
	}

	m.adoptTabMatches = nil
	m.adoptTabIdx = 0
	m.pathInput.SetValue(completed)
	m.pathInput.SetCursor(len(completed))
	return m, nil
}

// ── Screen: Adopt Name ───────────────────────────────────────────────────

func (m model) handleAdoptNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.adoptNameInput.Value())
		if name == "" {
			// Revert to detected name
			name = m.adoptDetectedName
		}
		if err := adoptProject(m.adoptPath, name); err != nil {
			// Adopt failed; go back to list
			m.screen = screenList
			m.adoptNameInput.Blur()
			m.searchInput.Focus()
			projects, _ := loadProjects()
			m.projects = projects
			m.filterProjects()
			return m, nil
		}
		m.screen = screenList
		m.adoptNameInput.Blur()
		m.searchInput.Focus()
		projects, _ := loadProjects()
		m.projects = projects
		m.filtered = projects
		m.cursor = 0
		return m, nil
	case "esc":
		m.screen = screenAdopt
		m.adoptNameInput.Blur()
		m.pathInput.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	m.adoptNameInput, cmd = m.adoptNameInput.Update(msg)
	return m, cmd
}
