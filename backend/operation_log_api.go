package backend

import (
	"fmt"

	"mygui/backend/types"
)

// OperationLogAPI 提供操作日志相关的 API 方法，供前端调用

// GetOperationLogs 获取操作日志列表
func (a *App) GetOperationLogs(filter types.LogFilter) ([]types.LogEntry, error) {
	if a.logger == nil {
		return nil, fmt.Errorf("日志服务未初始化")
	}

	logs, err := a.logger.GetLogs(filter)
	if err != nil {
		a.logger.Error("Failed to get operation logs", err, nil)
		return nil, fmt.Errorf("获取操作日志失败: %w", err)
	}

	return logs, nil
}

// ClearOperationLogs 清空所有操作日志
func (a *App) ClearOperationLogs() error {
	if a.logger == nil {
		return fmt.Errorf("日志服务未初始化")
	}

	err := a.logger.ClearLogs()
	if err != nil {
		a.logger.Error("Failed to clear operation logs", err, nil)
		return fmt.Errorf("清空操作日志失败: %w", err)
	}

	a.logger.Info("Operation logs cleared", nil)
	return nil
}

// ClearOperationLogsByConnection 清空指定连接的操作日志
func (a *App) ClearOperationLogsByConnection(connectionID string) error {
	if a.logger == nil {
		return fmt.Errorf("日志服务未初始化")
	}

	err := a.logger.ClearLogsByConnection(connectionID)
	if err != nil {
		a.logger.Error("Failed to clear operation logs by connection", err, map[string]interface{}{
			"connection_id": connectionID,
		})
		return fmt.Errorf("清空连接操作日志失败: %w", err)
	}

	a.logger.Info("Operation logs cleared for connection", map[string]interface{}{
		"connection_id": connectionID,
	})
	return nil
}

// GetOperationLogCount 获取操作日志总数
func (a *App) GetOperationLogCount() (int64, error) {
	if a.logger == nil {
		return 0, fmt.Errorf("日志服务未初始化")
	}

	count, err := a.logger.GetLogCount()
	if err != nil {
		a.logger.Error("Failed to get operation log count", err, nil)
		return 0, fmt.Errorf("获取操作日志数量失败: %w", err)
	}

	return count, nil
}
