<template>
  <div class="table-list">
    <div class="table-list-header">
      <div class="header-left">
        <h3>数据表列表</h3>
        <span v-if="currentDatabase" class="db-badge">{{ currentDatabase }}</span>
      </div>
      <div class="header-actions">
        <el-button
          v-if="selectedTables.length > 0"
          type="success"
          size="small"
          :loading="exporting"
          @click="handleExportSelected"
        >
          <el-icon><Download /></el-icon>
          导出结构 ({{ selectedTables.length }})
        </el-button>
        <el-button 
          :icon="Refresh" 
          circle 
          size="small" 
          @click="refreshTables"
          :loading="loading"
          title="刷新"
        />
      </div>
    </div>

    <div class="table-list-filter">
      <el-input
        v-model="filterText"
        placeholder="输入表名进行筛选..."
        clearable
        :prefix-icon="Search"
      />
      <span class="filter-count">
        {{ filteredTables.length !== tables.length
          ? `显示 ${filteredTables.length} / ${tables.length} 个表`
          : `共 ${tables.length} 个表`
        }}
      </span>
    </div>

    <div class="table-list-content">
      <!-- 骨架屏，避免白屏卡顿 -->
      <div v-if="loading" class="skeleton-wrap">
        <el-skeleton :rows="8" animated />
      </div>

      <el-table
        v-else-if="filteredTables.length > 0"
        :data="filteredTables"
        stripe
        style="width: 100%"
        class="data-table"
        :max-height="tableMaxHeight"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="45" />
        <el-table-column prop="name" label="表名" min-width="150" show-overflow-tooltip align="left" header-align="left" />
        <el-table-column prop="engine" label="引擎" width="90" align="left" header-align="left" />
        <el-table-column prop="collation" label="字符集" width="140" show-overflow-tooltip align="left" header-align="left" />
        <el-table-column label="行数" width="100" align="left" header-align="left">
          <template #default="{ row }">
            {{ formatNumber(row.rows) }}
          </template>
        </el-table-column>
        <el-table-column label="大小" width="90" align="left" header-align="left">
          <template #default="{ row }">
            {{ formatSize((row.dataLength || 0) + (row.indexLength || 0)) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" align="left" header-align="left" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleViewData(row)">查看数据</el-button>
            <el-button type="info" size="small" @click="handleViewSchema(row)">查看结构</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-else-if="!currentDatabase" description="请从左侧选择数据库" />
      <el-empty v-else-if="tables.length === 0" description="该数据库没有数据表" />
      <el-empty v-else description="没有匹配的数据表" />
    </div>

    <!-- 导出结构预览对话框 -->
    <el-dialog
      v-model="exportDialogVisible"
      title="导出表结构"
      width="min(860px, 92vw)"
      :close-on-click-modal="false"
    >
      <div class="export-toolbar">
        <el-button size="small" :icon="CopyDocument" @click="copyDDL">复制</el-button>
        <el-button size="small" :icon="Download" @click="downloadDDL">下载 .sql</el-button>
      </div>
      <el-input
        v-model="exportContent"
        type="textarea"
        :rows="20"
        readonly
        class="export-textarea"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { ElMessage } from 'element-plus';
import { Refresh, Search, Download, CopyDocument } from '@element-plus/icons-vue';
import { useConnectionStore } from '../stores/connection';
import { useDatabaseStore } from '../stores/database';
import { DatabaseAPI, SchemaAPI } from '../api';
import type { Table } from '../types/api';

const emit = defineEmits<{
  viewSchema: [table: string];
  viewData: [table: string];
}>();

const connectionStore = useConnectionStore();
const databaseStore = useDatabaseStore();

const loading = ref(false);
const exporting = ref(false);
const tables = ref<Table[]>([]);
const filterText = ref('');
const selectedTables = ref<Table[]>([]);
const exportDialogVisible = ref(false);
const exportContent = ref('');

// ── Dynamic table height ───────────────────────────────────────────────────────
const contentRef = ref<HTMLElement | null>(null)
const tableMaxHeight = ref(400)

const updateHeight = () => {
  // Walk up to find the .table-list-content element
  const el = document.querySelector('.table-list-content') as HTMLElement | null
  if (el) tableMaxHeight.value = el.clientHeight || 400
}

let ro: ResizeObserver | null = null

const currentDatabase = computed(() => databaseStore.currentDatabase);

const filteredTables = computed(() => {
  if (!filterText.value) return tables.value;
  const q = filterText.value.toLowerCase();
  return tables.value.filter(t => t.name.toLowerCase().includes(q));
});

const formatNumber = (num: number): string => {
  if (!num) return '0';
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toString();
};

const formatSize = (bytes: number): string => {
  if (!bytes) return '0 B';
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(1)} GB`;
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(1)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${bytes} B`;
};

// 加载表列表 - 用 requestAnimationFrame 避免阻塞渲染
const loadTables = async () => {
  if (!connectionStore.currentConnection || !currentDatabase.value) {
    tables.value = [];
    return;
  }

  loading.value = true;
  // 先清空，让骨架屏立即显示，避免旧数据残留造成卡顿感
  tables.value = [];

  try {
    const result = await DatabaseAPI.listTables(
      connectionStore.currentConnection.id,
      currentDatabase.value
    );
    // 用 rAF 延迟赋值，让 loading 骨架屏先渲染完毕再填充数据
    requestAnimationFrame(() => {
      tables.value = result || [];
      databaseStore.setTables(result || []);
      loading.value = false;
    });
  } catch (error: any) {
    ElMessage.error(error.message || '加载表列表失败');
    tables.value = [];
    loading.value = false;
  }
};

const refreshTables = async () => {
  await loadTables();
  ElMessage.success('表列表已刷新');
};

const handleSelectionChange = (rows: Table[]) => {
  selectedTables.value = rows;
};

const handleViewData = (table: Table) => {
  databaseStore.setCurrentTable(table.name);
  emit('viewData', table.name);
};

const handleViewSchema = (table: Table) => {
  databaseStore.setCurrentTable(table.name);
  emit('viewSchema', table.name);
};

// 导出选中表的 DDL
const handleExportSelected = async () => {
  if (!connectionStore.currentConnection || !currentDatabase.value) return;
  if (selectedTables.value.length === 0) return;

  exporting.value = true;
  try {
    const ddlParts: string[] = [];
    for (const table of selectedTables.value) {
      const ddl = await SchemaAPI.getCreateTableDDL(
        connectionStore.currentConnection.id,
        currentDatabase.value,
        table.name
      );
      ddlParts.push(`-- Table: ${table.name}\n${ddl};\n`);
    }
    exportContent.value = ddlParts.join('\n');
    exportDialogVisible.value = true;
  } catch (error: any) {
    ElMessage.error(error.message || '导出表结构失败');
  } finally {
    exporting.value = false;
  }
};

const copyDDL = async () => {
  try {
    await navigator.clipboard.writeText(exportContent.value);
    ElMessage.success('已复制到剪贴板');
  } catch {
    ElMessage.error('复制失败，请手动选择复制');
  }
};

const downloadDDL = () => {
  const blob = new Blob([exportContent.value], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${currentDatabase.value}_schema_${Date.now()}.sql`;
  a.click();
  URL.revokeObjectURL(url);
};

watch(currentDatabase, async (newDb) => {
  filterText.value = '';
  selectedTables.value = [];
  if (newDb) {
    await loadTables();
  } else {
    tables.value = [];
  }
});

onMounted(async () => {
  if (currentDatabase.value) {
    await loadTables();
  }
  // Observe content area for size changes
  const el = document.querySelector('.table-list-content') as HTMLElement | null
  if (el) {
    updateHeight()
    ro = new ResizeObserver(updateHeight)
    ro.observe(el)
  }
});

onUnmounted(() => {
  ro?.disconnect()
});
</script>

<style scoped>
.table-list {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 4px;
  overflow: hidden;
}

.table-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-left h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 500;
  color: #303133;
}

.db-badge {
  font-size: 12px;
  color: #409eff;
  background: #ecf5ff;
  padding: 2px 8px;
  border-radius: 10px;
  border: 1px solid #b3d8ff;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.table-list-filter {
  padding: 10px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  border-bottom: 1px solid #f0f0f0;
}

.filter-count {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}

.table-list-content {
  flex: 1;
  overflow: hidden;
  padding: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.skeleton-wrap {
  padding: 16px;
}

:deep(.el-table) {
  font-size: 13px;
}

:deep(.el-table .el-table__header-wrapper) {
  flex-shrink: 0;
}

:deep(.el-table th.el-table__cell) {
  background-color: #fafafa;
  font-weight: 600;
  padding: 8px 0;
  position: sticky;
  top: 0;
  z-index: 2;
}

:deep(.el-table td.el-table__cell) {
  padding: 6px 0;
}

.export-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.export-textarea :deep(textarea) {
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.6;
}
</style>
