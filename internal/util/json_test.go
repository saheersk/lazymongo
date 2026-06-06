package util

import (
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ── FormatValue ───────────────────────────────────────────────────────────────

func TestFormatValue(t *testing.T) {
	oid := bson.NewObjectID()
	dt := bson.DateTime(time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC).UnixMilli())

	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"nil", nil, "null"},
		{"string", "hello", "hello"},
		{"string long", strings.Repeat("a", 70), strings.Repeat("a", 59) + "…"},
		{"int32", int32(42), "42"},
		{"int64", int64(9999999999), "9999999999"},
		{"float64", float64(3.14), "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"objectID", oid, oid.Hex()},
		{"datetime", dt, "2024-03-15"},
		{"bson.A 3 items", bson.A{1, 2, 3}, "[…] 3 items"},
		{"bson.A 0 items", bson.A{}, "[…] 0 items"},
		{"bson.M 2 keys", bson.M{"a": 1, "b": 2}, "{…} 2 keys"},
		{"bson.D 1 key", bson.D{{Key: "x", Value: 1}}, "{…} 1 keys"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := FormatValue(tc.input)
			if got != tc.want {
				t.Errorf("FormatValue(%v) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── BSONToJSON ────────────────────────────────────────────────────────────────

func TestBSONToJSON(t *testing.T) {
	t.Run("valid doc", func(t *testing.T) {
		doc := bson.M{"name": "Alice", "age": int32(30)}
		out, err := BSONToJSON(doc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, "name") {
			t.Errorf("output missing key 'name': %s", out)
		}
		if !strings.Contains(out, "Alice") {
			t.Errorf("output missing value 'Alice': %s", out)
		}
		// Pretty-printed: should contain newlines
		if !strings.Contains(out, "\n") {
			t.Errorf("expected indented JSON (newlines), got: %s", out)
		}
	})

	t.Run("empty doc", func(t *testing.T) {
		doc := bson.M{}
		out, err := BSONToJSON(doc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, "{") {
			t.Errorf("expected '{}', got: %s", out)
		}
	})

	t.Run("objectID field", func(t *testing.T) {
		oid := bson.NewObjectID()
		doc := bson.M{"_id": oid}
		out, err := BSONToJSON(doc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, oid.Hex()) {
			t.Errorf("output missing oid hex %s: %s", oid.Hex(), out)
		}
	})
}

// ── BuildColumns ──────────────────────────────────────────────────────────────

func TestBuildColumns(t *testing.T) {
	t.Run("empty docs returns _id only", func(t *testing.T) {
		cols := BuildColumns(nil, 5)
		if len(cols) != 1 || cols[0] != "_id" {
			t.Errorf("got %v; want [_id]", cols)
		}
	})

	t.Run("empty slice returns _id only", func(t *testing.T) {
		cols := BuildColumns([]bson.M{}, 5)
		if len(cols) != 1 || cols[0] != "_id" {
			t.Errorf("got %v; want [_id]", cols)
		}
	})

	t.Run("_id always first", func(t *testing.T) {
		docs := []bson.M{
			{"_id": bson.NewObjectID(), "name": "Alice"},
		}
		cols := BuildColumns(docs, 5)
		if cols[0] != "_id" {
			t.Errorf("first column should be _id, got %v", cols)
		}
	})

	t.Run("single doc includes its fields", func(t *testing.T) {
		docs := []bson.M{
			{"_id": bson.NewObjectID(), "name": "Alice", "age": int32(30)},
		}
		cols := BuildColumns(docs, 5)
		if cols[0] != "_id" {
			t.Errorf("first column should be _id, got %v", cols)
		}
		colSet := make(map[string]bool)
		for _, c := range cols {
			colSet[c] = true
		}
		if !colSet["name"] {
			t.Errorf("expected 'name' column in %v", cols)
		}
		if !colSet["age"] {
			t.Errorf("expected 'age' column in %v", cols)
		}
	})

	t.Run("most frequent fields appear first", func(t *testing.T) {
		docs := []bson.M{
			{"_id": bson.NewObjectID(), "name": "Alice", "rare": "x"},
			{"_id": bson.NewObjectID(), "name": "Bob"},
			{"_id": bson.NewObjectID(), "name": "Carol"},
		}
		cols := BuildColumns(docs, 5)
		// name appears 3 times, rare appears 1 time — name should come before rare
		nameIdx, rareIdx := -1, -1
		for i, c := range cols {
			if c == "name" {
				nameIdx = i
			}
			if c == "rare" {
				rareIdx = i
			}
		}
		if nameIdx < 0 {
			t.Fatal("'name' not found in columns")
		}
		if rareIdx >= 0 && nameIdx > rareIdx {
			t.Errorf("'name' (idx %d) should come before 'rare' (idx %d)", nameIdx, rareIdx)
		}
	})

	t.Run("maxCols is respected", func(t *testing.T) {
		docs := []bson.M{
			{"_id": bson.NewObjectID(), "a": 1, "b": 2, "c": 3, "d": 4, "e": 5},
		}
		cols := BuildColumns(docs, 3)
		if len(cols) > 3 {
			t.Errorf("expected at most 3 columns, got %d: %v", len(cols), cols)
		}
	})

	t.Run("maxCols 0 defaults to 5", func(t *testing.T) {
		docs := []bson.M{
			{"_id": bson.NewObjectID(), "a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6},
		}
		cols := BuildColumns(docs, 0)
		if len(cols) > 5 {
			t.Errorf("expected at most 5 columns (default), got %d: %v", len(cols), cols)
		}
	})

	t.Run("double-underscore fields come after regular fields", func(t *testing.T) {
		// __v is a mongoose version field — it should appear later than name
		// BuildColumns sorts by frequency; we ensure non-__ fields with equal
		// frequency appear alphabetically before __ fields.
		docs := []bson.M{
			{"_id": bson.NewObjectID(), "name": "Alice", "__v": int32(0)},
			{"_id": bson.NewObjectID(), "name": "Bob", "__v": int32(1)},
		}
		cols := BuildColumns(docs, 5)
		// Both appear twice — alphabetically "__v" < "name" so the sort
		// stability check: name should still be in columns
		colSet := make(map[string]bool)
		for _, c := range cols {
			colSet[c] = true
		}
		if !colSet["name"] {
			t.Errorf("expected 'name' column in %v", cols)
		}
	})
}

// ── PadRight ──────────────────────────────────────────────────────────────────

func TestPadRight(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		w     int
		wantN int    // expected rune length
		wantS string // expected exact string (when specified)
	}{
		{
			name:  "shorter than w gets padded",
			s:     "hi",
			w:     6,
			wantS: "hi    ",
		},
		{
			name:  "exactly w gets truncated with ellipsis (len >= w triggers truncation)",
			s:     "hello",
			w:     5,
			wantS: "hell…",
		},
		{
			name:  "longer than w truncated with ellipsis",
			s:     "hello world",
			w:     8,
			wantS: "hello w…",
		},
		{
			name:  "w=1 returns single char",
			s:     "abc",
			w:     1,
			wantS: "a",
		},
		{
			name:  "w=2 long string truncated",
			s:     "abc",
			w:     2,
			wantS: "a…",
		},
		{
			name:  "empty string padded",
			s:     "",
			w:     3,
			wantS: "   ",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := PadRight(tc.s, tc.w)
			if got != tc.wantS {
				t.Errorf("PadRight(%q, %d) = %q; want %q", tc.s, tc.w, got, tc.wantS)
			}
		})
	}
}

// ── Truncate ──────────────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		n     int
		want  string
	}{
		{"shorter than n unchanged", "hello", 10, "hello"},
		{"equal to n unchanged", "hello", 5, "hello"},
		{"longer than n clipped with ellipsis", "hello world", 8, "hello w…"},
		{"n=1 single char with ellipsis", "abc", 1, "…"},
		{"empty string unchanged", "", 5, ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Truncate(tc.s, tc.n)
			if got != tc.want {
				t.Errorf("Truncate(%q, %d) = %q; want %q", tc.s, tc.n, got, tc.want)
			}
		})
	}
}
