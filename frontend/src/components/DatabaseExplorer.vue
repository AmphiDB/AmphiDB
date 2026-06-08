<template>
  <div class="database-explorer">
    <div class="explorer-header">
      <h3>数据库浏览器</h3>
      <el-button
        :icon="Refresh"
        circle
        size="small"
        @click="refreshDatabases"
        :loading="loading"
        title="刷新"
      />
    </div>

    <div class="explorer-content">
      <el-empty v-if="!loading && !connectionStore.isConnected" description="请先连接到数据库" />
      <el-empty v-else-if="!loading && databases.length === 0" description="未找到数据库" />

      <div v-else class="db-list">
        <div v-for="db in databases" :key="db.name" class="db-item">
          <!-- 数据库行 -->
          <div
            class="db-row"
            :class="{ 'is-active': currentDatabase === db.name }"
            @click="handleDbClick(db.name)"
            @contextmenu.prevent="handleContextMenu($event, { type: 'database', database: db.name })"
          >
            <el-icon class="arrow-icon" :class="{ expanded: expandedDbs.has(db.name) }">
              <ArrowRight />
            </el-icon>
            <el-icon class="node-icon"><Coin /></el-icon>
            <span class="node-label">{{ db.name }}</span>
            <el-icon v-if="loadingDbs.has(db.name)" class="loading-icon is-loading"><Loading /></el-icon>
          </div>

          <!-- 展开内容：筛选框 + 表列表 -->
          <div v-if="expandedDbs.has(db.name)" class="db-children">
            <!-- 筛选框 -->
            <div class="table-filter" @click.stop>
              <el-input
                v-model="dbFilterMap[db.name]"
                placeholder="筛选表名..."
                clearable
                size="small"
                :prefix-icon="Search"
              />
            </div>

            <!-- 表节点 -->
            <div
              v-for="table in getFilteredTables(db.name)"
              :key="table.name"
              class="table-row"
              :class="{ 'is-active': currentDatabase === db.name && currentTable === table.name }"
              @click="handleTableClick(db.name, table.name)"
              @dblclick="handleTableDblClick(db.name, table.name)"
              @contextmenu.prevent="handleContextMenu($event, { type: 'table', database: db.name, table: table.name })"
            >
              <el-icon class="node-icon table-icon"><Document /></el-icon>
              <span class="node-label">{{ table.name }}</span>
              <span v-if="table.rows !== undefined" class="node-meta">({{ formatRowCount(table.rows) }})</span>
            </div>

            <!-- 无匹配提示 -->
            <div
              v-if="dbFilterMap[db.name] && getFilteredTables(db.name).length === 0 && !loadingDbs.has(db.name)"
              class="no-match"
            >
              无匹配表
            </div>

            <!-- 空表提示 -->
            <div
              v-if="!loadingDbs.has(db.name) && getTablesForDb(db.name).length === 0 && !dbFilterMap[db.name]"
              class="no-match"
            >
              暂无数据表
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右键菜单 -->
    <teleport to="body">
      <div
        v-if="contextMenuVisible"
        class="context-menu"
        :style="{ left: contextMenuPosition.x + 'px', top: contextMenuPosition.y + 'px' }"
        @click="handleMenuClick"
      >
        <template v-if="contextMenuData?.type === 'database'">
          <div class="context-menu-item" @click="handleMenuCommand('create-table')">
            <el-icon><Plus /></el-icon>
            <span>新建表</span>
          </div>
          <div class="context-menu-divider" />
          <div class="context-menu-item" @click="handleMenuCommand('refresh-tables')">
            <el-icon><Refresh /></el-icon>
            <span>刷新表列表</span>
          </div>
        </template>
        <template v-else-if="contextMenuData?.type === 'table'">
          <div class="context-menu-item" @click="handleMenuCommand('view-data')">
            <el-icon><Grid /></el-icon>
            <span>查看表数据</span>
          </div>
          <div class="context-menu-item" @click="handleMenuCommand('copy-select')">
            <el-icon><Tickets /></el-icon>
            <span>复制 SELECT</span>
          </div>
          <div class="context-menu-item" @click="handleMenuCommand('view-schema')">
            <el-icon><Document /></el-icon>
            <span>查看表结构</span>
          </div>
          <div class="context-menu-item" @click="handleMenuCommand('edit-schema')">
            <el-icon><Edit /></el-icon>
            <span>修改表结构</span>
          </div>
          <div class="context-menu-divider" />
          <div class="context-menu-item" @click="handleMenuCommand('copy-table-name')">
            <el-icon><CopyDocument /></el-icon>
            <span>复制表名</span>
          </div>
          <div class="context-menu-item" @click="handleMenuCommand('copy-full-name')">
            <el-icon><CopyDocument /></el-icon>
            <span>复制完整名称</span>
          </div>
          <div class="context-menu-divider" />
          <div class="context-menu-item danger" @click="handleMenuCommand('drop-table')">
            <el-icon><Delete /></el-icon>
            <span>删除表</span>
          </div>
        </template>
      </div>
    </teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, watch, onMounted, onUnmounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import {
  Refresh,
  Coin,
  Document,
  Grid,
  Delete,
  Edit,
  Loading,
  Plus,
  Search,
  ArrowRight,
  CopyDocument,
  Tickets,
} from '@element-plus/icons-vue';
import { useConnectionStore } from '../stores/connection';
import { useDatabaseStore } from '../stores/database';
import { DatabaseAPI, SchemaAPI } from '../api';
import type { Database, Table } from '../types/api';

const emit = defineEmits<{
  viewSchema: [];
  viewData: [];
  editSchema: [];
}>();

const connectionStore = useConnectionStore();
const databaseStore = useDatabaseStore();

const loading = ref(false);
const databases = ref<Database[]>([]);
// 已展开的数据库名集合
const expandedDbs = reactive(new Set<string>());
// 正在加载中的数据库
const loadingDbs = reactive(new Set<string>());
// 每个数据库的表列表
const dbTablesMap = reactive<Record<string, Table[]>>({});
// 每个数据库的筛选文本
const dbFilterMap = reactive<Record<string, string>>({});

const contextMenuVisible = ref(false);
const contextMenuPosition = ref({ x: 0, y: 0 });
const contextMenuData = ref<any>(null);

// 缓存
const tableCache = new Map<string, { data: Table[]; timestamp: number }>();
const CACHE_DURATION = 5 * 60 * 1000;

const currentDatabase = computed(() => databaseStore.currentDatabase);
const currentTable = computed(() => databaseStore.currentTable);

const isCacheValid = (ts: number) => Date.now() - ts < CACHE_DURATION;

const tableCacheKey = (dbName: string) => {
  return `${connectionStore.currentConnection?.id || 'no-connection'}:${dbName}`;
};

const formatRowCount = (count: number): string => {
  if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`;
  if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
  return count.toString();
};

const getTablesForDb = (dbName: string): Table[] => dbTablesMap[dbName] || [];

const getFilteredTables = (dbName: string): Table[] => {
  const tables = getTablesForDb(dbName);
  const filter = (dbFilterMap[dbName] || '').toLowerCase();
  if (!filter) return tables;
  return tables.filter(t => t.name.toLowerCase().includes(filter));
};

// 加载数据库列表
const loadDatabases = async () => {
  if (!connectionStore.currentConnection) {
    databases.value = [];
    return;
  }
  loading.value = true;
  try {
    const result = await DatabaseAPI.listDatabases(connectionStore.currentConnection.id);
    databases.value = result || [];
    databaseStore.setDatabases(result || []);
  } catch (error: any) {
    ElMessage.error(error.message || '加载数据库列表失败');
  } finally {
    loading.value = false;
  }
};

// 加载某个数据库的表列表
const loadTables = async (dbName: string) => {
  if (!connectionStore.currentConnection) return;

  const cacheKey = tableCacheKey(dbName);
  const cached = tableCache.get(cacheKey);
  if (cached && isCacheValid(cached.timestamp)) {
    dbTablesMap[dbName] = cached.data;
    return;
  }

  loadingDbs.add(dbName);
  try {
    const tables = await DatabaseAPI.listTables(connectionStore.currentConnection.id, dbName);
    tableCache.set(cacheKey, { data: tables, timestamp: Date.now() });
    dbTablesMap[dbName] = tables;
    if (dbName === databaseStore.currentDatabase) {
      databaseStore.setTables(tables);
    }
  } catch (error: any) {
    ElMessage.error(error.message || '加载表列表失败');
  } finally {
    loadingDbs.delete(dbName);
  }
};

// 点击数据库行：切换展开/折叠
const handleDbClick = async (dbName: string) => {
  databaseStore.setCurrentDatabase(dbName);
  databaseStore.setCurrentTable(null);

  if (expandedDbs.has(dbName)) {
    expandedDbs.delete(dbName);
  } else {
    expandedDbs.add(dbName);
    await loadTables(dbName);
  }
};

// 单击表
const handleTableClick = (dbName: string, tableName: string) => {
  databaseStore.setCurrentDatabase(dbName);
  databaseStore.setCurrentTable(tableName);
};

// 双击表 - 打开数据
const handleTableDblClick = (dbName: string, tableName: string) => {
  databaseStore.setCurrentDatabase(dbName);
  databaseStore.setCurrentTable(tableName);
  emit('viewData');
};

// 右键菜单
const handleContextMenu = (event: MouseEvent, data: any) => {
  contextMenuData.value = data;
  contextMenuPosition.value = { x: event.clientX, y: event.clientY };
  contextMenuVisible.value = true;
  const close = () => {
    contextMenuVisible.value = false;
    document.removeEventListener('click', close);
  };
  setTimeout(() => document.addEventListener('click', close), 100);
};

const handleMenuClick = (e: Event) => e.stopPropagation();

const copyText = async (text: string, successMessage: string) => {
  try {
    await navigator.clipboard.writeText(text);
    ElMessage.success(successMessage);
  } catch (error: any) {
    ElMessage.error(error?.message || '复制失败');
  }
};

const handleMenuCommand = async (command: string) => {
  const data = contextMenuData.value;
  contextMenuVisible.value = false;
  contextMenuData.value = null;
  if (!data) return;

  switch (command) {
    case 'create-table':
      databaseStore.setCurrentDatabase(data.database);
      databaseStore.setCurrentTable(null);
      emit('editSchema');
      break;

    case 'refresh-tables':
      tableCache.delete(tableCacheKey(data.database));
      dbFilterMap[data.database] = '';
      if (expandedDbs.has(data.database)) {
        await loadTables(data.database);
      }
      ElMessage.success('表列表已刷新');
      break;

    case 'view-schema':
      databaseStore.setCurrentDatabase(data.database);
      databaseStore.setCurrentTable(data.table);
      emit('viewSchema');
      break;

    case 'edit-schema':
      databaseStore.setCurrentDatabase(data.database);
      databaseStore.setCurrentTable(data.table);
      emit('editSchema');
      break;

    case 'view-data':
      databaseStore.setCurrentDatabase(data.database);
      databaseStore.setCurrentTable(data.table);
      emit('viewData');
      break;

    case 'copy-table-name':
      await copyText(data.table, '表名已复制');
      break;

    case 'copy-full-name':
      await copyText(`\`${data.database}\`.\`${data.table}\``, '完整表名已复制');
      break;

    case 'copy-select':
      await copyText(`SELECT * FROM \`${data.database}\`.\`${data.table}\` LIMIT 100;`, 'SELECT 语句已复制');
      break;

    case 'drop-table':
      await handleDropTable(data);
      break;
  }
};

const handleDropTable = async (data: any) => {
  if (!connectionStore.currentConnection) return;
  try {
    await ElMessageBox.confirm(
      `确定要删除表 "${data.database}.${data.table}" 吗？此操作不可恢复！`,
      '删除表',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    );
    await SchemaAPI.dropTable(connectionStore.currentConnection.id, data.database, data.table);
    ElMessage.success('表已删除');
    tableCache.delete(tableCacheKey(data.database));
    await loadTables(data.database);
    if (databaseStore.currentTable === data.table) {
      databaseStore.setCurrentTable(null);
    }
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error(error.message || '删除表失败');
  }
};

const refreshDatabases = async () => {
  tableCache.clear();
  expandedDbs.clear();
  await loadDatabases();
  ElMessage.success('数据库列表已刷新');
};

watch(
  () => connectionStore.currentConnection,
  async (conn) => {
    tableCache.clear();
    expandedDbs.clear();
    if (conn) {
      await loadDatabases();
    } else {
      databases.value = [];
      databaseStore.setCurrentDatabase(null);
      databaseStore.setCurrentTable(null);
    }
  }
);

onMounted(async () => {
  if (connectionStore.currentConnection) await loadDatabases();
});

onUnmounted(() => tableCache.clear());
</script>

<style scoped>
.database-explorer {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
  overflow: hidden;
}

.explorer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.explorer-header h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 500;
  color: #303133;
}

.explorer-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

/* 数据库行 */
.db-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  height: 32px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}

.db-row:hover {
  background: #f5f7fa;
}

.db-row.is-active {
  background: #ecf5ff;
}

.arrow-icon {
  font-size: 12px;
  color: #909399;
  transition: transform 0.2s;
  flex-shrink: 0;
}

.arrow-icon.expanded {
  transform: rotate(90deg);
}

.node-icon {
  font-size: 15px;
  color: #606266;
  flex-shrink: 0;
}

.loading-icon {
  font-size: 14px;
  color: #409eff;
  margin-left: 4px;
}

.node-label {
  font-size: 13px;
  color: #303133;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}

.node-meta {
  font-size: 11px;
  color: #c0c4cc;
  flex-shrink: 0;
}

/* 展开区域 */
.db-children {
  padding-left: 0;
}

/* 筛选框 */
.table-filter {
  padding: 4px 8px 4px 28px;
}

/* 表行 */
.table-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px 0 28px;
  height: 28px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}

.table-row:hover {
  background: #f5f7fa;
}

.table-row.is-active {
  background: #ecf5ff;
}

.table-row.is-active .node-label {
  color: #409eff;
  font-weight: 500;
}

.table-icon {
  font-size: 13px;
  color: #909399;
}

.no-match {
  padding: 6px 8px 6px 36px;
  font-size: 12px;
  color: #c0c4cc;
}

/* 右键菜单 */
.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  padding: 4px 0;
  min-width: 160px;
  z-index: 9999;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 13px;
  color: #606266;
  cursor: pointer;
  transition: background 0.15s;
}

.context-menu-item:hover {
  background: #f5f7fa;
}

.context-menu-item.danger { color: #f56c6c; }
.context-menu-item.danger:hover { background: #fef0f0; }

.context-menu-divider {
  height: 1px;
  background: #e4e7ed;
  margin: 4px 0;
}
</style>
