package util

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ParseImportFile reads a JSON, JSONL/NDJSON, or CSV file and returns a slice
// of documents ready for InsertMany.
//
// Detection rules:
//   - .csv              → CSV (first row = headers, all values are strings)
//   - .jsonl / .ndjson  → newline-delimited Extended JSON objects
//   - anything else     → JSON array first, then JSONL fallback
func ParseImportFile(path string) ([]bson.M, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return parseCSV(path)
	case ".jsonl", ".ndjson":
		return parseJSONLFile(path)
	default:
		return parseJSONFile(path)
	}
}

func parseJSONFile(path string) ([]bson.M, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	// JSON array
	if trimmed[0] == '[' {
		var docs []bson.M
		if err := bson.UnmarshalExtJSON([]byte(trimmed), false, &docs); err != nil {
			return nil, fmt.Errorf("invalid JSON array: %w", err)
		}
		return docs, nil
	}

	// Fall back to JSONL
	return parseJSONLBytes(trimmed)
}

func parseJSONLFile(path string) ([]bson.M, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseJSONLBytes(strings.TrimSpace(string(data)))
}

func parseJSONLBytes(content string) ([]bson.M, error) {
	var docs []bson.M
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4 MB per line
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var doc bson.M
		if err := bson.UnmarshalExtJSON([]byte(line), false, &doc); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		docs = append(docs, doc)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no documents found in file")
	}
	return docs, nil
}

func parseCSV(path string) ([]bson.M, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("CSV is empty")
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV has headers but no data rows")
	}

	headers := records[0]
	docs := make([]bson.M, 0, len(records)-1)
	for _, row := range records[1:] {
		doc := make(bson.M, len(headers))
		for i, h := range headers {
			if i < len(row) {
				doc[h] = row[i]
			} else {
				doc[h] = ""
			}
		}
		docs = append(docs, doc)
	}
	return docs, nil
}
