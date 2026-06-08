<template>
  <el-dialog
    v-model="visible"
    title="导出数据"
    width="min(760px, 90vw)"
    class="export-dialog"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="form" label-width="100px" size="default">
      <!-- 导出格式 -->
      <el-form-item label="导出格式">
        <el-radio-group v-model="form.format">
          <el-radio label="SQL">SQL INSERT 语句</el-radio>
          <el-radio label="CSV">CSV 格式</el-radio>
          <el-radio label="JSON">JSON 格式</el-radio>
        </el-radio-group>
      </el-form-item>

      <!-- 导出范围 -->
      <el-form-item label="导出范围">
        <el-radio-group v-model="form.scope">
          <el-radio label="all">全部数据</el-radio>
          <el-radio label="filtered" :disabled="!hasFilters">当前筛选结果</el-radio>
        </el-radio-group>
        <div v-if="form.scope === 'filtered' && hasFilters" class="scope-info">
          <el-text type="info" size="small">
            将导出当前筛选和排序条件下的数据
          </el-text>
        </div>
      </el-form-item>

      <el-alert
        :title="exportHintText"
        type="info"
        show-icon
        :closable="false"
        class="export-hint"
      />

      <!-- 文件保存路径 -->
      <el-form-item label="保存路径">
        <el-input
          v-model="form.outputPath"
          placeholder="点击浏览按钮选择保存位置"
          readonly
        >
          <template #append>
            <el-button @click="selectOutputPath">浏览</el-button>
          </template>
        </el-input>
      </el-form-item>

      <!-- 导出进度 -->
      <el-form-item v-if="exporting" label="启动状态">
        <el-progress
          :percentage="progress.percentage"
          :status="progress.status"
        />
        <div class="progress-info">
          <el-text size="small">
            正在创建后台导出任务，完成后会在右上角提示
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
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button
          type="primary"
          @click="handleExport"
          :loading="exporting"
          :disabled="!form.outputPath"
        >
          {{ exporting ? '启动中...' : '后台导出' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { ElMessage, ElNotification } from 'element-plus';
import { StartExport, SaveFileDialog, CancelTransferTask } from '../../wailsjs/go/backend/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { backend, repository } from '../../wailsjs/go/models';
import type { Filter, OrderBy } from '../types/api';

interface Props {
  modelValue: boolean;
  profileId: string;
  database: string;
  table: string;
  filters?: Filter[];
  orderBy?: OrderBy[];
}

const props = withDefaults(defineProps<Props>(), {
  filters: () => [],
  orderBy: () => [],
});

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  'success': [];
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
});

const form = ref({
  format: 'SQL' as 'SQL' | 'CSV' | 'JSON',
  scope: 'all' as 'all' | 'filtered',
  outputPath: '',
});

const exporting = ref(false);
const activeExportTasks = new Set<string>();
const lastTaskId = ref('');
const taskRunning = ref(false);
const taskStatusText = ref('');
const progress = ref({
  current: 0,
  total: 0,
  percentage: 0,
  status: undefined as 'success' | 'exception' | undefined,
});

const hasFilters = computed(() => {
  return props.filters && props.filters.length > 0;
});

const exportHintText = computed(() => {
  const scope = form.value.scope === 'filtered' && hasFilters.value ? '当前筛选结果' : '全部数据';
  return `将后台分批导出${scope}；后端会优先按单列主键游标逐批查询，复杂排序或偏移会自动降级为有界分页。`;
});

// 选择输出路径
const selectOutputPath = async () => {
  try {
    const ext = form.value.format.toLowerCase();
    const filters: backend.FileDialogFilter[] = [
      backend.FileDialogFilter.createFrom({
        displayName: `${form.value.format} 文件`,
        pattern: `*.${ext}`,
      }),
    ];
    
    const defaultFilename = `${props.table}_export.${ext}`;
    
    // 使用后端 API 打开文件保存对话框
    const path = await SaveFileDialog('选择保存位置', defaultFilename, filters);
    
    if (path) {
      form.value.outputPath = path;
    }
  } catch (error: any) {
    console.error('Failed to select output path:', error);
    ElMessage.error(`选择文件失败: ${error.message || error}`);
  }
};

// 构建查询条件
const buildQuery = (): repository.DataQuery => {
  const query = repository.DataQuery.createFrom({
    Database: props.database,
    Table: props.table,
    Columns: [],
    Filters: [],
    OrderBy: [],
    Limit: 0,
    Offset: 0,
  });

  // 如果选择导出筛选结果，应用筛选和排序条件
  if (form.value.scope === 'filtered') {
    query.Filters = props.filters.map(f => ({
      Column: (f as any).Column || f.column,
      Operator: (f as any).Operator || f.operator,
      Value: (f as any).Value ?? f.value,
    })) as repository.Filter[];
    
    query.OrderBy = props.orderBy.map(o => ({
      Column: (o as any).Column || o.column,
      Direction: (o as any).Direction || o.direction,
    })) as repository.OrderBy[];
  }

  return query;
};

// 处理导出
const handleExport = async () => {
  if (!form.value.outputPath) {
    ElMessage.warning('请选择保存路径');
    return;
  }

  exporting.value = true;
  progress.value = {
    current: 0,
    total: 0,
    percentage: 0,
    status: undefined,
  };

  try {
    const query = buildQuery();
    const taskId = await StartExport(
      props.profileId,
      props.database,
      props.table,
      form.value.format,
      query,
      form.value.outputPath
    );
    activeExportTasks.add(taskId);
    lastTaskId.value = taskId;
    taskRunning.value = true;
    taskStatusText.value = '后台导出运行中';
    progress.value.percentage = 100;
    progress.value.status = 'success';
    exporting.value = false;
    ElNotification.info({
      title: '导出任务已开始',
      message: `${props.database}.${props.table} 正在后台导出，完成后会通知你。`,
      duration: 4500,
    });
    visible.value = false;
    resetFormSoon();
  } catch (error: any) {
    console.error('Export failed:', error);
    progress.value.status = 'exception';
    ElMessage.error(`导出失败: ${error.message || error}`);
    exporting.value = false;
  }
};

const cancelCurrentTask = async () => {
  if (!lastTaskId.value) return;
  try {
    await CancelTransferTask(lastTaskId.value);
    taskStatusText.value = '正在停止导出任务...';
  } catch (error: any) {
    ElMessage.error(`停止导出失败: ${error.message || error}`);
  }
};

const matchesTask = (data: any) => {
  return data?.taskId && activeExportTasks.has(data.taskId);
};

const unlisteners: Array<() => void> = [];

onMounted(() => {
  unlisteners.push(
    EventsOn('export:progress', (data: any) => {
      if (!matchesTask(data)) return;
      progress.value.current = data.current;
      progress.value.total = data.total;
      progress.value.percentage = Math.round(data.percentage);
    })
  );
  unlisteners.push(
    EventsOn('export:completed', (data: any) => {
      if (!matchesTask(data)) return;
      activeExportTasks.delete(data.taskId);
      taskRunning.value = false;
      taskStatusText.value = '导出完成';
      emit('success');
      ElNotification.success({
        title: '导出完成',
        message: `${data.database}.${data.table} 已导出到 ${data.outputPath}`,
        duration: 0,
      });
    })
  );
  unlisteners.push(
    EventsOn('export:failed', (data: any) => {
      if (!matchesTask(data)) return;
      activeExportTasks.delete(data.taskId);
      taskRunning.value = false;
      taskStatusText.value = '导出失败';
      ElNotification.error({
        title: '导出失败',
        message: data.error || '后台导出任务失败',
        duration: 0,
      });
    })
  );
  unlisteners.push(
    EventsOn('export:cancelled', (data: any) => {
      if (!matchesTask(data)) return;
      activeExportTasks.delete(data.taskId);
      taskRunning.value = false;
      taskStatusText.value = '导出已停止';
    })
  );
});

onUnmounted(() => {
  unlisteners.splice(0).forEach((off) => off());
  activeExportTasks.clear();
});

// 关闭对话框
const handleClose = () => {
  visible.value = false;
  resetFormSoon();
};

const resetFormSoon = () => {
  setTimeout(() => {
    form.value = {
      format: 'SQL',
      scope: 'all',
      outputPath: '',
    };
    progress.value = {
      current: 0,
      total: 0,
      percentage: 0,
      status: undefined,
    };
    exporting.value = false;
  }, 300);
};

// 监听格式变化，清空输出路径
watch(() => form.value.format, () => {
  form.value.outputPath = '';
});
</script>

<style scoped>
.export-dialog :deep(.el-dialog__body) {
  padding-top: 12px;
}

.scope-info {
  margin-top: 8px;
}

.export-hint {
  margin: 0 0 16px;
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
