package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Kanagawa Colors ─────────────────────────────────────────────────────

const (
	colorFg     = "#dcd7ba" // foreground
	colorBg     = "#1f1f28" // background
	colorCursor = "#c8c093" // cursor, selection foreground
	colorSelBg  = "#2d4f67" // selection background

	colorBlack    = "#16161d" // palette 0
	colorRed      = "#c34043" // palette 1
	colorGreen    = "#76946a" // palette 2
	colorYellow   = "#c0a36e" // palette 3
	colorBlue     = "#7e9cd8" // palette 4
	colorPurple   = "#957fb8" // palette 5
	colorCyan     = "#6a9589" // palette 6
	colorWhite    = "#c8c093" // palette 7
	colorGray     = "#727169" // palette 8 (bright black)
	colorBrRed    = "#e82424" // palette 9
	colorBrGreen  = "#98bb6c" // palette 10
	colorBrYellow = "#e6c384" // palette 11
	colorBrBlue   = "#7fb4ca" // palette 12
	colorBrPurple = "#938aa9" // palette 13
	colorBrCyan   = "#7aa89f" // palette 14
	colorBrWhite  = "#dcd7ba" // palette 15
)

// ── Styles ───────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorBlue)).
			Padding(0, 1)

	projectNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorFg)).
				Bold(true)

	projectDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorGray))

	cursorIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			SetString("❯ ")

	selectedNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBlue)).
				Bold(true)

	selectedDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorWhite))

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			Italic(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorRed))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray))

	radioSelected = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			SetString("◉")

	radioUnselected = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGray)).
			SetString("○")

	docStyle = lipgloss.NewStyle().Padding(1, 2)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWhite))
)

// ── View ─────────────────────────────────────────────────────────────────

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing…"
	}

	var content string
	switch m.screen {
	case screenList:
		content = m.listView()
	case screenCreateName:
		content = m.createNameView()
	case screenCreateType:
		content = m.createTypeView()
	case screenCreateFramework:
		content = m.createFrameworkView()
	case screenCreating:
		content = m.creatingView()
	case screenCreateDone:
		content = m.createDoneView()
	case screenAdopt:
		content = m.adoptView()
	}

	return docStyle.Render(content)
}

// ── List View ────────────────────────────────────────────────────────────

func (m model) listView() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("👻 HYPERFOCUS ON:"))
	b.WriteString("\n")

	// Search
	b.WriteString(m.searchInput.View())
	b.WriteString("\n")
	b.WriteString("\n")

	// Project list or empty state
	if len(m.projects) == 0 {
		b.WriteString(emptyStyle.Render("  No projects yet.\n"))
		b.WriteString(emptyStyle.Render("  Press Ctrl+n to create one, or Ctrl+a to adopt an existing repo.\n"))
	} else if len(m.filtered) == 0 {
		b.WriteString(emptyStyle.Render("  No projects match your search.\n"))
	} else {
		for i, p := range m.filtered {
			selected := i == m.cursor
			prefix := "  "
			if selected {
				prefix = cursorIndicator.String()
			}

			// Project name
			nameStr := p.Name
			if selected {
				nameStr = selectedNameStyle.Render(nameStr)
			} else {
				nameStr = projectNameStyle.Render(nameStr)
			}
			b.WriteString(prefix + nameStr + "\n")

			// Description
			desc := m.config.Describe(p.Type, p.Framework)
			if selected {
				b.WriteString("   " + selectedDescStyle.Render(desc) + "\n")
			} else {
				b.WriteString("   " + projectDescStyle.Render(desc) + "\n")
			}
			b.WriteString("\n")
		}
	}

	// Separator
	b.WriteString(separatorStyle.Render(strings.Repeat("─", 40)))
	b.WriteString("\n")

	// Footer
	b.WriteString(footerStyle.Render("  [Ctrl+n] New  [Ctrl+a] Adopt  [Ctrl+e] Edit script  [↑/↓] Navigate  [Enter] Open  [Ctrl+q] Quit"))
	b.WriteString("\n")

	return b.String()
}

// ── Create Name View ─────────────────────────────────────────────────────

func (m model) createNameView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📦 New Project"))
	b.WriteString("\n\n")
	b.WriteString("  Enter a project name:\n\n")
	b.WriteString("  " + m.nameInput.View() + "\n\n")
	b.WriteString(footerStyle.Render("  [Enter] Continue  [Esc] Cancel"))
	b.WriteString("\n")
	return b.String()
}

// ── Create Type View ─────────────────────────────────────────────────────

func (m model) createTypeView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📦 New Project — Type"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Project: %s\n\n", projectNameStyle.Render(m.createName)))
	b.WriteString("  Select project type:\n\n")

	for i, lang := range m.typeOptions {
		selected := i == m.typeCursor
		radio := radioUnselected.String()
		prefix := "  "
		if selected {
			radio = radioSelected.String()
			prefix = cursorIndicator.String()
		}

		label := lang.Label
		if selected {
			label = selectedNameStyle.Render(label)
		}

		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, radio, label))
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("  [↑/↓] Navigate  [Enter] Confirm  [Esc] Back"))
	b.WriteString("\n")
	return b.String()
}

// ── Create Framework View ────────────────────────────────────────────────

func (m model) createFrameworkView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📦 New Project — Framework"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Project: %s\n", projectNameStyle.Render(m.createName)))
	langLabel := m.config.LanguageLabel(m.createType)
	b.WriteString(fmt.Sprintf("  Type:   %s\n\n", selectedDescStyle.Render(langLabel)))
	b.WriteString("  Select framework:\n\n")

	for i, fw := range m.fwOptions {
		selected := i == m.fwCursor
		radio := radioUnselected.String()
		prefix := "  "
		if selected {
			radio = radioSelected.String()
			prefix = cursorIndicator.String()
		}

		label := fw.Label
		if selected {
			label = selectedNameStyle.Render(label)
		}

		b.WriteString(fmt.Sprintf("%s%s %s\n", prefix, radio, label))
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("  [↑/↓] Navigate  [Enter] Confirm  [Esc] Back"))
	b.WriteString("\n")
	return b.String()
}

// ── Creating View (spinner) ──────────────────────────────────────────────

func (m model) creatingView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📦 Creating Project"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s Scaffolding %s…\n\n", m.spinner.View(), projectNameStyle.Render(m.createName)))
	b.WriteString(fmt.Sprintf("  Type:      %s\n", m.config.LanguageLabel(m.createType)))
	if m.createFW != "" {
		b.WriteString(fmt.Sprintf("  Framework: %s\n", m.config.FrameworkLabel(m.createType, m.createFW)))
	}
	b.WriteString(fmt.Sprintf("  Location:  ~/Repos/%s\n", m.createName))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  ⏳ This may take a moment…"))
	b.WriteString("\n")
	return b.String()
}

// ── Create Done View ─────────────────────────────────────────────────────

func (m model) createDoneView() string {
	var b strings.Builder

	if m.createResult != nil {
		b.WriteString(titleStyle.Render("❌ Creation Failed"))
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("  %v", m.createResult)))
		b.WriteString("\n\n")
		b.WriteString(footerStyle.Render("  [Esc] Back to project list"))
	} else {
		b.WriteString(titleStyle.Render("✅ Project Created"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("  %s %s\n\n", successStyle.Render("✓"), projectNameStyle.Render(m.createdProject.Name)))
		b.WriteString(fmt.Sprintf("  Type:      %s\n", m.config.LanguageLabel(m.createdProject.Type)))
		if m.createdProject.Framework != "" {
			b.WriteString(fmt.Sprintf("  Framework: %s\n", m.config.FrameworkLabel(m.createdProject.Type, m.createdProject.Framework)))
		}
		b.WriteString(fmt.Sprintf("  Location:  ~/Repos/%s\n", m.createdProject.Name))
		b.WriteString(fmt.Sprintf("  Git:       initialized\n"))
		b.WriteString(fmt.Sprintf("  Script:    %s\n", projectLaunchScript(m.createdProject.Name)))
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("  [Enter] Open project  [Esc] Back to list"))
	}
	b.WriteString("\n")
	return b.String()
}

// ── Adopt View ───────────────────────────────────────────────────────────

func (m model) adoptView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📂 Adopt Existing Repo"))
	b.WriteString("\n\n")
	b.WriteString("  Enter the path to an existing repository:\n\n")
	b.WriteString("  " + m.pathInput.View() + "\n\n")
	b.WriteString(footerStyle.Render("  [Enter] Adopt  [Esc] Cancel"))
	b.WriteString("\n")
	return b.String()
}
