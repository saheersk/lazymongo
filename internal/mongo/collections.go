package mongo

import (
	"fmt"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ListCollections returns metadata for every collection in the given database.
func (c *Client) ListCollections(dbName string) ([]msg.CollectionInfo, error) {
	ctx, cancel := opCtx()
	defer cancel()

	db := c.inner.Database(dbName)
	specs, err := db.ListCollectionSpecifications(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("list collections %q: %w", dbName, err)
	}

	infos := make([]msg.CollectionInfo, 0, len(specs))
	for _, s := range specs {
		infos = append(infos, msg.CollectionInfo{
			Name: s.Name,
			Type: s.Type,
		})
	}
	return infos, nil
}
