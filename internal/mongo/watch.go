package mongo

import (
	"context"
	"fmt"
	"time"

	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ChangeEvent describes one change stream event.
type ChangeEvent struct {
	OperationType string      // "insert", "update", "replace", "delete", "invalidate"
	DocID         interface{} // _id of the affected document
	Doc           bson.M      // full document (insert/replace, or updateLookup for update)
	UpdatedFields bson.M      // non-nil for update events
	Timestamp     time.Time
}

// WatchStream wraps a MongoDB change stream with a cancellable context.
type WatchStream struct {
	stream *driver.ChangeStream
	ctx    context.Context
	cancel context.CancelFunc
}

// WatchCollection opens a change stream on db.col.
// Uses UpdateLookup so update events include the full document.
// Returns an error immediately if change streams are not supported (e.g., standalone).
func (c *Client) WatchCollection(db, col string) (*WatchStream, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pipeline := driver.Pipeline{}
	opts := options.ChangeStream().
		SetFullDocument(options.UpdateLookup)

	stream, err := c.inner.Database(db).Collection(col).Watch(ctx, pipeline, opts)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("watch collection: %w", err)
	}

	return &WatchStream{
		stream: stream,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Next blocks until the next event or the stream is closed/cancelled.
// Returns (event, true) on success; (zero, false) when done.
func (s *WatchStream) Next() (ChangeEvent, bool) {
	if !s.stream.Next(s.ctx) {
		return ChangeEvent{}, false
	}

	var raw bson.M
	if err := s.stream.Decode(&raw); err != nil {
		return ChangeEvent{}, false
	}

	ev := ChangeEvent{
		Timestamp: time.Now(),
	}

	if op, ok := raw["operationType"].(string); ok {
		ev.OperationType = op
	}

	// Extract document ID from documentKey._id
	if dk, ok := raw["documentKey"].(bson.M); ok {
		ev.DocID = dk["_id"]
	}

	// Full document (insert, replace, or updateLookup result)
	if fd, ok := raw["fullDocument"].(bson.M); ok {
		ev.Doc = fd
	}

	// Updated fields (update events)
	if ud, ok := raw["updateDescription"].(bson.M); ok {
		if uf, ok := ud["updatedFields"].(bson.M); ok {
			ev.UpdatedFields = uf
		}
	}

	return ev, true
}

// Err returns the stream error (nil if cleanly cancelled).
func (s *WatchStream) Err() error {
	err := s.stream.Err()
	// If the context was cancelled, don't surface that as an error.
	if s.ctx.Err() != nil && err == context.Canceled {
		return nil
	}
	return err
}

// Close cancels the context and closes the stream.
func (s *WatchStream) Close() {
	s.cancel()
	_ = s.stream.Close(context.Background())
}
