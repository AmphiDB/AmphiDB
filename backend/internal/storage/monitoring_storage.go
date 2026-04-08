package storage

import (
	"database/sql"
	"fmt"

	"mygui/backend/types"
)

// GetExecutionLog returns a merged, timestamp-DESC sorted list of execution log entries
// from query_history (MySQL) and mongo_query_history (MongoDB).
//
// dbType controls which table(s) are queried:
//   - "mysql"   → query_history only
//   - "mongodb" → mongo_query_history only
//   - "" / "all" → UNION of both tables
func (cs *ConfigStorage) GetExecutionLog(profileID string, dbType string, limit int) ([]types.ExecutionLogEntry, error) {
	var query string
	var args []interface{}

	switch dbType {
	case "mysql":
		query = `
		SELECT id, timestamp, connection_id, 'mysql' AS db_type,
		       COALESCE(database, '') AS database,
		       '' AS collection,
		       sql AS query_text,
		       COALESCE(execution_time, 0) AS execution_time,
		       COALESCE(rows_affected, 0) AS rows_affected,
		       success
		FROM query_history
		WHERE connection_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`
		args = []interface{}{profileID, limit}

	case "mongodb":
		query = `
		SELECT id, timestamp, connection_id, 'mongodb' AS db_type,
		       COALESCE(database, '') AS database,
		       COALESCE(collection, '') AS collection,
		       pipeline AS query_text,
		       COALESCE(execution_time, 0) AS execution_time,
		       0 AS rows_affected,
		       success
		FROM mongo_query_history
		WHERE connection_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`
		args = []interface{}{profileID, limit}

	case "", "all":
		query = `
		SELECT id, timestamp, connection_id, 'mysql' AS db_type,
		       COALESCE(database, '') AS database,
		       '' AS collection,
		       sql AS query_text,
		       COALESCE(execution_time, 0) AS execution_time,
		       COALESCE(rows_affected, 0) AS rows_affected,
		       success
		FROM query_history
		WHERE connection_id = ?
		UNION ALL
		SELECT id, timestamp, connection_id, 'mongodb' AS db_type,
		       COALESCE(database, '') AS database,
		       COALESCE(collection, '') AS collection,
		       pipeline AS query_text,
		       COALESCE(execution_time, 0) AS execution_time,
		       0 AS rows_affected,
		       success
		FROM mongo_query_history
		WHERE connection_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`
		args = []interface{}{profileID, profileID, limit}

	default:
		return nil, fmt.Errorf("invalid dbType %q: must be \"mysql\", \"mongodb\", or \"all\"", dbType)
	}

	rows, err := cs.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution log: %w", err)
	}
	defer rows.Close()

	entries := []types.ExecutionLogEntry{}

	for rows.Next() {
		var e types.ExecutionLogEntry
		var successInt int
		var database, collection sql.NullString

		if err := rows.Scan(
			&e.ID,
			&e.Timestamp,
			&e.ConnectionID,
			&e.DBType,
			&database,
			&collection,
			&e.QueryText,
			&e.ExecutionTime,
			&e.RowsAffected,
			&successInt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan execution log entry: %w", err)
		}

		if database.Valid {
			e.Database = database.String
		}
		if collection.Valid {
			e.Collection = collection.String
		}
		e.Success = successInt == 1

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating execution log: %w", err)
	}

	return entries, nil
}

// ClearExecutionLog deletes execution log entries for the given profile and database type.
//
// dbType controls which table(s) are cleared:
//   - "mysql"   → query_history only
//   - "mongodb" → mongo_query_history only
//   - "" / "all" → both tables
func (cs *ConfigStorage) ClearExecutionLog(profileID string, dbType string) error {
	switch dbType {
	case "mysql":
		if _, err := cs.db.Exec(`DELETE FROM query_history WHERE connection_id = ?`, profileID); err != nil {
			return fmt.Errorf("failed to clear mysql execution log: %w", err)
		}

	case "mongodb":
		if _, err := cs.db.Exec(`DELETE FROM mongo_query_history WHERE connection_id = ?`, profileID); err != nil {
			return fmt.Errorf("failed to clear mongodb execution log: %w", err)
		}

	case "", "all":
		if _, err := cs.db.Exec(`DELETE FROM query_history WHERE connection_id = ?`, profileID); err != nil {
			return fmt.Errorf("failed to clear mysql execution log: %w", err)
		}
		if _, err := cs.db.Exec(`DELETE FROM mongo_query_history WHERE connection_id = ?`, profileID); err != nil {
			return fmt.Errorf("failed to clear mongodb execution log: %w", err)
		}

	default:
		return fmt.Errorf("invalid dbType %q: must be \"mysql\", \"mongodb\", or \"all\"", dbType)
	}

	return nil
}

// InsertSlowQuery inserts a slow query entry and auto-trims to 5000 rows per profile.
func (cs *ConfigStorage) InsertSlowQuery(entry types.SlowQueryEntry) error {
	_, err := cs.db.Exec(`
		INSERT INTO slow_query_log
		    (connection_id, db_type, database, collection, query_text, duration_ms, rows_affected, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ConnectionID,
		entry.DBType,
		entry.Database,
		entry.Collection,
		entry.QueryText,
		entry.DurationMs,
		entry.RowsAffected,
		entry.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to insert slow query entry: %w", err)
	}
	// Auto-trim: keep only the 5000 most recent rows per profile
	_, _ = cs.db.Exec(`
		DELETE FROM slow_query_log
		WHERE connection_id = ? AND id NOT IN (
			SELECT id FROM slow_query_log WHERE connection_id = ? ORDER BY timestamp DESC LIMIT 5000
		)`, entry.ConnectionID, entry.ConnectionID)
	return nil
}

// GetSlowQueryLog returns slow query entries for the given profile, ordered by duration DESC (Top N slowest).
// If profileID is empty, returns entries across all profiles.
func (cs *ConfigStorage) GetSlowQueryLog(profileID string, limit int) ([]types.SlowQueryEntry, error) {
	var query string
	var args []interface{}

	if profileID == "" {
		query = `
		SELECT id, timestamp, connection_id, db_type,
		       COALESCE(database, '') AS database,
		       COALESCE(collection, '') AS collection,
		       query_text, duration_ms,
		       COALESCE(rows_affected, 0) AS rows_affected,
		       COALESCE(error_message, '') AS error_message
		FROM slow_query_log
		ORDER BY duration_ms DESC
		LIMIT ?`
		args = []interface{}{limit}
	} else {
		query = `
		SELECT id, timestamp, connection_id, db_type,
		       COALESCE(database, '') AS database,
		       COALESCE(collection, '') AS collection,
		       query_text, duration_ms,
		       COALESCE(rows_affected, 0) AS rows_affected,
		       COALESCE(error_message, '') AS error_message
		FROM slow_query_log
		WHERE connection_id = ?
		ORDER BY duration_ms DESC
		LIMIT ?`
		args = []interface{}{profileID, limit}
	}

	rows, err := cs.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query slow query log: %w", err)
	}
	defer rows.Close()

	entries := []types.SlowQueryEntry{}
	for rows.Next() {
		var e types.SlowQueryEntry
		if err := rows.Scan(
			&e.ID,
			&e.Timestamp,
			&e.ConnectionID,
			&e.DBType,
			&e.Database,
			&e.Collection,
			&e.QueryText,
			&e.DurationMs,
			&e.RowsAffected,
			&e.ErrorMessage,
		); err != nil {
			return nil, fmt.Errorf("failed to scan slow query entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating slow query log: %w", err)
	}
	return entries, nil
}

// TrimSlowQueryLog removes oldest entries keeping only the most recent maxRows rows per profile.
func (cs *ConfigStorage) TrimSlowQueryLog(profileID string, maxRows int) error {
	var err error
	if profileID == "" {
		_, err = cs.db.Exec(`
			DELETE FROM slow_query_log
			WHERE id NOT IN (
				SELECT id FROM slow_query_log ORDER BY timestamp DESC LIMIT ?
			)`, maxRows)
	} else {
		_, err = cs.db.Exec(`
			DELETE FROM slow_query_log
			WHERE connection_id = ? AND id NOT IN (
				SELECT id FROM slow_query_log WHERE connection_id = ? ORDER BY timestamp DESC LIMIT ?
			)`, profileID, profileID, maxRows)
	}
	if err != nil {
		return fmt.Errorf("failed to trim slow query log: %w", err)
	}
	return nil
}

// ClearSlowQueryLog deletes all slow query entries for the given profile.
func (cs *ConfigStorage) ClearSlowQueryLog(profileID string) error {
	if _, err := cs.db.Exec(`DELETE FROM slow_query_log WHERE connection_id = ?`, profileID); err != nil {
		return fmt.Errorf("failed to clear slow query log: %w", err)
	}
	return nil
}

// SetSlowQueryThreshold stores the slow query threshold (in ms) for the given profile.
// Returns an error if thresholdMs is <= 0.
func (cs *ConfigStorage) SetSlowQueryThreshold(profileID string, thresholdMs int) error {
	if thresholdMs <= 0 {
		return fmt.Errorf("thresholdMs must be greater than 0, got %d", thresholdMs)
	}
	key := fmt.Sprintf("slow_query_threshold:%s", profileID)
	_, err := cs.db.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, fmt.Sprintf("%d", thresholdMs),
	)
	if err != nil {
		return fmt.Errorf("failed to set slow query threshold: %w", err)
	}
	return nil
}

// GetSlowQueryThreshold returns the slow query threshold (in ms) for the given profile.
// If no threshold has been set, it returns the default of 1000 ms.
func (cs *ConfigStorage) GetSlowQueryThreshold(profileID string) (int, error) {
	key := fmt.Sprintf("slow_query_threshold:%s", profileID)
	var value string
	err := cs.db.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return 200, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get slow query threshold: %w", err)
	}
	var threshold int
	if _, err := fmt.Sscanf(value, "%d", &threshold); err != nil {
		return 0, fmt.Errorf("invalid threshold value %q: %w", value, err)
	}
	return threshold, nil
}
