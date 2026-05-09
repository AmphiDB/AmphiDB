<template>
  <div class="mongo-workspace">
    <!-- No active connection -->
    <div v-if="!connStore.currentProfileId" class="no-connection">
      <el-empty description="请先连接到 MongoDB">
        <el-button type="primary" @click="router.push('/mongo/connections')">
          前往连接管理
        </el-button>
      </el-empty>
    </div>

    <template v-else>
      <!-- Top bar -->
      <div class="workspace-topbar">
        <div class="topbar-left">
          <el-icon class="topbar-icon"><Connection /></el-icon>
          <span class="connection-name">{{ connStore.currentProfile?.name }}</span>
          <template v-if="dbStore.currentDatabase">
            <span class="topbar-sep">/</span>
            <span class="db-name">{{ dbStore.currentDatabase }}</span>
          </template>
          <template v-if="dbStore.currentCollection">
            <span class="topbar-sep">/</span>
            <span class="coll-name">{{ dbStore.currentCollection }}</span>
          </template>
        </div>
      </div>

      <!-- Main layout -->
      <div class="workspace-body">
        <!-- Left sidebar -->
        <div class="workspace-sidebar">
          <MongoExplorer
            :profile-id="connStore.currentProfileId"
            @select-collection="handleSelectCollection"
            @view-indexes="handleViewIndexes"
          />
        </div>

        <!-- Right content -->
        <div class="workspace-content">
          <!-- Empty state -->
          <div v-if="!dbStore.currentCollection" class="empty-state">
            <el-empty description="请在左侧选择一个集合" />
          </div>

          <!-- Collection tabs -->
          <el-tabs
            v-else
            v-model="activeTab"
            class="collection-tabs"
            type="border-card"
          >
            <el-tab-pane label="文档" name="documents">
              <MongoDocumentList
                :profile-id="connStore.currentProfileId"
                :db-name="dbStore.currentDatabase!"
                :coll-name="dbStore.currentCollection"
                @export="showExportDialog"
                @insert-document="openInsertEditor"
                @edit-document="openEditEditor"
              />
            </el-tab-pane>

            <el-tab-pane label="索引" name="indexes">
              <MongoIndexManager
                :profile-id="connStore.currentProfileId"
                :db-name="dbStore.currentDatabase!"
                :coll-name="dbStore.currentCollection"
              />
            </el-tab-pane>

            <el-tab-pane label="Schema 分析" name="schema">
              <MongoSchemaViewer
                :profile-id="connStore.currentProfileId"
                :db-name="dbStore.currentDatabase!"
                :coll-name="dbStore.currentCollection"
              />
            </el-tab-pane>
          </el-tabs>
        </div>
      </div>
    </template>

    <!-- Document editor dialog (insert / edit) -->
    <MongoDocumentEditor
      v-if="connStore.currentProfileId && dbStore.currentDatabase && dbStore.currentCollection"
      v-model="editorVisible"
      :profile-id="connStore.currentProfileId"
      :db-name="dbStore.currentDatabase"
      :coll-name="dbStore.currentCollection"
      :mode="editorMode"
      :initial-doc="editorInitialDoc"
      @saved="onDocumentSaved"
    />

    <!-- Export dialog -->
    <el-dialog
      v-model="exportDialogVisible"
      title="导出文档"
      width="360px"
      :close-on-click-modal="false"
    >
      <el-form label-width="80px">
        <el-form-item label="导出格式">
          <el-radio-group v-model="exportFormat">
            <el-radio value="json">JSON</el-radio>
            <el-radio value="csv">CSV</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="exportDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="exporting" @click="confirmExport">导出</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Connection } from '@element-plus/icons-vue'
import { useMongoConnectionStore } from '../../stores/mongoConnection'
import { useMongoDatabaseStore } from '../../stores/mongoDatabase'
import { MongoDocumentAPI } from '../../api/mongo'
import MongoExplorer from '../../components/mongo/MongoExplorer.vue'
import MongoDocumentList from '../../components/mongo/MongoDocumentList.vue'
import MongoDocumentEditor from '../../components/mongo/MongoDocumentEditor.vue'
import MongoIndexManager from '../../components/mongo/MongoIndexManager.vue'
import MongoSchemaViewer from '../../components/mongo/MongoSchemaViewer.vue'

const router = useRouter()
const connStore = useMongoConnectionStore()
const dbStore = useMongoDatabaseStore()

// Active tab
const activeTab = ref<'documents' | 'indexes' | 'schema'>('documents')

// Document editor state
const editorVisible = ref(false)
const editorMode = ref<'insert' | 'edit'>('insert')
const editorInitialDoc = ref<string | undefined>(undefined)

// Export dialog state
const exportDialogVisible = ref(false)
const exportFormat = ref<'json' | 'csv'>('json')
const exporting = ref(false)

// Reset tab when collection changes
watch(
  () => dbStore.currentCollection,
  () => { activeTab.value = 'documents' }
)

// Reset database/collection selection when switching connections
watch(
  () => connStore.currentProfileId,
  (newId, oldId) => {
    if (newId && newId !== oldId) {
      dbStore.selectDatabase(null)
      dbStore.selectCollection(null)
      activeTab.value = 'documents'
    }
  }
)

// Explorer event handlers
const handleSelectCollection = (dbName: string, collName: string) => {
  dbStore.selectDatabase(dbName)
  dbStore.selectCollection(collName)
  activeTab.value = 'documents'
}

const handleViewIndexes = (dbName: string, collName: string) => {
  dbStore.selectDatabase(dbName)
  dbStore.selectCollection(collName)
  activeTab.value = 'indexes'
}

// Document editor
const openInsertEditor = () => {
  editorMode.value = 'insert'
  editorInitialDoc.value = undefined
  editorVisible.value = true
}

const openEditEditor = (doc: string) => {
  editorMode.value = 'edit'
  editorInitialDoc.value = doc
  editorVisible.value = true
}

const onDocumentSaved = () => {
  // MongoDocumentList will reload on its own via the 'saved' event chain;
  // nothing extra needed here.
}

// Export
const showExportDialog = () => {
  exportFormat.value = 'json'
  exportDialogVisible.value = true
}

const confirmExport = async () => {
  if (!connStore.currentProfileId || !dbStore.currentDatabase || !dbStore.currentCollection) return
  exporting.value = true
  try {
    await MongoDocumentAPI.exportDocuments(connStore.currentProfileId, {
      database: dbStore.currentDatabase,
      collection: dbStore.currentCollection,
      filter: '{}',
      format: exportFormat.value,
      filePath: '', // backend opens system file dialog
    })
    exportDialogVisible.value = false
    ElMessage.success('导出成功')
  } catch {
    // error already shown by API layer
  } finally {
    exporting.value = false
  }
}
</script>

<style scoped>
.mongo-workspace {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: #f5f7fa;
}

/* ── No connection ── */
.no-connection {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ── Top bar ── */
.workspace-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 44px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #303133;
  overflow: hidden;
}

.topbar-icon {
  color: #67c23a;
  font-size: 16px;
  flex-shrink: 0;
}

.connection-name {
  font-weight: 600;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
}

.topbar-sep {
  color: #c0c4cc;
  flex-shrink: 0;
}

.db-name {
  color: #606266;
  white-space: nowrap;
}

.coll-name {
  color: #409eff;
  font-weight: 500;
  white-space: nowrap;
}

/* ── Body ── */
.workspace-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* ── Sidebar ── */
.workspace-sidebar {
  width: 260px;
  flex-shrink: 0;
  border-right: 1px solid #e4e7ed;
  background: #fff;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* ── Content ── */
.workspace-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ── Tabs ── */
.collection-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.collection-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  padding: 0;
}

.collection-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
}
</style>
