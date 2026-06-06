package mongo

import (
	"fmt"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// QueryOptions parameterises a Find call.
type QueryOptions struct {
	Filter     bson.M
	Sort       bson.D
	Projection bson.M
	Page       int  // 0-indexed
	PageSize   int
}

// FindPage fetches a single page of documents from the named collection.
func (c *Client) FindPage(dbName, colName string, q QueryOptions) (msg.PageResult, error) {
	ctx, cancel := opCtx()
	defer cancel()

	if q.PageSize <= 0 {
		q.PageSize = 50
	}
	if q.Filter == nil {
		q.Filter = bson.M{}
	}

	col := c.inner.Database(dbName).Collection(colName)

	total, err := col.CountDocuments(ctx, q.Filter)
	if err != nil {
		return msg.PageResult{}, fmt.Errorf("count: %w", err)
	}

	skip := int64(q.Page * q.PageSize)
	limit := int64(q.PageSize)

	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(limit)

	if len(q.Sort) > 0 {
		findOpts.SetSort(q.Sort)
	}
	if len(q.Projection) > 0 {
		findOpts.SetProjection(q.Projection)
	}

	cursor, err := col.Find(ctx, q.Filter, findOpts)
	if err != nil {
		return msg.PageResult{}, fmt.Errorf("find: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return msg.PageResult{}, fmt.Errorf("cursor: %w", err)
	}

	return msg.PageResult{
		Docs:     docs,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}
