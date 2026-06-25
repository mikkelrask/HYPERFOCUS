package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	adoptPath := flag.String("adopt", "", "Adopt an existing repository at the given path")
	adoptName := flag.String("name", "", "Display name for the adopted project (default: directory name)")
	flag.Parse()

	// ── CLI: adopt an existing repo ─────────────────────────────────────
	if *adoptPath != "" {
		err := adoptProject(*adoptPath, *adoptName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Project adopted successfully!")
		return
	}

	// ── TUI ─────────────────────────────────────────────────────────────
	p := tea.NewProgram(
		newModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	finalModel, ok := m.(model)
	if !ok {
		return
	}

	// ── Post-TUI: open a project's launch script ────────────────────────
	if finalModel.openProject != "" {
		script := projectLaunchScript(finalModel.openProject)
		cmd := exec.Command("bash", script)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}
