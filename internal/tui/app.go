// Package tui is the root bubbletea application model.
// It owns all panels, routes messages between them, and manages focus.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saheersk/lazymongo/internal/mongo"
	"github.com/saheersk/lazymongo/internal/tui/keymap"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/tui/panels/detail"
	"github.com/saheersk/lazymongo/internal/tui/panels/documents"
	"github.com/saheersk/lazymongo/internal/tui/panels/indexes"
	"github.com/saheersk/lazymongo/internal/tui/panels/sidebar"
	"github.com/saheersk/lazymongo/internal/tui/panels/statusbar"
	"github.com/saheersk/lazymongo/internal/tui/style"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type focusedPanel int

const (
	focusSidebar   focusedPanel = iota
	focusDocuments
	focusDetail
	focusIndexes
)

// layout widths
const (
	sidebarWidth   = 28
	sidebarWidthS  = 20 // narrow terminals
	docsWithDetail = 38 // documents panel width when detail is open
)

// App is the root bubbletea model.
type App struct {
	width, height int
	focus         focusedPanel
	showDetail    bool
	showIndexes   bool

	sidebar   sidebar.Model
	documents documents.Model
	detail    detail.Model
	indexes   indexes.Model
	statusbar statusbar.Model

	client *mongo.Client
	th     *style.Theme
	km     *keymap.Map
}

// New creates the root App with all panels initialised.
func New(client *mongo.Client) *App {
	th := style.Default()
	km := keymap.Default()

	fetchDBs := func() tea.Cmd {
		return func() tea.Msg {
			dbs, err := client.ListDatabases()
			return msg.DatabasesLoaded{DBs: dbs, Err: err}
		}
	}

	fetchCols := func(db string) tea.Cmd {
		return func() tea.Msg {
			cols, err := client.ListCollections(db)
			return msg.CollectionsLoaded{DB: db, Collections: cols, Err: err}
		}
	}

	fetchPage := func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
		return func() tea.Msg {
			result, err := client.FindPage(db, col, mongo.QueryOptions{
				Filter:   filter,
				Sort:     sort,
				Page:     page,
				PageSize: 50,
			})
			return msg.DocumentsLoaded{Result: result, Err: err}
		}
	}

	insertDoc := func(db, col string, doc bson.M) tea.Cmd {
		return func() tea.Msg {
			id, err := client.InsertOne(db, col, doc)
			return msg.DocumentCreated{InsertedID: id, Err: err}
		}
	}

	replaceDoc := func(db, col string, id interface{}, doc bson.M) tea.Cmd {
		return func() tea.Msg {
			err := client.ReplaceOne(db, col, id, doc)
			return msg.DocumentUpdated{Err: err}
		}
	}

	deleteDoc := func(db, col string, id interface{}) tea.Cmd {
		return func() tea.Msg {
			err := client.DeleteOne(db, col, id)
			return msg.DocumentDeleted{Err: err}
		}
	}

	aggregateDocs := func(db, col string, pipeline bson.A) tea.Cmd {
		return func() tea.Msg {
			docs, err := client.Aggregate(db, col, pipeline)
			return msg.AggregateResult{Docs: docs, Err: err}
		}
	}

	fetchIndexes := func(db, col string) tea.Cmd {
		return func() tea.Msg {
			idxs, stats, err := client.ListIndexesAndStats(db, col)
			return msg.IndexesLoaded{
				DB:         db,
				Collection: col,
				Indexes:    idxs,
				Stats:      stats,
				Err:        err,
			}
		}
	}

	createIndex := func(db, col string, keys bson.D, unique, sparse bool) tea.Cmd {
		return func() tea.Msg {
			name, err := client.CreateIndex(db, col, keys, unique, sparse)
			return msg.IndexCreated{Name: name, Err: err}
		}
	}

	dropIndex := func(db, col, name string) tea.Cmd {
		return func() tea.Msg {
			err := client.DropIndex(db, col, name)
			return msg.IndexDropped{Err: err}
		}
	}

	return &App{
		client:    client,
		th:        th,
		km:        km,
		focus:     focusSidebar,
		sidebar:   sidebar.New(th, km, fetchDBs, fetchCols),
		documents: documents.New(th, km, fetchPage, insertDoc, replaceDoc, deleteDoc, aggregateDocs),
		detail:    detail.New(th, km),
		indexes:   indexes.New(th, km, fetchIndexes, createIndex, dropIndex),
		statusbar: statusbar.New(th, client.URI()),
	}
}

// Init fires startup commands for all panels.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.sidebar.Init(),
		a.documents.Init(),
		a.detail.Init(),
		a.statusbar.Init(),
	)
}

// Update is the central message dispatcher.
func (a App) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch message := message.(type) {

	// ── window resize ──────────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		a.width = message.Width
		a.height = message.Height
		a = a.applyLayout()
		return &a, nil

	// ── keyboard ───────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		// While the documents panel has an input bar open, route every
		// keystroke directly to it — including q, h, l, esc, etc.
		if a.focus == focusDocuments && a.documents.InInputMode() {
			var cmd tea.Cmd
			a.documents, cmd = a.documents.Update(message)
			return &a, cmd
		}

		// While the indexes panel is in drop-confirm mode, route keys there.
		if a.focus == focusIndexes && a.indexes.InConfirmMode() {
			var cmd tea.Cmd
			a.indexes, cmd = a.indexes.Update(message)
			return &a, cmd
		}

		// global: quit
		if message.String() == "q" || message.String() == "ctrl+c" {
			return &a, tea.Quit
		}

		// global: focus left
		if message.String() == "h" || message.String() == "left" {
			switch a.focus {
			case focusDetail:
				a.focus = focusDocuments
			case focusIndexes:
				a.focus = focusDocuments
			default:
				a.focus = focusSidebar
			}
			a = a.syncFocus()
			return &a, nil
		}

		// global: focus right
		if message.String() == "l" || message.String() == "right" {
			switch a.focus {
			case focusSidebar:
				if a.documents.Collection() != "" {
					a.focus = focusDocuments
					a = a.syncFocus()
				}
			case focusDocuments:
				if a.showDetail {
					a.focus = focusDetail
					a = a.syncFocus()
				} else if a.showIndexes {
					a.focus = focusIndexes
					a = a.syncFocus()
				}
			}
			return &a, nil
		}

		// esc handling
		if message.String() == "esc" {
			switch a.focus {
			case focusDetail:
				a.showDetail = false
				a.focus = focusDocuments
				a = a.syncFocus()
				a = a.applyLayout()
				return &a, nil
			case focusIndexes:
				a.showIndexes = false
				a.focus = focusDocuments
				a = a.syncFocus()
				a = a.applyLayout()
				return &a, nil
			case focusDocuments:
				// Let documents panel handle esc first (agg mode exit),
				// otherwise go to sidebar.
				if a.documents.InAggMode() {
					var cmd tea.Cmd
					a.documents, cmd = a.documents.Update(message)
					return &a, cmd
				}
				a.focus = focusSidebar
				a = a.syncFocus()
				return &a, nil
			}
		}

		// 'a' opens the aggregate pipeline editor from any panel when a
		// collection is loaded. This way focus on sidebar doesn't block it.
		if message.String() == "a" && a.documents.Collection() != "" {
			var cmd tea.Cmd
			a.documents, cmd = a.documents.Update(message)
			a.focus = focusDocuments
			a = a.syncFocus()
			cmds = append(cmds, cmd)
			return &a, tea.Batch(cmds...)
		}

		// I key: toggle indexes panel (from documents focus)
		if message.String() == "I" && a.focus == focusDocuments {
			if a.showIndexes {
				a.showIndexes = false
				a = a.applyLayout()
			} else {
				a.showDetail = false
				a.showIndexes = true
				col := a.documents.Collection()
				if col != "" {
					// documents.Collection() returns "db > col"; derive db/col
					db, c := a.currentDBCol()
					var idxCmd tea.Cmd
					a.indexes, idxCmd = a.indexes.Load(db, c)
					cmds = append(cmds, idxCmd)
				}
				a.focus = focusIndexes
				a = a.syncFocus()
				a = a.applyLayout()
			}
			return &a, tea.Batch(cmds...)
		}

		// route to focused panel
		switch a.focus {
		case focusSidebar:
			var cmd tea.Cmd
			a.sidebar, cmd = a.sidebar.Update(message)
			cmds = append(cmds, cmd)
		case focusDocuments:
			var cmd tea.Cmd
			a.documents, cmd = a.documents.Update(message)
			cmds = append(cmds, cmd)
		case focusDetail:
			var cmd tea.Cmd
			a.detail, cmd = a.detail.Update(message)
			cmds = append(cmds, cmd)
		case focusIndexes:
			var cmd tea.Cmd
			a.indexes, cmd = a.indexes.Update(message)
			cmds = append(cmds, cmd)
		}

	// ── async DB results ───────────────────────────────────────────────────────
	case msg.DatabasesLoaded:
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(message)
		cmds = append(cmds, cmd)
		if message.Err != nil {
			a.statusbar, _ = a.statusbar.Update(msg.StatusUpdate{
				Text:  fmt.Sprintf("error: %v", message.Err),
				IsErr: true,
			})
		}

	case msg.CollectionsLoaded:
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(message)
		cmds = append(cmds, cmd)

	// ── collection selected → load documents, clear detail/indexes ─────────────
	case msg.CollectionSelected:
		a.showDetail = false
		a.showIndexes = false
		a.focus = focusDocuments
		a = a.syncFocus()
		var docCmd, sbCmd, dtCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		a.statusbar, sbCmd = a.statusbar.Update(message)
		a.detail, dtCmd = a.detail.Update(message)
		a.indexes, _ = a.indexes.Update(message)
		a = a.applyLayout()
		cmds = append(cmds, docCmd, sbCmd, dtCmd)

	// ── document selected → open detail panel ──────────────────────────────────
	case msg.DocumentSelected:
		a.showIndexes = false
		if !a.showDetail {
			a.showDetail = true
			a = a.applyLayout()
		}
		a.focus = focusDetail
		a = a.syncFocus()
		var cmd tea.Cmd
		a.detail, cmd = a.detail.Update(message)
		cmds = append(cmds, cmd)

	// ── page of documents loaded ───────────────────────────────────────────────
	case msg.DocumentsLoaded:
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		a.statusbar, sbCmd = a.statusbar.Update(message)
		cmds = append(cmds, docCmd, sbCmd)

	case msg.FilterChanged:
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		a.statusbar, sbCmd = a.statusbar.Update(message)
		cmds = append(cmds, docCmd, sbCmd)

	// ── pipeline ready: editor closed, fire the DB query ──────────────────────
	case msg.PipelineReady:
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text:  "pipeline error: " + message.Err.Error(),
				IsErr: true,
			})
		} else {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: "running aggregate…",
			})
		}
		cmds = append(cmds, docCmd, sbCmd)

	// ── aggregate results returned from DB ─────────────────────────────────────
	case msg.AggregateResult:
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text:  "aggregate error: " + message.Err.Error(),
				IsErr: true,
			})
		} else {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: fmt.Sprintf("aggregate: %d results", len(message.Docs)),
			})
		}
		cmds = append(cmds, docCmd, sbCmd)

	// ── index panel results ────────────────────────────────────────────────────
	case msg.IndexesLoaded:
		var cmd tea.Cmd
		a.indexes, cmd = a.indexes.Update(message)
		if message.Err != nil {
			a.statusbar, _ = a.statusbar.Update(msg.StatusUpdate{
				Text:  "indexes error: " + message.Err.Error(),
				IsErr: true,
			})
		}
		cmds = append(cmds, cmd)

	case msg.IndexEditorDone:
		var cmd tea.Cmd
		a.indexes, cmd = a.indexes.Update(message)
		cmds = append(cmds, cmd)

	case msg.IndexCreated:
		var idxCmd, sbCmd tea.Cmd
		a.indexes, idxCmd = a.indexes.Update(message)
		text, isErr := "index created: "+message.Name, false
		if message.Err != nil {
			text, isErr = "create index failed: "+message.Err.Error(), true
		}
		a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{Text: text, IsErr: isErr})
		cmds = append(cmds, idxCmd, sbCmd)

	case msg.IndexDropped:
		var idxCmd, sbCmd tea.Cmd
		a.indexes, idxCmd = a.indexes.Update(message)
		text, isErr := "index dropped", false
		if message.Err != nil {
			text, isErr = "drop index failed: "+message.Err.Error(), true
		}
		a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{Text: text, IsErr: isErr})
		cmds = append(cmds, idxCmd, sbCmd)

	// ── CRUD results ───────────────────────────────────────────────────────────
	case msg.EditorDone:
		var cmd tea.Cmd
		a.documents, cmd = a.documents.Update(message)
		cmds = append(cmds, cmd)

	case msg.DocumentCreated:
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
			Text: fmt.Sprintf("inserted %v", message.InsertedID),
		})
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: "insert failed: " + message.Err.Error(), IsErr: true,
			})
		}
		cmds = append(cmds, docCmd, sbCmd)

	case msg.DocumentUpdated:
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		text, isErr := "document updated", false
		if message.Err != nil {
			text, isErr = "update failed: "+message.Err.Error(), true
		}
		a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{Text: text, IsErr: isErr})
		cmds = append(cmds, docCmd, sbCmd)

	case msg.DocumentDeleted:
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		text, isErr := "document deleted", false
		if message.Err != nil {
			text, isErr = "delete failed: "+message.Err.Error(), true
		}
		a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{Text: text, IsErr: isErr})
		cmds = append(cmds, docCmd, sbCmd)

	case msg.StatusUpdate:
		var sbCmd tea.Cmd
		a.statusbar, sbCmd = a.statusbar.Update(message)
		cmds = append(cmds, sbCmd)

	// ── everything else (spinner ticks, mouse, etc.) ───────────────────────────
	default:
		var sCmd, dCmd, dtCmd, idxCmd tea.Cmd
		a.sidebar, sCmd = a.sidebar.Update(message)
		a.documents, dCmd = a.documents.Update(message)
		a.detail, dtCmd = a.detail.Update(message)
		a.indexes, idxCmd = a.indexes.Update(message)
		cmds = append(cmds, sCmd, dCmd, dtCmd, idxCmd)
	}

	return &a, tea.Batch(cmds...)
}

// View composes the full-screen layout.
func (a App) View() string {
	if a.width == 0 {
		return "initialising…"
	}

	sw, dw, rightW := a.panelWidths()
	mainH := a.height - 1 // 1 row for status bar

	var row string
	switch {
	case a.showDetail:
		row = lipgloss.JoinHorizontal(lipgloss.Top,
			a.sidebar.SetSize(sw, mainH).View(),
			a.documents.SetSize(dw, mainH).View(),
			a.detail.SetSize(rightW, mainH).View(),
		)
	case a.showIndexes:
		row = lipgloss.JoinHorizontal(lipgloss.Top,
			a.sidebar.SetSize(sw, mainH).View(),
			a.documents.SetSize(dw, mainH).View(),
			a.indexes.SetSize(rightW, mainH).View(),
		)
	default:
		row = lipgloss.JoinHorizontal(lipgloss.Top,
			a.sidebar.SetSize(sw, mainH).View(),
			a.documents.SetSize(dw, mainH).View(),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		row,
		a.statusbar.SetWidth(a.width).View(),
	)
}

// ── internal helpers ───────────────────────────────────────────────────────────

func (a App) panelWidths() (sidebarW, docs, right int) {
	sw := sidebarWidth
	if a.width < 100 {
		sw = sidebarWidthS
	}
	remaining := a.width - sw
	if a.showDetail || a.showIndexes {
		dw := docsWithDetail
		if remaining < docsWithDetail+30 {
			dw = remaining / 3
		}
		return sw, dw, remaining - dw
	}
	return sw, remaining, 0
}

func (a App) syncFocus() App {
	a.sidebar = a.sidebar.SetFocused(a.focus == focusSidebar)
	a.documents = a.documents.SetFocused(a.focus == focusDocuments)
	a.detail = a.detail.SetFocused(a.focus == focusDetail)
	a.indexes = a.indexes.SetFocused(a.focus == focusIndexes)
	return a
}

func (a App) applyLayout() App {
	if a.width == 0 {
		return a
	}
	sw, dw, rightW := a.panelWidths()
	mainH := a.height - 1
	a.sidebar = a.sidebar.SetSize(sw, mainH)
	a.documents = a.documents.SetSize(dw, mainH)
	if a.showDetail {
		a.detail = a.detail.SetSize(rightW, mainH)
	}
	if a.showIndexes {
		a.indexes = a.indexes.SetSize(rightW, mainH)
	}
	a.statusbar = a.statusbar.SetWidth(a.width)
	return a
}

// currentDBCol extracts the current db and collection from documents panel.
func (a App) currentDBCol() (db, col string) {
	// documents.Collection() returns "db > col"
	s := a.documents.Collection()
	if s == "" {
		return "", ""
	}
	// find " > " separator
	for i := 0; i < len(s)-2; i++ {
		if s[i] == ' ' && s[i+1] == '>' && s[i+2] == ' ' {
			return s[:i], s[i+3:]
		}
	}
	return s, ""
}
