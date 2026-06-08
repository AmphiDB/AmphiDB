<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { LogAPI } from '../../api/index'
import type { LogEntry } from '../../types/api'

const entries = ref<LogEntry[]>([])
const loading = ref(false)
const total = ref(0)
const newEntryIds = ref<Set<number>>(new Set())

const selectedEntry = ref<LogEntry | null>(null)
const dialogVisible = ref(false)

// 筛选条件
const filterLevel = ref<string>('')
const filterKeyword = ref<string>('')

let refreshTimer: ReturnType<typeof setInterval> | null = null
let prevCount = 0

function formatTimestamp(ts: string): string {
  if (!ts) return '-'
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}

function truncate(text: string, max = 150): string {
  if (!text) return ''
  return text.length > max ? text.slice(0, max) + '…' : text
}

function getLevelType(level: string): string {
  switch (level?.toUpperCase()) {
    case 'INFO': return 'success'
    case 'ERROR': return 'danger'
    case 'WARNING': return 'warning'
    case 'DEBUG': return 'info'
    default: return 'info'
  }
}

function getOperationType(operation: string): string {
  if (!operation) return ''
  const upper = operation.toUpperCase()
  if (upper.includes('INSERT') || upper.includes('CREATE')) return 'primary'
  if (upper.includes('UPDATE') || upper.includes('ALTER') || upper.includes('MODIFY')) return 'warning'
  if (upper.includes('DELETE') || upper.includes('DROP')) return 'danger'
  if (upper.includes('SELECT') || upper.includes('QUERY') || upper.includes('AGGREGATE')) return 'success'
  return 'info'
}

function getDetailFromEntry(entry: LogEntry): string {
  if (!entry.details) return ''
  const details = entry.details
  const parts: string[] = []
  if (details.sql) parts.push(`SQL: ${details.sql}`)
  if (details.detail) parts.push(`Detail: ${details.detail}`)
  if (details.rows_affected !== undefined) parts.push(`Rows Affected: ${details.rows_affected}`)
  if (details.error) parts.push(`Error: ${details.error}`)
  return parts.join('\n')
}

function isMongoOperation(entry: LogEntry): boolean {
  if (!entry.details) return false
  return entry.details.db_type === 'mongodb'
}

function rowClassName({ row }: { row: LogEntry }): string {
  return newEntryIds.value.has(row.id) ? 'new-entry' : ''
}

async function loadLog() {
  loading.value = true
  try {
    const filter: any = {
      limit: 200,
      offset: 0,
    }
    if (filterLevel.value) {
      filter.level = filterLevel.value
    }
    if (filterKeyword.value) {
      filter.keyword = filterKeyword.value
    }
    const result = await LogAPI.getLogs(filter)
    const newEntries = result ?? []

    if (newEntries.length > prevCount && prevCount > 0) {
      const addedCount = newEntries.length - prevCount
      const ids = new Set<number>()
      for (let i = 0; i < addedCount; i++) {
        ids.add(newEntries[i].id)
      }
      newEntryIds.value = ids
      setTimeout(() => { newEntryIds.value = new Set() }, 3000)
    }

    prevCount = newEntries.length
    entries.value = newEntries
    total.value = newEntries.length
  } catch (e) {
    console.error('Failed to load operation logs', e)
  } finally {
    loading.value = false
  }
}

async function loadCount() {
  try {
    total.value = await LogAPI.getLogCount()
  } catch (e) {
    console.error('Failed to load log count', e)
  }
}

function onRowClick(row: LogEntry) {
  selectedEntry.value = row
  dialogVisible.value = true
}

async function clearAllLogs() {
  try {
    await ElMessageBox.confirm('确定要清空所有操作日志吗？此操作不可恢复。', '确认清空', {
      confirmButtonText: '清空',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await LogAPI.clearLogs()
    entries.value = []
    prevCount = 0
    total.value = 0
    ElMessage.success('操作日志已清空')
  } catch {
    // user cancelled
  }
}

function onFilterChange() {
  prevCount = 0
  entries.value = []
  loadLog()
}

function startRefresh() {
  loadLog()
  refreshTimer = setInterval(loadLog, 15000)
}

function stopRefresh() {
  if (refreshTimer !== null) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

onMounted(() => {
  startRefresh()
})

onUnmounted(stopRefresh)
</script>

<template>
  <div class="operation-log-tab">
    <div class="toolbar">
      <el-select
        v-model="filterLevel"
        placeholder="日志级别"
        clearable
        size="small"
        style="width: 120px"
        @change="onFilterChange"
      >
        <el-option label="INFO" value="INFO" />
        <el-option label="ERROR" value="ERROR" />
        <el-option label="DEBUG" value="DEBUG" />
      </el-select>

      <el-input
        v-model="filterKeyword"
        placeholder="搜索关键词..."
        clearable
        size="small"
        style="width: 200px"
        @clear="onFilterChange"
        @keyup.enter="onFilterChange"
      />

      <el-button size="small" @click="onFilterChange" :loading="loading">
        搜索
      </el-button>

      <span class="total-info">共 {{ total }} 条记录</span>

      <el-button type="danger" size="small" @click="clearAllLogs" style="margin-left: auto">
        清空日志
      </el-button>
    </div>

    <el-table
      :data="entries"
      v-loading="loading"
      stripe
      highlight-current-row
      style="width: 100%"
      :row-class-name="rowClassName"
      @row-click="onRowClick"
      max-height="calc(100vh - 280px)"
    >
      <el-table-column label="时间" width="170">
        <template #default="{ row }">
          {{ formatTimestamp(row.timestamp) }}
        </template>
      </el-table-column>

      <el-table-column label="级别" width="80" align="center">
        <template #default="{ row }">
          <el-tag :type="getLevelType(row.level)" size="small" effect="dark">
            {{ row.level }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="类型" width="100" align="center">
        <template #default="{ row }">
          <el-tag v-if="isMongoOperation(row)" type="success" size="small" effect="plain">
            MongoDB
          </el-tag>
          <el-tag v-else type="primary" size="small" effect="plain">
            MySQL
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-tag :type="getOperationType(row.operation)" size="small" effect="plain">
            {{ row.operation }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="消息" min-width="200">
        <template #default="{ row }">
          <span class="message-text">{{ truncate(row.message) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="连接ID" width="150" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="conn-id">{{ row.connectionId || '—' }}</span>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="entries.length === 0 && !loading" class="no-data-hint">
      <el-empty description="暂无操作日志记录" />
    </div>

    <!-- 详情弹窗 -->
    <el-dialog v-model="dialogVisible" title="操作日志详情" width="min(820px, 92vw)" destroy-on-close>
      <div v-if="selectedEntry" class="log-detail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="时间">{{ formatTimestamp(selectedEntry.timestamp) }}</el-descriptions-item>
          <el-descriptions-item label="级别">
            <el-tag :type="getLevelType(selectedEntry.level)" size="small">{{ selectedEntry.level }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="操作类型">
            <el-tag :type="getOperationType(selectedEntry.operation)" size="small">{{ selectedEntry.operation }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="数据库类型">
            <el-tag v-if="isMongoOperation(selectedEntry)" type="success" size="small">MongoDB</el-tag>
            <el-tag v-else type="primary" size="small">MySQL</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="连接ID" :span="2">{{ selectedEntry.connectionId || '—' }}</el-descriptions-item>
          <el-descriptions-item label="消息" :span="2">{{ selectedEntry.message }}</el-descriptions-item>
        </el-descriptions>

        <div v-if="selectedEntry.details" class="detail-section">
          <h4>详细信息</h4>
          <pre class="detail-pre">{{ getDetailFromEntry(selectedEntry) }}</pre>
        </div>

        <div v-if="selectedEntry.details?.sql" class="detail-section">
          <h4>SQL 语句</h4>
          <pre class="sql-pre">{{ selectedEntry.details.sql }}</pre>
        </div>

        <div v-if="selectedEntry.details?.detail" class="detail-section">
          <h4>操作详情</h4>
          <pre class="detail-pre">{{ selectedEntry.details.detail }}</pre>
        </div>

        <div v-if="selectedEntry.details?.error" class="detail-section">
          <h4>错误信息</h4>
          <pre class="error-pre">{{ selectedEntry.details.error }}</pre>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.operation-log-tab { padding: 12px; }

.toolbar {
  display: flex; align-items: center; gap: 10px; margin-bottom: 12px;
}

.total-info {
  font-size: 13px; color: var(--el-text-color-regular); white-space: nowrap;
}

.message-text {
  font-size: 12px; cursor: pointer;
}

.conn-id {
  font-size: 12px; color: var(--el-text-color-secondary); font-family: monospace;
}

.no-data-hint {
  padding: 40px 0;
}

.log-detail {
  max-height: 600px; overflow-y: auto;
}

.detail-section {
  margin-top: 16px;
}

.detail-section h4 {
  margin: 0 0 8px 0; font-size: 14px; color: var(--el-text-color-primary);
}

.detail-pre, .sql-pre, .error-pre {
  white-space: pre-wrap; word-break: break-all;
  font-family: monospace; font-size: 13px;
  background: var(--el-fill-color-light);
  padding: 12px; border-radius: 4px;
  max-height: 300px; overflow-y: auto; margin: 0;
}

.error-pre {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}
</style>

<style>
.el-table .new-entry td {
  background-color: var(--el-color-warning-light-9) !important;
  transition: background-color 0.5s ease;
}
</style>
