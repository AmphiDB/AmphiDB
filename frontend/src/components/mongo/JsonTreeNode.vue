<template>
  <div class="json-node">
    <!-- Object / Array -->
    <template v-if="isObject || isArray">
      <div class="node-line" @click="toggle">
        <span class="toggle-icon">{{ collapsed ? '▶' : '▼' }}</span>
        <span v-if="nodeKey !== undefined" class="node-key">"{{ nodeKey }}"<span class="colon">: </span></span>
        <span class="bracket">{{ isArray ? '[' : '{' }}</span>
        <span v-if="collapsed" class="collapsed-hint">
          {{ isArray ? `${(nodeValue as any[]).length} items` : `${Object.keys(nodeValue as object).length} fields` }}
        </span>
        <span v-if="collapsed" class="bracket">{{ isArray ? ']' : '}' }}</span>
      </div>
      <template v-if="!collapsed">
        <div class="node-children">
          <JsonTreeNode
            v-for="(child, k) in childEntries"
            :key="k"
            :node-key="isArray ? undefined : String(child[0])"
            :node-value="child[1]"
            :depth="depth + 1"
            :default-collapsed-depth="defaultCollapsedDepth"
          />
        </div>
        <div class="node-line close-bracket">
          <span class="bracket">{{ isArray ? ']' : '}' }}</span>
        </div>
      </template>
    </template>

    <!-- Primitive -->
    <template v-else>
      <div class="node-line">
        <span class="toggle-icon" style="visibility:hidden">▶</span>
        <span v-if="nodeKey !== undefined" class="node-key">"{{ nodeKey }}"<span class="colon">: </span></span>
        <span :class="valueClass">{{ displayValue }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

const props = defineProps<{
  nodeKey?: string
  nodeValue: unknown
  depth?: number
  defaultCollapsedDepth?: number  // collapse nodes deeper than this (default 2)
}>()

const depth = computed(() => props.depth ?? 0)
const defaultCollapsedDepth = computed(() => props.defaultCollapsedDepth ?? 2)

const isObject = computed(() =>
  props.nodeValue !== null && typeof props.nodeValue === 'object' && !Array.isArray(props.nodeValue)
)
const isArray = computed(() => Array.isArray(props.nodeValue))

const collapsed = ref(false)

onMounted(() => {
  collapsed.value = depth.value >= defaultCollapsedDepth.value
})

const toggle = () => { collapsed.value = !collapsed.value }

const childEntries = computed(() => {
  if (isArray.value) return (props.nodeValue as unknown[]).map((v, i) => [i, v])
  if (isObject.value) return Object.entries(props.nodeValue as object)
  return []
})

const displayValue = computed(() => {
  if (props.nodeValue === null) return 'null'
  if (typeof props.nodeValue === 'string') return `"${props.nodeValue}"`
  return String(props.nodeValue)
})

const valueClass = computed(() => {
  if (props.nodeValue === null) return 'val-null'
  if (typeof props.nodeValue === 'string') return 'val-string'
  if (typeof props.nodeValue === 'number') return 'val-number'
  if (typeof props.nodeValue === 'boolean') return 'val-bool'
  return 'val-other'
})
</script>

<style scoped>
.json-node {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.7;
}

.node-line {
  display: flex;
  align-items: baseline;
  gap: 2px;
  cursor: default;
  padding-left: 0;
  white-space: nowrap;
}

.node-line:hover { background: #f0f4ff; border-radius: 2px; }

.toggle-icon {
  font-size: 9px;
  color: #909399;
  cursor: pointer;
  width: 14px;
  flex-shrink: 0;
  user-select: none;
}

.node-key { color: #c0392b; }
.colon    { color: #606266; }
.bracket  { color: #606266; font-weight: 600; }

.collapsed-hint {
  font-size: 11px;
  color: #909399;
  margin: 0 4px;
  font-style: italic;
}

.node-children {
  padding-left: 20px;
  border-left: 1px dashed #e4e7ed;
  margin-left: 6px;
}

.close-bracket { padding-left: 0; }

.val-string  { color: #27ae60; }
.val-number  { color: #2980b9; }
.val-bool    { color: #8e44ad; }
.val-null    { color: #95a5a6; font-style: italic; }
.val-other   { color: #303133; }
</style>
