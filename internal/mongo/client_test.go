package mongo

import (
	"testing"
)

const testMongoURI = "mongodb://localhost:27017"

// TestNewClient_ValidURI verifies that connecting to a running local MongoDB
// succeeds. The test is skipped when MongoDB is not reachable.
func TestNewClient_ValidURI(t *testing.T) {
	c, err := NewClient(testMongoURI)
	if err != nil {
		t.Skipf("MongoDB not available at %s: %v", testMongoURI, err)
	}
	defer c.Disconnect()

	if c.URI() != testMongoURI {
		t.Errorf("URI() = %q; want %q", c.URI(), testMongoURI)
	}
}

// TestNewClient_UnreachableURI verifies that an unreachable URI results in an
// error rather than a nil error + bad client.
func TestNewClient_UnreachableURI(t *testing.T) {
	// Port 1 is almost universally refused / firewalled.
	_, err := NewClient("mongodb://localhost:1/?serverSelectionTimeoutMS=500&connectTimeoutMS=500")
	if err == nil {
		t.Error("expected error connecting to unreachable URI, got nil")
	}
}
