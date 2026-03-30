<template>
  <el-dialog
    :model-value="modelValue"
    :title="mode === 'insert' ? '插入文档' : '编辑文档'"
    width="700px"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
    @closed="onDialogClosed"
  >
    <div class="editor-body">
      <!-- JSON 编辑区 -->
      <el-input
        v-model="jsonContent"
        type="textarea"
        :rows="15"
        placeholder='{"key": "value"}'
        class="json-textarea"
        :class="{ 'has-error': jsonError }"
        spellcheck="false"
      />
      <div v-if="jsonError" class="json-error">{{ jsonError }}</div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button size="small" @click="formatJson">格式化</el-button>
        <div class="footer-right">
          <el-button @click="emit('update:modelValue', false)">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="handleSubmit">确认</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { MongoDocumentAPI } from '../../api/mongo'

// ── Props & Emits ──────────────────────────────────────────────────────────────

const props = defineProps<{
  profileId: string
  dbName: string
  collName: string
  mode: 'insert' | 'edit'
  initialDoc?: string
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'saved': []
}>()

// ── State ──────────────────────────────────────────────────────────────────────

const jsonContent = ref('')
const jsonError = ref('')
const submitting = ref(false)

// ── Watchers ───────────────────────────────────────────────────────────────────

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      jsonContent.value = props.initialDoc
        ? formatJsonStr(props.initialDoc)
        : ''
      jsonError.value = ''
    }
  }
)

// ── Helpers ────────────────────────────────────────────────────────────────────

const formatJsonStr = (str: string): string => {
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

const validateJson = (): boolean => {
  const trimmed = jsonContent.value.trim()
  if (!trimmed) {
    jsonError.value = 'JSON 内容不能为空'
    return false
  }
  try {
    JSON.parse(trimmed)
    jsonError.value = ''
    return true
  } catch (e: any) {
    jsonError.value = `JSON 格式错误: ${e.message}`
    return false
  }
}

const extractDocId = (parsed: any): string => {
  const id = parsed._id
  if (id === undefined || id === null) return ''
  if (typeof id === 'object' && id.$oid) return id.$oid
  if (typeof id === 'object') return JSON.stringify(id)
  return String(id)
}

// ── Actions ────────────────────────────────────────────────────────────────────

const formatJson = () => {
  const trimmed = jsonContent.value.trim()
  if (!trimmed) return
  try {
    jsonContent.value = JSON.stringify(JSON.parse(trimmed), null, 2)
    jsonError.value = ''
  } catch (e: any) {
    jsonError.value = `JSON 格式错误: ${e.message}`
  }
}

const handleSubmit = async () => {
  if (!validateJson()) return

  submitting.value = true
  try {
    const trimmed = jsonContent.value.trim()
    if (props.mode === 'insert') {
      await MongoDocumentAPI.insertDocument(props.profileId, props.dbName, props.collName, trimmed)
      ElMessage.success('文档插入成功')
    } else {
      const parsed = JSON.parse(trimmed)
      const docId = extractDocId(parsed)
      await MongoDocumentAPI.updateDocument(props.profileId, props.dbName, props.collName, docId, trimmed)
      ElMessage.success('文档更新成功')
    }
    emit('saved')
    emit('update:modelValue', false)
  } catch {
    // error already shown by API layer; keep dialog open
  } finally {
    submitting.value = false
  }
}

const onDialogClosed = () => {
  jsonContent.value = ''
  jsonError.value = ''
}
</script>

<style scoped>
.editor-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.json-textarea :deep(textarea) {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.5;
  resize: vertical;
}

.json-textarea.has-error :deep(textarea) {
  border-color: #f56c6c;
}

.json-error {
  font-size: 12px;
  color: #f56c6c;
  line-height: 1.4;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.footer-right {
  display: flex;
  gap: 8px;
}
</style>
