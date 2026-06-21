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
		return fmt.Errorf("unknown language %q — check ~/.hf/config.yaml", p.Type)
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

	script := fmt.Sprintf(`#!/usr/bin/env bash
# Hyperfocus launch script
# Project: %s
# Type: %s
# Path: %s
#
# ── Customize me! ───────────────────────────────────────────────────
# This script runs when you open the project from Hyperfocus.
# Edit the tmux layout below to match your workflow (e.g. additional
# panes for dev servers, tests, logs, etc.).

set -e

REPO_DIR="%s"
SESSION_NAME="%s"

# If the session already exists, just switch to it
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    if [ -n "$TMUX" ]; then
        tmux switch-client -t "$SESSION_NAME"
    else
        tmux attach-session -t "$SESSION_NAME"
    fi
    exit 0
fi

# ── Create new tmux session ─────────────────────────────────────────
tmux new-session -s "$SESSION_NAME" -c "$REPO_DIR" -d
tmux rename-window -t "$SESSION_NAME":1 'main'

# Top pane: open nvim
tmux send-keys -t "$SESSION_NAME":1.1 'nvim' C-m

# Bottom pane: shell
tmux split-window -v -t "$SESSION_NAME":1 -c "$REPO_DIR"
tmux resize-pane -t "$SESSION_NAME":1.1 -y "$(($(tput lines) * 80 / 100))"

# ── Optional: add more panes/windows here ───────────────────────────
# Example: open a second window with a dev server
# tmux new-window -t "$SESSION_NAME" -c "$REPO_DIR" -n 'server'
# tmux send-keys -t "$SESSION_NAME":2 'npm run dev' C-m

# Switch or attach
if [ -n "$TMUX" ]; then
    tmux switch-client -t "$SESSION_NAME"
else
    tmux attach-session -t "$SESSION_NAME"
fi
`, p.Name, desc, p.Path, p.Path, p.Name)

	path := projectLaunchScript(p.Name)
	return os.WriteFile(path, []byte(script), 0755)
}
