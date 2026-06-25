package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ── Types ────────────────────────────────────────────────────────────────

type Project struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`      // language name, e.g. "python", "javascript"
	Framework string    `json:"framework"` // framework name, e.g. "uv", "vite-react"
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
}

// ── Path helpers ─────────────────────────────────────────────────────────

func hfConfigDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "hf")
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

// Legacy mapping for projects.json files that used integer enums.
var legacyTypeMap = map[int]string{
	0: "python",
	1: "javascript",
	2: "unknown",
}

var legacyFwMap = map[int]string{
	0: "",
	1: "nextjs",
	2: "vite-react",
	3: "vite-vue",
	4: "vite-svelte",
	5: "astro",
	6: "plain",
}

func loadProjects() ([]Project, error) {
	path := projectsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}

	// Try new string-based format first
	var projects []Project
	if err := json.Unmarshal(data, &projects); err == nil {
		if projects == nil {
			return []Project{}, nil
		}
		return projects, nil
	}

	// Fallback: legacy integer-based format → migrate
	var legacy []struct {
		Name      string    `json:"name"`
		Type      int       `json:"type"`
		Framework int       `json:"framework"`
		Path      string    `json:"path"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("reading projects.json: %w", err)
	}

	for _, l := range legacy {
		projects = append(projects, Project{
			Name:      l.Name,
			Type:      legacyTypeMap[l.Type],
			Framework: legacyFwMap[l.Framework],
			Path:      l.Path,
			CreatedAt: l.CreatedAt,
		})
	}

	// Persist migrated data
	_ = saveProjects(projects)
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

func adoptProject(path, name string) error {
	// Expand ~
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	// Resolve to absolute path first so filepath.Base works for ".", ".." etc.
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

	// Use custom name if provided, otherwise derive from directory name
	if name == "" {
		name = filepath.Base(absPath)
	}

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

	// Try to auto-detect the language from project files
	cfg, _ := LoadConfig()
	lang := "unknown"
	if cfg != nil {
		lang = DetectLanguage(absPath, cfg)
	}

	project := Project{
		Name:      name,
		Type:      lang,
		Framework: "",
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
