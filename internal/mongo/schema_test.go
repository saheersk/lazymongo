package mongo

import (
	"testing"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const schemaTestCol = "test_schema"

func seedSchemaCollection(t *testing.T) {
	t.Helper()
	ctx, cancel := opCtx()
	defer cancel()
	_ = mongoClient.inner.Database(crudTestDB).Collection(schemaTestCol).Drop(ctx)

	docs := []bson.M{
		{"name": "Alice", "age": int32(30), "active": true, "score": 9.5},
		{"name": "Bob", "age": int32(25), "active": false},
		{"name": "Carol", "age": int32(35), "active": true, "score": 8.1, "tags": bson.A{"admin", "user"}},
		{"name": "Dave", "active": true},
		{"name": "Eve", "age": int32(28), "meta": bson.M{"level": int32(3)}},
	}
	for _, d := range docs {
		if _, err := mongoClient.InsertOne(crudTestDB, schemaTestCol, d); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func schemaField(fields []msg.SchemaField, name string) *msg.SchemaField {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func TestSampleSchema_FieldsPresent(t *testing.T) {
	skipIfNoMongo(t)
	seedSchemaCollection(t)

	result, err := mongoClient.SampleSchema(crudTestDB, schemaTestCol, 100)
	if err != nil {
		t.Fatalf("SampleSchema: %v", err)
	}
	if result.SampleSize != 5 {
		t.Errorf("SampleSize = %d; want 5", result.SampleSize)
	}
	if len(result.Fields) == 0 {
		t.Fatal("expected fields, got none")
	}

	// _id must always be first.
	if result.Fields[0].Name != "_id" {
		t.Errorf("first field = %q; want \"_id\"", result.Fields[0].Name)
	}

	// name appears in all 5 docs.
	f := schemaField(result.Fields, "name")
	if f == nil {
		t.Fatal("field 'name' missing")
	}
	if f.Count != 5 {
		t.Errorf("name.Count = %d; want 5", f.Count)
	}
	if len(f.Types) == 0 || f.Types[0].Type != "string" {
		t.Errorf("name.Types = %v; want string first", f.Types)
	}

	// active appears in all 5 docs.
	if a := schemaField(result.Fields, "active"); a != nil {
		if len(a.Types) == 0 || a.Types[0].Type != "bool" {
			t.Errorf("active.Types = %v; want bool", a.Types)
		}
	}

	// tags appears in 1 doc — should be array.
	if tg := schemaField(result.Fields, "tags"); tg != nil {
		if len(tg.Types) == 0 || tg.Types[0].Type != "array" {
			t.Errorf("tags.Types = %v; want array", tg.Types)
		}
	}

	// meta is an embedded object.
	if mt := schemaField(result.Fields, "meta"); mt != nil {
		if len(mt.Types) == 0 || mt.Types[0].Type != "object" {
			t.Errorf("meta.Types = %v; want object", mt.Types)
		}
	}

	t.Logf("schema (%d fields, %d docs sampled):", len(result.Fields), result.SampleSize)
	for _, fld := range result.Fields {
		t.Logf("  %-12s  count=%-3d  types=%v", fld.Name, fld.Count, fld.Types)
	}
}

func TestSampleSchema_SortedByFrequency(t *testing.T) {
	skipIfNoMongo(t)
	seedSchemaCollection(t)

	result, err := mongoClient.SampleSchema(crudTestDB, schemaTestCol, 100)
	if err != nil {
		t.Fatalf("SampleSchema: %v", err)
	}
	if result.Fields[0].Name != "_id" {
		t.Errorf("fields[0] = %q; want _id", result.Fields[0].Name)
	}
	// After _id, fields should be sorted frequency desc.
	for i := 1; i < len(result.Fields)-1; i++ {
		if result.Fields[i].Count < result.Fields[i+1].Count {
			t.Errorf("not sorted by frequency: fields[%d].Count=%d < fields[%d].Count=%d",
				i, result.Fields[i].Count, i+1, result.Fields[i+1].Count)
		}
	}
}

func TestSampleSchema_EmptyCollection(t *testing.T) {
	skipIfNoMongo(t)
	ctx, cancel := opCtx()
	defer cancel()
	_ = mongoClient.inner.Database(crudTestDB).Collection("test_schema_empty").Drop(ctx)

	result, err := mongoClient.SampleSchema(crudTestDB, "test_schema_empty", 100)
	if err != nil {
		t.Fatalf("SampleSchema on empty collection: %v", err)
	}
	if result.SampleSize != 0 || len(result.Fields) != 0 {
		t.Errorf("expected empty result, got SampleSize=%d Fields=%v", result.SampleSize, result.Fields)
	}
}
