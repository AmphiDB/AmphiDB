package importexport

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Importer 处理数据导入操作
type Importer struct {
	db *sql.DB
}

// NewImporter 创建新的 Importer 实例
func NewImporter(db *sql.DB) *Importer {
	return &Importer{db: db}
}

// ImportResult 表示导入结果
type ImportResult struct {
	TotalRows   int
	SuccessRows int
	FailedRows  int
	Errors      []ImportError
}

// ImportError 表示导入错误
type ImportError struct {
	Row     int
	Message string
}

// ColumnMapping 表示列映射关系
type ColumnMapping struct {
	FileColumns  []string
	TableColumns []string
}

type importRow struct {
	Values []interface{}
}

const importBatchSize = 1000
const maxImportErrors = 200

func (r *ImportResult) addError(row int, message string) {
	r.FailedRows++
	if len(r.Errors) >= maxImportErrors {
		return
	}
	r.Errors = append(r.Errors, ImportError{
		Row:     row,
		Message: message,
	})
}

// ImportFromSQL 从 SQL 文件导入数据
func (i *Importer) ImportFromSQL(database string, sqlFilePath string, progressCallback func(current, total int)) (*ImportResult, error) {
	return i.ImportFromSQLContext(context.Background(), database, sqlFilePath, progressCallback)
}

// ImportFromSQLContext 从 SQL 文件导入数据，支持取消
func (i *Importer) ImportFromSQLContext(ctx context.Context, database string, sqlFilePath string, progressCallback func(current, total int)) (*ImportResult, error) {
	if database == "" {
		return nil, fmt.Errorf("数据库名称不能为空")
	}
	if sqlFilePath == "" {
		return nil, fmt.Errorf("SQL 文件路径不能为空")
	}

	// 打开 SQL 文件
	file, err := os.Open(sqlFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开 SQL 文件失败: %w", err)
	}
	defer file.Close()

	// 切换到目标数据库
	_, err = i.db.ExecContext(ctx, fmt.Sprintf("USE %s", escapeIdentifier(database)))
	if err != nil {
		return nil, fmt.Errorf("切换数据库失败: %w", err)
	}

	result := &ImportResult{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 增加缓冲区大小以处理大语句

	var currentStatement strings.Builder
	lineNumber := 0

	// 逐行读取并执行 SQL 语句
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "#") {
			continue
		}

		// 累积 SQL 语句（直到遇到分号）
		currentStatement.WriteString(line)
		currentStatement.WriteString(" ")

		// 检查是否是完整的语句（以分号结尾）
		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSpace(currentStatement.String())
			result.TotalRows++

			// 执行 SQL 语句
			_, err := i.db.ExecContext(ctx, stmt)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return result, err
				}
				result.addError(lineNumber, fmt.Sprintf("执行 SQL 失败: %v", err))
			} else {
				result.SuccessRows++
			}

			// 重置语句构建器
			currentStatement.Reset()

			// 报告进度
			if progressCallback != nil {
				progressCallback(result.TotalRows, result.TotalRows)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("读取 SQL 文件失败: %w", err)
	}

	return result, nil
}

// ImportFromCSV 从 CSV 文件导入数据
func (i *Importer) ImportFromCSV(database, table string, csvFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	return i.ImportFromCSVContext(context.Background(), database, table, csvFilePath, mapping, progressCallback)
}

// ImportFromCSVContext 从 CSV 文件导入数据，支持取消
func (i *Importer) ImportFromCSVContext(ctx context.Context, database, table string, csvFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	if database == "" {
		return nil, fmt.Errorf("数据库名称不能为空")
	}
	if table == "" {
		return nil, fmt.Errorf("表名称不能为空")
	}
	if csvFilePath == "" {
		return nil, fmt.Errorf("CSV 文件路径不能为空")
	}

	totalRows, err := countCSVDataRows(ctx, csvFilePath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开 CSV 文件失败: %w", err)
	}
	defer file.Close()

	// 创建 CSV reader
	reader := csv.NewReader(file)

	// 读取表头
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("读取 CSV 表头失败: %w", err)
	}
	headers = normalizeColumns(headers)

	// 如果没有提供映射，使用表头作为列名
	if len(mapping.FileColumns) == 0 {
		mapping = ColumnMapping{
			FileColumns:  append([]string(nil), headers...),
			TableColumns: append([]string(nil), headers...),
		}
	}
	mapping = normalizeMapping(mapping)

	// 验证映射
	if len(mapping.FileColumns) != len(mapping.TableColumns) {
		return nil, fmt.Errorf("列映射不匹配: 文件列数 %d, 表列数 %d", len(mapping.FileColumns), len(mapping.TableColumns))
	}

	// 创建列索引映射
	columnIndexMap := make(map[string]int)
	for i, header := range headers {
		columnIndexMap[header] = i
	}

	result := &ImportResult{TotalRows: totalRows}
	rowNumber := 1

	tableColumns := append([]string(nil), mapping.TableColumns...)
	batch := make([]importRow, 0, importBatchSize)
	batchStartRow := 2

	// 逐行读取数据
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.addError(rowNumber, fmt.Sprintf("读取 CSV 行失败: %v", err))
			rowNumber++
			continue
		}

		rowNumber++

		// 构建数据映射
		row := importRow{Values: make([]interface{}, len(mapping.FileColumns))}
		valid := true

		for i, fileCol := range mapping.FileColumns {
			colIndex, exists := columnIndexMap[fileCol]

			if !exists {
				result.addError(rowNumber, fmt.Sprintf("CSV 文件中未找到列: %s", fileCol))
				valid = false
				break
			}

			if colIndex >= len(record) {
				result.addError(rowNumber, fmt.Sprintf("CSV 行数据不完整，缺少列: %s", fileCol))
				valid = false
				break
			}

			// 处理空值
			value := record[colIndex]
			if value == "" {
				row.Values[i] = nil
			} else {
				row.Values[i] = value
			}
		}

		if !valid {
			continue
		}

		// 添加到批次
		if len(batch) == 0 {
			batchStartRow = rowNumber
		}
		batch = append(batch, row)

		// 当批次满时执行插入
		if len(batch) >= importBatchSize {
			successCount, errors := i.executeBatchInsert(ctx, database, table, tableColumns, batch)
			result.SuccessRows += successCount
			for _, err := range errors {
				result.addError(batchStartRow+err.Row, err.Message)
			}
			if err := ctx.Err(); err != nil {
				return result, err
			}
			batch = batch[:0] // 清空批次

			// 报告进度
			if progressCallback != nil {
				progressCallback(minInt(rowNumber-1, result.TotalRows), result.TotalRows)
			}
		}
	}

	// 处理剩余的批次
	if len(batch) > 0 {
		successCount, errors := i.executeBatchInsert(ctx, database, table, tableColumns, batch)
		result.SuccessRows += successCount
		for _, err := range errors {
			result.addError(batchStartRow+err.Row, err.Message)
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
	}

	// 最终进度报告
	if progressCallback != nil {
		progressCallback(result.TotalRows, result.TotalRows)
	}

	return result, nil
}

// ImportFromJSON 从 JSON 文件导入数据
func (i *Importer) ImportFromJSON(database, table string, jsonFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	return i.ImportFromJSONContext(context.Background(), database, table, jsonFilePath, mapping, progressCallback)
}

// ImportFromJSONContext 从 JSON 文件导入数据，支持取消
func (i *Importer) ImportFromJSONContext(ctx context.Context, database, table string, jsonFilePath string, mapping ColumnMapping, progressCallback func(current, total int)) (*ImportResult, error) {
	if database == "" {
		return nil, fmt.Errorf("数据库名称不能为空")
	}
	if table == "" {
		return nil, fmt.Errorf("表名称不能为空")
	}
	if jsonFilePath == "" {
		return nil, fmt.Errorf("JSON 文件路径不能为空")
	}

	totalRows, err := countJSONArrayRows(ctx, jsonFilePath)
	if err != nil {
		return nil, err
	}
	if totalRows == 0 {
		return nil, fmt.Errorf("JSON 文件中没有数据")
	}

	file, err := os.Open(jsonFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开 JSON 文件失败: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := readJSONArrayStart(decoder); err != nil {
		return nil, err
	}

	// 如果没有提供映射，使用第一条记录的键作为列名
	if len(mapping.FileColumns) == 0 {
		firstRecord, err := peekFirstJSONObject(jsonFilePath)
		if err != nil {
			return nil, err
		}
		for key := range firstRecord {
			col := strings.TrimSpace(key)
			mapping.FileColumns = append(mapping.FileColumns, col)
			mapping.TableColumns = append(mapping.TableColumns, col)
		}
	}
	mapping = normalizeMapping(mapping)

	// 验证映射
	if len(mapping.FileColumns) != len(mapping.TableColumns) {
		return nil, fmt.Errorf("列映射不匹配: 文件列数 %d, 表列数 %d", len(mapping.FileColumns), len(mapping.TableColumns))
	}

	result := &ImportResult{
		TotalRows: totalRows,
	}

	tableColumns := append([]string(nil), mapping.TableColumns...)
	batch := make([]importRow, 0, importBatchSize)
	batchStartRow := 1
	rowIndex := 0

	// 逐条处理数据
	for decoder.More() {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		rowIndex++
		var record map[string]interface{}
		if err := decoder.Decode(&record); err != nil {
			return result, fmt.Errorf("解析 JSON 记录失败: %w", err)
		}

		// 构建数据映射
		row := importRow{Values: make([]interface{}, len(mapping.FileColumns))}
		valid := true

		for i, fileCol := range mapping.FileColumns {
			value, exists := record[fileCol]

			if !exists {
				result.addError(rowIndex, fmt.Sprintf("JSON 记录中未找到字段: %s", fileCol))
				valid = false
				break
			}

			row.Values[i] = value
		}

		if !valid {
			continue
		}

		// 添加到批次
		if len(batch) == 0 {
			batchStartRow = rowIndex
		}
		batch = append(batch, row)

		// 当批次满时执行插入
		if len(batch) >= importBatchSize {
			successCount, errors := i.executeBatchInsert(ctx, database, table, tableColumns, batch)
			result.SuccessRows += successCount
			for _, err := range errors {
				result.addError(batchStartRow+err.Row, err.Message)
			}
			if err := ctx.Err(); err != nil {
				return result, err
			}
			batch = batch[:0] // 清空批次

			// 报告进度
			if progressCallback != nil {
				progressCallback(rowIndex, result.TotalRows)
			}
		}
	}
	if err := readJSONArrayEnd(decoder); err != nil {
		return result, err
	}

	// 处理剩余的批次
	if len(batch) > 0 {
		successCount, errors := i.executeBatchInsert(ctx, database, table, tableColumns, batch)
		result.SuccessRows += successCount
		for _, err := range errors {
			result.addError(batchStartRow+err.Row, err.Message)
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
	}

	// 最终进度报告
	if progressCallback != nil {
		progressCallback(result.TotalRows, result.TotalRows)
	}

	return result, nil
}

// executeBatchInsert 执行批量插入
func (i *Importer) executeBatchInsert(ctx context.Context, database, table string, columns []string, batch []importRow) (int, []ImportError) {
	if len(batch) == 0 {
		return 0, nil
	}

	if result, err := i.execBatchInsert(ctx, database, table, columns, batch); err == nil {
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr == nil {
			return int(rowsAffected), nil
		}
		return len(batch), nil
	}

	successCount := 0
	var errors []ImportError

	for idx, row := range batch {
		if err := ctx.Err(); err != nil {
			errors = append(errors, ImportError{
				Row:     idx,
				Message: err.Error(),
			})
			break
		}
		_, err := i.execBatchInsert(ctx, database, table, columns, []importRow{row})
		if err != nil {
			errors = append(errors, ImportError{
				Row:     idx,
				Message: fmt.Sprintf("插入数据失败: %v", err),
			})
		} else {
			successCount++
		}
	}

	return successCount, errors
}

func (i *Importer) execBatchInsert(ctx context.Context, database, table string, columns []string, batch []importRow) (sql.Result, error) {
	if len(columns) == 0 || len(batch) == 0 {
		return nil, fmt.Errorf("插入数据不能为空")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	escapedColumns := make([]string, len(columns))
	for idx, column := range columns {
		escapedColumns[idx] = escapeIdentifier(column)
	}

	rowPlaceholder := "(" + strings.TrimRight(strings.Repeat("?, ", len(columns)), ", ") + ")"
	placeholders := make([]string, len(batch))
	values := make([]interface{}, 0, len(batch)*len(columns))
	for rowIdx, row := range batch {
		if len(row.Values) != len(columns) {
			return nil, fmt.Errorf("插入数据列数不匹配")
		}
		placeholders[rowIdx] = rowPlaceholder
		values = append(values, row.Values...)
	}

	sqlQuery := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES %s",
		escapeIdentifier(database),
		escapeIdentifier(table),
		strings.Join(escapedColumns, ", "),
		strings.Join(placeholders, ", "))

	if len(batch) == 1 {
		return i.db.ExecContext(ctx, sqlQuery, values...)
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, sqlQuery, values...)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// insertRow 插入单行数据
func (i *Importer) insertRow(database, table string, data map[string]interface{}) error {
	columns := make([]string, 0, len(data))
	row := importRow{Values: make([]interface{}, 0, len(data))}
	for column, value := range data {
		columns = append(columns, strings.TrimSpace(column))
		row.Values = append(row.Values, value)
	}
	_, err := i.execBatchInsert(context.Background(), database, table, columns, []importRow{row})
	return err
}

func normalizeColumns(columns []string) []string {
	normalized := make([]string, len(columns))
	for idx, column := range columns {
		normalized[idx] = strings.TrimSpace(column)
	}
	return normalized
}

func normalizeMapping(mapping ColumnMapping) ColumnMapping {
	return ColumnMapping{
		FileColumns:  normalizeColumns(mapping.FileColumns),
		TableColumns: normalizeColumns(mapping.TableColumns),
	}
}

func countCSVDataRows(ctx context.Context, csvFilePath string) (int, error) {
	file, err := os.Open(csvFilePath)
	if err != nil {
		return 0, fmt.Errorf("打开 CSV 文件失败: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	if _, err := reader.Read(); err != nil {
		return 0, fmt.Errorf("读取 CSV 表头失败: %w", err)
	}

	totalRows := 0
	for {
		if err := ctx.Err(); err != nil {
			return totalRows, err
		}
		if _, err := reader.Read(); err == io.EOF {
			break
		} else if err != nil {
			return 0, fmt.Errorf("读取 CSV 数据失败: %w", err)
		}
		totalRows++
	}
	return totalRows, nil
}

func countJSONArrayRows(ctx context.Context, jsonFilePath string) (int, error) {
	file, err := os.Open(jsonFilePath)
	if err != nil {
		return 0, fmt.Errorf("打开 JSON 文件失败: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := readJSONArrayStart(decoder); err != nil {
		return 0, err
	}

	totalRows := 0
	for decoder.More() {
		if err := ctx.Err(); err != nil {
			return totalRows, err
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return totalRows, fmt.Errorf("解析 JSON 记录失败: %w", err)
		}
		totalRows++
	}
	if err := readJSONArrayEnd(decoder); err != nil {
		return totalRows, err
	}
	return totalRows, nil
}

func peekFirstJSONObject(jsonFilePath string) (map[string]interface{}, error) {
	file, err := os.Open(jsonFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开 JSON 文件失败: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := readJSONArrayStart(decoder); err != nil {
		return nil, err
	}
	if !decoder.More() {
		return nil, fmt.Errorf("JSON 文件中没有数据")
	}

	var record map[string]interface{}
	if err := decoder.Decode(&record); err != nil {
		return nil, fmt.Errorf("解析 JSON 记录失败: %w", err)
	}
	return record, nil
}

func readJSONArrayStart(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("解析 JSON 文件失败: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '[' {
		return fmt.Errorf("JSON 文件必须是数组")
	}
	return nil
}

func readJSONArrayEnd(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("解析 JSON 文件失败: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != ']' {
		return fmt.Errorf("JSON 文件数组结束标记无效")
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ValidateCSVFormat 验证 CSV 文件格式
func (i *Importer) ValidateCSVFormat(csvFilePath string) error {
	file, err := os.Open(csvFilePath)
	if err != nil {
		return fmt.Errorf("打开 CSV 文件失败: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// 读取表头
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("读取 CSV 表头失败: %w", err)
	}

	if len(headers) == 0 {
		return fmt.Errorf("CSV 文件表头为空")
	}

	// 读取第一行数据验证格式
	_, err = reader.Read()
	if err != nil && err != io.EOF {
		return fmt.Errorf("读取 CSV 数据失败: %w", err)
	}

	return nil
}

// ValidateJSONFormat 验证 JSON 文件格式
func (i *Importer) ValidateJSONFormat(jsonFilePath string) error {
	file, err := os.Open(jsonFilePath)
	if err != nil {
		return fmt.Errorf("打开 JSON 文件失败: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := readJSONArrayStart(decoder); err != nil {
		return err
	}
	if !decoder.More() {
		return fmt.Errorf("JSON 文件中没有数据")
	}
	var firstRecord map[string]interface{}
	if err := decoder.Decode(&firstRecord); err != nil {
		return fmt.Errorf("JSON 格式无效: %w", err)
	}
	if len(firstRecord) == 0 {
		return fmt.Errorf("JSON 第一条记录为空")
	}

	return nil
}

// ValidateSQLFormat 验证 SQL 文件格式
func (i *Importer) ValidateSQLFormat(sqlFilePath string) error {
	file, err := os.Open(sqlFilePath)
	if err != nil {
		return fmt.Errorf("打开 SQL 文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hasValidSQL := false

	// 简单的 SQL 语句正则表达式
	sqlPattern := regexp.MustCompile(`(?i)^\s*(INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|SELECT)\s+`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "#") {
			continue
		}

		if sqlPattern.MatchString(line) {
			hasValidSQL = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 SQL 文件失败: %w", err)
	}

	if !hasValidSQL {
		return fmt.Errorf("SQL 文件中没有有效的 SQL 语句")
	}

	return nil
}
