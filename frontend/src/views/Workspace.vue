<template>
  <div class="workspace">
    <!-- 左侧数据库浏览器 -->
    <div class="workspace-sidebar">
      <DatabaseExplorer
        @view-schema="handleViewSchema"
        @view-data="handleViewData"
        @edit-schema="handleEditSchema"
      />
    </div>

    <!-- 右侧内容区域 -->
    <div class="workspace-content">
      <!-- 数据表列表（当选中数据库但未打开对象时显示） -->
      <div v-if="tabs.length === 0 && contentType === 'table-list'" class="table-list-panel">
        <TableList
          @view-schema="handleViewSchema"
          @view-data="handleViewData"
        />
      </div>

      <!-- 欢迎页面 -->
      <div v-else-if="tabs.length === 0 && contentType === 'welcome'" class="welcome-content">
        <el-empty description="请从左侧选择数据库和表">
          <template #image>
            <el-icon :size="100" color="#909399">
              <FolderOpened />
            </el-icon>
          </template>
          <div class="welcome-tips">
            <p><strong>提示：</strong></p>
            <ul>
              <li>单击数据库名自动展开表列表</li>
              <li>双击表名查看表数据</li>
              <li>右键点击表名选择"修改表结构"</li>
              <li>右键点击表名选择"查看表结构"</li>
            </ul>
          </div>
        </el-empty>
      </div>

      <!-- 多标签对象工作台 -->
      <el-tabs
        v-else
        v-model="activeTabId"
        type="card"
        closable
        class="workspace-tabs"
        @tab-remove="handleTabRemove"
        @tab-change="handleTabChange"
      >
        <el-tab-pane
          v-for="tab in tabs"
          :key="tab.id"
          :name="tab.id"
        >
          <template #label>
            <span class="tab-label" :title="tabTitle(tab)">
              <el-icon>
                <Grid v-if="tab.type === 'data'" />
                <Edit v-else-if="tab.type === 'schema-edit'" />
                <Document v-else />
              </el-icon>
              {{ tabLabel(tab) }}
            </span>
          </template>

          <div class="content-panel">
            <div class="content-header">
              <h3>
                <el-icon>
                  <Grid v-if="tab.type === 'data'" />
                  <Edit v-else-if="tab.type === 'schema-edit'" />
                  <Document v-else />
                </el-icon>
                {{ tabTitle(tab) }}
              </h3>
              <div class="content-actions">
                <el-button
                  v-if="tab.table && tab.type !== 'schema-view'"
                  size="small"
                  @click="openTab('schema-view', tab.database, tab.table)"
                >
                  <el-icon><Document /></el-icon>
                  查看结构
                </el-button>
                <el-button
                  v-if="tab.table && tab.type !== 'schema-edit'"
                  size="small"
                  @click="openTab('schema-edit', tab.database, tab.table)"
                >
                  <el-icon><Edit /></el-icon>
                  修改结构
                </el-button>
                <el-button
                  v-if="tab.table && tab.type !== 'data'"
                  size="small"
                  @click="openTab('data', tab.database, tab.table)"
                >
                  <el-icon><Grid /></el-icon>
                  查看数据
                </el-button>
                <el-button size="small" @click="handleTabRemove(tab.id)">
                  <el-icon><Close /></el-icon>
                  关闭
                </el-button>
              </div>
            </div>
            <div class="content-body">
              <SchemaViewer
                v-if="tab.type === 'schema-view' && currentConnection && tab.table"
                :profile-id="currentConnection.id"
                :database="tab.database"
                :table="tab.table"
              />
              <TableEditor
                v-else-if="tab.type === 'schema-edit' && currentConnection"
                :profile-id="currentConnection.id"
                :database="tab.database"
                :table="tab.table || ''"
                :mode="tab.table ? 'edit' : 'create'"
                @success="handleSchemaUpdateSuccess(tab)"
              />
              <DataManager
                v-else-if="tab.type === 'data' && currentConnection && tab.table"
                :profile-id="currentConnection.id"
                :database="tab.database"
                :table="tab.table"
              />
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { useConnectionStore } from '../stores/connection';
import { useDatabaseStore } from '../stores/database';
import DatabaseExplorer from '../components/DatabaseExplorer.vue';
import SchemaViewer from '../components/SchemaViewer.vue';
import TableEditor from '../components/TableEditor.vue';
import DataManager from '../components/DataManager.vue';
import TableList from '../components/TableList.vue';
import { filterTabsForDatabase } from './workspaceTabs';
import {
  FolderOpened,
  Document,
  Edit,
  Grid,
  Close,
} from '@element-plus/icons-vue';

const connectionStore = useConnectionStore();
const databaseStore = useDatabaseStore();

const currentConnection = computed(() => connectionStore.currentConnection);
const currentDatabase = computed(() => databaseStore.currentDatabase);
const currentTable = computed(() => databaseStore.currentTable);

type WorkspaceTabType = 'schema-view' | 'schema-edit' | 'data';

interface WorkspaceTab {
  id: string;
  type: WorkspaceTabType;
  database: string;
  table: string;
}

// 内容类型: 'welcome' | 'table-list'
const contentType = ref<string>('welcome');
const tabs = ref<WorkspaceTab[]>([]);
const activeTabId = ref('');

const makeTabId = (type: WorkspaceTabType, database: string, table: string) => {
  return `${type}:${database}.${table || '__new_table__'}`;
};

const tabActionText = (type: WorkspaceTabType) => {
  if (type === 'data') return '表数据';
  if (type === 'schema-edit') return '设计表';
  return '表结构';
};

const tabLabel = (tab: WorkspaceTab) => {
  const objectName = tab.table || '新建表';
  return `${objectName} · ${tabActionText(tab.type)}`;
};

const tabTitle = (tab: WorkspaceTab) => {
  const objectName = tab.table ? `${tab.database}.${tab.table}` : tab.database;
  return `${tabActionText(tab.type)} - ${objectName}`;
};

const setSelectionFromTab = (tab: WorkspaceTab) => {
  databaseStore.setCurrentDatabase(tab.database);
  databaseStore.setCurrentTable(tab.table || null);
};

const activateTab = (tab: WorkspaceTab) => {
  activeTabId.value = tab.id;
  setSelectionFromTab(tab);
};

const openTab = (type: WorkspaceTabType, database?: string | null, table?: string | null) => {
  if (!database) return;
  if (type !== 'schema-edit' && !table) return;

  const safeTable = table || '';
  const id = makeTabId(type, database, safeTable);
  let tab = tabs.value.find(item => item.id === id);
  if (!tab) {
    tab = { id, type, database, table: safeTable };
    tabs.value.push(tab);
  }
  activateTab(tab);
};

// 处理查看表结构
const handleViewSchema = () => {
  openTab('schema-view', currentDatabase.value, currentTable.value);
};

// 处理编辑表结构
const handleEditSchema = () => {
  // 支持新建表（currentTable 为 null）和编辑表（currentTable 有值）
  openTab('schema-edit', currentDatabase.value, currentTable.value);
};

// 处理查看表数据
const handleViewData = () => {
  openTab('data', currentDatabase.value, currentTable.value);
};

// 处理关闭
const handleClose = () => {
  // 关闭后显示表列表（如果有选中的数据库）
  if (currentDatabase.value) {
    contentType.value = 'table-list';
  } else {
    contentType.value = 'welcome';
  }
  databaseStore.setCurrentTable(null);
};

// 处理表结构更新成功
const handleSchemaUpdateSuccess = (tab: WorkspaceTab) => {
  // 新建表成功后表名可能由编辑器内部确定，保持当前标签并刷新列表即可。
  if (tab.table) {
    openTab('schema-view', tab.database, tab.table);
  }
};

const handleTabChange = (name: string | number) => {
  const tab = tabs.value.find(item => item.id === String(name));
  if (tab) setSelectionFromTab(tab);
};

const handleTabRemove = (name: string | number) => {
  const id = String(name);
  const index = tabs.value.findIndex(tab => tab.id === id);
  if (index < 0) return;

  const wasActive = activeTabId.value === id;
  tabs.value.splice(index, 1);

  if (tabs.value.length === 0) {
    activeTabId.value = '';
    handleClose();
    return;
  }

  if (wasActive) {
    const nextTab = tabs.value[Math.min(index, tabs.value.length - 1)];
    activateTab(nextTab);
  }
};

const closeTabsOutsideDatabase = (database: string | null) => {
  const nextTabs = filterTabsForDatabase(tabs.value, database);
  if (nextTabs.length === tabs.value.length) return;

  tabs.value = nextTabs;
  if (!activeTabId.value || !tabs.value.some(tab => tab.id === activeTabId.value)) {
    activeTabId.value = '';
  }
  if (tabs.value.length === 0) {
    contentType.value = database ? 'table-list' : 'welcome';
  }
};

// 监听当前数据库变化
watch(currentDatabase, (newDb) => {
  closeTabsOutsideDatabase(newDb);
  if (currentTable.value && !tabs.value.some(tab => tab.database === newDb && tab.table === currentTable.value)) {
    databaseStore.setCurrentTable(null);
  }
  if (tabs.value.length > 0) return;
  if (newDb && !currentTable.value) {
    // 选中数据库但没有选中表时，显示表列表
    contentType.value = 'table-list';
  } else if (!newDb) {
    contentType.value = 'welcome';
  }
});

// 监听当前表变化
watch(currentTable, (newTable) => {
  if (tabs.value.length > 0) return;
  if (!newTable && currentDatabase.value) {
    // 取消选中表时，显示表列表
    contentType.value = 'table-list';
  } else if (!newTable && !currentDatabase.value) {
    contentType.value = 'welcome';
  }
});
</script>

<style scoped>
.workspace {
  display: flex;
  height: 100%;
  gap: 0;
  background-color: #f5f7fa;
}

.workspace-sidebar {
  width: 280px;
  min-width: 280px;
  background-color: #fff;
  border-right: 1px solid #e4e7ed;
  overflow: hidden;
}

.workspace-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.workspace-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.workspace-tabs :deep(.el-tabs__header) {
  margin: 0;
  background: #f5f7fa;
}

.workspace-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
}

.workspace-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.welcome-content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  background-color: #fff;
  margin: 16px;
  border-radius: 4px;
}

.table-list-panel {
  height: 100%;
  background-color: #fff;
  margin: 16px;
  border-radius: 4px;
  overflow: hidden;
}

.welcome-tips {
  margin-top: 20px;
  text-align: left;
}

.welcome-tips p {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: #606266;
}

.welcome-tips ul {
  margin: 0;
  padding-left: 20px;
  list-style: disc;
}

.welcome-tips li {
  margin: 8px 0;
  font-size: 13px;
  color: #909399;
}

.content-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: #fff;
  border-radius: 4px;
  overflow: hidden;
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #e4e7ed;
  background-color: #fafafa;
}

.content-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: #303133;
  display: flex;
  align-items: center;
  gap: 8px;
}

.content-actions {
  display: flex;
  gap: 8px;
}

.content-body {
  flex: 1;
  overflow: auto;
  padding: 0;
}

/* 移除 DataManager 内部的 padding，因为外层已经有了 */
.content-body :deep(.data-manager) {
  padding: 0;
  height: 100%;
}
</style>
