// Package documents implements the centre document-list panel.
package documents

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/keymap"
	"github.com/saheersk/lazymongo/internal/tui/style"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FetchPageFn is the async command injected at construction.
type FetchPageFn func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd

// InsertFn inserts a new document and returns a DocumentCreated message.
type InsertFn func(db, col string, doc bson.M) tea.Cmd

// ReplaceFn replaces a document by _id and returns a DocumentUpdated message.
type ReplaceFn func(db, col string, id interface{}, doc bson.M) tea.Cmd

// DeleteFn deletes a document by _id and returns a DocumentDeleted message.
type DeleteFn func(db, col string, id interface{}) tea.Cmd

// AggregateFn runs a pipeline and returns an AggregateResult message.
type AggregateFn func(db, col string, pipeline bson.A) tea.Cmd

// ExportFn exports documents matching filter in the given format and returns an ExportDone message.
type ExportFn func(db, col string, filter bson.M, sort bson.D, format string) tea.Cmd

// inputMode distinguishes which inline bar is active.
type inputMode int

const (
	modeNone   inputMode = iota
	modeFilter           // '/' — filter query
	modeSort             // 's' — sort field
)

// Model is the bubbletea model for the document list panel.
type Model struct {
	db         string
	collection string

	docs     []bson.M
	columns  []string // field names shown as table columns
	cursor   int
	page     int
	total    int64
	pageSize int

	filter     bson.M
	sort       bson.D
	filterExpr string // raw text of the active filter (for display + re-edit)
	sortExpr   string // raw text of the active sort   (for display + re-edit)

	mode      inputMode
	input     textinput.Model
	inputErr  string

	deleteConfirm bool // waiting for y/N confirmation before deleting

	aggMode     bool   // true → showing aggregate results, not the live collection
	aggPipeline string // raw pipeline text used in the last aggregate run

	focused       bool
	loading       bool
	pendingBottom bool
	err           error

	width, height int

	fetchPage   FetchPageFn
	insertFn    InsertFn
	replaceFn   ReplaceFn
	deleteFn    DeleteFn
	aggregateFn AggregateFn
	exportFn    ExportFn

	spinner spinner.Model
	th      *style.Theme
	km      *keymap.Map
}

// New constructs a document panel. It starts empty; a CollectionSelected
// message triggers the first load.
func New(th *style.Theme, km *keymap.Map,
	fetchPage FetchPageFn,
	insertFn InsertFn,
	replaceFn ReplaceFn,
	deleteFn DeleteFn,
	aggregateFn AggregateFn,
	exportFn ExportFn,
) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	ti := textinput.New()
	ti.CharLimit = 256

	return Model{
		pageSize:    50,
		fetchPage:   fetchPage,
		insertFn:    insertFn,
		replaceFn:   replaceFn,
		deleteFn:    deleteFn,
		aggregateFn: aggregateFn,
		exportFn:    exportFn,
		spinner:     sp,
		input:       ti,
		th:          th,
		km:          km,
	}
}

// InInputMode reports whether the panel has captured focus for a modal
// interaction (filter/sort bar or delete confirmation). The app uses
// this to bypass global key handlers so keystrokes like q, h, esc don't
// accidentally trigger navigation while the user is interacting.
func (m Model) InInputMode() bool { return m.mode != modeNone || m.deleteConfirm }

// DB returns the currently loaded database name.
func (m Model) DB() string { return m.db }

// ColName returns the currently loaded collection name.
func (m Model) ColName() string { return m.collection }

// Filter returns the active query filter (nil if none).
func (m Model) Filter() bson.M { return m.filter }

// SortDoc returns the active sort document (nil if none).
func (m Model) SortDoc() bson.D { return m.sort }

// FilterExpr returns the raw filter expression string for display.
func (m Model) FilterExpr() string { return m.filterExpr }

// InAggMode reports whether aggregate results are currently displayed.
func (m Model) InAggMode() bool { return m.aggMode }

// Init is a no-op; document fetching begins when a collection is selected.
func (m Model) Init() tea.Cmd { return nil }

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

// ActiveDoc returns the document under the cursor, or nil.
func (m Model) ActiveDoc() bson.M {
	if m.cursor >= 0 && m.cursor < len(m.docs) {
		return m.docs[m.cursor]
	}
	return nil
}

// Collection returns "db > collection" for display.
func (m Model) Collection() string {
	if m.db == "" {
		return ""
	}
	if m.collection == "" {
		return m.db
	}
	return m.db + " > " + m.collection
}

func (m Model) PageInfo() (page, total int, docCount int64) {
	return m.page, m.pageCount(), m.total
}

func (m Model) pageCount() int {
	if m.total == 0 {
		return 1
	}
	pc := int(m.total) / m.pageSize
	if int(m.total)%m.pageSize != 0 {
		pc++
	}
	return pc
}

func (m Model) clamp() Model {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(m.docs) > 0 && m.cursor >= len(m.docs) {
		m.cursor = len(m.docs) - 1
	}
	return m
}

func (m Model) loadPage(page int) tea.Cmd {
	if m.db == "" || m.collection == "" {
		return nil
	}
	return m.fetchPage(m.db, m.collection, m.filter, m.sort, page)
}
