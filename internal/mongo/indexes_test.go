package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const idxTestCol = "test_indexes"

func setupIdxCol(t *testing.T) {
	t.Helper()
	skipIfNoMongo(t)
	dropCol(t, idxTestCol)
	// Insert one document so the collection exists.
	if _, err := mongoClient.InsertOne(crudTestDB, idxTestCol, bson.M{"seed": true}); err != nil {
		t.Fatalf("setup InsertOne: %v", err)
	}
}

// TestListIndexesAndStats_DefaultIndex verifies every collection has the
// built-in _id_ index listed and that DocCount reflects inserted docs.
func TestListIndexesAndStats_DefaultIndex(t *testing.T) {
	setupIdxCol(t)

	idxs, stats, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats error: %v", err)
	}

	found := false
	for _, idx := range idxs {
		if idx.Name == "_id_" {
			found = true
		}
	}
	if !found {
		t.Error("expected _id_ index to be present")
	}

	if stats.DocCount < 1 {
		t.Errorf("DocCount = %d; want >= 1", stats.DocCount)
	}
	if stats.IndexCount < 1 {
		t.Errorf("IndexCount = %d; want >= 1", stats.IndexCount)
	}
}

// TestCreateIndex_PlainIndex creates a simple ascending index and confirms
// it appears in ListIndexesAndStats.
func TestCreateIndex_PlainIndex(t *testing.T) {
	setupIdxCol(t)

	name, err := mongoClient.CreateIndex(crudTestDB, idxTestCol,
		bson.D{{Key: "email", Value: 1}}, false, false, -1)
	if err != nil {
		t.Fatalf("CreateIndex error: %v", err)
	}
	if name == "" {
		t.Fatal("expected non-empty index name")
	}

	idxs, _, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats error: %v", err)
	}
	found := false
	for _, idx := range idxs {
		if idx.Name == name {
			found = true
		}
	}
	if !found {
		t.Errorf("created index %q not found in list", name)
	}
}

// TestCreateIndex_UniqueFlag verifies that a unique index is reported as Unique.
func TestCreateIndex_UniqueFlag(t *testing.T) {
	setupIdxCol(t)

	name, err := mongoClient.CreateIndex(crudTestDB, idxTestCol,
		bson.D{{Key: "username", Value: 1}}, true, false, -1)
	if err != nil {
		t.Fatalf("CreateIndex unique error: %v", err)
	}

	idxs, _, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats error: %v", err)
	}
	for _, idx := range idxs {
		if idx.Name == name {
			if !idx.Unique {
				t.Errorf("index %q should be Unique", name)
			}
			return
		}
	}
	t.Errorf("index %q not found", name)
}

// TestCreateIndex_SparseFlag verifies that a sparse index is reported as Sparse.
func TestCreateIndex_SparseFlag(t *testing.T) {
	setupIdxCol(t)

	name, err := mongoClient.CreateIndex(crudTestDB, idxTestCol,
		bson.D{{Key: "phone", Value: 1}}, false, true, -1)
	if err != nil {
		t.Fatalf("CreateIndex sparse error: %v", err)
	}

	idxs, _, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats error: %v", err)
	}
	for _, idx := range idxs {
		if idx.Name == name {
			if !idx.Sparse {
				t.Errorf("index %q should be Sparse", name)
			}
			return
		}
	}
	t.Errorf("index %q not found", name)
}

// TestCreateIndex_CompoundIndex verifies a multi-key index is created.
func TestCreateIndex_CompoundIndex(t *testing.T) {
	setupIdxCol(t)

	name, err := mongoClient.CreateIndex(crudTestDB, idxTestCol,
		bson.D{{Key: "lastName", Value: 1}, {Key: "firstName", Value: 1}},
		false, false, -1)
	if err != nil {
		t.Fatalf("CreateIndex compound error: %v", err)
	}

	idxs, _, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats error: %v", err)
	}
	for _, idx := range idxs {
		if idx.Name == name {
			if len(idx.Keys) != 2 {
				t.Errorf("compound index: expected 2 keys, got %d", len(idx.Keys))
			}
			return
		}
	}
	t.Errorf("index %q not found", name)
}

// TestCreateIndex_TTL verifies a TTL index reports its expireAfterSeconds.
func TestCreateIndex_TTL(t *testing.T) {
	setupIdxCol(t)

	name, err := mongoClient.CreateIndex(crudTestDB, idxTestCol,
		bson.D{{Key: "createdAt", Value: 1}}, false, false, 3600)
	if err != nil {
		t.Fatalf("CreateIndex TTL error: %v", err)
	}

	idxs, _, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats error: %v", err)
	}
	for _, idx := range idxs {
		if idx.Name == name {
			if idx.TTLSeconds != 3600 {
				t.Errorf("TTLSeconds = %d; want 3600", idx.TTLSeconds)
			}
			return
		}
	}
	t.Errorf("index %q not found", name)
}

// TestDropIndex_IndexDisappears creates an index, drops it, and verifies
// it is gone from ListIndexesAndStats.
func TestDropIndex_IndexDisappears(t *testing.T) {
	setupIdxCol(t)

	name, err := mongoClient.CreateIndex(crudTestDB, idxTestCol,
		bson.D{{Key: "toDelete", Value: 1}}, false, false, -1)
	if err != nil {
		t.Fatalf("CreateIndex error: %v", err)
	}

	if err := mongoClient.DropIndex(crudTestDB, idxTestCol, name); err != nil {
		t.Fatalf("DropIndex error: %v", err)
	}

	idxs, _, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats error: %v", err)
	}
	for _, idx := range idxs {
		if idx.Name == name {
			t.Errorf("dropped index %q still present", name)
		}
	}
}

// TestDropIndex_NonExistentReturnsError verifies that dropping a name that
// doesn't exist returns an error (MongoDB raises one).
func TestDropIndex_NonExistentReturnsError(t *testing.T) {
	setupIdxCol(t)

	err := mongoClient.DropIndex(crudTestDB, idxTestCol, "nonexistent_index")
	if err == nil {
		t.Error("expected error dropping nonexistent index, got nil")
	}
}

// TestListIndexesAndStats_IndexCountMatchesCreated inserts N indexes and
// verifies IndexCount == N+1 (the extra 1 is the built-in _id_).
func TestListIndexesAndStats_IndexCountMatchesCreated(t *testing.T) {
	setupIdxCol(t)

	fields := []string{"alpha", "beta", "gamma"}
	for _, f := range fields {
		if _, err := mongoClient.CreateIndex(crudTestDB, idxTestCol,
			bson.D{{Key: f, Value: 1}}, false, false, -1); err != nil {
			t.Fatalf("CreateIndex %q: %v", f, err)
		}
	}

	_, stats, err := mongoClient.ListIndexesAndStats(crudTestDB, idxTestCol)
	if err != nil {
		t.Fatalf("ListIndexesAndStats: %v", err)
	}

	want := len(fields) + 1 // + _id_
	if stats.IndexCount != want {
		t.Errorf("IndexCount = %d; want %d", stats.IndexCount, want)
	}
}
