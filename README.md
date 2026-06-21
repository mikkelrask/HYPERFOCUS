# Hyperfocus

A terminal UI project launcher and scaffolder — browse, create, and switch between your development projects with fuzzy search, mouse support, and per-project tmux launch scripts.

![screenshot](screenshot.png)

## Dependencies

**Runtime:**

- [`tmux`](https://github.com/tmux/tmux) — session management
- [`bash`](https://www.gnu.org/software/bash/) — launch script execution
- [`git`](https://git-scm.com/) — initialised in every new project

**Optional (for scaffolding new projects):**

- Python projects — [`uv`](https://docs.astral.sh/uv/)
- JS/TS projects — [`Node.js`](https://nodejs.org/) + [`npm`](https://www.npmjs.com/)

**Build:**

- [Go](https://go.dev/) 1.22+

## Install

```bash
go install github.com/yourusername/hyperfocus@latest
```

Or build from source:

```bash
git clone https://github.com/yourusername/hyperfocus ~/Repos/hf
cd ~/Repos/hf
go build -o hf .
cp hf ~/.local/bin/    # or anywhere on your $PATH
```

## Quick start

```bash
# Launch the TUI
hf

# Adopt an existing repository
hf --adopt ~/Repos/my-project
```

## Usage

### TUI

```
hf
```

Opens a full-terminal list of your registered projects. Type to filter, click or press Enter to open one.

| Key | Action |
|---|---|
| `↑` `↓` | Navigate the list |
| `Enter` | Open selected project (runs its launch script) |
| `Ctrl+n` | Create a new project |
| `Ctrl+a` | Adopt an existing repository |
| `Ctrl+e` | Edit selected project's launch script (`$EDITOR`) |
| `Ctrl+q` / `Ctrl+c` | Quit |
| `Esc` | Clear search / quit |

**Mouse:** Click a project to select and open it.

### Adopt an existing repo

```bash
hf --adopt ~/Repos/some-project
# or from inside the repo:
cd ~/Repos/some-project && hf --adopt .
```

Registers the repo with Hyperfocus and creates a default launch script at `~/.hf/projects/some-project/launch.sh`.

### Create a new project

Press `Ctrl+n` in the TUI and follow the wizard:

1. **Project name** — alphanumeric, hyphens, underscores, dots
2. **Type** — Python or JavaScript / TypeScript
3. **Framework** (JS/TS only):
   - Next.js
   - Vite + React
   - Vite + Vue
   - Vite + Svelte
   - Astro
   - Plain npm

Hyperfocus will scaffold the project in `~/Repos/`, run `git init`, and create a launch script you can customise later.

## Per-project launch scripts

Each project gets a shell script at:

```
~/.hf/projects/<name>/launch.sh
```

When you open a project from the TUI, this script runs. The default opens a tmux session with `nvim` in the top pane and a shell in the bottom — the same layout as a classic fzf-based switcher.

Edit the script per project to customise the tmux layout (e.g. open a dev server in a second window, launch tests, etc.). Use `Ctrl+e` inside the TUI to jump straight to the editor.

## Data

All Hyperfocus state lives under `~/.hf/`:

```
~/.hf/
├── projects.json           # Project registry (JSON)
└── projects/
    └── <name>/
        └── launch.sh        # Per-project launch script
```

## Building

```bash
cd ~/Repos/hf
go build -o hf .
```

The binary is standalone — it only needs `~/.hf/` at runtime and the external tools listed under Dependencies.

## License

MIT
