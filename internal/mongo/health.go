package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Ping measures round-trip latency to the primary.
// Returns latency in milliseconds and any error.
func (c *Client) Ping() (latencyMs int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err = c.inner.Ping(ctx, readpref.Primary())
	latencyMs = time.Since(start).Milliseconds()
	if err != nil {
		return 0, err
	}
	return latencyMs, nil
}
