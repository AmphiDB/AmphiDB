<template>
  <div class="data-manager">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-button type="primary" size="small" @click="showInsertDialog" :icon="Plus">插入</el-button>
        <el-button type="danger" size="small" @click="handleDelete" :disabled="selectedRows.length === 0" :icon="Delete">
          删除 ({{ selectedRows.length }})
        </el-button>
        <el-button size="small" @click="refreshData" :icon="Refresh">刷新</el-button>
      </div>
      <div class="toolbar-right">
        <el-button size="small" @click="toggleFilter" :icon="Filter">
          {{ showFilter ? '隐藏筛选' : '显示筛选' }}
        </el-button>
        <el-button size="small" @click="showExportDialog" :icon="Download">导出</el-button>
        <el-button size="small" @click="showImportDialog" :icon="Upload">导入</el-button>
      </div>
    </div>

    <!-- SQL 输入栏 -->
    <div class="sql-bar">
      <div class="sql-input-row">
        <!-- Custom SQL input with manual suggestion dropdown -->
        <div class="sql-input-wrap" ref="sqlWrapRef">
          <el-input
            ref="sqlInputRef"
            v-model="sqlInput"
            placeholder="输入 SQL 查询，Ctrl+Enter 执行"
            class="sql-input"
            clearable
            @input="onSqlInput"
            @keydown.ctrl.enter.prevent="executeSql"
            @keydown.esc="closeSuggest"
            @keydown="onSqlKeydown"
            @blur="onSqlBlur"
          />
          <!-- Suggestion dropdown -->
          <div v-if="suggestVisible && suggestItems.length" class="sql-suggest-list">
            <div
              v-for="(item, idx) in suggestItems"
              :key="item.value"
              class="sql-suggest-item"
              :class="{ active: idx === suggestIndex }"
              @mousedown.prevent="pickSuggest(item)"
            >
              <span class="sql-suggest-value">{{ item.value }}</span>
              <span class="sql-suggest-meta">{{ item.meta }}</span>
            </div>
          </div>
        </div>
        <el-button type="primary" size="small" :loading="sqlLoading" @click="executeSql">执行</el-button>
        <el-button size="small" @click="resetSql">重置</el-button>
      </div>
      <div v-if="sqlError" class="sql-error">{{ sqlError }}</div>
      <div v-if="sqlMode && !sqlError" class="sql-hint">
        自定义 SQL 模式 — 共 {{ totalRows }} 行
        <el-button link size="small" @click="resetSql">退出</el-button>
      </div>
    </div>

    <!-- 筛选器 -->
    <div v-if="showFilter && !sqlMode" class="filter-container">
      <DataFilter :columns="columns" @apply="handleFilterApply" @clear="handleFilterClear" ref="filterRef" />
    </div>

    <!-- 数据表格 -->
    <div class="grid-container">
      <DataGrid
        :data="dataRows"
        :columns="columns"
        :column-schemas="columnSchemas"
        :foreign-keys="foreignKeys"
        :profile-id="profileId"
        :database="database"
        :total="totalRows"
        :loading="loading || sqlLoading"
        :sortable="!sqlMode"
        :editable="!sqlMode"
        :show-pagination="true"
        :page-size="pageSize"
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
        @cell-edit="handleCellEdit"
        @page-change="handlePageChange"
        ref="gridRef"
      />
    </div>

    <!-- 插入对话框 -->
    <DataInsertDialog
      v-model="insertDialogVisible"
      :columns="tableColumns"
      :foreign-keys="foreignKeys"
      :profile-id="profileId"
      :database="database"
      :table="table"
      @success="handleInsertSuccess"
    />

    <!-- 导出对话框 -->
    <ExportDialog
      v-model="exportDialogVisible"
      :profile-id="profileId"
      :database="database"
      :table="table"
      :filters="filters"
      :order-by="orderBy"
      @success="handleExportSuccess"
    />

    <!-- 导入对话框 -->
    <ImportDialog
      v-model="importDialogVisible"
      :profile-id="profileId"
      :database="database"
      :table="table"
      :table-columns="tableColumns"
      @success="handleImportSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Plus, Delete, Refresh, Filter, Download, Upload } from '@element-plus/icons-vue';
import DataGrid from './DataGrid.vue';
import DataFilter from './DataFilter.vue';
import DataInsertDialog from './DataInsertDialog.vue';
import ExportDialog from './ExportDialog.vue';
import ImportDialog from './ImportDialog.vue';
import { DataAPI, SchemaAPI, QueryAPI } from '../api';
import type { Filter as FilterType, OrderBy, Column, ForeignKey, TableSchema } from '../types/api';

interface Props {
  profileId: string;
  database: string;
  table: string;
}

const props = defineProps<Props>();

const loading = ref(false);
const showFilter = ref(false);
const insertDialogVisible = ref(false);
const exportDialogVisible = ref(false);
const importDialogVisible = ref(false);

// 数据
const dataRows = ref<any[][]>([]);
const columns = ref<string[]>([]);
const totalRows = ref(0);
const pageSize = ref(100);
const currentPage = ref(1);

// 表结构
const tableSchema = ref<TableSchema | null>(null);
const tableColumns = ref<Column[]>([]);
const foreignKeys = ref<ForeignKey[]>([]);

// 列结构映射
const columnSchemas = computed(() => {
  const schemas: Record<string, Column> = {};
  tableColumns.value.forEach(col => {
    schemas[col.name] = col;
  });
  return schemas;
});

// 主键列
const primaryKeyColumns = computed(() => {
  if (!tableSchema.value || !tableSchema.value.indexes) {
    return [];
  }
  // 主键在 indexes 中，type 为 "PRIMARY"
  const primaryIndex = tableSchema.value.indexes.find(idx => idx.type === 'PRIMARY');
  return primaryIndex ? primaryIndex.columns : [];
});

// 查询条件
const filters = ref<FilterType[]>([]);
const orderBy = ref<OrderBy[]>([]);
const selectedRows = ref<any[]>([]);

// 引用
const gridRef = ref<any>(null);
const filterRef = ref<any>(null);

// ── SQL 输入栏 ──────────────────────────────────────────────────────────────────
const sqlInput = ref('')
const sqlInputRef = ref<any>(null)
const sqlWrapRef = ref<HTMLElement | null>(null)
const sqlMode = ref(false)
const sqlLoading = ref(false)
const sqlError = ref('')

// Suggestion state
const suggestItems = ref<SqlSuggestItem[]>([])
const suggestVisible = ref(false)
const suggestIndex = ref(-1)

const defaultSql = () => `SELECT * FROM \`${props.table}\``

const resetSql = () => {
  sqlInput.value = defaultSql()
  sqlMode.value = false
  sqlError.value = ''
  suggestVisible.value = false
  currentPage.value = 1
  filters.value = []
  orderBy.value = []
  loadData()
}

const executeSql = async () => {
  const sql = sqlInput.value.trim()
  if (!sql) return
  suggestVisible.value = false
  sqlError.value = ''
  sqlLoading.value = true
  sqlMode.value = true
  try {
    // Use the database-aware API to avoid "No database selected" errors
    const go = (window as any)['go']['backend']['App']
    const result = await go.ExecuteQueryInDatabase(props.profileId, props.database, sql)
    if (result?.error?.message || result?.Error?.Message) {
      sqlError.value = result?.error?.message || result?.Error?.Message
      sqlMode.value = false
      return
    }
    columns.value = result?.columns || result?.Columns || []
    dataRows.value = result?.rows || result?.Rows || []
    totalRows.value = (result?.rows || result?.Rows || []).length
  } catch (e: any) {
    sqlError.value = e?.message || String(e)
    sqlMode.value = false
  } finally {
    sqlLoading.value = false
  }
}

// SQL keyword / function suggestions
const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'LIKE', 'BETWEEN',
  'ORDER BY', 'GROUP BY', 'HAVING', 'LIMIT', 'OFFSET', 'JOIN', 'LEFT JOIN',
  'RIGHT JOIN', 'INNER JOIN', 'ON', 'AS', 'DISTINCT', 'INSERT INTO', 'UPDATE',
  'SET', 'DELETE FROM', 'IS NULL', 'IS NOT NULL', 'ASC', 'DESC',
]

const SQL_FUNCTIONS = [
  'COUNT(*)', 'COUNT(DISTINCT )', 'SUM()', 'AVG()', 'MIN()', 'MAX()',
  'COALESCE()', 'IFNULL()', 'IF()', 'CONCAT()', 'SUBSTRING()', 'LENGTH()',
  'TRIM()', 'UPPER()', 'LOWER()', 'NOW()', 'DATE()', 'YEAR()', 'MONTH()',
  'DAY()', 'UNIX_TIMESTAMP()', 'FROM_UNIXTIME()', 'DATE_FORMAT()',
  'ROUND()', 'FLOOR()', 'CEIL()', 'ABS()', 'CAST()', 'CONVERT()',
  'JSON_EXTRACT()', 'JSON_UNQUOTE()',
]

interface SqlSuggestItem { value: string; meta: string }

const buildSuggestions = (fragment: string): SqlSuggestItem[] => {
  if (!fragment) return []
  const up = fragment.toUpperCase()
  const results: SqlSuggestItem[] = []
  for (const col of columns.value) {
    if (col.toUpperCase().includes(up)) results.push({ value: col, meta: '字段' })
  }
  for (const kw of SQL_KEYWORDS) {
    if (kw.startsWith(up)) results.push({ value: kw, meta: '关键字' })
  }
  for (const fn of SQL_FUNCTIONS) {
    if (fn.toUpperCase().startsWith(up)) results.push({ value: fn, meta: '函数' })
  }
  return results.slice(0, 20)
}

// Get the word fragment before the cursor
const getFragment = (): { fragment: string; start: number } => {
  const el = sqlInputRef.value?.$el?.querySelector('input') as HTMLInputElement | null
  const pos = el?.selectionStart ?? sqlInput.value.length
  const before = sqlInput.value.slice(0, pos)
  const m = before.match(/([\w.*`]*)$/)
  const fragment = m ? m[1] : ''
  return { fragment, start: pos - fragment.length }
}

const onSqlInput = () => {
  const { fragment } = getFragment()
  if (fragment.length >= 1) {
    suggestItems.value = buildSuggestions(fragment)
    suggestVisible.value = suggestItems.value.length > 0
    suggestIndex.value = -1
  } else {
    suggestVisible.value = false
  }
}

const closeSuggest = () => {
  suggestVisible.value = false
  suggestIndex.value = -1
}

const onSqlBlur = () => {
  // Delay so mousedown on suggestion fires first
  setTimeout(closeSuggest, 150)
}

const onSqlKeydown = (e: KeyboardEvent) => {
  if (!suggestVisible.value) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    suggestIndex.value = Math.min(suggestIndex.value + 1, suggestItems.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    suggestIndex.value = Math.max(suggestIndex.value - 1, 0)
  } else if (e.key === 'Enter' && !e.ctrlKey) {
    if (suggestIndex.value >= 0) {
      e.preventDefault()
      e.stopPropagation()
      pickSuggest(suggestItems.value[suggestIndex.value])
    }
    // If no item selected, let Enter fall through (do nothing — user must use Ctrl+Enter to execute)
  } else if (e.key === 'Tab') {
    if (suggestIndex.value >= 0) {
      e.preventDefault()
      pickSuggest(suggestItems.value[suggestIndex.value])
    } else if (suggestItems.value.length > 0) {
      e.preventDefault()
      pickSuggest(suggestItems.value[0])
    }
  }
}

const pickSuggest = (item: SqlSuggestItem) => {
  const el = sqlInputRef.value?.$el?.querySelector('input') as HTMLInputElement | null
  const pos = el?.selectionStart ?? sqlInput.value.length
  const before = sqlInput.value.slice(0, pos)
  const after = sqlInput.value.slice(pos)
  // Replace the fragment before cursor with the suggestion
  const replaced = before.replace(/([\w.*`]*)$/, item.value)
  sqlInput.value = replaced + after
  closeSuggest()
  // Restore cursor after inserted text
  const newPos = replaced.length
  setTimeout(() => {
    el?.focus()
    el?.setSelectionRange(newPos, newPos)
  }, 0)
}

// Keep fetchSqlSuggestions for any remaining references (unused now)
const fetchSqlSuggestions = (_q: string, cb: (items: SqlSuggestItem[]) => void) => cb([])
const onSuggestSelect = (_item: SqlSuggestItem) => {}

// 加载表结构
const loadTableSchema = async () => {
  try {
    const schema = await SchemaAPI.getTableSchema(props.profileId, props.database, props.table);
    tableSchema.value = schema;
    
    // 安全地处理可能为 null 的数组
    tableColumns.value = schema.columns || [];
    foreignKeys.value = schema.foreignKeys || [];
    
    // 立即从 schema 设置 columns，这样筛选器就能显示列名
    if (tableColumns.value.length > 0) {
      columns.value = tableColumns.value.map(col => col.name);
    }
    
    console.log('Table schema loaded:', {
      columns: columns.value,
      tableColumns: tableColumns.value.length,
      foreignKeys: foreignKeys.value.length
    });
  } catch (error: any) {
    console.error('Failed to load table schema:', error);
    ElMessage.error(`加载表结构失败: ${error.message || error}`);
  }
};

// 加载数据
const loadData = async () => {
  loading.value = true;

  try {
    // 查询数据 - 注意：Wails 生成的类型使用大写开头的属性名
    const result = await DataAPI.queryData(props.profileId, {
      Database: props.database,
      Table: props.table,
      Columns: [],
      Filters: filters.value,
      OrderBy: orderBy.value,
      Limit: pageSize.value,
      Offset: (currentPage.value - 1) * pageSize.value,
    } as any);

    // Wails 返回的属性名也是大写开头的
    dataRows.value = result.Rows || [];
    // 只在 columns 为空时才从结果中设置（优先使用 schema 中的列名）
    if (columns.value.length === 0) {
      columns.value = result.Columns || [];
    }
    totalRows.value = result.Total || 0;
    
    console.log('Data loaded:', {
      rows: dataRows.value.length,
      columns: columns.value,
      total: totalRows.value
    });
  } catch (error: any) {
    console.error('Failed to load data:', error);
    ElMessage.error(`加载数据失败: ${error.message || error}`);
  } finally {
    loading.value = false;
  }
};

// 刷新数据
const refreshData = () => {
  loadData();
};

// 切换筛选器
const toggleFilter = () => {
  showFilter.value = !showFilter.value;
};

// 显示插入对话框
const showInsertDialog = () => {
  insertDialogVisible.value = true;
};

// 显示导出对话框
const showExportDialog = () => {
  exportDialogVisible.value = true;
};

// 显示导入对话框
const showImportDialog = () => {
  importDialogVisible.value = true;
};

// 处理导出成功
const handleExportSuccess = () => {
  ElMessage.success('导出完成');
};

// 处理导入成功
const handleImportSuccess = () => {
  refreshData();
  ElMessage.success('导入完成');
};

// 处理插入成功
const handleInsertSuccess = () => {
  refreshData();
};

// 处理选择变化
const handleSelectionChange = (rows: any[]) => {
  selectedRows.value = rows;
};

// 处理排序变化
const handleSortChange = (newOrderBy: OrderBy[]) => {
  // 转换为 Wails 需要的大写格式
  orderBy.value = newOrderBy.map(o => ({
    Column: (o as any).column || (o as any).Column,
    Direction: (o as any).direction || (o as any).Direction,
  } as any));
  currentPage.value = 1;
  loadData();
};

// 处理筛选应用
const handleFilterApply = (newFilters: FilterType[]) => {
  // 转换为 Wails 需要的大写格式
  filters.value = newFilters.map(f => ({
    Column: (f as any).column || (f as any).Column,
    Operator: (f as any).operator || (f as any).Operator,
    Value: (f as any).value || (f as any).Value,
  } as any));
  currentPage.value = 1;
  loadData();
};

// 处理筛选清除
const handleFilterClear = () => {
  filters.value = [];
  currentPage.value = 1;
  loadData();
};

// 处理单元格编辑
const handleCellEdit = async (rowIndex: number, column: string, oldValue: any, newValue: any) => {
  loading.value = true;

  try {
    // 获取主键值
    const pk: Record<string, any> = {};
    primaryKeyColumns.value.forEach((pkCol) => {
      const colIndex = columns.value.indexOf(pkCol);
      if (colIndex >= 0) {
        pk[pkCol] = dataRows.value[rowIndex][colIndex];
      }
    });

    if (Object.keys(pk).length === 0) {
      ElMessage.error('无法确定主键，无法更新数据');
      return;
    }

    // 更新数据
    const data: Record<string, any> = { [column]: newValue };
    await DataAPI.updateRow(props.profileId, props.database, props.table, pk, data);

    ElMessage.success('更新成功');
    
    // 更新本地数据
    const colIndex = columns.value.indexOf(column);
    if (colIndex >= 0) {
      dataRows.value[rowIndex][colIndex] = newValue;
    }
  } catch (error: any) {
    console.error('Failed to update row:', error);
    ElMessage.error(`更新失败: ${error.message || error}`);
    
    // 恢复原值
    const colIndex = columns.value.indexOf(column);
    if (colIndex >= 0) {
      dataRows.value[rowIndex][colIndex] = oldValue;
    }
  } finally {
    loading.value = false;
  }
};

// 处理删除
const handleDelete = async () => {
  if (selectedRows.value.length === 0) {
    return;
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedRows.value.length} 行数据吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );
  } catch {
    return;
  }

  loading.value = true;

  try {
    // 构建主键列表
    const pks: Array<Record<string, any>> = [];
    
    for (const row of selectedRows.value) {
      const pk: Record<string, any> = {};
      primaryKeyColumns.value.forEach((pkCol) => {
        if (row[pkCol] !== undefined) {
          pk[pkCol] = row[pkCol];
        }
      });
      
      if (Object.keys(pk).length > 0) {
        pks.push(pk);
      }
    }

    if (pks.length === 0) {
      ElMessage.error('无法确定主键，无法删除数据');
      return;
    }

    await DataAPI.deleteRows(props.profileId, props.database, props.table, pks);

    ElMessage.success(`成功删除 ${pks.length} 行数据`);
    
    // 清除选择
    gridRef.value?.clearSelection();
    selectedRows.value = [];
    
    // 刷新数据
    refreshData();
  } catch (error: any) {
    console.error('Failed to delete rows:', error);
    ElMessage.error(`删除失败: ${error.message || error}`);
  } finally {
    loading.value = false;
  }
};

// 处理分页变化
const handlePageChange = (page: number, size: number) => {
  currentPage.value = page;
  pageSize.value = size;
  loadData();
};

// 监听表变化
watch(() => [props.profileId, props.database, props.table], () => {
  currentPage.value = 1;
  filters.value = [];
  orderBy.value = [];
  selectedRows.value = [];
  sqlMode.value = false;
  sqlError.value = '';
  sqlInput.value = defaultSql();
  loadTableSchema();
  loadData();
}, { immediate: false });

// 组件挂载
onMounted(() => {
  sqlInput.value = defaultSql();
  loadTableSchema();
  loadData();
});
</script>

<style scoped>
.data-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  background-color: #fff;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  gap: 8px;
}

.filter-container {
  margin-bottom: 16px;
}

.grid-container {
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

/* ── SQL bar ── */
.sql-bar {
  margin-bottom: 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sql-input-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.sql-input-wrap {
  flex: 1;
  position: relative;
}

.sql-input {
  width: 100%;
}

.sql-input :deep(.el-input__wrapper) {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 13px;
}

/* Custom suggestion dropdown */
.sql-suggest-list {
  position: absolute;
  top: calc(100% + 2px);
  left: 0;
  right: 0;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  max-height: 220px;
  overflow-y: auto;
  z-index: 9999;
}

.sql-suggest-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.1s;
}

.sql-suggest-item:hover,
.sql-suggest-item.active {
  background: #ecf5ff;
}

.sql-suggest-value {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  color: #303133;
}

.sql-suggest-meta {
  font-size: 11px;
  color: #909399;
  margin-left: 8px;
  flex-shrink: 0;
}

.sql-error {
  font-size: 12px;
  color: #f56c6c;
  padding: 2px 4px;
}

.sql-hint {
  font-size: 12px;
  color: #67c23a;
  padding: 2px 4px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.sql-suggest-value {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 13px;
  color: #303133;
}

.sql-suggest-meta {
  float: right;
  font-size: 11px;
  color: #909399;
  margin-left: 8px;
}
</style>
