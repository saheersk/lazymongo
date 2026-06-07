package documents

import (
	"errors"
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/keymap"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/tui/style"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// newTestModel returns a Model wired with stub functions so unit tests can
// exercise Update without touching MongoDB.
func newTestModel(
	fetchPage FetchPageFn,
	insertFn InsertFn,
	replaceFn ReplaceFn,
	deleteFn DeleteFn,
) Model {
	th := style.Default()
	km := keymap.Default()
	if fetchPage == nil {
		fetchPage = func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
			return nil
		}
	}
	if insertFn == nil {
		insertFn = func(db, col string, doc bson.M) tea.Cmd { return nil }
	}
	if replaceFn == nil {
		replaceFn = func(db, col string, id interface{}, doc bson.M) tea.Cmd { return nil }
	}
	if deleteFn == nil {
		deleteFn = func(db, col string, id interface{}) tea.Cmd { return nil }
	}
	bulkDeleteFn := func(db, col string, ids []interface{}) tea.Cmd { return nil }
	aggregateFn := func(db, col string, pipeline bson.A) tea.Cmd { return nil }
	exportFn := func(db, col string, filter bson.M, sort bson.D, format string) tea.Cmd { return nil }
	return New(th, km, fetchPage, insertFn, replaceFn, deleteFn, bulkDeleteFn, aggregateFn, exportFn)
}

// pressKey simulates a key-press message.
func pressKey(m Model, k string) (Model, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
}

// pressSpecialKey simulates a named key (enter, esc, etc.).
func pressSpecialKey(m Model, kt tea.KeyType) (Model, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: kt})
}

// ── CollectionSelected ────────────────────────────────────────────────────────

func TestUpdate_CollectionSelected_ResetsState(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	// Give the model some state to reset.
	m.filter = bson.M{"x": 1}
	m.filterExpr = "filter"
	m.sortExpr = "sort"
	m.cursor = 5
	m.page = 3
	m.deleteConfirm = true
	m.err = errors.New("previous error")

	sel := msg.CollectionSelected{DB: "testdb", Collection: "col1"}
	m, _ = m.Update(sel)

	if m.db != "testdb" {
		t.Errorf("db = %q; want %q", m.db, "testdb")
	}
	if m.collection != "col1" {
		t.Errorf("collection = %q; want %q", m.collection, "col1")
	}
	if m.page != 0 {
		t.Errorf("page = %d; want 0", m.page)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d; want 0", m.cursor)
	}
	if m.filter != nil {
		t.Errorf("filter should be nil, got %v", m.filter)
	}
	if m.filterExpr != "" {
		t.Errorf("filterExpr should be empty, got %q", m.filterExpr)
	}
	if m.sortExpr != "" {
		t.Errorf("sortExpr should be empty, got %q", m.sortExpr)
	}
	if m.deleteConfirm {
		t.Error("deleteConfirm should be false after reset")
	}
	if m.err != nil {
		t.Errorf("err should be nil, got %v", m.err)
	}
	if !m.loading {
		t.Error("loading should be true after CollectionSelected")
	}
}

// ── DocumentsLoaded ───────────────────────────────────────────────────────────

func TestUpdate_DocumentsLoaded_SetsDocs(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "testdb"
	m.collection = "col"
	m.loading = true
	m.width = 120 // affects maxColumns

	docs := []bson.M{
		{"_id": bson.NewObjectID(), "name": "Alice"},
		{"_id": bson.NewObjectID(), "name": "Bob"},
	}
	loaded := msg.DocumentsLoaded{
		Result: msg.PageResult{Docs: docs, Total: 2, Page: 0, PageSize: 50},
	}
	m, _ = m.Update(loaded)

	if m.loading {
		t.Error("loading should be false after DocumentsLoaded")
	}
	if len(m.docs) != 2 {
		t.Errorf("docs len = %d; want 2", len(m.docs))
	}
	if m.err != nil {
		t.Errorf("err should be nil, got %v", m.err)
	}
	if len(m.columns) == 0 {
		t.Error("columns should be populated after DocumentsLoaded")
	}
	if m.columns[0] != "_id" {
		t.Errorf("first column should be _id, got %q", m.columns[0])
	}
}

func TestUpdate_DocumentsLoaded_WithError(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.loading = true
	theErr := errors.New("timeout")

	m, _ = m.Update(msg.DocumentsLoaded{Err: theErr})

	if m.loading {
		t.Error("loading should be false after DocumentsLoaded with error")
	}
	if m.err == nil {
		t.Error("err should be set after DocumentsLoaded with error")
	}
	if len(m.docs) != 0 {
		t.Errorf("docs should be empty on error, got %d", len(m.docs))
	}
}

// ── Navigation keys ───────────────────────────────────────────────────────────

func TestUpdate_Key_J_IncrementsCursor(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
	}
	m.cursor = 0

	m, _ = pressKey(m, "j")

	if m.cursor != 1 {
		t.Errorf("cursor = %d; want 1", m.cursor)
	}
}

func TestUpdate_Key_J_ClampsAtEnd(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{
		{"_id": bson.NewObjectID()},
	}
	m.cursor = 0
	m.total = 1
	m.pageSize = 50

	m, _ = pressKey(m, "j")

	// Only 1 doc, cursor should not exceed 0.
	if m.cursor != 0 {
		t.Errorf("cursor = %d; want 0 (clamped)", m.cursor)
	}
}

func TestUpdate_Key_K_DecrementsCursor(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
	}
	m.cursor = 1

	m, _ = pressKey(m, "k")

	if m.cursor != 0 {
		t.Errorf("cursor = %d; want 0", m.cursor)
	}
}

func TestUpdate_Key_K_ClampsAtZero(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID()}}
	m.cursor = 0
	m.page = 0

	m, _ = pressKey(m, "k")

	if m.cursor < 0 {
		t.Errorf("cursor = %d; must not go below 0", m.cursor)
	}
}

func TestUpdate_Key_G_JumpsToTop(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
	}
	m.cursor = 1
	m.page = 0

	m, _ = pressKey(m, "g")

	if m.cursor != 0 {
		t.Errorf("cursor = %d; want 0 after 'g'", m.cursor)
	}
}

func TestUpdate_Key_CapGJumpsToBottom(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
	}
	m.cursor = 0
	m.total = 3
	m.pageSize = 50

	m, _ = pressKey(m, "G")

	if m.cursor != 2 {
		t.Errorf("cursor = %d; want 2 after 'G'", m.cursor)
	}
}

// ── Filter mode ───────────────────────────────────────────────────────────────

func TestUpdate_Key_Slash_OpensFilterMode(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	m, _ = pressKey(m, "/")

	if m.mode != modeFilter {
		t.Errorf("mode = %v; want modeFilter (%v)", m.mode, modeFilter)
	}
}

func TestUpdate_FilterMode_KeysGoToInput(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
	}
	m.cursor = 0

	// Enter filter mode.
	m, _ = pressKey(m, "/")

	// Now press "j" — it should feed into the text input, not move cursor.
	cursorBefore := m.cursor
	m, _ = pressKey(m, "j")

	if m.cursor != cursorBefore {
		t.Errorf("cursor moved in filter mode: before=%d after=%d", cursorBefore, m.cursor)
	}
	if m.mode != modeFilter {
		t.Error("should still be in filter mode after typing 'j'")
	}
}

func TestUpdate_FilterMode_EscCancels(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m, _ = pressKey(m, "/")

	m, _ = pressSpecialKey(m, tea.KeyEsc)

	if m.mode != modeNone {
		t.Errorf("mode = %v; want modeNone after esc", m.mode)
	}
}

// ── Sort mode ─────────────────────────────────────────────────────────────────

func TestUpdate_Key_S_OpensSortMode(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	m, _ = pressKey(m, "s")

	if m.mode != modeSort {
		t.Errorf("mode = %v; want modeSort (%v)", m.mode, modeSort)
	}
}

// ── Reset key ─────────────────────────────────────────────────────────────────

func TestUpdate_Key_R_NoOpWhenNoFilter(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.filterExpr = ""
	m.sortExpr = ""
	m.loading = false

	m, cmd := pressKey(m, "r")

	if m.loading {
		t.Error("'r' with no filter should not set loading=true")
	}
	if cmd != nil {
		t.Error("'r' with no filter should return nil cmd")
	}
}

func TestUpdate_Key_R_ClearsFilterWhenSet(t *testing.T) {
	fetchCalled := false
	fetchPage := func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
		fetchCalled = true
		return nil
	}
	m := newTestModel(fetchPage, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.filterExpr = `{"name":"Alice"}`
	m.filter = bson.M{"name": "Alice"}

	m, _ = pressKey(m, "r")

	if m.filterExpr != "" {
		t.Errorf("filterExpr = %q; want empty after reset", m.filterExpr)
	}
	if m.filter != nil {
		t.Errorf("filter should be nil after reset, got %v", m.filter)
	}
	if !fetchCalled {
		t.Error("fetchPage should have been called after reset")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestUpdate_Key_D_NoDocIsNoOp(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = nil

	m, _ = pressKey(m, "d")

	if m.deleteConfirm {
		t.Error("deleteConfirm should not be set when no doc")
	}
}

func TestUpdate_Key_D_WithDocSetsDeleteConfirm(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID(), "name": "test"}}
	m.cursor = 0

	m, _ = pressKey(m, "d")

	if !m.deleteConfirm {
		t.Error("deleteConfirm should be true after pressing 'd' on a doc")
	}
}

func TestUpdate_DeleteConfirm_Y_FiresDeleteFn(t *testing.T) {
	deleteCalled := false
	deleteFn := func(db, col string, id interface{}) tea.Cmd {
		deleteCalled = true
		return func() tea.Msg { return msg.DocumentDeleted{} }
	}
	m := newTestModel(nil, nil, nil, deleteFn)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID(), "name": "test"}}
	m.cursor = 0
	m.deleteConfirm = true

	m, _ = pressKey(m, "y")

	if m.deleteConfirm {
		t.Error("deleteConfirm should be false after confirm")
	}
	if !deleteCalled {
		t.Error("deleteFn should have been called")
	}
}

func TestUpdate_DeleteConfirm_N_Cancels(t *testing.T) {
	deleteCalled := false
	deleteFn := func(db, col string, id interface{}) tea.Cmd {
		deleteCalled = true
		return nil
	}
	m := newTestModel(nil, nil, nil, deleteFn)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID()}}
	m.cursor = 0
	m.deleteConfirm = true

	m, _ = pressKey(m, "n")

	if m.deleteConfirm {
		t.Error("deleteConfirm should be false after cancel")
	}
	if deleteCalled {
		t.Error("deleteFn should NOT have been called on cancel")
	}
}

func TestUpdate_DeleteConfirm_AnyKeyOtherThanY_Cancels(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID()}}
	m.cursor = 0
	m.deleteConfirm = true

	m, _ = pressKey(m, "x")

	if m.deleteConfirm {
		t.Error("deleteConfirm should be false after non-y key")
	}
}

// ── EditorDone ────────────────────────────────────────────────────────────────

func TestUpdate_EditorDone_WithError_ProducesStatusCmd(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	_, cmd := m.Update(msg.EditorDone{Err: errors.New("editor crashed")})

	if cmd == nil {
		t.Fatal("expected a cmd after EditorDone with error, got nil")
	}
	result := cmd()
	su, ok := result.(msg.StatusUpdate)
	if !ok {
		t.Fatalf("expected StatusUpdate, got %T", result)
	}
	if su.Text == "" {
		t.Error("StatusUpdate.Text should not be empty")
	}
}

func TestUpdate_EditorDone_IsNew_CallsInsertFn(t *testing.T) {
	insertCalled := false
	insertFn := func(db, col string, doc bson.M) tea.Cmd {
		insertCalled = true
		return func() tea.Msg { return msg.DocumentCreated{InsertedID: bson.NewObjectID()} }
	}
	m := newTestModel(nil, insertFn, nil, nil)
	m.db = "db"
	m.collection = "col"

	_, cmd := m.Update(msg.EditorDone{Doc: bson.M{"name": "new"}, IsNew: true})

	if cmd == nil {
		t.Fatal("expected cmd from EditorDone IsNew=true")
	}
	cmd() // execute to trigger insertFn

	if !insertCalled {
		t.Error("insertFn should have been called for IsNew=true")
	}
}

func TestUpdate_EditorDone_IsNotNew_CallsReplaceFn(t *testing.T) {
	replaceCalled := false
	replaceFn := func(db, col string, id interface{}, doc bson.M) tea.Cmd {
		replaceCalled = true
		return func() tea.Msg { return msg.DocumentUpdated{} }
	}
	m := newTestModel(nil, nil, replaceFn, nil)
	m.db = "db"
	m.collection = "col"
	origID := bson.NewObjectID()

	_, cmd := m.Update(msg.EditorDone{
		Doc:    bson.M{"name": "updated"},
		IsNew:  false,
		OrigID: origID,
	})

	if cmd == nil {
		t.Fatal("expected cmd from EditorDone IsNew=false")
	}
	cmd()

	if !replaceCalled {
		t.Error("replaceFn should have been called for IsNew=false")
	}
}

func TestUpdate_EditorDone_NilDoc_ProducesNoChangesStatus(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	_, cmd := m.Update(msg.EditorDone{Doc: nil, IsNew: false})

	if cmd == nil {
		t.Fatal("expected cmd when doc is nil")
	}
	result := cmd()
	su, ok := result.(msg.StatusUpdate)
	if !ok {
		t.Fatalf("expected StatusUpdate, got %T", result)
	}
	if su.Text != "no changes" {
		t.Errorf("StatusUpdate.Text = %q; want 'no changes'", su.Text)
	}
}

// ── DocumentCreated ───────────────────────────────────────────────────────────

func TestUpdate_DocumentCreated_Success_SetsLoading(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.loading = false

	m, _ = m.Update(msg.DocumentCreated{InsertedID: bson.NewObjectID()})

	if !m.loading {
		t.Error("loading should be true after DocumentCreated success")
	}
}

func TestUpdate_DocumentCreated_Error_ProducesErrorStatus(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	_, cmd := m.Update(msg.DocumentCreated{Err: errors.New("dup key")})

	if cmd == nil {
		t.Fatal("expected cmd after DocumentCreated with error")
	}
	result := cmd()
	su, ok := result.(msg.StatusUpdate)
	if !ok {
		t.Fatalf("expected StatusUpdate, got %T", result)
	}
	if su.Text == "" {
		t.Error("StatusUpdate.Text should not be empty on error")
	}
}

// ── DocumentUpdated ───────────────────────────────────────────────────────────

func TestUpdate_DocumentUpdated_Success_SetsLoading(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.loading = false

	m, _ = m.Update(msg.DocumentUpdated{})

	if !m.loading {
		t.Error("loading should be true after DocumentUpdated success")
	}
}

func TestUpdate_DocumentUpdated_Error_ProducesErrorStatus(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	_, cmd := m.Update(msg.DocumentUpdated{Err: errors.New("write concern error")})

	if cmd == nil {
		t.Fatal("expected cmd after DocumentUpdated with error")
	}
	result := cmd()
	su, ok := result.(msg.StatusUpdate)
	if !ok {
		t.Fatalf("expected StatusUpdate, got %T", result)
	}
	if su.Text == "" {
		t.Error("StatusUpdate.Text should not be empty on error")
	}
}

// ── DocumentDeleted ───────────────────────────────────────────────────────────

func TestUpdate_DocumentDeleted_Success_DecrementsAndSetsLoading(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{
		{"_id": bson.NewObjectID()},
		{"_id": bson.NewObjectID()},
	}
	m.cursor = 1
	m.loading = false

	m, _ = m.Update(msg.DocumentDeleted{})

	if !m.loading {
		t.Error("loading should be true after DocumentDeleted success")
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d; want 0 after delete with cursor=1", m.cursor)
	}
}

func TestUpdate_DocumentDeleted_CursorAtZero_StaysZero(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID()}}
	m.cursor = 0

	m, _ = m.Update(msg.DocumentDeleted{})

	if m.cursor < 0 {
		t.Errorf("cursor = %d; should not go negative", m.cursor)
	}
}

func TestUpdate_DocumentDeleted_Error_ProducesErrorStatus(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	_, cmd := m.Update(msg.DocumentDeleted{Err: errors.New("not found")})

	if cmd == nil {
		t.Fatal("expected cmd after DocumentDeleted with error")
	}
	result := cmd()
	su, ok := result.(msg.StatusUpdate)
	if !ok {
		t.Fatalf("expected StatusUpdate, got %T", result)
	}
	if su.Text == "" {
		t.Error("StatusUpdate.Text should not be empty on error")
	}
}

// ── After filter applied ──────────────────────────────────────────────────────

func TestUpdate_CommitFilter_SetsFilterExprAndResetsPage(t *testing.T) {
	fetchPage := func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
		return nil
	}
	m := newTestModel(fetchPage, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.page = 3

	// Open filter mode.
	m, _ = pressKey(m, "/")

	// Type the filter expression one rune at a time via the Update machinery.
	filterExpr := `{"name":"Alice"}`
	for _, r := range filterExpr {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Commit with Enter.
	m, _ = pressSpecialKey(m, tea.KeyEnter)

	if m.filterExpr != filterExpr {
		t.Errorf("filterExpr = %q; want %q", m.filterExpr, filterExpr)
	}
	if m.page != 0 {
		t.Errorf("page = %d; want 0 after filter applied", m.page)
	}
	if m.mode != modeNone {
		t.Errorf("mode = %v; want modeNone after commit", m.mode)
	}
}

func TestUpdate_CommitFilter_BadJSON_SetsInputErr(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	m, _ = pressKey(m, "/")

	badExpr := `{bad json`
	for _, r := range badExpr {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = pressSpecialKey(m, tea.KeyEnter)

	if m.inputErr == "" {
		t.Error("inputErr should be set for bad JSON filter")
	}
	if m.mode != modeFilter {
		t.Errorf("mode = %v; want modeFilter (stay in filter mode on error)", m.mode)
	}
}

// ── parseSort (tested indirectly via commitSort) ──────────────────────────────

func TestUpdate_ParseSort_FieldName_AscendingByDefault(t *testing.T) {
	fetchPage := func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
		return nil
	}
	m := newTestModel(fetchPage, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	m, _ = pressKey(m, "s")

	for _, r := range "name" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = pressSpecialKey(m, tea.KeyEnter)

	if m.inputErr != "" {
		t.Fatalf("unexpected inputErr: %q", m.inputErr)
	}
	if m.sortExpr != "name" {
		t.Errorf("sortExpr = %q; want %q", m.sortExpr, "name")
	}
	if len(m.sort) != 1 {
		t.Fatalf("sort len = %d; want 1", len(m.sort))
	}
	if m.sort[0].Key != "name" {
		t.Errorf("sort key = %q; want 'name'", m.sort[0].Key)
	}
	if m.sort[0].Value != 1 {
		t.Errorf("sort value = %v; want 1 (ascending)", m.sort[0].Value)
	}
}

func TestUpdate_ParseSort_DashPrefix_Descending(t *testing.T) {
	fetchPage := func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
		return nil
	}
	m := newTestModel(fetchPage, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	m, _ = pressKey(m, "s")

	for _, r := range "-age" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = pressSpecialKey(m, tea.KeyEnter)

	if m.inputErr != "" {
		t.Fatalf("unexpected inputErr: %q", m.inputErr)
	}
	if len(m.sort) != 1 {
		t.Fatalf("sort len = %d; want 1", len(m.sort))
	}
	if m.sort[0].Key != "age" {
		t.Errorf("sort key = %q; want 'age'", m.sort[0].Key)
	}
	if m.sort[0].Value != -1 {
		t.Errorf("sort value = %v; want -1 (descending)", m.sort[0].Value)
	}
}

func TestUpdate_ParseSort_JSONObject(t *testing.T) {
	fetchPage := func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
		return nil
	}
	m := newTestModel(fetchPage, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	m, _ = pressKey(m, "s")

	for _, r := range `{"score":1}` {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = pressSpecialKey(m, tea.KeyEnter)

	if m.inputErr != "" {
		t.Fatalf("unexpected inputErr: %q", m.inputErr)
	}
	if len(m.sort) != 1 {
		t.Fatalf("sort len = %d; want 1", len(m.sort))
	}
	if m.sort[0].Key != "score" {
		t.Errorf("sort key = %q; want 'score'", m.sort[0].Key)
	}
}

func TestUpdate_ParseSort_Empty_ClearsSort(t *testing.T) {
	fetchPage := func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd {
		return nil
	}
	m := newTestModel(fetchPage, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.sort = bson.D{{Key: "name", Value: 1}}
	m.sortExpr = "name"

	// Open sort — the input is pre-filled with the existing sortExpr ("name").
	m, _ = pressKey(m, "s")

	// Clear the input with Ctrl+U, then commit with Enter → should clear sort.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m, _ = pressSpecialKey(m, tea.KeyEnter)

	if m.sort != nil {
		t.Errorf("sort should be nil after empty commit, got %v", m.sort)
	}
	if m.sortExpr != "" {
		t.Errorf("sortExpr should be empty, got %q", m.sortExpr)
	}
}

// ── InInputMode helper ────────────────────────────────────────────────────────

func TestModel_InInputMode(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	if m.InInputMode() {
		t.Error("InInputMode should be false initially")
	}

	m.mode = modeFilter
	if !m.InInputMode() {
		t.Error("InInputMode should be true when mode=modeFilter")
	}

	m.mode = modeNone
	m.deleteConfirm = true
	if !m.InInputMode() {
		t.Error("InInputMode should be true when deleteConfirm=true")
	}
}

// ── Collection display helper ─────────────────────────────────────────────────

func TestModel_Collection_Display(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	if m.Collection() != "" {
		t.Errorf("Collection() = %q; want empty when no db", m.Collection())
	}

	m.db = "mydb"
	if m.Collection() != "mydb" {
		t.Errorf("Collection() = %q; want 'mydb' when no collection", m.Collection())
	}

	m.collection = "mycol"
	want := "mydb > mycol"
	if m.Collection() != want {
		t.Errorf("Collection() = %q; want %q", m.Collection(), want)
	}
}

// ── PageInfo ──────────────────────────────────────────────────────────────────

func TestModel_PageInfo(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.page = 2
	m.total = 150
	m.pageSize = 50
	m.docs = make([]bson.M, 50)

	page, _, total := m.PageInfo()
	if page != 2 {
		t.Errorf("page = %d; want 2", page)
	}
	if total != 150 {
		t.Errorf("total = %d; want 150", total)
	}
}

// ── Verify statusCmd helper produces correct message ─────────────────────────

func TestStatusCmd(t *testing.T) {
	cmd := statusCmd("hello")
	result := cmd()
	su, ok := result.(msg.StatusUpdate)
	if !ok {
		t.Fatalf("expected StatusUpdate, got %T", result)
	}
	if su.Text != "hello" {
		t.Errorf("StatusUpdate.Text = %q; want 'hello'", su.Text)
	}
}

// ── filterChanged is passed through (no-op handler) ──────────────────────────

func TestUpdate_FilterChanged_IsNoOp(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.page = 5

	m2, cmd := m.Update(msg.FilterChanged{Filter: bson.M{"x": 1}})

	// FilterChanged handler just returns m, nil.
	if cmd != nil {
		t.Error("FilterChanged should return nil cmd")
	}
	_ = m2 // no state changes expected
}

// ── Multi-select (space key) ──────────────────────────────────────────────────

func TestUpdate_Space_TogglesSelectionOn(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	id := bson.NewObjectID()
	m.docs = []bson.M{{"_id": id, "name": "Alice"}, {"_id": bson.NewObjectID()}}
	m.cursor = 0

	m, _ = pressKey(m, " ")

	if m.SelectionCount() != 1 {
		t.Errorf("SelectionCount = %d; want 1 after space", m.SelectionCount())
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d; want 1 (space advances cursor)", m.cursor)
	}
}

func TestUpdate_Space_TogglesSelectionOff(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	id := bson.NewObjectID()
	m.docs = []bson.M{{"_id": id}}
	m.cursor = 0

	// First press selects.
	m, _ = pressKey(m, " ")
	if m.SelectionCount() != 1 {
		t.Fatalf("expected 1 selected after first space, got %d", m.SelectionCount())
	}

	// Reset cursor to 0 (it advanced to 1, but only 1 doc so it clamped to 0).
	m.cursor = 0
	// Second press deselects.
	m, _ = pressKey(m, " ")
	if m.SelectionCount() != 0 {
		t.Errorf("SelectionCount = %d; want 0 after second space (deselect)", m.SelectionCount())
	}
}

func TestUpdate_Space_NoDocIsNoOp(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = nil

	m, _ = pressKey(m, " ")

	if m.SelectionCount() != 0 {
		t.Errorf("SelectionCount = %d; want 0 when no docs", m.SelectionCount())
	}
}

func TestUpdate_Esc_ClearsSelection(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	id := bson.NewObjectID()
	m.docs = []bson.M{{"_id": id}}
	m.cursor = 0

	// Select a doc.
	m, _ = pressKey(m, " ")
	if m.SelectionCount() == 0 {
		t.Fatal("precondition: expected selection")
	}

	m.cursor = 0
	m, _ = pressSpecialKey(m, tea.KeyEsc)

	if m.SelectionCount() != 0 {
		t.Errorf("SelectionCount = %d; want 0 after esc", m.SelectionCount())
	}
}

// ── Bulk delete confirm ───────────────────────────────────────────────────────

func TestUpdate_BulkDelete_D_WithSelectionSetsBulkDeleteConfirm(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	id := bson.NewObjectID()
	m.docs = []bson.M{{"_id": id}}
	m.cursor = 0
	m, _ = pressKey(m, " ") // select
	m.cursor = 0

	m, _ = pressKey(m, "D")

	if !m.bulkDeleteConfirm {
		t.Error("bulkDeleteConfirm should be true after D with selection")
	}
}

func TestUpdate_BulkDelete_D_WithNoSelectionIsNoOp(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID()}}
	m.cursor = 0

	m, _ = pressKey(m, "D")

	if m.bulkDeleteConfirm {
		t.Error("bulkDeleteConfirm should be false when no selection")
	}
}

func TestUpdate_BulkDeleteConfirm_Y_FiresBulkDeleteFn(t *testing.T) {
	bulkCalled := false
	bulkDeleteFn := func(db, col string, ids []interface{}) tea.Cmd {
		bulkCalled = true
		return func() tea.Msg { return msg.BulkDeleted{Count: int64(len(ids))} }
	}
	th := style.Default()
	km := keymap.Default()
	m := New(th, km,
		func(db, col string, filter bson.M, sort bson.D, page int) tea.Cmd { return nil },
		func(db, col string, doc bson.M) tea.Cmd { return nil },
		func(db, col string, id interface{}, doc bson.M) tea.Cmd { return nil },
		func(db, col string, id interface{}) tea.Cmd { return nil },
		bulkDeleteFn,
		func(db, col string, pipeline bson.A) tea.Cmd { return nil },
		func(db, col string, filter bson.M, sort bson.D, format string) tea.Cmd { return nil },
	)
	m.db = "db"
	m.collection = "col"
	id := bson.NewObjectID()
	m.docs = []bson.M{{"_id": id}}
	m.cursor = 0
	m, _ = pressKey(m, " ") // select
	m.cursor = 0
	m, _ = pressKey(m, "D") // set confirm

	m, cmd := pressKey(m, "y")

	if m.bulkDeleteConfirm {
		t.Error("bulkDeleteConfirm should be false after confirm")
	}
	if cmd == nil {
		t.Fatal("expected a cmd after bulk delete confirm")
	}
	cmd() // execute to trigger bulkDeleteFn
	if !bulkCalled {
		t.Error("bulkDeleteFn should have been called")
	}
}

func TestUpdate_BulkDeleteConfirm_AnyOtherKey_Cancels(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID()}}
	m.cursor = 0
	m, _ = pressKey(m, " ")
	m.cursor = 0
	m.bulkDeleteConfirm = true

	m, _ = pressKey(m, "n")

	if m.bulkDeleteConfirm {
		t.Error("bulkDeleteConfirm should be false after non-y key")
	}
}

func TestUpdate_InInputMode_WithBulkDeleteConfirm(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	if m.InInputMode() {
		t.Error("InInputMode should be false initially")
	}

	m.bulkDeleteConfirm = true
	if !m.InInputMode() {
		t.Error("InInputMode should be true when bulkDeleteConfirm=true")
	}
}

func TestUpdate_BulkDeleted_Msg_ClearsSelection(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"
	m.docs = []bson.M{{"_id": bson.NewObjectID()}}
	m.cursor = 0
	m, _ = pressKey(m, " ") // select
	if m.SelectionCount() == 0 {
		t.Fatal("precondition: expected selection")
	}

	m, _ = m.Update(msg.BulkDeleted{Count: 1})

	if m.SelectionCount() != 0 {
		t.Errorf("SelectionCount = %d; want 0 after BulkDeleted", m.SelectionCount())
	}
	if !m.loading {
		t.Error("loading should be true after BulkDeleted (triggers page reload)")
	}
}

func TestUpdate_BulkDeleted_Error_EmitsStatus(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	_, cmd := m.Update(msg.BulkDeleted{Err: fmt.Errorf("write error")})

	if cmd == nil {
		t.Fatal("expected status cmd after BulkDeleted error")
	}
	result := cmd()
	su, ok := result.(msg.StatusUpdate)
	if !ok {
		t.Fatalf("expected StatusUpdate, got %T", result)
	}
	if su.Text == "" {
		t.Error("StatusUpdate.Text should not be empty on error")
	}
}

// ── Filter history ────────────────────────────────────────────────────────────

func TestUpdate_FilterHistory_PushOnCommit(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	// Push 3 distinct entries directly (avoids textinput accumulation).
	m = m.pushFilterHistory(`{"a":1}`)
	m = m.pushFilterHistory(`{"b":2}`)
	m = m.pushFilterHistory(`{"c":3}`)

	if len(m.filterHistory) != 3 {
		t.Fatalf("filterHistory len = %d; want 3", len(m.filterHistory))
	}
	// Newest first.
	if m.filterHistory[0] != `{"c":3}` {
		t.Errorf("filterHistory[0] = %q; want newest entry", m.filterHistory[0])
	}
}

func TestUpdate_FilterHistory_Deduplicates(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	expr := `{"x":1}`
	m = m.pushFilterHistory(expr)
	m = m.pushFilterHistory(`{"other":1}`)
	m = m.pushFilterHistory(expr) // re-push same expr — should deduplicate

	if len(m.filterHistory) != 2 {
		t.Errorf("filterHistory len = %d; want 2 (deduplicated)", len(m.filterHistory))
	}
	if m.filterHistory[0] != expr {
		t.Errorf("filterHistory[0] = %q; want %q (most recent)", m.filterHistory[0], expr)
	}
}

func TestUpdate_FilterHistory_CapsAtMax(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	for i := 0; i < 25; i++ {
		m = m.pushFilterHistory(fmt.Sprintf(`{"i":%d}`, i))
	}

	if len(m.filterHistory) > maxFilterHistory {
		t.Errorf("filterHistory len = %d; must not exceed %d", len(m.filterHistory), maxFilterHistory)
	}
}

func TestUpdate_FilterHistory_NavigateUpAndDown(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "db"
	m.collection = "col"

	// Pre-seed history directly.
	m = m.pushFilterHistory(`{"a":1}`)
	m = m.pushFilterHistory(`{"b":2}`)
	// filterHistory is now: [{"b":2}, {"a":1}] (newest first)

	// Open filter mode — prefilled with current filterExpr (""), so clear it.
	m, _ = pressKey(m, "/")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU}) // clear input
	// Type a fresh draft.
	for _, r := range "draft" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Up — should show history[0] (newest = {"b":2}).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.input.Value() != `{"b":2}` {
		t.Errorf("after ↑: input = %q; want %q", m.input.Value(), `{"b":2}`)
	}
	if m.filterHistoryCursor != 0 {
		t.Errorf("filterHistoryCursor = %d; want 0", m.filterHistoryCursor)
	}

	// Press Up again — should show history[1] ({"a":1}).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.input.Value() != `{"a":1}` {
		t.Errorf("after ↑↑: input = %q; want %q", m.input.Value(), `{"a":1}`)
	}

	// Press Down — back to history[0].
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.input.Value() != `{"b":2}` {
		t.Errorf("after ↑↑↓: input = %q; want %q", m.input.Value(), `{"b":2}`)
	}

	// Press Down again — should restore draft.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.input.Value() != "draft" {
		t.Errorf("after ↑↑↓↓: input = %q; want draft restored", m.input.Value())
	}
	if m.filterHistoryCursor != -1 {
		t.Errorf("filterHistoryCursor = %d; want -1 (back to editing)", m.filterHistoryCursor)
	}
}

// ── Ensure parseSort is exercised directly (exported path) ────────────────────

func TestParseSort_DirectCases(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantKey string
		wantVal int
		wantErr bool
	}{
		{"ascending", "field", "field", 1, false},
		{"descending", "-field", "field", -1, false},
		{"json object", `{"a":1}`, "a", 1, false},
		{"empty dash only", "-", "", 0, true},
		{"empty", "", "", 0, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSort(tc.expr)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for expr=%q", tc.expr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) == 0 {
				t.Fatal("expected at least one element in bson.D")
			}
			if got[0].Key != tc.wantKey {
				t.Errorf("Key = %q; want %q", got[0].Key, tc.wantKey)
			}
			// Value is stored as int (from JSON unmarshal) or as int literal.
			valStr := fmt.Sprintf("%v", got[0].Value)
			wantStr := fmt.Sprintf("%d", tc.wantVal)
			if valStr != wantStr {
				t.Errorf("Value = %v; want %d", got[0].Value, tc.wantVal)
			}
		})
	}
}
