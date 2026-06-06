package mongo

import (
	"context"
	"fmt"
	"time"

	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	connectTimeout    = 10 * time.Second
	operationTimeout  = 30 * time.Second
)

// Client wraps the mongo driver client with convenience helpers.
type Client struct {
	inner *driver.Client
	uri   string
}

// NewClient connects to MongoDB at the given URI and verifies reachability.
func NewClient(uri string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	opts := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(connectTimeout).
		SetServerSelectionTimeout(connectTimeout).
		SetAppName("lazymongo")

	inner, err := driver.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := inner.Ping(ctx, readpref.Primary()); err != nil {
		_ = inner.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return &Client{inner: inner, uri: uri}, nil
}

// Disconnect cleanly closes all driver connections.
func (c *Client) Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.inner.Disconnect(ctx)
}

// URI returns the connection URI this client was opened with.
func (c *Client) URI() string { return c.uri }

// DB returns a handle to a specific database.
func (c *Client) DB(name string) *driver.Database {
	return c.inner.Database(name)
}

// opCtx returns a context appropriate for a single DB operation.
func opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), operationTimeout)
}
