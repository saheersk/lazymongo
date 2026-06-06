// Package tui is the root bubbletea application model.
// It owns all panels, routes messages between them, and manages focus.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	xansi "github.com/charmbracelet/x/ansi"
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

// docsWithDetail is the documents panel width when the detail/index panel is visible.
const docsWithDetail = 38

// themeList is the ordered set of available themes shown in the picker.
var themeList = []string{
	"catppuccin",
	"high-contrast",
	"tokyo-night",
	"nord",
	"dracula",
}

// App is the root bubbletea model.
type App struct {
	width, height int
	focus         focusedPanel
	showDetail    bool
	showIndexes   bool
	showHelp      bool
	showTheme     bool // theme-picker overlay
	themeCursor   int  // cursor inside theme picker
	showDropDB    bool              // drop-database confirmation overlay
	dropDBTarget  string            // database name being confirmed
	dropInput     textinput.Model   // typed confirmation input

	sidebar   sidebar.Model
	documents documents.Model
	detail    detail.Model
	indexes   indexes.Model
	statusbar statusbar.Model

	client    *mongo.Client
	th        *style.Theme
	km        *keymap.Map
	themeName string
}

// New creates the root App with all panels initialised.
func New(client *mongo.Client, themeName string) *App {
	th := style.ByName(themeName)
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
		themeName: themeName,
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
		// help overlay — any key closes it; ? opens it from ANY mode.
		// These checks must come first so ? works even when a search/input is active.
		if a.showHelp {
			a.showHelp = false
			return &a, nil
		}
		if message.String() == "?" {
			a.showHelp = true
			return &a, nil
		}

		// theme picker overlay
		if a.showTheme {
			switch message.String() {
			case "esc", "q", "T":
				a.showTheme = false
			case "j", "down":
				a.themeCursor = (a.themeCursor + 1) % len(themeList)
			case "k", "up":
				a.themeCursor = (a.themeCursor - 1 + len(themeList)) % len(themeList)
			case "enter":
				selected := themeList[a.themeCursor]
				// Mutate the theme in place so every panel picks it up instantly.
				*a.th = *style.ByName(selected)
				a.themeName = selected
				a.showTheme = false
			}
			return &a, nil
		}
		if message.String() == "T" {
			for i, name := range themeList {
				if name == a.themeName {
					a.themeCursor = i
					break
				}
			}
			a.showTheme = true
			return &a, nil
		}

		// drop-database confirmation dialog
		if a.showDropDB {
			switch message.String() {
			case "esc":
				a.showDropDB = false
				a.dropInput.SetValue("")
			case "enter":
				typed := a.dropInput.Value()
				if typed == a.dropDBTarget {
					a.showDropDB = false
					a.dropInput.SetValue("")
					db := a.dropDBTarget
					return &a, func() tea.Msg {
						return msg.DatabaseDropped{DB: db, Err: a.client.DropDatabase(db)}
					}
				}
				// Wrong name — flash error, leave dialog open.
				var sbCmd tea.Cmd
				a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
					Text:  "name mismatch — type the exact database name",
					IsErr: true,
				})
				return &a, sbCmd
			default:
				var tiCmd tea.Cmd
				a.dropInput, tiCmd = a.dropInput.Update(message)
				return &a, tiCmd
			}
			return &a, nil
		}

		// While the sidebar search is open, route all keys directly to it.
		if a.focus == focusSidebar && a.sidebar.InSearchMode() {
			var cmd tea.Cmd
			a.sidebar, cmd = a.sidebar.Update(message)
			return &a, cmd
		}

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

		// D on a database in the sidebar → open drop confirmation overlay.
		if message.String() == "D" && a.focus == focusSidebar && !a.sidebar.InSearchMode() {
			if db := a.sidebar.ActiveDB(); db != "" {
				ti := textinput.New()
				ti.Placeholder = db
				ti.CharLimit = 128
				a.dropDBTarget = db
				a.dropInput = ti
				a.showDropDB = true
				return &a, a.dropInput.Focus()
			}
			return &a, nil
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

	case msg.DatabaseDropped:
		var sbCmd tea.Cmd
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text:  fmt.Sprintf("drop %q failed: %v", message.DB, message.Err),
				IsErr: true,
			})
			cmds = append(cmds, sbCmd)
		} else {
			var sCmd tea.Cmd
			a.sidebar, sCmd = a.sidebar.Refresh()
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: fmt.Sprintf("dropped database %q", message.DB),
			})
			cmds = append(cmds, sCmd, sbCmd)
		}

	case msg.StatusUpdate:
		var sbCmd tea.Cmd
		a.statusbar, sbCmd = a.statusbar.Update(message)
		cmds = append(cmds, sbCmd)

	// ── everything else (spinner ticks, cursor blinks, mouse, etc.) ──────────────
	default:
		var sCmd, dCmd, dtCmd, idxCmd tea.Cmd
		a.sidebar, sCmd = a.sidebar.Update(message)
		a.documents, dCmd = a.documents.Update(message)
		a.detail, dtCmd = a.detail.Update(message)
		a.indexes, idxCmd = a.indexes.Update(message)
		cmds = append(cmds, sCmd, dCmd, dtCmd, idxCmd)
		if a.showDropDB {
			var tiCmd tea.Cmd
			a.dropInput, tiCmd = a.dropInput.Update(message)
			cmds = append(cmds, tiCmd)
		}
	}

	return &a, tea.Batch(cmds...)
}

// View composes the full-screen layout.
func (a App) View() string {
	if a.width == 0 || a.height == 0 {
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

	base := lipgloss.JoinVertical(lipgloss.Left,
		row,
		a.statusbar.SetWidth(a.width).View(),
	)

	if a.showHelp {
		return renderHelp(base, a.width, a.height, a.th, a.themeName)
	}
	if a.showTheme {
		return renderThemePicker(base, a.width, a.height, a.th, a.themeName, a.themeCursor)
	}
	if a.showDropDB {
		return renderDropDB(base, a.width, a.height, a.th, a.dropDBTarget, a.dropInput.View())
	}
	return base
}

// ── internal helpers ───────────────────────────────────────────────────────────

func (a App) panelWidths() (sidebarW, docs, right int) {
	// Derive sidebar width from the longest item name currently loaded.
	sw := a.sidebar.PreferredWidth()
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

// renderHelp overlays a centred help panel on the dimmed base view (lazygit-style).
func renderHelp(base string, w, h int, th *style.Theme, themeName string) string {
	return centerOverlay(base, buildHelpBox(w, h, th, themeName), w, h)
}

// buildHelpBox returns the rendered help box string (without positioning).
func buildHelpBox(w, h int, th *style.Theme, themeName string) string {
	type section struct {
		title string
		rows  [][2]string
	}
	sections := []section{
		{"Global", [][2]string{
			{"?", "toggle this help"},
			{"T", "change color theme"},
			{"h / ←", "focus sidebar"},
			{"l / →", "focus documents"},
			{"esc", "close panel / go back"},
			{"q / ctrl+c", "quit"},
		}},
		{"Sidebar", [][2]string{
			{"j / k", "navigate"},
			{"enter", "expand db / select collection"},
			{"/", "search  (esc to close)"},
			{"db:col", "filter by db and collection"},
			{"D", "drop database (type name to confirm)"},
			{"R", "refresh list"},
		}},
		{"Documents", [][2]string{
			{"j / k", "next / previous row"},
			{"g / G", "first / last row"},
			{"ctrl+d / ctrl+u", "next / previous page"},
			{"enter", "open detail panel"},
			{"n / e / d", "new / edit / delete doc"},
			{"/", "filter  (MongoDB query JSON)"},
			{"s", "sort  (field / -field / {…})"},
			{"r", "reset filter and sort"},
			{"a", "aggregate pipeline  ($EDITOR)"},
			{"I", "toggle index panel"},
			{"y / Y", "copy _id / full JSON"},
		}},
		{"Index panel", [][2]string{
			{"n / d", "create / drop index"},
			{"esc / h", "close panel"},
		}},
		{"Detail panel", [][2]string{
			{"j / k", "scroll"},
			{"esc / h", "close"},
		}},
	}

	boxW := 72
	if w < boxW+8 {
		boxW = w - 8
	}
	if boxW < 44 {
		boxW = 44
	}

	// Title bar: full-width colored header
	themeTag := "theme: " + themeName + "  "
	titleText := "  KEY BINDINGS"
	// pad between title and theme tag
	padW := boxW - len(titleText) - len(themeTag)
	if padW < 1 {
		padW = 1
	}
	titleLine := th.HelpTitle.Width(boxW).Render(
		titleText + strings.Repeat(" ", padW) + themeTag,
	)

	var lines []string
	lines = append(lines,
		titleLine,
		th.DimText.Render("  press any key to close"),
		"",
	)
	for _, sec := range sections {
		lines = append(lines, th.HelpSection.Render("  ▸ "+sec.title))
		for _, row := range sec.rows {
			k := fmt.Sprintf("  %-22s", row[0])
			lines = append(lines, th.HelpKey.Render(k)+" "+th.HelpDesc.Render(row[1]))
		}
		lines = append(lines, "")
	}

	maxInnerH := h - 4
	if maxInnerH < 8 {
		maxInnerH = 8
	}
	if len(lines) > maxInnerH {
		lines = lines[:maxInnerH-1]
		lines = append(lines, th.DimText.Render("  … see README for full key list"))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.HelpKey.GetForeground()).
		Width(boxW).
		Render(strings.Join(lines, "\n"))
}

// themeDisplayNames are the human-readable labels shown in the picker.
var themeDisplayNames = map[string]string{
	"catppuccin":    "Catppuccin   lavender / mocha",
	"high-contrast": "High Contrast  green / amber",
	"tokyo-night":   "Tokyo Night  blue / purple",
	"nord":          "Nord         arctic blues",
	"dracula":       "Dracula      purple / pink",
}

// renderDropDB overlays the drop-database confirmation dialog.
// The user must type the exact database name before the drop is executed.
func renderDropDB(base string, w, h int, th *style.Theme, dbName, inputView string) string {
	boxW := 56
	if w < boxW+6 {
		boxW = w - 6
	}
	if boxW < 34 {
		boxW = 34
	}

	dangerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.ErrText.GetForeground()).
		Background(lipgloss.Color("#3b0000")).
		Width(boxW).
		Render("  ⚠  DROP DATABASE")

	dbLabel := th.ErrText.Render("  " + dbName)

	var rows []string
	rows = append(rows,
		dangerTitle,
		"",
		th.DimText.Render("  This will permanently delete all data in:"),
		dbLabel,
		"",
		th.DimText.Render("  Type the database name to confirm:"),
		"  "+inputView,
		"",
		th.DimText.Render("  enter drop  esc cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.ErrText.GetForeground()).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return centerOverlay(base, box, w, h)
}

// centerOverlay composites a rendered box string centred over the dimmed base.
func centerOverlay(base, box string, w, h int) string {
	boxLines := strings.Split(box, "\n")
	bh := len(boxLines)
	bw := lipgloss.Width(boxLines[0])
	startY := (h - bh) / 2
	if startY < 0 {
		startY = 0
	}
	startX := (w - bw) / 2
	if startX < 0 {
		startX = 0
	}
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	baseLines := strings.Split(base, "\n")
	for len(baseLines) < h {
		baseLines = append(baseLines, "")
	}
	out := make([]string, h)
	for y := 0; y < h; y++ {
		plain := xansi.Strip(baseLines[y])
		vw := lipgloss.Width(plain)
		if vw < w {
			plain += strings.Repeat(" ", w-vw)
		} else if vw > w {
			plain = xansi.Truncate(plain, w, "")
		}
		if y < startY || y >= startY+bh {
			out[y] = dim.Render(plain)
			continue
		}
		bi := y - startY
		left := xansi.Truncate(plain, startX, "")
		lw := lipgloss.Width(left)
		if lw < startX {
			left += strings.Repeat(" ", startX-lw)
		}
		right := xansi.TruncateLeft(plain, startX+bw, "")
		if rightW := w - startX - bw; rightW > 0 {
			if rpw := lipgloss.Width(right); rpw < rightW {
				right += strings.Repeat(" ", rightW-rpw)
			}
		} else {
			right = ""
		}
		out[y] = dim.Render(left) + boxLines[bi] + dim.Render(right)
	}
	return strings.Join(out, "\n")
}

// renderThemePicker overlays the theme chooser on the dimmed base view.
func renderThemePicker(base string, w, h int, th *style.Theme, current string, cursor int) string {
	// Target 52 chars; shrink if terminal is narrower but keep ≥ 34.
	boxW := 52
	if w < boxW+6 {
		boxW = w - 6
	}
	if boxW < 34 {
		boxW = 34
	}

	// Max chars available for the label inside a row prefix of 4 chars.
	labelMax := boxW - 4

	var rows []string
	titleLine := th.HelpTitle.Width(boxW).Render("  THEME")
	rows = append(rows, titleLine, "")

	for i, name := range themeList {
		label := themeDisplayNames[name]
		// Truncate label if it would overflow the box.
		runes := []rune(label)
		if len(runes) > labelMax {
			label = string(runes[:labelMax-1]) + "…"
		}
		if i == cursor {
			rows = append(rows, th.TableSelected.Width(boxW).Render("  ▶ "+label))
		} else if name == current {
			rows = append(rows, th.HelpSection.Render("  ● "+label))
		} else {
			rows = append(rows, th.DimText.Render("    "+label))
		}
	}

	hint := "  j/k navigate  enter apply  esc cancel"
	hintRunes := []rune(hint)
	if len(hintRunes) > boxW {
		hint = string(hintRunes[:boxW-1]) + "…"
	}
	rows = append(rows, "",
		th.DimText.Render(hint),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.HelpKey.GetForeground()).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return centerOverlay(base, box, w, h)
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
