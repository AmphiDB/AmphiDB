package document

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mygui/backend/types"
)

// Manager handles document CRUD operations for a MongoDB connection.
type Manager struct {
	client *mongo.Client
}

// NewManager creates a new document Manager for the given client.
func NewManager(client *mongo.Client) *Manager {
	return &Manager{client: client}
}

// parseJSONToBsonD parses a JSON string into a bson.D document.
// Returns an empty bson.D if the input string is empty.
func parseJSONToBsonD(jsonStr string) (bson.D, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return bson.D{}, nil
	}
	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(jsonStr), true, &doc); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return doc, nil
}

// QueryDocuments queries documents in a collection with filter, sort, projection and pagination.
func (m *Manager) QueryDocuments(ctx context.Context, params types.MongoQueryParams) (*types.MongoDocumentResult, error) {
	filter, err := parseJSONToBsonD(params.Filter)
	if err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	sort, err := parseJSONToBsonD(params.Sort)
	if err != nil {
		return nil, fmt.Errorf("invalid sort: %w", err)
	}

	projection, err := parseJSONToBsonD(params.Projection)
	if err != nil {
		return nil, fmt.Errorf("invalid projection: %w", err)
	}

	coll := m.client.Database(params.Database).Collection(params.Collection)

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 50
	}

	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(limit)

	if len(sort) > 0 {
		findOpts.SetSort(sort)
	}
	if len(projection) > 0 {
		findOpts.SetProjection(projection)
	}

	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []string
	for cursor.Next(ctx) {
		var raw bson.Raw
		if err := cursor.Decode(&raw); err != nil {
			return nil, fmt.Errorf("failed to decode document: %w", err)
		}
		jsonBytes, err := bson.MarshalExtJSON(raw, false, false)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize document: %w", err)
		}
		docs = append(docs, string(jsonBytes))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if docs == nil {
		docs = []string{}
	}

	return &types.MongoDocumentResult{
		Documents: docs,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

// InsertDocument inserts a JSON document into the specified collection and returns the inserted _id as a string.
func (m *Manager) InsertDocument(ctx context.Context, dbName, collName, jsonDoc string) (string, error) {
	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(jsonDoc), true, &doc); err != nil {
		return "", fmt.Errorf("invalid document JSON: %w", err)
	}

	coll := m.client.Database(dbName).Collection(collName)
	result, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("failed to insert document: %w", err)
	}

	return fmt.Sprintf("%v", result.InsertedID), nil
}

// UpdateDocument replaces the document identified by docID with the provided JSON document.
func (m *Manager) UpdateDocument(ctx context.Context, dbName, collName, docID, jsonDoc string) error {
	var replacement bson.D
	if err := bson.UnmarshalExtJSON([]byte(jsonDoc), true, &replacement); err != nil {
		return fmt.Errorf("invalid document JSON: %w", err)
	}

	filter, err := buildIDFilter(docID)
	if err != nil {
		return fmt.Errorf("invalid document ID: %w", err)
	}

	coll := m.client.Database(dbName).Collection(collName)
	result, err := coll.ReplaceOne(ctx, filter, replacement)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("document not found: %s", docID)
	}
	return nil
}

// DeleteDocuments deletes documents by their IDs and returns the number of deleted documents.
func (m *Manager) DeleteDocuments(ctx context.Context, dbName, collName string, docIDs []string) (int64, error) {
	if len(docIDs) == 0 {
		return 0, nil
	}

	ids := make(bson.A, 0, len(docIDs))
	for _, id := range docIDs {
		parsed, err := parseDocID(id)
		if err != nil {
			return 0, fmt.Errorf("invalid document ID %q: %w", id, err)
		}
		ids = append(ids, parsed)
	}

	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}}
	coll := m.client.Database(dbName).Collection(collName)
	result, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete documents: %w", err)
	}
	return result.DeletedCount, nil
}

// buildIDFilter creates a filter document for matching a single _id.
func buildIDFilter(docID string) (bson.D, error) {
	id, err := parseDocID(docID)
	if err != nil {
		return nil, err
	}
	return bson.D{{Key: "_id", Value: id}}, nil
}

// parseDocID parses a document ID string, returning an ObjectID if it looks like one,
// otherwise returning the raw string.
func parseDocID(docID string) (interface{}, error) {
	// Try to parse as ObjectID (24-char hex string)
	if oid, err := primitive.ObjectIDFromHex(docID); err == nil {
		return oid, nil
	}
	// Try to parse as a JSON value (number, boolean, etc.)
	var v interface{}
	if err := json.Unmarshal([]byte(docID), &v); err == nil {
		return v, nil
	}
	// Fall back to raw string
	return docID, nil
}

// ExportDocuments exports documents matching the filter to a file in JSON or CSV format.
// It returns the file path on success.
func (m *Manager) ExportDocuments(ctx context.Context, params types.MongoExportParams) (string, error) {
	filter, err := parseJSONToBsonD(params.Filter)
	if err != nil {
		return "", fmt.Errorf("invalid filter: %w", err)
	}

	coll := m.client.Database(params.Database).Collection(params.Collection)
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("failed to query documents: %w", err)
	}
	defer cursor.Close(ctx)

	switch strings.ToLower(params.Format) {
	case "json":
		if err := exportJSON(ctx, cursor, params.FilePath); err != nil {
			return "", err
		}
	case "csv":
		if err := exportCSV(ctx, cursor, params.FilePath); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported format %q: must be \"json\" or \"csv\"", params.Format)
	}

	return params.FilePath, nil
}

// exportJSON writes all cursor documents as a JSON array to filePath.
func exportJSON(ctx context.Context, cursor *mongo.Cursor, filePath string) error {
	f, err := createFile(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString("[\n"); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	first := true
	for cursor.Next(ctx) {
		var raw bson.Raw
		if err := cursor.Decode(&raw); err != nil {
			return fmt.Errorf("failed to decode document: %w", err)
		}
		jsonBytes, err := bson.MarshalExtJSON(raw, false, false)
		if err != nil {
			return fmt.Errorf("failed to serialize document: %w", err)
		}
		if !first {
			if _, err := f.WriteString(",\n"); err != nil {
				return fmt.Errorf("failed to write JSON: %w", err)
			}
		}
		if _, err := f.Write(jsonBytes); err != nil {
			return fmt.Errorf("failed to write JSON: %w", err)
		}
		first = false
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	if _, err := f.WriteString("\n]"); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}
	return nil
}

// exportCSV writes all cursor documents as CSV to filePath.
// Headers are inferred from all unique field names, sorted alphabetically.
func exportCSV(ctx context.Context, cursor *mongo.Cursor, filePath string) error {
	// Collect all documents first to infer headers
	var rows []map[string]any
	for cursor.Next(ctx) {
		var doc map[string]any
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("failed to decode document: %w", err)
		}
		rows = append(rows, doc)
	}
	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	// Collect unique headers
	headerSet := make(map[string]struct{})
	for _, row := range rows {
		for k := range row {
			headerSet[k] = struct{}{}
		}
	}
	headers := make([]string, 0, len(headerSet))
	for k := range headerSet {
		headers = append(headers, k)
	}
	sortStrings(headers)

	f, err := createFile(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := newCSVWriter(f)
	if err := w.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, row := range rows {
		record := make([]string, len(headers))
		for i, h := range headers {
			if v, ok := row[h]; ok {
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	w.Flush()
	return w.Error()
}

// createFile creates (or truncates) a file at the given path.
func createFile(filePath string) (*os.File, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %q: %w", filePath, err)
	}
	return f, nil
}

// sortStrings sorts a string slice in place (alphabetically).
func sortStrings(s []string) {
	sort.Strings(s)
}

// newCSVWriter returns a new csv.Writer for the given file.
func newCSVWriter(f *os.File) *csv.Writer {
	return csv.NewWriter(f)
}
