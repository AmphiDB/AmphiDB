<template>
  <div class="data-grid" ref="gridRootRef">
    <!-- 传统表格（支持编辑、正确颜色，分页已限制行数） -->
    <el-table
      :data="displayData"
      border
      stripe
      highlight-current-row
      :max-height="tableHeight"
      @selection-change="handleSelectionChange"
      @sort-change="handleSortChange"
      @cell-dblclick="handleCellDoubleClick"
      style="width: 100%"
      v-loading="loading"
    >
      <!-- 选择列 -->
      <el-table-column type="selection" width="55" />
      
      <!-- 数据列 -->
      <el-table-column
        v-for="column in columns"
        :key="column"
        :prop="column"
        :label="column"
        :sortable="sortable ? 'custom' : false"
        min-width="120"
        show-overflow-tooltip
      >
        <template #default="{ row, $index }">
          <div
            v-if="editingCell.row === $index && editingCell.column === column"
            class="cell-editor-wrapper"
            @click.stop
          >
            <CellEditor
              v-if="columnSchemas && columnSchemas[column]"
              :value="editingValue"
              :column="columnSchemas[column]"
              :foreign-keys="foreignKeys"
              :profile-id="profileId"
              :database="database"
              @change="handleCellEditComplete"
              @cancel="handleCellEditCancel"
            />
            <el-input
              v-else
              v-model="editingValue"
              size="small"
              @blur="handleCellEditComplete"
              @keyup.enter="handleCellEditComplete"
              @keyup.esc="handleCellEditCancel"
              ref="cellInput"
            />
          </div>
          <div 
            v-else 
            class="cell-content"
            @dblclick="handleCellContentDoubleClick($index, column, row[column])"
          >
            {{ formatCellValue(row[column]) }}
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- JSON 查看/编辑对话框 -->
    <el-dialog
      v-model="jsonDialogVisible"
      title="JSON 字段"
      width="700px"
      :close-on-click-modal="false"
    >
      <div class="json-dialog-toolbar">
        <el-button size="small" @click="copyJsonContent">复制</el-button>
        <el-button
          v-if="props.editable"
          size="small"
          type="primary"
          @click="confirmJsonEdit"
        >保存</el-button>
      </div>
      <el-input
        v-model="jsonEditContent"
        type="textarea"
        :rows="18"
        :readonly="!props.editable"
        class="json-textarea"
        spellcheck="false"
      />
    </el-dialog>

    <!-- 分页 -->
    <div class="pagination-container" v-if="showPagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[50, 100, 200, 500, 1000]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handlePageSizeChange"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, h, onMounted, onUnmounted } from 'vue';
import { ElCheckbox, ElMessage } from 'element-plus';
import CellEditor from './CellEditor.vue';
import type { OrderBy, Column, ForeignKey } from '../types/api';

interface Props {
  data?: any[][];
  columns?: string[];
  columnSchemas?: Record<string, Column>;
  foreignKeys?: ForeignKey[];
  profileId?: string;
  database?: string;
  total?: number;
  loading?: boolean;
  sortable?: boolean;
  editable?: boolean;
  showPagination?: boolean;
  pageSize?: number;
  virtualScrollThreshold?: number; // 超过此行数启用虚拟滚动
}

const props = withDefaults(defineProps<Props>(), {
  data: () => [],
  columns: () => [],
  total: 0,
  loading: false,
  sortable: true,
  editable: true,
  showPagination: true,
  pageSize: 100,
  columnSchemas: () => ({}),
  foreignKeys: () => [],
  profileId: '',
  database: '',
  virtualScrollThreshold: 50, // 超过此行数启用虚拟滚动，减少滚动卡顿
});

const emit = defineEmits<{
  selectionChange: [rows: any[]];
  sortChange: [orderBy: OrderBy[]];
  cellEdit: [rowIndex: number, column: string, oldValue: any, newValue: any];
  pageChange: [page: number, pageSize: number];
}>();

// 当前页码
const currentPage = ref(1);
const pageSize = ref(props.pageSize);

// 动态表格高度
const tableHeight = ref(500)
const gridRootRef = ref<HTMLElement | null>(null)
let ro: ResizeObserver | null = null

const updateTableHeight = () => {
  const el = gridRootRef.value
  if (!el) return
  const paginationH = props.showPagination ? 60 : 0
  const h = el.clientHeight - paginationH
  if (h > 100) tableHeight.value = h
}

onMounted(() => {
  // Use nextTick to ensure the element has been laid out
  nextTick(() => {
    updateTableHeight()
    if (gridRootRef.value) {
      ro = new ResizeObserver(updateTableHeight)
      ro.observe(gridRootRef.value)
    }
  })
})

onUnmounted(() => { ro?.disconnect() })

// 选中的行
const selectedRows = ref<any[]>([]);
const selectedRowIndices = ref<number[]>([]);

// 编辑状态
const editingCell = ref<{ row: number; column: string }>({ row: -1, column: '' });
const editingValue = ref<any>(null);
const cellInput = ref<any>(null);

// JSON 对话框状态
const jsonDialogVisible = ref(false);
const jsonEditContent = ref('');
const jsonEditCell = ref<{ row: number; column: string }>({ row: -1, column: '' });

// 是否使用虚拟滚动（基于数据量）
const useVirtualScroll = computed(() => {
  return props.data.length > props.virtualScrollThreshold;
});

// 将二维数组转换为对象数组以供 el-table 使用
const displayData = computed(() => {
  if (!props.data || !Array.isArray(props.data)) {
    return [];
  }
  return props.data.map((row, rowIndex) => {
    const obj: Record<string, any> = { _rowIndex: rowIndex };
    props.columns.forEach((col, index) => {
      obj[col] = row[index];
    });
    return obj;
  });
});

// 虚拟表格列配置
const virtualColumns = computed(() => {
  const cols: any[] = [];
  
  // 选择列
  cols.push({
    key: 'selection',
    dataKey: 'selection',
    title: '',
    width: 55,
    cellRenderer: ({ rowData, rowIndex }: any) => {
      const isSelected = selectedRowIndices.value.indexOf(rowIndex) >= 0;
      return h(ElCheckbox, {
        modelValue: isSelected,
        'onUpdate:modelValue': (checked: boolean) => handleVirtualRowSelection(rowIndex, checked, rowData),
      });
    },
    headerCellRenderer: () => {
      const allSelected = displayData.value.length > 0 && 
        selectedRowIndices.value.length === displayData.value.length;
      return h(ElCheckbox, {
        modelValue: allSelected,
        'onUpdate:modelValue': (checked: boolean) => handleVirtualSelectAll(checked),
      });
    },
  });
  
  // 数据列
  props.columns.forEach((column) => {
    cols.push({
      key: column,
      dataKey: column,
      title: column,
      width: 150,
      cellRenderer: ({ rowData, rowIndex }: any) => {
        const value = rowData[column];
        const formattedValue = formatCellValue(value);
        
        return h('div', {
          class: 'virtual-cell-content',
          title: formattedValue, // 添加 tooltip
        }, formattedValue);
      },
    });
  });
  
  return cols;
});

// 处理虚拟表格行选择
const handleVirtualRowSelection = (rowIndex: number, checked: boolean, rowData: any) => {
  const index = selectedRowIndices.value.indexOf(rowIndex);
  if (checked && index < 0) {
    selectedRowIndices.value.push(rowIndex);
  } else if (!checked && index >= 0) {
    selectedRowIndices.value.splice(index, 1);
  }
  
  // 更新选中的行数据
  const selected = selectedRowIndices.value.map(idx => displayData.value[idx]);
  selectedRows.value = selected;
  emit('selectionChange', selected);
};

// 处理虚拟表格全选
const handleVirtualSelectAll = (checked: boolean) => {
  if (checked) {
    selectedRowIndices.value = displayData.value.map((_, index) => index);
    selectedRows.value = [...displayData.value];
  } else {
    selectedRowIndices.value = [];
    selectedRows.value = [];
  }
  emit('selectionChange', selectedRows.value);
};

// 处理选择变化
const handleSelectionChange = (rows: any[]) => {
  selectedRows.value = rows;
  emit('selectionChange', rows);
};

// 处理排序变化
const handleSortChange = ({ column, prop, order }: any) => {
  if (!prop || !order) {
    emit('sortChange', []);
    return;
  }

  const orderBy: OrderBy = {
    column: prop,
    direction: order === 'ascending' ? 'ASC' : 'DESC',
  };
  emit('sortChange', [orderBy]);
};

// 检测值是否为 JSON
const isJsonValue = (value: any): boolean => {
  if (value === null || value === undefined) return false;
  const str = typeof value === 'string' ? value.trim() : JSON.stringify(value);
  if (typeof value === 'object') return true;
  return (str.startsWith('{') && str.endsWith('}')) || (str.startsWith('[') && str.endsWith(']'));
};

// 格式化 JSON 字符串
const formatJson = (value: any): string => {
  try {
    const obj = typeof value === 'string' ? JSON.parse(value) : value;
    return JSON.stringify(obj, null, 2);
  } catch {
    return typeof value === 'string' ? value : JSON.stringify(value);
  }
};

// 处理单元格双击（备用方法 - 直接在 cell-content 上监听）
const handleCellContentDoubleClick = (rowIndex: number, columnName: string, value: any) => {
  if (!props.editable) return;

  // JSON 字段弹出格式化对话框
  if (isJsonValue(value)) {
    jsonEditCell.value = { row: rowIndex, column: columnName };
    jsonEditContent.value = formatJson(value);
    jsonDialogVisible.value = true;
    return;
  }

  editingCell.value = { row: rowIndex, column: columnName };
  editingValue.value = value;

  nextTick(() => {
    if (cellInput.value) {
      if (Array.isArray(cellInput.value)) {
        cellInput.value[0]?.focus();
      } else {
        cellInput.value.focus();
      }
    }
  });
};

// 复制 JSON 内容
const copyJsonContent = async () => {
  try {
    await navigator.clipboard.writeText(jsonEditContent.value);
    ElMessage.success('已复制到剪贴板');
  } catch {
    ElMessage.error('复制失败');
  }
};

// 确认 JSON 编辑
const confirmJsonEdit = () => {
  const { row, column } = jsonEditCell.value;
  if (row < 0 || !column) return;
  try {
    // 验证 JSON 格式
    JSON.parse(jsonEditContent.value);
    const oldValue = displayData.value[row][column];
    emit('cellEdit', row, column, oldValue, jsonEditContent.value);
    jsonDialogVisible.value = false;
  } catch {
    ElMessage.error('JSON 格式不正确，请检查后重试');
  }
};

// 处理单元格双击
const handleCellDoubleClick = (row: any, column: any, cell: any, event: any) => {
  if (!props.editable) return;

  const rowIndex = displayData.value.indexOf(row);
  const columnName = column.property;

  if (rowIndex >= 0 && columnName) {
    const value = row[columnName];
    if (isJsonValue(value)) {
      jsonEditCell.value = { row: rowIndex, column: columnName };
      jsonEditContent.value = formatJson(value);
      jsonDialogVisible.value = true;
      return;
    }
    editingCell.value = { row: rowIndex, column: columnName };
    editingValue.value = value;
    nextTick(() => {
      if (cellInput.value) {
        Array.isArray(cellInput.value) ? cellInput.value[0]?.focus() : cellInput.value.focus();
      }
    });
  }
};

// 完成单元格编辑
const handleCellEditComplete = (newValue?: any) => {
  const { row, column } = editingCell.value;
  if (row >= 0 && column) {
    const oldValue = displayData.value[row][column];
    const finalValue = newValue !== undefined ? newValue : editingValue.value;
    
    if (oldValue !== finalValue) {
      emit('cellEdit', row, column, oldValue, finalValue);
    }
  }
  
  editingCell.value = { row: -1, column: '' };
  editingValue.value = null;
};

// 取消单元格编辑
const handleCellEditCancel = () => {
  editingCell.value = { row: -1, column: '' };
  editingValue.value = null;
};

// 格式化单元格值
const formatCellValue = (value: any): string => {
  if (value === null || value === undefined) {
    return 'NULL';
  }
  if (typeof value === 'boolean') {
    return value ? 'true' : 'false';
  }
  if (typeof value === 'object') {
    return JSON.stringify(value);
  }
  return String(value);
};

// 处理页码变化
const handlePageChange = (page: number) => {
  emit('pageChange', page, pageSize.value);
};

// 处理每页大小变化
const handlePageSizeChange = (size: number) => {
  currentPage.value = 1;
  emit('pageChange', 1, size);
};

// 获取选中的行
const getSelectedRows = () => {
  return selectedRows.value;
};

// 清除选择
const clearSelection = () => {
  selectedRows.value = [];
};

// 暴露方法给父组件
defineExpose({
  getSelectedRows,
  clearSelection,
});
</script>

<style scoped>
.data-grid {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.virtual-table-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.virtual-cell-content {
  padding: 8px 12px;
  min-height: 20px;
  cursor: default;
  user-select: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 26px;
}

.cell-content {
  padding: 0 8px;
  height: 40px;
  line-height: 40px;
  cursor: pointer;
  user-select: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #303133;
}

.cell-content:hover {
  background-color: #f5f7fa;
}

/* Reduce paint cost for the scrollable table body */
:deep(.el-table__body-wrapper) {
  contain: layout;
}

:deep(.el-table__body tr) {
  contain: layout style;
}

/* Ensure cell text is always black */
:deep(.el-table td.el-table__cell) {
  color: #303133;
}

.json-dialog-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}

.json-textarea :deep(textarea) {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 13px;
  color: #303133;
  line-height: 1.6;
}
.cell-editor-wrapper {
  width: 100%;
}

.pagination-container {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

:deep(.el-table__cell) {
  padding: 0 !important;
  height: 40px;
  max-height: 40px;
  overflow: hidden;
  font-size: 13px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  color: #303133;
}

:deep(.el-table th.el-table__cell) {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  background-color: #f5f7fa;
}

:deep(.el-table__row) {
  height: 40px;
  max-height: 40px;
}

/* 虚拟表格样式优化 */
:deep(.el-table-v2__header-row) {
  background-color: #f5f7fa;
  font-weight: 600;
}

:deep(.el-table-v2__row) {
  border-bottom: 1px solid #ebeef5;
}

:deep(.el-table-v2__row:hover) {
  background-color: #f5f7fa;
}

:deep(.el-table-v2__cell) {
  padding: 0;
  border-right: 1px solid #ebeef5;
}

:deep(.el-table-v2__header-cell) {
  padding: 0 12px;
  border-right: 1px solid #ebeef5;
}
</style>
