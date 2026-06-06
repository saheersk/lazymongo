package mongo

import (
	"fmt"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ListDatabases returns metadata for every database visible to the client.
func (c *Client) ListDatabases() ([]msg.DatabaseInfo, error) {
	ctx, cancel := opCtx()
	defer cancel()

	result, err := c.inner.ListDatabases(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}

	infos := make([]msg.DatabaseInfo, 0, len(result.Databases))
	for _, db := range result.Databases {
		infos = append(infos, msg.DatabaseInfo{
			Name:       db.Name,
			SizeOnDisk: db.SizeOnDisk,
			Empty:      db.Empty,
		})
	}
	return infos, nil
}
