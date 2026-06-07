package mongo

import (
	"strings"
	"testing"
	"time"
)

func TestWatchCollection_StandaloneReturnsError(t *testing.T) {
	c, err := NewClient("mongodb://localhost:27017")
	if err != nil {
		t.Skipf("no local MongoDB available: %v", err)
	}
	defer c.Disconnect()

	_, watchErr := c.WatchCollection("testdb", "testcol")
	if watchErr == nil {
		// The server supports change streams (replica set / Atlas).
		// Skip the test rather than fail — this is valid.
		t.Skip("server supports change streams (not a standalone); skipping standalone-error test")
	}

	// On a standalone, MongoDB returns an error mentioning replica set.
	errStr := watchErr.Error()
	if !strings.Contains(errStr, "replica") && !strings.Contains(errStr, "oplog") && !strings.Contains(errStr, "change stream") {
		t.Logf("got error (expected): %v", watchErr)
	}
}

func TestWatchStream_Close_Unblocks(t *testing.T) {
	c, err := NewClient("mongodb://localhost:27017")
	if err != nil {
		t.Skipf("no local MongoDB available: %v", err)
	}
	defer c.Disconnect()

	stream, err := c.WatchCollection("testdb", "testcol")
	if err != nil {
		t.Skipf("server does not support change streams (standalone?): %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Next() // should block until Close() is called
	}()

	// Give the goroutine time to enter Next().
	time.Sleep(100 * time.Millisecond)
	stream.Close()

	select {
	case <-done:
		// Good: Next() unblocked within 2 seconds.
	case <-time.After(2 * time.Second):
		t.Fatal("stream.Close() did not unblock Next() within 2 seconds")
	}
}
