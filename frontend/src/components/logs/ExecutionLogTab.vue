<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { GetExecutionLog, ClearExecutionLog } from '../../api/monitoring'
import type { ExecutionLogEntry } from '../../api/monitoring'

// ─── State ────────────────────────────────────────────────────────────────────

const entries = ref<ExecutionLogEntry[]>([])
const loading = ref(false)
const filterDbType = ref('all')
const filterStatus = ref('all')
const currentPage = ref(1)
const pageSize = 10

const selectedEntry = ref<ExecutionLogEntry | null>(null)
const dialogVisible = ref(false)

let refreshTimer: ReturnType<typeof setInterval> | null = null

// ─── Computed ─────────────────────────────────────────────────────────────────

const filteredEntries = computed(() => {
  return entries.value.filter(e => {
    if (filterDbType.value !== 'all' && e.dbType !== filterDbType.value) return false
    if (filterStatus.value === 'success' && !e.success) return false
    if (filterStatus.value === 'error' && e.success) return false
    return true
  })
})

const pagedEntries = computed(() => {
  const start = (currentPage.value - 1) * pageSize
  return filteredEntries.value.slice(start, start + pageSize)
})

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatTimestamp(ts: string): string {
  if (!ts) return '-'
  const d = new Date(ts)
  return isNaN(d.getTime()) ? ts : d.toLocaleString()
}

function truncate(text: string, max = 120): string {
  if (!text) return ''
  return text.length > max ? text.slice(0, max) + '…' : text
}

function dbCollection(entry: ExecutionLogEntry): string {
  if (entry.dbType === 'mongodb' && entry.collection) {
    return `${entry.database} / ${entry.collection}`
  }
  return entry.database || '-'
}

// ─── Data Loading ─────────────────────────────────────────────────────────────

async function loadLog() {
  loading.value = true
  try {
    const result = await GetExecutionLog('', 'all', 200)
    entries.value = result ?? []
  } catch (e) {
    console.error('Failed to load execution log', e)
  } finally {
    loading.value = false
  }
}

// ─── Actions ──────────────────────────────────────────────────────────────────

function onRowClick(row: ExecutionLogEntry) {
  selectedEntry.value = row
  dialogVisible.value = true
}

async function clearLog() {
  try {
    await ElMessageBox.confirm('Clear all execution log entries?', 'Confirm', {
      confirmButtonText: 'Clear',
      cancelButtonText: 'Cancel',
      type: 'warning',
    })
    await ClearExecutionLog('', 'all')
    entries.value = []
    ElMessage.success('Log cleared')
  } catch {
    // user cancelled
  }
}

function onFilterChange() {
  currentPage.value = 1
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────

onMounted(() => {
  loadLog()
  refreshTimer = setInterval(loadLog, 5000)
})

onUnmounted(() => {
  if (refreshTimer !== null) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<template>
  <div class="execution-log-tab">
    <!-- Filter bar -->
    <div class="filter-bar">
      <el-select
        v-model="filterDbType"
        size="small"
        style="width: 130px"
        @change="onFilterChange"
      >
        <el-option label="All Types" value="all" />
        <el-option label="MySQL" value="mysql" />
        <el-option label="MongoDB" value="mongodb" />
      </el-select>

      <el-select
        v-model="filterStatus"
        size="small"
        style="width: 120px"
        @change="onFilterChange"
      >
        <el-option label="All Status" value="all" />
        <el-option label="Success" value="success" />
        <el-option label="Error" value="error" />
      </el-select>

      <el-button type="danger" size="small" @click="clearLog">Clear Log</el-button>
    </div>

    <!-- Table -->
    <el-table
      :data="pagedEntries"
      v-loading="loading"
      stripe
      highlight-current-row
      style="width: 100%"
      @row-click="onRowClick"
    >
      <el-table-column label="Timestamp" width="170">
        <template #default="{ row }">
          {{ formatTimestamp(row.timestamp) }}
        </template>
      </el-table-column>

      <el-table-column label="Connection" prop="connectionId" width="150" show-overflow-tooltip />

      <el-table-column label="Database / Collection" width="180">
        <template #default="{ row }">
          {{ dbCollection(row) }}
        </template>
      </el-table-column>

      <el-table-column label="Query">
        <template #default="{ row }">
          <el-tooltip :content="row.queryText" placement="top" :show-after="300">
            <span class="query-text">{{ truncate(row.queryText) }}</span>
          </el-tooltip>
        </template>
      </el-table-column>

      <el-table-column label="Duration (ms)" prop="executionTime" width="120" align="right" />

      <el-table-column label="Rows" prop="rowsAffected" width="80" align="right" />

      <el-table-column label="Status" width="90" align="center">
        <template #default="{ row }">
          <el-tag :type="row.success ? 'success' : 'danger'" size="small">
            {{ row.success ? 'Success' : 'Error' }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <el-pagination
      v-if="filteredEntries.length > pageSize"
      v-model:current-page="currentPage"
      :page-size="pageSize"
      :total="filteredEntries.length"
      layout="prev, pager, next, total"
      style="margin-top: 12px; justify-content: flex-end"
    />

    <!-- Row detail dialog -->
    <el-dialog
      v-model="dialogVisible"
      title="Query Detail"
      width="min(760px, 90vw)"
      destroy-on-close
    >
      <pre v-if="selectedEntry" class="query-detail">{{ selectedEntry.queryText }}</pre>
    </el-dialog>
  </div>
</template>

<style scoped>
.execution-log-tab {
  padding: 12px;
}

.filter-bar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}

.query-text {
  font-family: monospace;
  font-size: 12px;
  cursor: pointer;
}

.query-detail {
  white-space: pre-wrap;
  word-break: break-all;
  font-family: monospace;
  font-size: 13px;
  background: var(--el-fill-color-light);
  padding: 12px;
  border-radius: 4px;
  max-height: 400px;
  overflow-y: auto;
  margin: 0;
}
</style>
