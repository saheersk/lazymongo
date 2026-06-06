package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const aggTestCol = "test_aggregate"

func setupAggData(t *testing.T) {
	t.Helper()
	skipIfNoMongo(t)
	dropCol(t, aggTestCol)

	docs := []bson.M{
		{"category": "fruit", "name": "apple", "price": int32(3)},
		{"category": "fruit", "name": "banana", "price": int32(1)},
		{"category": "fruit", "name": "cherry", "price": int32(5)},
		{"category": "veggie", "name": "carrot", "price": int32(2)},
		{"category": "veggie", "name": "broccoli", "price": int32(4)},
	}
	for _, d := range docs {
		if _, err := mongoClient.InsertOne(crudTestDB, aggTestCol, d); err != nil {
			t.Fatalf("setup InsertOne: %v", err)
		}
	}
}

func dropCol(t *testing.T, col string) {
	t.Helper()
	ctx, cancel := opCtx()
	defer cancel()
	_ = mongoClient.inner.Database(crudTestDB).Collection(col).Drop(ctx)
}

// TestAggregate_Match verifies $match filters documents correctly.
func TestAggregate_Match(t *testing.T) {
	setupAggData(t)

	pipeline := bson.A{
		bson.M{"$match": bson.M{"category": "fruit"}},
	}
	docs, err := mongoClient.Aggregate(crudTestDB, aggTestCol, pipeline)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if len(docs) != 3 {
		t.Errorf("expected 3 fruits, got %d", len(docs))
	}
	for _, d := range docs {
		if d["category"] != "fruit" {
			t.Errorf("unexpected category %v", d["category"])
		}
	}
}

// TestAggregate_Group verifies $group produces the right document count.
func TestAggregate_Group(t *testing.T) {
	setupAggData(t)

	pipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":   "$category",
			"count": bson.M{"$sum": 1},
		}},
		bson.M{"$sort": bson.M{"_id": 1}},
	}
	docs, err := mongoClient.Aggregate(crudTestDB, aggTestCol, pipeline)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(docs))
	}
	// sorted: fruit=3, veggie=2
	fruitCount, _ := docs[0]["count"].(int32)
	veggieCount, _ := docs[1]["count"].(int32)
	if fruitCount != 3 {
		t.Errorf("fruit count: got %d, want 3", fruitCount)
	}
	if veggieCount != 2 {
		t.Errorf("veggie count: got %d, want 2", veggieCount)
	}
}

// TestAggregate_Sort verifies $sort ordering.
func TestAggregate_Sort(t *testing.T) {
	setupAggData(t)

	pipeline := bson.A{
		bson.M{"$sort": bson.M{"price": 1}},
	}
	docs, err := mongoClient.Aggregate(crudTestDB, aggTestCol, pipeline)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if len(docs) != 5 {
		t.Fatalf("expected 5 docs, got %d", len(docs))
	}
	// Prices should be ascending: 1, 2, 3, 4, 5
	wantPrices := []int32{1, 2, 3, 4, 5}
	for i, want := range wantPrices {
		got, _ := docs[i]["price"].(int32)
		if got != want {
			t.Errorf("doc[%d].price = %d; want %d", i, got, want)
		}
	}
}

// TestAggregate_AutoLimit verifies that a pipeline without $limit
// has one injected, so we get results back without error.
func TestAggregate_AutoLimit(t *testing.T) {
	setupAggData(t)

	// Pipeline has no $limit — the function should inject one automatically.
	pipeline := bson.A{
		bson.M{"$match": bson.M{}},
	}
	docs, err := mongoClient.Aggregate(crudTestDB, aggTestCol, pipeline)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if len(docs) != 5 {
		t.Errorf("expected 5 docs, got %d", len(docs))
	}
}

// TestAggregate_ExplicitLimitRespected verifies that a $limit in the pipeline
// is not overridden.
func TestAggregate_ExplicitLimitRespected(t *testing.T) {
	setupAggData(t)

	pipeline := bson.A{
		bson.M{"$sort": bson.M{"price": 1}},
		bson.M{"$limit": int32(2)},
	}
	docs, err := mongoClient.Aggregate(crudTestDB, aggTestCol, pipeline)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 docs (explicit $limit), got %d", len(docs))
	}
}

// TestAggregate_EmptyResult verifies that a pipeline matching nothing returns
// an empty slice, not an error.
func TestAggregate_EmptyResult(t *testing.T) {
	setupAggData(t)

	pipeline := bson.A{
		bson.M{"$match": bson.M{"category": "nonexistent"}},
	}
	docs, err := mongoClient.Aggregate(crudTestDB, aggTestCol, pipeline)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

// TestHasLimitOrSink unit-tests the helper without a real DB.
func TestHasLimitOrSink(t *testing.T) {
	cases := []struct {
		name   string
		pipe   bson.A
		expect bool
	}{
		{"empty", bson.A{}, false},
		{"match only", bson.A{bson.M{"$match": bson.M{}}}, false},
		{"has $limit", bson.A{bson.M{"$limit": 100}}, true},
		{"has $out", bson.A{bson.M{"$out": "dest"}}, true},
		{"has $merge", bson.A{bson.M{"$merge": bson.M{"into": "dest"}}}, true},
		{"bson.D $limit", bson.A{bson.D{{Key: "$limit", Value: 10}}}, true},
		{"match then limit", bson.A{bson.M{"$match": bson.M{}}, bson.M{"$limit": 5}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasLimitOrSink(tc.pipe); got != tc.expect {
				t.Errorf("hasLimitOrSink(%v) = %v; want %v", tc.pipe, got, tc.expect)
			}
		})
	}
}
