<template>
  <div class="mongo-explorer">
    <div class="explorer-header">
      <h3>MongoDB 浏览器</h3>
      <el-button
        :icon="Refresh"
        circle
        size="small"
        @click="refreshDatabases"
        :loading="dbStore.isLoading"
        title="刷新"
      />
    </div>

    <div class="explorer-content">
      <el-empty v-if="!dbStore.isLoading && databases.length === 0" description="暂无数据库" />

      <div v-else class="db-list">
        <div v-for="db in databases" :key="db.name" class="db-item">
          <!-- 数据库行 -->
          <div
            class="db-row"
            :class="{ 'is-active': activeDb === db.name && !activeCollection }"
            @click="handleDbClick(db.name)"
            @contextmenu.prevent="handleContextMenu($event, { type: 'database', dbName: db.name })"
          >
            <el-icon class="arrow-icon" :class="{ expanded: expandedDbs.has(db.name) }">
              <ArrowRight />
            </el-icon>
            <el-icon class="node-icon"><Coin /></el-icon>
            <span class="node-label">{{ db.name }}</span>
            <el-icon v-if="loadingDbs.has(db.name)" class="loading-icon is-loading"><Loading /></el-icon>
          </div>

          <!-- 展开内容：筛选框 + 集合列表 -->
          <div v-if="expandedDbs.has(db.name)" class="db-children">
            <!-- 筛选框 -->
            <div class="coll-filter" @click.stop>
              <el-input
                v-model="dbFilterMap[db.name]"
                placeholder="筛选集合名..."
                clearable
                size="small"
                :prefix-icon="Search"
              />
            </div>

            <!-- 集合节点 -->
            <div
              v-for="coll in getFilteredCollections(db.name)"
              :key="coll.name"
              class="coll-row"
              :class="{ 'is-active': activeDb === db.name && activeCollection === coll.name }"
              @click="handleCollClick(db.name, coll.name)"
              @dblclick="handleCollDblClick(db.name, coll.name)"
              @contextmenu.prevent="handleContextMenu($event, { type: 'collection', dbName: db.name, collName: coll.name })"
            >
              <el-icon class="node-icon coll-icon"><Collection /></el-icon>
              <span class="node-label">{{ coll.name }}</span>
              <span class="node-meta">({{ formatCount(coll.documentCount) }})</span>
            </div>

            <!-- 无匹配提示 -->
            <div
              v-if="dbFilterMap[db.name] && getFilteredCollections(db.name).length === 0 && !loadingDbs.has(db.name)"
              class="no-match"
            >
              无匹配集合
            </div>

            <!-- 空集合提示 -->
            <div
              v-if="!loadingDbs.has(db.name) && getCollectionsForDb(db.name).length === 0 && !dbFilterMap[db.name]"
              class="no-match"
            >
              暂无集合
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
        @click.stop
      >
        <template v-if="contextMenuData?.type === 'database'">
          <div class="context-menu-item" @click="handleMenuCommand('create-collection')">
            <el-icon><Plus /></el-icon>
            <span>新建集合</span>
          </div>
          <div class="context-menu-divider" />
          <div class="context-menu-item" @click="handleMenuCommand('refresh-collections')">
            <el-icon><Refresh /></el-icon>
            <span>刷新</span>
          </div>
        </template>
        <template v-else-if="contextMenuData?.type === 'collection'">
          <div class="context-menu-item" @click="handleMenuCommand('view-documents')">
            <el-icon><Grid /></el-icon>
            <span>查看文档</span>
          </div>
          <div class="context-menu-item" @click="handleMenuCommand('view-indexes')">
            <el-icon><Document /></el-icon>
            <span>查看索引</span>
          </div>
          <div class="context-menu-divider" />
          <div class="context-menu-item danger" @click="handleMenuCommand('drop-collection')">
            <el-icon><Delete /></el-icon>
            <span>删除集合</span>
          </div>
        </template>
      </div>
    </teleport>

    <!-- 新建集合对话框 -->
    <el-dialog
      v-model="createCollDialogVisible"
      title="新建集合"
      width="min(520px, 86vw)"
      @closed="newCollName = ''"
    >
      <el-form @submit.prevent="confirmCreateCollection">
        <el-form-item label="集合名称" label-width="80px">
          <el-input
            v-model="newCollName"
            placeholder="请输入集合名称"
            autofocus
            clearable
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createCollDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="confirmCreateCollection">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh,
  Coin,
  Document,
  Grid,
  Delete,
  Loading,
  Plus,
  Search,
  ArrowRight,
  Collection,
} from '@element-plus/icons-vue'
import { useMongoDatabaseStore } from '../../stores/mongoDatabase'
import type { MongoCollection } from '../../types/mongo'

const props = defineProps<{
  profileId: string
}>()

const emit = defineEmits<{
  'select-collection': [dbName: string, collName: string]
  'view-indexes': [dbName: string, collName: string]
}>()

const dbStore = useMongoDatabaseStore()

// Local state
const databases = computed(() => dbStore.databases)
const expandedDbs = reactive(new Set<string>())
const loadingDbs = reactive(new Set<string>())
const dbCollectionsMap = reactive<Record<string, MongoCollection[]>>({})
const dbFilterMap = reactive<Record<string, string>>({})

const activeDb = ref<string | null>(null)
const activeCollection = ref<string | null>(null)

// Context menu
const contextMenuVisible = ref(false)
const contextMenuPosition = ref({ x: 0, y: 0 })
const contextMenuData = ref<{ type: string; dbName: string; collName?: string } | null>(null)

// Create collection dialog
const createCollDialogVisible = ref(false)
const newCollName = ref('')
const creating = ref(false)
const createTargetDb = ref('')

// Helpers
const formatCount = (count: number): string => {
  if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`
  if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`
  return count.toString()
}

const getCollectionsForDb = (dbName: string): MongoCollection[] =>
  dbCollectionsMap[dbName] || []

const getFilteredCollections = (dbName: string): MongoCollection[] => {
  const colls = getCollectionsForDb(dbName)
  const filter = (dbFilterMap[dbName] || '').toLowerCase()
  if (!filter) return colls
  return colls.filter(c => c.name.toLowerCase().includes(filter))
}

// Load databases
const loadDatabases = async () => {
  await dbStore.loadDatabases(props.profileId)
}

// Load collections for a database
const loadCollections = async (dbName: string) => {
  loadingDbs.add(dbName)
  try {
    await dbStore.loadCollections(props.profileId, dbName)
    dbCollectionsMap[dbName] = [...dbStore.collections]
  } catch {
    // error already handled in store/api
  } finally {
    loadingDbs.delete(dbName)
  }
}

// Click handlers
const handleDbClick = async (dbName: string) => {
  activeDb.value = dbName
  activeCollection.value = null
  dbStore.selectDatabase(dbName)

  if (expandedDbs.has(dbName)) {
    expandedDbs.delete(dbName)
  } else {
    expandedDbs.add(dbName)
    await loadCollections(dbName)
  }
}

const handleCollClick = (dbName: string, collName: string) => {
  activeDb.value = dbName
  activeCollection.value = collName
  dbStore.selectDatabase(dbName)
  dbStore.selectCollection(collName)
}

const handleCollDblClick = (dbName: string, collName: string) => {
  activeDb.value = dbName
  activeCollection.value = collName
  dbStore.selectDatabase(dbName)
  dbStore.selectCollection(collName)
  emit('select-collection', dbName, collName)
}

// Context menu
const handleContextMenu = (event: MouseEvent, data: { type: string; dbName: string; collName?: string }) => {
  contextMenuData.value = data
  contextMenuPosition.value = { x: event.clientX, y: event.clientY }
  contextMenuVisible.value = true
  const close = () => {
    contextMenuVisible.value = false
    document.removeEventListener('click', close)
  }
  setTimeout(() => document.addEventListener('click', close), 100)
}

const handleMenuCommand = async (command: string) => {
  const data = contextMenuData.value
  contextMenuVisible.value = false
  contextMenuData.value = null
  if (!data) return

  switch (command) {
    case 'create-collection':
      createTargetDb.value = data.dbName
      createCollDialogVisible.value = true
      break

    case 'refresh-collections':
      dbFilterMap[data.dbName] = ''
      if (expandedDbs.has(data.dbName)) {
        await loadCollections(data.dbName)
      }
      ElMessage.success('集合列表已刷新')
      break

    case 'view-documents':
      if (data.collName) {
        activeDb.value = data.dbName
        activeCollection.value = data.collName
        emit('select-collection', data.dbName, data.collName)
      }
      break

    case 'view-indexes':
      if (data.collName) {
        activeDb.value = data.dbName
        activeCollection.value = data.collName
        emit('view-indexes', data.dbName, data.collName)
      }
      break

    case 'drop-collection':
      if (data.collName) await handleDropCollection(data.dbName, data.collName)
      break
  }
}

const handleDropCollection = async (dbName: string, collName: string) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除集合 "${dbName}.${collName}" 吗？此操作不可恢复！`,
      '删除集合',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
    await dbStore.dropCollection(props.profileId, dbName, collName)
    dbCollectionsMap[dbName] = [...dbStore.collections]
    ElMessage.success('集合已删除')
    if (activeCollection.value === collName && activeDb.value === dbName) {
      activeCollection.value = null
    }
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error(error?.message || '删除集合失败')
  }
}

const confirmCreateCollection = async () => {
  const name = newCollName.value.trim()
  if (!name) {
    ElMessage.warning('请输入集合名称')
    return
  }
  creating.value = true
  try {
    await dbStore.createCollection(props.profileId, createTargetDb.value, name)
    dbCollectionsMap[createTargetDb.value] = [...dbStore.collections]
    ElMessage.success(`集合 "${name}" 已创建`)
    createCollDialogVisible.value = false
    // Ensure the db is expanded to show the new collection
    if (!expandedDbs.has(createTargetDb.value)) {
      expandedDbs.add(createTargetDb.value)
    }
  } catch (error: any) {
    ElMessage.error(error?.message || '创建集合失败')
  } finally {
    creating.value = false
  }
}

const refreshDatabases = async () => {
  expandedDbs.clear()
  Object.keys(dbCollectionsMap).forEach(k => delete dbCollectionsMap[k])
  await loadDatabases()
  ElMessage.success('数据库列表已刷新')
}

// Watch for profileId changes (when switching connections)
watch(
  () => props.profileId,
  async (newId, oldId) => {
    if (newId && newId !== oldId) {
      // Clear all cached data
      expandedDbs.clear()
      Object.keys(dbCollectionsMap).forEach(k => delete dbCollectionsMap[k])
      Object.keys(dbFilterMap).forEach(k => delete dbFilterMap[k])
      activeDb.value = null
      activeCollection.value = null
      await loadDatabases()
    }
  }
)

onMounted(async () => {
  if (props.profileId) await loadDatabases()
})

onUnmounted(() => {
  expandedDbs.clear()
})
</script>

<style scoped>
.mongo-explorer {
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

.db-row:hover { background: #f5f7fa; }
.db-row.is-active { background: #ecf5ff; }

.arrow-icon {
  font-size: 12px;
  color: #909399;
  transition: transform 0.2s;
  flex-shrink: 0;
}

.arrow-icon.expanded { transform: rotate(90deg); }

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
.db-children { padding-left: 0; }

/* 筛选框 */
.coll-filter {
  padding: 4px 8px 4px 28px;
}

/* 集合行 */
.coll-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 8px 0 28px;
  height: 28px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}

.coll-row:hover { background: #f5f7fa; }
.coll-row.is-active { background: #ecf5ff; }
.coll-row.is-active .node-label {
  color: #409eff;
  font-weight: 500;
}

.coll-icon {
  font-size: 13px;
  color: #67c23a;
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

.context-menu-item:hover { background: #f5f7fa; }
.context-menu-item.danger { color: #f56c6c; }
.context-menu-item.danger:hover { background: #fef0f0; }

.context-menu-divider {
  height: 1px;
  background: #e4e7ed;
  margin: 4px 0;
}
</style>
