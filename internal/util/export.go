package util

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ToJSON serialises docs as a pretty-printed JSON array.
func ToJSON(docs []bson.M) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("[\n")
	for i, doc := range docs {
		raw, err := bson.MarshalExtJSON(doc, false, false)
		if err != nil {
			return nil, fmt.Errorf("marshal doc %d: %w", i, err)
		}
		buf.WriteString("  ")
		buf.Write(raw)
		if i < len(docs)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("]\n")
	return buf.Bytes(), nil
}

// ToCSV serialises docs as CSV with a header row.
// columns specifies the column order; if empty, all keys from the first doc are used.
func ToCSV(docs []bson.M, columns []string) ([]byte, error) {
	if len(docs) == 0 {
		return []byte{}, nil
	}

	// If no columns provided, derive from first doc.
	if len(columns) == 0 {
		columns = BuildColumns(docs, 20)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Header row.
	if err := w.Write(columns); err != nil {
		return nil, fmt.Errorf("write CSV header: %w", err)
	}

	// Data rows.
	row := make([]string, len(columns))
	for _, doc := range docs {
		for i, col := range columns {
			row[i] = FormatValue(doc[col])
		}
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("write CSV row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush CSV: %w", err)
	}
	return buf.Bytes(), nil
}
