package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"mygui/backend/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxDataPoints = 60

type session struct {
	profileID  string
	dbType     string
	dataPoints []types.DataPoint
	cancel     context.CancelFunc
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*session)}
}

func (m *Manager) StartMySQL(profileID string, db *sql.DB, intervalSec int) {
	if intervalSec <= 0 {
		intervalSec = 2
	}
	m.mu.Lock()
	if existing, ok := m.sessions[profileID]; ok {
		existing.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &session{profileID: profileID, dbType: "mysql", dataPoints: make([]types.DataPoint, 0, maxDataPoints), cancel: cancel}
	m.sessions[profileID] = s
	m.mu.Unlock()
	go m.pollMySQL(ctx, s, db, intervalSec)
}

func (m *Manager) StartMongo(profileID string, client *mongo.Client, intervalSec int) {
	if intervalSec <= 0 {
		intervalSec = 2
	}
	m.mu.Lock()
	if existing, ok := m.sessions[profileID]; ok {
		existing.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &session{profileID: profileID, dbType: "mongodb", dataPoints: make([]types.DataPoint, 0, maxDataPoints), cancel: cancel}
	m.sessions[profileID] = s
	m.mu.Unlock()
	go m.pollMongo(ctx, s, client, intervalSec)
}

func (m *Manager) Stop(profileID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[profileID]; ok {
		s.cancel()
		delete(m.sessions, profileID)
	}
}

func (m *Manager) GetSnapshot(profileID string) (*types.MonitoringSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[profileID]
	if !ok {
		return nil, fmt.Errorf("profile not found: %s", profileID)
	}
	pts := make([]types.DataPoint, len(s.dataPoints))
	copy(pts, s.dataPoints)
	return &types.MonitoringSnapshot{ProfileID: s.profileID, DBType: s.dbType, DataPoints: pts}, nil
}

func appendDataPoint(s *session, dp types.DataPoint) {
	s.dataPoints = append(s.dataPoints, dp)
	if len(s.dataPoints) > maxDataPoints {
		s.dataPoints = s.dataPoints[len(s.dataPoints)-maxDataPoints:]
	}
}

// ── MySQL polling ─────────────────────────────────────────────────────────────

type mysqlStatus struct {
	questions        int64
	commit           int64
	rollback         int64
	threadsConnected int64
	threadsRunning   int64
	bufPoolReads     int64
	bufPoolReadReqs  int64
	rowLockWaits     int64
}

func (m *Manager) pollMySQL(ctx context.Context, s *session, db *sql.DB, intervalSec int) {
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	var prev mysqlStatus
	var prevTime time.Time
	hasPrev := false

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cur, err := queryMySQLStatus(ctx, db)
			if err != nil {
				continue
			}
			if hasPrev {
				elapsed := now.Sub(prevTime).Seconds()
				if elapsed > 0 {
					qps := maxF(float64(cur.questions-prev.questions)/elapsed, 0)
					tps := maxF(float64((cur.commit-prev.commit)+(cur.rollback-prev.rollback))/elapsed, 0)

					// Buffer pool hit rate: (read_requests - reads) / read_requests * 100
					readReqDelta := cur.bufPoolReadReqs - prev.bufPoolReadReqs
					readDelta := cur.bufPoolReads - prev.bufPoolReads
					var hitRate float64
					if readReqDelta > 0 {
						hitRate = float64(readReqDelta-readDelta) / float64(readReqDelta) * 100
						if hitRate < 0 {
							hitRate = 0
						}
					}

					dp := types.DataPoint{
						Timestamp:          now,
						QPS:                qps,
						TPS:                tps,
						ThreadsConnected:   cur.threadsConnected,
						ThreadsRunning:     cur.threadsRunning,
						InnodbBufHitRate:   hitRate,
						InnodbRowLockWaits: cur.rowLockWaits,
						InnodbBufPoolReads: cur.bufPoolReads,
					}
					m.mu.Lock()
					if _, ok := m.sessions[s.profileID]; ok {
						appendDataPoint(s, dp)
					}
					m.mu.Unlock()
				}
			}
			prev = cur
			prevTime = now
			hasPrev = true
		}
	}
}

func queryMySQLStatus(ctx context.Context, db *sql.DB) (mysqlStatus, error) {
	rows, err := db.QueryContext(ctx, `SHOW GLOBAL STATUS WHERE Variable_name IN (
		'Questions','Com_commit','Com_rollback',
		'Threads_connected','Threads_running',
		'Innodb_buffer_pool_reads','Innodb_buffer_pool_read_requests',
		'Innodb_row_lock_waits'
	)`)
	if err != nil {
		return mysqlStatus{}, err
	}
	defer rows.Close()

	var s mysqlStatus
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return mysqlStatus{}, err
		}
		switch name {
		case "Questions":
			s.questions = value
		case "Com_commit":
			s.commit = value
		case "Com_rollback":
			s.rollback = value
		case "Threads_connected":
			s.threadsConnected = value
		case "Threads_running":
			s.threadsRunning = value
		case "Innodb_buffer_pool_reads":
			s.bufPoolReads = value
		case "Innodb_buffer_pool_read_requests":
			s.bufPoolReadReqs = value
		case "Innodb_row_lock_waits":
			s.rowLockWaits = value
		}
	}
	return s, rows.Err()
}

// ── MongoDB polling ───────────────────────────────────────────────────────────

type mongoStatus struct {
	query      int64
	insert     int64
	update     int64
	delete_    int64
	connCur    int64
	pageFaults int64
	memResMB   int64
	globalLock float64
}

func (m *Manager) pollMongo(ctx context.Context, s *session, client *mongo.Client, intervalSec int) {
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	var prev mongoStatus
	var prevTime time.Time
	hasPrev := false

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cur, err := queryMongoStatus(ctx, client)
			if err != nil {
				continue
			}
			if hasPrev {
				elapsed := now.Sub(prevTime).Seconds()
				if elapsed > 0 {
					qps := maxF(float64(cur.query-prev.query)/elapsed, 0)
					tps := maxF(float64((cur.insert-prev.insert)+(cur.update-prev.update)+(cur.delete_-prev.delete_))/elapsed, 0)

					dp := types.DataPoint{
						Timestamp:        now,
						QPS:              qps,
						TPS:              tps,
						MongoConnections: cur.connCur,
						MongoPageFaults:  cur.pageFaults,
						MongoMemResident: cur.memResMB,
						MongoGlobalLock:  cur.globalLock,
					}
					m.mu.Lock()
					if _, ok := m.sessions[s.profileID]; ok {
						appendDataPoint(s, dp)
					}
					m.mu.Unlock()
				}
			}
			prev = cur
			prevTime = now
			hasPrev = true
		}
	}
}

func queryMongoStatus(ctx context.Context, client *mongo.Client) (mongoStatus, error) {
	result := client.Database("admin").RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}})
	if result.Err() != nil {
		return mongoStatus{}, result.Err()
	}
	var raw bson.M
	if err := result.Decode(&raw); err != nil {
		return mongoStatus{}, err
	}

	var s mongoStatus

	// opcounters
	if oc, ok := raw["opcounters"].(bson.M); ok {
		s.query = toInt64(oc["query"])
		s.insert = toInt64(oc["insert"])
		s.update = toInt64(oc["update"])
		s.delete_ = toInt64(oc["delete"])
	}

	// connections.current
	if conn, ok := raw["connections"].(bson.M); ok {
		s.connCur = toInt64(conn["current"])
	}

	// extra_info.page_faults
	if ei, ok := raw["extra_info"].(bson.M); ok {
		s.pageFaults = toInt64(ei["page_faults"])
	}

	// mem.resident (MB)
	if mem, ok := raw["mem"].(bson.M); ok {
		s.memResMB = toInt64(mem["resident"])
	}

	// globalLock.currentQueue.total
	if gl, ok := raw["globalLock"].(bson.M); ok {
		if cq, ok := gl["currentQueue"].(bson.M); ok {
			s.globalLock = float64(toInt64(cq["total"]))
		}
	}

	return s, nil
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int32:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	}
	return 0
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
