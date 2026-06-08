package importexport

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mygui/backend/internal/repository"
	"os"
	"strings"
)

// Exporter 处理数据导出操作
type Exporter struct {
	db *sql.DB
}

const exportBatchSize = 1000

// NewExporter 创建新的 Exporter 实例
func NewExporter(db *sql.DB) *Exporter {
	return &Exporter{db: db}
}

// ExportToSQL 导出数据为 SQL INSERT 语句
func (e *Exporter) ExportToSQL(database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return e.ExportToSQLContext(context.Background(), database, table, query, outputPath, progressCallback)
}

// ExportToSQLContext 导出数据为 SQL INSERT 语句，支持取消
func (e *Exporter) ExportToSQLContext(ctx context.Context, database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	if database == "" {
		return fmt.Errorf("数据库名称不能为空")
	}
	if table == "" {
		return fmt.Errorf("表名称不能为空")
	}
	if outputPath == "" {
		return fmt.Errorf("输出路径不能为空")
	}

	// 创建输出文件
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 设置查询参数以获取所有数据
	query.Database = database
	query.Table = table

	// 获取数据仓库
	repo := repository.NewDataRepository(e.db)

	// 获取总行数
	totalRows, err := repo.GetRowCountContext(ctx, database, table, query.Filters)
	if err != nil {
		return fmt.Errorf("获取总行数失败: %w", err)
	}

	if totalRows == 0 {
		return fmt.Errorf("没有数据可导出")
	}
	exportTotal := exportRowTotal(totalRows, query.Limit, query.Offset)
	if exportTotal == 0 {
		return fmt.Errorf("没有数据可导出")
	}

	// 写入文件头注释
	_, err = file.WriteString(fmt.Sprintf("-- MySQL dump for table %s.%s\n", database, table))
	if err != nil {
		return fmt.Errorf("写入文件头失败: %w", err)
	}
	_, err = file.WriteString(fmt.Sprintf("-- Total rows: %d\n\n", exportTotal))
	if err != nil {
		return fmt.Errorf("写入文件头失败: %w", err)
	}

	processedRows := 0
	err = e.forEachExportBatch(ctx, repo, database, table, query, exportTotal, func(result *repository.DataResult) error {
		// 生成 INSERT 语句
		for _, row := range result.Rows {
			if err := ctx.Err(); err != nil {
				return err
			}
			insertSQL := e.generateInsertStatement(database, table, result.Columns, row)
			_, err = file.WriteString(insertSQL + "\n")
			if err != nil {
				return fmt.Errorf("写入 INSERT 语句失败: %w", err)
			}

			processedRows++
			if progressCallback != nil {
				progressCallback(processedRows, exportTotal)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// ExportToCSV 导出数据为 CSV 格式
func (e *Exporter) ExportToCSV(database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return e.ExportToCSVContext(context.Background(), database, table, query, outputPath, progressCallback)
}

// ExportToCSVContext 导出数据为 CSV 格式，支持取消
func (e *Exporter) ExportToCSVContext(ctx context.Context, database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	if database == "" {
		return fmt.Errorf("数据库名称不能为空")
	}
	if table == "" {
		return fmt.Errorf("表名称不能为空")
	}
	if outputPath == "" {
		return fmt.Errorf("输出路径不能为空")
	}

	// 创建输出文件
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 创建 CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 设置查询参数
	query.Database = database
	query.Table = table

	// 获取数据仓库
	repo := repository.NewDataRepository(e.db)

	// 获取总行数
	totalRows, err := repo.GetRowCountContext(ctx, database, table, query.Filters)
	if err != nil {
		return fmt.Errorf("获取总行数失败: %w", err)
	}

	if totalRows == 0 {
		return fmt.Errorf("没有数据可导出")
	}
	exportTotal := exportRowTotal(totalRows, query.Limit, query.Offset)
	if exportTotal == 0 {
		return fmt.Errorf("没有数据可导出")
	}

	// 写入列标题
	firstBatch := true
	processedRows := 0

	err = e.forEachExportBatch(ctx, repo, database, table, query, exportTotal, func(result *repository.DataResult) error {
		// 写入列标题（仅第一批）
		if firstBatch {
			if err := writer.Write(result.Columns); err != nil {
				return fmt.Errorf("写入列标题失败: %w", err)
			}
			firstBatch = false
		}

		// 写入数据行
		for _, row := range result.Rows {
			if err := ctx.Err(); err != nil {
				return err
			}
			record := make([]string, len(row))
			for i, val := range row {
				record[i] = e.formatValue(val)
			}
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("写入数据行失败: %w", err)
			}

			processedRows++
			if progressCallback != nil {
				progressCallback(processedRows, exportTotal)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// ExportToJSON 导出数据为 JSON 格式
func (e *Exporter) ExportToJSON(database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	return e.ExportToJSONContext(context.Background(), database, table, query, outputPath, progressCallback)
}

// ExportToJSONContext 导出数据为 JSON 格式，支持取消
func (e *Exporter) ExportToJSONContext(ctx context.Context, database, table string, query repository.DataQuery, outputPath string, progressCallback func(current, total int)) error {
	if database == "" {
		return fmt.Errorf("数据库名称不能为空")
	}
	if table == "" {
		return fmt.Errorf("表名称不能为空")
	}
	if outputPath == "" {
		return fmt.Errorf("输出路径不能为空")
	}

	// 创建输出文件
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 设置查询参数
	query.Database = database
	query.Table = table

	// 获取数据仓库
	repo := repository.NewDataRepository(e.db)

	// 获取总行数
	totalRows, err := repo.GetRowCountContext(ctx, database, table, query.Filters)
	if err != nil {
		return fmt.Errorf("获取总行数失败: %w", err)
	}

	if totalRows == 0 {
		return fmt.Errorf("没有数据可导出")
	}
	exportTotal := exportRowTotal(totalRows, query.Limit, query.Offset)
	if exportTotal == 0 {
		return fmt.Errorf("没有数据可导出")
	}

	// 写入 JSON 数组开始
	_, err = file.WriteString("[\n")
	if err != nil {
		return fmt.Errorf("写入 JSON 开始标记失败: %w", err)
	}

	processedRows := 0
	firstRow := true

	err = e.forEachExportBatch(ctx, repo, database, table, query, exportTotal, func(result *repository.DataResult) error {
		// 转换为 JSON 对象
		for _, row := range result.Rows {
			if err := ctx.Err(); err != nil {
				return err
			}
			// 构建 JSON 对象
			obj := make(map[string]interface{})
			for i, col := range result.Columns {
				obj[col] = row[i]
			}

			// 序列化为 JSON
			jsonData, err := json.Marshal(obj)
			if err != nil {
				return fmt.Errorf("序列化 JSON 失败: %w", err)
			}

			// 写入 JSON（添加逗号分隔符）
			if !firstRow {
				_, err = file.WriteString(",\n")
				if err != nil {
					return fmt.Errorf("写入 JSON 分隔符失败: %w", err)
				}
			}
			_, err = file.WriteString("  " + string(jsonData))
			if err != nil {
				return fmt.Errorf("写入 JSON 数据失败: %w", err)
			}

			firstRow = false
			processedRows++
			if progressCallback != nil {
				progressCallback(processedRows, exportTotal)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 写入 JSON 数组结束
	_, err = file.WriteString("\n]\n")
	if err != nil {
		return fmt.Errorf("写入 JSON 结束标记失败: %w", err)
	}

	return nil
}

func (e *Exporter) forEachExportBatch(ctx context.Context, repo *repository.DataRepository, database, table string, query repository.DataQuery, exportTotal int, handleBatch func(*repository.DataResult) error) error {
	query.Database = database
	query.Table = table

	if err := ctx.Err(); err != nil {
		return err
	}
	if primaryKey, ok, err := e.singlePrimaryKeyColumn(database, table); err != nil {
		return fmt.Errorf("获取主键列失败: %w", err)
	} else if ok && canUseCursorExport(query, primaryKey) {
		return e.forEachCursorBatch(ctx, repo, query, primaryKey, exportTotal, handleBatch)
	}

	return e.forEachOffsetBatch(ctx, repo, query, exportTotal, handleBatch)
}

func (e *Exporter) forEachCursorBatch(ctx context.Context, repo *repository.DataRepository, query repository.DataQuery, primaryKey string, exportTotal int, handleBatch func(*repository.DataResult) error) error {
	processedRows := 0
	var lastCursor interface{}
	hasCursor := false

	for processedRows < exportTotal {
		if err := ctx.Err(); err != nil {
			return err
		}
		batchLimit := minInt(exportBatchSize, exportTotal-processedRows)
		batchQuery := buildCursorQuery(query, primaryKey, lastCursor, hasCursor, batchLimit)

		result, err := repo.QueryDataContext(ctx, batchQuery)
		if err != nil {
			return fmt.Errorf("查询数据失败: %w", err)
		}
		if len(result.Rows) == 0 {
			break
		}

		if err := handleBatch(result); err != nil {
			return err
		}

		cursorIndex := columnIndex(result.Columns, primaryKey)
		if cursorIndex < 0 {
			return fmt.Errorf("游标列 %s 不在查询结果中", primaryKey)
		}
		lastRow := result.Rows[len(result.Rows)-1]
		if cursorIndex >= len(lastRow) {
			return fmt.Errorf("游标列 %s 数据缺失", primaryKey)
		}
		lastCursor = lastRow[cursorIndex]
		hasCursor = true
		processedRows += len(result.Rows)
	}

	return nil
}

func (e *Exporter) forEachOffsetBatch(ctx context.Context, repo *repository.DataRepository, query repository.DataQuery, exportTotal int, handleBatch func(*repository.DataResult) error) error {
	processedRows := 0
	baseOffset := query.Offset
	for offset := 0; processedRows < exportTotal; offset += exportBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		batchQuery := query
		batchQuery.Limit = minInt(exportBatchSize, exportTotal-processedRows)
		batchQuery.Offset = baseOffset + offset

		result, err := repo.QueryDataContext(ctx, batchQuery)
		if err != nil {
			return fmt.Errorf("查询数据失败: %w", err)
		}
		if len(result.Rows) == 0 {
			break
		}

		if err := handleBatch(result); err != nil {
			return err
		}
		processedRows += len(result.Rows)
	}

	return nil
}

func (e *Exporter) singlePrimaryKeyColumn(database, table string) (string, bool, error) {
	rows, err := e.db.Query(`
		SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND CONSTRAINT_NAME = 'PRIMARY'
		ORDER BY ORDINAL_POSITION
	`, database, table)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return "", false, err
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return "", false, err
	}
	if len(columns) != 1 {
		return "", false, nil
	}
	return columns[0], true, nil
}

func canUseCursorExport(query repository.DataQuery, primaryKey string) bool {
	if query.Offset > 0 {
		return false
	}
	if len(query.Columns) > 0 && !containsColumn(query.Columns, primaryKey) {
		return false
	}
	if len(query.OrderBy) == 0 {
		return true
	}
	if len(query.OrderBy) != 1 {
		return false
	}
	order := query.OrderBy[0]
	return strings.EqualFold(order.Column, primaryKey) && strings.ToUpper(order.Direction) != "DESC"
}

func containsColumn(columns []string, target string) bool {
	for _, column := range columns {
		if strings.EqualFold(column, target) {
			return true
		}
	}
	return false
}

func buildCursorQuery(base repository.DataQuery, primaryKey string, cursor interface{}, hasCursor bool, limit int) repository.DataQuery {
	query := base
	query.Offset = 0
	query.Limit = limit
	query.OrderBy = []repository.OrderBy{{Column: primaryKey, Direction: "ASC"}}
	query.Filters = append([]repository.Filter{}, base.Filters...)
	if hasCursor {
		query.Filters = append(query.Filters, repository.Filter{
			Column:   primaryKey,
			Operator: ">",
			Value:    cursor,
		})
	}
	return query
}

func columnIndex(columns []string, target string) int {
	for i, column := range columns {
		if strings.EqualFold(column, target) {
			return i
		}
	}
	return -1
}

func exportRowTotal(totalRows int64, requestedLimit, offset int) int {
	total := int(totalRows)
	if offset > 0 {
		total -= offset
	}
	if total <= 0 {
		return 0
	}
	if requestedLimit > 0 && requestedLimit < total {
		return requestedLimit
	}
	return total
}

// generateInsertStatement 生成 INSERT 语句
func (e *Exporter) generateInsertStatement(database, table string, columns []string, values []interface{}) string {
	// 转义列名
	escapedColumns := make([]string, len(columns))
	for i, col := range columns {
		escapedColumns[i] = escapeIdentifier(col)
	}

	// 格式化值
	formattedValues := make([]string, len(values))
	for i, val := range values {
		formattedValues[i] = e.formatSQLValue(val)
	}

	return fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s);",
		escapeIdentifier(database),
		escapeIdentifier(table),
		strings.Join(escapedColumns, ", "),
		strings.Join(formattedValues, ", "))
}

// formatSQLValue 格式化 SQL 值
func (e *Exporter) formatSQLValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		// 转义单引号
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case []byte:
		// 字节数组转为字符串
		escaped := strings.ReplaceAll(string(v), "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		// 其他类型转为字符串
		return fmt.Sprintf("'%v'", v)
	}
}

// formatValue 格式化值为字符串（用于 CSV）
func (e *Exporter) formatValue(val interface{}) string {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// escapeIdentifier 转义数据库标识符
func escapeIdentifier(identifier string) string {
	identifier = strings.ReplaceAll(identifier, "`", "")
	return "`" + identifier + "`"
}
