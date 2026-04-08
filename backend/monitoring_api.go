package backend

import (
	"fmt"

	"mygui/backend/types"
)

const defaultMonitoringLimit = 200

// GetExecutionLog returns execution log entries for the given profile and database type.
// dbType can be "mysql", "mongodb", or "" / "all" for both.
// If limit <= 0, defaults to 200.
func (a *App) GetExecutionLog(profileID string, dbType string, limit int) ([]types.ExecutionLogEntry, error) {
	if limit <= 0 {
		limit = defaultMonitoringLimit
	}
	entries, err := a.configStorage.GetExecutionLog(profileID, dbType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution log for profile %q: %w", profileID, err)
	}
	return entries, nil
}

// ClearExecutionLog deletes execution log entries for the given profile and database type.
// dbType can be "mysql", "mongodb", or "" / "all" for both.
func (a *App) ClearExecutionLog(profileID string, dbType string) error {
	if err := a.configStorage.ClearExecutionLog(profileID, dbType); err != nil {
		return fmt.Errorf("failed to clear execution log for profile %q: %w", profileID, err)
	}
	return nil
}

// GetMonitoringSnapshot returns the current rolling window of TPS/QPS data points
// for the given profile. The dbType parameter is accepted for API consistency but
// the monitor manager resolves the type from the active session.
func (a *App) GetMonitoringSnapshot(profileID string, dbType string) (*types.MonitoringSnapshot, error) {
	snapshot, err := a.monitorManager.GetSnapshot(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring snapshot for profile %q: %w", profileID, err)
	}
	return snapshot, nil
}

// GetSlowQueryLog returns slow query log entries for the given profile.
// If limit <= 0, defaults to 200.
func (a *App) GetSlowQueryLog(profileID string, limit int) ([]types.SlowQueryEntry, error) {
	if limit <= 0 {
		limit = defaultMonitoringLimit
	}
	entries, err := a.configStorage.GetSlowQueryLog(profileID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get slow query log for profile %q: %w", profileID, err)
	}
	return entries, nil
}

// ClearSlowQueryLog deletes all slow query log entries for the given profile.
func (a *App) ClearSlowQueryLog(profileID string) error {
	if err := a.configStorage.ClearSlowQueryLog(profileID); err != nil {
		return fmt.Errorf("failed to clear slow query log for profile %q: %w", profileID, err)
	}
	return nil
}

// SetSlowQueryThreshold sets the slow query threshold in milliseconds for the given profile.
// Returns an error if thresholdMs <= 0.
func (a *App) SetSlowQueryThreshold(profileID string, thresholdMs int) error {
	if err := a.configStorage.SetSlowQueryThreshold(profileID, thresholdMs); err != nil {
		return fmt.Errorf("failed to set slow query threshold for profile %q: %w", profileID, err)
	}
	return nil
}

// GetSlowQueryThreshold returns the slow query threshold in milliseconds for the given profile.
// Returns 1000 ms if no threshold has been configured.
func (a *App) GetSlowQueryThreshold(profileID string) (int, error) {
	threshold, err := a.configStorage.GetSlowQueryThreshold(profileID)
	if err != nil {
		return 0, fmt.Errorf("failed to get slow query threshold for profile %q: %w", profileID, err)
	}
	return threshold, nil
}
