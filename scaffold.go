package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ── Scaffolding ──────────────────────────────────────────────────────────

func scaffoldProject(p Project) error {
	repos := expandPath("~/Repos")
	if err := os.MkdirAll(repos, 0755); err != nil {
		return fmt.Errorf("cannot create ~/Repos: %w", err)
	}

	projectPath := filepath.Join(repos, p.Name)

	switch p.Type {
	case TypePython:
		if err := scaffoldPython(p, repos); err != nil {
			return err
		}
	case TypeJSTS:
		if err := scaffoldJSTS(p, repos); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown project type")
	}

	// git init
	if err := runGitInit(projectPath); err != nil {
		return err
	}

	// Create launch script
	if err := createLaunchScript(p); err != nil {
		return fmt.Errorf("project created but launch script failed: %w", err)
	}

	return nil
}

func scaffoldPython(p Project, repos string) error {
	cmd := exec.Command("uv", "init", p.Name)
	cmd.Dir = repos
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uv init failed: %w\n%s", err, string(out))
	}
	return nil
}

func scaffoldJSTS(p Project, repos string) error {
	switch p.Framework {
	case FrameworkNextJS:
		return scaffoldNextJS(p, repos)
	case FrameworkViteReact:
		return scaffoldVite(p, repos, "react-ts")
	case FrameworkViteVue:
		return scaffoldVite(p, repos, "vue-ts")
	case FrameworkViteSvelte:
		return scaffoldVite(p, repos, "svelte-ts")
	case FrameworkAstro:
		return scaffoldAstro(p, repos)
	case FrameworkPlainNPM:
		return scaffoldPlainNPM(p, repos)
	default:
		return fmt.Errorf("unknown JS/TS framework")
	}
}

func scaffoldNextJS(p Project, repos string) error {
	args := []string{
		"--yes",
		"create-next-app@latest",
		p.Name,
		"--ts",
		"--eslint",
		"--app",
		"--no-src-dir",
		"--import-alias", "@/*",
		"--use-npm",
		"--no-tailwind",
	}
	cmd := exec.Command("npx", args...)
	cmd.Dir = repos
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create-next-app failed: %w\n%s", err, string(out))
	}
	return nil
}

func scaffoldVite(p Project, repos, template string) error {
	cmd := exec.Command("npx", "--yes", "create-vite@latest", p.Name, "--template", template)
	cmd.Dir = repos
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create-vite failed: %w\n%s", err, string(out))
	}

	// npm install
	projectPath := filepath.Join(repos, p.Name)
	install := exec.Command("npm", "install")
	install.Dir = projectPath
	install.Stdin = nil
	out, err = install.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w\n%s", err, string(out))
	}
	return nil
}

func scaffoldAstro(p Project, repos string) error {
	cmd := exec.Command("npx", "--yes", "create-astro@latest", p.Name,
		"--template", "basics",
		"--typescript", "strict",
		"--no-git",
	)
	cmd.Dir = repos
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create-astro failed: %w\n%s", err, string(out))
	}

	// npm install
	projectPath := filepath.Join(repos, p.Name)
	install := exec.Command("npm", "install")
	install.Dir = projectPath
	install.Stdin = nil
	out, err = install.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w\n%s", err, string(out))
	}
	return nil
}

func scaffoldPlainNPM(p Project, repos string) error {
	projectPath := filepath.Join(repos, p.Name)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return err
	}
	cmd := exec.Command("npm", "init", "-y")
	cmd.Dir = projectPath
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm init failed: %w\n%s", err, string(out))
	}
	return nil
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
`, p.Name, p.Type.String(), p.Path, p.Path, p.Name)

	path := projectLaunchScript(p.Name)
	return os.WriteFile(path, []byte(script), 0755)
}
