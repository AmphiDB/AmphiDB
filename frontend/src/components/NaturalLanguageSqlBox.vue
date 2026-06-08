<template>
  <div class="nl-sql-box">
    <div class="nl-main">
      <el-input
        v-model="prompt"
        :disabled="disabled || generating"
        :placeholder="placeholder"
        clearable
        @keyup.enter="generate"
      >
        <template #prefix>
          <el-icon><MagicStick /></el-icon>
        </template>
      </el-input>
      <el-button
        type="primary"
        plain
        :loading="generating"
        :disabled="disabled || !prompt.trim()"
        @click="generate"
      >
        生成 SQL
      </el-button>
    </div>
    <div v-if="lastExplanation" class="nl-explanation">{{ lastExplanation }}</div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { MagicStick } from '@element-plus/icons-vue'
import { LLMAPI } from '../api/llm'

const props = withDefaults(defineProps<{
  profileId: string
  database: string
  currentTable?: string
  allowWrite?: boolean
  disabled?: boolean
  placeholder?: string
}>(), {
  currentTable: '',
  allowWrite: false,
  disabled: false,
  placeholder: '用白话描述要查询的数据，例如：最近 20 个下单用户',
})

const emit = defineEmits<{
  (e: 'generated', sql: string): void
}>()

const prompt = ref('')
const generating = ref(false)
const lastExplanation = ref('')

const generate = async () => {
  const text = prompt.value.trim()
  if (!text) return
  if (!props.profileId) {
    ElMessage.warning('请先选择连接')
    return
  }
  if (!props.database) {
    ElMessage.warning('请先选择数据库')
    return
  }

  generating.value = true
  lastExplanation.value = ''
  try {
    const res = await LLMAPI.generateSQL({
      prompt: text,
      profileId: props.profileId,
      database: props.database,
      currentTable: props.currentTable,
      allowWrite: props.allowWrite,
    })
    emit('generated', res.sql)
    lastExplanation.value = res.explanation || `由 ${res.model} 生成，已填入 SQL 输入框`
  } catch (error: any) {
    ElMessage.error(`生成 SQL 失败: ${error?.message || error}`)
  } finally {
    generating.value = false
  }
}
</script>

<style scoped>
.nl-sql-box {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.nl-main {
  display: flex;
  gap: 8px;
  align-items: center;
}

.nl-main :deep(.el-input) {
  flex: 1;
}

.nl-explanation {
  color: #606266;
  font-size: 12px;
  line-height: 1.4;
}

@media (max-width: 720px) {
  .nl-main {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
