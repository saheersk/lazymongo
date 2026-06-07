package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const importBatchSize = 500

// InsertMany inserts docs in batches of 500 with unordered writes so a single
// duplicate-key error doesn't abort the rest. Returns the total inserted count
// and any per-batch errors encountered.
func (c *Client) InsertMany(dbName, colName string, docs []bson.M) (int, []error) {
	if len(docs) == 0 {
		return 0, nil
	}

	// Imports can be large — allow up to 5 minutes for the full operation.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)
	opts := options.InsertMany().SetOrdered(false)

	inserted := 0
	var errs []error

	for i := 0; i < len(docs); i += importBatchSize {
		end := i + importBatchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := make([]interface{}, end-i)
		for j, d := range docs[i:end] {
			batch[j] = d
		}

		res, err := col.InsertMany(ctx, batch, opts)
		if err != nil {
			errs = append(errs, err)
		}
		if res != nil {
			inserted += len(res.InsertedIDs)
		}
	}

	return inserted, errs
}
