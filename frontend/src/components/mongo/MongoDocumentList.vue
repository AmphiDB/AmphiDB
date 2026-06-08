<template>
  <div class="mongo-document-list">
    <!-- 筛选条件区 -->
    <div class="filter-area">
      <div class="filter-header">
        <span class="filter-label">Filter</span>
        <el-radio-group v-model="filterMode" size="small" class="mode-toggle">
          <el-radio-button value="visual">傻瓜模式</el-radio-button>
          <el-radio-button value="json">JSON 模式</el-radio-button>
        </el-radio-group>
        <el-button v-if="filterMode === 'visual'" :icon="Plus" size="small" type="primary" plain @click="addCondition">添加条件</el-button>
        <el-button v-if="filterMode === 'visual' && conditions.length > 0" size="small" @click="clearConditions">清空</el-button>
      </div>

      <div v-if="filterMode === 'visual'" class="conditions-list">
        <div v-if="conditions.length === 0" class="conditions-empty">点击「添加条件」添加筛选条件，留空则查询全部文档</div>
        <div v-for="(cond, idx) in conditions" :key="cond.id" class="condition-row">
          <span class="cond-logic">{{ idx === 0 ? 'WHERE' : 'AND' }}</span>
          <el-autocomplete v-model="cond.field" :fetch-suggestions="fetchFieldSuggestions" placeholder="字段名 (如 data.goods.title)" size="small" class="cond-field" clearable @select="(item: SuggestItem) => { cond.field = item.value }">
            <template #default="{ item }"><span class="suggest-field">{{ item.value }}</span><span class="suggest-type">{{ item.type }}</span></template>
          </el-autocomplete>
          <el-select v-model="cond.op" size="small" class="cond-op">
            <el-option value="eq"     label="= 等于" />
            <el-option value="ne"     label="≠ 不等于" />
            <el-option value="like"   label="≈ 模糊匹配" />
            <el-option value="gt"     label="> 大于" />
            <el-option value="gte"    label="≥ 大于等于" />
            <el-option value="lt"     label="< 小于" />
            <el-option value="lte"    label="≤ 小于等于" />
            <el-option value="in"     label="∈ 包含(逗号分隔)" />
            <el-option value="exists" label="∃ 字段存在" />
          </el-select>
          <el-input v-if="cond.op !== 'exists'" v-model="cond.value" size="small" class="cond-value" :placeholder="cond.op === 'in' ? '值1,值2,值3' : cond.op === 'like' ? '关键词' : '值'" />
          <el-button :icon="Delete" size="small" type="danger" plain circle @click="removeCondition(idx)" />
        </div>
        <div v-if="conditions.length > 0" class="filter-preview">
          <span class="preview-label">生成 Filter：</span>
          <code class="preview-json">{{ filterInput || '{}' }}</code>
        </div>
      </div>

      <div v-else class="json-mode">
        <el-input v-model="filterInput" type="textarea" :rows="3" placeholder='{"field": "value"}' @blur="onJsonFilterBlur" />
        <span v-if="filterError" class="json-error">{{ filterError }}</span>
      </div>

      <el-row :gutter="12" class="sort-proj-row">
        <el-col :span="12">
          <div class="filter-item">
            <label class="filter-label">Sort (JSON)</label>
            <el-autocomplete v-model="sortInput" :fetch-suggestions="fetchFieldSuggestions" placeholder='{"field": 1}' clearable size="small" style="width:100%" @blur="validateSort" @clear="onSortClear" @select="(item: SuggestItem) => onSuggestAppend(item, 'sort')">
              <template #default="{ item }"><span class="suggest-field">{{ item.value }}</span><span class="suggest-type">{{ item.type }}</span></template>
            </el-autocomplete>
            <span v-if="sortError" class="json-error">{{ sortError }}</span>
          </div>
        </el-col>
        <el-col :span="12">
          <div class="filter-item">
            <label class="filter-label">Projection (JSON)</label>
            <el-autocomplete v-model="projectionInput" :fetch-suggestions="fetchFieldSuggestions" placeholder='{"field": 1}' clearable size="small" style="width:100%" @blur="validateProjection" @clear="onProjectionClear" @select="(item: SuggestItem) => onSuggestAppend(item, 'projection')">
              <template #default="{ item }"><span class="suggest-field">{{ item.value }}</span><span class="suggest-type">{{ item.type }}</span></template>
            </el-autocomplete>
            <span v-if="projectionError" class="json-error">{{ projectionError }}</span>
          </div>
        </el-col>
      </el-row>

      <div class="filter-actions">
        <el-button type="primary" size="small" @click="onSearch">查询</el-button>
        <el-button size="small" @click="onReset">重置</el-button>
      </div>
    </div>

    <!-- 操作按钮栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-button :icon="Refresh" size="small" @click="loadDocuments" :loading="loading">刷新</el-button>
        <el-button :icon="Plus" size="small" type="primary" @click="emit('insert-document')">插入文档</el-button>
        <el-button :icon="Edit" size="small" :disabled="selectedDocs.length !== 1" @click="handleEdit">编辑</el-button>
        <el-button :icon="Delete" size="small" type="danger" :disabled="selectedDocs.length === 0" @click="handleDelete">删除 {{ selectedDocs.length > 0 ? `(${selectedDocs.length})` : '' }}</el-button>
        <el-button :icon="Download" size="small" @click="emit('export')">导出</el-button>
      </div>
      <div class="toolbar-right">
        <span class="doc-count">共 {{ total }} 条文档</span>
      </div>
    </div>

    <!-- 文档表格 -->
    <div class="table-wrapper">
      <el-table ref="tableRef" v-loading="loading" :data="tableData" border stripe size="small" style="width:100%" @selection-change="handleSelectionChange" row-key="_rowIndex">
        <el-table-column type="selection" width="40" fixed="left" />
        <el-table-column label="_id" prop="_id" width="200" fixed="left" show-overflow-tooltip>
          <template #default="{ row }"><span class="cell-value">{{ truncate(row['_id']) }}</span></template>
        </el-table-column>
        <el-table-column v-for="col in inferredColumns" :key="col" :label="col" :prop="col" min-width="140" show-overflow-tooltip>
          <template #default="{ row }"><span class="cell-value">{{ truncate(row[col]) }}</span></template>
        </el-table-column>
        <el-table-column label="查看" width="60" fixed="right">
          <template #default="{ row }">
            <el-button link size="small" type="primary" @click.stop="openDocDialog(row)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分页 -->
    <div class="pagination-area">
      <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50, 100, 200]" layout="total, sizes, prev, pager, next, jumper" background @current-change="loadDocuments" @size-change="onPageSizeChange" />
    </div>

    <!-- 文档详情 Dialog -->
    <el-dialog v-model="docDialogVisible" title="文档详情" width="min(800px, 92vw)" :close-on-click-modal="true" destroy-on-close>
      <div class="doc-dialog-toolbar">
        <span class="doc-dialog-hint">点击 ▶/▼ 折叠/展开节点</span>
        <div class="doc-dialog-actions">
          <el-button size="small" @click="copyDocJson">复制 JSON</el-button>
          <el-button size="small" @click="expandAll">全部展开</el-button>
          <el-button size="small" @click="collapseAll">全部折叠</el-button>
        </div>
      </div>
      <div class="doc-tree-wrap">
        <JsonTreeNode v-if="dialogDocParsed !== null" :node-value="dialogDocParsed" :depth="0" :default-collapsed-depth="dialogCollapseDepth" :key="dialogTreeKey" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Plus, Edit, Delete, Download } from '@element-plus/icons-vue'
import { MongoDocumentAPI } from '../../api/mongo'
import type { MongoQueryParams } from '../../types/mongo'
import JsonTreeNode from './JsonTreeNode.vue'

const props = defineProps<{ profileId: string; dbName: string; collName: string }>()
const emit = defineEmits<{ 'insert-document': []; 'edit-document': [doc: string]; 'export': [] }>()

// ── State ──────────────────────────────────────────────────────────────────────
const loading = ref(false)
const rawDocs = ref<string[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(50)
const filterMode = ref<'visual' | 'json'>('visual')
const filterInput = ref('')
const sortInput = ref('')
const projectionInput = ref('')
const filterError = ref('')
const sortError = ref('')
const projectionError = ref('')
const selectedDocs = ref<any[]>([])
const tableRef = ref()

// ── Document detail dialog ─────────────────────────────────────────────────────
const docDialogVisible = ref(false)
const dialogDocParsed = ref<unknown>(null)
const dialogCollapseDepth = ref(2)
const dialogTreeKey = ref(0)

const openDocDialog = (row: TableRow) => {
  try { dialogDocParsed.value = JSON.parse(row._raw) } catch { dialogDocParsed.value = row._raw }
  dialogCollapseDepth.value = 2
  dialogTreeKey.value++
  docDialogVisible.value = true
}

const copyDocJson = async () => {
  if (dialogDocParsed.value === null) return
  try {
    const jsonStr = JSON.stringify(dialogDocParsed.value, null, 2)
    await navigator.clipboard.writeText(jsonStr)
    ElMessage.success('JSON 已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

const expandAll = () => { dialogCollapseDepth.value = 999; dialogTreeKey.value++ }
const collapseAll = () => { dialogCollapseDepth.value = 0; dialogTreeKey.value++ }

// ── Visual filter conditions ───────────────────────────────────────────────────
type FilterOp = 'eq' | 'ne' | 'like' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'exists'
interface Condition { id: number; field: string; op: FilterOp; value: string }
let condIdSeq = 0
const conditions = ref<Condition[]>([])

const addCondition = () => conditions.value.push({ id: ++condIdSeq, field: '', op: 'eq', value: '' })
const removeCondition = (idx: number) => { conditions.value.splice(idx, 1); syncConditionsToJson() }
const clearConditions = () => { conditions.value = []; filterInput.value = '' }

const parseValue = (v: string): any => {
  if (v === 'null') return null
  if (v === 'true') return true
  if (v === 'false') return false
  const n = Number(v)
  if (v !== '' && !isNaN(n)) return n
  return v
}

const buildFilterFromConditions = (): string => {
  const valid = conditions.value.filter(c => c.field.trim())
  if (valid.length === 0) return '{}'
  const obj: Record<string, any> = {}
  for (const c of valid) {
    const f = c.field.trim()
    if (c.op === 'eq')     obj[f] = parseValue(c.value)
    else if (c.op === 'ne')   obj[f] = { $ne: parseValue(c.value) }
    else if (c.op === 'like') obj[f] = { $regex: c.value, $options: 'i' }
    else if (c.op === 'gt')   obj[f] = { $gt: parseValue(c.value) }
    else if (c.op === 'gte')  obj[f] = { $gte: parseValue(c.value) }
    else if (c.op === 'lt')   obj[f] = { $lt: parseValue(c.value) }
    else if (c.op === 'lte')  obj[f] = { $lte: parseValue(c.value) }
    else if (c.op === 'in')   obj[f] = { $in: c.value.split(',').map(s => parseValue(s.trim())) }
    else if (c.op === 'exists') obj[f] = { $exists: true }
  }
  return JSON.stringify(obj)
}

const syncConditionsToJson = () => { filterInput.value = buildFilterFromConditions() }
watch(conditions, syncConditionsToJson, { deep: true })
watch(filterMode, mode => { if (mode === 'visual') tryParseJsonToConditions() })

const tryParseJsonToConditions = () => {
  const raw = filterInput.value.trim()
  if (!raw || raw === '{}') { conditions.value = []; return }
  try {
    const obj = JSON.parse(raw)
    const parsed: Condition[] = []
    for (const [field, val] of Object.entries(obj)) {
      if (val && typeof val === 'object' && !Array.isArray(val)) {
        const v = val as Record<string, any>
        if ('$regex' in v)  parsed.push({ id: ++condIdSeq, field, op: 'like',   value: String(v.$regex) })
        else if ('$ne'  in v) parsed.push({ id: ++condIdSeq, field, op: 'ne',   value: String(v.$ne) })
        else if ('$gt'  in v) parsed.push({ id: ++condIdSeq, field, op: 'gt',   value: String(v.$gt) })
        else if ('$gte' in v) parsed.push({ id: ++condIdSeq, field, op: 'gte',  value: String(v.$gte) })
        else if ('$lt'  in v) parsed.push({ id: ++condIdSeq, field, op: 'lt',   value: String(v.$lt) })
        else if ('$lte' in v) parsed.push({ id: ++condIdSeq, field, op: 'lte',  value: String(v.$lte) })
        else if ('$in'  in v && Array.isArray(v.$in)) parsed.push({ id: ++condIdSeq, field, op: 'in', value: v.$in.join(',') })
        else if ('$exists' in v) parsed.push({ id: ++condIdSeq, field, op: 'exists', value: '' })
      } else {
        parsed.push({ id: ++condIdSeq, field, op: 'eq', value: val === null ? 'null' : String(val) })
      }
    }
    conditions.value = parsed
  } catch { /* leave as-is */ }
}

// ── Field suggestions ──────────────────────────────────────────────────────────
interface SuggestItem { value: string; type: string }

const knownFields = computed<SuggestItem[]>(() => {
  const freq: Record<string, number> = {}
  const typeMap: Record<string, string> = {}
  const collect = (obj: any, prefix = '') => {
    if (!obj || typeof obj !== 'object' || Array.isArray(obj)) return
    for (const [k, v] of Object.entries(obj)) {
      const path = prefix ? `${prefix}.${k}` : k
      freq[path] = (freq[path] || 0) + 1
      if (!typeMap[path]) typeMap[path] = v === null ? 'null' : Array.isArray(v) ? 'array' : typeof v === 'object' ? 'object' : typeof v
      if (v && typeof v === 'object' && !Array.isArray(v)) collect(v, path)
    }
  }
  for (const s of rawDocs.value) { try { collect(JSON.parse(s)) } catch { /* skip */ } }
  return Object.entries(freq).sort((a, b) => b[1] - a[1]).map(([k]) => ({ value: k, type: typeMap[k] || '' }))
})

const fetchFieldSuggestions = (q: string, cb: (items: SuggestItem[]) => void) => {
  const frag = (q.match(/([\w.]*)$/) || ['', ''])[1].toLowerCase()
  cb(frag ? knownFields.value.filter(f => f.value.toLowerCase().includes(frag)) : knownFields.value)
}

const onSuggestAppend = (item: SuggestItem, target: 'sort' | 'projection') => {
  const cur = (target === 'sort' ? sortInput.value : projectionInput.value).trim()
  const next = (!cur || cur === '{}') ? `{"${item.value}": 1}` : cur.replace(/\s*}$/, `, "${item.value}": 1}`)
  if (target === 'sort') sortInput.value = next
  else projectionInput.value = next
}

// ── Parsed documents ───────────────────────────────────────────────────────────
interface TableRow { _rowIndex: number; _raw: string; [key: string]: any }

const parsedDocs = computed<TableRow[]>(() =>
  rawDocs.value.map((s, idx) => {
    try {
      const obj = JSON.parse(s)
      if (obj._id && typeof obj._id === 'object') obj._id = obj._id.$oid || JSON.stringify(obj._id)
      return { ...obj, _rowIndex: idx, _raw: s }
    } catch { return { _rowIndex: idx, _raw: s, _id: `[parse error] ${s.slice(0, 40)}` } }
  })
)

const inferredColumns = computed<string[]>(() => {
  const freq: Record<string, number> = {}
  for (const row of parsedDocs.value)
    for (const k of Object.keys(row))
      if (k !== '_id' && k !== '_rowIndex' && k !== '_raw') freq[k] = (freq[k] || 0) + 1
  return Object.entries(freq).sort((a, b) => b[1] - a[1]).slice(0, 6).map(([k]) => k)
})

const tableData = computed(() => parsedDocs.value)

// ── Helpers ────────────────────────────────────────────────────────────────────
const truncate = (val: any, max = 50): string => {
  if (val === undefined || val === null) return ''
  const s = typeof val === 'object' ? JSON.stringify(val) : String(val)
  return s.length > max ? s.slice(0, max) + '…' : s
}

const validateJson = (v: string) => { if (!v.trim()) return ''; try { JSON.parse(v); return '' } catch (e: any) { return `JSON 格式错误: ${e.message}` } }
const validateSort = () => { sortError.value = validateJson(sortInput.value) }
const validateProjection = () => { projectionError.value = validateJson(projectionInput.value) }
const onSortClear = () => { sortError.value = ''; loadDocuments() }
const onProjectionClear = () => { projectionError.value = ''; loadDocuments() }
const onJsonFilterBlur = () => { filterError.value = validateJson(filterInput.value) }

// ── Search / Reset ─────────────────────────────────────────────────────────────
const onSearch = () => { currentPage.value = 1; loadDocuments() }
const onReset = () => {
  conditions.value = []; filterInput.value = ''; sortInput.value = ''; projectionInput.value = ''
  filterError.value = ''; sortError.value = ''; projectionError.value = ''
  currentPage.value = 1; loadDocuments()
}

// ── Data loading ───────────────────────────────────────────────────────────────
const loadDocuments = async () => {
  validateSort(); validateProjection()
  if (filterError.value || sortError.value || projectionError.value) { ElMessage.warning('请修正 JSON 格式错误后再查询'); return }
  loading.value = true; selectedDocs.value = []
  try {
    const params: MongoQueryParams = {
      database: props.dbName, collection: props.collName,
      filter: filterInput.value.trim() || '{}',
      sort: sortInput.value.trim() || '{}',
      projection: projectionInput.value.trim() || '{}',
      page: currentPage.value, pageSize: pageSize.value,
    }
    const result = await MongoDocumentAPI.queryDocuments(props.profileId, params)
    rawDocs.value = result.documents || []; total.value = result.total; currentPage.value = result.page
  } catch { /* shown by API */ } finally { loading.value = false }
}

const onPageSizeChange = () => { currentPage.value = 1; loadDocuments() }

// ── Selection & actions ────────────────────────────────────────────────────────
const handleSelectionChange = (rows: any[]) => { selectedDocs.value = rows }

const handleEdit = () => {
  if (selectedDocs.value.length !== 1) return
  emit('edit-document', (selectedDocs.value[0] as TableRow)._raw)
}

const handleDelete = async () => {
  if (selectedDocs.value.length === 0) return
  try { await ElMessageBox.confirm(`确定要删除选中的 ${selectedDocs.value.length} 条文档吗？此操作不可恢复！`, '删除文档', { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }) } catch { return }
  const docIds = selectedDocs.value.map((row: TableRow) => { const raw = JSON.parse(row._raw); const id = raw._id; return (id && typeof id === 'object' && id.$oid) ? id.$oid : String(id) })
  try { const deleted = await MongoDocumentAPI.deleteDocuments(props.profileId, props.dbName, props.collName, docIds); ElMessage.success(`已删除 ${deleted} 条文档`); await loadDocuments() } catch { /* shown */ }
}

// ── Watchers ───────────────────────────────────────────────────────────────────
watch(() => [props.profileId, props.dbName, props.collName], () => {
  currentPage.value = 1; conditions.value = []; filterInput.value = ''; sortInput.value = ''; projectionInput.value = ''
  filterError.value = ''; sortError.value = ''; projectionError.value = ''; loadDocuments()
})

onMounted(() => { if (props.profileId && props.dbName && props.collName) loadDocuments() })

</script>

<style scoped>
.mongo-document-list { display: flex; flex-direction: column; height: 100%; background: #fff; }

.filter-area { padding: 10px 16px 8px; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; display: flex; flex-direction: column; gap: 8px; }
.filter-header { display: flex; align-items: center; gap: 8px; }
.filter-label { font-size: 12px; color: #606266; font-weight: 600; white-space: nowrap; }
.mode-toggle { flex-shrink: 0; }
.conditions-list { display: flex; flex-direction: column; gap: 6px; }
.conditions-empty { font-size: 12px; color: #c0c4cc; padding: 4px 0; }
.condition-row { display: flex; align-items: center; gap: 6px; }
.cond-logic { font-size: 11px; color: #909399; width: 40px; text-align: right; flex-shrink: 0; font-weight: 600; }
.cond-field { width: 200px; flex-shrink: 0; }
.cond-op { width: 150px; flex-shrink: 0; }
.cond-value { flex: 1; min-width: 100px; }
.filter-preview { display: flex; align-items: center; gap: 6px; padding: 4px 6px; background: #f5f7fa; border-radius: 4px; font-size: 11px; }
.preview-label { color: #909399; white-space: nowrap; flex-shrink: 0; }
.preview-json { font-family: 'Menlo','Monaco','Consolas',monospace; color: #303133; word-break: break-all; white-space: pre-wrap; }
.json-mode { display: flex; flex-direction: column; gap: 4px; }
.json-error { font-size: 11px; color: #f56c6c; }
.sort-proj-row { margin-top: 2px; }
.filter-item { display: flex; flex-direction: column; gap: 4px; }
.filter-actions { display: flex; gap: 8px; }

.toolbar { display: flex; align-items: center; justify-content: space-between; padding: 8px 16px; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; gap: 8px; }
.toolbar-left { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.toolbar-right { flex-shrink: 0; }
.doc-count { font-size: 13px; color: #909399; }

.table-wrapper { flex: 1; overflow: auto; padding: 0 16px; }
.cell-value { font-size: 12px; color: #303133; font-family: 'Menlo','Monaco','Consolas',monospace; }

.pagination-area { padding: 12px 16px; border-top: 1px solid #e4e7ed; display: flex; justify-content: flex-end; flex-shrink: 0; }

/* ── Doc dialog ── */
.doc-dialog-toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.doc-dialog-hint { font-size: 12px; color: #909399; }
.doc-dialog-actions { display: flex; gap: 6px; }
.doc-tree-wrap { max-height: 60vh; overflow-y: auto; padding: 8px 12px; background: #fafafa; border: 1px solid #e4e7ed; border-radius: 4px; }

/* ── Suggestion dropdown ── */
.suggest-field { font-family: 'Menlo','Monaco','Consolas',monospace; font-size: 13px; color: #303133; }
.suggest-type { float: right; font-size: 11px; color: #909399; margin-left: 8px; }
</style>
