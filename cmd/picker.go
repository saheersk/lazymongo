package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/config"
)

// pickerModel is a minimal bubbletea program for choosing a connection profile.
type pickerModel struct {
	profiles []config.Connection
	cursor   int
	chosen   string
	cancelled bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "j", "down":
			m.cursor = (m.cursor + 1) % len(m.profiles)
		case "k", "up":
			m.cursor = (m.cursor - 1 + len(m.profiles)) % len(m.profiles)
		case "enter":
			m.chosen = m.profiles[m.cursor].Name
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	var sb strings.Builder
	sb.WriteString("\n  Select a connection profile:\n\n")
	for i, conn := range m.profiles {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}
		uri := conn.URI
		if len(uri) > 50 {
			uri = uri[:49] + "…"
		}
		sb.WriteString(fmt.Sprintf("  %s%-20s  %s\n", cursor, conn.Name, uri))
	}
	sb.WriteString("\n  j/k navigate  enter select  q/esc use first\n")
	return sb.String()
}
