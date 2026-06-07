package mongo

import (
	"fmt"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// CreateCollection creates a new collection in dbName.
func (c *Client) CreateCollection(dbName, colName string) error {
	ctx, cancel := opCtx()
	defer cancel()

	if err := c.inner.Database(dbName).CreateCollection(ctx, colName); err != nil {
		return fmt.Errorf("create collection %q.%q: %w", dbName, colName, err)
	}
	return nil
}

// DropCollection permanently drops a collection.
func (c *Client) DropCollection(dbName, colName string) error {
	ctx, cancel := opCtx()
	defer cancel()

	if err := c.inner.Database(dbName).Collection(colName).Drop(ctx); err != nil {
		return fmt.Errorf("drop collection %q.%q: %w", dbName, colName, err)
	}
	return nil
}

// RenameCollection renames a collection using the admin renameCollection command.
func (c *Client) RenameCollection(dbName, oldCol, newCol string) error {
	ctx, cancel := opCtx()
	defer cancel()

	cmd := bson.D{
		{Key: "renameCollection", Value: dbName + "." + oldCol},
		{Key: "to", Value: dbName + "." + newCol},
	}
	res := c.inner.Database("admin").RunCommand(ctx, cmd)
	if err := res.Err(); err != nil {
		return fmt.Errorf("rename collection %q → %q: %w", oldCol, newCol, err)
	}
	return nil
}

// CollectionStats retrieves statistics for a collection.
// It uses the collStats command (works on MongoDB 3.x–7.x) and falls back to
// the $collStats aggregation pipeline on error. Document count always comes
// from CountDocuments for accuracy.
func (c *Client) CollectionStats(dbName, colName string) (msg.CollectionStatsDetail, error) {
	ctx, cancel := opCtx()
	defer cancel()

	// CountDocuments is always accurate regardless of storage engine.
	count, _ := c.inner.Database(dbName).Collection(colName).CountDocuments(ctx, bson.D{})

	// Try the collStats command first — top-level fields, no nested-doc decoding issue.
	var raw bson.M
	res := c.inner.Database(dbName).RunCommand(ctx, bson.D{{Key: "collStats", Value: colName}})
	if err := res.Decode(&raw); err == nil {
		return msg.CollectionStatsDetail{
			DocCount:    count,
			TotalSize:   toInt64(raw["size"]),
			AvgDocSize:  toFloat64(raw["avgObjSize"]),
			StorageSize: toInt64(raw["storageSize"]),
			IndexCount:  int(toInt64(raw["nindexes"])),
			IndexSize:   toInt64(raw["totalIndexSize"]),
		}, nil
	}

	// Fallback: $collStats aggregation pipeline (MongoDB 4.4+).
	// In mongo-driver v2, nested BSON docs inside bson.M may decode as bson.D,
	// so we use extractBSONDoc to handle both types.
	pipeline := bson.A{
		bson.D{{Key: "$collStats", Value: bson.D{{Key: "storageStats", Value: bson.D{}}}}},
	}
	cursor, err := c.inner.Database(dbName).Collection(colName).Aggregate(ctx, pipeline)
	if err != nil {
		return msg.CollectionStatsDetail{DocCount: count}, nil
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil || len(results) == 0 {
		return msg.CollectionStatsDetail{DocCount: count}, nil
	}

	ss := extractBSONDoc(results[0]["storageStats"])
	if ss == nil {
		return msg.CollectionStatsDetail{DocCount: count}, nil
	}

	return msg.CollectionStatsDetail{
		DocCount:    count,
		TotalSize:   toInt64(ss["size"]),
		AvgDocSize:  toFloat64(ss["avgObjSize"]),
		StorageSize: toInt64(ss["storageSize"]),
		IndexCount:  int(toInt64(ss["nindexes"])),
		IndexSize:   toInt64(ss["totalIndexSize"]),
	}, nil
}

// extractBSONDoc converts a bson.M or bson.D value to a plain map for field access.
// In mongo-driver v2, nested documents in a bson.M may decode as bson.D.
func extractBSONDoc(v interface{}) map[string]interface{} {
	switch m := v.(type) {
	case bson.M:
		return map[string]interface{}(m)
	case bson.D:
		result := make(map[string]interface{}, len(m))
		for _, e := range m {
			result[e.Key] = e.Value
		}
		return result
	}
	return nil
}

// toInt64 converts a bson.M numeric value (int32/int64/float64) to int64.
func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int32:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	}
	return 0
}

// toFloat64 converts a bson.M numeric value to float64.
func toFloat64(v interface{}) float64 {
	switch t := v.(type) {
	case int32:
		return float64(t)
	case int64:
		return float64(t)
	case float64:
		return t
	}
	return 0
}
