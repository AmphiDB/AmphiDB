<template>
  <div class="unified-conn-mgr">
    <!-- Left panel: unified list -->
    <div class="left-panel">
      <div class="panel-header">
        <span class="panel-title">连接管理</span>
        <el-button type="primary" size="small" @click="handleCreate">
          <el-icon><Plus /></el-icon>新建
        </el-button>
      </div>

      <!-- DB type filter tabs -->
      <div class="type-tabs">
        <el-radio-group v-model="filterType" size="small">
          <el-radio-button value="all">全部</el-radio-button>
          <el-radio-button value="mysql">MySQL</el-radio-button>
          <el-radio-button value="mongodb">MongoDB</el-radio-button>
        </el-radio-group>
      </div>

      <div class="profile-list" v-loading="loadingList">
        <div v-if="filteredProfiles.length === 0" class="list-empty">
          <el-empty description="暂无连接配置" :image-size="60" />
        </div>

        <div
          v-for="item in filteredProfiles"
          :key="item.key"
          class="profile-item"
          :class="{ active: selectedKey === item.key }"
          @click="handleSelect(item)"
        >
          <div class="item-left">
            <el-tag :type="item.dbType === 'mysql' ? 'primary' : 'success'" size="small" class="db-tag">
              {{ item.dbType === 'mysql' ? 'MySQL' : 'MongoDB' }}
            </el-tag>
            <div class="item-info">
              <div class="item-name">{{ item.name }}</div>
              <div class="item-host">{{ item.hostLabel }}</div>
            </div>
          </div>
          <div class="item-right">
            <el-tag :type="item.connected ? 'success' : 'info'" size="small">
              {{ item.connected ? '已连接' : '未连接' }}
            </el-tag>
          </div>
        </div>
      </div>
    </div>

    <!-- Right panel: form or detail -->
    <div class="right-panel">
      <template v-if="formVisible">
        <div class="panel-header">
          <span class="panel-title">{{ isEditing ? '编辑连接' : '新建连接' }}</span>
          <el-button size="small" @click="cancelForm"><el-icon><Close /></el-icon></el-button>
        </div>

        <el-form ref="formRef" :model="form" :rules="formRules" label-width="110px" class="conn-form">
          <!-- DB type selector (only on create) -->
          <el-form-item label="数据库类型" v-if="!isEditing">
            <el-radio-group v-model="form.dbType">
              <el-radio value="mysql">MySQL</el-radio>
              <el-radio value="mongodb">MongoDB</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="数据库类型" v-else>
            <el-tag :type="form.dbType === 'mysql' ? 'primary' : 'success'">
              {{ form.dbType === 'mysql' ? 'MySQL' : 'MongoDB' }}
            </el-tag>
          </el-form-item>

          <el-form-item label="连接名称" prop="name">
            <el-input v-model="form.name" placeholder="请输入连接名称" clearable />
          </el-form-item>

          <!-- MongoDB: URI mode toggle -->
          <el-form-item v-if="form.dbType === 'mongodb'" label="连接模式">
            <el-radio-group v-model="form.useUri">
              <el-radio :value="false">直连模式</el-radio>
              <el-radio :value="true">URI 模式</el-radio>
            </el-radio-group>
          </el-form-item>

          <!-- URI mode (MongoDB only) -->
          <template v-if="form.dbType === 'mongodb' && form.useUri">
            <el-form-item label="连接 URI" prop="uri">
              <el-input v-model="form.uri" placeholder="mongodb://user:pass@host:27017/dbname" type="textarea" :rows="2" />
            </el-form-item>
          </template>

          <!-- Direct mode fields (both MySQL and MongoDB direct) -->
          <template v-if="form.dbType === 'mysql' || (form.dbType === 'mongodb' && !form.useUri)">
            <el-form-item label="主机地址" prop="host">
              <el-input v-model="form.host" placeholder="localhost" clearable />
            </el-form-item>
            <el-form-item label="端口" prop="port">
              <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" style="width:100%" />
            </el-form-item>
            <el-form-item label="用户名" prop="username">
              <el-input v-model="form.username" placeholder="用户名" clearable />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="form.password" type="password" show-password placeholder="密码" clearable />
            </el-form-item>
          </template>

          <!-- MySQL-specific -->
          <template v-if="form.dbType === 'mysql'">
            <el-form-item label="数据库">
              <el-input v-model="form.database" placeholder="默认数据库（可选）" clearable />
            </el-form-item>
            <el-form-item label="字符集">
              <el-select v-model="form.charset" style="width:100%">
                <el-option label="utf8mb4" value="utf8mb4" />
                <el-option label="utf8" value="utf8" />
                <el-option label="latin1" value="latin1" />
                <el-option label="gbk" value="gbk" />
              </el-select>
            </el-form-item>
          </template>

          <!-- MongoDB-specific (direct mode) -->
          <template v-if="form.dbType === 'mongodb' && !form.useUri">
            <el-form-item label="认证数据库">
              <el-input v-model="form.authDb" placeholder="admin" clearable />
            </el-form-item>
          </template>

          <el-form-item label="超时（秒）">
            <el-input-number v-model="form.timeout" :min="1" :max="60" controls-position="right" style="width:100%" />
          </el-form-item>

          <!-- SSH tunnel -->
          <el-collapse v-model="sshOpen" class="ssh-collapse">
            <el-collapse-item name="ssh">
              <template #title>
                <div class="ssh-title">
                  <el-checkbox v-model="form.sshEnabled" @click.stop />
                  <span style="margin-left:8px">SSH 隧道</span>
                </div>
              </template>
              <template v-if="form.sshEnabled">
                <el-form-item label="SSH 主机" prop="sshHost">
                  <el-input v-model="form.sshHost" placeholder="ssh.example.com" clearable />
                </el-form-item>
                <el-form-item label="SSH 端口">
                  <el-input-number v-model="form.sshPort" :min="1" :max="65535" controls-position="right" style="width:100%" />
                </el-form-item>
                <el-form-item label="SSH 用户名" prop="sshUsername">
                  <el-input v-model="form.sshUsername" placeholder="ubuntu" clearable />
                </el-form-item>
                <el-form-item label="SSH 密码">
                  <el-input v-model="form.sshPassword" type="password" show-password placeholder="（可选）" clearable />
                </el-form-item>
                <el-form-item label="私钥路径">
                  <el-input v-model="form.sshKeyPath" placeholder="/path/to/key.pem（可选）" clearable />
                </el-form-item>
              </template>
            </el-collapse-item>
          </el-collapse>

          <!-- Test result -->
          <div v-if="testResult" class="test-result" :class="testResult.success ? 'ok' : 'err'">
            <el-icon v-if="testResult.success"><CircleCheck /></el-icon>
            <el-icon v-else><CircleClose /></el-icon>
            <span>{{ testResult.success ? `连接成功${testResult.serverVersion ? ' — ' + testResult.serverVersion : ''}` : (testResult.error || testResult.message) }}</span>
          </div>

          <el-form-item class="form-btns">
            <el-button :loading="testing" @click="handleTest">测试连接</el-button>
            <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
            <el-button @click="cancelForm">取消</el-button>
          </el-form-item>
        </el-form>
      </template>

      <!-- Detail / actions panel when a profile is selected -->
      <template v-else-if="selectedItem">
        <div class="panel-header">
          <span class="panel-title">{{ selectedItem.name }}</span>
        </div>
        <div class="detail-panel">
          <div class="detail-info">
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="类型">
                <el-tag :type="selectedItem.dbType === 'mysql' ? 'primary' : 'success'" size="small">
                  {{ selectedItem.dbType === 'mysql' ? 'MySQL' : 'MongoDB' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="连接信息">{{ selectedItem.hostLabel }}</el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="selectedItem.connected ? 'success' : 'info'" size="small">
                  {{ selectedItem.connected ? '已连接' : '未连接' }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <div class="detail-actions">
            <el-button
              v-if="!selectedItem.connected"
              type="primary"
              :loading="connectingKey === selectedItem.key"
              @click="handleConnect(selectedItem)"
            >连接</el-button>
            <el-button
              v-else
              type="warning"
              :loading="connectingKey === selectedItem.key"
              @click="handleDisconnect(selectedItem)"
            >断开</el-button>
            <el-button @click="handleEdit(selectedItem)">编辑</el-button>
            <el-button type="danger" :disabled="selectedItem.connected" @click="handleDelete(selectedItem)">删除</el-button>
          </div>
        </div>
      </template>

      <template v-else>
        <div class="right-empty">
          <el-empty description="选择左侧连接查看详情，或新建连接" :image-size="80" />
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Close, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import type { FormInstance } from 'element-plus'
import { useConnectionStore } from '../stores/connection'
import { useMongoConnectionStore } from '../stores/mongoConnection'
import { ConnectionAPI } from '../api'
import { MongoConnectionAPI } from '../api/mongo'
import type { ConnectionProfile } from '../types/api'
import type { MongoConnectionProfile, MongoTestResult } from '../types/mongo'

const router = useRouter()
const mysqlStore = useConnectionStore()
const mongoStore = useMongoConnectionStore()

// ── Unified profile item ───────────────────────────────────────────────────────
interface ProfileItem {
  key: string          // 'mysql:id' or 'mongo:id'
  dbType: 'mysql' | 'mongodb'
  id: string
  name: string
  hostLabel: string
  connected: boolean
  raw: ConnectionProfile | MongoConnectionProfile
}

const loadingList = ref(false)
const filterType = ref<'all' | 'mysql' | 'mongodb'>('all')

const mysqlItems = computed<ProfileItem[]>(() =>
  (mysqlStore.profiles || []).map(p => ({
    key: `mysql:${p.id}`,
    dbType: 'mysql',
    id: p.id,
    name: p.name,
    hostLabel: `${p.username}@${p.host}:${p.port}`,
    connected: mysqlStore.currentConnection?.id === p.id,
    raw: p,
  }))
)

const mongoItems = computed<ProfileItem[]>(() =>
  (mongoStore.profiles || []).map(p => ({
    key: `mongo:${p.id}`,
    dbType: 'mongodb',
    id: p.id,
    name: p.name,
    hostLabel: p.useUri ? (p.uri || 'URI 模式') : `${p.host}:${p.port}`,
    connected: mongoStore.isConnected(p.id),
    raw: p,
  }))
)

const allProfiles = computed<ProfileItem[]>(() => [...mysqlItems.value, ...mongoItems.value])

const filteredProfiles = computed<ProfileItem[]>(() => {
  if (filterType.value === 'mysql') return mysqlItems.value
  if (filterType.value === 'mongodb') return mongoItems.value
  return allProfiles.value
})

// ── Selection ──────────────────────────────────────────────────────────────────
const selectedKey = ref<string | null>(null)
const selectedItem = computed(() => allProfiles.value.find(p => p.key === selectedKey.value) ?? null)

const handleSelect = (item: ProfileItem) => {
  selectedKey.value = item.key
  formVisible.value = false
}

// ── Form state ─────────────────────────────────────────────────────────────────
const formVisible = ref(false)
const isEditing = ref(false)
const formRef = ref<FormInstance>()
const saving = ref(false)
const testing = ref(false)
const testResult = ref<{ success: boolean; message: string; serverVersion?: string; error?: string } | null>(null)
const sshOpen = ref<string[]>([])
const connectingKey = ref<string | null>(null)

interface FormModel {
  dbType: 'mysql' | 'mongodb'
  id: string
  name: string
  // common
  host: string; port: number; username: string; password: string; timeout: number
  sshEnabled: boolean; sshHost: string; sshPort: number; sshUsername: string; sshPassword: string; sshKeyPath: string
  // mysql
  database: string; charset: string
  // mongodb
  useUri: boolean; uri: string; authDb: string
}

const defaultForm = (): FormModel => ({
  dbType: 'mysql', id: '', name: '',
  host: 'localhost', port: 3306, username: '', password: '', timeout: 10,
  sshEnabled: false, sshHost: '', sshPort: 22, sshUsername: '', sshPassword: '', sshKeyPath: '',
  database: '', charset: 'utf8mb4',
  useUri: false, uri: '', authDb: 'admin',
})

const form = reactive<FormModel>(defaultForm())

watch(() => form.dbType, (t) => {
  form.port = t === 'mysql' ? 3306 : 27017
  testResult.value = null
})

watch(() => form.sshEnabled, (v) => {
  if (v) sshOpen.value = ['ssh']
  else sshOpen.value = []
})

const formRules = computed(() => ({
  name: [{ required: true, message: '请输入连接名称', trigger: 'blur' }],
  host: form.dbType === 'mysql' || (form.dbType === 'mongodb' && !form.useUri)
    ? [{ required: true, message: '请输入主机地址', trigger: 'blur' }] : [],
  uri: form.dbType === 'mongodb' && form.useUri
    ? [{ required: true, message: '请输入连接 URI', trigger: 'blur' }] : [],
  username: form.dbType === 'mysql'
    ? [{ required: true, message: '请输入用户名', trigger: 'blur' }] : [],
}))

const handleCreate = () => {
  isEditing.value = false
  Object.assign(form, defaultForm())
  sshOpen.value = []
  testResult.value = null
  selectedKey.value = null
  formVisible.value = true
}

const handleEdit = (item: ProfileItem) => {
  isEditing.value = true
  testResult.value = null
  if (item.dbType === 'mysql') {
    const p = item.raw as ConnectionProfile
    Object.assign(form, defaultForm(), {
      dbType: 'mysql', id: p.id, name: p.name,
      host: p.host, port: p.port, username: p.username, password: p.password,
      database: p.database || '', charset: p.charset || 'utf8mb4', timeout: p.timeout || 10,
      sshEnabled: p.sshEnabled || false, sshHost: p.sshHost || '', sshPort: p.sshPort || 22,
      sshUsername: p.sshUsername || '', sshPassword: p.sshPassword || '', sshKeyPath: p.sshKeyPath || '',
    })
  } else {
    const p = item.raw as MongoConnectionProfile
    Object.assign(form, defaultForm(), {
      dbType: 'mongodb', id: p.id, name: p.name,
      host: p.host || '', port: p.port || 27017, username: p.username || '', password: p.password || '',
      useUri: p.useUri || false, uri: p.uri || '', authDb: p.authDb || 'admin', timeout: p.timeout || 10,
      sshEnabled: p.sshEnabled || false, sshHost: p.sshHost || '', sshPort: p.sshPort || 22,
      sshUsername: p.sshUsername || '', sshPassword: p.sshPassword || '', sshKeyPath: p.sshKeyPath || '',
    })
  }
  if (form.sshEnabled) sshOpen.value = ['ssh']
  formVisible.value = true
}

const cancelForm = () => { formVisible.value = false; testResult.value = null }

const handleTest = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  testing.value = true; testResult.value = null
  try {
    if (form.dbType === 'mysql') {
      await ConnectionAPI.testConnection(buildMysqlProfile())
      testResult.value = { success: true, message: '连接成功' }
    } else {
      const r = await MongoConnectionAPI.testConnection(buildMongoProfile())
      testResult.value = r
    }
  } catch (e: any) {
    testResult.value = { success: false, message: e?.message || String(e), error: e?.message }
  } finally { testing.value = false }
}

const handleSave = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  saving.value = true
  try {
    if (form.dbType === 'mysql') {
      const p = buildMysqlProfile()
      if (isEditing.value) { await ConnectionAPI.updateProfile(p.id, p); ElMessage.success('已更新') }
      else { await ConnectionAPI.createProfile(p); ElMessage.success('已创建') }
      await loadMysql()
    } else {
      const p = buildMongoProfile()
      if (isEditing.value) { await mongoStore.updateProfile(p.id, p); ElMessage.success('已更新') }
      else { await mongoStore.createProfile(p); ElMessage.success('已创建') }
    }
    formVisible.value = false
  } catch (e: any) { ElMessage.error(e?.message || '保存失败') }
  finally { saving.value = false }
}

const buildMysqlProfile = (): ConnectionProfile => ({
  id: form.id || `conn_${Date.now()}`,
  name: form.name, host: form.host, port: form.port,
  username: form.username, password: form.password,
  database: form.database, charset: form.charset, timeout: form.timeout,
  sshEnabled: form.sshEnabled, sshHost: form.sshHost, sshPort: form.sshPort,
  sshUsername: form.sshUsername, sshPassword: form.sshPassword, sshKeyPath: form.sshKeyPath,
} as ConnectionProfile)

const buildMongoProfile = (): MongoConnectionProfile => ({
  id: form.id || '', name: form.name,
  host: form.host, port: form.port, username: form.username, password: form.password,
  authDb: form.authDb || 'admin', useUri: form.useUri, uri: form.uri, timeout: form.timeout,
  sshEnabled: form.sshEnabled, sshHost: form.sshHost, sshPort: form.sshPort,
  sshUsername: form.sshUsername, sshPassword: form.sshPassword, sshKeyPath: form.sshKeyPath,
} as MongoConnectionProfile)

// ── Connect / Disconnect ───────────────────────────────────────────────────────
const handleConnect = async (item: ProfileItem) => {
  connectingKey.value = item.key
  try {
    if (item.dbType === 'mysql') {
      await ConnectionAPI.connect(item.id)
      mysqlStore.setCurrentConnection(item.raw as ConnectionProfile)
      ElMessage.success(`已连接到 ${item.name}`)
      router.push('/workspace')
    } else {
      await mongoStore.connect(item.id)
      ElMessage.success(`已连接到 ${item.name}`)
      router.push('/mongo/workspace')
    }
  } catch (e: any) { ElMessage.error(e?.message || '连接失败') }
  finally { connectingKey.value = null }
}

const handleDisconnect = async (item: ProfileItem) => {
  connectingKey.value = item.key
  try {
    if (item.dbType === 'mysql') {
      await ConnectionAPI.disconnect(item.id)
      mysqlStore.setCurrentConnection(null)
    } else {
      await mongoStore.disconnect(item.id)
    }
    ElMessage.success(`已断开 ${item.name}`)
  } catch (e: any) { ElMessage.error(e?.message || '断开失败') }
  finally { connectingKey.value = null }
}

const handleDelete = async (item: ProfileItem) => {
  try {
    await ElMessageBox.confirm(`确定要删除连接配置 "${item.name}" 吗？`, '确认删除', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
    if (item.dbType === 'mysql') { await ConnectionAPI.deleteProfile(item.id); mysqlStore.removeProfile(item.id) }
    else { await mongoStore.deleteProfile(item.id) }
    if (selectedKey.value === item.key) selectedKey.value = null
    ElMessage.success('已删除')
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || '删除失败') }
}

// ── Load ───────────────────────────────────────────────────────────────────────
const loadMysql = async () => {
  try { const ps = await ConnectionAPI.listProfiles(); mysqlStore.setProfiles(ps || []) } catch { mysqlStore.setProfiles([]) }
}

onMounted(async () => {
  loadingList.value = true
  await Promise.all([loadMysql(), mongoStore.loadProfiles()])
  loadingList.value = false
})
</script>

<style scoped>
.unified-conn-mgr { display: flex; height: 100%; overflow: hidden; }

/* ── Left panel ── */
.left-panel { width: 300px; flex-shrink: 0; border-right: 1px solid #e4e7ed; display: flex; flex-direction: column; background: #fff; }
.panel-header { display: flex; align-items: center; justify-content: space-between; padding: 14px 16px 10px; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.panel-title { font-size: 15px; font-weight: 600; color: #303133; }
.type-tabs { padding: 8px 12px; border-bottom: 1px solid #f0f2f5; flex-shrink: 0; }
.profile-list { flex: 1; overflow-y: auto; padding: 4px 0; }
.list-empty { padding: 20px; }

.profile-item { display: flex; align-items: center; justify-content: space-between; padding: 10px 14px; cursor: pointer; border-bottom: 1px solid #f5f7fa; transition: background 0.15s; }
.profile-item:hover { background: #f5f7fa; }
.profile-item.active { background: #ecf5ff; }
.item-left { display: flex; align-items: center; gap: 8px; overflow: hidden; }
.db-tag { flex-shrink: 0; }
.item-info { overflow: hidden; }
.item-name { font-size: 13px; font-weight: 500; color: #303133; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.item-host { font-size: 11px; color: #909399; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.item-right { flex-shrink: 0; margin-left: 8px; }

/* ── Right panel ── */
.right-panel { flex: 1; overflow-y: auto; background: #fff; display: flex; flex-direction: column; }
.right-empty { flex: 1; display: flex; align-items: center; justify-content: center; }

.conn-form { padding: 16px 24px; }
.ssh-collapse { margin-bottom: 16px; border: 1px solid #e4e7ed; border-radius: 4px; }
.ssh-title { display: flex; align-items: center; font-size: 14px; }
.form-btns :deep(.el-form-item__content) { display: flex; gap: 10px; }

.test-result { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border-radius: 4px; margin-bottom: 14px; font-size: 13px; }
.test-result.ok { background: #f0f9eb; color: #67c23a; border: 1px solid #c2e7b0; }
.test-result.err { background: #fef0f0; color: #f56c6c; border: 1px solid #fbc4c4; }

/* ── Detail panel ── */
.detail-panel { padding: 16px 24px; display: flex; flex-direction: column; gap: 16px; }
.detail-actions { display: flex; gap: 10px; flex-wrap: wrap; }
</style>
