package importexport

import (
	"context"
	"database/sql"
	"mygui/backend/internal/repository"
)

// Service 提供数据导入导出服务
type Service struct {
	exporter *Exporter
	importer *Importer
}

// NewService 创建新的 ImportExportService 实例
func NewService(db *sql.DB) *Service {
	return &Service{
		exporter: NewExporter(db),
		importer: NewImporter(db),
	}
}

// ExportToSQL 导出数据为 SQL INSERT 语句
func (s *Service) ExportToSQL(database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return s.exporter.ExportToSQL(database, table, query, outputPath, progressCallback)
}

// ExportToSQLContext 导出数据为 SQL INSERT 语句，支持取消
func (s *Service) ExportToSQLContext(ctx context.Context, database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return s.exporter.ExportToSQLContext(ctx, database, table, query, outputPath, progressCallback)
}

// ExportToCSV 导出数据为 CSV 格式
func (s *Service) ExportToCSV(database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return s.exporter.ExportToCSV(database, table, query, outputPath, progressCallback)
}

// ExportToCSVContext 导出数据为 CSV 格式，支持取消
func (s *Service) ExportToCSVContext(ctx context.Context, database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return s.exporter.ExportToCSVContext(ctx, database, table, query, outputPath, progressCallback)
}

// ExportToJSON 导出数据为 JSON 格式
func (s *Service) ExportToJSON(database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return s.exporter.ExportToJSON(database, table, query, outputPath, progressCallback)
}

// ExportToJSONContext 导出数据为 JSON 格式，支持取消
func (s *Service) ExportToJSONContext(ctx context.Context, database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return s.exporter.ExportToJSONContext(ctx, database, table, query, outputPath, progressCallback)
}

// ImportFromSQL 从 SQL 文件导入数据
func (s *Service) ImportFromSQL(database string, sqlFilePath string, progressCallback func(current, total int)) (*ImportResult, error) {
	return s.importer.ImportFromSQL(database, sqlFilePath, progressCallback)
}

// ImportFromSQLContext 从 SQL 文件导入数据，支持取消
func (s *Service) ImportFromSQLContext(ctx context.Context, database string, sqlFilePath string, progressCallback func(current, total int)) (*ImportResult, error) {
	return s.importer.ImportFromSQLContext(ctx, database, sqlFilePath, progressCallback)
}

// ImportFromCSV 从 CSV 文件导入数据
func (s *Service) ImportFromCSV(database, table string, csvFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	return s.importer.ImportFromCSV(database, table, csvFilePath, mapping, progressCallback)
}

// ImportFromCSVContext 从 CSV 文件导入数据，支持取消
func (s *Service) ImportFromCSVContext(ctx context.Context, database, table string, csvFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	return s.importer.ImportFromCSVContext(ctx, database, table, csvFilePath, mapping, progressCallback)
}

// ImportFromJSON 从 JSON 文件导入数据
func (s *Service) ImportFromJSON(database, table string, jsonFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	return s.importer.ImportFromJSON(database, table, jsonFilePath, mapping, progressCallback)
}

// ImportFromJSONContext 从 JSON 文件导入数据，支持取消
func (s *Service) ImportFromJSONContext(ctx context.Context, database, table string, jsonFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	return s.importer.ImportFromJSONContext(ctx, database, table, jsonFilePath, mapping, progressCallback)
}

// ValidateCSVFormat 验证 CSV 文件格式
func (s *Service) ValidateCSVFormat(csvFilePath string) error {
	return s.importer.ValidateCSVFormat(csvFilePath)
}

// ValidateJSONFormat 验证 JSON 文件格式
func (s *Service) ValidateJSONFormat(jsonFilePath string) error {
	return s.importer.ValidateJSONFormat(jsonFilePath)
}

// ValidateSQLFormat 验证 SQL 文件格式
func (s *Service) ValidateSQLFormat(sqlFilePath string) error {
	return s.importer.ValidateSQLFormat(sqlFilePath)
}
