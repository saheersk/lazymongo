package mongo

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// InsertOne inserts a new document and returns the generated _id.
func (c *Client) InsertOne(dbName, colName string, doc bson.M) (interface{}, error) {
	ctx, cancel := opCtx()
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)
	res, err := col.InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	return res.InsertedID, nil
}

// ReplaceOne replaces the document identified by id with replacement.
// The _id field is stripped from replacement before sending to avoid
// the "immutable field _id" error.
func (c *Client) ReplaceOne(dbName, colName string, id interface{}, replacement bson.M) error {
	ctx, cancel := opCtx()
	defer cancel()

	repl := make(bson.M, len(replacement))
	for k, v := range replacement {
		if k != "_id" {
			repl[k] = v
		}
	}

	col := c.inner.Database(dbName).Collection(colName)
	_, err := col.ReplaceOne(ctx, bson.M{"_id": id}, repl)
	return err
}

// DeleteOne deletes the document with the given _id.
func (c *Client) DeleteOne(dbName, colName string, id interface{}) error {
	ctx, cancel := opCtx()
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)
	_, err := col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteMany deletes all documents whose _id is in ids.
// Returns the number of documents deleted.
func (c *Client) DeleteMany(dbName, colName string, ids []interface{}) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ctx, cancel := opCtx()
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)
	res, err := col.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}
