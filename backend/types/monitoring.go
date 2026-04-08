package types

import "time"

// ExecutionLogEntry — unified view over query_history + mongo_query_history
type ExecutionLogEntry struct {
	ID            int       `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	ConnectionID  string    `json:"connectionId"`
	DBType        string    `json:"dbType"` // "mysql" | "mongodb"
	Database      string    `json:"database"`
	Collection    string    `json:"collection"`    // MongoDB only
	QueryText     string    `json:"queryText"`     // SQL or pipeline JSON
	ExecutionTime int64     `json:"executionTime"` // ms
	RowsAffected  int64     `json:"rowsAffected"`
	Success       bool      `json:"success"`
}

// SlowQueryEntry — row in slow_query_log
type SlowQueryEntry struct {
	ID           int       `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	ConnectionID string    `json:"connectionId"`
	DBType       string    `json:"dbType"`
	Database     string    `json:"database"`
	Collection   string    `json:"collection"`
	QueryText    string    `json:"queryText"`
	DurationMs   int64     `json:"durationMs"`
	RowsAffected int64     `json:"rowsAffected"`
	ErrorMessage string    `json:"errorMessage"`
}

// DataPoint is a single metrics sample (MySQL or MongoDB)
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`

	// Common
	QPS float64 `json:"qps"`
	TPS float64 `json:"tps"`

	// MySQL-specific
	ThreadsConnected   int64   `json:"threadsConnected"`   // Threads_connected
	ThreadsRunning     int64   `json:"threadsRunning"`     // Threads_running
	InnodbBufHitRate   float64 `json:"innodbBufHitRate"`   // buffer pool hit rate (0-100)
	InnodbRowLockWaits int64   `json:"innodbRowLockWaits"` // Innodb_row_lock_waits (cumulative)
	InnodbBufPoolReads int64   `json:"innodbBufPoolReads"` // Innodb_buffer_pool_reads (physical reads)

	// MongoDB-specific
	MongoConnections int64   `json:"mongoConnections"` // connections.current
	MongoPageFaults  int64   `json:"mongoPageFaults"`  // extra_info.page_faults
	MongoMemResident int64   `json:"mongoMemResident"` // mem.resident (MB)
	MongoGlobalLock  float64 `json:"mongoGlobalLock"`  // globalLock.currentQueue.total
}

// MonitoringSnapshot is the rolling window returned to the frontend
type MonitoringSnapshot struct {
	ProfileID  string      `json:"profileId"`
	DBType     string      `json:"dbType"` // "mysql" | "mongodb"
	DataPoints []DataPoint `json:"dataPoints"`
}
