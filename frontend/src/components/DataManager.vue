<template>
  <div class="data-manager">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-main">
        <div class="toolbar-left">
          <el-button type="primary" size="small" @click="showInsertDialog" :icon="Plus">插入</el-button>
          <el-button size="small" @click="refreshData" :icon="Refresh">刷新</el-button>
          <el-button size="small" @click="toggleFilter" :icon="Filter" :type="showFilter ? 'primary' : undefined" plain>
            {{ showFilter ? '隐藏筛选' : '筛选' }}
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-button size="small" @click="showExportDialog" :icon="Download">导出</el-button>
          <el-button size="small" @click="showImportDialog" :icon="Upload">导入</el-button>
          <el-button
            size="small"
            :type="selectedRows.length > 0 ? 'danger' : undefined"
            :plain="selectedRows.length === 0"
            @click="handleDelete"
            :disabled="selectedRows.length === 0"
            :icon="Delete"
            class="delete-button"
          >
            删除选中<span v-if="selectedRows.length > 0"> {{ selectedRows.length }}</span>
          </el-button>
        </div>
      </div>
      <div class="toolbar-status">
        <span class="table-name" :title="tableIdentity">{{ tableIdentity }}</span>
        <span>{{ rowCountLabel }}</span>
        <span>每页 {{ pageSize }}</span>
        <span :class="{ active: selectedRows.length > 0 }">已选 {{ selectedRows.length }}</span>
        <el-tag size="small" effect="plain" :type="queryStateType">{{ queryStateLabel }}</el-tag>
      </div>
    </div>

    <!-- SQL 输入栏 -->
    <div class="sql-bar">
      <NaturalLanguageSqlBox
        class="data-ai-box"
        :profile-id="profileId"
        :database="database"
        :current-table="table"
        placeholder="白话查询当前表，例如：查最近 7 天状态异常的数据"
        @generated="applyGeneratedSql"
      />
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
        <el-button size="small" @click="resetSql">重置</el-button>
        <el-button type="primary" size="small" :loading="sqlLoading" @click="executeSql">执行</el-button>
        <span class="sql-command-hint">Ctrl+Enter 执行</span>
      </div>
      <div v-if="sqlError" class="sql-error">{{ sqlError }}</div>
      <div v-if="sqlMode && !sqlError" class="sql-hint">
        自定义 SQL 结果，共 {{ totalRows }} 行
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
        :editable="true"
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
import NaturalLanguageSqlBox from './NaturalLanguageSqlBox.vue';
import { DataAPI, SchemaAPI, QueryAPI } from '../api';
import type { Filter as FilterType, OrderBy, Column, ForeignKey, TableSchema } from '../types/api';
import {
  SQL_FUNCTIONS,
  SQL_KEYWORDS,
  extractTablesFromSql,
  filterCompletionValues,
  getCompletionContext,
  preferredFieldTables,
  tableForQualifier,
} from '../utils/sqlCompletion';

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

const tableIdentity = computed(() => `${props.database}.${props.table}`);
const rowCountLabel = computed(() => sqlMode.value ? `结果 ${totalRows.value} 行` : `总行 ${totalRows.value}`);
const queryStateLabel = computed(() => {
  if (sqlMode.value) {
    return 'SQL';
  }
  if (filters.value.length > 0) {
    return `筛选 ${filters.value.length}`;
  }
  return '默认视图';
});
type QueryStateTagType = 'success' | 'warning' | 'info';

const queryStateType = computed<QueryStateTagType>(() => {
  if (sqlMode.value) {
    return 'success';
  }
  if (filters.value.length > 0) {
    return 'warning';
  }
  return 'info';
});

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
let dataRequestToken = 0
let schemaRequestToken = 0
let sqlRequestToken = 0

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
  const requestToken = ++sqlRequestToken
  const requestProfileId = props.profileId
  const requestDatabase = props.database
  let sql = sqlInput.value.trim()
  if (!sql) return
  suggestVisible.value = false
  sqlError.value = ''
  sqlLoading.value = true
  sqlMode.value = true
  
  // 如果是SELECT查询且没有LIMIT子句，默认添加LIMIT 20
  const upperSql = sql.toUpperCase()
  if (upperSql.startsWith('SELECT') && !upperSql.includes('LIMIT')) {
    sql = sql.replace(/;?\s*$/, '') + ' LIMIT 20'
  }
  
  try {
    // Use the database-aware API to avoid "No database selected" errors
    const go = (window as any)['go']['backend']['App']
    const result = await go.ExecuteQueryInDatabase(requestProfileId, requestDatabase, sql)
    if (requestToken !== sqlRequestToken || requestDatabase !== props.database) return
    if (result?.error?.message || result?.Error?.Message) {
      sqlError.value = result?.error?.message || result?.Error?.Message
      sqlMode.value = false
      return
    }
    columns.value = result?.columns || result?.Columns || []
    dataRows.value = result?.rows || result?.Rows || []
    totalRows.value = (result?.rows || result?.Rows || []).length
  } catch (e: any) {
    if (requestToken !== sqlRequestToken || requestDatabase !== props.database) return
    sqlError.value = e?.message || String(e)
    sqlMode.value = false
  } finally {
    if (requestToken === sqlRequestToken) {
      sqlLoading.value = false
    }
  }
}

const applyGeneratedSql = (sql: string) => {
  sqlInput.value = sql
  sqlMode.value = false
  sqlError.value = ''
  suggestVisible.value = false
  ElMessage.success('SQL 已生成，请确认后执行')
}

interface SqlSuggestItem { value: string; meta: string; from?: number }

const availableTables = computed(() => [props.table]);
const availableColumnsByTable = computed<Record<string, string[]>>(() => ({
  [props.table]: tableColumns.value.length > 0 ? tableColumns.value.map(column => column.name) : columns.value,
}));

const buildSuggestions = (sqlBeforeCursor: string): SqlSuggestItem[] => {
  const context = getCompletionContext(sqlBeforeCursor);
  const tableRefs = extractTablesFromSql(sqlBeforeCursor);

  if (context.type === 'table') {
    return filterCompletionValues(availableTables.value, context.fragment)
      .map((value: string) => ({ value, meta: '表', from: context.from }))
      .slice(0, 20);
  }

  if (context.type === 'member') {
    const tableName = tableForQualifier(context.qualifier, tableRefs) || props.table;
    return filterCompletionValues(availableColumnsByTable.value[tableName] || [], context.fragment)
      .map((value: string) => ({ value, meta: `${tableName} 字段`, from: context.from }))
      .slice(0, 20);
  }

  const preferredTables = preferredFieldTables(sqlBeforeCursor);
  const fieldTables = preferredTables.length > 0 ? preferredTables : [props.table];
  const fieldSuggestions = fieldTables.flatMap((tableName: string) => {
    return filterCompletionValues(availableColumnsByTable.value[tableName] || [], context.fragment)
      .map((value: string) => ({ value, meta: `${tableName} 字段`, from: context.from }));
  });
  const functionSuggestions = filterCompletionValues(SQL_FUNCTIONS, context.fragment)
    .map((value: string) => ({ value, meta: 'SQL 函数', from: context.from }));
  const keywordSuggestions = filterCompletionValues(SQL_KEYWORDS, context.fragment)
    .map((value: string) => ({ value, meta: '关键字', from: context.from }));

  return [...fieldSuggestions, ...functionSuggestions, ...keywordSuggestions].slice(0, 20);
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
  const { fragment, start } = getFragment()
  const sqlBeforeCursor = sqlInput.value.slice(0, start + fragment.length)
  if (fragment.length >= 1 || /\.\s*$/.test(sqlBeforeCursor)) {
    suggestItems.value = buildSuggestions(sqlBeforeCursor)
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
  const replaceFrom = item.from ?? pos
  const before = sqlInput.value.slice(0, replaceFrom)
  const after = sqlInput.value.slice(pos)
  const replaced = before + item.value
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
  const requestToken = ++schemaRequestToken
  const requestProfileId = props.profileId
  const requestDatabase = props.database
  const requestTable = props.table
  try {
    const schema = await SchemaAPI.getTableSchema(requestProfileId, requestDatabase, requestTable);
    if (
      requestToken !== schemaRequestToken ||
      requestDatabase !== props.database ||
      requestTable !== props.table
    ) return;
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
    if (
      requestToken !== schemaRequestToken ||
      requestDatabase !== props.database ||
      requestTable !== props.table
    ) return;
    console.error('Failed to load table schema:', error);
    ElMessage.error(`加载表结构失败: ${error.message || error}`);
  }
};

// 加载数据
const loadData = async () => {
  const requestToken = ++dataRequestToken
  const requestProfileId = props.profileId
  const requestDatabase = props.database
  const requestTable = props.table
  loading.value = true;

  try {
    // 查询数据 - 注意：Wails 生成的类型使用大写开头的属性名
    const result = await DataAPI.queryData(requestProfileId, {
      Database: requestDatabase,
      Table: requestTable,
      Columns: [],
      Filters: filters.value,
      OrderBy: orderBy.value,
      Limit: pageSize.value,
      Offset: (currentPage.value - 1) * pageSize.value,
    } as any);
    if (
      requestToken !== dataRequestToken ||
      requestDatabase !== props.database ||
      requestTable !== props.table
    ) return;

    // Wails 返回的属性名在不同调用链里可能是大写或小写。
    const normalizedResult = result as any;
    dataRows.value = normalizedResult.rows || normalizedResult.Rows || [];
    // 只在 columns 为空时才从结果中设置（优先使用 schema 中的列名）
    if (columns.value.length === 0) {
      columns.value = normalizedResult.columns || normalizedResult.Columns || [];
    }
    totalRows.value = normalizedResult.total || normalizedResult.Total || 0;
    
    console.log('Data loaded:', {
      rows: dataRows.value.length,
      columns: columns.value,
      total: totalRows.value
    });
  } catch (error: any) {
    if (
      requestToken !== dataRequestToken ||
      requestDatabase !== props.database ||
      requestTable !== props.table
    ) return;
    console.error('Failed to load data:', error);
    ElMessage.error(`加载数据失败: ${error.message || error}`);
  } finally {
    if (requestToken === dataRequestToken) {
      loading.value = false;
    }
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
  flex-direction: column;
  margin-bottom: 10px;
  padding: 8px 10px;
  background-color: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  gap: 6px;
}

.toolbar-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.delete-button {
  margin-left: 2px;
}

.toolbar-status {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 22px;
  color: #606266;
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
  overflow: hidden;
}

.toolbar-status > span {
  position: relative;
  flex-shrink: 0;
}

.toolbar-status > span + span::before {
  content: '';
  display: inline-block;
  width: 1px;
  height: 10px;
  margin-right: 10px;
  background-color: #dcdfe6;
  vertical-align: -1px;
}

.toolbar-status .table-name {
  max-width: 320px;
  overflow: hidden;
  color: #303133;
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-weight: 600;
  text-overflow: ellipsis;
}

.toolbar-status .active {
  color: #f56c6c;
  font-weight: 600;
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
  flex-wrap: wrap;
}

.sql-input-wrap {
  flex: 1;
  min-width: 280px;
  position: relative;
}

.sql-input {
  width: 100%;
}

.sql-input :deep(.el-input__wrapper) {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 13px;
}

.sql-command-hint {
  flex-shrink: 0;
  color: #909399;
  font-size: 12px;
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
