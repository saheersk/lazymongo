// Package sidebar implements the left-hand database/collection tree panel.
package sidebar

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	items   []treeItem
	allCols map[string][]treeItem // all loaded collections by db name (for search)
	cursor  int

	searchMode  bool
	searchInput textinput.Model

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

// InSearchMode reports whether the search bar is active.
func (m Model) InSearchMode() bool { return m.searchMode }

// visibleItems returns the filtered item list when searching, or all items.
//
// Search syntax:
//   - "orders"       → match any DB or collection containing "orders"
//   - "mydb:orders"  → match DBs containing "mydb", then collections containing "orders"
//   - "mydb:"        → match DBs containing "mydb", show all their collections
//   - ":orders"      → any DB, only collections containing "orders"
func (m Model) visibleItems() []treeItem {
	raw := strings.TrimSpace(m.searchInput.Value())
	if !m.searchMode || raw == "" {
		return m.items
	}

	if idx := strings.Index(raw, ":"); idx >= 0 {
		dbQ := strings.ToLower(raw[:idx])
		colQ := strings.ToLower(raw[idx+1:])
		return m.filterColon(dbQ, colQ)
	}

	return m.filterBoth(strings.ToLower(raw))
}

// filterBoth matches the query against both DB names and collection names.
func (m Model) filterBoth(q string) []treeItem {
	var result []treeItem
	seen := map[string]bool{}

	for _, it := range m.items {
		if it.kind != kindDatabase || seen[it.name] {
			continue
		}
		dbMatch := strings.Contains(strings.ToLower(it.name), q)
		cols := m.allCols[it.name]

		var matchCols []treeItem
		for _, c := range cols {
			if strings.Contains(strings.ToLower(c.name), q) {
				matchCols = append(matchCols, c)
			}
		}

		if !dbMatch && len(matchCols) == 0 {
			continue
		}
		seen[it.name] = true
		result = append(result, it)
		if dbMatch {
			result = append(result, cols...)
		} else {
			result = append(result, matchCols...)
		}
	}
	return result
}

// filterColon handles "dbQuery:colQuery" syntax.
func (m Model) filterColon(dbQ, colQ string) []treeItem {
	var result []treeItem
	seen := map[string]bool{}

	for _, it := range m.items {
		if it.kind != kindDatabase || seen[it.name] {
			continue
		}
		// DB must match dbQ (empty dbQ matches all DBs)
		if dbQ != "" && !strings.Contains(strings.ToLower(it.name), dbQ) {
			continue
		}
		cols := m.allCols[it.name]

		var matchCols []treeItem
		for _, c := range cols {
			if colQ == "" || strings.Contains(strings.ToLower(c.name), colQ) {
				matchCols = append(matchCols, c)
			}
		}

		seen[it.name] = true
		result = append(result, it)
		result = append(result, matchCols...)
	}
	return result
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

	ti := textinput.New()
	ti.Placeholder = "search db / collection…"
	ti.CharLimit = 60

	return Model{
		loading:     true,
		fetchDBs:    fetchDBs,
		fetchCols:   fetchCols,
		spinner:     sp,
		searchInput: ti,
		allCols:     map[string][]treeItem{},
		th:          th,
		km:          km,
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
	m.searchInput.Width = max(4, w-8)
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

// PreferredWidth returns the ideal sidebar panel width based on the longest
// item name currently loaded, clamped to [20, 52].
func (m Model) PreferredWidth() int {
	longest := 10
	check := func(n int) {
		if n > longest {
			longest = n
		}
	}
	for _, it := range m.items {
		switch it.kind {
		case kindDatabase:
			check(len([]rune(it.name)) + 6) // "  ▸ " + border/pad
		case kindCollection:
			check(len([]rune(it.name)) + 8) // "    ● " + border/pad
		}
	}
	for _, cols := range m.allCols {
		for _, col := range cols {
			check(len([]rune(col.name)) + 8)
		}
	}
	w := longest + 4 // 2 border + 2 inner margin
	if w < 20 {
		return 20
	}
	if w > 52 {
		return 52
	}
	return w
}

// Refresh reloads the database list (equivalent to pressing R).
func (m Model) Refresh() (Model, tea.Cmd) {
	m.loading = true
	return m, m.fetchDBs()
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

// applyCollections stores a fetched collection list in allCols and marks the
// database as loaded. rebuildFlat is called by the Update caller afterwards.
func (m *Model) applyCollections(dbName string, cols []msg.CollectionInfo) {
	loaded := make([]treeItem, 0, len(cols))
	for _, c := range cols {
		loaded = append(loaded, treeItem{kind: kindCollection, name: c.Name, db: dbName})
	}
	m.allCols[dbName] = loaded
	for i, it := range m.items {
		if it.kind == kindDatabase && it.name == dbName {
			m.items[i].colsLoaded = true
			break
		}
	}
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

	m = m.rebuildFlat()
	return m, nil
}

// rebuildFlat reconstructs items[] using allCols as the source of collections.
// This ensures that pre-fetched (but not yet expanded) databases show their
// collections correctly when expanded.
func (m Model) rebuildFlat() Model {
	var dbs []treeItem
	for _, it := range m.items {
		if it.kind == kindDatabase {
			dbs = append(dbs, it)
		}
	}
	var flat []treeItem
	for _, db := range dbs {
		flat = append(flat, db)
		if db.expanded {
			flat = append(flat, m.allCols[db.name]...)
		}
	}
	m.items = flat
	return m
}
