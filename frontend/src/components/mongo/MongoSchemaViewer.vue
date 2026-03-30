<template>
  <div class="schema-viewer">
    <!-- 工具栏 -->
    <div class="toolbar">
      <el-button type="primary" size="small" :loading="loading" @click="handleAnalyze">
        分析 Schema
      </el-button>
      <span v-if="analysis" class="summary-text">
        采样 {{ analysis.sampleSize }} 条 / 共 {{ analysis.totalDocs }} 条文档
      </span>
    </div>

    <!-- 字段列表 -->
    <el-table
      v-if="sortedFields.length > 0"
      :data="sortedFields"
      v-loading="loading"
      size="small"
      border
      style="width: 100%"
    >
      <el-table-column prop="name" label="字段名" min-width="160">
        <template #default="{ row }">
          <span class="field-name">{{ row.name }}</span>
        </template>
      </el-table-column>

      <el-table-column label="出现频率" min-width="180">
        <template #default="{ row }">
          <div class="frequency-cell">
            <el-progress
              :percentage="frequencyPercent(row.frequency)"
              :stroke-width="10"
              :show-text="false"
              style="flex: 1"
            />
            <span class="freq-label">{{ frequencyPercent(row.frequency) }}%</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="类型分布" min-width="220">
        <template #default="{ row }">
          <div class="types-cell">
            <span
              v-for="(count, typeName) in row.types"
              :key="typeName"
              class="type-item"
            >
              <el-tag size="small" type="info">{{ typeName }}</el-tag>
              <span class="type-pct">{{ typePercent(count, row.frequency) }}%</span>
            </span>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- 空状态 -->
    <el-empty
      v-else-if="!loading"
      description="点击「分析 Schema」开始分析集合字段结构"
      :image-size="80"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { MongoSchemaAPI } from '../../api/mongo'
import type { MongoSchemaAnalysis, MongoSchemaField } from '../../types/mongo'

// ── Props ──────────────────────────────────────────────────────────────────────

const props = defineProps<{
  profileId: string
  dbName: string
  collName: string
}>()

// ── State ──────────────────────────────────────────────────────────────────────

const loading = ref(false)
const analysis = ref<MongoSchemaAnalysis | null>(null)

// ── Computed ───────────────────────────────────────────────────────────────────

const sortedFields = computed<MongoSchemaField[]>(() => {
  if (!analysis.value) return []
  return [...analysis.value.fields].sort((a, b) => b.frequency - a.frequency)
})

// ── Helpers ────────────────────────────────────────────────────────────────────

function frequencyPercent(frequency: number): number {
  if (!analysis.value || analysis.value.sampleSize === 0) return 0
  return Math.round((frequency / analysis.value.sampleSize) * 100)
}

function typePercent(count: number, fieldFrequency: number): number {
  if (fieldFrequency === 0) return 0
  return Math.round((count / fieldFrequency) * 100)
}

// ── Actions ────────────────────────────────────────────────────────────────────

async function handleAnalyze() {
  if (!props.profileId || !props.dbName || !props.collName) return
  loading.value = true
  try {
    analysis.value = await MongoSchemaAPI.analyzeSchema(
      props.profileId,
      props.dbName,
      props.collName,
      1000
    )
  } catch {
    // error shown by API layer
  } finally {
    loading.value = false
  }
}

// Reset when collection changes
watch(
  () => [props.profileId, props.dbName, props.collName],
  () => { analysis.value = null }
)
</script>

<style scoped>
.schema-viewer {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.summary-text {
  font-size: 13px;
  color: #606266;
}

.field-name {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 12px;
}

.frequency-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.freq-label {
  font-size: 12px;
  color: #606266;
  min-width: 36px;
  text-align: right;
}

.types-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.type-item {
  display: flex;
  align-items: center;
  gap: 3px;
}

.type-pct {
  font-size: 11px;
  color: #909399;
}
</style>
