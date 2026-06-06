package documents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/util"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Update handles all messages for the documents panel.
func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {

	// ── collection navigation ────────────────────────────────────────────────
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
		m.deleteConfirm = false
		m.err = nil
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPage(0))

	// ── page loaded ──────────────────────────────────────────────────────────
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

	// ── filter/sort applied ──────────────────────────────────────────────────
	case msg.FilterChanged:
		return m, nil

	// ── editor closed ────────────────────────────────────────────────────────
	case msg.EditorDone:
		if message.Err != nil {
			return m, statusCmd("error: " + message.Err.Error())
		}
		if message.Doc == nil {
			return m, statusCmd("no changes")
		}
		if message.IsNew {
			return m, m.insertFn(m.db, m.collection, message.Doc)
		}
		return m, m.replaceFn(m.db, m.collection, message.OrigID, message.Doc)

	// ── CRUD results ─────────────────────────────────────────────────────────
	case msg.DocumentCreated:
		if message.Err != nil {
			return m, statusCmd("insert failed: " + message.Err.Error())
		}
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.loadPage(m.page),
			statusCmd(fmt.Sprintf("inserted %v", message.InsertedID)),
		)

	case msg.DocumentUpdated:
		if message.Err != nil {
			return m, statusCmd("update failed: " + message.Err.Error())
		}
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.loadPage(m.page),
			statusCmd("document updated"),
		)

	case msg.DocumentDeleted:
		if message.Err != nil {
			return m, statusCmd("delete failed: " + message.Err.Error())
		}
		// If the cursor was at the last slot on the page and we deleted it,
		// clamp so we don't go out of range after the reload.
		if m.cursor > 0 {
			m.cursor--
		}
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.loadPage(m.page),
			statusCmd("document deleted"),
		)

	// ── keyboard ─────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		return m.handleKey(message)

	default:
		var spCmd tea.Cmd
		m.spinner, spCmd = m.spinner.Update(message)
		if m.mode != modeNone {
			var tiCmd tea.Cmd
			m.input, tiCmd = m.input.Update(message)
			return m, tea.Batch(spCmd, tiCmd)
		}
		return m, spCmd
	}
}

// ── key dispatch ─────────────────────────────────────────────────────────────

func (m Model) handleKey(key tea.KeyMsg) (Model, tea.Cmd) {
	// delete confirmation takes over all input
	if m.deleteConfirm {
		return m.handleDeleteConfirm(key)
	}
	// filter/sort bar captures all input
	if m.mode != modeNone {
		return m.handleInputKey(key)
	}

	switch {
	// ── navigation ──────────────────────────────────────────────────────────
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

	// ── filter / sort ────────────────────────────────────────────────────────
	case isKey(key, m.km.Filter):
		return m.openInput(modeFilter, m.filterExpr)

	case key.String() == "s":
		return m.openInput(modeSort, m.sortExpr)

	case key.String() == "r":
		if m.filterExpr == "" && m.sortExpr == "" {
			return m, nil
		}
		return m.applyFilterSort(nil, nil, "", "")

	// ── document open ────────────────────────────────────────────────────────
	case isKey(key, m.km.Select):
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		return m, func() tea.Msg { return msg.DocumentSelected{Doc: doc} }

	// ── CRUD ─────────────────────────────────────────────────────────────────
	case isKey(key, m.km.NewDoc):
		if m.db == "" {
			return m, nil
		}
		return m.openEditorNew()

	case isKey(key, m.km.EditDoc):
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		return m.openEditorEdit(doc)

	case isKey(key, m.km.DeleteDoc):
		if m.ActiveDoc() == nil {
			return m, nil
		}
		m.deleteConfirm = true
		return m, nil

	// ── clipboard ────────────────────────────────────────────────────────────
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

// ── delete confirmation ───────────────────────────────────────────────────────

func (m Model) handleDeleteConfirm(key tea.KeyMsg) (Model, tea.Cmd) {
	m.deleteConfirm = false
	if key.String() == "y" || key.String() == "Y" {
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		return m, m.deleteFn(m.db, m.collection, doc["_id"])
	}
	return m, nil
}

// ── filter / sort input bar ───────────────────────────────────────────────────

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
	case tea.KeyEsc:
		m.mode = modeNone
		m.inputErr = ""
	case tea.KeyCtrlU:
		m.input.SetValue("")
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(key)
		return m, cmd
	}
	return m, nil
}

func (m Model) commitFilter(expr string) (Model, tea.Cmd) {
	if expr == "" {
		return m.applyFilterSort(nil, m.sort, "", m.sortExpr)
	}
	var filter bson.M
	if err := bson.UnmarshalExtJSON([]byte(expr), false, &filter); err != nil {
		m.inputErr = "bad JSON: " + err.Error()
		return m, nil
	}
	return m.applyFilterSort(filter, m.sort, expr, m.sortExpr)
}

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

	changed := msg.FilterChanged{Filter: filter, Sort: sort, Expr: filterExpr}
	return m, tea.Batch(
		m.spinner.Tick,
		m.loadPage(0),
		func() tea.Msg { return changed },
	)
}

func (m Model) openInput(mode inputMode, prefill string) (Model, tea.Cmd) {
	m.mode = mode
	m.inputErr = ""
	m.input.SetValue(prefill)
	m.input.CursorEnd()
	m.input.Focus()
	return m, m.input.Focus()
}

// ── editor ────────────────────────────────────────────────────────────────────

const newDocTemplate = `{

}`

func (m Model) openEditorNew() (Model, tea.Cmd) {
	cmd, err := buildEditorCmd(newDocTemplate)
	if err != nil {
		return m, statusCmd("error: " + err.Error())
	}
	return m, tea.ExecProcess(cmd.cmd, func(execErr error) tea.Msg {
		defer os.Remove(cmd.path)
		if execErr != nil {
			return msg.EditorDone{Err: execErr}
		}
		doc, err := readDocFromFile(cmd.path)
		if err != nil {
			return msg.EditorDone{Err: err}
		}
		return msg.EditorDone{Doc: doc, IsNew: true}
	})
}

func (m Model) openEditorEdit(doc bson.M) (Model, tea.Cmd) {
	raw, err := util.BSONToJSON(doc)
	if err != nil {
		return m, statusCmd("marshal error: " + err.Error())
	}
	origID := doc["_id"]

	cmd, err := buildEditorCmd(raw)
	if err != nil {
		return m, statusCmd("error: " + err.Error())
	}
	return m, tea.ExecProcess(cmd.cmd, func(execErr error) tea.Msg {
		defer os.Remove(cmd.path)
		if execErr != nil {
			return msg.EditorDone{Err: execErr}
		}
		newDoc, err := readDocFromFile(cmd.path)
		if err != nil {
			return msg.EditorDone{Err: err}
		}
		return msg.EditorDone{Doc: newDoc, IsNew: false, OrigID: origID}
	})
}

type editorCmd struct {
	cmd  *exec.Cmd
	path string
}

func buildEditorCmd(content string) (editorCmd, error) {
	f, err := os.CreateTemp("", "lazymongo-*.json")
	if err != nil {
		return editorCmd{}, err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return editorCmd{}, err
	}
	f.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	// Support editors with args, e.g. "code --wait"
	parts := strings.Fields(editor)
	args := append(parts[1:], f.Name())
	cmd := exec.Command(parts[0], args...)

	return editorCmd{cmd: cmd, path: f.Name()}, nil
}

func readDocFromFile(path string) (bson.M, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil, fmt.Errorf("file is empty — no changes saved")
	}
	var doc bson.M
	if err := bson.UnmarshalExtJSON(data, false, &doc); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return doc, nil
}

// ── sort parsing ──────────────────────────────────────────────────────────────

func parseSort(expr string) (bson.D, error) {
	expr = strings.TrimSpace(expr)
	if strings.HasPrefix(expr, "{") {
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
	dir := 1
	if strings.HasPrefix(expr, "-") {
		dir = -1
		expr = expr[1:]
	}
	if expr == "" {
		return nil, fmt.Errorf("field name cannot be empty")
	}
	return bson.D{{Key: expr, Value: dir}}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func statusCmd(text string) tea.Cmd {
	return func() tea.Msg { return msg.StatusUpdate{Text: text} }
}

func isKey(km tea.KeyMsg, b interface{ Keys() []string }) bool {
	for _, k := range b.Keys() {
		if km.String() == k {
			return true
		}
	}
	return false
}

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
