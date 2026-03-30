<template>
  <div class="query-editor">
    <!-- No connection -->
    <div v-if="!connStore.currentProfileId" class="no-connection">
      <el-empty description="请先连接到 MongoDB">
        <el-button type="primary" @click="router.push('/mongo/connections')">
          前往连接管理
        </el-button>
      </el-empty>
    </div>

    <template v-else>
      <div class="editor-layout">
        <!-- Main area -->
        <div class="editor-main">
          <!-- Top bar: db + collection inputs -->
          <div class="editor-topbar">
            <span class="topbar-label">数据库</span>
            <el-input
              v-model="dbName"
              placeholder="数据库名称"
              class="topbar-input"
              clearable
            />
            <span class="topbar-label">集合</span>
            <el-input
              v-model="collName"
              placeholder="集合名称"
              class="topbar-input"
              clearable
            />
            <el-button
              type="primary"
              :loading="running"
              :disabled="!dbName.trim() || !collName.trim()"
              @click="runQuery"
            >
              执行
            </el-button>
            <span v-if="execTime !== null" class="exec-time">
              执行耗时: {{ execTime }}ms
            </span>
          </div>

          <!-- Pipeline editor -->
          <div class="pipeline-area">
            <el-input
              v-model="pipeline"
              type="textarea"
              :rows="10"
              placeholder='输入聚合 Pipeline，例如：[{"$match": {}}, {"$limit": 10}]'
              class="pipeline-input"
              resize="vertical"
            />
          </div>

          <!-- Results area -->
          <div class="results-area">
            <div class="results-header">
              <span class="results-title">
                查询结果
                <span v-if="resultDocs.length > 0" class="results-count">
                  ({{ resultDocs.length }} 条)
                </span>
              </span>
              <el-radio-group v-model="viewMode" size="small">
                <el-radio-button value="table">表格视图</el-radio-button>
                <el-radio-button value="json">JSON 视图</el-radio-button>
              </el-radio-group>
            </div>

            <div v-if="resultDocs.length === 0 && !running" class="results-empty">
              <el-empty description="暂无结果" :image-size="60" />
            </div>

            <!-- Table view -->
            <div v-else-if="viewMode === 'table' && resultDocs.length > 0" class="results-table">
              <el-table
                :data="parsedDocs"
                border
                stripe
                size="small"
                max-height="360"
                style="width: 100%"
              >
                <el-table-column
                  v-for="col in inferredColumns"
                  :key="col"
                  :prop="col"
                  :label="col"
                  min-width="120"
                  show-overflow-tooltip
                >
                  <template #default="{ row }">
                    {{ formatCell(row[col]) }}
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <!-- JSON view -->
            <div v-else-if="viewMode === 'json' && resultDocs.length > 0" class="results-json">
              <div
                v-for="(doc, idx) in resultDocs"
                :key="idx"
                class="json-doc"
              >
                <div class="json-doc-index"># {{ idx + 1 }}</div>
                <pre class="json-pre">{{ formatJson(doc) }}</pre>
              </div>
            </div>
          </div>
        </div>

        <!-- History sidebar -->
        <div class="history-sidebar" :class="{ collapsed: historyCollapsed }">
          <div class="history-header" @click="historyCollapsed = !historyCollapsed">
            <span class="history-title">历史记录</span>
            <el-icon class="history-toggle">
              <ArrowRight v-if="historyCollapsed" />
              <ArrowLeft v-else />
            </el-icon>
          </div>

          <template v-if="!historyCollapsed">
            <div v-if="history.length === 0" class="history-empty">
              暂无历史记录
            </div>
            <div
              v-for="(item, idx) in history"
              :key="idx"
              class="history-item"
              :title="item"
              @click="fillFromHistory(item)"
            >
              <span class="history-index">{{ idx + 1 }}</span>
              <span class="history-text">{{ truncate(item, 60) }}</span>
            </div>
          </template>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { useMongoConnectionStore } from '../../stores/mongoConnection'
import { MongoQueryAPI } from '../../api/mongo'

const router = useRouter()
const connStore = useMongoConnectionStore()

// Inputs
const dbName = ref('')
const collName = ref('')
const pipeline = ref('[\n  { "$match": {} }\n]')

// Execution state
const running = ref(false)
const execTime = ref<number | null>(null)

// Results
const resultDocs = ref<string[]>([])
const viewMode = ref<'table' | 'json'>('table')

// History (local, last 20)
const history = ref<string[]>([])
const historyCollapsed = ref(false)

// Infer columns from parsed documents
const parsedDocs = computed<Record<string, unknown>[]>(() => {
  return resultDocs.value.map(d => {
    try { return JSON.parse(d) } catch { return {} }
  })
})

const inferredColumns = computed<string[]>(() => {
  const freq: Record<string, number> = {}
  for (const doc of parsedDocs.value) {
    for (const key of Object.keys(doc)) {
      freq[key] = (freq[key] ?? 0) + 1
    }
  }
  // Sort by frequency desc, take top 20
  return Object.entries(freq)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 20)
    .map(([k]) => k)
})

// Run aggregation
async function runQuery() {
  const profileId = connStore.currentProfileId
  if (!profileId) return

  const pipelineStr = pipeline.value.trim()
  if (!pipelineStr) {
    ElMessage.warning('请输入聚合 Pipeline')
    return
  }

  // Validate JSON
  try {
    const parsed = JSON.parse(pipelineStr)
    if (!Array.isArray(parsed)) {
      ElMessage.error('Pipeline 必须是一个 JSON 数组')
      return
    }
  } catch {
    ElMessage.error('Pipeline JSON 格式错误')
    return
  }

  running.value = true
  execTime.value = null
  resultDocs.value = []

  try {
    const result = await MongoQueryAPI.runAggregation(
      profileId,
      dbName.value.trim(),
      collName.value.trim(),
      pipelineStr
    )

    resultDocs.value = result.documents ?? []
    execTime.value = result.executionTime

    // Add to history (front, deduplicate, max 20)
    const trimmed = pipelineStr
    history.value = [trimmed, ...history.value.filter(h => h !== trimmed)].slice(0, 20)
  } catch {
    // error already shown by API layer
  } finally {
    running.value = false
  }
}

function fillFromHistory(item: string) {
  pipeline.value = item
}

function formatJson(jsonStr: string): string {
  try {
    return JSON.stringify(JSON.parse(jsonStr), null, 2)
  } catch {
    return jsonStr
  }
}

function formatCell(val: unknown): string {
  if (val === null || val === undefined) return ''
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}

function truncate(str: string, len: number): string {
  const single = str.replace(/\s+/g, ' ').trim()
  return single.length > len ? single.slice(0, len) + '…' : single
}
</script>

<style scoped>
.query-editor {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: #f5f7fa;
}

.no-connection {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ── Layout ── */
.editor-layout {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.editor-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 12px;
  gap: 10px;
}

/* ── Top bar ── */
.editor-topbar {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff;
  padding: 10px 12px;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.topbar-label {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
}

.topbar-input {
  width: 180px;
}

.exec-time {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
  margin-left: 4px;
}

/* ── Pipeline area ── */
.pipeline-area {
  background: #fff;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  padding: 10px;
  flex-shrink: 0;
}

.pipeline-input :deep(.el-textarea__inner) {
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.5;
}

/* ── Results ── */
.results-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  overflow: hidden;
}

.results-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.results-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.results-count {
  font-weight: 400;
  color: #909399;
  margin-left: 4px;
}

.results-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.results-table {
  flex: 1;
  overflow: auto;
  padding: 8px;
}

.results-json {
  flex: 1;
  overflow: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.json-doc {
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  overflow: hidden;
}

.json-doc-index {
  background: #f5f7fa;
  padding: 2px 8px;
  font-size: 11px;
  color: #909399;
  border-bottom: 1px solid #e4e7ed;
}

.json-pre {
  margin: 0;
  padding: 8px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  color: #303133;
  background: #fff;
}

/* ── History sidebar ── */
.history-sidebar {
  width: 240px;
  flex-shrink: 0;
  background: #fff;
  border-left: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: width 0.2s;
}

.history-sidebar.collapsed {
  width: 36px;
}

.history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 10px;
  border-bottom: 1px solid #e4e7ed;
  cursor: pointer;
  user-select: none;
  flex-shrink: 0;
}

.history-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
}

.history-sidebar.collapsed .history-title {
  display: none;
}

.history-toggle {
  font-size: 14px;
  color: #909399;
  flex-shrink: 0;
}

.history-empty {
  padding: 16px 10px;
  font-size: 12px;
  color: #c0c4cc;
  text-align: center;
}

.history-item {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 10px;
  cursor: pointer;
  border-bottom: 1px solid #f0f2f5;
  transition: background 0.15s;
}

.history-item:hover {
  background: #f5f7fa;
}

.history-index {
  font-size: 11px;
  color: #c0c4cc;
  flex-shrink: 0;
  padding-top: 1px;
}

.history-text {
  font-size: 12px;
  color: #606266;
  font-family: 'Courier New', Courier, monospace;
  word-break: break-all;
  line-height: 1.4;
}
</style>
