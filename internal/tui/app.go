// Package tui is the root bubbletea application model.
// It owns all panels, routes messages between them, and manages focus.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	"github.com/saheersk/lazymongo/internal/util"
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
	prevFocus     focusedPanel // focus before opening detail/index panels
	showDetail    bool
	showIndexes   bool
	showHelp      bool
	showTheme     bool // theme-picker overlay
	themeCursor   int  // cursor inside theme picker
	showDropDB    bool            // drop-database confirmation overlay
	dropDBTarget  string          // database name being confirmed
	dropInput     textinput.Model // typed confirmation input

	// Collection management overlays
	showCreateCol  bool
	createColDB    string
	createColInput textinput.Model

	showCreateDB  bool
	createDBInput textinput.Model

	showDropCol   bool
	dropColDB     string
	dropColTarget string
	dropColInput  textinput.Model

	showRenameCol  bool
	renameColDB    string
	renameColOld   string
	renameColInput textinput.Model

	showColStats    bool
	colStatsDB      string
	colStatsCol     string
	colStatsData    *msg.CollectionStatsDetail
	colStatsLoading bool
	colStatsErr     error

	showExportPicker   bool
	exportPickerCursor int // 0 = JSON, 1 = CSV
	exportField        int // 0 = format selector, 1 = limit input, 2 = dir input
	exportLimitInput   textinput.Model
	exportDirInput     textinput.Model
	exportDocsFn       func(db, col string, filter bson.M, sort bson.D, format string, limit int, outDir string) tea.Cmd

	sidebar   sidebar.Model
	documents documents.Model
	detail    detail.Model
	indexes   indexes.Model
	statusbar statusbar.Model

	// Collection management callbacks
	createColFn func(db, col string) tea.Cmd
	dropColFn   func(db, col string) tea.Cmd
	renameColFn func(db, oldCol, newCol string) tea.Cmd
	loadStatsFn func(db, col string) tea.Cmd

	client    *mongo.Client
	th        *style.Theme
	km        *keymap.Map
	themeName string
}

// New creates the root App with all panels initialised.
func New(client *mongo.Client, themeName, editor string, keybindOverrides map[string]string) *App {
	th := style.ByName(themeName)
	km := keymap.Default()
	if len(keybindOverrides) > 0 {
		km.ApplyOverrides(keybindOverrides)
	}

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

	exportDocs := func(db, col string, filter bson.M, sort bson.D, format string, limit int, outDir string) tea.Cmd {
		return func() tea.Msg {
			docs, err := client.ExportDocs(db, col, filter, sort, limit) // limit=0 → all docs
			if err != nil {
				return msg.ExportDone{Err: err}
			}

			dir := resolveExportDir(outDir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return msg.ExportDone{Err: fmt.Errorf("cannot create dir %q: %w", dir, err)}
			}

			stamp := time.Now().Format("20060102-150405")
			var data []byte
			var path string
			if format == "csv" {
				cols := util.BuildColumns(docs, 20)
				data, err = util.ToCSV(docs, cols)
				path = filepath.Join(dir, col+"-"+stamp+".csv")
			} else {
				data, err = util.ToJSON(docs)
				path = filepath.Join(dir, col+"-"+stamp+".json")
			}
			if err != nil {
				return msg.ExportDone{Err: err}
			}
			if err := os.WriteFile(path, data, 0o644); err != nil {
				return msg.ExportDone{Err: err}
			}
			return msg.ExportDone{Path: path, Count: len(docs)}
		}
	}

	createColFn := func(db, col string) tea.Cmd {
		return func() tea.Msg {
			return msg.CollectionCreated{DB: db, Col: col, Err: client.CreateCollection(db, col)}
		}
	}

	dropColFn := func(db, col string) tea.Cmd {
		return func() tea.Msg {
			return msg.CollectionDropped{DB: db, Col: col, Err: client.DropCollection(db, col)}
		}
	}

	renameColFn := func(db, oldCol, newCol string) tea.Cmd {
		return func() tea.Msg {
			return msg.CollectionRenamed{DB: db, OldCol: oldCol, NewCol: newCol, Err: client.RenameCollection(db, oldCol, newCol)}
		}
	}

	loadStatsFn := func(db, col string) tea.Cmd {
		return func() tea.Msg {
			stats, err := client.CollectionStats(db, col)
			return msg.CollectionStatsLoaded{DB: db, Col: col, Stats: stats, Err: err}
		}
	}

	return &App{
		client:       client,
		th:           th,
		km:           km,
		themeName:    themeName,
		focus:        focusSidebar,
		sidebar:  sidebar.New(th, km, fetchDBs, fetchCols),
		documents: documents.New(th, km, fetchPage, insertDoc, replaceDoc, deleteDoc, aggregateDocs,
			func(db, col string, filter bson.M, sort bson.D, format string) tea.Cmd {
				return exportDocs(db, col, filter, sort, format, 0, "")
			}).SetEditor(editor),
		detail:       detail.New(th, km),
		indexes:      indexes.New(th, km, fetchIndexes, createIndex, dropIndex).SetEditor(editor),
		statusbar:    statusbar.New(th, client.URI()),
		createColFn:  createColFn,
		dropColFn:    dropColFn,
		renameColFn:  renameColFn,
		loadStatsFn:  loadStatsFn,
		exportDocsFn: exportDocs,
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
				// Wrong name — give a specific hint for near-misses.
				errText := fmt.Sprintf("type exactly: %s", a.dropDBTarget)
				if strings.EqualFold(typed, a.dropDBTarget) {
					errText = fmt.Sprintf("name is case-sensitive — type exactly: %s", a.dropDBTarget)
				}
				var sbCmd tea.Cmd
				a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{Text: errText, IsErr: true})
				return &a, sbCmd
			default:
				var tiCmd tea.Cmd
				a.dropInput, tiCmd = a.dropInput.Update(message)
				return &a, tiCmd
			}
			return &a, nil
		}

		// Collection management overlays (check before sidebar search)
		if a.showColStats {
			// Any key closes the stats overlay
			a.showColStats = false
			return &a, nil
		}

		if a.showCreateCol {
			switch message.String() {
			case "esc":
				a.showCreateCol = false
				a.createColInput.SetValue("")
			case "enter":
				col := strings.TrimSpace(a.createColInput.Value())
				db := a.createColDB
				a.showCreateCol = false
				a.createColInput.SetValue("")
				if col != "" {
					return &a, a.createColFn(db, col)
				}
			default:
				var tiCmd tea.Cmd
				a.createColInput, tiCmd = a.createColInput.Update(message)
				return &a, tiCmd
			}
			return &a, nil
		}

		if a.showCreateDB {
			switch message.String() {
			case "esc":
				a.showCreateDB = false
				a.createDBInput.SetValue("")
			case "enter":
				raw := strings.TrimSpace(a.createDBInput.Value())
				a.showCreateDB = false
				a.createDBInput.SetValue("")
				if raw != "" {
					var db, col string
					if idx := strings.Index(raw, "/"); idx >= 0 {
						db = strings.TrimSpace(raw[:idx])
						col = strings.TrimSpace(raw[idx+1:])
					} else {
						db = raw
						col = "default"
					}
					if db != "" && col != "" {
						return &a, a.createColFn(db, col)
					}
				}
			default:
				var tiCmd tea.Cmd
				a.createDBInput, tiCmd = a.createDBInput.Update(message)
				return &a, tiCmd
			}
			return &a, nil
		}

		if a.showDropCol {
			switch message.String() {
			case "esc":
				a.showDropCol = false
				a.dropColInput.SetValue("")
			case "enter":
				typed := a.dropColInput.Value()
				if typed == a.dropColTarget {
					db, col := a.dropColDB, a.dropColTarget
					a.showDropCol = false
					a.dropColInput.SetValue("")
					return &a, a.dropColFn(db, col)
				}
				var sbCmd tea.Cmd
				a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
					Text:  "name mismatch — type the exact collection name",
					IsErr: true,
				})
				return &a, sbCmd
			default:
				var tiCmd tea.Cmd
				a.dropColInput, tiCmd = a.dropColInput.Update(message)
				return &a, tiCmd
			}
			return &a, nil
		}

		if a.showRenameCol {
			switch message.String() {
			case "esc":
				a.showRenameCol = false
				a.renameColInput.SetValue("")
			case "enter":
				newName := strings.TrimSpace(a.renameColInput.Value())
				db, old := a.renameColDB, a.renameColOld
				a.showRenameCol = false
				a.renameColInput.SetValue("")
				if newName != "" && newName != old {
					return &a, a.renameColFn(db, old, newName)
				}
			default:
				var tiCmd tea.Cmd
				a.renameColInput, tiCmd = a.renameColInput.Update(message)
				return &a, tiCmd
			}
			return &a, nil
		}

		// Export picker — exportField: 0=format selector, 1=limit input, 2=dir input.
		// Use message.Type for Tab so it works in all terminals.
		if a.showExportPicker {
			isTab := message.Type == tea.KeyTab
			isEsc := message.Type == tea.KeyEsc || message.String() == "esc"
			isEnter := message.Type == tea.KeyEnter

			switch a.exportField {
			case 1: // limit input active
				if isEsc {
					a.showExportPicker = false
					a.exportField = 0
					return &a, nil
				}
				if isTab {
					a.exportLimitInput.Blur()
					a.exportField = 2
					return &a, a.exportDirInput.Focus()
				}
				if isEnter {
					a.exportLimitInput.Blur()
					a.exportField = 0
					return &a, nil
				}
				var limCmd tea.Cmd
				a.exportLimitInput, limCmd = a.exportLimitInput.Update(message)
				return &a, limCmd

			case 2: // dir input active
				if isEsc {
					a.showExportPicker = false
					a.exportField = 0
					return &a, nil
				}
				if isTab {
					a.exportDirInput.Blur()
					a.exportField = 0 // cycle back to format selector
					return &a, nil
				}
				if isEnter {
					a.exportDirInput.Blur()
					a.exportField = 0
					return &a, nil
				}
				var dirCmd tea.Cmd
				a.exportDirInput, dirCmd = a.exportDirInput.Update(message)
				return &a, dirCmd

			default: // 0 — format selector
				if isEsc || message.String() == "q" {
					a.showExportPicker = false
					return &a, nil
				}
				if isTab {
					a.exportField = 1
					return &a, a.exportLimitInput.Focus()
				}
				if isEnter {
					a.showExportPicker = false
					format := "json"
					if a.exportPickerCursor == 1 {
						format = "csv"
					}
					limit, _ := strconv.Atoi(strings.TrimSpace(a.exportLimitInput.Value()))
					if limit < 0 {
						limit = 0
					}
					outDir := strings.TrimSpace(a.exportDirInput.Value())
					return &a, a.exportDocsFn(
						a.documents.DB(), a.documents.ColName(),
						a.documents.Filter(), a.documents.SortDoc(),
						format, limit, outDir,
					)
				}
				switch message.String() {
				case "j", "down":
					a.exportPickerCursor = (a.exportPickerCursor + 1) % 2
				case "k", "up":
					a.exportPickerCursor = (a.exportPickerCursor - 1 + 2) % 2
				}
				return &a, nil
			}
		}

		// While the sidebar search is open, route all keys directly to it.
		// Exception: if Enter is pressed with no results, close search and open
		// a create-collection dialog pre-filled with the search query.
		if a.focus == focusSidebar && a.sidebar.InSearchMode() {
			if message.String() == "enter" {
				query := strings.TrimSpace(a.sidebar.SearchValue())
				if query != "" && a.sidebar.VisibleCount() == 0 {
					db := a.sidebar.SearchDB()
					a.sidebar, _ = a.sidebar.Update(tea.KeyMsg{Type: tea.KeyEsc}) // close search
					if db != "" {
						ti := textinput.New()
						ti.Placeholder = "collection name"
						ti.CharLimit = 128
						ti.SetValue(query)
						a.createColDB = db
						a.createColInput = ti
						a.showCreateCol = true
						return &a, a.createColInput.Focus()
					}
					return &a, nil
				}
			}
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
				a.focus = a.prevFocus
				if a.focus == focusDetail {
					a.focus = focusDocuments
				}
				a = a.syncFocus()
				a = a.applyLayout()
				return &a, nil
			case focusIndexes:
				a.showIndexes = false
				a.focus = a.prevFocus
				if a.focus == focusIndexes {
					a.focus = focusDocuments
				}
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

		// 'a' opens the aggregate pipeline editor when a collection is loaded.
		if message.String() == "a" {
			if a.documents.Collection() != "" {
				var cmd tea.Cmd
				a.documents, cmd = a.documents.Update(message)
				a.focus = focusDocuments
				a = a.syncFocus()
				cmds = append(cmds, cmd)
				return &a, tea.Batch(cmds...)
			}
			var sbCmd tea.Cmd
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{Text: "select a collection first to run an aggregate pipeline"})
			return &a, sbCmd
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
				a.prevFocus = a.focus
				a.focus = focusIndexes
				a = a.syncFocus()
				a = a.applyLayout()
			}
			return &a, tea.Batch(cmds...)
		}

		// Collection management keys (sidebar focus, not in search mode)
		if a.focus == focusSidebar && !a.sidebar.InSearchMode() {
			switch message.String() {
			case "n":
				// Create new collection in the active DB (works on both DB and collection rows)
				if db := a.sidebar.ActiveDB(); db != "" {
					ti := textinput.New()
					ti.Placeholder = "collection name"
					ti.CharLimit = 128
					a.createColDB = db
					a.createColInput = ti
					a.showCreateCol = true
					return &a, a.createColInput.Focus()
				}

			case "N":
				// Create new database (format: db/collection)
				ti := textinput.New()
				ti.Placeholder = "db/collection"
				ti.CharLimit = 256
				a.createDBInput = ti
				a.showCreateDB = true
				return &a, a.createDBInput.Focus()

			case "r":
				// Rename collection (only when cursor is on a collection)
				if a.sidebar.CursorIsCollection() {
					db := a.sidebar.ActiveDB()
					col := a.sidebar.ActiveCollection()
					if db != "" && col != "" {
						ti := textinput.New()
						ti.Placeholder = col
						ti.SetValue(col)
						ti.CharLimit = 128
						a.renameColDB = db
						a.renameColOld = col
						a.renameColInput = ti
						a.showRenameCol = true
						return &a, a.renameColInput.Focus()
					}
				}

			case "s":
				// Collection stats (only when cursor is on a collection)
				if a.sidebar.CursorIsCollection() {
					db := a.sidebar.ActiveDB()
					col := a.sidebar.ActiveCollection()
					if db != "" && col != "" {
						a.colStatsDB = db
						a.colStatsCol = col
						a.colStatsData = nil
						a.colStatsLoading = true
						a.colStatsErr = nil
						a.showColStats = true
						return &a, a.loadStatsFn(db, col)
					}
				}

			case "D":
				if a.sidebar.CursorIsCollection() {
					// Drop collection
					db := a.sidebar.ActiveDB()
					col := a.sidebar.ActiveCollection()
					if db != "" && col != "" {
						ti := textinput.New()
						ti.Placeholder = col
						ti.CharLimit = 128
						a.dropColDB = db
						a.dropColTarget = col
						a.dropColInput = ti
						a.showDropCol = true
						return &a, a.dropColInput.Focus()
					}
				} else {
					// Drop database (existing flow)
					if db := a.sidebar.ActiveDB(); db != "" {
						ti := textinput.New()
						ti.Placeholder = db
						ti.CharLimit = 128
						a.dropDBTarget = db
						a.dropInput = ti
						a.showDropDB = true
						return &a, a.dropInput.Focus()
					}
				}
				return &a, nil
			}
		}

		// D on a database in the sidebar → open drop confirmation overlay (legacy path, now handled above).
		if message.String() == "D" && a.focus == focusSidebar && !a.sidebar.InSearchMode() {
			// Already handled above; this block remains as a no-op fallthrough.
			return &a, nil
		}

		// x/X when documents focused → show export format picker
		if a.focus == focusDocuments && !a.documents.InInputMode() {
			if message.String() == "x" || message.String() == "X" {
				if a.documents.DB() != "" {
					dl := downloadsDir()

					lim := textinput.New()
					lim.Placeholder = "0 = all docs"
					lim.CharLimit = 12
					lim.SetValue("0")

					dir := textinput.New()
					dir.Placeholder = dl
					dir.CharLimit = 256
					dir.SetValue(dl)

					a.exportLimitInput = lim
					a.exportDirInput = dir
					a.exportField = 0
					a.exportPickerCursor = 0
					if message.String() == "X" {
						a.exportPickerCursor = 1
					}
					a.showExportPicker = true
					return &a, nil
				}
			}
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
		a.prevFocus = a.focus
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

	case msg.CollectionCreated:
		var sbCmd tea.Cmd
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text:  fmt.Sprintf("create collection failed: %v", message.Err),
				IsErr: true,
			})
		} else {
			var sCmd tea.Cmd
			a.sidebar, sCmd = a.sidebar.Refresh()
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: fmt.Sprintf("collection %s.%s created", message.DB, message.Col),
			})
			cmds = append(cmds, sCmd)
		}
		cmds = append(cmds, sbCmd)

	case msg.CollectionDropped:
		var sbCmd tea.Cmd
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text:  fmt.Sprintf("drop collection failed: %v", message.Err),
				IsErr: true,
			})
		} else {
			var sCmd tea.Cmd
			a.sidebar, sCmd = a.sidebar.Refresh()
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: fmt.Sprintf("collection %s.%s dropped", message.DB, message.Col),
			})
			cmds = append(cmds, sCmd)
			// If documents panel is showing this collection, clear it
			db, col := a.currentDBCol()
			if db == message.DB && col == message.Col {
				var docCmd tea.Cmd
				a.documents, docCmd = a.documents.Update(msg.CollectionSelected{DB: "", Collection: ""})
				cmds = append(cmds, docCmd)
			}
		}
		cmds = append(cmds, sbCmd)

	case msg.CollectionRenamed:
		var sbCmd tea.Cmd
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text:  fmt.Sprintf("rename collection failed: %v", message.Err),
				IsErr: true,
			})
		} else {
			var sCmd tea.Cmd
			a.sidebar, sCmd = a.sidebar.Refresh()
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: fmt.Sprintf("renamed %s.%s → %s", message.DB, message.OldCol, message.NewCol),
			})
			cmds = append(cmds, sCmd)
		}
		cmds = append(cmds, sbCmd)

	case msg.CollectionStatsLoaded:
		a.colStatsLoading = false
		if message.Err != nil {
			a.colStatsErr = message.Err
		} else {
			stats := message.Stats
			a.colStatsData = &stats
		}

	case msg.ExportDone:
		var sbCmd tea.Cmd
		if message.Err != nil {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text:  "export failed: " + message.Err.Error(),
				IsErr: true,
			})
		} else {
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: fmt.Sprintf("exported %d docs → %s", message.Count, message.Path),
			})
		}
		cmds = append(cmds, sbCmd)

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
		if a.showCreateCol {
			var tiCmd tea.Cmd
			a.createColInput, tiCmd = a.createColInput.Update(message)
			cmds = append(cmds, tiCmd)
		}
		if a.showCreateDB {
			var tiCmd tea.Cmd
			a.createDBInput, tiCmd = a.createDBInput.Update(message)
			cmds = append(cmds, tiCmd)
		}
		if a.showDropCol {
			var tiCmd tea.Cmd
			a.dropColInput, tiCmd = a.dropColInput.Update(message)
			cmds = append(cmds, tiCmd)
		}
		if a.showRenameCol {
			var tiCmd tea.Cmd
			a.renameColInput, tiCmd = a.renameColInput.Update(message)
			cmds = append(cmds, tiCmd)
		}
		if a.showExportPicker && a.exportField == 1 {
			var tiCmd tea.Cmd
			a.exportLimitInput, tiCmd = a.exportLimitInput.Update(message)
			cmds = append(cmds, tiCmd)
		}
		if a.showExportPicker && a.exportField == 2 {
			var tiCmd tea.Cmd
			a.exportDirInput, tiCmd = a.exportDirInput.Update(message)
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

	if a.showExportPicker {
		return renderExportPicker(base, a.width, a.height, a.th,
			a.documents.DB(), a.documents.ColName(), a.documents.FilterExpr(),
			a.exportPickerCursor, a.exportField,
			a.exportLimitInput.View(),
			a.exportDirInput.View(), a.exportDirInput.Value())
	}
	if a.showColStats {
		return renderColStats(base, a.width, a.height, a.th, a.colStatsDB, a.colStatsCol, a.colStatsLoading, a.colStatsData, a.colStatsErr)
	}
	if a.showCreateCol {
		return renderCreateCol(base, a.width, a.height, a.th, a.createColDB, a.createColInput.View())
	}
	if a.showCreateDB {
		return renderCreateDB(base, a.width, a.height, a.th, a.createDBInput.View())
	}
	if a.showDropCol {
		return renderDropCol(base, a.width, a.height, a.th, a.dropColDB, a.dropColTarget, a.dropColInput.View())
	}
	if a.showRenameCol {
		return renderRenameCol(base, a.width, a.height, a.th, a.renameColDB, a.renameColOld, a.renameColInput.View())
	}
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
			{"n", "create collection (in current db)"},
			{"N", "create database  (format: db/collection)"},
			{"r", "rename collection"},
			{"s", "collection stats"},
			{"D", "drop db or collection (type name to confirm)"},
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
			{"x / X", "export  (format picker → JSON or CSV, limit 0 = unlimited)"},
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

// renderCreateCol overlays the create-collection dialog.
func renderCreateCol(base string, w, h int, th *style.Theme, db, inputView string) string {
	boxW := 44
	if w < boxW+6 {
		boxW = w - 6
	}
	if boxW < 30 {
		boxW = 30
	}

	titleLine := th.PanelTitle.Width(boxW).Render("  NEW COLLECTION IN " + db)

	var rows []string
	rows = append(rows,
		titleLine,
		"",
		th.DimText.Render("  Collection name:"),
		"  "+inputView,
		"",
		th.DimText.Render("  enter create  esc cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.HelpKey.GetForeground()).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return centerOverlay(base, box, w, h)
}

// renderCreateDB overlays the create-database dialog.
func renderCreateDB(base string, w, h int, th *style.Theme, inputView string) string {
	boxW := 44
	if w < boxW+6 {
		boxW = w - 6
	}
	if boxW < 30 {
		boxW = 30
	}

	titleLine := th.PanelTitle.Width(boxW).Render("  NEW DATABASE  (format: db/collection)")

	var rows []string
	rows = append(rows,
		titleLine,
		"",
		th.DimText.Render("  Enter db/collection (e.g. mydb/users):"),
		"  "+inputView,
		"",
		th.DimText.Render("  enter create  esc cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.HelpKey.GetForeground()).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return centerOverlay(base, box, w, h)
}

// renderDropCol overlays the drop-collection confirmation dialog.
func renderDropCol(base string, w, h int, th *style.Theme, db, col, inputView string) string {
	boxW := 54
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
		Render("  ⚠  DROP COLLECTION")

	dbColLabel := th.ErrText.Render("  " + db + "." + col)

	var rows []string
	rows = append(rows,
		dangerTitle,
		"",
		th.DimText.Render("  This will permanently delete all data in:"),
		dbColLabel,
		"",
		th.DimText.Render("  Type the collection name to confirm:"),
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

// renderRenameCol overlays the rename-collection dialog.
func renderRenameCol(base string, w, h int, th *style.Theme, db, col, inputView string) string {
	boxW := 54
	if w < boxW+6 {
		boxW = w - 6
	}
	if boxW < 34 {
		boxW = 34
	}

	amberTitle := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("#3d2a00")).
		Foreground(lipgloss.Color("#fbbf24")).
		Width(boxW).
		Render("  RENAME COLLECTION")

	var rows []string
	rows = append(rows,
		amberTitle,
		"",
		th.DimText.Render(fmt.Sprintf("  Renaming: %s.%s", db, col)),
		"",
		th.DimText.Render("  New collection name:"),
		"  "+inputView,
		"",
		th.DimText.Render("  enter rename  esc cancel"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#fbbf24")).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return centerOverlay(base, box, w, h)
}

// renderColStats overlays the collection stats panel.
func renderColStats(base string, w, h int, th *style.Theme, db, col string, loading bool, data *msg.CollectionStatsDetail, err error) string {
	boxW := 50
	if w < boxW+6 {
		boxW = w - 6
	}
	if boxW < 34 {
		boxW = 34
	}

	titleLine := th.PanelTitle.Width(boxW).Render(fmt.Sprintf("  STATS  %s.%s", db, col))

	var rows []string
	rows = append(rows, titleLine, "")

	if loading {
		rows = append(rows, th.DimText.Render("  loading…"))
	} else if err != nil {
		rows = append(rows, th.ErrText.Render("  error: "+err.Error()))
	} else if data != nil {
		kw := 20
		stat := func(label string, value string) string {
			l := fmt.Sprintf("  %-*s", kw, label)
			return th.HelpKey.Render(l) + "  " + th.HelpDesc.Render(value)
		}
		rows = append(rows,
			stat("Documents:", fmt.Sprintf("%d", data.DocCount)),
			stat("Avg doc size:", fmt.Sprintf("%.0f B", data.AvgDocSize)),
			stat("Total size:", formatBytes(data.TotalSize)),
			stat("Storage size:", formatBytes(data.StorageSize)),
			stat("Indexes:", fmt.Sprintf("%d", data.IndexCount)),
			stat("Index size:", formatBytes(data.IndexSize)),
		)
	} else {
		rows = append(rows, th.DimText.Render("  no data available"))
	}

	rows = append(rows, "", th.DimText.Render("  press any key to close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.HelpKey.GetForeground()).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return centerOverlay(base, box, w, h)
}

// renderExportPicker overlays a format-selection, limit, and directory dialog for export.
func renderExportPicker(base string, w, h int, th *style.Theme,
	db, col, filterExpr string, cursor, exportField int,
	limitView string,
	dirView, dirRaw string,
) string {
	boxW := 58
	if w < boxW+6 {
		boxW = w - 6
	}
	if boxW < 38 {
		boxW = 38
	}

	titleLine := th.PanelTitle.Width(boxW).Render(fmt.Sprintf("  EXPORT  %s.%s", db, col))

	var filterLine string
	if filterExpr != "" {
		short := filterExpr
		if len([]rune(short)) > 34 {
			short = string([]rune(short)[:33]) + "…"
		}
		filterLine = th.StatusFilter.Render("  filter: " + short)
	} else {
		filterLine = th.DimText.Render("  filter: none")
	}

	type fmtOpt struct{ name, desc, ext string }
	opts := []fmtOpt{
		{"JSON", "pretty-printed Extended JSON", "json"},
		{"CSV", "comma-separated values", "csv"},
	}

	var fmtRows []string
	for i, f := range opts {
		label := fmt.Sprintf("%-5s  %s", f.name, f.desc)
		if i == cursor {
			fmtRows = append(fmtRows, th.TableSelected.Width(boxW).Render("  ▶ "+label))
		} else {
			fmtRows = append(fmtRows, th.DimText.Render("    "+label))
		}
	}

	var limitLabel string
	if exportField == 1 {
		limitLabel = th.StatusFilter.Render("  Limit  › ") + limitView
	} else {
		limitLabel = th.DimText.Render("  Limit:   ") + limitView + th.DimText.Render("  (0 = unlimited)")
	}

	var dirLabel string
	if exportField == 2 {
		dirLabel = th.StatusFilter.Render("  Dir    › ") + dirView
	} else {
		dirLabel = th.DimText.Render("  Dir:     ") + dirView
	}

	// Live preview of the full output path, truncated in the middle so both
	// the directory and filename are always visible.
	outDir := resolveExportDir(dirRaw)
	ext := opts[cursor].ext
	preview := filepath.Join(outDir, col+"-[date]."+ext)
	maxPreview := boxW - 4
	if runes := []rune(preview); len(runes) > maxPreview {
		half := maxPreview/2 - 1
		preview = string(runes[:half]) + "…" + string(runes[len(runes)-half:])
	}
	previewLine := th.DimText.Render("  → " + preview)

	hint := "  j/k format  tab switch  enter export  esc cancel"

	var rows []string
	rows = append(rows,
		titleLine,
		"",
		filterLine,
		"",
		th.DimText.Render("  Format:"),
	)
	rows = append(rows, fmtRows...)
	rows = append(rows,
		"",
		limitLabel,
		dirLabel,
		"",
		previewLine,
		"",
		th.DimText.Render(hint),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.HelpKey.GetForeground()).
		Width(boxW).
		Render(strings.Join(rows, "\n"))

	return centerOverlay(base, box, w, h)
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// downloadsDir returns ~/Downloads on every OS, falling back to home if it
// doesn't exist (e.g. a headless Linux server with no Downloads folder).
func downloadsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dl := filepath.Join(home, "Downloads")
	if info, err := os.Stat(dl); err == nil && info.IsDir() {
		return dl
	}
	return home
}

// resolveExportDir expands ~ and returns an absolute directory path.
// Falls back to Downloads (or home) if dir is empty.
func resolveExportDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "~" {
		return downloadsDir()
	}
	if strings.HasPrefix(dir, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, dir[2:])
	}
	return dir
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
