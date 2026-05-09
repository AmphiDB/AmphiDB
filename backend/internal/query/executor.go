package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// QueryType represents the type of SQL query
type QueryType string

const (
	QueryTypeSelect QueryType = "SELECT"
	QueryTypeInsert QueryType = "INSERT"
	QueryTypeUpdate QueryType = "UPDATE"
	QueryTypeDelete QueryType = "DELETE"
	QueryTypeDDL    QueryType = "DDL"
	QueryTypeOther  QueryType = "OTHER"
)

// QueryResult represents the result of a query execution
type QueryResult struct {
	ID            string          `json:"id"`
	Type          QueryType       `json:"type"`
	Columns       []string        `json:"columns,omitempty"`
	Rows          [][]interface{} `json:"rows,omitempty"`
	RowsAffected  int64           `json:"rowsAffected"`
	ExecutionTime time.Duration   `json:"executionTime"`
	Error         *QueryError     `json:"error,omitempty"`
}

// QueryError represents an error that occurred during query execution
type QueryError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	Position int    `json:"position"`
}

// Executor executes SQL queries with timeout and cancellation support
type Executor struct {
	db              *sql.DB
	defaultTimeout  time.Duration
	activeQueries   map[string]context.CancelFunc
	activeQueriesMu sync.RWMutex
}

// NewExecutor creates a new query executor
func NewExecutor(db *sql.DB) *Executor {
	return &Executor{
		db:             db,
		defaultTimeout: 30 * time.Second,
		activeQueries:  make(map[string]context.CancelFunc),
	}
}

// Execute executes a SQL query with the default timeout
func (e *Executor) Execute(sql string) (*QueryResult, error) {
	return e.ExecuteWithTimeout(sql, e.defaultTimeout)
}

// ExecuteWithTimeout executes a SQL query with a specified timeout
func (e *Executor) ExecuteWithTimeout(sqlQuery string, timeout time.Duration) (*QueryResult, error) {
	startTime := time.Now()
	queryID := uuid.New().String()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Register the query for cancellation
	e.registerQuery(queryID, cancel)
	defer e.unregisterQuery(queryID)

	result := &QueryResult{
		ID: queryID,
	}

	// Determine query type
	queryType := e.determineQueryType(sqlQuery)
	result.Type = queryType

	// Execute based on query type
	var err error
	switch queryType {
	case QueryTypeSelect:
		err = e.executeSelect(ctx, sqlQuery, result)
	case QueryTypeInsert, QueryTypeUpdate, QueryTypeDelete:
		err = e.executeDML(ctx, sqlQuery, result)
	case QueryTypeDDL, QueryTypeOther:
		err = e.executeDDL(ctx, sqlQuery, result)
	}

	result.ExecutionTime = time.Since(startTime)

	if err != nil {
		result.Error = e.parseError(err)
		return result, err
	}

	return result, nil
}

// Cancel cancels a running query by its ID
func (e *Executor) Cancel(queryID string) error {
	e.activeQueriesMu.RLock()
	cancel, exists := e.activeQueries[queryID]
	e.activeQueriesMu.RUnlock()

	if !exists {
		return fmt.Errorf("query %s not found or already completed", queryID)
	}

	cancel()
	return nil
}

// executeSelect executes a SELECT query and populates the result
func (e *Executor) executeSelect(ctx context.Context, sqlQuery string, result *QueryResult) error {
	rows, err := e.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	result.Columns = columns

	// Get column types
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	// Fetch all rows
	var allRows [][]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold each column value
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		// Convert values to appropriate types
		row := make([]interface{}, len(columns))
		for i, val := range values {
			row[i] = e.convertValue(val, columnTypes[i])
		}

		allRows = append(allRows, row)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	result.Rows = allRows
	result.RowsAffected = int64(len(allRows))

	return nil
}

// executeDML executes INSERT, UPDATE, or DELETE queries
func (e *Executor) executeDML(ctx context.Context, sqlQuery string, result *QueryResult) error {
	res, err := e.db.ExecContext(ctx, sqlQuery)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	result.RowsAffected = rowsAffected
	return nil
}

// executeDDL executes DDL statements (CREATE, ALTER, DROP, etc.)
func (e *Executor) executeDDL(ctx context.Context, sqlQuery string, result *QueryResult) error {
	_, err := e.db.ExecContext(ctx, sqlQuery)
	if err != nil {
		return err
	}

	result.RowsAffected = 0
	return nil
}

// determineQueryType determines the type of SQL query
func (e *Executor) determineQueryType(sqlQuery string) QueryType {
	trimmed := strings.TrimSpace(sqlQuery)
	upper := strings.ToUpper(trimmed)

	if strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "DESCRIBE") || strings.HasPrefix(upper, "DESC") ||
		strings.HasPrefix(upper, "EXPLAIN") {
		return QueryTypeSelect
	}

	if strings.HasPrefix(upper, "INSERT") {
		return QueryTypeInsert
	}

	if strings.HasPrefix(upper, "UPDATE") {
		return QueryTypeUpdate
	}

	if strings.HasPrefix(upper, "DELETE") {
		return QueryTypeDelete
	}

	if strings.HasPrefix(upper, "CREATE") || strings.HasPrefix(upper, "ALTER") ||
		strings.HasPrefix(upper, "DROP") || strings.HasPrefix(upper, "TRUNCATE") ||
		strings.HasPrefix(upper, "RENAME") {
		return QueryTypeDDL
	}

	return QueryTypeOther
}

// convertValue converts database values to appropriate Go types
func (e *Executor) convertValue(val interface{}, colType *sql.ColumnType) interface{} {
	if val == nil {
		return nil
	}

	// Handle byte arrays (common for strings and binary data)
	if b, ok := val.([]byte); ok {
		// Try to convert to string for text types
		scanType := colType.ScanType()
		if scanType != nil {
			switch scanType.Kind() {
			case 24: // reflect.String
				return string(b)
			}
		}
		return string(b)
	}

	// Handle BIGINT columns - convert to string to avoid JavaScript precision loss
	dbType := strings.ToUpper(colType.DatabaseTypeName())
	if dbType == "BIGINT" || dbType == "BIGINT UNSIGNED" {
		switch v := val.(type) {
		case int64:
			return fmt.Sprintf("%d", v)
		case uint64:
			return fmt.Sprintf("%d", v)
		}
	}

	return val
}

// parseError parses a database error into a QueryError
func (e *Executor) parseError(err error) *QueryError {
	qErr := &QueryError{
		Message:  err.Error(),
		Position: -1,
	}

	// Try to extract MySQL error code if available
	if err.Error() != "" {
		// MySQL errors typically start with "Error XXXX:"
		if strings.Contains(err.Error(), "Error ") {
			var code int
			fmt.Sscanf(err.Error(), "Error %d:", &code)
			qErr.Code = code
		}
	}

	return qErr
}

// registerQuery registers a query for cancellation
func (e *Executor) registerQuery(queryID string, cancel context.CancelFunc) {
	e.activeQueriesMu.Lock()
	defer e.activeQueriesMu.Unlock()
	e.activeQueries[queryID] = cancel
}

// unregisterQuery removes a query from the active queries map
func (e *Executor) unregisterQuery(queryID string) {
	e.activeQueriesMu.Lock()
	defer e.activeQueriesMu.Unlock()
	delete(e.activeQueries, queryID)
}

// ExecuteInDatabase executes a SQL query within a specific database context.
// It acquires a dedicated connection, runs USE `database`, then executes the query.
func (e *Executor) ExecuteInDatabase(database, sqlQuery string) (*QueryResult, error) {
	return e.ExecuteInDatabaseWithTimeout(database, sqlQuery, e.defaultTimeout)
}

// ExecuteInDatabaseWithTimeout executes a SQL query within a specific database context with a timeout.
func (e *Executor) ExecuteInDatabaseWithTimeout(database, sqlQuery string, timeout time.Duration) (*QueryResult, error) {
	startTime := time.Now()
	queryID := uuid.New().String()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	e.registerQuery(queryID, cancel)
	defer e.unregisterQuery(queryID)

	result := &QueryResult{ID: queryID}

	// Acquire a single connection so USE persists for the subsequent query
	conn, err := e.db.Conn(ctx)
	if err != nil {
		result.Error = e.parseError(err)
		return result, err
	}
	defer conn.Close()

	// Switch to the target database
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE `%s`", database)); err != nil {
		result.Error = e.parseError(err)
		return result, err
	}

	// Determine query type and execute
	queryType := e.determineQueryType(sqlQuery)
	result.Type = queryType

	switch queryType {
	case QueryTypeSelect:
		err = e.executeSelectOnConn(ctx, conn, sqlQuery, result)
	default:
		err = e.executeDMLOnConn(ctx, conn, sqlQuery, result)
	}

	result.ExecutionTime = time.Since(startTime)
	if err != nil {
		result.Error = e.parseError(err)
		return result, err
	}
	return result, nil
}

func (e *Executor) executeSelectOnConn(ctx context.Context, conn *sql.Conn, sqlQuery string, result *QueryResult) error {
	rows, err := conn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	result.Columns = columns

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	var allRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}
		row := make([]interface{}, len(columns))
		for i, val := range values {
			row[i] = e.convertValue(val, columnTypes[i])
		}
		allRows = append(allRows, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	result.Rows = allRows
	result.RowsAffected = int64(len(allRows))
	return nil
}

func (e *Executor) executeDMLOnConn(ctx context.Context, conn *sql.Conn, sqlQuery string, result *QueryResult) error {
	res, err := conn.ExecContext(ctx, sqlQuery)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	result.RowsAffected = affected
	return nil
}
