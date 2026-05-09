<template>
  <div class="index-manager">
    <!-- 工具栏 -->
    <div class="toolbar">
      <el-button type="primary" size="small" :loading="loading" @click="openCreateDialog">
        创建索引
      </el-button>
      <el-button size="small" :loading="loading" @click="loadIndexes">刷新</el-button>
    </div>

    <!-- 索引列表 -->
    <el-table :data="indexes" v-loading="loading" size="small" border style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="字段" min-width="200">
        <template #default="{ row }">
          <span class="keys-text">{{ formatKeys(row.keys) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="100" />
      <el-table-column label="唯一性" width="90" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.unique" type="success" size="small">唯一</el-tag>
          <el-tag v-else type="info" size="small">普通</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="稀疏" width="80" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.sparse" type="warning" size="small">稀疏</el-tag>
          <span v-else class="text-muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="90" align="center">
        <template #default="{ row }">
          <el-button
            type="danger"
            size="small"
            :disabled="row.name === '_id_'"
            @click="handleDelete(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建索引对话框 -->
    <el-dialog
      v-model="createDialogVisible"
      title="创建索引"
      width="520px"
      :close-on-click-modal="false"
      @closed="resetForm"
    >
      <el-form :model="form" label-width="90px" size="small">
        <!-- 字段列表 -->
        <el-form-item label="索引字段" required>
          <div class="field-list">
            <div
              v-for="(entry, idx) in form.fields"
              :key="idx"
              class="field-row"
            >
              <el-input
                v-model="entry.field"
                placeholder="字段名"
                style="flex: 1"
              />
              <el-select v-model="entry.direction" style="width: 100px; margin-left: 8px">
                <el-option :value="1" label="升序 (1)" />
                <el-option :value="-1" label="降序 (-1)" />
              </el-select>
              <el-button
                v-if="form.fields.length > 1"
                type="danger"
                :icon="Delete"
                circle
                size="small"
                style="margin-left: 6px"
                @click="removeField(idx)"
              />
            </div>
          </div>
          <el-button size="small" style="margin-top: 6px" @click="addField">
            + 添加字段
          </el-button>
        </el-form-item>

        <!-- 唯一索引 -->
        <el-form-item label="唯一索引">
          <el-checkbox v-model="form.unique">唯一索引</el-checkbox>
        </el-form-item>

        <!-- 稀疏索引 -->
        <el-form-item label="稀疏索引">
          <el-checkbox v-model="form.sparse">稀疏索引</el-checkbox>
        </el-form-item>

        <!-- 索引名称（可选） -->
        <el-form-item label="索引名称">
          <el-input v-model="form.name" placeholder="留空则自动生成" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreate">确认创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import { MongoIndexAPI } from '../../api/mongo'
import type { MongoIndex, MongoIndexSpec } from '../../types/mongo'

// ── Props ──────────────────────────────────────────────────────────────────────

const props = defineProps<{
  profileId: string
  dbName: string
  collName: string
}>()

// ── State ──────────────────────────────────────────────────────────────────────

const indexes = ref<MongoIndex[]>([])
const loading = ref(false)
const createDialogVisible = ref(false)
const submitting = ref(false)

interface FieldEntry {
  field: string
  direction: 1 | -1
}

const defaultForm = () => ({
  fields: [{ field: '', direction: 1 as 1 | -1 }] as FieldEntry[],
  unique: false,
  sparse: false,
  name: '',
})

const form = ref(defaultForm())

// ── Lifecycle ──────────────────────────────────────────────────────────────────

onMounted(loadIndexes)

watch(
  () => [props.profileId, props.dbName, props.collName],
  () => loadIndexes()
)

// ── Helpers ────────────────────────────────────────────────────────────────────

function formatKeys(keys: Record<string, number>): string {
  return Object.entries(keys)
    .map(([k, v]) => `${k}: ${v}`)
    .join(', ')
}

// ── Actions ────────────────────────────────────────────────────────────────────

async function loadIndexes() {
  if (!props.profileId || !props.dbName || !props.collName) return
  loading.value = true
  try {
    indexes.value = await MongoIndexAPI.listIndexes(props.profileId, props.dbName, props.collName)
  } catch (e: any) {
    const msg = e?.message || String(e)
    if (msg.includes('Unauthorized') || msg.includes('not authorized')) {
      ElMessage.warning('当前用户没有查看索引的权限，请使用有足够权限的账号连接')
      indexes.value = []
    }
    // other errors already shown by API layer
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  form.value = defaultForm()
  createDialogVisible.value = true
}

function resetForm() {
  form.value = defaultForm()
}

function addField() {
  form.value.fields.push({ field: '', direction: 1 })
}

function removeField(idx: number) {
  form.value.fields.splice(idx, 1)
}

async function handleCreate() {
  const validFields = form.value.fields.filter(f => f.field.trim())
  if (validFields.length === 0) {
    ElMessage.warning('请至少填写一个字段名')
    return
  }

  const keys: Record<string, number> = {}
  for (const entry of validFields) {
    keys[entry.field.trim()] = entry.direction
  }

  const spec: MongoIndexSpec = {
    keys,
    unique: form.value.unique,
    sparse: form.value.sparse,
  }
  if (form.value.name.trim()) {
    spec.name = form.value.name.trim()
  }

  submitting.value = true
  try {
    const indexName = await MongoIndexAPI.createIndex(props.profileId, props.dbName, props.collName, spec)
    ElMessage.success(`索引 "${indexName}" 创建成功`)
    createDialogVisible.value = false
    await loadIndexes()
  } catch {
    // error shown by API layer
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: MongoIndex) {
  try {
    await ElMessageBox.confirm(
      `确定要删除索引 "${row.name}" 吗？此操作不可撤销。`,
      '删除索引',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger',
      }
    )
  } catch {
    return // user cancelled
  }

  loading.value = true
  try {
    await MongoIndexAPI.dropIndex(props.profileId, props.dbName, props.collName, row.name)
    ElMessage.success(`索引 "${row.name}" 已删除`)
    await loadIndexes()
  } catch {
    // error shown by API layer
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.index-manager {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
}

.toolbar {
  display: flex;
  gap: 8px;
}

.keys-text {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 12px;
}

.field-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
}

.field-row {
  display: flex;
  align-items: center;
}

.text-muted {
  color: #c0c4cc;
}
</style>
