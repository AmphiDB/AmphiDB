<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import {
  GetSlowQueryLog,
  ClearSlowQueryLog,
  GetSlowQueryThreshold,
  SetSlowQueryThreshold,
} from '../../api/monitoring'
import type { SlowQueryEntry } from '../../api/monitoring'
import { useConnectionStore } from '../../stores/connection'
import { useMongoConnectionStore } from '../../stores/mongoConnection'

const mysqlStore = useConnectionStore()
const mongoStore = useMongoConnectionStore()

// Active profile IDs
const mysqlProfileId = computed(() => mysqlStore.isConnected ? mysqlStore.currentConnection?.id ?? '' : '')
const mongoProfileId = computed(() => mongoStore.currentProfileId ?? '')

// Current active profile (prefer MySQL if both connected)
const activeProfileId = computed(() => mysqlProfileId.value || mongoProfileId.value)

const entries = ref<SlowQueryEntry[]>([])
const loading = ref(false)
const threshold = ref(1000)
const newEntryIds = ref<Set<number>>(new Set())

const selectedEntry = ref<SlowQueryEntry | null>(null)
const dialogVisible = ref(false)

let refreshTimer: ReturnType<typeof setInterval> | null = null
let prevCount = 0

function formatTimestamp(ts: string): string {
  if (!ts) return '-'
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}

function truncate(text: string, max = 120): string {
  if (!text) return ''
  return text.length > max ? text.slice(0, max) + '…' : text
}

function dbCollection(entry: SlowQueryEntry): string {
  if (entry.dbType === 'mongodb' && entry.collection) {
    return `${entry.database} / ${entry.collection}`
  }
  return entry.database || '-'
}

function isDurationRed(durationMs: number): boolean {
  return durationMs > 3 * threshold.value
}

function rowClassName({ row }: { row: SlowQueryEntry }): string {
  return newEntryIds.value.has(row.id) ? 'new-entry' : ''
}

async function loadLog() {
  loading.value = true
  try {
    // Pass activeProfileId (may be empty — backend returns all profiles when empty)
    const result = await GetSlowQueryLog(activeProfileId.value, 50)
    const newEntries = result ?? []

    if (newEntries.length > prevCount) {
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
  } catch (e) {
    console.error('Failed to load slow query log', e)
  } finally {
    loading.value = false
  }
}

async function loadThreshold() {
  if (!activeProfileId.value) return
  try {
    threshold.value = await GetSlowQueryThreshold(activeProfileId.value)
  } catch (e) {
    console.error('Failed to load threshold', e)
  }
}

async function onThresholdChange(val: number | null) {
  if (val === null || val <= 0 || !activeProfileId.value) return
  try {
    await SetSlowQueryThreshold(activeProfileId.value, val)
  } catch (e) {
    console.error('Failed to set threshold', e)
    ElMessage.error('Failed to update threshold')
  }
}

function onRowClick(row: SlowQueryEntry) {
  selectedEntry.value = row
  dialogVisible.value = true
}

async function clearLog() {
  if (!activeProfileId.value) return
  try {
    await ElMessageBox.confirm('Clear all slow query log entries?', 'Confirm', {
      confirmButtonText: 'Clear',
      cancelButtonText: 'Cancel',
      type: 'warning',
    })
    await ClearSlowQueryLog(activeProfileId.value)
    entries.value = []
    prevCount = 0
    ElMessage.success('Slow query log cleared')
  } catch {
    // user cancelled
  }
}

function startRefresh() {
  loadThreshold()
  loadLog()
  refreshTimer = setInterval(loadLog, 10000)
}

function stopRefresh() {
  if (refreshTimer !== null) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

// Reload when active profile changes
watch(activeProfileId, () => {
  stopRefresh()
  prevCount = 0
  entries.value = []
  if (activeProfileId.value) startRefresh()
})

onMounted(() => {
  startRefresh()
})

onUnmounted(stopRefresh)
</script>

<template>
  <div class="slow-query-tab">
    <div class="toolbar">
      <span class="threshold-label">慢查询阈值 (ms):</span>
      <el-input-number
        v-model="threshold"
        :min="1"
        :step="100"
        size="small"
        style="width: 140px"
        :disabled="!activeProfileId"
        @change="onThresholdChange"
      />
      <el-tag v-if="!activeProfileId" type="info" size="small">未连接</el-tag>
      <el-button type="danger" size="small" @click="clearLog" :disabled="!activeProfileId" style="margin-left: auto">
        清空
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
    >
      <el-table-column label="时间" width="170">
        <template #default="{ row }">
          {{ formatTimestamp(row.timestamp) }}
        </template>
      </el-table-column>

      <el-table-column label="连接" prop="connectionId" width="150" show-overflow-tooltip />

      <el-table-column label="数据库 / 集合" width="180">
        <template #default="{ row }">
          {{ dbCollection(row) }}
        </template>
      </el-table-column>

      <el-table-column label="查询语句">
        <template #default="{ row }">
          <span class="query-text">{{ truncate(row.queryText) }}</span>
        </template>
      </el-table-column>

      <el-table-column label="耗时 (ms)" width="130" align="right">
        <template #default="{ row }">
          <span :class="{ 'duration-red': isDurationRed(row.durationMs) }">
            {{ row.durationMs }}
          </span>
        </template>
      </el-table-column>

      <el-table-column label="错误信息" width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.errorMessage" class="error-text">{{ row.errorMessage }}</span>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="entries.length === 0 && !loading" class="no-data-hint">
      <el-empty description="暂无慢查询记录（阈值: {{ threshold }}ms）" />
    </div>

    <el-dialog v-model="dialogVisible" title="慢查询详情" width="640px" destroy-on-close>
      <pre v-if="selectedEntry" class="query-detail">{{ selectedEntry.queryText }}</pre>
    </el-dialog>
  </div>
</template>

<style scoped>
.slow-query-tab { padding: 12px; }

.no-conn {
  display: flex; align-items: center; justify-content: center; height: 300px;
}

.toolbar {
  display: flex; align-items: center; gap: 10px; margin-bottom: 12px;
}

.threshold-label {
  font-size: 13px; color: var(--el-text-color-regular); white-space: nowrap;
}

.query-text {
  font-family: monospace; font-size: 12px; cursor: pointer;
}

.duration-red {
  color: var(--el-color-danger); font-weight: 600;
}

.error-text {
  color: var(--el-color-danger); font-size: 12px;
}

.muted {
  color: var(--el-text-color-placeholder);
}

.query-detail {
  white-space: pre-wrap; word-break: break-all;
  font-family: monospace; font-size: 13px;
  background: var(--el-fill-color-light);
  padding: 12px; border-radius: 4px;
  max-height: 400px; overflow-y: auto; margin: 0;
}
</style>

<style>
.el-table .new-entry td {
  background-color: var(--el-color-warning-light-9) !important;
  transition: background-color 0.5s ease;
}
</style>
