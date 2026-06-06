package util

import (
	"regexp"
	"strings"
	"testing"

	"github.com/saheersk/lazymongo/internal/tui/style"
)

// stripANSI removes ANSI escape sequences from a string so we can inspect
// the plain-text content of highlighted output.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func TestSyntaxHighlight(t *testing.T) {
	th := style.Default()

	t.Run("empty string returns empty string", func(t *testing.T) {
		got := SyntaxHighlight("", th)
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("non-empty input produces non-empty output", func(t *testing.T) {
		input := `{"name": "Alice"}`
		got := SyntaxHighlight(input, th)
		if got == "" {
			t.Error("expected non-empty output")
		}
	})

	t.Run("plain text still contains original keys", func(t *testing.T) {
		input := `{
  "name": "Alice",
  "age": 30
}`
		got := SyntaxHighlight(input, th)
		plain := stripANSI(got)

		for _, token := range []string{"name", "Alice", "age", "30"} {
			if !strings.Contains(plain, token) {
				t.Errorf("stripped output missing %q\nraw output: %q", token, got)
			}
		}
	})

	t.Run("does not panic on malformed JSON", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SyntaxHighlight panicked: %v", r)
			}
		}()
		malformed := `{"key": "unclosed string`
		_ = SyntaxHighlight(malformed, th)
	})

	t.Run("does not panic on deeply nested JSON", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SyntaxHighlight panicked: %v", r)
			}
		}()
		nested := `{"a": {"b": {"c": [1, 2, null, true, false]}}}`
		got := SyntaxHighlight(nested, th)
		plain := stripANSI(got)
		if !strings.Contains(plain, "null") {
			t.Errorf("expected 'null' in output, got: %s", plain)
		}
	})

	t.Run("boolean values highlighted", func(t *testing.T) {
		input := `{
  "active": true,
  "deleted": false
}`
		got := SyntaxHighlight(input, th)
		plain := stripANSI(got)
		if !strings.Contains(plain, "true") {
			t.Errorf("expected 'true' in output: %s", plain)
		}
		if !strings.Contains(plain, "false") {
			t.Errorf("expected 'false' in output: %s", plain)
		}
	})

	t.Run("number values highlighted", func(t *testing.T) {
		input := `{
  "count": 42,
  "pi": 3.14
}`
		got := SyntaxHighlight(input, th)
		plain := stripANSI(got)
		if !strings.Contains(plain, "42") {
			t.Errorf("expected '42' in output: %s", plain)
		}
	})

	t.Run("null value highlighted", func(t *testing.T) {
		input := `{
  "field": null
}`
		got := SyntaxHighlight(input, th)
		plain := stripANSI(got)
		if !strings.Contains(plain, "null") {
			t.Errorf("expected 'null' in output: %s", plain)
		}
	})

	t.Run("ObjectID extended JSON highlighted", func(t *testing.T) {
		input := `{
  "_id": {"$oid": "507f1f77bcf86cd799439011"}
}`
		got := SyntaxHighlight(input, th)
		plain := stripANSI(got)
		if !strings.Contains(plain, "507f1f77bcf86cd799439011") {
			t.Errorf("expected OID hex in output: %s", plain)
		}
	})

	t.Run("multi-line input produces multi-line output", func(t *testing.T) {
		input := "{\n  \"key\": \"value\"\n}"
		got := SyntaxHighlight(input, th)
		if !strings.Contains(got, "\n") {
			t.Errorf("expected newlines preserved, got: %q", got)
		}
	})
}
