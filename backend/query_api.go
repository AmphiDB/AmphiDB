package backend

import (
	"fmt"
	"time"

	"mygui/backend/internal/query"
	"mygui/backend/internal/storage"
	"mygui/backend/types"
)

// QueryAPI 提供查询执行相关的 API 方法，供前端调用

// ExecuteQueryInDatabase 在指定数据库上下文中执行 SQL 查询
func (a *App) ExecuteQueryInDatabase(profileID, database, sql string) (*query.QueryResult, error) {
	executor, err := a.getOrCreateQueryExecutor(profileID)
	if err != nil {
		return &query.QueryResult{
			Error: &query.QueryError{Code: -1, Message: fmt.Sprintf("执行查询失败: %v", err), Position: -1},
		}, nil
	}

	a.logger.Info("Executing query in database", map[string]interface{}{
		"profile_id": profileID,
		"database":   database,
		"sql_length": len(sql),
	})

	result, err := executor.ExecuteInDatabase(database, sql)
	if err != nil {
		if result == nil {
			result = &query.QueryResult{}
		}
		if result.Error == nil {
			result.Error = &query.QueryError{Code: -1, Message: fmt.Sprintf("执行查询失败: %v", err), Position: -1}
		}
		// 记录失败的 SQL 操作
		queryType := executor.DetermineQueryType(sql)
		a.logger.LogSQLOperation(string(queryType), sql, database, profileID, 0, false, err.Error())
		return result, nil
	}

	a.saveQueryHistory(profileID, database, sql, result)
	a.checkAndLogSlowQuery(profileID, database, sql, result)

	// 记录成功的 SQL 操作
	queryType := executor.DetermineQueryType(sql)
	a.logger.LogSQLOperation(string(queryType), sql, database, profileID, result.RowsAffected, true, "")

	return result, nil
}

// ExecuteQuery 执行 SQL 查询（使用默认超时）
func (a *App) ExecuteQuery(profileID, sql string) (*query.QueryResult, error) {
	executor, err := a.getOrCreateQueryExecutor(profileID)
	if err != nil {
		a.logger.Error("Failed to get query executor", err, map[string]interface{}{
			"profile_id": profileID,
		})
		// 返回一个包含错误信息的 QueryResult
		return &query.QueryResult{
			ID:            "",
			Type:          query.QueryTypeOther,
			RowsAffected:  0,
			ExecutionTime: 0,
			Error: &query.QueryError{
				Code:     -1,
				Message:  fmt.Sprintf("执行查询失败: %v", err),
				Position: -1,
			},
		}, nil
	}

	a.logger.Info("Executing query", map[string]interface{}{
		"profile_id": profileID,
		"sql_length": len(sql),
	})

	result, err := executor.Execute(sql)
	if err != nil {
		a.logger.Error("Query execution failed", err, map[string]interface{}{
			"profile_id": profileID,
			"sql":        sql,
		})
		// 如果 result 为 nil，创建一个包含错误的结果
		if result == nil {
			result = &query.QueryResult{
				ID:            "",
				Type:          query.QueryTypeOther,
				RowsAffected:  0,
				ExecutionTime: 0,
			}
		}
		// 确保 Error 字段被设置
		if result.Error == nil {
			result.Error = &query.QueryError{
				Code:     -1,
				Message:  fmt.Sprintf("执行查询失败: %v", err),
				Position: -1,
			}
		}
		// 记录失败的 SQL 操作
		queryType := executor.DetermineQueryType(sql)
		a.logger.LogSQLOperation(string(queryType), sql, "", profileID, 0, false, err.Error())
		return result, nil
	}

	a.logger.Info("Query executed successfully", map[string]interface{}{
		"profile_id":     profileID,
		"query_type":     result.Type,
		"rows_affected":  result.RowsAffected,
		"execution_time": result.ExecutionTime.String(),
	})

	// 保存查询历史
	a.saveQueryHistory(profileID, "", sql, result)
	a.checkAndLogSlowQuery(profileID, "", sql, result)

	// 记录成功的 SQL 操作
	queryType := executor.DetermineQueryType(sql)
	a.logger.LogSQLOperation(string(queryType), sql, "", profileID, result.RowsAffected, true, "")

	return result, nil
}

// ExecuteQueryWithTimeout 执行 SQL 查询（指定超时时间）
func (a *App) ExecuteQueryWithTimeout(profileID, sql string, timeoutSeconds int) (*query.QueryResult, error) {
	executor, err := a.getOrCreateQueryExecutor(profileID)
	if err != nil {
		a.logger.Error("Failed to get query executor", err, map[string]interface{}{
			"profile_id": profileID,
		})
		// 返回一个包含错误信息的 QueryResult
		return &query.QueryResult{
			ID:            "",
			Type:          query.QueryTypeOther,
			RowsAffected:  0,
			ExecutionTime: 0,
			Error: &query.QueryError{
				Code:     -1,
				Message:  fmt.Sprintf("执行查询失败: %v", err),
				Position: -1,
			},
		}, nil
	}

	timeout := time.Duration(timeoutSeconds) * time.Second

	a.logger.Info("Executing query with timeout", map[string]interface{}{
		"profile_id": profileID,
		"sql_length": len(sql),
		"timeout":    timeout.String(),
	})

	result, err := executor.ExecuteWithTimeout(sql, timeout)
	if err != nil {
		a.logger.Error("Query execution failed", err, map[string]interface{}{
			"profile_id": profileID,
			"sql":        sql,
		})
		// 如果 result 为 nil，创建一个包含错误的结果
		if result == nil {
			result = &query.QueryResult{
				ID:            "",
				Type:          query.QueryTypeOther,
				RowsAffected:  0,
				ExecutionTime: 0,
			}
		}
		// 确保 Error 字段被设置
		if result.Error == nil {
			result.Error = &query.QueryError{
				Code:     -1,
				Message:  fmt.Sprintf("执行查询失败: %v", err),
				Position: -1,
			}
		}
		// 记录失败的 SQL 操作
		queryType := executor.DetermineQueryType(sql)
		a.logger.LogSQLOperation(string(queryType), sql, "", profileID, 0, false, err.Error())
		return result, nil
	}

	a.logger.Info("Query executed successfully", map[string]interface{}{
		"profile_id":     profileID,
		"query_type":     result.Type,
		"rows_affected":  result.RowsAffected,
		"execution_time": result.ExecutionTime.String(),
	})

	// 保存查询历史
	a.saveQueryHistory(profileID, "", sql, result)
	a.checkAndLogSlowQuery(profileID, "", sql, result)

	// 记录成功的 SQL 操作
	queryType := executor.DetermineQueryType(sql)
	a.logger.LogSQLOperation(string(queryType), sql, "", profileID, result.RowsAffected, true, "")

	return result, nil
}

// CancelQuery 取消正在执行的查询
func (a *App) CancelQuery(profileID, queryID string) error {
	executor, err := a.getOrCreateQueryExecutor(profileID)
	if err != nil {
		a.logger.Error("Failed to get query executor", err, map[string]interface{}{
			"profile_id": profileID,
			"query_id":   queryID,
		})
		return fmt.Errorf("取消查询失败: %w", err)
	}

	a.logger.Info("Cancelling query", map[string]interface{}{
		"profile_id": profileID,
		"query_id":   queryID,
	})

	err = executor.Cancel(queryID)
	if err != nil {
		a.logger.Error("Failed to cancel query", err, map[string]interface{}{
			"profile_id": profileID,
			"query_id":   queryID,
		})
		return fmt.Errorf("取消查询失败: %w", err)
	}

	a.logger.Info("Query cancelled", map[string]interface{}{
		"profile_id": profileID,
		"query_id":   queryID,
	})

	return nil
}

// GetQueryHistory 获取查询历史记录
func (a *App) GetQueryHistory(profileID string, limit int) ([]storage.QueryHistoryEntry, error) {
	if limit <= 0 {
		limit = 50 // 默认返回最近 50 条
	}

	// 从配置存储中获取查询历史
	history, err := a.configStorage.GetQueryHistory(profileID, limit)
	if err != nil {
		a.logger.Error("Failed to get query history", err, map[string]interface{}{
			"profile_id": profileID,
		})
		return nil, fmt.Errorf("获取查询历史失败: %w", err)
	}

	return history, nil
}

// ClearQueryHistory 清空查询历史记录
func (a *App) ClearQueryHistory(profileID string) error {
	err := a.configStorage.ClearQueryHistory(profileID)
	if err != nil {
		a.logger.Error("Failed to clear query history", err, map[string]interface{}{
			"profile_id": profileID,
		})
		return fmt.Errorf("清空查询历史失败: %w", err)
	}

	a.logger.Info("Query history cleared", map[string]interface{}{
		"profile_id": profileID,
	})

	return nil
}

// checkAndLogSlowQuery checks if the query exceeded the slow query threshold and logs it if so.
// This is non-blocking — errors are logged but not surfaced to the caller.
func (a *App) checkAndLogSlowQuery(profileID, database, sqlText string, result *query.QueryResult) {
	threshold, err := a.configStorage.GetSlowQueryThreshold(profileID)
	if err != nil {
		a.logger.Error("Failed to get slow query threshold", err, map[string]interface{}{
			"profile_id": profileID,
		})
		return
	}

	durationMs := result.ExecutionTime.Milliseconds()
	if durationMs <= int64(threshold) {
		return
	}

	errorMessage := ""
	if result.Error != nil {
		errorMessage = result.Error.Message
	}

	entry := types.SlowQueryEntry{
		ConnectionID: profileID,
		DBType:       "mysql",
		Database:     database,
		QueryText:    sqlText,
		DurationMs:   durationMs,
		RowsAffected: result.RowsAffected,
		ErrorMessage: errorMessage,
	}

	if err := a.configStorage.InsertSlowQuery(entry); err != nil {
		a.logger.Error("Failed to insert slow query entry", err, map[string]interface{}{
			"profile_id": profileID,
		})
	}
}

// saveQueryHistory 保存查询历史到数据库
func (a *App) saveQueryHistory(profileID, database, sql string, result *query.QueryResult) {
	entry := storage.QueryHistoryEntry{
		Timestamp:     time.Now(),
		ConnectionID:  profileID,
		Database:      database,
		SQL:           sql,
		ExecutionTime: result.ExecutionTime.Milliseconds(),
		RowsAffected:  result.RowsAffected,
		Success:       result.Error == nil,
	}

	err := a.configStorage.SaveQueryHistory(entry)
	if err != nil {
		a.logger.Error("Failed to save query history", err, map[string]interface{}{
			"profile_id": profileID,
		})
	}
}

// SaveQuery 保存 SQL 查询
func (a *App) SaveQuery(profileID, name, sql, description, database string) (int64, error) {
	query := storage.SavedQuery{
		Name:         name,
		SQL:          sql,
		Description:  description,
		ConnectionID: profileID,
		Database:     database,
	}

	id, err := a.configStorage.SaveQuery(query)
	if err != nil {
		a.logger.Error("Failed to save query", err, map[string]interface{}{
			"profile_id": profileID,
			"name":       name,
		})
		return 0, fmt.Errorf("保存查询失败: %w", err)
	}

	a.logger.Info("Query saved", map[string]interface{}{
		"profile_id": profileID,
		"name":       name,
		"id":         id,
	})

	return id, nil
}

// GetSavedQueries 获取已保存的查询列表
func (a *App) GetSavedQueries(profileID string) ([]storage.SavedQuery, error) {
	queries, err := a.configStorage.GetSavedQueries(profileID)
	if err != nil {
		a.logger.Error("Failed to get saved queries", err, map[string]interface{}{
			"profile_id": profileID,
		})
		return nil, fmt.Errorf("获取已保存查询失败: %w", err)
	}

	return queries, nil
}

// GetSavedQuery 获取单个已保存的查询
func (a *App) GetSavedQuery(id int64) (*storage.SavedQuery, error) {
	query, err := a.configStorage.GetSavedQuery(id)
	if err != nil {
		a.logger.Error("Failed to get saved query", err, map[string]interface{}{
			"id": id,
		})
		return nil, fmt.Errorf("获取已保存查询失败: %w", err)
	}

	return query, nil
}

// UpdateSavedQuery 更新已保存的查询
func (a *App) UpdateSavedQuery(id int64, name, sql, description, database string) error {
	query := storage.SavedQuery{
		ID:          id,
		Name:        name,
		SQL:         sql,
		Description: description,
		Database:    database,
	}

	err := a.configStorage.UpdateSavedQuery(query)
	if err != nil {
		a.logger.Error("Failed to update saved query", err, map[string]interface{}{
			"id": id,
		})
		return fmt.Errorf("更新已保存查询失败: %w", err)
	}

	a.logger.Info("Saved query updated", map[string]interface{}{
		"id": id,
	})

	return nil
}

// DeleteSavedQuery 删除已保存的查询
func (a *App) DeleteSavedQuery(id int64) error {
	err := a.configStorage.DeleteSavedQuery(id)
	if err != nil {
		a.logger.Error("Failed to delete saved query", err, map[string]interface{}{
			"id": id,
		})
		return fmt.Errorf("删除已保存查询失败: %w", err)
	}

	a.logger.Info("Saved query deleted", map[string]interface{}{
		"id": id,
	})

	return nil
}
