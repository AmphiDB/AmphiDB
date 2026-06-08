<template>
  <div class="schema-viewer">
    <el-card v-loading="loading" shadow="never">
      <template #header>
        <div class="card-header">
          <div class="table-title">
            <el-icon><Document /></el-icon>
            <span>{{ schema?.name || '表结构' }}</span>
          </div>
          <div v-if="schema" class="header-actions">
            <el-tooltip content="复制表名" placement="bottom">
              <el-button
                :icon="CopyDocument"
                size="small"
                text
                @click="copyTableName"
              >
                表名
              </el-button>
            </el-tooltip>
            <el-tooltip content="复制 CREATE TABLE DDL" placement="bottom">
              <el-button
                :icon="CopyDocument"
                size="small"
                type="primary"
                plain
                @click="copyDDL"
              >
                复制 DDL
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </template>

      <el-empty v-if="!schema && !loading" description="请选择一个表查看结构" />

      <div v-else-if="schema" class="schema-content">
        <div class="summary-strip">
          <div
            v-for="item in summaryItems"
            :key="item.label"
            class="summary-item"
          >
            <span class="summary-label">{{ item.label }}</span>
            <span class="summary-value">{{ item.value }}</span>
          </div>
        </div>

        <!-- 表信息 -->
        <div class="info-section">
          <div class="section-header">
            <h4>表信息</h4>
            <el-button
              :icon="CopyDocument"
              size="small"
              text
              @click="copyTableName"
            >
              复制表名
            </el-button>
          </div>
          <el-descriptions :column="3" size="small">
            <el-descriptions-item label="表名">
              <span class="table-name-text">{{ schema.name }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="引擎">{{ schema.engine }}</el-descriptions-item>
            <el-descriptions-item label="字符集">{{ schema.charset }}</el-descriptions-item>
            <el-descriptions-item label="注释" :span="3">
              {{ schema.comment || '无' }}
            </el-descriptions-item>
          </el-descriptions>
        </div>

        <!-- 列信息 -->
        <div class="info-section">
          <div class="section-header">
            <div class="section-title">
              <h4>列信息</h4>
              <span class="section-count">
                {{ filteredColumns.length }} / {{ schema.columns.length }}
              </span>
            </div>
            <el-input
              v-model="columnKeyword"
              class="column-search"
              placeholder="筛选列名、类型、注释"
              clearable
              size="small"
              :prefix-icon="Search"
            />
          </div>
          <el-table
            :data="filteredColumns"
            stripe
            size="small"
            class="schema-table"
            empty-text="没有匹配的列"
          >
            <el-table-column prop="name" label="列名" width="180" />
            <el-table-column prop="type" label="数据类型" width="150" />
            <el-table-column label="允许NULL" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="row.nullable ? 'info' : 'success'" size="small">
                  {{ row.nullable ? '是' : '否' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="默认值" width="150">
              <template #default="{ row }">
                <span v-if="row.defaultValue !== undefined && row.defaultValue !== null">
                  {{ row.defaultValue }}
                </span>
                <span v-else class="text-muted">无</span>
              </template>
            </el-table-column>
            <el-table-column label="自增" width="80" align="center">
              <template #default="{ row }">
                <el-icon v-if="row.autoIncrement" color="#67c23a"><Check /></el-icon>
              </template>
            </el-table-column>
            <el-table-column prop="comment" label="注释" min-width="200" show-overflow-tooltip />
          </el-table>
        </div>

        <!-- 主键信息 -->
        <div v-if="schema.primaryKey && schema.primaryKey.columns.length > 0" class="info-section">
          <div class="section-header">
            <h4>主键</h4>
          </div>
          <div class="tag-list">
            <el-tag v-for="col in schema.primaryKey.columns" :key="col" type="danger" class="key-tag">
              {{ col }}
            </el-tag>
          </div>
        </div>

        <!-- 索引信息 -->
        <div v-if="schema.indexes && schema.indexes.length > 0" class="info-section">
          <div class="section-header">
            <h4>索引</h4>
          </div>
          <el-table :data="schema.indexes" stripe size="small" class="schema-table">
            <el-table-column prop="name" label="索引名" width="200" />
            <el-table-column prop="type" label="类型" width="120">
              <template #default="{ row }">
                <el-tag :type="getIndexTypeTag(row.type)" size="small">
                  {{ row.type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="列" min-width="300">
              <template #default="{ row }">
                <el-tag v-for="col in row.columns" :key="col" class="column-tag">
                  {{ col }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 外键信息 -->
        <div v-if="schema.foreignKeys && schema.foreignKeys.length > 0" class="info-section">
          <div class="section-header">
            <h4>外键约束</h4>
          </div>
          <el-table :data="schema.foreignKeys" stripe size="small" class="schema-table">
            <el-table-column prop="name" label="约束名" width="200" />
            <el-table-column label="列" width="150">
              <template #default="{ row }">
                <el-tag v-for="col in row.columns" :key="col" size="small" class="column-tag">
                  {{ col }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="referencedTable" label="引用表" width="150" />
            <el-table-column label="引用列" width="150">
              <template #default="{ row }">
                <el-tag v-for="col in row.referencedColumns" :key="col" size="small" class="column-tag">
                  {{ col }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="onDelete" label="ON DELETE" width="120" />
            <el-table-column prop="onUpdate" label="ON UPDATE" width="120" />
          </el-table>
        </div>

        <!-- DDL 语句 -->
        <div class="info-section">
          <div class="section-header">
            <h4>CREATE TABLE DDL</h4>
            <el-button 
              :icon="CopyDocument" 
              size="small" 
              type="primary"
              plain
              @click="copyDDL"
            >
              复制 DDL
            </el-button>
          </div>
          <el-input
            v-model="ddl"
            type="textarea"
            :rows="10"
            readonly
            class="ddl-textarea"
          />
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue';
import { ElMessage } from 'element-plus';
import { 
  Document, 
  Check, 
  CopyDocument,
  Search
} from '@element-plus/icons-vue';
import { SchemaAPI } from '../api';
import type { Column, TableSchema } from '../types/api';

// Props
interface Props {
  profileId: string;
  database: string;
  table: string;
}

const props = defineProps<Props>();

defineEmits<{
  edit: [schema: TableSchema]
}>();

// State
const loading = ref(false);
const schema = ref<TableSchema | null>(null);
const ddl = ref('');
const columnKeyword = ref('');
let schemaRequestToken = 0;

const filteredColumns = computed<Column[]>(() => {
  if (!schema.value) return [];
  const keyword = columnKeyword.value.trim().toLowerCase();
  if (!keyword) return schema.value.columns;

  return schema.value.columns.filter((column) => {
    return [
      column.name,
      column.type,
      column.comment,
      String(column.defaultValue ?? ''),
    ].some((value) => value.toLowerCase().includes(keyword));
  });
});

const summaryItems = computed(() => {
  if (!schema.value) return [];

  return [
    { label: '引擎', value: schema.value.engine || '-' },
    { label: '字符集', value: schema.value.charset || '-' },
    { label: '列', value: schema.value.columns.length },
    { label: '索引', value: schema.value.indexes?.length || 0 },
    { label: '外键', value: schema.value.foreignKeys?.length || 0 },
  ];
});

// 获取索引类型标签颜色
const getIndexTypeTag = (type: string) => {
  switch (type.toUpperCase()) {
    case 'PRIMARY':
      return 'danger';
    case 'UNIQUE':
      return 'warning';
    case 'FULLTEXT':
      return 'success';
    default:
      return 'info';
  }
};

// 加载表结构
const loadSchema = async () => {
  const requestToken = ++schemaRequestToken;
  const requestDatabase = props.database;
  const requestTable = props.table;
  if (!props.profileId || !props.database || !props.table) {
    schema.value = null;
    ddl.value = '';
    return;
  }

  loading.value = true;
  try {
    // 加载表结构
    const tableSchema = await SchemaAPI.getTableSchema(
      props.profileId,
      requestDatabase,
      requestTable
    );
    if (requestToken !== schemaRequestToken || requestDatabase !== props.database || requestTable !== props.table) return;
    schema.value = tableSchema;

    // 加载 DDL
    const tableDDL = await SchemaAPI.getCreateTableDDL(
      props.profileId,
      requestDatabase,
      requestTable
    );
    if (requestToken !== schemaRequestToken || requestDatabase !== props.database || requestTable !== props.table) return;
    ddl.value = tableDDL;
  } catch (error: any) {
    if (requestToken !== schemaRequestToken || requestDatabase !== props.database || requestTable !== props.table) return;
    ElMessage.error(error.message || '加载表结构失败');
    console.error('Failed to load schema:', error);
    schema.value = null;
    ddl.value = '';
  } finally {
    if (requestToken === schemaRequestToken) {
      loading.value = false;
    }
  }
};

// 复制表名
const copyTableName = async () => {
  if (!schema.value) return;

  try {
    await navigator.clipboard.writeText(schema.value.name);
    ElMessage.success('表名已复制到剪贴板');
  } catch (error) {
    ElMessage.error('复制失败');
  }
};

// 复制 DDL
const copyDDL = async () => {
  try {
    await navigator.clipboard.writeText(ddl.value);
    ElMessage.success('DDL 已复制到剪贴板');
  } catch (error) {
    ElMessage.error('复制失败');
  }
};

// 组件挂载时加载数据
onMounted(() => {
  loadSchema();
});

// 监听 props 变化
watch(() => [props.profileId, props.database, props.table], () => {
  columnKeyword.value = '';
  loadSchema();
}, { immediate: false });
</script>

<style scoped>
.schema-viewer {
  height: 100%;
  overflow-y: auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.table-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 500;
  color: #303133;
  min-width: 0;
}

.table-title span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.schema-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.summary-strip {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  border: 1px solid #ebeef5;
  border-radius: 6px;
  background: #fafafa;
}

.summary-item {
  min-width: 0;
  padding: 10px 14px;
  border-right: 1px solid #ebeef5;
}

.summary-item:last-child {
  border-right: 0;
}

.summary-label {
  display: block;
  margin-bottom: 4px;
  font-size: 12px;
  color: #909399;
}

.summary-value {
  display: block;
  overflow: hidden;
  color: #303133;
  font-size: 14px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.section-title {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.section-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  display: flex;
  align-items: center;
}

.section-count {
  font-size: 12px;
  color: #909399;
}

.column-search {
  width: 240px;
}

.table-name-text {
  font-weight: 500;
  color: #303133;
}

.text-muted {
  color: #909399;
  font-style: italic;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.key-tag,
.column-tag {
  margin-right: 8px;
  margin-bottom: 4px;
}

.ddl-textarea {
  font-family: 'Courier New', monospace;
  font-size: 13px;
}

.schema-table {
  width: 100%;
}

:deep(.el-card__body) {
  padding: 20px;
}

:deep(.el-card__header) {
  padding: 12px 20px;
}

:deep(.el-descriptions__label) {
  font-weight: 500;
}

:deep(.schema-table .el-table__cell) {
  padding: 7px 0;
}

@media (max-width: 900px) {
  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .summary-item {
    border-right: 0;
    border-bottom: 1px solid #ebeef5;
  }

  .summary-item:nth-last-child(-n + 1) {
    border-bottom: 0;
  }

  .section-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .column-search {
    width: 100%;
  }
}
</style>
