<template>
  <el-dialog
    v-model="visible"
    title="导入数据"
    width="min(920px, 92vw)"
    class="import-dialog"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="form" label-width="100px" size="default">
      <!-- 文件选择 -->
      <el-form-item label="选择文件">
        <el-input
          v-model="form.filePath"
          placeholder="点击浏览按钮选择文件"
          readonly
        >
          <template #append>
            <el-button @click="selectFile">浏览</el-button>
          </template>
        </el-input>
      </el-form-item>

      <!-- 文件格式 -->
      <el-form-item v-if="form.filePath" label="文件格式">
        <el-tag :type="formatTagType">{{ detectedFormat || '未知' }}</el-tag>
        <el-text type="info" size="small" class="file-summary">
          {{ fileName }}
        </el-text>
        <el-text v-if="formatValidated" type="success" size="small" style="margin-left: 12px;">
          <el-icon><CircleCheck /></el-icon>
          格式验证通过
        </el-text>
        <el-text v-else-if="formatError" type="danger" size="small" style="margin-left: 12px;">
          <el-icon><CircleClose /></el-icon>
          {{ formatError }}
        </el-text>
      </el-form-item>

      <el-alert
        v-if="form.filePath"
        title="大文件导入会在后台流式分批写入，确认列映射后即可启动；任务期间可以继续使用其他查询和表格。"
        type="info"
        show-icon
        :closable="false"
        class="import-hint"
      />

      <!-- 列映射配置 (CSV 和 JSON) -->
      <el-form-item
        v-if="showMapping && tableColumns.length > 0"
        label="列映射"
        class="mapping-item"
      >
        <div class="mapping-container">
          <div class="mapping-actions">
            <el-button size="small" @click="autoMapColumns">自动匹配</el-button>
            <el-button size="small" @click="resetMapping">重置映射</el-button>
            <el-text size="small" type="info">
              已映射 {{ mappedColumnCount }} / {{ fileColumns.length }}
            </el-text>
          </div>
          <div class="mapping-header">
            <span class="mapping-col">文件列</span>
            <span class="mapping-arrow"></span>
            <span class="mapping-col">表列</span>
          </div>
          <div
            v-for="(fileCol, index) in fileColumns"
            :key="index"
            class="mapping-row"
          >
            <el-input
              v-model="fileColumns[index]"
              size="small"
              placeholder="文件列名"
              class="mapping-input"
            />
            <el-icon class="mapping-arrow-icon"><Right /></el-icon>
            <el-select
              v-model="form.mapping.TableColumns[index]"
              size="small"
              placeholder="选择表列"
              class="mapping-select"
              clearable
            >
              <el-option
                v-for="col in tableColumns"
                :key="col.name"
                :label="`${col.name} (${col.type})`"
                :value="col.name"
              />
            </el-select>
            <el-button
              size="small"
              type="danger"
              :icon="Delete"
              circle
              @click="removeMapping(index)"
            />
          </div>
          <el-button
            size="small"
            type="primary"
            :icon="Plus"
            @click="addMapping"
            style="margin-top: 8px;"
          >
            添加映射
          </el-button>
        </div>
      </el-form-item>

      <!-- 导入进度 -->
      <el-form-item v-if="importing" label="启动状态">
        <el-progress
          :percentage="progress.percentage"
          :status="progress.status"
        />
        <div class="progress-info">
          <el-text size="small">
            正在创建后台导入任务，完成后会在右上角提示
          </el-text>
        </div>
      </el-form-item>

      <el-form-item v-if="lastTaskId" label="当前任务">
        <div class="task-inline">
          <span>{{ taskStatusText }}</span>
          <el-button
            v-if="taskRunning"
            size="small"
            type="danger"
            link
            @click="cancelCurrentTask"
          >
            停止
          </el-button>
        </div>
      </el-form-item>

      <!-- 导入结果 -->
      <el-form-item v-if="importResult" label="导入结果">
        <div class="result-container">
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="总行数">
              {{ resultValue('totalRows') }}
            </el-descriptions-item>
            <el-descriptions-item label="成功行数">
              <el-text type="success">{{ resultValue('successRows') }}</el-text>
            </el-descriptions-item>
            <el-descriptions-item label="失败行数">
              <el-text :type="resultValue('failedRows') > 0 ? 'danger' : 'info'">
                {{ resultValue('failedRows') }}
              </el-text>
            </el-descriptions-item>
          </el-descriptions>

          <!-- 错误详情 -->
          <div v-if="resultErrors.length > 0" class="error-details">
            <el-divider content-position="left">错误详情</el-divider>
            <el-scrollbar max-height="200px">
              <div
                v-for="(error, index) in resultErrors"
                :key="index"
                class="error-item"
              >
                <el-text type="danger" size="small">
                  第 {{ error.row ?? error.Row }} 行: {{ error.message ?? error.Message }}
                </el-text>
              </div>
            </el-scrollbar>
          </div>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">
          {{ importResult ? '关闭' : '取消' }}
        </el-button>
        <el-button
          v-if="!importResult"
          type="primary"
          @click="handleImport"
          :loading="importing"
          :disabled="!canImport"
        >
          {{ importing ? '启动中...' : '后台导入' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { ElMessage, ElNotification } from 'element-plus';
import { CircleCheck, CircleClose, Right, Delete, Plus } from '@element-plus/icons-vue';
import {
  StartImport,
  CancelTransferTask,
  ValidateCSVFormat,
  ValidateJSONFormat,
  ValidateSQLFormat,
  OpenFileDialog,
} from '../../wailsjs/go/backend/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { backend, importexport } from '../../wailsjs/go/models';
import type { Column } from '../types/api';

interface Props {
  modelValue: boolean;
  profileId: string;
  database: string;
  table: string;
  tableColumns: Column[];
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  'success': [];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
});

const form = ref({
  filePath: '',
  mapping: {
    FileColumns: [] as string[],
    TableColumns: [] as string[],
  },
});

const fileColumns = ref<string[]>([]);
const detectedFormat = ref<'SQL' | 'CSV' | 'JSON' | null>(null);
const formatValidated = ref(false);
const formatError = ref('');
const importing = ref(false);
const importResult = ref<importexport.ImportResult | null>(null);
const activeImportTasks = new Set<string>();
const lastTaskId = ref('');
const taskRunning = ref(false);
const taskStatusText = ref('');
const fileName = computed(() => form.value.filePath.split(/[\\/]/).pop() || form.value.filePath);
const mappedColumnCount = computed(() => form.value.mapping.TableColumns.filter(Boolean).length);

const progress = ref({
  current: 0,
  total: 0,
  percentage: 0,
  status: undefined as 'success' | 'exception' | undefined,
});

const formatTagType = computed(() => {
  if (!detectedFormat.value) return 'info';
  if (formatValidated.value) return 'success';
  if (formatError.value) return 'danger';
  return 'warning';
});

const showMapping = computed(() => {
  return (
    detectedFormat.value === 'CSV' ||
    detectedFormat.value === 'JSON'
  ) && formatValidated.value;
});

const canImport = computed(() => {
  if (!form.value.filePath || !formatValidated.value) {
    return false;
  }
  
  // SQL 文件不需要映射
  if (detectedFormat.value === 'SQL') {
    return true;
  }
  
  // CSV 和 JSON 需要至少一个映射
  return mappedColumnCount.value > 0;
});

const resultValue = (key: 'totalRows' | 'successRows' | 'failedRows') => {
  if (!importResult.value) return 0;
  const pascalKey = key.charAt(0).toUpperCase() + key.slice(1);
  return (importResult.value as any)[key] ?? (importResult.value as any)[pascalKey] ?? 0;
};

const resultErrors = computed(() => {
  if (!importResult.value) return [];
  return (importResult.value as any).errors ?? (importResult.value as any).Errors ?? [];
});

// 选择文件
const selectFile = async () => {
  try {
    const filters: backend.FileDialogFilter[] = [
      backend.FileDialogFilter.createFrom({
        displayName: 'SQL 文件',
        pattern: '*.sql',
      }),
      backend.FileDialogFilter.createFrom({
        displayName: 'CSV 文件',
        pattern: '*.csv',
      }),
      backend.FileDialogFilter.createFrom({
        displayName: 'JSON 文件',
        pattern: '*.json',
      }),
      backend.FileDialogFilter.createFrom({
        displayName: '所有文件',
        pattern: '*',
      }),
    ];
    
    // 使用后端 API 打开文件选择对话框
    const path = await OpenFileDialog('选择导入文件', filters);
    
    if (path) {
      form.value.filePath = path;
      detectFormat(path);
    }
  } catch (error: any) {
    console.error('Failed to select file:', error);
    ElMessage.error(`选择文件失败: ${error.message || error}`);
  }
};

// 检测文件格式
const detectFormat = async (filePath: string) => {
  formatValidated.value = false;
  formatError.value = '';
  detectedFormat.value = null;
  
  // 根据文件扩展名检测格式
  const ext = filePath.split('.').pop()?.toLowerCase();
  
  if (ext === 'sql') {
    detectedFormat.value = 'SQL';
    await validateFormat('SQL', filePath);
  } else if (ext === 'csv') {
    detectedFormat.value = 'CSV';
    await validateFormat('CSV', filePath);
  } else if (ext === 'json') {
    detectedFormat.value = 'JSON';
    await validateFormat('JSON', filePath);
  } else {
    formatError.value = '不支持的文件格式';
  }
};

// 验证文件格式
const validateFormat = async (format: 'SQL' | 'CSV' | 'JSON', filePath: string) => {
  try {
    if (format === 'SQL') {
      await ValidateSQLFormat(props.profileId, filePath);
    } else if (format === 'CSV') {
      await ValidateCSVFormat(props.profileId, filePath);
      // CSV 验证通过后，初始化映射
      initializeMappingForCSV();
    } else if (format === 'JSON') {
      await ValidateJSONFormat(props.profileId, filePath);
      // JSON 验证通过后，初始化映射
      initializeMappingForJSON();
    }
    
    formatValidated.value = true;
    formatError.value = '';
  } catch (error: any) {
    formatValidated.value = false;
    formatError.value = error.message || '格式验证失败';
    ElMessage.error(`文件格式验证失败: ${error.message || error}`);
  }
};

// 初始化 CSV 映射
const initializeMappingForCSV = () => {
  // 默认使用表的列名作为文件列名
  fileColumns.value = props.tableColumns.map(col => col.name);
  resetMapping();
};

// 初始化 JSON 映射
const initializeMappingForJSON = () => {
  // 默认使用表的列名作为文件列名
  fileColumns.value = props.tableColumns.map(col => col.name);
  resetMapping();
};

const autoMapColumns = () => {
  form.value.mapping.FileColumns = [...fileColumns.value];
  form.value.mapping.TableColumns = fileColumns.value.map((fileCol) => {
    const exact = props.tableColumns.find(col => col.name === fileCol);
    if (exact) return exact.name;
    const lower = fileCol.toLowerCase();
    return props.tableColumns.find(col => col.name.toLowerCase() === lower)?.name || '';
  });
};

const resetMapping = () => {
  form.value.mapping.FileColumns = [...fileColumns.value];
  form.value.mapping.TableColumns = fileColumns.value.map(fileCol => {
    return props.tableColumns.some(col => col.name === fileCol) ? fileCol : '';
  });
};

// 添加映射
const addMapping = () => {
  fileColumns.value.push('');
  form.value.mapping.FileColumns.push('');
  form.value.mapping.TableColumns.push('');
};

// 删除映射
const removeMapping = (index: number) => {
  fileColumns.value.splice(index, 1);
  form.value.mapping.FileColumns.splice(index, 1);
  form.value.mapping.TableColumns.splice(index, 1);
};

// 处理导入
const handleImport = async () => {
  if (!canImport.value) {
    ElMessage.warning('请完成必要的配置');
    return;
  }

  importing.value = true;
  importResult.value = null;
  progress.value = {
    current: 0,
    total: 0,
    percentage: 0,
    status: undefined,
  };

  try {
    const mapping = importexport.ColumnMapping.createFrom({
      FileColumns: fileColumns.value.filter((_, i) => form.value.mapping.TableColumns[i]),
      TableColumns: form.value.mapping.TableColumns.filter(col => col),
    });

    if (!detectedFormat.value) {
      throw new Error('不支持的文件格式');
    }

    const taskId = await StartImport(
      props.profileId,
      props.database,
      detectedFormat.value === 'SQL' ? '' : props.table,
      detectedFormat.value,
      form.value.filePath,
      mapping
    );
    activeImportTasks.add(taskId);
    lastTaskId.value = taskId;
    taskRunning.value = true;
    taskStatusText.value = '后台导入运行中';
    progress.value.percentage = 100;
    progress.value.status = 'success';
    importing.value = false;
    ElNotification.info({
      title: '导入任务已开始',
      message: `${fileName.value} 正在后台导入，完成后会通知你。`,
      duration: 4500,
    });
    visible.value = false;
    resetFormSoon();
  } catch (error: any) {
    console.error('Import failed:', error);
    progress.value.status = 'exception';
    ElMessage.error(`导入失败: ${error.message || error}`);
  } finally {
    importing.value = false;
  }
};

const cancelCurrentTask = async () => {
  if (!lastTaskId.value) return;
  try {
    await CancelTransferTask(lastTaskId.value);
    taskStatusText.value = '正在停止导入任务...';
  } catch (error: any) {
    ElMessage.error(`停止导入失败: ${error.message || error}`);
  }
};

const matchesTask = (data: any) => {
  return data?.taskId && activeImportTasks.has(data.taskId);
};

const eventUnlisteners: Array<() => void> = [];

const resultFromEvent = (data: any): importexport.ImportResult => {
  return importexport.ImportResult.createFrom(
    data.result || {
      TotalRows: data.totalRows ?? data.TotalRows ?? 0,
      SuccessRows: data.successRows ?? data.SuccessRows ?? 0,
      FailedRows: data.failedRows ?? data.FailedRows ?? 0,
      Errors: data.errors ?? data.Errors ?? [],
    }
  );
};

onMounted(() => {
  eventUnlisteners.push(
    EventsOn('import:progress', (data: any) => {
      if (!matchesTask(data)) return;
      progress.value.current = data.current;
      progress.value.total = data.total;
      progress.value.percentage = Math.round(data.percentage);
    })
  );
  eventUnlisteners.push(
    EventsOn('import:completed', (data: any) => {
      if (!matchesTask(data)) return;
      activeImportTasks.delete(data.taskId);
      taskRunning.value = false;
      taskStatusText.value = '导入完成';
      const result = resultFromEvent(data);
      importResult.value = result;
      emit('success');
      const failedRows = (result as any).failedRows ?? (result as any).FailedRows ?? 0;
      const successRows = (result as any).successRows ?? (result as any).SuccessRows ?? 0;
      ElNotification({
        type: failedRows > 0 ? 'warning' : 'success',
        title: failedRows > 0 ? '导入完成，有失败行' : '导入完成',
        message: `成功 ${successRows} 行，失败 ${failedRows} 行`,
        duration: failedRows > 0 ? 0 : 6000,
      });
    })
  );
  eventUnlisteners.push(
    EventsOn('import:failed', (data: any) => {
      if (!matchesTask(data)) return;
      activeImportTasks.delete(data.taskId);
      taskRunning.value = false;
      taskStatusText.value = '导入失败';
      if (data.result) {
        importResult.value = resultFromEvent(data);
      }
      ElNotification.error({
        title: '导入失败',
        message: data.error || '后台导入任务失败',
        duration: 0,
      });
    })
  );
  eventUnlisteners.push(
    EventsOn('import:cancelled', (data: any) => {
      if (!matchesTask(data)) return;
      activeImportTasks.delete(data.taskId);
      taskRunning.value = false;
      taskStatusText.value = '导入已停止';
      if (data.result) {
        importResult.value = resultFromEvent(data);
      }
    })
  );
});

onUnmounted(() => {
  eventUnlisteners.splice(0).forEach((off) => off());
  activeImportTasks.clear();
});

// 关闭对话框
const handleClose = () => {
  visible.value = false;
  resetFormSoon();
};

const resetFormSoon = () => {
  setTimeout(() => {
    form.value = {
      filePath: '',
      mapping: {
        FileColumns: [],
        TableColumns: [],
      },
    };
    fileColumns.value = [];
    detectedFormat.value = null;
    formatValidated.value = false;
    formatError.value = '';
    importResult.value = null;
    progress.value = {
      current: 0,
      total: 0,
      percentage: 0,
      status: undefined,
    };
    importing.value = false;
  }, 300);
};

// 监听文件路径变化
watch(() => form.value.filePath, (newPath) => {
  if (!newPath) {
    detectedFormat.value = null;
    formatValidated.value = false;
    formatError.value = '';
    fileColumns.value = [];
    form.value.mapping = {
      FileColumns: [],
      TableColumns: [],
    };
  }
});
</script>

<style scoped>
.import-dialog :deep(.el-dialog__body) {
  padding-top: 12px;
}

.file-summary {
  margin-left: 12px;
}

.import-hint {
  margin: 0 0 16px;
}

.mapping-container {
  width: 100%;
}

.mapping-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.mapping-header,
.mapping-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 28px minmax(0, 1fr) 32px;
  align-items: center;
  gap: 8px;
}

.mapping-header {
  margin-bottom: 6px;
  color: #909399;
  font-size: 12px;
}

.mapping-row + .mapping-row {
  margin-top: 6px;
}

.mapping-input,
.mapping-select {
  width: 100%;
}

.mapping-arrow-icon {
  color: #909399;
}

.progress-info {
  margin-top: 8px;
}

.task-inline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  color: var(--app-text-secondary);
  font-size: 13px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>

<style scoped>
.mapping-item {
  margin-bottom: 24px;
}

.mapping-container {
  width: 100%;
  padding: 12px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.mapping-header {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  font-weight: 600;
  color: #606266;
}

.mapping-col {
  flex: 1;
  text-align: center;
}

.mapping-arrow {
  width: 40px;
  text-align: center;
}

.mapping-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.mapping-input,
.mapping-select {
  flex: 1;
}

.mapping-arrow-icon {
  color: #909399;
  font-size: 16px;
}

.progress-info {
  margin-top: 8px;
}

.result-container {
  width: 100%;
}

.error-details {
  margin-top: 16px;
}

.error-item {
  padding: 4px 0;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
