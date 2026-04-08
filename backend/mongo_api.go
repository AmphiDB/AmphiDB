package backend

import (
	"context"
	"fmt"

	"mygui/backend/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ===== 连接管理 =====

// MongoCreateProfile 创建 MongoDB 连接配置
func (a *App) MongoCreateProfile(profile types.MongoConnectionProfile) error {
	a.logger.Info("MongoCreateProfile", map[string]any{"name": profile.Name})
	if err := a.mongoConnectionManager.CreateProfile(profile); err != nil {
		a.logger.Error("MongoCreateProfile failed", err, nil)
		return err
	}
	return nil
}

// MongoUpdateProfile 更新 MongoDB 连接配置
func (a *App) MongoUpdateProfile(id string, profile types.MongoConnectionProfile) error {
	a.logger.Info("MongoUpdateProfile", map[string]any{"id": id})
	if err := a.mongoConnectionManager.UpdateProfile(id, profile); err != nil {
		a.logger.Error("MongoUpdateProfile failed", err, nil)
		return err
	}
	return nil
}

// MongoDeleteProfile 删除 MongoDB 连接配置
func (a *App) MongoDeleteProfile(id string) error {
	a.logger.Info("MongoDeleteProfile", map[string]any{"id": id})
	if err := a.mongoConnectionManager.DeleteProfile(id); err != nil {
		a.logger.Error("MongoDeleteProfile failed", err, nil)
		return err
	}
	return nil
}

// MongoListProfiles 列出所有 MongoDB 连接配置
func (a *App) MongoListProfiles() ([]types.MongoConnectionProfile, error) {
	a.logger.Info("MongoListProfiles", nil)
	profiles, err := a.mongoConnectionManager.ListProfiles()
	if err != nil {
		a.logger.Error("MongoListProfiles failed", err, nil)
		return nil, err
	}
	return profiles, nil
}

// MongoTestConnection 测试 MongoDB 连接
func (a *App) MongoTestConnection(profile types.MongoConnectionProfile) (*types.MongoTestResult, error) {
	a.logger.Info("MongoTestConnection", map[string]any{"name": profile.Name})
	result, err := a.mongoConnectionManager.TestConnection(profile)
	if err != nil {
		a.logger.Error("MongoTestConnection failed", err, nil)
		return nil, err
	}
	return result, nil
}

// MongoConnect 建立 MongoDB 连接
func (a *App) MongoConnect(profileID string) error {
	a.logger.Info("MongoConnect", map[string]any{"profileID": profileID})
	if _, err := a.mongoConnectionManager.Connect(profileID); err != nil {
		a.logger.Error("MongoConnect failed", err, nil)
		return err
	}
	if client, clientErr := a.mongoConnectionManager.GetClient(profileID); clientErr == nil {
		a.monitorManager.StartMongo(profileID, client, 2)
	}
	// 清除该 profileID 的缓存管理器，确保使用新连接
	a.mu.Lock()
	delete(a.mongoCollManagers, profileID)
	delete(a.mongoDocManagers, profileID)
	delete(a.mongoQueryExecutors, profileID)
	delete(a.mongoIndexManagers, profileID)
	delete(a.mongoSchemaAnalyzers, profileID)
	a.mu.Unlock()
	return nil
}

// MongoDisconnect 断开 MongoDB 连接
func (a *App) MongoDisconnect(profileID string) error {
	a.logger.Info("MongoDisconnect", map[string]any{"profileID": profileID})
	if err := a.mongoConnectionManager.Disconnect(profileID); err != nil {
		a.logger.Error("MongoDisconnect failed", err, nil)
		return err
	}
	a.monitorManager.Stop(profileID)
	// 清除该 profileID 的缓存管理器
	a.mu.Lock()
	delete(a.mongoCollManagers, profileID)
	delete(a.mongoDocManagers, profileID)
	delete(a.mongoQueryExecutors, profileID)
	delete(a.mongoIndexManagers, profileID)
	delete(a.mongoSchemaAnalyzers, profileID)
	a.mu.Unlock()
	return nil
}

// MongoGetConnectionStatus 获取 MongoDB 连接状态，返回 "connected" 或 "disconnected"
func (a *App) MongoGetConnectionStatus(profileID string) string {
	a.logger.Info("MongoGetConnectionStatus", map[string]any{"profileID": profileID})
	if _, err := a.mongoConnectionManager.GetClient(profileID); err == nil {
		return "connected"
	}
	return "disconnected"
}

// ===== 数据库与集合 =====

// MongoListDatabases 列出所有数据库
func (a *App) MongoListDatabases(profileID string) ([]types.MongoDatabase, error) {
	a.logger.Info("MongoListDatabases", map[string]any{"profileID": profileID})
	manager, err := a.getOrCreateMongoCollManager(profileID)
	if err != nil {
		a.logger.Error("MongoListDatabases failed", err, nil)
		return nil, err
	}
	dbs, err := manager.ListDatabases(context.Background())
	if err != nil {
		a.logger.Error("MongoListDatabases failed", err, nil)
		return nil, err
	}
	return dbs, nil
}

// MongoListCollections 列出指定数据库中的所有集合
func (a *App) MongoListCollections(profileID string, dbName string) ([]types.MongoCollection, error) {
	a.logger.Info("MongoListCollections", map[string]any{"profileID": profileID, "db": dbName})
	manager, err := a.getOrCreateMongoCollManager(profileID)
	if err != nil {
		a.logger.Error("MongoListCollections failed", err, nil)
		return nil, err
	}
	colls, err := manager.ListCollections(context.Background(), dbName)
	if err != nil {
		a.logger.Error("MongoListCollections failed", err, nil)
		return nil, err
	}
	return colls, nil
}

// MongoCreateCollection 在指定数据库中创建集合
func (a *App) MongoCreateCollection(profileID string, dbName string, collName string) error {
	a.logger.Info("MongoCreateCollection", map[string]any{"profileID": profileID, "db": dbName, "coll": collName})
	manager, err := a.getOrCreateMongoCollManager(profileID)
	if err != nil {
		a.logger.Error("MongoCreateCollection failed", err, nil)
		return err
	}
	if err := manager.CreateCollection(context.Background(), dbName, collName); err != nil {
		a.logger.Error("MongoCreateCollection failed", err, nil)
		return err
	}
	return nil
}

// MongoDropCollection 删除指定集合
func (a *App) MongoDropCollection(profileID string, dbName string, collName string) error {
	a.logger.Info("MongoDropCollection", map[string]any{"profileID": profileID, "db": dbName, "coll": collName})
	manager, err := a.getOrCreateMongoCollManager(profileID)
	if err != nil {
		a.logger.Error("MongoDropCollection failed", err, nil)
		return err
	}
	if err := manager.DropCollection(context.Background(), dbName, collName); err != nil {
		a.logger.Error("MongoDropCollection failed", err, nil)
		return err
	}
	return nil
}

// ===== 文档管理 =====

// MongoQueryDocuments 查询文档
func (a *App) MongoQueryDocuments(profileID string, params types.MongoQueryParams) (*types.MongoDocumentResult, error) {
	a.logger.Info("MongoQueryDocuments", map[string]any{"profileID": profileID, "db": params.Database, "coll": params.Collection})
	manager, err := a.getOrCreateMongoDocManager(profileID)
	if err != nil {
		a.logger.Error("MongoQueryDocuments failed", err, nil)
		return nil, err
	}
	result, err := manager.QueryDocuments(context.Background(), params)
	if err != nil {
		a.logger.Error("MongoQueryDocuments failed", err, nil)
		return nil, err
	}
	return result, nil
}

// MongoInsertDocument 插入文档，返回插入文档的 _id
func (a *App) MongoInsertDocument(profileID string, dbName string, collName string, jsonDoc string) (string, error) {
	a.logger.Info("MongoInsertDocument", map[string]any{"profileID": profileID, "db": dbName, "coll": collName})
	manager, err := a.getOrCreateMongoDocManager(profileID)
	if err != nil {
		a.logger.Error("MongoInsertDocument failed", err, nil)
		return "", err
	}
	id, err := manager.InsertDocument(context.Background(), dbName, collName, jsonDoc)
	if err != nil {
		a.logger.Error("MongoInsertDocument failed", err, nil)
		return "", err
	}
	return id, nil
}

// MongoUpdateDocument 更新文档（以 _id 为条件替换）
func (a *App) MongoUpdateDocument(profileID string, dbName string, collName string, docID string, jsonDoc string) error {
	a.logger.Info("MongoUpdateDocument", map[string]any{"profileID": profileID, "db": dbName, "coll": collName, "docID": docID})
	manager, err := a.getOrCreateMongoDocManager(profileID)
	if err != nil {
		a.logger.Error("MongoUpdateDocument failed", err, nil)
		return err
	}
	if err := manager.UpdateDocument(context.Background(), dbName, collName, docID, jsonDoc); err != nil {
		a.logger.Error("MongoUpdateDocument failed", err, nil)
		return err
	}
	return nil
}

// MongoDeleteDocuments 删除文档，返回删除数量
func (a *App) MongoDeleteDocuments(profileID string, dbName string, collName string, docIDs []string) (int64, error) {
	a.logger.Info("MongoDeleteDocuments", map[string]any{"profileID": profileID, "db": dbName, "coll": collName, "count": len(docIDs)})
	manager, err := a.getOrCreateMongoDocManager(profileID)
	if err != nil {
		a.logger.Error("MongoDeleteDocuments failed", err, nil)
		return 0, err
	}
	deleted, err := manager.DeleteDocuments(context.Background(), dbName, collName, docIDs)
	if err != nil {
		a.logger.Error("MongoDeleteDocuments failed", err, nil)
		return 0, err
	}
	return deleted, nil
}

// ===== 索引管理 =====

// MongoListIndexes 列出集合的所有索引
func (a *App) MongoListIndexes(profileID string, dbName string, collName string) ([]types.MongoIndex, error) {
	a.logger.Info("MongoListIndexes", map[string]any{"profileID": profileID, "db": dbName, "coll": collName})
	manager, err := a.getOrCreateMongoIndexManager(profileID)
	if err != nil {
		a.logger.Error("MongoListIndexes failed", err, nil)
		return nil, err
	}
	indexes, err := manager.ListIndexes(context.Background(), dbName, collName)
	if err != nil {
		a.logger.Error("MongoListIndexes failed", err, nil)
		return nil, err
	}
	return indexes, nil
}

// MongoCreateIndex 创建索引，返回索引名称
func (a *App) MongoCreateIndex(profileID string, dbName string, collName string, spec types.MongoIndexSpec) (string, error) {
	a.logger.Info("MongoCreateIndex", map[string]any{"profileID": profileID, "db": dbName, "coll": collName})
	manager, err := a.getOrCreateMongoIndexManager(profileID)
	if err != nil {
		a.logger.Error("MongoCreateIndex failed", err, nil)
		return "", err
	}
	name, err := manager.CreateIndex(context.Background(), dbName, collName, spec)
	if err != nil {
		a.logger.Error("MongoCreateIndex failed", err, nil)
		return "", err
	}
	return name, nil
}

// MongoDropIndex 删除索引
func (a *App) MongoDropIndex(profileID string, dbName string, collName string, indexName string) error {
	a.logger.Info("MongoDropIndex", map[string]any{"profileID": profileID, "db": dbName, "coll": collName, "index": indexName})
	manager, err := a.getOrCreateMongoIndexManager(profileID)
	if err != nil {
		a.logger.Error("MongoDropIndex failed", err, nil)
		return err
	}
	if err := manager.DropIndex(context.Background(), dbName, collName, indexName); err != nil {
		a.logger.Error("MongoDropIndex failed", err, nil)
		return err
	}
	return nil
}

// ===== 聚合查询 =====

// MongoRunAggregation 执行聚合查询，并将结果保存到查询历史
func (a *App) MongoRunAggregation(profileID string, dbName string, collName string, pipelineJSON string) (*types.MongoAggregationResult, error) {
	a.logger.Info("MongoRunAggregation", map[string]any{"profileID": profileID, "db": dbName, "coll": collName})
	executor, err := a.getOrCreateMongoQueryExecutor(profileID)
	if err != nil {
		a.logger.Error("MongoRunAggregation failed", err, nil)
		return nil, err
	}
	result, err := executor.RunAggregation(context.Background(), dbName, collName, pipelineJSON)
	success := err == nil
	execTimeMs := int64(0)
	if result != nil {
		execTimeMs = result.ExecutionTime.Milliseconds()
	}
	// 保存查询历史（无论成功与否）
	if histErr := a.configStorage.SaveMongoQueryHistory(profileID, dbName, collName, pipelineJSON, execTimeMs, success); histErr != nil {
		a.logger.Error("SaveMongoQueryHistory failed", histErr, nil)
	}
	// 检查并记录慢查询
	a.checkAndLogMongoSlowQuery(profileID, dbName, collName, pipelineJSON, execTimeMs, err)
	if err != nil {
		a.logger.Error("MongoRunAggregation failed", err, nil)
		return nil, err
	}
	return result, nil
}

// checkAndLogMongoSlowQuery checks if the MongoDB aggregation exceeded the slow query threshold
// and logs it if so. Non-blocking — errors are logged but not surfaced to the caller.
func (a *App) checkAndLogMongoSlowQuery(profileID, dbName, collName, pipelineJSON string, durationMs int64, execErr error) {
	threshold, err := a.configStorage.GetSlowQueryThreshold(profileID)
	if err != nil {
		a.logger.Error("Failed to get slow query threshold", err, map[string]any{"profileID": profileID})
		return
	}
	if durationMs <= int64(threshold) {
		return
	}
	errorMessage := ""
	if execErr != nil {
		errorMessage = execErr.Error()
	}
	entry := types.SlowQueryEntry{
		ConnectionID: profileID,
		DBType:       "mongodb",
		Database:     dbName,
		Collection:   collName,
		QueryText:    pipelineJSON,
		DurationMs:   durationMs,
		ErrorMessage: errorMessage,
	}
	if err := a.configStorage.InsertSlowQuery(entry); err != nil {
		a.logger.Error("Failed to insert mongo slow query entry", err, map[string]any{"profileID": profileID})
	}
}

// ===== Schema 分析 =====

// MongoAnalyzeSchema 分析集合的字段结构
func (a *App) MongoAnalyzeSchema(profileID string, dbName string, collName string, sampleSize int) (*types.MongoSchemaAnalysis, error) {
	a.logger.Info("MongoAnalyzeSchema", map[string]any{"profileID": profileID, "db": dbName, "coll": collName, "sampleSize": sampleSize})
	analyzer, err := a.getOrCreateMongoSchemaAnalyzer(profileID)
	if err != nil {
		a.logger.Error("MongoAnalyzeSchema failed", err, nil)
		return nil, err
	}
	analysis, err := analyzer.AnalyzeSchema(context.Background(), dbName, collName, sampleSize)
	if err != nil {
		a.logger.Error("MongoAnalyzeSchema failed", err, nil)
		return nil, err
	}
	return analysis, nil
}

// ===== 数据导出 =====

// MongoExportDocuments 导出文档到文件
// 若 params.FilePath 为空，则通过系统文件对话框选择保存路径
func (a *App) MongoExportDocuments(profileID string, params types.MongoExportParams) error {
	a.logger.Info("MongoExportDocuments", map[string]any{"profileID": profileID, "db": params.Database, "coll": params.Collection, "format": params.Format})

	if params.FilePath == "" {
		ext := params.Format
		if ext == "" {
			ext = "json"
		}
		filters := []runtime.FileFilter{
			{DisplayName: fmt.Sprintf("%s files (*.%s)", ext, ext), Pattern: fmt.Sprintf("*.%s", ext)},
		}
		savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			DefaultFilename: fmt.Sprintf("%s.%s", params.Collection, ext),
			Filters:         filters,
		})
		if err != nil {
			a.logger.Error("MongoExportDocuments SaveFileDialog failed", err, nil)
			return err
		}
		if savePath == "" {
			// 用户取消了对话框
			return nil
		}
		params.FilePath = savePath
	}

	manager, err := a.getOrCreateMongoDocManager(profileID)
	if err != nil {
		a.logger.Error("MongoExportDocuments failed", err, nil)
		return err
	}
	if _, err := manager.ExportDocuments(context.Background(), params); err != nil {
		a.logger.Error("MongoExportDocuments failed", err, nil)
		return err
	}
	return nil
}
