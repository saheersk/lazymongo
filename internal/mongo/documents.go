package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// countTimeout caps the best-effort total count. On large collections an
// exact filtered count can take far longer than fetching the page itself,
// so it gets its own short deadline instead of sharing operationTimeout.
const countTimeout = 5 * time.Second

// countTotal returns the number of documents matching filter, or -1 when the
// count fails or times out ("unknown"). An empty filter uses the metadata-based
// EstimatedDocumentCount so opening a huge collection never waits on a scan.
func countTotal(col *driver.Collection, filter bson.M) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), countTimeout)
	defer cancel()

	var (
		n   int64
		err error
	)
	if len(filter) == 0 {
		n, err = col.EstimatedDocumentCount(ctx)
	} else {
		n, err = col.CountDocuments(ctx, filter)
	}
	if err != nil {
		return -1
	}
	return n
}

// QueryOptions parameterises a Find call.
type QueryOptions struct {
	Filter     bson.M
	Sort       bson.D
	Projection bson.M
	Page       int // 0-indexed
	PageSize   int
}

// FindPage fetches a single page of documents from the named collection.
func (c *Client) FindPage(dbName, colName string, q QueryOptions) (msg.PageResult, error) {
	start := time.Now()
	ctx, cancel := opCtx()
	defer cancel()

	if q.PageSize <= 0 {
		q.PageSize = 50
	}
	if q.Filter == nil {
		q.Filter = bson.M{}
	}

	col := c.inner.Database(dbName).Collection(colName)

	// Count concurrently with the page fetch; a slow or timed-out count
	// degrades to Total = -1 rather than failing the whole page load.
	countCh := make(chan int64, 1)
	go func() { countCh <- countTotal(col, q.Filter) }()

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
		Docs:       docs,
		Total:      <-countCh,
		Page:       q.Page,
		PageSize:   q.PageSize,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// ExportDocs fetches all documents matching filter (up to limit) for export.
func (c *Client) ExportDocs(dbName, colName string, filter bson.M, sort bson.D, limit int) ([]bson.M, error) {
	ctx, cancel := opCtx()
	defer cancel()

	if filter == nil {
		filter = bson.M{}
	}

	col := c.inner.Database(dbName).Collection(colName)

	findOpts := options.Find()
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}
	if len(sort) > 0 {
		findOpts.SetSort(sort)
	}

	cursor, err := col.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("find: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("cursor: %w", err)
	}
	return docs, nil
}
