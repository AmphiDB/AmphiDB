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

        <!-- MySQL 已连接时显示 -->
        <template v-if="isConnected">
          <div class="section-label">
            <el-tag type="primary" size="small" effect="plain">MySQL</el-tag>
            <span class="section-conn-name">{{ mysqlConnName }}</span>
          </div>
          <el-menu-item index="/workspace">
            <el-icon><FolderOpened /></el-icon>
            <span>数据库工作台</span>
          </el-menu-item>
          <el-menu-item index="/query">
            <el-icon><EditPen /></el-icon>
            <span>SQL 查询</span>
          </el-menu-item>
          <el-menu-item index="/sync">
            <el-icon><Refresh /></el-icon>
            <span>结构同步</span>
          </el-menu-item>
          <el-menu-item index="/data-sync">
            <el-icon><Switch /></el-icon>
            <span>数据同步</span>
          </el-menu-item>
        </template>

        <!-- MongoDB 已连接时显示 -->
        <template v-if="isMongoConnected">
          <div class="section-label">
            <el-tag type="success" size="small" effect="plain">MongoDB</el-tag>
            <span class="section-conn-name">{{ mongoConnName }}</span>
          </div>
          <el-menu-item index="/mongo/workspace">
            <el-icon><FolderOpened /></el-icon>
            <span>数据库工作台</span>
          </el-menu-item>
          <el-menu-item index="/mongo/query">
            <el-icon><EditPen /></el-icon>
            <span>聚合查询</span>
          </el-menu-item>
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
          <template v-if="isConnected">
            <el-tag type="primary" size="small" effect="dark" class="conn-badge">
              MySQL: {{ currentConnection?.name }}
            </el-tag>
          </template>
          <template v-if="isMongoConnected">
            <el-tag type="success" size="small" effect="dark" class="conn-badge">
              MongoDB: {{ mongoConnName }}
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
import { useConnectionStore } from '../stores/connection'
import { useDatabaseStore } from '../stores/database'
import { useMongoConnectionStore } from '../stores/mongoConnection'
import logoIcon from '../assets/images/appicon.png'
import {
  Connection, FolderOpened, EditPen, Refresh, Document,
  Coin, WarningFilled, InfoFilled, Switch,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const mysqlStore = useConnectionStore()
const dbStore = useDatabaseStore()
const mongoStore = useMongoConnectionStore()

const activeMenu = computed(() => route.path)
const isConnected = computed(() => mysqlStore.isConnected)
const isMongoConnected = computed(() => !!mongoStore.currentProfileId)

const currentConnection = computed(() => mysqlStore.currentConnection)
const mysqlConnName = computed(() => mysqlStore.currentConnection?.name ?? '')
const mongoConnName = computed(() => mongoStore.currentProfile?.name ?? '')

const currentDatabase = computed(() => dbStore.currentDatabase)
const currentTable = computed(() => dbStore.currentTable)

const handleMenuSelect = (index: string) => router.push(index)
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
}

.section-conn-name {
  font-size: 11px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100px;
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

.conn-badge { cursor: default; }

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
