package documents

import (
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/util"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Update handles all messages for the documents panel.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {

	case msg.CollectionSelected:
		m.db = message.DB
		m.collection = message.Collection
		m.page = 0
		m.cursor = 0
		m.docs = nil
		m.columns = nil
		m.filter = nil
		m.sort = nil
		m.filterExpr = ""
		m.sortExpr = ""
		m.mode = modeNone
		m.inputErr = ""
		m.err = nil
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(0))

	case msg.DocumentsLoaded:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		m.docs = message.Result.Docs
		m.total = message.Result.Total
		m.page = message.Result.Page
		m.columns = util.BuildColumns(m.docs, maxColumns(m.width))
		if m.pendingBottom {
			m.cursor = len(m.docs) - 1
			m.pendingBottom = false
		}
		return m.clamp(), nil

	case msg.FilterChanged:
		// FilterChanged sent by ourselves; nothing extra to do in the panel.
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(message)

	default:
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(message)
		// Forward to textinput when active (for blinking cursor).
		if m.mode != modeNone {
			var tiCmd tea.Cmd
			m.input, tiCmd = m.input.Update(message)
			return m, tea.Batch(spCmd, tiCmd)
		}
		return m, spCmd
	}
}

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	// While an input bar is open, capture all keystrokes.
	if m.mode != modeNone {
		return m.handleInputKey(key)
	}

	switch {
	case isKey(key, m.km.Down):
		m.cursor++
		if m.cursor >= len(m.docs) && m.page < m.pageCount()-1 {
			m.cursor = 0
			m.page++
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m.clamp(), nil

	case isKey(key, m.km.Up):
		m.cursor--
		if m.cursor < 0 && m.page > 0 {
			m.page--
			m.loading = true
			m.cursor = m.pageSize - 1
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m.clamp(), nil

	case isKey(key, m.km.Top):
		if m.page == 0 {
			m.cursor = 0
			return m, nil
		}
		m.page = 0
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(0))

	case isKey(key, m.km.Bottom):
		lastPage := m.pageCount() - 1
		if m.page == lastPage {
			m.cursor = len(m.docs) - 1
			return m.clamp(), nil
		}
		m.page = lastPage
		m.cursor = 0
		m.pendingBottom = true
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(lastPage))

	case isKey(key, m.km.PageDown):
		if m.page < m.pageCount()-1 {
			m.page++
			m.cursor = 0
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m, nil

	case isKey(key, m.km.PageUp):
		if m.page > 0 {
			m.page--
			m.cursor = 0
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
		return m, nil

	case isKey(key, m.km.Refresh):
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))

	// ── filter ──────────────────────────────────────────────────────────────

	case isKey(key, m.km.Filter): // '/'
		return m.openInput(modeFilter, m.filterExpr)

	case key.String() == "s": // sort bar
		return m.openInput(modeSort, m.sortExpr)

	case key.String() == "r": // reset filter + sort
		if m.filterExpr == "" && m.sortExpr == "" {
			return m, nil
		}
		return m.applyFilterSort(nil, nil, "", "")

	// ── document interaction ─────────────────────────────────────────────────

	case isKey(key, m.km.Select):
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			return msg.DocumentSelected{Doc: doc}
		}

	case key.String() == "y":
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		id := util.FormatValue(doc["_id"])
		_ = util.CopyToClipboard(id)
		return m, statusCmd("copied _id: " + id)

	case key.String() == "Y":
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		raw, err := util.BSONToJSON(doc)
		if err == nil {
			_ = util.CopyToClipboard(raw)
		}
		return m, statusCmd("copied document JSON to clipboard")
	}

	return m, nil
}

// handleInputKey processes keystrokes while the filter or sort bar is open.
func (m Model) handleInputKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {

	case tea.KeyEnter:
		expr := strings.TrimSpace(m.input.Value())
		switch m.mode {
		case modeFilter:
			return m.commitFilter(expr)
		case modeSort:
			return m.commitSort(expr)
		}
		return m, nil

	case tea.KeyEsc:
		m.mode = modeNone
		m.inputErr = ""
		return m, nil

	case tea.KeyCtrlU: // clear the input line
		m.input.SetValue("")
		return m, nil

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(key)
		return m, cmd
	}
}

// commitFilter parses expr as a MongoDB filter JSON and applies it.
func (m Model) commitFilter(expr string) (Model, tea.Cmd) {
	if expr == "" {
		// empty input → clear filter
		return m.applyFilterSort(nil, m.sort, "", m.sortExpr)
	}
	var filter bson.M
	if err := bson.UnmarshalExtJSON([]byte(expr), false, &filter); err != nil {
		m.inputErr = "bad JSON: " + err.Error()
		return m, nil
	}
	return m.applyFilterSort(filter, m.sort, expr, m.sortExpr)
}

// commitSort parses expr as a sort document: "field" or "-field" or
// {"field":1,"other":-1} and applies it.
func (m Model) commitSort(expr string) (Model, tea.Cmd) {
	if expr == "" {
		return m.applyFilterSort(m.filter, nil, m.filterExpr, "")
	}
	sortDoc, err := parseSort(expr)
	if err != nil {
		m.inputErr = "bad sort: " + err.Error()
		return m, nil
	}
	return m.applyFilterSort(m.filter, sortDoc, m.filterExpr, expr)
}

// applyFilterSort stores the new filter/sort, reloads page 1, and emits
// FilterChanged so the status bar can update.
func (m Model) applyFilterSort(filter bson.M, sort bson.D, filterExpr, sortExpr string) (Model, tea.Cmd) {
	m.mode = modeNone
	m.inputErr = ""
	m.filter = filter
	m.sort = sort
	m.filterExpr = filterExpr
	m.sortExpr = sortExpr
	m.page = 0
	m.cursor = 0
	m.loading = true

	filterMsg := msg.FilterChanged{
		Filter: filter,
		Sort:   sort,
		Expr:   filterExpr,
	}
	return m, tea.Batch(
		m.spinner.Tick,
		m.loadPage(0),
		func() tea.Msg { return filterMsg },
	)
}

// openInput switches to the given input mode, pre-filling with current value.
func (m Model) openInput(mode inputMode, prefill string) (Model, tea.Cmd) {
	m.mode = mode
	m.inputErr = ""
	m.input.SetValue(prefill)
	m.input.CursorEnd()
	m.input.Focus()
	return m, m.input.Focus()
}

// parseSort converts user sort text to bson.D.
//   - "field"   → {field: 1}
//   - "-field"  → {field: -1}
//   - {"a":1}   → parsed as bson.D directly
func parseSort(expr string) (bson.D, error) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "{") {
		// JSON object
		var raw map[string]int
		if err := json.Unmarshal([]byte(expr), &raw); err != nil {
			return nil, err
		}
		var d bson.D
		for k, v := range raw {
			d = append(d, bson.E{Key: k, Value: v})
		}
		return d, nil
	}
	// simple "field" or "-field"
	dir := 1
	if strings.HasPrefix(expr, "-") {
		dir = -1
		expr = expr[1:]
	}
	return bson.D{{Key: expr, Value: dir}}, nil
}

func statusCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return msg.StatusUpdate{Text: text}
	}
}

func isKey(km tea.KeyMsg, b interface{ Keys() []string }) bool {
	for _, k := range b.Keys() {
		if km.String() == k {
			return true
		}
	}
	return false
}

// maxColumns decides how many document fields to show based on panel width.
func maxColumns(width int) int {
	switch {
	case width < 80:
		return 2
	case width < 120:
		return 4
	default:
		return 5
	}
}
