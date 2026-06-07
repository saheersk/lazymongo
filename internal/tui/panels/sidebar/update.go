package sidebar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
)

// Update handles all bubbletea messages for the sidebar panel.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {

	case msg.DatabasesLoaded:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		m.applyDatabases(message.DBs)
		// Pre-fetch collections for all databases so sidebar search works immediately.
		cmds := make([]tea.Cmd, 0, len(message.DBs))
		for _, db := range message.DBs {
			cmds = append(cmds, m.fetchCols(db.Name))
		}
		return m, tea.Batch(cmds...)

	case msg.CollectionsLoaded:
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		m.applyCollections(message.DB, message.Collections)
		m = m.rebuildFlat()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(message)

	default:
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(message)
		if m.searchMode {
			var tiCmd tea.Cmd
			m.searchInput, tiCmd = m.searchInput.Update(message)
			return m, tea.Batch(spCmd, tiCmd)
		}
		return m, spCmd
	}
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.searchMode {
		return m.handleSearchKey(key)
	}

	if len(m.items) == 0 {
		if key.String() == "/" {
			return m.openSearch()
		}
		return m, nil
	}

	switch {
	case key.String() == "/":
		return m.openSearch()

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
		return m, func() tea.Msg {
			return msg.CollectionSelected{DB: it.db, Collection: it.name}
		}

	case isKey(key, m.km.Refresh):
		m.loading = true
		return m, m.fetchDBs()
	}

	return m, nil
}

func (m Model) openSearch() (Model, tea.Cmd) {
	m.searchDB = m.ActiveDB() // remember current DB before cursor resets
	m.searchMode = true
	m.searchInput.SetValue("")
	m.cursor = 0
	return m, m.searchInput.Focus()
}

func (m Model) handleSearchKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		m.searchMode = false
		m.searchInput.SetValue("")
		m.cursor = 0
		return m, nil

	case tea.KeyEnter:
		visible := m.visibleItems()
		if len(visible) == 0 || m.cursor >= len(visible) {
			return m, nil
		}
		it := visible[m.cursor]
		if it.kind == kindCollection {
			m.searchMode = false
			m.searchInput.SetValue("")
			return m, func() tea.Msg {
				return msg.CollectionSelected{DB: it.db, Collection: it.name}
			}
		}
		// Cursor is on a database row — select its first visible collection.
		if it.kind == kindDatabase {
			for i := m.cursor + 1; i < len(visible); i++ {
				if visible[i].kind == kindDatabase {
					break
				}
				if visible[i].kind == kindCollection {
					m.searchMode = false
					m.searchInput.SetValue("")
					col := visible[i]
					return m, func() tea.Msg {
						return msg.CollectionSelected{DB: col.db, Collection: col.name}
					}
				}
			}
		}
		return m, nil

	default:
		// j/k navigate filtered results
		if isKey(key, m.km.Down) {
			m.cursor++
			visible := m.visibleItems()
			if m.cursor >= len(visible) {
				m.cursor = max(0, len(visible)-1)
			}
			return m, nil
		}
		if isKey(key, m.km.Up) {
			m.cursor--
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil
		}
		// everything else feeds the search input
		prev := m.searchInput.Value()
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(key)
		if m.searchInput.Value() != prev {
			m.cursor = 0
		}
		return m, cmd
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
