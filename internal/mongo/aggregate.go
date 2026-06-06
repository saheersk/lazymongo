package mongo

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Aggregate runs a pipeline against a collection and returns its documents.
// If the pipeline contains no $limit, $out, or $merge stage an automatic
// {"$limit":1000} is appended so the TUI stays responsive.
func (c *Client) Aggregate(dbName, colName string, pipeline bson.A) ([]bson.M, error) {
	if !hasLimitOrSink(pipeline) {
		pipeline = append(pipeline, bson.M{"$limit": 1000})
	}

	ctx, cancel := opCtx()
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// hasLimitOrSink returns true when the pipeline already contains a $limit,
// $out, or $merge stage.
func hasLimitOrSink(pipeline bson.A) bool {
	sinks := map[string]bool{"$limit": true, "$out": true, "$merge": true}
	for _, stage := range pipeline {
		if m, ok := stage.(bson.M); ok {
			for k := range m {
				if sinks[k] {
					return true
				}
			}
		}
		if d, ok := stage.(bson.D); ok {
			for _, e := range d {
				if sinks[e.Key] {
					return true
				}
			}
		}
	}
	return false
}
