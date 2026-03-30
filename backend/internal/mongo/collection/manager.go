package collection

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"mygui/backend/types"
)

// Manager handles database and collection operations for a MongoDB connection.
type Manager struct {
	client *mongo.Client
}

// NewManager creates a new collection Manager for the given client.
func NewManager(client *mongo.Client) *Manager {
	return &Manager{client: client}
}

// ListDatabases returns all databases with their sizes on disk.
func (m *Manager) ListDatabases(ctx context.Context) ([]types.MongoDatabase, error) {
	result, err := m.client.ListDatabases(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	dbs := make([]types.MongoDatabase, 0, len(result.Databases))
	for _, db := range result.Databases {
		dbs = append(dbs, types.MongoDatabase{
			Name:       db.Name,
			SizeOnDisk: db.SizeOnDisk,
		})
	}
	return dbs, nil
}

// ListCollections returns all collections in the given database, including document counts.
func (m *Manager) ListCollections(ctx context.Context, dbName string) ([]types.MongoCollection, error) {
	db := m.client.Database(dbName)

	names, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections in %q: %w", dbName, err)
	}

	collections := make([]types.MongoCollection, 0, len(names))
	for _, name := range names {
		var stats bson.M
		err := db.RunCommand(ctx, bson.D{{Key: "collStats", Value: name}}).Decode(&stats)
		var docCount int64
		if err == nil {
			switch v := stats["count"].(type) {
			case int32:
				docCount = int64(v)
			case int64:
				docCount = v
			case float64:
				docCount = int64(v)
			}
		}
		collections = append(collections, types.MongoCollection{
			Name:          name,
			DocumentCount: docCount,
		})
	}
	return collections, nil
}

// CreateCollection creates a new collection in the specified database.
func (m *Manager) CreateCollection(ctx context.Context, dbName, collName string) error {
	if err := m.client.Database(dbName).CreateCollection(ctx, collName); err != nil {
		return fmt.Errorf("failed to create collection %q in %q: %w", collName, dbName, err)
	}
	return nil
}

// DropCollection drops the specified collection from the database.
func (m *Manager) DropCollection(ctx context.Context, dbName, collName string) error {
	if err := m.client.Database(dbName).Collection(collName).Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop collection %q from %q: %w", collName, dbName, err)
	}
	return nil
}
