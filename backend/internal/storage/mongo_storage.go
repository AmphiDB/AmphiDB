package storage

import (
	"database/sql"
	"fmt"
	"time"

	"mygui/backend/types"
)

// MongoQueryHistoryEntry represents a MongoDB aggregation query history record
type MongoQueryHistoryEntry struct {
	ID            int       `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	ConnectionID  string    `json:"connectionId"`
	Database      string    `json:"database"`
	Collection    string    `json:"collection"`
	Pipeline      string    `json:"pipeline"`      // aggregation pipeline JSON
	ExecutionTime int64     `json:"executionTime"` // milliseconds
	Success       bool      `json:"success"`
}

// initMongoSchema creates MongoDB-related tables if they don't exist.
// Called from initSchema() in config_storage.go.
func (cs *ConfigStorage) initMongoSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS mongo_connection_profiles (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		host        TEXT,
		port        INTEGER,
		username    TEXT,
		password    TEXT NOT NULL DEFAULT '',
		auth_db     TEXT DEFAULT 'admin',
		use_uri     INTEGER DEFAULT 0,
		uri         TEXT,
		ssh_enabled INTEGER DEFAULT 0,
		ssh_host    TEXT,
		ssh_port    INTEGER,
		ssh_username TEXT,
		ssh_password TEXT,
		ssh_key_path TEXT,
		timeout     INTEGER DEFAULT 10,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS mongo_query_history (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp      DATETIME DEFAULT CURRENT_TIMESTAMP,
		connection_id  TEXT NOT NULL,
		database       TEXT,
		collection     TEXT,
		pipeline       TEXT NOT NULL,
		execution_time INTEGER,
		success        INTEGER DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_mongo_query_history_conn
		ON mongo_query_history(connection_id);
	`

	_, err := cs.db.Exec(schema)
	return err
}

// SaveMongoProfile saves or updates a MongoDB connection profile.
// Passwords are expected to be already encrypted by the caller.
func (cs *ConfigStorage) SaveMongoProfile(profile types.MongoConnectionProfile) error {
	query := `
	INSERT INTO mongo_connection_profiles (
		id, name, host, port, username, password, auth_db,
		use_uri, uri,
		ssh_enabled, ssh_host, ssh_port, ssh_username, ssh_password, ssh_key_path,
		timeout, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name        = excluded.name,
		host        = excluded.host,
		port        = excluded.port,
		username    = excluded.username,
		password    = excluded.password,
		auth_db     = excluded.auth_db,
		use_uri     = excluded.use_uri,
		uri         = excluded.uri,
		ssh_enabled = excluded.ssh_enabled,
		ssh_host    = excluded.ssh_host,
		ssh_port    = excluded.ssh_port,
		ssh_username = excluded.ssh_username,
		ssh_password = excluded.ssh_password,
		ssh_key_path = excluded.ssh_key_path,
		timeout     = excluded.timeout,
		updated_at  = excluded.updated_at
	`

	useURI := 0
	if profile.UseURI {
		useURI = 1
	}
	sshEnabled := 0
	if profile.SSHEnabled {
		sshEnabled = 1
	}

	now := time.Now()
	if profile.CreatedAt == nil || profile.CreatedAt.IsZero() {
		profile.CreatedAt = &now
	}
	profile.UpdatedAt = &now

	_, err := cs.db.Exec(query,
		profile.ID, profile.Name, profile.Host, profile.Port,
		profile.Username, profile.Password, profile.AuthDB,
		useURI, profile.URI,
		sshEnabled, profile.SSHHost, profile.SSHPort,
		profile.SSHUsername, profile.SSHPassword, profile.SSHKeyPath,
		profile.Timeout, profile.CreatedAt, profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save mongo profile: %w", err)
	}
	return nil
}

// GetMongoProfile retrieves a MongoDB connection profile by ID.
func (cs *ConfigStorage) GetMongoProfile(id string) (*types.MongoConnectionProfile, error) {
	query := `
	SELECT id, name, host, port, username, password, auth_db,
	       use_uri, uri,
	       ssh_enabled, ssh_host, ssh_port, ssh_username, ssh_password, ssh_key_path,
	       timeout, created_at, updated_at
	FROM mongo_connection_profiles
	WHERE id = ?
	`

	var p types.MongoConnectionProfile
	var useURI, sshEnabled int
	var host, username, authDB, uri sql.NullString
	var port sql.NullInt64
	var sshHost, sshUsername, sshPassword, sshKeyPath sql.NullString
	var sshPort sql.NullInt64
	var createdAt, updatedAt time.Time

	err := cs.db.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &host, &port,
		&username, &p.Password, &authDB,
		&useURI, &uri,
		&sshEnabled, &sshHost, &sshPort,
		&sshUsername, &sshPassword, &sshKeyPath,
		&p.Timeout, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mongo profile not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get mongo profile: %w", err)
	}

	p.UseURI = useURI == 1
	p.SSHEnabled = sshEnabled == 1

	if host.Valid {
		p.Host = host.String
	}
	if port.Valid {
		p.Port = int(port.Int64)
	}
	if username.Valid {
		p.Username = username.String
	}
	if authDB.Valid {
		p.AuthDB = authDB.String
	}
	if uri.Valid {
		p.URI = uri.String
	}
	if sshHost.Valid {
		p.SSHHost = sshHost.String
	}
	if sshPort.Valid {
		p.SSHPort = int(sshPort.Int64)
	}
	if sshUsername.Valid {
		p.SSHUsername = sshUsername.String
	}
	if sshPassword.Valid {
		p.SSHPassword = sshPassword.String
	}
	if sshKeyPath.Valid {
		p.SSHKeyPath = sshKeyPath.String
	}

	p.CreatedAt = &createdAt
	p.UpdatedAt = &updatedAt

	return &p, nil
}

// ListMongoProfiles retrieves all MongoDB connection profiles.
func (cs *ConfigStorage) ListMongoProfiles() ([]types.MongoConnectionProfile, error) {
	query := `
	SELECT id, name, host, port, username, password, auth_db,
	       use_uri, uri,
	       ssh_enabled, ssh_host, ssh_port, ssh_username, ssh_password, ssh_key_path,
	       timeout, created_at, updated_at
	FROM mongo_connection_profiles
	ORDER BY name
	`

	rows, err := cs.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list mongo profiles: %w", err)
	}
	defer rows.Close()

	profiles := []types.MongoConnectionProfile{}

	for rows.Next() {
		var p types.MongoConnectionProfile
		var useURI, sshEnabled int
		var host, username, authDB, uri sql.NullString
		var port sql.NullInt64
		var sshHost, sshUsername, sshPassword, sshKeyPath sql.NullString
		var sshPort sql.NullInt64
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&p.ID, &p.Name, &host, &port,
			&username, &p.Password, &authDB,
			&useURI, &uri,
			&sshEnabled, &sshHost, &sshPort,
			&sshUsername, &sshPassword, &sshKeyPath,
			&p.Timeout, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mongo profile: %w", err)
		}

		p.UseURI = useURI == 1
		p.SSHEnabled = sshEnabled == 1

		if host.Valid {
			p.Host = host.String
		}
		if port.Valid {
			p.Port = int(port.Int64)
		}
		if username.Valid {
			p.Username = username.String
		}
		if authDB.Valid {
			p.AuthDB = authDB.String
		}
		if uri.Valid {
			p.URI = uri.String
		}
		if sshHost.Valid {
			p.SSHHost = sshHost.String
		}
		if sshPort.Valid {
			p.SSHPort = int(sshPort.Int64)
		}
		if sshUsername.Valid {
			p.SSHUsername = sshUsername.String
		}
		if sshPassword.Valid {
			p.SSHPassword = sshPassword.String
		}
		if sshKeyPath.Valid {
			p.SSHKeyPath = sshKeyPath.String
		}

		p.CreatedAt = &createdAt
		p.UpdatedAt = &updatedAt

		profiles = append(profiles, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mongo profiles: %w", err)
	}

	return profiles, nil
}

// DeleteMongoProfile deletes a MongoDB connection profile by ID.
func (cs *ConfigStorage) DeleteMongoProfile(id string) error {
	result, err := cs.db.Exec(`DELETE FROM mongo_connection_profiles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete mongo profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("mongo profile not found: %s", id)
	}

	return nil
}

// SaveMongoQueryHistory saves a MongoDB aggregation query history entry.
func (cs *ConfigStorage) SaveMongoQueryHistory(connID, db, coll, pipeline string, execTime int64, success bool) error {
	query := `
	INSERT INTO mongo_query_history (timestamp, connection_id, database, collection, pipeline, execution_time, success)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	successInt := 0
	if success {
		successInt = 1
	}

	_, err := cs.db.Exec(query,
		time.Now(), connID, db, coll, pipeline, execTime, successInt,
	)
	if err != nil {
		return fmt.Errorf("failed to save mongo query history: %w", err)
	}
	return nil
}

// GetMongoQueryHistory retrieves aggregation query history for a connection.
func (cs *ConfigStorage) GetMongoQueryHistory(connID string, limit int) ([]MongoQueryHistoryEntry, error) {
	query := `
	SELECT id, timestamp, connection_id, database, collection, pipeline, execution_time, success
	FROM mongo_query_history
	WHERE connection_id = ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := cs.db.Query(query, connID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get mongo query history: %w", err)
	}
	defer rows.Close()

	var history []MongoQueryHistoryEntry

	for rows.Next() {
		var entry MongoQueryHistoryEntry
		var success int
		var database, collection sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.ConnectionID,
			&database,
			&collection,
			&entry.Pipeline,
			&entry.ExecutionTime,
			&success,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mongo query history: %w", err)
		}

		if database.Valid {
			entry.Database = database.String
		}
		if collection.Valid {
			entry.Collection = collection.String
		}
		entry.Success = success == 1

		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mongo query history: %w", err)
	}

	return history, nil
}
