<script setup lang="ts">
import MonitoringTab from '../components/logs/MonitoringTab.vue'
import SlowQueryTab from '../components/logs/SlowQueryTab.vue'
</script>

<template>
  <div class="log-viewer">
    <el-tabs type="border-card" class="log-tabs">
      <!-- 服务器监控 -->
      <el-tab-pane label="服务器监控">
        <MonitoringTab />
      </el-tab-pane>

      <!-- 慢查询日志 -->
      <el-tab-pane label="慢查询日志">
        <SlowQueryTab />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.log-viewer {
  padding: 16px;
  height: 100%;
  box-sizing: border-box;
}

.log-tabs {
  height: 100%;
}

.app-log-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 300px;
}
</style>
