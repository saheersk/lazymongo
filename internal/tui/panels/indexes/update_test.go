package indexes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexTemplateFor_Empty(t *testing.T) {
	if got := indexTemplateFor(nil); got != indexTemplate {
		t.Errorf("nil fields should return the default template, got %q", got)
	}
}

func TestIndexTemplateFor_Fields(t *testing.T) {
	got := indexTemplateFor([]string{"age", "status"})
	if !strings.Contains(got, `"age": 1`) || !strings.Contains(got, `"status": 1`) {
		t.Errorf("template missing pre-filled keys: %q", got)
	}
	if !strings.Contains(got, `"ttlSeconds": -1`) {
		t.Errorf("template missing ttlSeconds: %q", got)
	}
}

func writeTempIndexFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadIndexFile_TTL(t *testing.T) {
	done := readIndexFile(writeTempIndexFile(t,
		`{"keys": {"createdAt": 1}, "unique": false, "sparse": false, "ttlSeconds": 3600}`))
	if done.Err != nil {
		t.Fatalf("unexpected error: %v", done.Err)
	}
	if done.TTLSeconds != 3600 {
		t.Errorf("TTLSeconds = %d, want 3600", done.TTLSeconds)
	}
}

func TestReadIndexFile_TTLDisabled(t *testing.T) {
	for name, content := range map[string]string{
		"explicit -1": `{"keys": {"a": 1}, "ttlSeconds": -1}`,
		"omitted":     `{"keys": {"a": 1}, "unique": true}`,
	} {
		done := readIndexFile(writeTempIndexFile(t, content))
		if done.Err != nil {
			t.Fatalf("%s: unexpected error: %v", name, done.Err)
		}
		if done.TTLSeconds != -1 {
			t.Errorf("%s: TTLSeconds = %d, want -1 (disabled)", name, done.TTLSeconds)
		}
	}
}

func TestReadIndexFile_TTLZeroIsValid(t *testing.T) {
	done := readIndexFile(writeTempIndexFile(t, `{"keys": {"a": 1}, "ttlSeconds": 0}`))
	if done.Err != nil {
		t.Fatalf("unexpected error: %v", done.Err)
	}
	if done.TTLSeconds != 0 {
		t.Errorf("explicit 0 must be kept (expire immediately), got %d", done.TTLSeconds)
	}
}

func TestReadIndexFile_EmptyKeysRejected(t *testing.T) {
	done := readIndexFile(writeTempIndexFile(t, `{"keys": {}, "unique": false}`))
	if done.Err == nil {
		t.Fatal("empty keys must be rejected")
	}
}
