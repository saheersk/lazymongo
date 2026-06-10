package documents

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/saheersk/lazymongo/internal/tui/msg"
)

// ── operator autocomplete ─────────────────────────────────────────────────────

func TestFilterFieldComplete_OperatorPrefix(t *testing.T) {
	input, matches := filterFieldComplete(`{"age": {"$gt`, []string{"age", "name"})
	if len(matches) != 2 {
		t.Fatalf("expected $gt and $gte, got %v", matches)
	}
	if input != `{"age": {"$gt` {
		t.Errorf("input should be unchanged at the $gt/$gte fork, got %q", input)
	}
}

func TestFilterFieldComplete_OperatorLCP(t *testing.T) {
	input, matches := filterFieldComplete(`{"age": {"$reg`, nil)
	if !reflect.DeepEqual(matches, []string{"$regex"}) {
		t.Fatalf("expected [$regex], got %v", matches)
	}
	if !strings.HasSuffix(input, "$regex") {
		t.Errorf("input should be completed to $regex, got %q", input)
	}
}

func TestFilterFieldComplete_OperatorsWorkWithoutFields(t *testing.T) {
	_, matches := filterFieldComplete(`{$e`, nil)
	if len(matches) == 0 {
		t.Fatal("operator completion must work even when no field names are known")
	}
	for _, m := range matches {
		if !strings.HasPrefix(m, "$e") {
			t.Errorf("match %q does not share the $e prefix", m)
		}
	}
}

func TestFilterFieldComplete_FieldsStillWork(t *testing.T) {
	input, matches := filterFieldComplete(`{na`, []string{"name", "nationality"})
	if len(matches) != 2 {
		t.Fatalf("expected 2 field matches, got %v", matches)
	}
	if input != `{na` {
		t.Errorf("LCP of name/nationality is 'na', input should be unchanged, got %q", input)
	}
}

// ── shared history helper ─────────────────────────────────────────────────────

func TestPushHistory_PrependsDedupesAndCaps(t *testing.T) {
	h := []string{"b", "a"}
	h = pushHistory(h, "a", 3)
	if !reflect.DeepEqual(h, []string{"a", "b"}) {
		t.Fatalf("dedupe+prepend failed: %v", h)
	}
	h = pushHistory(h, "c", 3)
	h = pushHistory(h, "d", 3)
	if !reflect.DeepEqual(h, []string{"d", "c", "a"}) {
		t.Fatalf("cap at 3 failed: %v", h)
	}
	if got := pushHistory(h, "", 3); !reflect.DeepEqual(got, h) {
		t.Errorf("empty entry must be a no-op, got %v", got)
	}
}

// ── pipeline history picker ───────────────────────────────────────────────────

func TestAggregateResult_PushesHistory(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m, _ = m.Update(msg.AggregateResult{PipelineText: `[{"$match": {}}]`})
	if len(m.aggHistory) != 1 || m.aggHistory[0] != `[{"$match": {}}]` {
		t.Fatalf("aggHistory not updated: %v", m.aggHistory)
	}
}

func TestAggKey_WithHistory_OpensPicker(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db, m.collection = "db", "col"
	m.aggHistory = []string{`[{"$match": {}}]`}

	m, cmd := pressKey(m, "a")
	if !m.aggPick {
		t.Fatal("expected picker to open when history exists")
	}
	if cmd != nil {
		t.Error("picker open should not fire a command")
	}
	if !m.InInputMode() {
		t.Error("picker must count as input mode so global keys are bypassed")
	}
}

func TestAggPick_NavigateAndCancel(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db, m.collection = "db", "col"
	m.aggHistory = []string{"p1", "p2"}
	m, _ = pressKey(m, "a")

	m, _ = pressKey(m, "j")
	if m.aggPickIdx != 1 {
		t.Fatalf("j should move to 1, got %d", m.aggPickIdx)
	}
	m, _ = pressKey(m, "k")
	if m.aggPickIdx != 0 {
		t.Fatalf("k should move back to 0, got %d", m.aggPickIdx)
	}
	// wrap upward from 0 → last item (2 history + 1 "new")
	m, _ = pressKey(m, "k")
	if m.aggPickIdx != 2 {
		t.Fatalf("k at top should wrap to 2, got %d", m.aggPickIdx)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.aggPick {
		t.Fatal("esc should close the picker")
	}
}

func TestAggPick_EnterOpensEditor(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db, m.collection = "db", "col"
	m.aggHistory = []string{"p1"}
	m, _ = pressKey(m, "a")

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.aggPick {
		t.Fatal("enter should close the picker")
	}
	if cmd == nil {
		t.Fatal("enter should open the editor (non-nil cmd)")
	}
}

func TestAggKey_InAggMode_SkipsPicker(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db, m.collection = "db", "col"
	m.aggMode = true
	m.aggHistory = []string{"p1"}

	m, cmd := pressKey(m, "a")
	if m.aggPick {
		t.Fatal("'a' in agg mode should re-edit directly, not open the picker")
	}
	if cmd == nil {
		t.Fatal("'a' in agg mode should open the editor")
	}
}

// ── agg-mode guards ───────────────────────────────────────────────────────────

func TestAggMode_FilterKeyShowsStatus(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.db, m.collection = "db", "col"
	m.aggMode = true

	m, cmd := pressKey(m, "/")
	if m.mode == modeFilter {
		t.Fatal("filter bar must not open in agg mode")
	}
	assertStatus(t, cmd)
}

func TestAggMode_EditKeysShowStatus(t *testing.T) {
	for _, k := range []string{"n", "e", "d", "c", " ", "s", "r"} {
		m := newTestModel(nil, nil, nil, nil)
		m.db, m.collection = "db", "col"
		m.aggMode = true

		var cmd tea.Cmd
		m, cmd = pressKey(m, k)
		if m.deleteConfirm || m.mode != modeNone {
			t.Fatalf("key %q must not enter an action mode in agg mode", k)
		}
		assertStatus(t, cmd)
	}
}

func assertStatus(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a StatusUpdate cmd, got nil")
	}
	if _, ok := cmd().(msg.StatusUpdate); !ok {
		t.Fatalf("expected StatusUpdate, got %T", cmd())
	}
}
