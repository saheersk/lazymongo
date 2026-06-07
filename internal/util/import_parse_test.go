package util

import (
	"os"
	"path/filepath"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func writeTmp(t *testing.T, ext, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*"+ext)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

// ── JSON array ────────────────────────────────────────────────────────────────

func TestParseImportFile_JSONArray(t *testing.T) {
	path := writeTmp(t, ".json", `[
  {"name": "Alice", "age": 30},
  {"name": "Bob",   "age": 25}
]`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs; want 2", len(docs))
	}
	if docs[0]["name"] != "Alice" {
		t.Errorf("docs[0].name = %v; want Alice", docs[0]["name"])
	}
}

func TestParseImportFile_JSONArray_ExtendedJSON(t *testing.T) {
	path := writeTmp(t, ".json", `[
  {"_id": {"$oid": "507f1f77bcf86cd799439011"}, "val": 1}
]`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("got %d docs; want 1", len(docs))
	}
}

func TestParseImportFile_JSON_Empty(t *testing.T) {
	path := writeTmp(t, ".json", `[]`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestParseImportFile_JSON_Invalid(t *testing.T) {
	path := writeTmp(t, ".json", `[{bad json}]`)
	_, err := ParseImportFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ── JSONL ─────────────────────────────────────────────────────────────────────

func TestParseImportFile_JSONL(t *testing.T) {
	path := writeTmp(t, ".jsonl", `{"name":"Alice"}
{"name":"Bob"}
{"name":"Carol"}
`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("got %d docs; want 3", len(docs))
	}
	if docs[2]["name"] != "Carol" {
		t.Errorf("docs[2].name = %v; want Carol", docs[2]["name"])
	}
}

func TestParseImportFile_JSONL_BlankLines(t *testing.T) {
	path := writeTmp(t, ".jsonl", `{"a":1}

{"b":2}

`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs; want 2", len(docs))
	}
}

func TestParseImportFile_JSONL_BadLine(t *testing.T) {
	path := writeTmp(t, ".jsonl", `{"ok":1}
not json
{"ok":2}`)
	_, err := ParseImportFile(path)
	if err == nil {
		t.Error("expected error for bad line, got nil")
	}
}

func TestParseImportFile_NDJSON(t *testing.T) {
	path := writeTmp(t, ".ndjson", `{"x":1}`+"\n"+`{"x":2}`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs; want 2", len(docs))
	}
}

// ── JSON fallback (object instead of array) ───────────────────────────────────

func TestParseImportFile_JSONFallbackToJSONL(t *testing.T) {
	// A .json file that contains JSONL (not a JSON array) should still parse.
	path := writeTmp(t, ".json", `{"name":"Alice"}
{"name":"Bob"}`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs; want 2", len(docs))
	}
}

// ── CSV ───────────────────────────────────────────────────────────────────────

func TestParseImportFile_CSV(t *testing.T) {
	path := writeTmp(t, ".csv", "name,age,role\nAlice,30,admin\nBob,25,user\n")
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs; want 2", len(docs))
	}
	if docs[0]["name"] != "Alice" {
		t.Errorf("docs[0].name = %v; want Alice", docs[0]["name"])
	}
	if docs[0]["role"] != "admin" {
		t.Errorf("docs[0].role = %v; want admin", docs[0]["role"])
	}
	if docs[1]["age"] != "25" {
		t.Errorf("docs[1].age = %v; want \"25\" (CSV values are always strings)", docs[1]["age"])
	}
}

func TestParseImportFile_CSV_HeaderOnly(t *testing.T) {
	path := writeTmp(t, ".csv", "name,age\n")
	_, err := ParseImportFile(path)
	if err == nil {
		t.Error("expected error for header-only CSV, got nil")
	}
}

func TestParseImportFile_CSV_Quoted(t *testing.T) {
	path := writeTmp(t, ".csv", `name,bio
Alice,"loves go, hates yaml"
Bob,"one liner"
`)
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs; want 2", len(docs))
	}
	if docs[0]["bio"] != "loves go, hates yaml" {
		t.Errorf("quoted CSV field = %q; want 'loves go, hates yaml'", docs[0]["bio"])
	}
}

// ── real testdata files ───────────────────────────────────────────────────────

func TestParseImportFile_SampleUsers(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample-users.json")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("testdata not found: %v", err)
	}
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("ParseImportFile sample-users.json: %v", err)
	}
	if len(docs) == 0 {
		t.Error("expected docs from sample-users.json, got none")
	}
	t.Logf("parsed %d docs from sample-users.json", len(docs))
}

func TestParseImportFile_SampleProducts(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample-products.json")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("testdata not found: %v", err)
	}
	docs, err := ParseImportFile(path)
	if err != nil {
		t.Fatalf("ParseImportFile sample-products.json: %v", err)
	}
	if len(docs) == 0 {
		t.Error("expected docs from sample-products.json, got none")
	}
	t.Logf("parsed %d docs from sample-products.json", len(docs))
}
