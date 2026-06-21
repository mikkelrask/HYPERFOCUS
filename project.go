package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ── Types ────────────────────────────────────────────────────────────────

type ProjectType int

const (
	TypePython ProjectType = iota
	TypeJSTS
	TypeUnknown
)

func (pt ProjectType) String() string {
	switch pt {
	case TypePython:
		return "Python"
	case TypeJSTS:
		return "JS/TS"
	default:
		return "Unknown"
	}
}

type Framework int

const (
	FrameworkNone Framework = iota
	FrameworkNextJS
	FrameworkViteReact
	FrameworkViteVue
	FrameworkViteSvelte
	FrameworkAstro
	FrameworkPlainNPM
)

var frameworkNames = map[Framework]string{
	FrameworkNone:       "",
	FrameworkNextJS:     "Next.js",
	FrameworkViteReact:  "Vite + React",
	FrameworkViteVue:    "Vite + Vue",
	FrameworkViteSvelte: "Vite + Svelte",
	FrameworkAstro:      "Astro",
	FrameworkPlainNPM:   "Plain npm",
}

func (f Framework) String() string {
	if s, ok := frameworkNames[f]; ok {
		return s
	}
	return "?"
}

type Project struct {
	Name      string      `json:"name"`
	Type      ProjectType `json:"type"`
	Framework Framework   `json:"framework"`
	Path      string      `json:"path"`
	CreatedAt time.Time   `json:"created_at"`
}

// ── Path helpers ─────────────────────────────────────────────────────────

func hfConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hf")
}

func projectsFilePath() string {
	return filepath.Join(hfConfigDir(), "projects.json")
}

func projectScriptDir(name string) string {
	return filepath.Join(hfConfigDir(), "projects", name)
}

func projectLaunchScript(name string) string {
	return filepath.Join(projectScriptDir(name), "launch.sh")
}

// ── Persistence ──────────────────────────────────────────────────────────

func loadProjects() ([]Project, error) {
	path := projectsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}
	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	if projects == nil {
		return []Project{}, nil
	}
	return projects, nil
}

func saveProjects(projects []Project) error {
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(hfConfigDir(), 0755); err != nil {
		return err
	}
	return os.WriteFile(projectsFilePath(), data, 0644)
}

func findProject(name string) (Project, error) {
	projects, err := loadProjects()
	if err != nil {
		return Project{}, err
	}
	for _, p := range projects {
		if p.Name == name {
			return p, nil
		}
	}
	return Project{}, fmt.Errorf("project %q not found", name)
}

// ── Adopt (CLI) ──────────────────────────────────────────────────────────

func adoptProject(path string) error {
	// Expand ~
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	// Resolve to absolute path FIRST so filepath.Base works for ".", "..", etc.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve path %s: %w", path, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("cannot access %s: %w", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absPath)
	}

	name := filepath.Base(absPath)

	projects, err := loadProjects()
	if err != nil {
		return err
	}
	for _, p := range projects {
		if p.Name == name {
			return fmt.Errorf("project %q already exists in hyperfocus", name)
		}
		if p.Path == absPath {
			return fmt.Errorf("path %s is already adopted as project %q", path, p.Name)
		}
	}

	project := Project{
		Name:      name,
		Type:      TypeUnknown,
		Framework: FrameworkNone,
		Path:      absPath,
		CreatedAt: time.Now(),
	}

	projects = append(projects, project)
	if err := saveProjects(projects); err != nil {
		return err
	}
	if err := createLaunchScript(project); err != nil {
		return fmt.Errorf("project saved but launch script creation failed: %w", err)
	}
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
