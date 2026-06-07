// Package indexes implements the index-viewer panel.
package indexes

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/keymap"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/tui/style"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FetchIndexesFn loads indexes + stats for a collection.
type FetchIndexesFn func(db, col string) tea.Cmd

// CreateIndexFn creates a new index and returns an IndexCreated message.
type CreateIndexFn func(db, col string, keys bson.D, unique, sparse bool) tea.Cmd

// DropIndexFn drops a named index and returns an IndexDropped message.
type DropIndexFn func(db, col, name string) tea.Cmd

// Model is the bubbletea model for the index panel.
type Model struct {
	db         string
	collection string

	indexes []msg.IndexInfo
	stats   msg.CollectionStats
	cursor  int

	deleteConfirm bool // waiting for y/N before dropping

	focused bool
	loading bool
	err     error

	width, height int

	fetchIndexes FetchIndexesFn
	createIndex  CreateIndexFn
	dropIndex    DropIndexFn

	editor string // editor binary (e.g. "vim", "nvim", "nano")

	spinner spinner.Model
	th      *style.Theme
	km      *keymap.Map
}

// New constructs an indexes panel.
func New(th *style.Theme, km *keymap.Map,
	fetchIndexes FetchIndexesFn,
	createIndex CreateIndexFn,
	dropIndex DropIndexFn,
) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{
		fetchIndexes: fetchIndexes,
		createIndex:  createIndex,
		dropIndex:    dropIndex,
		spinner:      sp,
		th:           th,
		km:           km,
	}
}

// SetEditor sets the editor binary used when creating indexes.
func (m Model) SetEditor(e string) Model {
	if e != "" {
		m.editor = e
	}
	return m
}

// Init is a no-op; loading begins when CollectionSelected arrives.
func (m Model) Init() tea.Cmd { return nil }

// SetSize updates panel dimensions.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

// SetFocused controls the focused-border style.
func (m Model) SetFocused(f bool) Model {
	m.focused = f
	return m
}

// InConfirmMode reports whether the panel is waiting for a drop confirmation.
func (m Model) InConfirmMode() bool { return m.deleteConfirm }

// Load triggers a reload for the current db/collection.
func (m Model) Load(db, col string) (Model, tea.Cmd) {
	m.db = db
	m.collection = col
	m.cursor = 0
	m.indexes = nil
	m.err = nil
	m.deleteConfirm = false
	m.loading = true
	return m, tea.Batch(m.spinner.Tick, m.fetchIndexes(db, col))
}

func (m Model) clamp() Model {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(m.indexes) > 0 && m.cursor >= len(m.indexes) {
		m.cursor = len(m.indexes) - 1
	}
	return m
}

// ActiveIndex returns the index under the cursor, or nil.
func (m Model) ActiveIndex() *msg.IndexInfo {
	if m.cursor >= 0 && m.cursor < len(m.indexes) {
		return &m.indexes[m.cursor]
	}
	return nil
}
