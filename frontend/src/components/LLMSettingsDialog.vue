<template>
  <el-dialog
    v-model="visible"
    title="LLM 配置"
    width="min(640px, 92vw)"
    class="llm-settings-dialog"
  >
    <el-form :model="form" label-width="110px" class="llm-form">
      <el-form-item label="启用 AI">
        <el-switch v-model="form.enabled" />
      </el-form-item>
      <el-form-item label="接口地址">
        <el-input v-model="form.baseUrl" placeholder="https://api.openai.com/v1" />
      </el-form-item>
      <el-form-item label="模型">
        <el-input v-model="form.model" placeholder="gpt-4o-mini" />
      </el-form-item>
      <el-form-item label="API Key">
        <el-input
          v-model="form.apiKey"
          type="password"
          show-password
          autocomplete="off"
          :placeholder="savedConfig?.hasApiKey ? `已保存 ${savedConfig.maskedApiKey}，留空不修改` : '请输入 API Key'"
        />
      </el-form-item>
      <el-form-item label="温度">
        <el-slider v-model="form.temperature" :min="0" :max="1" :step="0.1" />
      </el-form-item>
      <el-form-item label="超时">
        <el-input-number v-model="form.timeoutSec" :min="10" :max="180" :step="5" />
        <span class="field-unit">秒</span>
      </el-form-item>
    </el-form>

    <div class="settings-hint">
      兼容 OpenAI Chat Completions 协议的服务都可以接入，Base URL 填到 /v1 级别。
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button :loading="testing" @click="testConfig">测试连接</el-button>
      <el-button type="primary" :loading="saving" @click="saveConfig">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { LLMAPI } from '../api/llm'
import type { LLMConfig, PublicLLMConfig } from '../types/llm'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved', value: PublicLLMConfig): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const defaultForm = (): LLMConfig => ({
  enabled: false,
  baseUrl: 'https://api.openai.com/v1',
  apiKey: '',
  model: 'gpt-4o-mini',
  temperature: 0.1,
  timeoutSec: 45,
})

const form = reactive<LLMConfig>(defaultForm())
const savedConfig = ref<PublicLLMConfig | null>(null)
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)

const applyConfig = (config: PublicLLMConfig) => {
  savedConfig.value = config
  form.enabled = config.enabled
  form.baseUrl = config.baseUrl || 'https://api.openai.com/v1'
  form.model = config.model || 'gpt-4o-mini'
  form.temperature = typeof config.temperature === 'number' ? config.temperature : 0.1
  form.timeoutSec = config.timeoutSec || 45
  form.apiKey = ''
}

const loadConfig = async () => {
  loading.value = true
  try {
    applyConfig(await LLMAPI.getConfig())
  } catch (error: any) {
    ElMessage.error(`读取 LLM 配置失败: ${error?.message || error}`)
  } finally {
    loading.value = false
  }
}

const validate = () => {
  if (!form.baseUrl.trim()) {
    ElMessage.warning('请填写接口地址')
    return false
  }
  if (!form.model.trim()) {
    ElMessage.warning('请填写模型')
    return false
  }
  if (form.enabled && !form.apiKey && !savedConfig.value?.hasApiKey) {
    ElMessage.warning('启用 AI 前请填写 API Key')
    return false
  }
  return true
}

const payload = (): LLMConfig => ({
  enabled: form.enabled,
  baseUrl: form.baseUrl.trim(),
  model: form.model.trim(),
  apiKey: form.apiKey?.trim() || '',
  temperature: form.temperature,
  timeoutSec: form.timeoutSec,
})

const testConfig = async () => {
  if (!validate()) return
  testing.value = true
  try {
    await LLMAPI.testConfig(payload())
    ElMessage.success('LLM 连接测试通过')
  } catch (error: any) {
    ElMessage.error(`LLM 连接测试失败: ${error?.message || error}`)
  } finally {
    testing.value = false
  }
}

const saveConfig = async () => {
  if (!validate()) return
  saving.value = true
  try {
    const saved = await LLMAPI.saveConfig(payload())
    applyConfig(saved)
    emit('saved', saved)
    ElMessage.success('LLM 配置已保存')
    visible.value = false
  } catch (error: any) {
    ElMessage.error(`保存 LLM 配置失败: ${error?.message || error}`)
  } finally {
    saving.value = false
  }
}

watch(visible, (value) => {
  if (value) loadConfig()
})
</script>

<style scoped>
.llm-form {
  padding-top: 4px;
}

.field-unit {
  margin-left: 8px;
  color: #606266;
}

.settings-hint {
  margin-top: 8px;
  padding: 10px 12px;
  border-radius: 6px;
  background: #f5f7fa;
  color: #606266;
  font-size: 13px;
  line-height: 1.5;
}
</style>
