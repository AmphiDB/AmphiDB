<template>
  <el-container class="main-layout">
    <el-aside width="210px" class="sidebar">
      <div class="logo">
        <img :src="logoIcon" alt="AmphiDB" class="logo-icon" />
        <span class="logo-text">AmphiDB</span>
      </div>

      <el-menu :default-active="activeMenu" class="sidebar-menu" @select="handleMenuSelect">

        <!-- 始终显示：连接管理 -->
        <el-menu-item index="/connections">
          <el-icon><Connection /></el-icon>
          <span>连接管理</span>
        </el-menu-item>

        <!-- MySQL 多连接：每个活跃连接显示一个分组 -->
        <template v-for="conn in mysqlActiveList" :key="conn.id">
          <div
            class="section-label"
            :class="{ 'section-active': currentConnection?.id === conn.id }"
            @click="switchMysqlConnection(conn)"
          >
            <el-tag type="primary" size="small" effect="plain">MySQL</el-tag>
            <span class="section-conn-name" :title="conn.name">{{ conn.name }}</span>
            <el-icon v-if="currentConnection?.id === conn.id" class="section-check"><Check /></el-icon>
            <el-icon class="section-close" title="关闭连接" @click.stop="handleDisconnectMysql(conn)"><Close /></el-icon>
          </div>
          <template v-if="currentConnection?.id === conn.id">
            <el-menu-item :index="'/workspace:' + conn.id">
              <el-icon><FolderOpened /></el-icon>
              <span>数据库工作台</span>
            </el-menu-item>
            <el-menu-item :index="'/query:' + conn.id">
              <el-icon><EditPen /></el-icon>
              <span>SQL 查询</span>
            </el-menu-item>
            <el-menu-item :index="'/sync:' + conn.id">
              <el-icon><Refresh /></el-icon>
              <span>结构同步</span>
            </el-menu-item>
            <el-menu-item :index="'/data-sync:' + conn.id">
              <el-icon><Switch /></el-icon>
              <span>数据同步</span>
            </el-menu-item>
          </template>
        </template>

        <!-- MongoDB 多连接：每个活跃连接显示一个分组 -->
        <template v-for="conn in mongoActiveList" :key="conn.id">
          <div
            class="section-label"
            :class="{ 'section-active': mongoStore.currentProfileId === conn.id }"
            @click="switchMongoConnection(conn)"
          >
            <el-tag type="success" size="small" effect="plain">MongoDB</el-tag>
            <span class="section-conn-name" :title="conn.name">{{ conn.name }}</span>
            <el-icon v-if="mongoStore.currentProfileId === conn.id" class="section-check"><Check /></el-icon>
            <el-icon class="section-close" title="关闭连接" @click.stop="handleDisconnectMongo(conn)"><Close /></el-icon>
          </div>
          <template v-if="mongoStore.currentProfileId === conn.id">
            <el-menu-item :index="'/mongo/workspace:' + conn.id">
              <el-icon><FolderOpened /></el-icon>
              <span>数据库工作台</span>
            </el-menu-item>
            <el-menu-item :index="'/mongo/query:' + conn.id">
              <el-icon><EditPen /></el-icon>
              <span>聚合查询</span>
            </el-menu-item>
          </template>
        </template>

        <!-- 未连接时的提示 -->
        <div v-if="!isConnected && !isMongoConnected" class="no-conn-hint">
          <el-icon><InfoFilled /></el-icon>
          <span>请先建立连接</span>
        </div>

        <div class="menu-divider" />

        <!-- 始终显示：日志查看 -->
        <el-menu-item index="/logs">
          <el-icon><Document /></el-icon>
          <span>日志查看</span>
        </el-menu-item>

      </el-menu>
    </el-aside>

    <el-container>
      <!-- Top bar: active connection badges -->
      <el-header class="header">
        <div class="header-left">
          <template v-for="conn in mysqlActiveList" :key="'mysql-badge-' + conn.id">
            <el-tag
              type="primary"
              size="small"
              :effect="currentConnection?.id === conn.id ? 'dark' : 'plain'"
              class="conn-badge"
              closable
              @click="switchMysqlConnection(conn)"
              @close="handleDisconnectMysql(conn)"
            >
              MySQL: {{ conn.name }}
            </el-tag>
          </template>
          <template v-for="conn in mongoActiveList" :key="'mongo-badge-' + conn.id">
            <el-tag
              type="success"
              size="small"
              :effect="mongoStore.currentProfileId === conn.id ? 'dark' : 'plain'"
              class="conn-badge"
              closable
              @click="switchMongoConnection(conn)"
              @close="handleDisconnectMongo(conn)"
            >
              MongoDB: {{ conn.name }}
            </el-tag>
          </template>
          <span v-if="!isConnected && !isMongoConnected" class="no-conn-text">
            <el-icon><WarningFilled /></el-icon> 未连接
          </span>
        </div>
        <div class="header-right">
          <span v-if="currentDatabase" class="db-info">
            <el-icon><Coin /></el-icon>{{ currentDatabase }}
          </span>
          <span v-if="currentTable" class="table-info">
            <el-icon><Document /></el-icon>{{ currentTable }}
          </span>
        </div>
      </el-header>

      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useConnectionStore } from '../stores/connection'
import { useDatabaseStore } from '../stores/database'
import { useMongoConnectionStore } from '../stores/mongoConnection'
import { ConnectionAPI } from '../api'
import type { ConnectionProfile } from '../types/api'
import type { MongoConnectionProfile } from '../types/mongo'
import logoIcon from '../assets/images/appicon.png'
import {
  Connection, FolderOpened, EditPen, Refresh, Document,
  Coin, WarningFilled, InfoFilled, Switch, Check, Close,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const mysqlStore = useConnectionStore()
const dbStore = useDatabaseStore()
const mongoStore = useMongoConnectionStore()

const activeMenu = computed(() => route.path)
const isConnected = computed(() => mysqlStore.isConnected)
const isMongoConnected = computed(() => mongoStore.isConnected)

const currentConnection = computed(() => mysqlStore.currentConnection)
const mysqlActiveList = computed(() => mysqlStore.activeConnectionList)
const mongoActiveList = computed(() => mongoStore.activeConnectionList)

const currentDatabase = computed(() => dbStore.currentDatabase)
const currentTable = computed(() => dbStore.currentTable)

// 切换 MySQL 当前连接
const switchMysqlConnection = (conn: ConnectionProfile) => {
  mysqlStore.setCurrentConnection(conn)
}

// 切换 MongoDB 当前连接
const switchMongoConnection = (conn: MongoConnectionProfile) => {
  mongoStore.setCurrentProfile(conn.id)
}

// 关闭 MySQL 连接
const handleDisconnectMysql = async (conn: ConnectionProfile) => {
  try {
    await ConnectionAPI.disconnect(conn.id)
    mysqlStore.removeActiveConnection(conn.id)
    ElMessage.success(`已断开连接 ${conn.name}`)
  } catch (e: any) {
    ElMessage.error(`断开连接失败: ${e?.message || e}`)
  }
}

// 关闭 MongoDB 连接
const handleDisconnectMongo = async (conn: MongoConnectionProfile) => {
  try {
    await mongoStore.disconnect(conn.id)
    ElMessage.success(`已断开连接 ${conn.name}`)
  } catch (e: any) {
    ElMessage.error(`断开连接失败: ${e?.message || e}`)
  }
}

const handleMenuSelect = (index: string) => {
  // 路由中可能包含连接ID后缀（如 /workspace:conn-id），需要提取实际路径
  const colonIdx = index.indexOf(':')
  const path = colonIdx > 0 ? index.substring(0, colonIdx) : index
  router.push(path)
}
</script>

<style scoped>
.main-layout { height: 100vh; width: 100vw; }

/* ── Sidebar ── */
.sidebar {
  background-color: #1e2a3a;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 14px;
  border-bottom: 1px solid #2d3f52;
  flex-shrink: 0;
  transform: translateY(8px);
}

.logo-icon {
  width: 24px;
  height: 24px;
  object-fit: contain;
  flex-shrink: 0;
}
.logo-text { font-size: 14px; font-weight: 600; color: #e2e8f0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.sidebar-menu {
  flex: 1;
  overflow-y: auto;
  border: none;
  background-color: #1e2a3a;
}

/* Section label (MySQL / MongoDB group header) */
.section-label {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px 4px;
  margin-top: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
  border-radius: 4px;
  margin-left: 4px;
  margin-right: 4px;
}

.section-label:hover {
  background-color: #2d3f52;
}

.section-label.section-active {
  background-color: rgba(59, 130, 246, 0.15);
}

.section-conn-name {
  font-size: 11px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100px;
  flex: 1;
}

.section-check {
  font-size: 12px;
  color: #3b82f6;
  flex-shrink: 0;
}

.section-close {
  font-size: 12px;
  color: #64748b;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.2s, color 0.2s;
  margin-left: 2px;
}

.section-label:hover .section-close {
  opacity: 1;
}

.section-close:hover {
  color: #f56c6c;
}

/* No connection hint */
.no-conn-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 16px;
  font-size: 12px;
  color: #64748b;
}

.menu-divider {
  height: 1px;
  background-color: #2d3f52;
  margin: 6px 12px;
}

/* Menu item styles */
.sidebar-menu :deep(.el-menu-item) {
  color: #94a3b8;
  height: 40px;
  line-height: 40px;
  font-size: 13px;
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background-color: #2d3f52 !important;
  color: #e2e8f0;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background-color: #3b82f6 !important;
  color: #fff;
}

.sidebar-menu :deep(.el-menu-item.is-disabled) {
  color: #475569;
  opacity: 0.6;
}

/* ── Header ── */
.header {
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 48px;
}

.header-left, .header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.conn-badge { cursor: pointer; }
.conn-badge:hover { opacity: 0.8; }

.no-conn-text {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #9ca3af;
}

.db-info, .table-info {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #606266;
}

.table-info {
  padding-left: 10px;
  border-left: 1px solid #e5e7eb;
}

/* ── Main content ── */
.main-content {
  background-color: #f5f7fa;
  overflow: auto;
  padding: 0;
}
</style>
