package documents

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAggregateKey_NonNilCmd(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "testdb"
	m.collection = "testcol"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd == nil {
		t.Fatal("pressing 'a' returned nil cmd — openAggregateEditor was not triggered")
	}
}

func TestAggregateKey_NoDB_NilCmd(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "" // no collection selected

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd != nil {
		t.Errorf("expected nil cmd with no collection, got %T", cmd)
	}
}

func TestAggregateResult_SetsAggMode(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db = "testdb"
	m.collection = "testcol"

	docs := []bson.M{{"_id": "1", "x": int32(1)}, {"_id": "2", "x": int32(2)}}
	m2, _ := m.Update(msg.AggregateResult{Docs: docs})

	if !m2.aggMode {
		t.Error("aggMode should be true after AggregateResult")
	}
	if len(m2.docs) != 2 {
		t.Errorf("docs: got %d, want 2", len(m2.docs))
	}
	if m2.total != 2 {
		t.Errorf("total: got %d, want 2", m2.total)
	}
}

func TestAggregateResult_Error_AggModeFalse(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m2, _ := m.Update(msg.AggregateResult{Err: errors.New("pipeline error")})
	if m2.aggMode {
		t.Error("aggMode should be false when AggregateResult has an error")
	}
}

func TestEscInAggMode_ExitsAndReloads(t *testing.T) {
	var fetchCalled bool
	fetchFn := func(db, col string, _ bson.M, _ bson.D, _ int) tea.Cmd {
		fetchCalled = true
		return func() tea.Msg { return msg.DocumentsLoaded{} }
	}
	m := newTestModel(fetchFn, nil, nil, nil)
	m.db = "testdb"
	m.collection = "testcol"
	m.aggMode = true
	m.docs = []bson.M{{"x": 1}}

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m2.aggMode {
		t.Error("aggMode should be false after esc")
	}
	if cmd == nil {
		t.Error("expected reload cmd after esc in aggMode")
	}
	_ = fetchCalled
}

func TestAggregateResult_SetsColumns(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m = m.SetSize(120, 40)
	m.db = "testdb"
	m.collection = "testcol"

	docs := []bson.M{{"_id": "1", "name": "alice"}, {"_id": "2", "name": "bob"}}
	m2, _ := m.Update(msg.AggregateResult{Docs: docs})

	if len(m2.columns) == 0 {
		t.Error("columns should be populated after AggregateResult")
	}
}
