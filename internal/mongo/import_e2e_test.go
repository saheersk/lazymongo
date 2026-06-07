package mongo

import (
	"fmt"
	"os"
	"testing"

	"github.com/saheersk/lazymongo/internal/util"
)

const importE2ECol = "test_import_e2e"

// writeTemp writes content to a temp file with the given extension and returns its path.
func writeTemp(t *testing.T, ext, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "lazymongo_import_*"+ext)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_ = f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
	return f.Name()
}

func dropImportE2ECol(t *testing.T) {
	t.Helper()
	ctx, cancel := opCtx()
	defer cancel()
	_ = mongoClient.inner.Database(crudTestDB).Collection(importE2ECol).Drop(ctx)
}

// TestImportE2E_JSONArray writes a JSON array file, parses it, inserts into
// MongoDB and verifies the document count.
func TestImportE2E_JSONArray(t *testing.T) {
	skipIfNoMongo(t)
	dropImportE2ECol(t)

	content := `[
		{"name":"Alice","role":"admin"},
		{"name":"Bob","role":"user"},
		{"name":"Carol","role":"user"}
	]`
	path := writeTemp(t, ".json", content)

	docs, err := util.ParseImportFile(path)
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("parsed %d docs; want 3", len(docs))
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importE2ECol, docs)
	if len(errs) != 0 {
		t.Fatalf("InsertMany errors: %v", errs)
	}
	if inserted != 3 {
		t.Errorf("inserted = %d; want 3", inserted)
	}

	result, err := mongoClient.FindPage(crudTestDB, importE2ECol, QueryOptions{PageSize: 10})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("total in collection = %d; want 3", result.Total)
	}
}

// TestImportE2E_JSONL writes a JSONL file and verifies round-trip.
func TestImportE2E_JSONL(t *testing.T) {
	skipIfNoMongo(t)
	dropImportE2ECol(t)

	content := `{"product":"laptop","price":999}
{"product":"mouse","price":29}
{"product":"keyboard","price":79}
{"product":"monitor","price":399}
`
	path := writeTemp(t, ".jsonl", content)

	docs, err := util.ParseImportFile(path)
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(docs) != 4 {
		t.Fatalf("parsed %d docs; want 4", len(docs))
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importE2ECol, docs)
	if len(errs) != 0 {
		t.Fatalf("InsertMany errors: %v", errs)
	}
	if inserted != 4 {
		t.Errorf("inserted = %d; want 4", inserted)
	}

	result, err := mongoClient.FindPage(crudTestDB, importE2ECol, QueryOptions{PageSize: 10})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if result.Total != 4 {
		t.Errorf("total in collection = %d; want 4", result.Total)
	}
}

// TestImportE2E_CSV writes a CSV file and verifies round-trip.
func TestImportE2E_CSV(t *testing.T) {
	skipIfNoMongo(t)
	dropImportE2ECol(t)

	content := "username,email,city\nalice,alice@example.com,London\nbob,bob@example.com,Paris\n"
	path := writeTemp(t, ".csv", content)

	docs, err := util.ParseImportFile(path)
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("parsed %d docs; want 2", len(docs))
	}

	// Verify field mapping from CSV headers.
	if docs[0]["username"] != "alice" {
		t.Errorf("docs[0][username] = %v; want 'alice'", docs[0]["username"])
	}
	if docs[0]["email"] != "alice@example.com" {
		t.Errorf("docs[0][email] = %v; want 'alice@example.com'", docs[0]["email"])
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importE2ECol, docs)
	if len(errs) != 0 {
		t.Fatalf("InsertMany errors: %v", errs)
	}
	if inserted != 2 {
		t.Errorf("inserted = %d; want 2", inserted)
	}
}

// TestImportE2E_NDJSON verifies the .ndjson extension is handled as JSONL.
func TestImportE2E_NDJSON(t *testing.T) {
	skipIfNoMongo(t)
	dropImportE2ECol(t)

	content := `{"event":"login","user":"alice"}
{"event":"logout","user":"alice"}
`
	path := writeTemp(t, ".ndjson", content)

	docs, err := util.ParseImportFile(path)
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("parsed %d docs; want 2", len(docs))
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importE2ECol, docs)
	if len(errs) != 0 {
		t.Fatalf("InsertMany errors: %v", errs)
	}
	if inserted != 2 {
		t.Errorf("inserted = %d; want 2", inserted)
	}
}

// TestImportE2E_LargeFile verifies batching for > 500 docs.
func TestImportE2E_LargeFile(t *testing.T) {
	skipIfNoMongo(t)
	dropImportE2ECol(t)

	// Build 750 docs (forces 2 batches: 500 + 250).
	const count = 750
	var lines string
	for i := 0; i < count; i++ {
		lines += fmt.Sprintf(`{"seq":%d,"tag":"bulk"}`, i) + "\n"
	}
	path := writeTemp(t, ".jsonl", lines)

	docs, err := util.ParseImportFile(path)
	if err != nil {
		t.Fatalf("ParseImportFile: %v", err)
	}
	if len(docs) != count {
		t.Fatalf("parsed %d docs; want %d", len(docs), count)
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importE2ECol, docs)
	if len(errs) != 0 {
		t.Fatalf("InsertMany errors on large file: %v", errs)
	}
	if inserted != count {
		t.Errorf("inserted = %d; want %d", inserted, count)
	}

	result, err := mongoClient.FindPage(crudTestDB, importE2ECol, QueryOptions{PageSize: 10})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if result.Total != int64(count) {
		t.Errorf("total = %d; want %d", result.Total, count)
	}
}

// TestImportE2E_BadFile verifies that a corrupt file is rejected at parse time.
func TestImportE2E_BadFile(t *testing.T) {
	path := writeTemp(t, ".json", `{not valid json}`)

	docs, err := util.ParseImportFile(path)
	if err == nil {
		t.Errorf("expected parse error for corrupt file, got %d docs", len(docs))
	}
}

// TestImportE2E_EmptyFile verifies that an empty file is rejected at parse time.
func TestImportE2E_EmptyFile(t *testing.T) {
	path := writeTemp(t, ".json", "")

	docs, err := util.ParseImportFile(path)
	if err == nil {
		t.Errorf("expected parse error for empty file, got %d docs", len(docs))
	}
}
