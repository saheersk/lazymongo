package documents

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/util"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// unquotedKeyRe matches unquoted object keys: {name: or , age: → {"name": or , "age":
var unquotedKeyRe = regexp.MustCompile(`([{,]\s*)(\$?[a-zA-Z_][a-zA-Z0-9_$.]*)(\s*:)`)

// relaxJSON converts Compass-style relaxed JSON (unquoted keys, single-quoted
// strings) into strict JSON so bson.UnmarshalExtJSON can parse it.
func relaxJSON(s string) string {
	// Single-quoted strings → double-quoted.
	s = strings.ReplaceAll(s, `'`, `"`)
	// Unquoted object keys → quoted keys.
	return unquotedKeyRe.ReplaceAllString(s, `${1}"${2}"${3}`)
}

// queryOperators are the MongoDB query operators offered by Tab completion
// when the partial token starts with '$'.
var queryOperators = []string{
	"$all", "$and", "$elemMatch", "$eq", "$exists", "$expr",
	"$gt", "$gte", "$in", "$lt", "$lte", "$mod", "$ne", "$nin",
	"$nor", "$not", "$or", "$regex", "$size", "$text", "$type",
}

// filterFieldComplete finds the last partial identifier in input and completes
// it. Tokens starting with '$' complete against MongoDB query operators;
// anything else completes against the provided field names.
// Returns (newInput, allMatches).
func filterFieldComplete(input string, fields []string) (string, []string) {
	if input == "" {
		return input, nil
	}
	// Find the last delimiter that precedes a potential field token.
	lastDelim := strings.LastIndexAny(input, `{, "`)
	partial, prefix := input, ""
	if lastDelim >= 0 {
		partial = input[lastDelim+1:]
		prefix = input[:lastDelim+1]
	}
	if partial == "" {
		return input, nil
	}
	candidates := fields
	if strings.HasPrefix(partial, "$") {
		candidates = queryOperators
	}
	var matches []string
	for _, f := range candidates {
		if strings.HasPrefix(f, partial) {
			matches = append(matches, f)
		}
	}
	if len(matches) == 0 {
		return input, nil
	}
	// If the partial is already an exact match, show the other matches as hints.
	if len(matches) == 1 && matches[0] == partial {
		return input, nil
	}
	// Longest common prefix of all matches.
	lcp := matches[0]
	for _, m := range matches[1:] {
		for !strings.HasPrefix(m, lcp) {
			lcp = lcp[:len(lcp)-1]
		}
	}
	if len(lcp) > len(partial) {
		return prefix + lcp, matches
	}
	return input, matches
}

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
		m.aggMode = false
		m.aggPipeline = ""
		m.err = nil
		m.loading = true
		m.selectedIDs = map[string]interface{}{}
		m.filterHistoryCursor = -1
		return m, tea.Batch(m.spinner.Tick, m.loadPage(0))

	// ── page loaded ──────────────────────────────────────────────────────────
	case msg.DocumentsLoaded:
		m.loading = false
		if message.Err != nil {
			m.err = message.Err
			return m, nil
		}
		m.err = nil
		// If we are showing aggregate results, ignore incoming page loads so
		// background refreshes don't clobber the agg view.
		if m.aggMode {
			return m, nil
		}
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

	// ── pipeline ready (editor closed, file parsed) ───────────────────────────
	case msg.PipelineReady:
		if message.Err != nil {
			return m, statusCmd("pipeline error: " + message.Err.Error())
		}
		// Store the raw text for re-run prefill and fire the actual DB call.
		m.aggPipeline = message.PipelineText
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.aggregateFn(m.db, m.collection, message.Pipeline),
		)

	// ── aggregate result (DB call completed) ──────────────────────────────────
	case msg.AggregateResult:
		m.loading = false
		if message.Err != nil {
			return m, statusCmd("aggregate error: " + message.Err.Error())
		}
		m.err = nil
		m.aggMode = true
		m.aggHistory = pushHistory(m.aggHistory, message.PipelineText, 10)
		m.docs = message.Docs
		m.total = int64(len(message.Docs))
		m.page = 0
		m.cursor = 0
		m.columns = util.BuildColumns(m.docs, maxColumns(m.width))
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
		m.aggMode = false
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
		m.aggMode = false
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
		m.aggMode = false
		if m.cursor > 0 {
			m.cursor--
		}
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.loadPage(m.page),
			statusCmd("document deleted"),
		)

	case msg.BulkDeleted:
		if message.Err != nil {
			return m, statusCmd(fmt.Sprintf("bulk delete failed: %v", message.Err))
		}
		m.selectedIDs = map[string]interface{}{}
		m.aggMode = false
		m.cursor = 0
		m.loading = true
		return m, tea.Batch(
			m.spinner.Tick,
			m.loadPage(m.page),
			statusCmd(fmt.Sprintf("deleted %d documents", message.Count)),
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
	// bulk-delete confirmation takes over all input
	if m.bulkDeleteConfirm {
		return m.handleBulkDeleteConfirm(key)
	}
	// single-delete confirmation takes over all input
	if m.deleteConfirm {
		return m.handleDeleteConfirm(key)
	}
	// filter/sort bar captures all input
	if m.mode != modeNone {
		return m.handleInputKey(key)
	}
	// pipeline picker captures all input
	if m.aggPick {
		return m.handleAggPick(key)
	}

	// esc: clear selection first if any; then exit agg mode
	if key.String() == "esc" {
		if len(m.selectedIDs) > 0 {
			m.selectedIDs = map[string]interface{}{}
			return m, nil
		}
		if m.aggMode {
			m.aggMode = false
			m.aggPipeline = ""
			m.docs = nil
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPage(m.page))
		}
	}

	// Aggregate results are a read-only snapshot: filtering, sorting and
	// editing operate on the live collection, so tell the user instead of
	// silently doing nothing (or worse, mutating data they aren't looking at).
	if m.aggMode {
		switch {
		case isKey(key, m.km.Filter), key.String() == "s", key.String() == "r":
			return m, statusCmd("filter/sort don't apply to aggregate results — edit $match/$sort in the pipeline (a), or esc to exit")
		case isKey(key, m.km.NewDoc), isKey(key, m.km.EditDoc), isKey(key, m.km.DeleteDoc),
			key.String() == "c", key.String() == " ", key.String() == "D":
			return m, statusCmd("editing is not available on aggregate results — esc to return to the collection")
		}
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

	// ── clone doc ────────────────────────────────────────────────────────────
	case key.String() == "c":
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		return m.openEditorClone(doc)

	// ── multi-select ─────────────────────────────────────────────────────────
	case key.String() == " ":
		doc := m.ActiveDoc()
		if doc == nil {
			return m, nil
		}
		k := util.FormatValue(doc["_id"])
		if _, ok := m.selectedIDs[k]; ok {
			delete(m.selectedIDs, k)
		} else {
			m.selectedIDs[k] = doc["_id"]
		}
		// Advance cursor after toggle so space+↓ feels natural.
		m.cursor++
		return m.clamp(), nil

	// ── bulk delete ───────────────────────────────────────────────────────────
	case key.String() == "D":
		if len(m.selectedIDs) == 0 {
			return m, nil
		}
		m.bulkDeleteConfirm = true
		return m, nil

	// ── aggregate pipeline ───────────────────────────────────────────────────
	case key.String() == "a":
		if m.db == "" {
			return m, statusCmd("select a collection first")
		}
		// In agg mode 'a' re-edits the current pipeline directly. Otherwise,
		// when history exists, show a picker of recent pipelines first.
		if !m.aggMode && len(m.aggHistory) > 0 {
			m.aggPick = true
			m.aggPickIdx = 0
			return m, nil
		}
		return m.openAggregateEditor()

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

func (m Model) handleBulkDeleteConfirm(key tea.KeyMsg) (Model, tea.Cmd) {
	m.bulkDeleteConfirm = false
	if key.String() == "y" || key.String() == "Y" {
		ids := make([]interface{}, 0, len(m.selectedIDs))
		for _, id := range m.selectedIDs {
			ids = append(ids, id)
		}
		return m, m.bulkDeleteFn(m.db, m.collection, ids)
	}
	return m, nil
}

// ── filter / sort input bar ───────────────────────────────────────────────────

func (m Model) handleInputKey(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyEnter:
		expr := strings.TrimSpace(m.input.Value())
		m.filterHistoryCursor = -1
		m.filterCompletions = nil
		m.filterCompletionIdx = -1
		switch m.mode {
		case modeFilter:
			return m.commitFilter(expr)
		case modeSort:
			return m.commitSort(expr)
		}

	case tea.KeyTab:
		if m.mode == modeFilter {
			if len(m.filterCompletions) == 0 {
				// First Tab: compute completions and select the first item.
				_, completions := filterFieldComplete(m.input.Value(), m.columns)
				if len(completions) > 0 {
					m.filterCompletions = completions
					m.filterCompletionIdx = 0
					m.input.SetValue(m.applyCompletion(completions[0]))
					m.input.CursorEnd()
				}
			} else {
				// Subsequent Tab: advance selection, cycling through all matches.
				m.filterCompletionIdx = (m.filterCompletionIdx + 1) % len(m.filterCompletions)
				m.input.SetValue(m.applyCompletion(m.filterCompletions[m.filterCompletionIdx]))
				m.input.CursorEnd()
			}
			return m, nil
		}

	case tea.KeyUp:
		if m.mode == modeFilter && len(m.filterCompletions) > 0 {
			// Dropdown is open — navigate upward.
			m.filterCompletionIdx--
			if m.filterCompletionIdx < 0 {
				m.filterCompletionIdx = len(m.filterCompletions) - 1
			}
			m.input.SetValue(m.applyCompletion(m.filterCompletions[m.filterCompletionIdx]))
			m.input.CursorEnd()
			return m, nil
		}
		// Dropdown closed — navigate filter history.
		if m.mode != modeFilter || len(m.filterHistory) == 0 {
			break
		}
		if m.filterHistoryCursor == -1 {
			m.filterHistoryDraft = m.input.Value()
		}
		next := m.filterHistoryCursor + 1
		if next >= len(m.filterHistory) {
			return m, nil
		}
		m.filterHistoryCursor = next
		m.input.SetValue(m.filterHistory[next])
		m.input.CursorEnd()
		return m, nil

	case tea.KeyDown:
		if m.mode == modeFilter && len(m.filterCompletions) > 0 {
			// Dropdown is open — navigate downward.
			m.filterCompletionIdx = (m.filterCompletionIdx + 1) % len(m.filterCompletions)
			m.input.SetValue(m.applyCompletion(m.filterCompletions[m.filterCompletionIdx]))
			m.input.CursorEnd()
			return m, nil
		}
		// Dropdown closed — navigate filter history.
		if m.mode != modeFilter || m.filterHistoryCursor < 0 {
			break
		}
		prev := m.filterHistoryCursor - 1
		if prev < 0 {
			m.filterHistoryCursor = -1
			m.input.SetValue(m.filterHistoryDraft)
			m.input.CursorEnd()
			return m, nil
		}
		m.filterHistoryCursor = prev
		m.input.SetValue(m.filterHistory[prev])
		m.input.CursorEnd()
		return m, nil

	case tea.KeyEsc:
		// If dropdown open, close it first; second Esc cancels the filter bar.
		if len(m.filterCompletions) > 0 {
			m.filterCompletions = nil
			m.filterCompletionIdx = -1
			return m, nil
		}
		m.mode = modeNone
		m.inputErr = ""
		m.filterHistoryCursor = -1

	case tea.KeyCtrlU:
		m.input.SetValue("")
		m.filterHistoryCursor = -1
		m.filterCompletions = nil
		m.filterCompletionIdx = -1

	default:
		// Any character typed closes the dropdown and detaches from history.
		if m.filterHistoryCursor >= 0 {
			m.filterHistoryCursor = -1
		}
		m.filterCompletions = nil
		m.filterCompletionIdx = -1
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(key)
		return m, cmd
	}
	return m, nil
}

// applyCompletion replaces the partial word at the end of the current input
// with the given completion, preserving everything before the last delimiter.
func (m Model) applyCompletion(completion string) string {
	input := m.input.Value()
	lastDelim := strings.LastIndexAny(input, `{, "`)
	if lastDelim >= 0 {
		return input[:lastDelim+1] + completion
	}
	return completion
}

func (m Model) commitFilter(expr string) (Model, tea.Cmd) {
	m.filterCompletions = nil
	m.filterCompletionIdx = -1
	if expr == "" {
		return m.applyFilterSort(nil, m.sort, "", m.sortExpr)
	}
	var filter bson.M
	if err := bson.UnmarshalExtJSON([]byte(expr), false, &filter); err != nil {
		// Try relaxed mode: quote unquoted keys and convert single quotes.
		if err2 := bson.UnmarshalExtJSON([]byte(relaxJSON(expr)), false, &filter); err2 != nil {
			m.inputErr = "bad filter: " + err.Error()
			return m, nil
		}
	}
	m = m.pushFilterHistory(expr)
	return m.applyFilterSort(filter, m.sort, expr, m.sortExpr)
}

const maxFilterHistory = 20

// pushHistory prepends entry to list (newest first), deduplicating any
// existing occurrence and capping the result at max entries.
func pushHistory(list []string, entry string, max int) []string {
	if entry == "" {
		return list
	}
	filtered := make([]string, 0, len(list)+1)
	filtered = append(filtered, entry)
	for _, h := range list {
		if h != entry {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) > max {
		filtered = filtered[:max]
	}
	return filtered
}

func (m Model) pushFilterHistory(expr string) Model {
	m.filterHistory = pushHistory(m.filterHistory, expr, maxFilterHistory)
	return m
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
	m.filterCompletions = nil
	m.input.SetValue(prefill)
	m.input.CursorEnd()
	m.input.Focus()
	return m, m.input.Focus()
}

// BeginFilter opens the filter bar pre-filled with the current filter expression.
// Used by the app when the user presses "/" from the detail panel.
func (m Model) BeginFilter() (Model, tea.Cmd) {
	return m.openInput(modeFilter, m.filterExpr)
}

// EditDoc opens the given document in the editor for editing.
// Used when the detail panel requests an edit via msg.EditDocRequested.
func (m Model) EditDoc(doc bson.M) (Model, tea.Cmd) {
	if doc == nil {
		return m, nil
	}
	return m.openEditorEdit(doc)
}

// ── editor ────────────────────────────────────────────────────────────────────

func newDocTemplate() string {
	id := bson.NewObjectID()
	return fmt.Sprintf("{\n  \"_id\": { \"$oid\": \"%s\" }\n}", id.Hex())
}

func (m Model) openEditorNew() (Model, tea.Cmd) {
	cmd, err := buildEditorCmd(newDocTemplate(), m.editor)
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

	cmd, err := buildEditorCmd(raw, m.editor)
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

func (m Model) openEditorClone(src bson.M) (Model, tea.Cmd) {
	// Copy all fields except _id, then assign a fresh ObjectId so the
	// Extended JSON type syntax is immediately visible in the editor.
	clone := make(bson.M, len(src))
	for k, v := range src {
		if k != "_id" {
			clone[k] = v
		}
	}
	clone["_id"] = bson.NewObjectID()

	raw, err := util.BSONToJSON(clone)
	if err != nil {
		return m, statusCmd("marshal error: " + err.Error())
	}

	cmd, err := buildEditorCmd(raw, m.editor)
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

const aggTemplate = `[
  { "$match": {} }
]`

// handleAggPick drives the recent-pipeline picker: ↑/↓/j/k move, enter opens
// the editor with the chosen pipeline (index 0 = fresh template), esc cancels.
func (m Model) handleAggPick(key tea.KeyMsg) (Model, tea.Cmd) {
	total := len(m.aggHistory) + 1 // +1 for "new pipeline"
	switch key.String() {
	case "j", "down", "tab":
		m.aggPickIdx = (m.aggPickIdx + 1) % total
		return m, nil
	case "k", "up":
		m.aggPickIdx--
		if m.aggPickIdx < 0 {
			m.aggPickIdx = total - 1
		}
		return m, nil
	case "enter":
		idx := m.aggPickIdx
		m.aggPick = false
		m.aggPickIdx = 0
		if idx == 0 {
			return m.openAggregateEditorWith(aggTemplate)
		}
		if idx-1 < len(m.aggHistory) {
			return m.openAggregateEditorWith(m.aggHistory[idx-1])
		}
		return m, nil
	case "esc", "q":
		m.aggPick = false
		m.aggPickIdx = 0
		return m, nil
	}
	return m, nil
}

func (m Model) openAggregateEditor() (Model, tea.Cmd) {
	content := m.aggPipeline
	if content == "" {
		content = aggTemplate
	}
	return m.openAggregateEditorWith(content)
}

func (m Model) openAggregateEditorWith(content string) (Model, tea.Cmd) {
	if content == "" {
		content = aggTemplate
	}
	ec, err := buildEditorCmd(content, m.editor)
	if err != nil {
		return m, statusCmd("error: " + err.Error())
	}
	return m, tea.ExecProcess(ec.cmd, func(execErr error) tea.Msg {
		defer os.Remove(ec.path)
		if execErr != nil {
			return msg.PipelineReady{Err: execErr}
		}
		data, err := os.ReadFile(ec.path)
		if err != nil {
			return msg.PipelineReady{Err: err}
		}
		raw := strings.TrimSpace(string(data))
		if raw == "" {
			return msg.PipelineReady{Err: fmt.Errorf("empty pipeline — no changes")}
		}
		var pipeline bson.A
		if err := bson.UnmarshalExtJSON([]byte(raw), false, &pipeline); err != nil {
			return msg.PipelineReady{Err: fmt.Errorf("invalid pipeline JSON: %w", err)}
		}
		return msg.PipelineReady{Pipeline: pipeline, PipelineText: raw}
	})
}

type editorCmd struct {
	cmd  *exec.Cmd
	path string
}

func buildEditorCmd(content, editor string) (editorCmd, error) {
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

	if editor == "" {
		editor = "vim"
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
