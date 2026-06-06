package mongo

import (
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/saheersk/lazymongo/internal/tui/msg"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ListIndexesAndStats returns all indexes and lightweight stats for a collection.
func (c *Client) ListIndexesAndStats(dbName, colName string) ([]msg.IndexInfo, msg.CollectionStats, error) {
	ctx, cancel := opCtx()
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)

	cursor, err := col.Indexes().List(ctx)
	if err != nil {
		return nil, msg.CollectionStats{}, err
	}
	defer cursor.Close(ctx)

	var rawSpecs []bson.M
	if err := cursor.All(ctx, &rawSpecs); err != nil {
		return nil, msg.CollectionStats{}, err
	}

	infos := make([]msg.IndexInfo, 0, len(rawSpecs))
	for _, s := range rawSpecs {
		info := msg.IndexInfo{TTLSeconds: -1}
		if name, ok := s["name"].(string); ok {
			info.Name = name
		}
		switch keyDoc := s["key"].(type) {
		case bson.M:
			for k, v := range keyDoc {
				info.Keys = append(info.Keys, bson.E{Key: k, Value: v})
			}
		case bson.D:
			info.Keys = append(info.Keys, keyDoc...)
		}
		if u, ok := s["unique"].(bool); ok {
			info.Unique = u
		}
		if sp, ok := s["sparse"].(bool); ok {
			info.Sparse = sp
		}
		if ttl, ok := s["expireAfterSeconds"].(int32); ok {
			info.TTLSeconds = ttl
		}
		infos = append(infos, info)
	}

	count, err := col.CountDocuments(ctx, bson.D{})
	if err != nil {
		return infos, msg.CollectionStats{}, err
	}

	stats := msg.CollectionStats{
		DocCount:   count,
		IndexCount: len(infos),
	}
	return infos, stats, nil
}

// CreateIndex creates an index on colName with the given key document.
func (c *Client) CreateIndex(dbName, colName string, keys bson.D, unique, sparse bool) (string, error) {
	ctx, cancel := opCtx()
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)

	idxOpts := options.Index()
	if unique {
		idxOpts.SetUnique(true)
	}
	if sparse {
		idxOpts.SetSparse(true)
	}

	model := driver.IndexModel{Keys: keys, Options: idxOpts}
	name, err := col.Indexes().CreateOne(ctx, model)
	if err != nil {
		return "", err
	}
	return name, nil
}

// DropIndex drops the named index from colName.
func (c *Client) DropIndex(dbName, colName, name string) error {
	ctx, cancel := opCtx()
	defer cancel()

	col := c.inner.Database(dbName).Collection(colName)
	err := col.Indexes().DropOne(ctx, name)
	return err
}
