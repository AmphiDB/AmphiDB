<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, TitleComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { Loading } from '@element-plus/icons-vue'
import { GetMonitoringSnapshot } from '../../api/monitoring'
import type { MonitoringSnapshot, DataPoint } from '../../api/monitoring'
import { useConnectionStore } from '../../stores/connection'
import { useMongoConnectionStore } from '../../stores/mongoConnection'

use([LineChart, GridComponent, TooltipComponent, TitleComponent, LegendComponent, CanvasRenderer])

const mysqlStore = useConnectionStore()
const mongoStore = useMongoConnectionStore()

const mysqlProfileId = computed(() => mysqlStore.isConnected ? mysqlStore.currentConnection?.id ?? '' : '')
const mongoProfileId = computed(() => mongoStore.currentProfileId ?? '')

const snapshots = ref<Record<string, MonitoringSnapshot>>({})
let pollTimer: ReturnType<typeof setInterval> | null = null

function formatTime(ts: string): string {
  if (!ts) return ''
  const d = new Date(ts)
  if (isNaN(d.getTime())) return ts
  return `${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}:${String(d.getSeconds()).padStart(2,'0')}`
}

async function fetchAll() {
  const ids: string[] = []
  if (mysqlProfileId.value) ids.push(mysqlProfileId.value)
  if (mongoProfileId.value) ids.push(mongoProfileId.value)
  for (const id of ids) {
    try {
      const snap = await GetMonitoringSnapshot(id, '')
      if (snap) snapshots.value = { ...snapshots.value, [id]: snap }
    } catch {
      // session not yet started or no data, ignore
    }
  }
}

function startPolling() { fetchAll(); pollTimer = setInterval(fetchAll, 2000) }
function stopPolling() { if (pollTimer) { clearInterval(pollTimer); pollTimer = null } }

onMounted(startPolling)
onUnmounted(stopPolling)
watch([mysqlProfileId, mongoProfileId], fetchAll)

function lineOption(title: string, color: string, pts: DataPoint[], key: keyof DataPoint) {
  return {
    title: { text: title, left: 'center', textStyle: { fontSize: 12 } },
    tooltip: { trigger: 'axis' },
    grid: { left: 48, right: 12, top: 36, bottom: 24 },
    xAxis: { type: 'category', data: pts.map(p => formatTime(p.timestamp)), axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', axisLabel: { fontSize: 10 } },
    series: [{ type: 'line', smooth: true, data: pts.map(p => +(p[key] as number).toFixed(2)), lineStyle: { color }, itemStyle: { color }, areaStyle: { color: color + '22' } }],
  }
}

function latest(snap: MonitoringSnapshot | undefined): DataPoint | null {
  if (!snap || !snap.dataPoints.length) return null
  return snap.dataPoints[snap.dataPoints.length - 1]
}

const mysqlSnap = computed(() => snapshots.value[mysqlProfileId.value])
const mongoSnap = computed(() => snapshots.value[mongoProfileId.value])
const mysqlLatest = computed(() => latest(mysqlSnap.value))
const mongoLatest = computed(() => latest(mongoSnap.value))
const hasMysql = computed(() => !!mysqlProfileId.value)
const hasMongo = computed(() => !!mongoProfileId.value)
const hasAny = computed(() => hasMysql.value || hasMongo.value)
</script>

<template>
  <div class="monitoring-tab">
    <div v-if="!hasAny" class="no-data">
      <el-empty description="暂无连接，请先建立 MySQL 或 MongoDB 连接" />
    </div>

    <div v-else class="sections">

      <!-- ── MySQL ── -->
      <template v-if="hasMysql">
        <div class="section-title">
          <el-tag type="primary" size="small" effect="dark">MySQL</el-tag>
          <span class="conn-name">{{ mysqlStore.currentConnection?.name }}</span>
          <span v-if="!mysqlSnap" class="hint">等待数据...</span>
        </div>
        <div class="metric-cards">
          <div class="card">
            <div class="card-label">QPS</div>
            <div class="card-value">{{ mysqlLatest ? mysqlLatest.qps.toFixed(1) : '—' }}</div>
          </div>
          <div class="card">
            <div class="card-label">TPS</div>
            <div class="card-value">{{ mysqlLatest ? mysqlLatest.tps.toFixed(1) : '—' }}</div>
          </div>
          <div class="card">
            <div class="card-label">连接数</div>
            <div class="card-value">{{ mysqlLatest ? mysqlLatest.threadsConnected : '—' }}</div>
          </div>
          <div class="card" :class="{ warn: mysqlLatest && mysqlLatest.threadsRunning > 10 }">
            <div class="card-label">运行队列</div>
            <div class="card-value">{{ mysqlLatest ? mysqlLatest.threadsRunning : '—' }}</div>
          </div>
          <div class="card" :class="{ danger: mysqlLatest && mysqlLatest.innodbBufHitRate < 95 }">
            <div class="card-label">缓存命中率</div>
            <div class="card-value">{{ mysqlLatest ? mysqlLatest.innodbBufHitRate.toFixed(1) + '%' : '—' }}</div>
          </div>
          <div class="card" :class="{ warn: mysqlLatest && mysqlLatest.innodbRowLockWaits > 0 }">
            <div class="card-label">行锁等待</div>
            <div class="card-value">{{ mysqlLatest ? mysqlLatest.innodbRowLockWaits : '—' }}</div>
          </div>
        </div>
        <div v-if="mysqlSnap && mysqlSnap.dataPoints.length > 0" class="charts-row">
          <div class="chart-box"><v-chart :option="lineOption('QPS', '#409EFF', mysqlSnap.dataPoints, 'qps')" autoresize /></div>
          <div class="chart-box"><v-chart :option="lineOption('TPS', '#67C23A', mysqlSnap.dataPoints, 'tps')" autoresize /></div>
          <div class="chart-box"><v-chart :option="lineOption('连接数', '#E6A23C', mysqlSnap.dataPoints, 'threadsConnected')" autoresize /></div>
          <div class="chart-box"><v-chart :option="lineOption('运行队列', '#F56C6C', mysqlSnap.dataPoints, 'threadsRunning')" autoresize /></div>
        </div>
      </template>

      <!-- ── MongoDB ── -->
      <template v-if="hasMongo">
        <div class="section-title" :style="hasMysql ? 'margin-top:24px' : ''">
          <el-tag type="success" size="small" effect="dark">MongoDB</el-tag>
          <span class="conn-name">{{ mongoStore.currentProfile?.name }}</span>
          <span v-if="!mongoSnap" class="hint">等待数据...</span>
        </div>
        <div class="metric-cards">
          <div class="card">
            <div class="card-label">QPS</div>
            <div class="card-value">{{ mongoLatest ? mongoLatest.qps.toFixed(1) : '—' }}</div>
          </div>
          <div class="card">
            <div class="card-label">TPS</div>
            <div class="card-value">{{ mongoLatest ? mongoLatest.tps.toFixed(1) : '—' }}</div>
          </div>
          <div class="card">
            <div class="card-label">连接数</div>
            <div class="card-value">{{ mongoLatest ? mongoLatest.mongoConnections : '—' }}</div>
          </div>
          <div class="card" :class="{ warn: mongoLatest && mongoLatest.mongoPageFaults > 0 }">
            <div class="card-label">Page Faults</div>
            <div class="card-value">{{ mongoLatest ? mongoLatest.mongoPageFaults : '—' }}</div>
          </div>
          <div class="card">
            <div class="card-label">物理内存 (MB)</div>
            <div class="card-value">{{ mongoLatest ? mongoLatest.mongoMemResident : '—' }}</div>
          </div>
          <div class="card" :class="{ warn: mongoLatest && mongoLatest.mongoGlobalLock > 0 }">
            <div class="card-label">锁等待队列</div>
            <div class="card-value">{{ mongoLatest ? mongoLatest.mongoGlobalLock : '—' }}</div>
          </div>
        </div>
        <!-- Charts area: always render when snap exists -->
        <div v-if="mongoSnap">
          <div v-if="mongoSnap.dataPoints.length > 0" class="charts-row">
            <div class="chart-box"><v-chart :option="lineOption('QPS', '#409EFF', mongoSnap.dataPoints, 'qps')" autoresize /></div>
            <div class="chart-box"><v-chart :option="lineOption('TPS', '#67C23A', mongoSnap.dataPoints, 'tps')" autoresize /></div>
            <div class="chart-box"><v-chart :option="lineOption('连接数', '#E6A23C', mongoSnap.dataPoints, 'mongoConnections')" autoresize /></div>
            <div class="chart-box"><v-chart :option="lineOption('锁等待', '#F56C6C', mongoSnap.dataPoints, 'mongoGlobalLock')" autoresize /></div>
            <div class="chart-box"><v-chart :option="lineOption('Page Faults', '#909399', mongoSnap.dataPoints, 'mongoPageFaults')" autoresize /></div>
            <div class="chart-box"><v-chart :option="lineOption('物理内存 (MB)', '#9B59B6', mongoSnap.dataPoints, 'mongoMemResident')" autoresize /></div>
          </div>
          <div v-else class="chart-waiting">
            <el-icon class="is-loading"><Loading /></el-icon>&nbsp;等待采集数据（约 2 秒）...
          </div>
        </div>
      </template>

    </div>
  </div>
</template>

<style scoped>
.monitoring-tab { padding: 16px; overflow-y: auto; }
.no-data { display: flex; align-items: center; justify-content: center; height: 300px; }
.sections { display: flex; flex-direction: column; gap: 12px; }
.section-title { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.conn-name { color: var(--el-text-color-secondary); font-weight: 400; }
.hint { color: var(--el-text-color-placeholder); font-size: 12px; font-weight: 400; }
.metric-cards { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 12px; }
.card { background: var(--el-fill-color-light); border: 1px solid var(--el-border-color-lighter); border-radius: 6px; padding: 10px 16px; min-width: 110px; text-align: center; }
.card.warn { border-color: var(--el-color-warning); background: var(--el-color-warning-light-9); }
.card.danger { border-color: var(--el-color-danger); background: var(--el-color-danger-light-9); }
.card-label { font-size: 11px; color: var(--el-text-color-secondary); margin-bottom: 4px; }
.card-value { font-size: 20px; font-weight: 700; color: var(--el-text-color-primary); }
.charts-row { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; }
.chart-box { height: 180px; border: 1px solid var(--el-border-color-lighter); border-radius: 4px; padding: 4px; }
.chart-waiting { display: flex; align-items: center; padding: 16px; color: var(--el-text-color-secondary); font-size: 13px; }
</style>
