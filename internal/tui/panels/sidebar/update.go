package sidebar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
)

// Update handles all bubbletea messages for the sidebar panel.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {

	// ---- async results ----

	case msg.DatabasesLoaded:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		m.applyDatabases(message.DBs)
		return m, nil

	case msg.CollectionsLoaded:
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		m.applyCollections(message.DB, message.Collections)
		m = m.rebuildFlat()
		return m, nil

	// ---- spinner tick ----

	case tea.KeyMsg:
		return m.handleKey(message)

	default:
		// forward spinner ticks even when not loading so it stays alive
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(message)
		return m, spCmd
	}
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}

	switch {
	case isKey(key, m.km.Down):
		m.cursor++
		return m.clamp(), nil

	case isKey(key, m.km.Up):
		m.cursor--
		return m.clamp(), nil

	case isKey(key, m.km.Top):
		m.cursor = 0
		return m, nil

	case isKey(key, m.km.Bottom):
		m.cursor = len(m.items) - 1
		return m, nil

	case isKey(key, m.km.Select):
		it := m.items[m.cursor]
		if it.kind == kindDatabase {
			return m.toggleExpand(m.cursor)
		}
		// Collection selected — emit navigation message
		return m, func() tea.Msg {
			return msg.CollectionSelected{
				DB:         it.db,
				Collection: it.name,
			}
		}

	case isKey(key, m.km.Refresh):
		m.loading = true
		return m, m.fetchDBs()
	}

	return m, nil
}

// isKey reports whether a KeyMsg matches any of the keys in b.
func isKey(km tea.KeyMsg, b interface{ Keys() []string }) bool {
	for _, k := range b.Keys() {
		if km.String() == k {
			return true
		}
	}
	return false
}
