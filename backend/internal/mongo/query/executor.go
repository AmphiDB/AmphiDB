package query

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"mygui/backend/types"
)

const defaultTimeout = 30 * time.Second

// Executor runs aggregation pipelines against a MongoDB collection.
type Executor struct {
	client  *mongo.Client
	timeout time.Duration
}

// NewExecutor creates a new Executor with a 30-second default timeout.
func NewExecutor(client *mongo.Client) *Executor {
	return &Executor{
		client:  client,
		timeout: defaultTimeout,
	}
}

// RunAggregation executes an aggregation pipeline on the specified collection.
// pipelineJSON must be a JSON array of stage documents, e.g. [{"$match":{"status":"active"}}].
// It returns the result documents serialized as JSON strings along with the execution time.
func (e *Executor) RunAggregation(ctx context.Context, dbName, collName, pipelineJSON string) (*types.MongoAggregationResult, error) {
	var pipeline []bson.D
	if err := bson.UnmarshalExtJSON([]byte(pipelineJSON), true, &pipeline); err != nil {
		return nil, fmt.Errorf("invalid pipeline JSON: %w", err)
	}

	start := time.Now()

	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	coll := e.client.Database(dbName).Collection(collName)
	cursor, err := coll.Aggregate(timeoutCtx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation failed: %w", err)
	}
	defer cursor.Close(timeoutCtx)

	var docs []string
	for cursor.Next(timeoutCtx) {
		var raw bson.Raw
		if err := cursor.Decode(&raw); err != nil {
			return nil, fmt.Errorf("failed to decode result: %w", err)
		}
		jsonBytes, err := bson.MarshalExtJSON(raw, false, false)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize result: %w", err)
		}
		docs = append(docs, string(jsonBytes))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	executionTime := time.Since(start)

	if docs == nil {
		docs = []string{}
	}

	return &types.MongoAggregationResult{
		Documents:     docs,
		ExecutionTime: executionTime,
	}, nil
}
