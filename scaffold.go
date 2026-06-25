package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ── Scaffolding (config-driven) ──────────────────────────────────────────

func scaffoldProject(p Project, cfg *Config) error {
	repos := expandPath("~/Repos")
	if err := os.MkdirAll(repos, 0755); err != nil {
		return fmt.Errorf("cannot create ~/Repos: %w", err)
	}

	projectPath := filepath.Join(repos, p.Name)

	// Look up the language in config
	lang := cfg.FindLanguage(p.Type)
	if lang == nil {
		return fmt.Errorf("unknown language %q — check ~/.config/hf/config.yaml", p.Type)
	}

	// Find the framework (or use first one if none specified)
	var fw *FrameworkConfig
	if p.Framework != "" {
		fw = lang.FindFramework(p.Framework)
		if fw == nil {
			return fmt.Errorf("unknown framework %q for language %q", p.Framework, p.Type)
		}
	} else if len(lang.Frameworks) > 0 {
		fw = &lang.Frameworks[0]
	} else {
		return fmt.Errorf("no frameworks defined for language %q", p.Type)
	}

	// Create / scaffold command
	if len(fw.Create) > 0 {
		args := substitute(fw.Create, p.Name, repos, projectPath)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repos
		cmd.Stdin = nil
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("scaffold command failed: %w\n%s", err, string(out))
		}
	}

	// Post-create steps (npm install, etc.)
	for _, step := range fw.PostCreate {
		if len(step) == 0 {
			continue
		}
		args := substitute(step, p.Name, repos, projectPath)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = projectPath
		cmd.Stdin = nil
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("post-create step failed: %w\n%s", err, string(out))
		}
	}

	// git init (always)
	if err := runGitInit(projectPath); err != nil {
		return err
	}

	// Launch script
	if err := createLaunchScript(p); err != nil {
		return fmt.Errorf("project created but launch script failed: %w", err)
	}

	return nil
}

// substitute replaces {name}, {project_path}, {repos_path} in command arguments.
func substitute(args []string, name, reposPath, projectPath string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		a = strings.ReplaceAll(a, "{name}", name)
		a = strings.ReplaceAll(a, "{project_path}", projectPath)
		a = strings.ReplaceAll(a, "{repos_path}", reposPath)
		out[i] = a
	}
	return out
}

func runGitInit(projectPath string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectPath
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\n%s", err, string(out))
	}
	return nil
}

// ── Launch script ────────────────────────────────────────────────────────

func createLaunchScript(p Project) error {
	dir := projectScriptDir(p.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Load config for display labels
	cfg, _ := LoadConfig()
	desc := p.Type
	if cfg != nil {
		desc = cfg.Describe(p.Type, p.Framework)
	}

	// Read launch template — user-editable copy or embedded default
	templatePath := filepath.Join(hfConfigDir(), "launch-template.sh")
	template, err := os.ReadFile(templatePath)
	if err != nil {
		template = defaultLaunchTemplate[:]
	}

	// Substitute project values
	script := string(template)
	script = strings.ReplaceAll(script, "{name}", p.Name)
	script = strings.ReplaceAll(script, "{path}", p.Path)
	script = strings.ReplaceAll(script, "{type}", desc)

	path := projectLaunchScript(p.Name)
	return os.WriteFile(path, []byte(script), 0755)
}
