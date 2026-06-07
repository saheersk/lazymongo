package mongo

import (
	"fmt"
	"sort"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SampleSchema samples up to sampleSize documents and returns a per-field
// breakdown of BSON types and occurrence frequency, sorted by field prevalence.
func (c *Client) SampleSchema(dbName, colName string, sampleSize int) (msg.SchemaResult, error) {
	ctx, cancel := opCtx()
	defer cancel()

	if sampleSize <= 0 {
		sampleSize = 100
	}

	pipeline := bson.A{
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: int32(sampleSize)}}}},
	}

	cursor, err := c.inner.Database(dbName).Collection(colName).Aggregate(ctx, pipeline)
	if err != nil {
		return msg.SchemaResult{DB: dbName, Col: colName}, err
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return msg.SchemaResult{DB: dbName, Col: colName}, err
	}

	// fieldTypes[field][typeName] = count
	fieldTypes := map[string]map[string]int{}
	fieldCount := map[string]int{}

	for _, doc := range docs {
		for k, v := range doc {
			if _, ok := fieldTypes[k]; !ok {
				fieldTypes[k] = map[string]int{}
			}
			fieldTypes[k][bsonTypeName(v)]++
			fieldCount[k]++
		}
	}

	fields := make([]msg.SchemaField, 0, len(fieldTypes))
	for name, types := range fieldTypes {
		var tf []msg.TypeFreq
		for t, cnt := range types {
			tf = append(tf, msg.TypeFreq{Type: t, Count: cnt})
		}
		sort.Slice(tf, func(i, j int) bool { return tf[i].Count > tf[j].Count })
		fields = append(fields, msg.SchemaField{
			Name:  name,
			Types: tf,
			Count: fieldCount[name],
		})
	}

	// _id first, then frequency desc, then alphabetical.
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].Name == "_id" {
			return true
		}
		if fields[j].Name == "_id" {
			return false
		}
		if fields[i].Count != fields[j].Count {
			return fields[i].Count > fields[j].Count
		}
		return fields[i].Name < fields[j].Name
	})

	return msg.SchemaResult{
		DB:         dbName,
		Col:        colName,
		Fields:     fields,
		SampleSize: len(docs),
	}, nil
}

func bsonTypeName(v interface{}) string {
	switch v.(type) {
	case bson.ObjectID:
		return "objectId"
	case string:
		return "string"
	case int32:
		return "int32"
	case int64:
		return "int64"
	case float64:
		return "double"
	case bool:
		return "bool"
	case bson.DateTime:
		return "date"
	case bson.A:
		return "array"
	case bson.M, bson.D:
		return "object"
	case bson.Binary:
		return "binData"
	case bson.Decimal128:
		return "decimal"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}
