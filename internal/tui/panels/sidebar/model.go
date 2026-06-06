// Package sidebar implements the left-hand database/collection tree panel.
package sidebar

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/keymap"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/tui/style"
)

type itemKind int

const (
	kindDatabase   itemKind = iota
	kindCollection          // indented under a database
)

// treeItem is one row in the flattened sidebar list.
type treeItem struct {
	kind       itemKind
	name       string
	db         string // parent database name (collections only)
	expanded   bool   // databases only
	colsLoaded bool   // true once collections have been fetched for this db
}

// FetchDatabasesFn is the async command injected at construction time.
type FetchDatabasesFn func() tea.Cmd

// FetchCollectionsFn is the async command injected at construction time.
type FetchCollectionsFn func(db string) tea.Cmd

// Model is the bubbletea model for the sidebar panel.
type Model struct {
	// flat list rebuilt whenever expand/collapse state changes
	items  []treeItem
	cursor int

	focused bool
	loading bool
	err     error

	width, height int

	// injected so the panel stays decoupled from the mongo package
	fetchDBs  FetchDatabasesFn
	fetchCols FetchCollectionsFn

	spinner spinner.Model
	th      *style.Theme
	km      *keymap.Map
}

// New returns an initialised sidebar model.
// It begins in a loading state; Init() fires the database fetch.
func New(
	th *style.Theme,
	km *keymap.Map,
	fetchDBs FetchDatabasesFn,
	fetchCols FetchCollectionsFn,
) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		loading:   true,
		fetchDBs:  fetchDBs,
		fetchCols: fetchCols,
		spinner:   sp,
		th:        th,
		km:        km,
	}
}

// Init fires the first async command.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchDBs())
}

// SetSize updates the panel dimensions.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

// SetFocused controls focused-border rendering.
func (m Model) SetFocused(f bool) Model {
	m.focused = f
	return m
}

// ActiveDB returns the database name at or above the cursor.
func (m Model) ActiveDB() string {
	if len(m.items) == 0 {
		return ""
	}
	it := m.items[m.cursor]
	if it.kind == kindDatabase {
		return it.name
	}
	return it.db
}

// ActiveCollection returns the collection name at the cursor, or "".
func (m Model) ActiveCollection() string {
	if len(m.items) == 0 {
		return ""
	}
	it := m.items[m.cursor]
	if it.kind == kindCollection {
		return it.name
	}
	return ""
}

// ---- internal helpers ----

func (m Model) clamp() Model {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(m.items) > 0 && m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	return m
}

// applyDatabases replaces the flat list with a fresh set of database rows,
// preserving any existing expand/load state.
func (m *Model) applyDatabases(dbs []msg.DatabaseInfo) {
	prevState := map[string]treeItem{}
	for _, it := range m.items {
		if it.kind == kindDatabase {
			prevState[it.name] = it
		}
	}

	// Collect existing collection items to preserve them
	colsByDB := map[string][]treeItem{}
	for _, it := range m.items {
		if it.kind == kindCollection {
			colsByDB[it.db] = append(colsByDB[it.db], it)
		}
	}

	var flat []treeItem
	for _, db := range dbs {
		prev, had := prevState[db.Name]
		dbItem := treeItem{
			kind:       kindDatabase,
			name:       db.Name,
			expanded:   had && prev.expanded,
			colsLoaded: had && prev.colsLoaded,
		}
		flat = append(flat, dbItem)
		if dbItem.expanded {
			flat = append(flat, colsByDB[db.Name]...)
		}
	}
	m.items = flat
}

// applyCollections merges a fetched collection list into the existing items.
func (m *Model) applyCollections(dbName string, cols []msg.CollectionInfo) {
	// Build new flat list: keep everything, but replace the collections block
	// for this specific database.
	var fresh []treeItem
	for _, it := range m.items {
		if it.kind == kindCollection && it.db == dbName {
			continue // will be re-inserted below
		}
		if it.kind == kindDatabase && it.name == dbName {
			it.colsLoaded = true
			fresh = append(fresh, it)
			if it.expanded {
				for _, c := range cols {
					fresh = append(fresh, treeItem{
						kind: kindCollection,
						name: c.Name,
						db:   dbName,
					})
				}
			}
			continue
		}
		fresh = append(fresh, it)
	}
	m.items = fresh
}

// toggleExpand expands or collapses the database at position idx.
// If expanding for the first time it triggers a collection fetch.
func (m Model) toggleExpand(idx int) (Model, tea.Cmd) {
	it := m.items[idx]
	if it.kind != kindDatabase {
		return m, nil
	}

	newExpanded := !it.expanded
	m.items[idx].expanded = newExpanded

	if newExpanded && !it.colsLoaded {
		// trigger async fetch; rebuild happens when CollectionsLoaded arrives
		return m, m.fetchCols(it.name)
	}

	// Rebuild the flat list in-place for already-loaded dbs
	m.applyCollections(it.name, nil) // no-op but triggers correct rebuild via applyCollections
	// Simpler: just re-expand by reconstructing the flat list
	var flat []treeItem
	skipDB := ""
	colsForDB := map[string][]treeItem{}
	for _, item := range m.items {
		if item.kind == kindCollection {
			colsForDB[item.db] = append(colsForDB[item.db], item)
		}
	}
	for _, item := range m.items {
		if item.kind == kindCollection && item.db != skipDB {
			continue
		}
		if item.kind == kindDatabase {
			flat = append(flat, item)
			if item.expanded {
				flat = append(flat, colsForDB[item.name]...)
			}
		}
	}
	// If the above logic is confusing, use a cleaner approach:
	m = m.rebuildFlat()
	return m, nil
}

// rebuildFlat reconstructs items[] from scratch using the stored expansion state.
func (m Model) rebuildFlat() Model {
	// First pass: gather databases and their collections separately
	var dbs []treeItem
	colsByDB := map[string][]treeItem{}

	for _, it := range m.items {
		switch it.kind {
		case kindDatabase:
			dbs = append(dbs, it)
		case kindCollection:
			colsByDB[it.db] = append(colsByDB[it.db], it)
		}
	}

	var flat []treeItem
	for _, db := range dbs {
		flat = append(flat, db)
		if db.expanded {
			flat = append(flat, colsByDB[db.name]...)
		}
	}
	m.items = flat
	return m
}
