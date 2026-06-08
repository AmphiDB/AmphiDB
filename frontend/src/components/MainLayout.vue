<template>
  <el-container class="main-layout">
    <el-aside width="210px" class="sidebar">
      <div class="logo">
        <img :src="logoIcon" alt="AmphiDB" class="logo-icon" />
        <span class="logo-text">AmphiDB</span>
      </div>

      <el-menu :default-active="activeMenu" class="sidebar-menu" @select="handleMenuSelect">

        <!-- 始终显示：连接管理 -->
        <el-menu-item index="/connections">
          <span class="sidebar-icon icon-connections"><el-icon><Connection /></el-icon></span>
          <span>连接管理</span>
        </el-menu-item>

        <!-- MySQL 多连接：每个活跃连接显示一个分组 -->
        <template v-for="conn in mysqlActiveList" :key="conn.id">
          <div
            class="section-label"
            :class="{ 'section-active': currentConnection?.id === conn.id }"
            :style="connectionAccentStyle(conn.id, 'mysql')"
            @click="switchMysqlConnection(conn)"
          >
            <span class="connection-glyph mysql-glyph">My</span>
            <span class="section-kind mysql-kind">MySQL</span>
            <span class="section-conn-name" :title="conn.name">{{ conn.name }}</span>
            <span v-if="currentConnection?.id === conn.id" class="section-status">当前</span>
            <el-icon class="section-close" title="关闭连接" @click.stop="handleDisconnectMysql(conn)"><Close /></el-icon>
          </div>
          <template v-if="currentConnection?.id === conn.id">
            <el-menu-item :index="'/workspace:' + conn.id" class="connection-menu-item">
              <span class="sidebar-icon icon-workspace"><el-icon><FolderOpened /></el-icon></span>
              <span>数据库工作台</span>
            </el-menu-item>
            <el-menu-item :index="'/query:' + conn.id" class="connection-menu-item">
              <span class="sidebar-icon icon-query"><el-icon><EditPen /></el-icon></span>
              <span>SQL 查询</span>
            </el-menu-item>
            <el-menu-item :index="'/sync:' + conn.id" class="connection-menu-item">
              <span class="sidebar-icon icon-schema"><el-icon><Refresh /></el-icon></span>
              <span>结构同步</span>
            </el-menu-item>
            <el-menu-item :index="'/data-sync:' + conn.id" class="connection-menu-item">
              <span class="sidebar-icon icon-data-sync"><el-icon><Switch /></el-icon></span>
              <span>数据同步</span>
            </el-menu-item>
          </template>
        </template>

        <!-- MongoDB 多连接：每个活跃连接显示一个分组 -->
        <template v-for="conn in mongoActiveList" :key="conn.id">
          <div
            class="section-label"
            :class="{ 'section-active': mongoStore.currentProfileId === conn.id }"
            :style="connectionAccentStyle(conn.id, 'mongo')"
            @click="switchMongoConnection(conn)"
          >
            <span class="connection-glyph mongo-glyph">Mo</span>
            <span class="section-kind mongo-kind">MongoDB</span>
            <span class="section-conn-name" :title="conn.name">{{ conn.name }}</span>
            <span v-if="mongoStore.currentProfileId === conn.id" class="section-status">当前</span>
            <el-icon class="section-close" title="关闭连接" @click.stop="handleDisconnectMongo(conn)"><Close /></el-icon>
          </div>
          <template v-if="mongoStore.currentProfileId === conn.id">
            <el-menu-item :index="'/mongo/workspace:' + conn.id" class="connection-menu-item">
              <span class="sidebar-icon icon-workspace"><el-icon><FolderOpened /></el-icon></span>
              <span>数据库工作台</span>
            </el-menu-item>
            <el-menu-item :index="'/mongo/query:' + conn.id" class="connection-menu-item">
              <span class="sidebar-icon icon-query"><el-icon><EditPen /></el-icon></span>
              <span>聚合查询</span>
            </el-menu-item>
          </template>
        </template>

        <!-- 未连接时的提示 -->
        <div v-if="!isConnected && !isMongoConnected" class="no-conn-hint">
          <span class="sidebar-icon icon-muted"><el-icon><InfoFilled /></el-icon></span>
          <span>请先建立连接</span>
        </div>

        <div class="menu-divider" />

        <!-- 始终显示：日志查看 -->
        <el-menu-item index="/logs">
          <span class="sidebar-icon icon-logs"><el-icon><Document /></el-icon></span>
          <span>日志查看</span>
        </el-menu-item>

      </el-menu>
    </el-aside>

    <el-container>
      <!-- Top bar: active connection badges -->
      <el-header class="header">
        <div class="header-left">
          <span class="header-label">活动连接</span>
          <div class="connection-strip">
            <template v-for="conn in mysqlActiveList" :key="'mysql-badge-' + conn.id">
              <el-tag
                type="primary"
                size="small"
                :effect="currentConnection?.id === conn.id ? 'dark' : 'plain'"
                class="conn-badge"
                closable
                @click="switchMysqlConnection(conn)"
                @close="handleDisconnectMysql(conn)"
              >
                MySQL: {{ conn.name }}
              </el-tag>
            </template>
            <template v-for="conn in mongoActiveList" :key="'mongo-badge-' + conn.id">
              <el-tag
                type="success"
                size="small"
                :effect="mongoStore.currentProfileId === conn.id ? 'dark' : 'plain'"
                class="conn-badge"
                closable
                @click="switchMongoConnection(conn)"
                @close="handleDisconnectMongo(conn)"
              >
                MongoDB: {{ conn.name }}
              </el-tag>
            </template>
          </div>
          <span v-if="!isConnected && !isMongoConnected" class="no-conn-text">
            <el-icon><WarningFilled /></el-icon> 未连接
          </span>
        </div>
        <div class="header-right">
          <el-button
            size="small"
            plain
            class="llm-button"
            @click="llmDialogVisible = true"
          >
            <el-icon><MagicStick /></el-icon>
            AI 配置
          </el-button>
          <el-popover
            placement="bottom-end"
            width="360"
            trigger="click"
            popper-class="transfer-popover"
          >
            <template #reference>
              <el-button
                size="small"
                :type="runningTransferCount > 0 ? 'primary' : 'default'"
                plain
                class="transfer-button"
              >
                <el-icon><Sort /></el-icon>
                传输任务
                <span v-if="runningTransferCount > 0" class="transfer-count">{{ runningTransferCount }}</span>
              </el-button>
            </template>
            <div class="transfer-panel">
              <div class="transfer-title">后台导入/导出</div>
              <el-empty
                v-if="transferTasks.length === 0"
                description="暂无后台任务"
                :image-size="72"
              />
              <div v-else class="transfer-list">
                <div
                  v-for="task in transferTasks"
                  :key="task.taskId"
                  class="transfer-item"
                >
                  <div class="transfer-item-head">
                    <div class="transfer-name">
                      <span class="transfer-kind" :class="task.kind">{{ task.kind === 'export' ? '导出' : '导入' }}</span>
                      <span class="transfer-object" :title="task.label">{{ task.label }}</span>
                    </div>
                    <el-tag size="small" :type="taskStatusType(task.status)">
                      {{ taskStatusText(task.status) }}
                    </el-tag>
                  </div>
                  <el-progress
                    :percentage="task.percentage"
                    :status="task.status === 'failed' || task.status === 'cancelled' ? 'exception' : task.status === 'completed' ? 'success' : undefined"
                    :stroke-width="6"
                  />
                  <div class="transfer-meta">
                    <span>{{ task.current }} / {{ task.total || '?' }} 行</span>
                    <el-button
                      v-if="task.status === 'running'"
                      size="small"
                      type="danger"
                      link
                      @click="stopTransferTask(task.taskId)"
                    >
                      停止
                    </el-button>
                  </div>
                  <div v-if="task.message" class="transfer-message" :title="task.message">
                    {{ task.message }}
                  </div>
                </div>
              </div>
            </div>
          </el-popover>
          <span class="active-count">
            {{ totalActiveConnections }} 个连接
          </span>
          <span v-if="activeContextText" class="context-path" :title="activeContextText">
            <el-icon><Coin /></el-icon>{{ activeContextText }}
          </span>
          <span v-else class="context-empty">
            未选择对象
          </span>
        </div>
      </el-header>

      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>

    <LLMSettingsDialog
      v-model="llmDialogVisible"
      @saved="handleLLMConfigSaved"
    />
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElNotification } from 'element-plus'
import { useConnectionStore } from '../stores/connection'
import { useDatabaseStore } from '../stores/database'
import { useMongoConnectionStore } from '../stores/mongoConnection'
import { ConnectionAPI } from '../api'
import { CancelTransferTask } from '../../wailsjs/go/backend/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import type { ConnectionProfile } from '../types/api'
import type { MongoConnectionProfile } from '../types/mongo'
import logoIcon from '../assets/images/appicon.png'
import LLMSettingsDialog from './LLMSettingsDialog.vue'
import {
  Connection, FolderOpened, EditPen, Refresh, Document,
  Coin, WarningFilled, InfoFilled, Switch, Close, Sort,
  MagicStick,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const mysqlStore = useConnectionStore()
const dbStore = useDatabaseStore()
const mongoStore = useMongoConnectionStore()

const activeMenu = computed(() => route.path)
const isConnected = computed(() => mysqlStore.isConnected)
const isMongoConnected = computed(() => mongoStore.isConnected)

const currentConnection = computed(() => mysqlStore.currentConnection)
const mysqlActiveList = computed(() => mysqlStore.activeConnectionList)
const mongoActiveList = computed(() => mongoStore.activeConnectionList)
const totalActiveConnections = computed(() => mysqlActiveList.value.length + mongoActiveList.value.length)

const currentDatabase = computed(() => dbStore.currentDatabase)
const currentTable = computed(() => dbStore.currentTable)
const activeContextText = computed(() => {
  if (!currentDatabase.value) return ''
  return currentTable.value ? `${currentDatabase.value} / ${currentTable.value}` : currentDatabase.value
})

type TransferStatus = 'running' | 'completed' | 'failed' | 'cancelled'
type TransferKind = 'export' | 'import'

interface TransferTask {
  taskId: string
  kind: TransferKind
  label: string
  format: string
  current: number
  total: number
  percentage: number
  status: TransferStatus
  message: string
}

const transferTasks = reactive<TransferTask[]>([])
const llmDialogVisible = ref(false)
const runningTransferCount = computed(() => transferTasks.filter(task => task.status === 'running').length)
const eventUnlisteners: Array<() => void> = []

const accentPalettes = {
  mysql: [
    { accent: '#4aa7ff', soft: 'rgba(74, 167, 255, 0.12)', active: 'rgba(74, 167, 255, 0.17)', text: '#dcefff' },
    { accent: '#8b8cff', soft: 'rgba(139, 140, 255, 0.12)', active: 'rgba(139, 140, 255, 0.17)', text: '#e8e8ff' },
    { accent: '#32c6a6', soft: 'rgba(50, 198, 166, 0.11)', active: 'rgba(50, 198, 166, 0.16)', text: '#dcf7f0' },
  ],
  mongo: [
    { accent: '#65c878', soft: 'rgba(101, 200, 120, 0.12)', active: 'rgba(101, 200, 120, 0.17)', text: '#e2f7e7' },
    { accent: '#2cc7b7', soft: 'rgba(44, 199, 183, 0.11)', active: 'rgba(44, 199, 183, 0.16)', text: '#ddf8f5' },
    { accent: '#a9c958', soft: 'rgba(169, 201, 88, 0.11)', active: 'rgba(169, 201, 88, 0.16)', text: '#f0f8dc' },
  ],
}

const stableIndex = (value: string, length: number) => {
  let hash = 0
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0
  }
  return hash % length
}

const connectionAccentStyle = (id: string, type: 'mysql' | 'mongo') => {
  const palette = accentPalettes[type]
  const color = palette[stableIndex(id || type, palette.length)]
  return {
    '--conn-accent': color.accent,
    '--conn-accent-soft': color.soft,
    '--conn-accent-active': color.active,
    '--conn-accent-text': color.text,
  }
}

const taskLabel = (data: any) => {
  const objectName = data.table ? `${data.database}.${data.table}` : data.database
  return `${objectName || '未命名任务'} · ${data.format || ''}`
}

const upsertTransferTask = (kind: TransferKind, data: any, patch: Partial<TransferTask> = {}) => {
  if (!data?.taskId) return
  const index = transferTasks.findIndex(task => task.taskId === data.taskId)
  const base: TransferTask = {
    taskId: data.taskId,
    kind,
    label: taskLabel(data),
    format: data.format || '',
    current: 0,
    total: 0,
    percentage: 0,
    status: 'running',
    message: '',
  }
  if (index >= 0) {
    Object.assign(transferTasks[index], base, patch)
    return
  }
  transferTasks.unshift({ ...base, ...patch })
}

const taskStatusText = (status: TransferStatus) => {
  return {
    running: '运行中',
    completed: '完成',
    failed: '失败',
    cancelled: '已停止',
  }[status]
}

const taskStatusType = (status: TransferStatus) => {
  return {
    running: 'primary',
    completed: 'success',
    failed: 'danger',
    cancelled: 'warning',
  }[status] as 'primary' | 'success' | 'danger' | 'warning'
}

const stopTransferTask = async (taskId: string) => {
  try {
    await CancelTransferTask(taskId)
    const task = transferTasks.find(item => item.taskId === taskId)
    if (task) task.message = '正在停止任务...'
  } catch (e: any) {
    ElMessage.error(`停止任务失败: ${e?.message || e}`)
  }
}

const handleLLMConfigSaved = () => {
  ElNotification.success({ title: 'AI 配置已更新', message: '白话查询已使用新的 LLM 配置', duration: 3000 })
}

onMounted(() => {
  eventUnlisteners.push(
    EventsOn('export:started', (data: any) => upsertTransferTask('export', data)),
    EventsOn('import:started', (data: any) => upsertTransferTask('import', data)),
    EventsOn('export:progress', (data: any) => upsertTransferTask('export', data, {
      current: data.current || 0,
      total: data.total || 0,
      percentage: Math.round(data.percentage || 0),
      status: 'running',
    })),
    EventsOn('import:progress', (data: any) => upsertTransferTask('import', data, {
      current: data.current || 0,
      total: data.total || 0,
      percentage: Math.round(data.percentage || 0),
      status: 'running',
    })),
    EventsOn('export:completed', (data: any) => {
      upsertTransferTask('export', data, {
        percentage: 100,
        status: 'completed',
        message: data.outputPath || '',
      })
    }),
    EventsOn('import:completed', (data: any) => {
      upsertTransferTask('import', data, {
        percentage: 100,
        current: data.totalRows || data.TotalRows || 0,
        total: data.totalRows || data.TotalRows || 0,
        status: 'completed',
        message: `成功 ${data.successRows || 0} 行，失败 ${data.failedRows || 0} 行`,
      })
    }),
    EventsOn('export:failed', (data: any) => upsertTransferTask('export', data, {
      status: 'failed',
      message: data.error || '导出失败',
    })),
    EventsOn('import:failed', (data: any) => upsertTransferTask('import', data, {
      status: 'failed',
      message: data.error || '导入失败',
    })),
    EventsOn('export:cancelled', (data: any) => {
      upsertTransferTask('export', data, {
        status: 'cancelled',
        message: '用户已停止导出任务',
      })
      ElNotification.warning({ title: '导出已停止', message: taskLabel(data), duration: 4000 })
    }),
    EventsOn('import:cancelled', (data: any) => {
      upsertTransferTask('import', data, {
        status: 'cancelled',
        message: '用户已停止导入任务',
      })
      ElNotification.warning({ title: '导入已停止', message: taskLabel(data), duration: 4000 })
    }),
  )
})

onUnmounted(() => {
  eventUnlisteners.splice(0).forEach(off => off())
})

// 切换 MySQL 当前连接
const switchMysqlConnection = (conn: ConnectionProfile) => {
  mysqlStore.setCurrentConnection(conn)
}

// 切换 MongoDB 当前连接
const switchMongoConnection = (conn: MongoConnectionProfile) => {
  mongoStore.setCurrentProfile(conn.id)
}

// 关闭 MySQL 连接
const handleDisconnectMysql = async (conn: ConnectionProfile) => {
  try {
    await ConnectionAPI.disconnect(conn.id)
    mysqlStore.removeActiveConnection(conn.id)
    ElMessage.success(`已断开连接 ${conn.name}`)
  } catch (e: any) {
    ElMessage.error(`断开连接失败: ${e?.message || e}`)
  }
}

// 关闭 MongoDB 连接
const handleDisconnectMongo = async (conn: MongoConnectionProfile) => {
  try {
    await mongoStore.disconnect(conn.id)
    ElMessage.success(`已断开连接 ${conn.name}`)
  } catch (e: any) {
    ElMessage.error(`断开连接失败: ${e?.message || e}`)
  }
}

const handleMenuSelect = (index: string) => {
  // 路由中可能包含连接ID后缀（如 /workspace:conn-id），需要提取实际路径
  const colonIdx = index.indexOf(':')
  const path = colonIdx > 0 ? index.substring(0, colonIdx) : index
  router.push(path)
}
</script>

<style scoped>
.main-layout { height: 100vh; width: 100vw; }

/* ── Sidebar ── */
.sidebar {
  background:
    radial-gradient(circle at 24px 18px, rgba(81, 159, 255, 0.18), transparent 120px),
    linear-gradient(180deg, #202f42 0%, #1c2736 52%, #1a2431 100%);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 14px;
  border-bottom: 1px solid rgba(221, 227, 234, 0.14);
  flex-shrink: 0;
  transform: translateY(8px);
}

.logo-icon {
  width: 24px;
  height: 24px;
  object-fit: contain;
  flex-shrink: 0;
}
.logo-text { font-size: 14px; font-weight: 700; color: #f4f8fb; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.sidebar-menu {
  flex: 1;
  overflow-y: auto;
  border: none;
  background: transparent;
}

/* Section label (MySQL / MongoDB group header) */
.section-label {
  position: relative;
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 30px;
  padding: 5px 8px;
  margin-top: 7px;
  cursor: pointer;
  transition: background-color 0.18s, border-color 0.18s;
  border: 1px solid transparent;
  border-left: 0;
  border-radius: 8px;
  margin-left: 8px;
  margin-right: 8px;
  background: transparent;
}

.connection-glyph,
.sidebar-icon {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid transparent;
  box-shadow: none;
}

.connection-glyph::after,
.sidebar-icon::after {
  display: none;
}

.sidebar-icon .el-icon {
  margin: 0;
  color: currentColor;
  font-size: 14px;
  line-height: 1;
}

.connection-glyph {
  color: #fff;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0;
}

.mysql-glyph {
  background:
    linear-gradient(180deg, rgba(89, 176, 255, 0.92), rgba(39, 118, 211, 0.82));
  border-color: rgba(154, 211, 255, 0.24);
  box-shadow: 0 3px 8px rgba(39, 118, 211, 0.16);
}

.mongo-glyph {
  background:
    linear-gradient(180deg, rgba(110, 211, 126, 0.92), rgba(42, 139, 82, 0.82));
  border-color: rgba(169, 236, 181, 0.24);
  box-shadow: 0 3px 8px rgba(42, 139, 82, 0.14);
}

.section-label:hover {
  background: rgba(255, 255, 255, 0.045);
  border-color: transparent;
}

.section-label.section-active {
  background: rgba(255, 255, 255, 0.055);
  border-color: transparent;
  box-shadow: none;
}

.section-label.section-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 7px;
  bottom: 7px;
  width: 2px;
  border-radius: 999px;
  background: var(--conn-accent, #4aa7ff);
}

.section-kind {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  height: auto;
  border-radius: 0;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0;
}

.mysql-kind {
  color: var(--conn-accent-text, #d8efff);
  background: transparent;
  border: 0;
}

.mongo-kind {
  min-width: 0;
  color: var(--conn-accent-text, #def7e5);
  background: transparent;
  border: 0;
}

.section-conn-name {
  font-size: 12px;
  color: #d6e0e9;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 72px;
  flex: 1;
}

.section-status {
  flex-shrink: 0;
  padding: 2px 5px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.09);
  color: #e5eef5;
  font-size: 10px;
  font-weight: 600;
  line-height: 1;
}

.section-label.section-active .section-conn-name {
  color: #fff;
  font-weight: 600;
}

.section-label + .connection-menu-item {
  margin-top: 2px;
}

.sidebar-menu :deep(.connection-menu-item) {
  margin-left: 18px;
  padding-left: 8px !important;
}

.section-close {
  flex-shrink: 0;
}

.section-close {
  font-size: 12px;
  color: #8492a0;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.2s, color 0.2s;
  margin-left: 2px;
}

.section-label:hover .section-close {
  opacity: 1;
}

.section-close:hover {
  color: #f56c6c;
}

/* No connection hint */
.no-conn-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 14px;
  font-size: 12px;
  color: #64748b;
}

.menu-divider {
  height: 1px;
  background-color: rgba(221, 227, 234, 0.12);
  margin: 6px 12px;
}

/* Menu item styles */
.sidebar-menu :deep(.el-menu-item) {
  margin: 2px 8px;
  padding: 0 8px !important;
  color: #aab6c2;
  height: 32px;
  line-height: 32px;
  border-radius: 7px;
  font-size: 13px;
  gap: 9px;
  transition: background-color 0.16s, color 0.16s, box-shadow 0.16s;
}

.sidebar-menu :deep(.el-menu-item .el-icon) {
  margin-right: 0;
}

.icon-connections {
  color: #b9dcff;
}

.icon-workspace {
  color: #d7e8f5;
}

.icon-query {
  color: #e6dcff;
}

.icon-schema {
  color: #d9f0e7;
}

.icon-data-sync {
  color: #e9e2ff;
}

.icon-logs,
.icon-muted {
  color: #c9d3de;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background-color: rgba(255, 255, 255, 0.052) !important;
  color: #f4f8fb;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.20), rgba(255, 255, 255, 0.115)) !important;
  color: #f7fafc;
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.09) inset,
    0 6px 16px rgba(0, 0, 0, 0.11);
}

.sidebar-menu :deep(.el-menu-item.is-active .sidebar-icon) {
  color: #fff;
  background: rgba(255, 255, 255, 0.10);
  border-color: rgba(255, 255, 255, 0.12);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.10) inset;
}

.sidebar-menu :deep(.el-menu-item.is-disabled) {
  color: #475569;
  opacity: 0.6;
}

/* ── Header ── */
.header {
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 16px;
  height: 48px;
}

.header-left, .header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.header-left {
  flex: 1;
}

.header-right {
  flex-shrink: 0;
}

.header-label,
.active-count,
.context-empty {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}

.connection-strip {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
}

.connection-strip::-webkit-scrollbar {
  display: none;
}

.conn-badge {
  cursor: pointer;
  flex-shrink: 0;
}
.conn-badge:hover { opacity: 0.8; }

.no-conn-text {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #9ca3af;
}

.context-path {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #606266;
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-left: 10px;
  border-left: 1px solid #e5e7eb;
}

.transfer-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.transfer-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: var(--app-primary);
  color: #fff;
  font-size: 11px;
  line-height: 1;
}

.transfer-panel {
  min-height: 120px;
}

.transfer-title {
  margin-bottom: 10px;
  color: var(--app-text);
  font-size: 13px;
  font-weight: 600;
}

.transfer-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 360px;
  overflow: auto;
}

.transfer-item {
  padding: 10px;
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: #fff;
}

.transfer-item-head,
.transfer-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.transfer-name {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.transfer-kind {
  flex-shrink: 0;
  padding: 2px 5px;
  border-radius: 4px;
  color: #fff;
  font-size: 11px;
  line-height: 1.2;
}

.transfer-kind.export {
  background: #3f6f8f;
}

.transfer-kind.import {
  background: #578469;
}

.transfer-object {
  min-width: 0;
  color: var(--app-text);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.transfer-meta,
.transfer-message {
  margin-top: 6px;
  color: var(--app-text-muted);
  font-size: 12px;
}

.transfer-message {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ── Main content ── */
.main-content {
  background-color: var(--app-bg);
  overflow: auto;
  padding: 0;
}
</style>
