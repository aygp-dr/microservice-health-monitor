package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type model struct {
	items  []string
	cursor int
}

func initialModel() model {
	return model{
		items: []string{"Loading...", "See CLAUDE.md for implementation plan"},
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.items)-1 { m.cursor++ }
		case "k", "up":
			if m.cursor > 0 { m.cursor-- }
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Observability dashboard for microservices"))
	b.WriteString("\n\n")
	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor { cursor = "> " }
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, item))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate  q: quit"))
	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
