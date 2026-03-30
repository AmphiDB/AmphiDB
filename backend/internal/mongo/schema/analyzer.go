package schema

import (
	"context"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"mygui/backend/types"
)

// Analyzer 集合 Schema 分析器
type Analyzer struct {
	client *mongo.Client
}

// NewAnalyzer 创建新的 Schema 分析器
func NewAnalyzer(client *mongo.Client) *Analyzer {
	return &Analyzer{client: client}
}

// AnalyzeSchema 分析集合的字段结构，采样最多 min(sampleSize, 1000) 条文档
func (a *Analyzer) AnalyzeSchema(ctx context.Context, dbName, collName string, sampleSize int) (*types.MongoSchemaAnalysis, error) {
	// Cap sampleSize to 1000
	if sampleSize > 1000 {
		sampleSize = 1000
	}
	if sampleSize <= 0 {
		sampleSize = 100
	}

	collection := a.client.Database(dbName).Collection(collName)

	// Get total document count
	totalDocs, err := collection.CountDocuments(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	// Use $sample aggregation to sample documents
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: int64(sampleSize)}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Track field occurrence counts and type distributions
	fieldCounts := make(map[string]int)
	fieldTypeCounts := make(map[string]map[string]int)
	actualSampled := 0

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		actualSampled++

		for key, val := range doc {
			fieldCounts[key]++
			typeName := goTypeName(val)
			if fieldTypeCounts[key] == nil {
				fieldTypeCounts[key] = make(map[string]int)
			}
			fieldTypeCounts[key][typeName]++
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// Build result fields
	fields := make([]types.MongoSchemaField, 0, len(fieldCounts))
	for fieldName, count := range fieldCounts {
		frequency := float64(count) / float64(actualSampled)

		typeDistribution := make(map[string]float64)
		for typeName, typeCount := range fieldTypeCounts[fieldName] {
			typeDistribution[typeName] = float64(typeCount) / float64(count)
		}

		fields = append(fields, types.MongoSchemaField{
			Name:      fieldName,
			Frequency: frequency,
			Types:     typeDistribution,
		})
	}

	return &types.MongoSchemaAnalysis{
		Collection: collName,
		SampleSize: actualSampled,
		TotalDocs:  totalDocs,
		Fields:     fields,
	}, nil
}

// goTypeName 将 Go 值映射为可读的类型名称
func goTypeName(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case bool:
		return "Boolean"
	case int32, int64, float64:
		return "Number"
	case string:
		return "String"
	case primitive.ObjectID:
		return "ObjectId"
	case primitive.DateTime:
		return "Date"
	case bson.M, bson.D:
		return "Object"
	case primitive.A:
		return "Array"
	default:
		t := reflect.TypeOf(v)
		if t == nil {
			return "null"
		}
		return t.String()
	}
}
