package detail

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/util"
)

// Update handles all messages for the detail panel.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {

	case msg.DocumentSelected:
		return m.load(message.Doc), nil

	case msg.CollectionSelected:
		// clear the panel when the user navigates to a new collection
		m.doc = nil
		m.rawJSON = ""
		m.docID = ""
		if m.ready {
			m.viewport.SetContent("")
			m.viewport.GotoTop()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(message)
	}

	// forward to viewport (handles mouse scroll, etc.)
	if m.ready {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(message)
		return m, vpCmd
	}
	return m, nil
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case isKey(key, m.km.Down):
		m.viewport.ScrollDown(1)
		return m, nil

	case isKey(key, m.km.Up):
		m.viewport.ScrollUp(1)
		return m, nil

	case isKey(key, m.km.PageDown):
		m.viewport.HalfPageDown()
		return m, nil

	case isKey(key, m.km.PageUp):
		m.viewport.HalfPageUp()
		return m, nil

	case isKey(key, m.km.Top):
		m.viewport.GotoTop()
		return m, nil

	case isKey(key, m.km.Bottom):
		m.viewport.GotoBottom()
		return m, nil

	case key.String() == "y":
		// copy _id to clipboard
		if m.docID != "" {
			_ = util.CopyToClipboard(m.docID)
			return m, statusCmd("copied _id to clipboard")
		}
		return m, nil

	case key.String() == "Y":
		// copy full JSON to clipboard
		if m.rawJSON != "" {
			_ = util.CopyToClipboard(m.rawJSON)
			return m, statusCmd("copied document JSON to clipboard")
		}
		return m, nil
	}

	return m, nil
}

func statusCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return msg.StatusUpdate{Text: text}
	}
}

func isKey(km tea.KeyMsg, b interface{ Keys() []string }) bool {
	for _, k := range b.Keys() {
		if km.String() == k {
			return true
		}
	}
	return false
}
