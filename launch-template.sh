#!/usr/bin/env bash
# Hyperfocus launch script
# Project: {name}
# Type: {type}
# Path: {path}
#
# ── Customize me! ───────────────────────────────────────────────────
# This script runs when you open the project from Hyperfocus.
# Edit the tmux layout below to match your workflow (e.g. additional
# panes for dev servers, tests, logs, etc.).
#
# Available placeholders (substituted when the project is created):
#   {name}  — project name
#   {path}  — absolute path to the project directory
#   {type}  — project type description

set -e

REPO_DIR="{path}"
SESSION_NAME="{name}"

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
