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
	"github.com/saheersk/lazymongo/internal/tui/panels/documents"
	"github.com/saheersk/lazymongo/internal/tui/panels/sidebar"
	"github.com/saheersk/lazymongo/internal/tui/panels/statusbar"
	"github.com/saheersk/lazymongo/internal/tui/style"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type focusedPanel int

const (
	focusSidebar   focusedPanel = iota
	focusDocuments              // Phase 2 will add focusDetail
)

// App is the root bubbletea model.
type App struct {
	width, height int
	focus         focusedPanel

	sidebar   sidebar.Model
	documents documents.Model
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

	return &App{
		client:    client,
		th:        th,
		km:        km,
		focus:     focusSidebar,
		sidebar:   sidebar.New(th, km, fetchDBs, fetchCols),
		documents: documents.New(th, km, fetchPage),
		statusbar: statusbar.New(th, client.URI()),
	}
}

// Init fires startup commands for all panels.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.sidebar.Init(),
		a.documents.Init(),
		a.statusbar.Init(),
	)
}

// Update is the central message dispatcher.
func (a App) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch message := message.(type) {

	case tea.WindowSizeMsg:
		a.width = message.Width
		a.height = message.Height
		a = a.resizePanels()
		return &a, nil

	case tea.KeyMsg:
		// global keys always win
		if message.String() == "q" || message.String() == "ctrl+c" {
			return &a, tea.Quit
		}
		// focus switching
		if message.String() == "h" || message.String() == "left" {
			a.focus = focusSidebar
			a = a.syncFocus()
			return &a, nil
		}
		if message.String() == "l" || message.String() == "right" {
			if a.documents.Collection() != "" {
				a.focus = focusDocuments
				a = a.syncFocus()
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
		}

	// ---- async results: route to relevant panels ----

	case msg.DatabasesLoaded:
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(message)
		cmds = append(cmds, cmd)
		if message.Err != nil {
			var sbCmd tea.Cmd
			a.statusbar, sbCmd = a.statusbar.Update(msg.StatusUpdate{
				Text: fmt.Sprintf("error: %v", message.Err), IsErr: true,
			})
			cmds = append(cmds, sbCmd)
		}

	case msg.CollectionsLoaded:
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(message)
		cmds = append(cmds, cmd)

	case msg.CollectionSelected:
		// move focus to documents panel and kick off load
		a.focus = focusDocuments
		a = a.syncFocus()
		var docCmd, sbCmd tea.Cmd
		a.documents, docCmd = a.documents.Update(message)
		a.statusbar, sbCmd = a.statusbar.Update(message)
		cmds = append(cmds, docCmd, sbCmd)

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

	case msg.StatusUpdate:
		var sbCmd tea.Cmd
		a.statusbar, sbCmd = a.statusbar.Update(message)
		cmds = append(cmds, sbCmd)

	default:
		// spinner ticks etc. — forward to all panels
		var sCmd, dCmd tea.Cmd
		a.sidebar, sCmd = a.sidebar.Update(message)
		a.documents, dCmd = a.documents.Update(message)
		cmds = append(cmds, sCmd, dCmd)
	}

	return &a, tea.Batch(cmds...)
}

// View composes the full-screen layout.
func (a App) View() string {
	if a.width == 0 {
		return "loading…"
	}

	sidebarW := 28
	if a.width < 80 {
		sidebarW = 20
	}
	docsW := a.width - sidebarW

	mainH := a.height - 1 // leave 1 row for status bar

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		a.sidebar.SetSize(sidebarW, mainH).View(),
		a.documents.SetSize(docsW, mainH).View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		row,
		a.statusbar.SetWidth(a.width).View(),
	)
}

// ---- helpers ----

func (a App) syncFocus() App {
	a.sidebar = a.sidebar.SetFocused(a.focus == focusSidebar)
	a.documents = a.documents.SetFocused(a.focus == focusDocuments)
	return a
}

func (a App) resizePanels() App {
	sidebarW := 28
	if a.width < 80 {
		sidebarW = 20
	}
	docsW := a.width - sidebarW
	mainH := a.height - 1

	a.sidebar = a.sidebar.SetSize(sidebarW, mainH)
	a.documents = a.documents.SetSize(docsW, mainH)
	a.statusbar = a.statusbar.SetWidth(a.width)
	return a
}

// shortURI trims long URIs for display.
func shortURI(uri string) string {
	const max = 40
	if len(uri) <= max {
		return uri
	}
	return uri[:max-3] + "…"
}
