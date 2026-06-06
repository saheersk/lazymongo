package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// BSONToJSON converts a bson.M to a pretty-printed relaxed-extended-JSON string.
func BSONToJSON(doc bson.M) (string, error) {
	raw, err := bson.MarshalExtJSON(doc, false, false)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return "", fmt.Errorf("indent: %w", err)
	}
	return buf.String(), nil
}

// FormatValue renders a BSON field value as a concise display string.
func FormatValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch t := v.(type) {
	case string:
		return truncateStr(t, 60)
	case int32:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float64:
		return fmt.Sprintf("%g", t)
	case bool:
		return fmt.Sprintf("%v", t)
	case bson.ObjectID:
		return t.Hex()
	case bson.DateTime:
		return t.Time().UTC().Format(time.DateOnly)
	case bson.A:
		return fmt.Sprintf("[…] %d items", len(t))
	case bson.M:
		return fmt.Sprintf("{…} %d keys", len(t))
	case bson.D:
		return fmt.Sprintf("{…} %d keys", len(t))
	default:
		return truncateStr(fmt.Sprintf("%v", t), 60)
	}
}

// DocPreview returns a compact single-line summary of a document.
func DocPreview(doc bson.M, maxLen int) string {
	keys := make([]string, 0, len(doc))
	for k := range doc {
		if k != "_id" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, FormatValue(doc[k])))
	}
	preview := "{ " + strings.Join(parts, ", ") + " }"
	return truncateStr(preview, maxLen)
}

// BuildColumns picks which fields to show as table columns from a slice of docs.
// _id is always first; remaining slots go to the most-frequent other fields.
func BuildColumns(docs []bson.M, maxCols int) []string {
	if len(docs) == 0 {
		return []string{"_id"}
	}
	if maxCols < 1 {
		maxCols = 5
	}

	freq := map[string]int{}
	for _, doc := range docs {
		for k := range doc {
			if k != "_id" {
				freq[k]++
			}
		}
	}

	type kv struct {
		k string
		v int
	}
	ranked := make([]kv, 0, len(freq))
	for k, v := range freq {
		ranked = append(ranked, kv{k, v})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].v != ranked[j].v {
			return ranked[i].v > ranked[j].v
		}
		return ranked[i].k < ranked[j].k
	})

	cols := []string{"_id"}
	for i := 0; i < len(ranked) && len(cols) < maxCols; i++ {
		cols = append(cols, ranked[i].k)
	}
	return cols
}

// Truncate clips s to n runes, appending "…" if clipped.
func Truncate(s string, n int) string { return truncateStr(s, n) }

// PadRight right-pads s with spaces to width w, truncating if needed.
func PadRight(s string, w int) string {
	runes := []rune(s)
	if len(runes) >= w {
		if w <= 1 {
			return string(runes[:w])
		}
		return string(runes[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(runes))
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
