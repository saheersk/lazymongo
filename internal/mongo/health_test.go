package mongo

import (
	"testing"
)

func TestPing_Connected(t *testing.T) {
	c, err := NewClient("mongodb://localhost:27017")
	if err != nil {
		t.Skipf("no local MongoDB available: %v", err)
	}
	defer c.Disconnect()

	latency, err := c.Ping()
	if err != nil {
		t.Fatalf("Ping() returned error: %v", err)
	}
	if latency < 0 {
		t.Fatalf("Ping() returned negative latency: %d", latency)
	}
}

func TestPing_Disconnected(t *testing.T) {
	c, err := NewClient("mongodb://localhost:27017")
	if err != nil {
		t.Skipf("no local MongoDB available: %v", err)
	}
	c.Disconnect()

	_, err = c.Ping()
	if err == nil {
		t.Fatal("expected Ping() to return an error after disconnect, got nil")
	}
}
