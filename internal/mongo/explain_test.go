package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const explainTestCol = "test_explain"

func TestExplainQuery_NoFilter(t *testing.T) {
	skipIfNoMongo(t)

	// Seed a couple of docs so the collection exists.
	col := mongoClient.inner.Database(crudTestDB).Collection(explainTestCol)
	ctx, cancel := opCtx()
	defer cancel()
	_ = col.Drop(ctx)
	for _, name := range []string{"alice", "bob"} {
		_, _ = mongoClient.InsertOne(crudTestDB, explainTestCol, bson.M{"name": name})
	}

	stats, err := mongoClient.ExplainQuery(crudTestDB, explainTestCol, nil, nil)
	if err != nil {
		t.Fatalf("ExplainQuery: %v", err)
	}

	if stats.DB != crudTestDB {
		t.Errorf("DB = %q; want %q", stats.DB, crudTestDB)
	}
	if stats.Col != explainTestCol {
		t.Errorf("Col = %q; want %q", stats.Col, explainTestCol)
	}
	if stats.Raw == nil {
		t.Error("Raw explain output should not be nil")
	}
	if stats.NReturned < 0 {
		t.Errorf("NReturned = %d; should be >= 0", stats.NReturned)
	}
	if stats.ExecutionTimeMs < 0 {
		t.Errorf("ExecutionTimeMs = %d; should be >= 0", stats.ExecutionTimeMs)
	}

	t.Logf("explain: nReturned=%d docsExamined=%d keysExamined=%d ms=%d index=%q",
		stats.NReturned, stats.DocsExamined, stats.KeysExamined,
		stats.ExecutionTimeMs, stats.IndexUsed)
}

func TestExplainQuery_WithFilter(t *testing.T) {
	skipIfNoMongo(t)

	for _, name := range []string{"charlie", "diana"} {
		_, _ = mongoClient.InsertOne(crudTestDB, explainTestCol, bson.M{"name": name, "role": "admin"})
	}

	stats, err := mongoClient.ExplainQuery(crudTestDB, explainTestCol,
		bson.M{"role": "admin"}, nil)
	if err != nil {
		t.Fatalf("ExplainQuery with filter: %v", err)
	}
	if stats.NReturned < 1 {
		t.Errorf("expected at least 1 returned doc, got %d", stats.NReturned)
	}
	// Without an index on role this must be a COLLSCAN — IndexUsed should be "".
	t.Logf("index used: %q (empty = COLLSCAN)", stats.IndexUsed)
}

func TestExplainQuery_WithSort(t *testing.T) {
	skipIfNoMongo(t)

	stats, err := mongoClient.ExplainQuery(crudTestDB, explainTestCol,
		nil, bson.D{{Key: "name", Value: 1}})
	if err != nil {
		t.Fatalf("ExplainQuery with sort: %v", err)
	}
	if stats.Raw == nil {
		t.Error("Raw should not be nil")
	}
	t.Logf("sorted explain: nReturned=%d ms=%d", stats.NReturned, stats.ExecutionTimeMs)
}
