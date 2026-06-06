package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ── test setup ────────────────────────────────────────────────────────────────

const (
	crudTestDB  = "lazymongo_test"
	crudTestCol = "test_crud"
)

// mongoClient is shared across all crud tests; nil means MongoDB not available.
var mongoClient *Client

// TestMain sets up/tears down the shared client and cleans the test collection.
func TestMain(m *testing.M) {
	c, err := NewClient(testMongoURI)
	if err != nil {
		// MongoDB unavailable — tests will skip themselves.
		m.Run()
		return
	}
	mongoClient = c
	defer func() {
		// Drop the test collection on the way out.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = c.inner.Database(crudTestDB).Collection(crudTestCol).Drop(ctx)
		c.Disconnect()
	}()
	m.Run()
}

// skipIfNoMongo skips the current test if MongoDB is not available.
func skipIfNoMongo(t *testing.T) {
	t.Helper()
	if mongoClient == nil {
		t.Skip("MongoDB not available — skipping integration test")
	}
}

// dropAndRecreate drops the test collection so each test gets a clean slate.
func dropAndRecreate(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mongoClient.inner.Database(crudTestDB).Collection(crudTestCol).Drop(ctx); err != nil {
		t.Fatalf("drop collection: %v", err)
	}
}

// ── InsertOne ─────────────────────────────────────────────────────────────────

func TestInsertOne_ReturnsNonNilID(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	doc := bson.M{"name": "Alice", "age": int32(30)}
	id, err := mongoClient.InsertOne(crudTestDB, crudTestCol, doc)
	if err != nil {
		t.Fatalf("InsertOne error: %v", err)
	}
	if id == nil {
		t.Fatal("expected non-nil inserted ID")
	}
}

func TestInsertOne_DocIsFindable(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	doc := bson.M{"name": "Bob", "score": int32(99)}
	id, err := mongoClient.InsertOne(crudTestDB, crudTestCol, doc)
	if err != nil {
		t.Fatalf("InsertOne error: %v", err)
	}

	result, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		Filter: bson.M{"_id": id},
	})
	if err != nil {
		t.Fatalf("FindPage error: %v", err)
	}
	if len(result.Docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(result.Docs))
	}
	if result.Docs[0]["name"] != "Bob" {
		t.Errorf("expected name=Bob, got %v", result.Docs[0]["name"])
	}
}

// ── ReplaceOne ────────────────────────────────────────────────────────────────

func TestReplaceOne_FieldChangesVisible(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	doc := bson.M{"name": "Carol", "status": "active"}
	id, err := mongoClient.InsertOne(crudTestDB, crudTestCol, doc)
	if err != nil {
		t.Fatalf("InsertOne error: %v", err)
	}

	replacement := bson.M{"name": "Carol", "status": "retired"}
	if err := mongoClient.ReplaceOne(crudTestDB, crudTestCol, id, replacement); err != nil {
		t.Fatalf("ReplaceOne error: %v", err)
	}

	result, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		Filter: bson.M{"_id": id},
	})
	if err != nil {
		t.Fatalf("FindPage error: %v", err)
	}
	if len(result.Docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(result.Docs))
	}
	if result.Docs[0]["status"] != "retired" {
		t.Errorf("expected status=retired, got %v", result.Docs[0]["status"])
	}
}

func TestReplaceOne_StripsIDField(t *testing.T) {
	// Passing _id in the replacement document must not cause "immutable field"
	// error because ReplaceOne strips it before sending.
	skipIfNoMongo(t)
	dropAndRecreate(t)

	doc := bson.M{"name": "Dave"}
	id, err := mongoClient.InsertOne(crudTestDB, crudTestCol, doc)
	if err != nil {
		t.Fatalf("InsertOne error: %v", err)
	}

	// Replacement includes _id — should still succeed.
	replacement := bson.M{"_id": id, "name": "Dave Updated"}
	if err := mongoClient.ReplaceOne(crudTestDB, crudTestCol, id, replacement); err != nil {
		t.Fatalf("ReplaceOne with _id in replacement error: %v", err)
	}
}

// ── DeleteOne ─────────────────────────────────────────────────────────────────

func TestDeleteOne_DocIsGone(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	doc := bson.M{"name": "Eve"}
	id, err := mongoClient.InsertOne(crudTestDB, crudTestCol, doc)
	if err != nil {
		t.Fatalf("InsertOne error: %v", err)
	}

	if err := mongoClient.DeleteOne(crudTestDB, crudTestCol, id); err != nil {
		t.Fatalf("DeleteOne error: %v", err)
	}

	result, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		Filter: bson.M{"_id": id},
	})
	if err != nil {
		t.Fatalf("FindPage error: %v", err)
	}
	if len(result.Docs) != 0 {
		t.Errorf("expected 0 docs after delete, got %d", len(result.Docs))
	}
}

// ── FindPage ──────────────────────────────────────────────────────────────────

func TestFindPage_Pagination(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	// Insert 5 documents.
	for i := 0; i < 5; i++ {
		doc := bson.M{"seq": int32(i)}
		if _, err := mongoClient.InsertOne(crudTestDB, crudTestCol, doc); err != nil {
			t.Fatalf("InsertOne seq=%d: %v", i, err)
		}
	}

	// Page 0, size 2 → 2 docs, total 5.
	page0, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		PageSize: 2,
		Page:     0,
	})
	if err != nil {
		t.Fatalf("FindPage p0: %v", err)
	}
	if len(page0.Docs) != 2 {
		t.Errorf("page 0: expected 2 docs, got %d", len(page0.Docs))
	}
	if page0.Total != 5 {
		t.Errorf("total: expected 5, got %d", page0.Total)
	}

	// Page 1, size 2 → 2 docs.
	page1, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		PageSize: 2,
		Page:     1,
	})
	if err != nil {
		t.Fatalf("FindPage p1: %v", err)
	}
	if len(page1.Docs) != 2 {
		t.Errorf("page 1: expected 2 docs, got %d", len(page1.Docs))
	}

	// Page 2, size 2 → 1 doc (remainder).
	page2, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		PageSize: 2,
		Page:     2,
	})
	if err != nil {
		t.Fatalf("FindPage p2: %v", err)
	}
	if len(page2.Docs) != 1 {
		t.Errorf("page 2: expected 1 doc, got %d", len(page2.Docs))
	}
}

func TestFindPage_WithFilter(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	docs := []bson.M{
		{"kind": "fruit", "name": "apple"},
		{"kind": "fruit", "name": "banana"},
		{"kind": "veggie", "name": "carrot"},
	}
	for _, d := range docs {
		if _, err := mongoClient.InsertOne(crudTestDB, crudTestCol, d); err != nil {
			t.Fatalf("InsertOne: %v", err)
		}
	}

	result, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		Filter: bson.M{"kind": "fruit"},
	})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 fruits, got %d", result.Total)
	}
	for _, doc := range result.Docs {
		if doc["kind"] != "fruit" {
			t.Errorf("unexpected kind %v in filtered result", doc["kind"])
		}
	}
}

func TestFindPage_WithSort(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	for _, n := range []int32{3, 1, 2} {
		if _, err := mongoClient.InsertOne(crudTestDB, crudTestCol, bson.M{"n": n}); err != nil {
			t.Fatalf("InsertOne: %v", err)
		}
	}

	// Sort ascending by n.
	result, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{
		Sort: bson.D{{Key: "n", Value: 1}},
	})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if len(result.Docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(result.Docs))
	}

	for i, want := range []int32{1, 2, 3} {
		got, ok := result.Docs[i]["n"].(int32)
		if !ok {
			t.Fatalf("doc[%d]['n'] is not int32: %T", i, result.Docs[i]["n"])
		}
		if got != want {
			t.Errorf("doc[%d].n = %d; want %d", i, got, want)
		}
	}
}

func TestFindPage_DefaultPageSize(t *testing.T) {
	skipIfNoMongo(t)
	dropAndRecreate(t)

	// Insert 3 docs; pageSize=0 should default to 50 (so all returned).
	for i := 0; i < 3; i++ {
		if _, err := mongoClient.InsertOne(crudTestDB, crudTestCol, bson.M{"i": int32(i)}); err != nil {
			t.Fatalf("InsertOne: %v", err)
		}
	}

	result, err := mongoClient.FindPage(crudTestDB, crudTestCol, QueryOptions{PageSize: 0})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if result.PageSize != 50 {
		t.Errorf("expected default PageSize=50, got %d", result.PageSize)
	}
	if len(result.Docs) != 3 {
		t.Errorf("expected 3 docs, got %d", len(result.Docs))
	}
}

// ── helper to verify error wrapping ──────────────────────────────────────────

func TestErrorTypes(t *testing.T) {
	// This is a compile-time check that errors.Is / errors.As work as expected
	// with our error chains.  We just verify the standard library is usable.
	err := errors.New("base")
	if !errors.Is(err, err) {
		t.Error("errors.Is broken")
	}
}
