<template>
  <div class="mongo-connection-manager">
    <!-- Left panel: connection list -->
    <div class="left-panel">
      <div class="panel-header">
        <span class="panel-title">MongoDB 连接</span>
        <el-button type="primary" size="small" @click="handleCreate">
          <el-icon><Plus /></el-icon>
          新建
        </el-button>
      </div>

      <div v-loading="store.isLoading" class="profile-list">
        <div
          v-for="profile in store.profiles"
          :key="profile.id"
          class="profile-item"
          :class="{ active: selectedProfileId === profile.id }"
          @click="handleSelect(profile)"
        >
          <div class="profile-info">
            <div class="profile-name">{{ profile.name }}</div>
            <div class="profile-host">
              {{ profile.useUri ? profile.uri : `${profile.host}:${profile.port}` }}
            </div>
          </div>
          <div class="profile-status">
            <el-tag
              :type="store.isActive(profile.id) ? 'success' : 'info'"
              size="small"
            >
              {{ store.isActive(profile.id) ? '已连接' : '未连接' }}
            </el-tag>
          </div>
          <div class="profile-actions" @click.stop>
            <el-button
              v-if="!store.isActive(profile.id)"
              type="primary"
              size="small"
              :loading="connectingId === profile.id"
              @click="handleConnect(profile)"
            >连接</el-button>
            <el-button
              v-else
              type="warning"
              size="small"
              :loading="connectingId === profile.id"
              @click="handleDisconnect(profile)"
            >断开</el-button>
            <el-button size="small" @click="handleEdit(profile)">编辑</el-button>
            <el-button
              type="danger"
              size="small"
              :disabled="store.isActive(profile.id)"
              @click="handleDelete(profile)"
            >删除</el-button>
          </div>
        </div>

        <el-empty
          v-if="!store.isLoading && store.profiles.length === 0"
          description="暂无连接配置"
          :image-size="80"
        />
      </div>
    </div>

    <!-- Right panel: create/edit form -->
    <div v-if="formVisible" class="right-panel">
      <div class="panel-header">
        <span class="panel-title">{{ isEditing ? '编辑连接' : '新建连接' }}</span>
        <el-button size="small" @click="handleCancel">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
        class="connection-form"
      >
        <!-- Connection name -->
        <el-form-item label="连接名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入连接名称" />
        </el-form-item>

        <!-- Mode toggle -->
        <el-form-item label="连接模式">
          <el-radio-group v-model="form.useUri">
            <el-radio :value="false">直连模式</el-radio>
            <el-radio :value="true">URI 模式</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- Direct mode fields -->
        <template v-if="!form.useUri">
          <el-form-item label="主机地址" prop="host">
            <el-input v-model="form.host" placeholder="localhost" />
          </el-form-item>
          <el-form-item label="端口" prop="port">
            <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
          </el-form-item>
          <el-form-item label="用户名">
            <el-input v-model="form.username" placeholder="（可选）" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" type="password" show-password placeholder="（可选）" />
          </el-form-item>
          <el-form-item label="认证数据库">
            <el-input v-model="form.authDb" placeholder="admin" />
          </el-form-item>
          <el-form-item label="超时（秒）">
            <el-input-number v-model="form.timeout" :min="1" :max="60" style="width: 100%" />
          </el-form-item>
        </template>

        <!-- URI mode -->
        <template v-else>
          <el-form-item label="连接 URI" prop="uri">
            <el-input
              v-model="form.uri"
              placeholder="mongodb://user:pass@host:27017/dbname"
              type="textarea"
              :rows="2"
            />
          </el-form-item>
        </template>

        <!-- SSH tunnel -->
        <el-collapse v-model="sshCollapse" class="ssh-collapse">
          <el-collapse-item name="ssh">
            <template #title>
              <div class="ssh-title">
                <el-checkbox v-model="form.sshEnabled" @click.stop />
                <span style="margin-left: 8px">SSH 隧道</span>
              </div>
            </template>
            <template v-if="form.sshEnabled">
              <el-form-item label="SSH 主机" prop="sshHost">
                <el-input v-model="form.sshHost" placeholder="ssh.example.com" />
              </el-form-item>
              <el-form-item label="SSH 端口">
                <el-input-number v-model="form.sshPort" :min="1" :max="65535" style="width: 100%" />
              </el-form-item>
              <el-form-item label="SSH 用户名" prop="sshUsername">
                <el-input v-model="form.sshUsername" placeholder="ubuntu" />
              </el-form-item>
              <el-form-item label="SSH 密码">
                <el-input v-model="form.sshPassword" type="password" show-password placeholder="（可选）" />
              </el-form-item>
              <el-form-item label="私钥路径">
                <el-input v-model="form.sshKeyPath" placeholder="/path/to/key.pem（可选）" />
              </el-form-item>
            </template>
          </el-collapse-item>
        </el-collapse>

        <!-- Test connection result -->
        <div v-if="testResult" class="test-result" :class="testResult.success ? 'success' : 'error'">
          <el-icon v-if="testResult.success"><CircleCheck /></el-icon>
          <el-icon v-else><CircleClose /></el-icon>
          <span v-if="testResult.success">
            连接成功 — MongoDB {{ testResult.serverVersion }}
          </span>
          <span v-else>
            {{ testResult.error || testResult.message }}
          </span>
        </div>

        <!-- Form actions -->
        <el-form-item class="form-actions">
          <el-button
            :loading="testing"
            @click="handleTestConnection"
          >测试连接</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
          <el-button @click="handleCancel">取消</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Close, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { useMongoConnectionStore } from '../../stores/mongoConnection'
import type { MongoConnectionProfile, MongoTestResult } from '../../types/mongo'

const router = useRouter()
const store = useMongoConnectionStore()

// UI state
const formVisible = ref(false)
const isEditing = ref(false)
const selectedProfileId = ref<string | null>(null)
const connectingId = ref<string | null>(null)
const testing = ref(false)
const saving = ref(false)
const testResult = ref<MongoTestResult | null>(null)
const sshCollapse = ref<string[]>([])
const formRef = ref<FormInstance>()

// Form model
const emptyForm = (): MongoConnectionProfile => ({
  id: '',
  name: '',
  host: 'localhost',
  port: 27017,
  username: '',
  password: '',
  authDb: 'admin',
  useUri: false,
  uri: '',
  sshEnabled: false,
  sshHost: '',
  sshPort: 22,
  sshUsername: '',
  sshPassword: '',
  sshKeyPath: '',
  timeout: 10,
})

const form = reactive<MongoConnectionProfile>(emptyForm())

const rules: FormRules = {
  name: [{ required: true, message: '请输入连接名称', trigger: 'blur' }],
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  uri: [{ required: true, message: '请输入连接 URI', trigger: 'blur' }],
  sshHost: [
    {
      validator: (_rule, _value, callback) => {
        if (form.sshEnabled && !form.sshHost) {
          callback(new Error('请输入 SSH 主机地址'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  sshUsername: [
    {
      validator: (_rule, _value, callback) => {
        if (form.sshEnabled && !form.sshUsername) {
          callback(new Error('请输入 SSH 用户名'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
}

function applyFormData(profile: MongoConnectionProfile) {
  Object.assign(form, profile)
  if (profile.sshEnabled) {
    sshCollapse.value = ['ssh']
  }
}

function resetForm() {
  Object.assign(form, emptyForm())
  sshCollapse.value = []
  testResult.value = null
  formRef.value?.clearValidate()
}

function handleCreate() {
  isEditing.value = false
  resetForm()
  formVisible.value = true
}

function handleEdit(profile: MongoConnectionProfile) {
  isEditing.value = true
  resetForm()
  applyFormData(profile)
  formVisible.value = true
}

function handleSelect(profile: MongoConnectionProfile) {
  selectedProfileId.value = profile.id
}

function handleCancel() {
  formVisible.value = false
  resetForm()
}

async function handleTestConnection() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  testing.value = true
  testResult.value = null
  try {
    const result = await store.testConnection({ ...form })
    testResult.value = result
  } catch (e: any) {
    testResult.value = { success: false, message: e?.message || String(e), error: e?.message || String(e) }
  } finally {
    testing.value = false
  }
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    if (isEditing.value && form.id) {
      await store.updateProfile(form.id, { ...form })
      ElMessage.success('连接配置已更新')
    } else {
      await store.createProfile({ ...form })
      ElMessage.success('连接配置已创建')
    }
    formVisible.value = false
    resetForm()
  } catch (e: any) {
    ElMessage.error(`保存失败: ${e?.message || e}`)
  } finally {
    saving.value = false
  }
}

async function handleConnect(profile: MongoConnectionProfile) {
  connectingId.value = profile.id
  try {
    await store.connect(profile.id)
    ElMessage.success(`已连接到 ${profile.name}`)
    router.push('/mongo/workspace')
  } catch (e: any) {
    ElMessage.error(`连接失败: ${e?.message || e}`)
  } finally {
    connectingId.value = null
  }
}

async function handleDisconnect(profile: MongoConnectionProfile) {
  connectingId.value = profile.id
  try {
    await store.disconnect(profile.id)
    ElMessage.success(`已断开 ${profile.name}`)
  } catch (e: any) {
    ElMessage.error(`断开失败: ${e?.message || e}`)
  } finally {
    connectingId.value = null
  }
}

async function handleDelete(profile: MongoConnectionProfile) {
  try {
    await ElMessageBox.confirm(
      `确定要删除连接配置 "${profile.name}" 吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    await store.deleteProfile(profile.id)
    ElMessage.success('连接配置已删除')
    if (selectedProfileId.value === profile.id) {
      selectedProfileId.value = null
    }
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(`删除失败: ${e?.message || e}`)
    }
  }
}

onMounted(() => {
  store.loadProfiles()
})
</script>

<style scoped>
.mongo-connection-manager {
  display: flex;
  height: 100%;
  gap: 0;
  background: #f5f7fa;
}

.left-panel {
  width: 340px;
  min-width: 280px;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.right-panel {
  flex: 1;
  background: #fff;
  overflow-y: auto;
  padding: 0 0 20px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 12px;
  border-bottom: 1px solid #e4e7ed;
}

.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.profile-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.profile-item {
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f2f5;
  transition: background 0.15s;
}

.profile-item:hover {
  background: #f5f7fa;
}

.profile-item.active {
  background: #ecf5ff;
}

.profile-info {
  margin-bottom: 6px;
}

.profile-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.profile-host {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-status {
  margin-bottom: 8px;
}

.profile-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.connection-form {
  padding: 20px 24px;
}

.ssh-collapse {
  margin-bottom: 18px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
}

.ssh-title {
  display: flex;
  align-items: center;
  font-size: 14px;
}

.test-result {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 4px;
  margin-bottom: 16px;
  font-size: 13px;
}

.test-result.success {
  background: #f0f9eb;
  color: #67c23a;
  border: 1px solid #c2e7b0;
}

.test-result.error {
  background: #fef0f0;
  color: #f56c6c;
  border: 1px solid #fbc4c4;
}

.form-actions {
  margin-top: 8px;
}

.form-actions :deep(.el-form-item__content) {
  display: flex;
  gap: 10px;
}
</style>
