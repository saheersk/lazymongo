package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"github.com/saheersk/lazymongo/internal/tui/style"
)

// base is a blank screen used as the backdrop for all overlay tests.
const testBase = ""

func testTheme() *style.Theme { return style.Default() }

// ── renderExplain ─────────────────────────────────────────────────────────────

func TestRenderExplain_Loading(t *testing.T) {
	out := renderExplain(testBase, 120, 40, testTheme(), true, nil)
	if !strings.Contains(out, "running explain") {
		t.Errorf("loading state should contain 'running explain', got:\n%s", out)
	}
}

func TestRenderExplain_Error(t *testing.T) {
	stats := &msg.ExplainStats{Err: errors.New("connection reset")}
	out := renderExplain(testBase, 120, 40, testTheme(), false, stats)
	if !strings.Contains(out, "connection reset") {
		t.Errorf("error state should contain error text, got:\n%s", out)
	}
}

func TestRenderExplain_CollScan_ShowsWarning(t *testing.T) {
	stats := &msg.ExplainStats{
		DB:              "mydb",
		Col:             "users",
		IndexUsed:       "", // COLLSCAN
		NReturned:       5,
		DocsExamined:    1000,
		KeysExamined:    0,
		ExecutionTimeMs: 12,
	}
	out := renderExplain(testBase, 120, 40, testTheme(), false, stats)
	if !strings.Contains(out, "COLLSCAN") {
		t.Errorf("COLLSCAN should appear in output, got:\n%s", out)
	}
	if !strings.Contains(out, "full collection scan") {
		t.Errorf("full scan warning should appear, got:\n%s", out)
	}
	if !strings.Contains(out, "mydb.users") {
		t.Errorf("collection name should appear, got:\n%s", out)
	}
	if !strings.Contains(out, "12 ms") {
		t.Errorf("execution time should appear, got:\n%s", out)
	}
}

func TestRenderExplain_IxScan_ShowsIndexName(t *testing.T) {
	stats := &msg.ExplainStats{
		DB:              "mydb",
		Col:             "orders",
		IndexUsed:       "status_1",
		NReturned:       10,
		DocsExamined:    10,
		KeysExamined:    10,
		ExecutionTimeMs: 2,
	}
	out := renderExplain(testBase, 120, 40, testTheme(), false, stats)
	if !strings.Contains(out, "IXSCAN") {
		t.Errorf("IXSCAN should appear in output, got:\n%s", out)
	}
	if !strings.Contains(out, "status_1") {
		t.Errorf("index name should appear, got:\n%s", out)
	}
	if !strings.Contains(out, "efficient") {
		t.Errorf("efficiency note should appear for good scan, got:\n%s", out)
	}
}

func TestRenderExplain_LowSelectivity_ShowsWarning(t *testing.T) {
	stats := &msg.ExplainStats{
		DB:           "mydb",
		Col:          "logs",
		IndexUsed:    "ts_1",
		NReturned:    1,
		DocsExamined: 500, // 500x ratio → bad selectivity
	}
	out := renderExplain(testBase, 120, 40, testTheme(), false, stats)
	if !strings.Contains(out, "selectivity") {
		t.Errorf("low selectivity warning should appear, got:\n%s", out)
	}
}

func TestRenderExplain_ClosingHint(t *testing.T) {
	out := renderExplain(testBase, 120, 40, testTheme(), true, nil)
	if !strings.Contains(out, "press any key") {
		t.Errorf("closing hint should always appear, got:\n%s", out)
	}
}

// ── renderSchema ──────────────────────────────────────────────────────────────

func TestRenderSchema_Loading(t *testing.T) {
	out := renderSchema(testBase, 120, 40, testTheme(), true, nil, 0)
	if !strings.Contains(out, "sampling") {
		t.Errorf("loading state should contain 'sampling', got:\n%s", out)
	}
}

func TestRenderSchema_Error(t *testing.T) {
	result := &msg.SchemaResult{Err: errors.New("timeout after 30s")}
	out := renderSchema(testBase, 120, 40, testTheme(), false, result, 0)
	if !strings.Contains(out, "timeout after 30s") {
		t.Errorf("error text should appear, got:\n%s", out)
	}
}

func TestRenderSchema_ShowsFields(t *testing.T) {
	result := &msg.SchemaResult{
		DB:         "mydb",
		Col:        "users",
		SampleSize: 50,
		Fields: []msg.SchemaField{
			{Name: "_id", Types: []msg.TypeFreq{{Type: "objectId", Count: 50}}, Count: 50},
			{Name: "name", Types: []msg.TypeFreq{{Type: "string", Count: 48}}, Count: 48},
			{Name: "age", Types: []msg.TypeFreq{{Type: "int32", Count: 40}}, Count: 40},
		},
	}
	out := renderSchema(testBase, 120, 40, testTheme(), false, result, 0)
	for _, want := range []string{"_id", "name", "age", "objectId", "string", "int32", "mydb", "users", "50"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in schema output, got:\n%s", want, out)
		}
	}
}

func TestRenderSchema_ShowsPresencePercentage(t *testing.T) {
	result := &msg.SchemaResult{
		DB:         "mydb",
		Col:        "col",
		SampleSize: 100,
		Fields: []msg.SchemaField{
			{Name: "email", Types: []msg.TypeFreq{{Type: "string", Count: 75}}, Count: 75},
		},
	}
	out := renderSchema(testBase, 120, 40, testTheme(), false, result, 0)
	if !strings.Contains(out, "75%") {
		t.Errorf("presence percentage should appear, got:\n%s", out)
	}
}

func TestRenderSchema_ScrollHint_WhenManyFields(t *testing.T) {
	fields := make([]msg.SchemaField, 30)
	for i := range fields {
		fields[i] = msg.SchemaField{
			Name:  strings.Repeat("f", i+1),
			Types: []msg.TypeFreq{{Type: "string", Count: 10}},
			Count: 10,
		}
	}
	result := &msg.SchemaResult{DB: "db", Col: "c", SampleSize: 10, Fields: fields}
	// Small terminal height forces scroll hint.
	out := renderSchema(testBase, 120, 25, testTheme(), false, result, 0)
	if !strings.Contains(out, "scroll") {
		t.Errorf("scroll hint should appear when fields > visible, got:\n%s", out)
	}
}

func TestRenderSchema_Scroll_ShowsDifferentFields(t *testing.T) {
	fields := make([]msg.SchemaField, 10)
	for i := range fields {
		fields[i] = msg.SchemaField{
			Name:  strings.Repeat(string(rune('a'+i)), 3),
			Types: []msg.TypeFreq{{Type: "string", Count: 5}},
			Count: 5,
		}
	}
	result := &msg.SchemaResult{DB: "db", Col: "c", SampleSize: 5, Fields: fields}

	// Scroll=0 should show first field.
	out0 := renderSchema(testBase, 120, 20, testTheme(), false, result, 0)
	if !strings.Contains(out0, "aaa") {
		t.Errorf("scroll=0 should show first field 'aaa', got:\n%s", out0)
	}

	// Scroll=5 should show field at index 5, not the first one.
	out5 := renderSchema(testBase, 120, 20, testTheme(), false, result, 5)
	if strings.Contains(out5, "aaa") {
		t.Errorf("scroll=5 should NOT show first field 'aaa', got:\n%s", out5)
	}
}

func TestRenderSchema_NoFields(t *testing.T) {
	result := &msg.SchemaResult{DB: "db", Col: "empty", SampleSize: 0, Fields: nil}
	out := renderSchema(testBase, 120, 40, testTheme(), false, result, 0)
	if !strings.Contains(out, "no fields") {
		t.Errorf("empty result should say 'no fields', got:\n%s", out)
	}
}

// ── renderImport ──────────────────────────────────────────────────────────────

func TestRenderImport_ShowsFormatHints(t *testing.T) {
	out := renderImport(testBase, 120, 40, testTheme(), "", false, nil, nil)
	for _, want := range []string{".json", ".jsonl", ".csv", "File path"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in import overlay, got:\n%s", want, out)
		}
	}
}

func TestRenderImport_ShowsActionHint_WhenIdle(t *testing.T) {
	out := renderImport(testBase, 120, 40, testTheme(), "", false, nil, nil)
	if !strings.Contains(out, "enter import") {
		t.Errorf("idle state should show 'enter import', got:\n%s", out)
	}
}

func TestRenderImport_Loading_ShowsLoadingText(t *testing.T) {
	out := renderImport(testBase, 120, 40, testTheme(), "/data/docs.json", true, nil, nil)
	if !strings.Contains(out, "importing") {
		t.Errorf("loading state should show 'importing', got:\n%s", out)
	}
}

func TestRenderImport_Error_ShowsErrorAndRetryHint(t *testing.T) {
	err := errors.New("no such file or directory")
	out := renderImport(testBase, 120, 40, testTheme(), "/bad/path.json", false, err, nil)
	if !strings.Contains(out, "no such file") {
		t.Errorf("error text should appear, got:\n%s", out)
	}
	if !strings.Contains(out, "try again") {
		t.Errorf("retry hint should appear on error, got:\n%s", out)
	}
}

func TestRenderImport_LongError_IsTruncated(t *testing.T) {
	longErr := errors.New(strings.Repeat("x", 200))
	out := renderImport(testBase, 120, 40, testTheme(), "/bad/path.json", false, longErr, nil)
	// Should not panic and should contain the overlay structure.
	if !strings.Contains(out, "error") {
		t.Errorf("long error should still show error label, got:\n%s", out)
	}
}

func TestRenderImport_Completions_ShowsUpToThree(t *testing.T) {
	completions := []string{"/tmp/a.json", "/tmp/b.json", "/tmp/c.json", "/tmp/d.json", "/tmp/e.json"}
	out := renderImport(testBase, 120, 40, testTheme(), "/tmp/", false, nil, completions)
	// Should show 3 names.
	if !strings.Contains(out, "a.json") || !strings.Contains(out, "b.json") || !strings.Contains(out, "c.json") {
		t.Errorf("first 3 completions should appear, got:\n%s", out)
	}
	// Should show "more" hint for the extra 2.
	if !strings.Contains(out, "2 more") {
		t.Errorf("should show '2 more' for overflow completions, got:\n%s", out)
	}
	// Should NOT show d.json or e.json directly.
	if strings.Contains(out, "d.json") || strings.Contains(out, "e.json") {
		t.Errorf("4th/5th completion should be hidden behind '... N more', got:\n%s", out)
	}
}

func TestRenderImport_Completions_ExactlyThree_NoMore(t *testing.T) {
	completions := []string{"/tmp/a.json", "/tmp/b.json", "/tmp/c.json"}
	out := renderImport(testBase, 120, 40, testTheme(), "/tmp/", false, nil, completions)
	if !strings.Contains(out, "a.json") {
		t.Errorf("completions should appear, got:\n%s", out)
	}
	if strings.Contains(out, "more") {
		t.Errorf("should NOT show 'more' when exactly 3 completions, got:\n%s", out)
	}
}

func TestRenderImport_TabHint_AlwaysShown(t *testing.T) {
	out := renderImport(testBase, 120, 40, testTheme(), "", false, nil, nil)
	if !strings.Contains(out, "tab") {
		t.Errorf("tab hint should always appear in import overlay, got:\n%s", out)
	}
}

// ── tabCompleteImportPath ─────────────────────────────────────────────────────

// mkCompletionDir creates a temp dir with the given entries (suffix "/" = dir).
func mkCompletionDir(t *testing.T, entries []string) string {
	t.Helper()
	dir := t.TempDir()
	for _, e := range entries {
		if strings.HasSuffix(e, "/") {
			if err := os.Mkdir(filepath.Join(dir, strings.TrimSuffix(e, "/")), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", e, err)
			}
		} else {
			if err := os.WriteFile(filepath.Join(dir, e), []byte{}, 0o644); err != nil {
				t.Fatalf("create %s: %v", e, err)
			}
		}
	}
	return dir
}

func TestTabComplete_SingleMatch_Completes(t *testing.T) {
	dir := mkCompletionDir(t, []string{"users.json"})
	input := filepath.Join(dir, "users")

	completed, matches := tabCompleteImportPath(input)

	if len(matches) != 1 {
		t.Fatalf("matches = %v; want 1", matches)
	}
	if !strings.HasSuffix(completed, "users.json") {
		t.Errorf("completed = %q; want suffix 'users.json'", completed)
	}
}

func TestTabComplete_DirectoryMatch_AddsTrailingSlash(t *testing.T) {
	dir := mkCompletionDir(t, []string{"exports/"})
	input := filepath.Join(dir, "exp")

	completed, matches := tabCompleteImportPath(input)

	if len(matches) != 1 {
		t.Fatalf("matches = %v; want 1", matches)
	}
	if !strings.HasSuffix(completed, "exports/") {
		t.Errorf("completed = %q; directory should end with '/'", completed)
	}
}

func TestTabComplete_MultipleMatches_AdvancesToCommonPrefix(t *testing.T) {
	dir := mkCompletionDir(t, []string{"users.json", "users.csv", "orders.json"})
	input := filepath.Join(dir, "users")

	completed, matches := tabCompleteImportPath(input)

	// Both users.json and users.csv match — 2 matches.
	if len(matches) != 2 {
		t.Fatalf("matches = %v; want 2", matches)
	}
	// Common prefix is "users." — should advance input to that.
	if !strings.HasSuffix(completed, "users.") {
		t.Errorf("completed = %q; want common prefix 'users.'", completed)
	}
}

func TestTabComplete_MultipleMatches_ReturnsAll(t *testing.T) {
	dir := mkCompletionDir(t, []string{"a.json", "b.json", "c.json", "d.json"})
	// Trailing slash: list all entries.
	input := dir + "/"

	_, matches := tabCompleteImportPath(input)

	if len(matches) != 4 {
		t.Errorf("matches = %v; want 4", matches)
	}
}

func TestTabComplete_NoMatch_ReturnsInputUnchanged(t *testing.T) {
	dir := mkCompletionDir(t, []string{"users.json"})
	input := filepath.Join(dir, "zzz")

	completed, matches := tabCompleteImportPath(input)

	if len(matches) != 0 {
		t.Errorf("matches = %v; want 0", matches)
	}
	if completed != input {
		t.Errorf("completed = %q; want input unchanged %q", completed, input)
	}
}

func TestTabComplete_BadDir_ReturnsInputUnchanged(t *testing.T) {
	input := "/nonexistent/path/prefix"

	completed, matches := tabCompleteImportPath(input)

	if len(matches) != 0 {
		t.Errorf("matches = %v; want 0", matches)
	}
	if completed != input {
		t.Errorf("completed = %q; want input unchanged", completed)
	}
}

func TestTabComplete_Tilde_ExpandsAndRestores(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("~/ expansion uses Unix path separators; not applicable on Windows")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	// If ~/Downloads exists, tabbing "~/Down" should find it.
	dl := filepath.Join(home, "Downloads")
	if _, err := os.Stat(dl); os.IsNotExist(err) {
		t.Skip("~/Downloads not present on this system")
	}

	completed, matches := tabCompleteImportPath("~/Down")

	if len(matches) == 0 {
		t.Fatal("expected at least one match for ~/Down")
	}
	if !strings.HasPrefix(completed, "~/") {
		t.Errorf("completed = %q; should keep ~ prefix", completed)
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		a, b, want string
	}{
		{"abc", "abd", "ab"},
		{"abc", "abc", "abc"},
		{"abc", "xyz", ""},
		{"", "abc", ""},
		{"/usr/local/bin", "/usr/local/lib", "/usr/local/"},
	}
	for _, tc := range tests {
		got := longestCommonPrefix(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("longestCommonPrefix(%q, %q) = %q; want %q", tc.a, tc.b, got, tc.want)
		}
	}
}
