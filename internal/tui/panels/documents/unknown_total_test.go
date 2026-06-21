package documents

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// docsOf returns n stub documents.
func docsOf(n int) []bson.M {
	out := make([]bson.M, n)
	for i := range out {
		out[i] = bson.M{"i": i}
	}
	return out
}

// Total = -1 means the count timed out (huge collection). Paging forward must
// stay possible while the current page is full, and stop on a partial page.
func TestPageCount_UnknownTotal(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)
	m.total = -1

	m.page = 0
	m.docs = docsOf(m.pageSize) // full page → assume there's more
	if got := m.pageCount(); got != 2 {
		t.Errorf("full page: pageCount = %d, want 2", got)
	}

	m.page = 3
	m.docs = docsOf(m.pageSize - 1) // partial page → this is the last one
	if got := m.pageCount(); got != 4 {
		t.Errorf("partial page: pageCount = %d, want 4", got)
	}
}

func TestPageCount_KnownTotalUnchanged(t *testing.T) {
	m := newTestModel(nil, nil, nil, nil)

	m.total = 0
	if got := m.pageCount(); got != 1 {
		t.Errorf("empty: pageCount = %d, want 1", got)
	}

	m.total = 120 // 50/pg → 3 pages
	if got := m.pageCount(); got != 3 {
		t.Errorf("120 docs: pageCount = %d, want 3", got)
	}
}
