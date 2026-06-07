package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const importTestCol = "test_import"

func TestInsertMany_AllSucceed(t *testing.T) {
	skipIfNoMongo(t)
	ctx, cancel := opCtx()
	defer cancel()
	_ = mongoClient.inner.Database(crudTestDB).Collection(importTestCol).Drop(ctx)

	docs := []bson.M{
		{"name": "Alice", "role": "admin"},
		{"name": "Bob", "role": "user"},
		{"name": "Carol", "role": "user"},
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importTestCol, docs)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if inserted != 3 {
		t.Errorf("inserted = %d; want 3", inserted)
	}

	// Verify they're actually in the collection.
	result, err := mongoClient.FindPage(crudTestDB, importTestCol, QueryOptions{PageSize: 10})
	if err != nil {
		t.Fatalf("FindPage: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("total = %d; want 3", result.Total)
	}
}

func TestInsertMany_Empty(t *testing.T) {
	skipIfNoMongo(t)

	inserted, errs := mongoClient.InsertMany(crudTestDB, importTestCol, nil)
	if inserted != 0 || len(errs) != 0 {
		t.Errorf("empty input: inserted=%d errs=%v; want 0 and no errors", inserted, errs)
	}
}

func TestInsertMany_LargeBatch(t *testing.T) {
	skipIfNoMongo(t)
	ctx, cancel := opCtx()
	defer cancel()
	_ = mongoClient.inner.Database(crudTestDB).Collection(importTestCol).Drop(ctx)

	// 1,200 docs — forces 3 batches of 500 / 500 / 200.
	const total = 1200
	docs := make([]bson.M, total)
	for i := range docs {
		docs[i] = bson.M{"seq": int32(i)}
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importTestCol, docs)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors on large batch: %v", errs)
	}
	if inserted != total {
		t.Errorf("inserted = %d; want %d", inserted, total)
	}
}

func TestInsertMany_DuplicateIDContinues(t *testing.T) {
	skipIfNoMongo(t)
	ctx, cancel := opCtx()
	defer cancel()
	_ = mongoClient.inner.Database(crudTestDB).Collection(importTestCol).Drop(ctx)

	fixedID := bson.NewObjectID()
	docs := []bson.M{
		{"_id": fixedID, "name": "first"},
		{"_id": fixedID, "name": "duplicate — should fail"},
		{"name": "no id — should succeed"},
	}

	inserted, errs := mongoClient.InsertMany(crudTestDB, importTestCol, docs)
	// With SetOrdered(false) the duplicate fails but the third doc still inserts.
	if inserted < 2 {
		t.Errorf("inserted = %d; want >= 2 (first + third)", inserted)
	}
	if len(errs) == 0 {
		t.Error("expected at least one error for duplicate _id, got none")
	}
	t.Logf("inserted=%d errors=%d (expected: 2 inserted, 1 error)", inserted, len(errs))
}
