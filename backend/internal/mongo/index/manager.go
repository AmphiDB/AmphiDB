package index

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mygui/backend/types"
)

// Manager handles index operations for a MongoDB connection.
type Manager struct {
	client *mongo.Client
}

// NewManager creates a new index Manager for the given client.
func NewManager(client *mongo.Client) *Manager {
	return &Manager{client: client}
}

// ListIndexes returns all indexes for the specified collection.
func (m *Manager) ListIndexes(ctx context.Context, dbName, collName string) ([]types.MongoIndex, error) {
	coll := m.client.Database(dbName).Collection(collName)
	cursor, err := coll.Indexes().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var indexes []types.MongoIndex
	for cursor.Next(ctx) {
		var spec bson.M
		if err := cursor.Decode(&spec); err != nil {
			return nil, fmt.Errorf("failed to decode index spec: %w", err)
		}

		idx := types.MongoIndex{
			Keys: make(map[string]int),
		}

		// Extract name
		if name, ok := spec["name"].(string); ok {
			idx.Name = name
		}

		// Extract unique flag
		if unique, ok := spec["unique"].(bool); ok {
			idx.Unique = unique
		}

		// Extract sparse flag
		if sparse, ok := spec["sparse"].(bool); ok {
			idx.Sparse = sparse
		}

		// Extract keys and determine index type
		if keyDoc, ok := spec["key"].(bson.M); ok {
			hasText := false
			hasGeo := false

			for field, dir := range keyDoc {
				switch v := dir.(type) {
				case int32:
					idx.Keys[field] = int(v)
				case int64:
					idx.Keys[field] = int(v)
				case float64:
					idx.Keys[field] = int(v)
				case string:
					if v == "text" {
						hasText = true
						idx.Keys[field] = 1
					} else if v == "2dsphere" || v == "2d" {
						hasGeo = true
						idx.Keys[field] = 1
					} else {
						idx.Keys[field] = 1
					}
				}
			}

			// Determine index type
			switch {
			case hasText:
				idx.Type = "text"
			case hasGeo:
				idx.Type = "geospatial"
			case len(idx.Keys) == 1:
				idx.Type = "single"
			default:
				idx.Type = "compound"
			}
		}

		indexes = append(indexes, idx)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if indexes == nil {
		indexes = []types.MongoIndex{}
	}
	return indexes, nil
}

// CreateIndex creates an index on the specified collection and returns the index name.
func (m *Manager) CreateIndex(ctx context.Context, dbName, collName string, spec types.MongoIndexSpec) (string, error) {
	// Build the key document preserving order
	keys := bson.D{}
	for field, dir := range spec.Keys {
		keys = append(keys, bson.E{Key: field, Value: int32(dir)})
	}

	indexModel := mongo.IndexModel{
		Keys: keys,
	}

	opts := options.Index()
	if spec.Unique {
		opts.SetUnique(true)
	}
	if spec.Sparse {
		opts.SetSparse(true)
	}
	if spec.Name != "" {
		opts.SetName(spec.Name)
	}
	indexModel.Options = opts

	coll := m.client.Database(dbName).Collection(collName)
	name, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return "", fmt.Errorf("failed to create index: %w", err)
	}
	return name, nil
}

// DropIndex drops the named index from the specified collection.
// Returns an error if the caller attempts to drop the _id index.
func (m *Manager) DropIndex(ctx context.Context, dbName, collName, indexName string) error {
	if indexName == "_id_" {
		return fmt.Errorf("cannot drop the _id index")
	}

	coll := m.client.Database(dbName).Collection(collName)
	if _, err := coll.Indexes().DropOne(ctx, indexName); err != nil {
		return fmt.Errorf("failed to drop index %q: %w", indexName, err)
	}
	return nil
}
