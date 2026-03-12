<template>
  <div class="status-view">
    <div class="status-section" v-if="connState.status === 'connected'">
      <!-- 第一行：连接名 + 标签 + 自动刷新 -->
      <div class="info-header">
        <span class="conn-title">{{ connName }}</span>
        <div class="info-tags">
          <a-tooltip v-if="summary.redis_version" :content="$t('status.redisVersion')" mini>
            <a-tag size="small" color="arcoblue">v{{ summary.redis_version }}</a-tag>
          </a-tooltip>
          <a-tooltip v-if="summary.redis_mode" :content="$t('status.mode')" mini>
            <a-tag size="small" color="green">{{ summary.redis_mode }}</a-tag>
          </a-tooltip>
          <a-tooltip v-if="summary.role" :content="$t('status.role')" mini>
            <a-tag size="small" color="orangered">{{ summary.role }}</a-tag>
          </a-tooltip>
        </div>
        <div class="header-spacer" />
        <a-popover trigger="hover" position="br" :content-style="{ padding: '12px 16px' }">
          <a-button size="small" class="refresh-icon-btn" :class="{ spinning: autoRefreshEnabled }">
            <template #icon><IconSync :size="16" /></template>
          </a-button>
          <template #content>
            <div class="refresh-popover">
              <div class="refresh-row">
                <span class="refresh-label">{{ $t('status.autoRefresh') }}</span>
                <a-switch v-model="autoRefreshEnabled" size="small" @change="onToggleAutoRefresh" />
              </div>
              <div v-if="autoRefreshEnabled" class="refresh-row">
                <span class="refresh-label">{{ $t('settings.heartbeatInterval') }}</span>
                <a-select
                  v-model="autoRefreshInterval"
                  size="mini"
                  style="width: 80px"
                  @change="onChangeInterval"
                >
                  <a-option v-for="opt in intervalOptions" :key="opt" :value="opt">{{ opt }}s</a-option>
                </a-select>
              </div>
            </div>
          </template>
        </a-popover>
      </div>

      <!-- 第二行：运行时统计（4 项均分） -->
      <div class="info-stats">
        <div class="stat-cell">
          <span class="stat-label">{{ $t('status.uptime') }}</span>
          <span class="stat-value">{{ formatUptime(summary.uptime_in_seconds) }}</span>
        </div>
        <div class="stat-cell">
          <span class="stat-label">{{ $t('status.clients') }}</span>
          <span class="stat-value">
            {{ summary.connected_clients || '0' }}
            <span class="client-link-btn" @click="openClientModal">
              <IconLink :size="12" />
            </span>
          </span>
        </div>
        <div class="stat-cell">
          <span class="stat-label">{{ $t('status.memory') }}</span>
          <span class="stat-value">{{ summary.used_memory_human || '-' }}</span>
        </div>
        <div class="stat-cell">
          <span class="stat-label">{{ $t('status.totalKeys') }}</span>
          <span class="stat-value">{{ summary.total_keys ?? '-' }}</span>
        </div>
      </div>

      <!-- 主 Tab：活动状态 / 状态信息 -->
      <div class="main-tabs">
        <div
          :class="['main-tab', { active: activeStatusTab === 'activity' }]"
          @click="activeStatusTab = 'activity'"
        >{{ $t('status.activityStatus') }}</div>
        <div
          :class="['main-tab', { active: activeStatusTab === 'info' }]"
          @click="activeStatusTab = 'info'"
        >{{ $t('status.statusInfo') }}</div>
      </div>

      <!-- 活动状态 Tab -->
      <div v-show="activeStatusTab === 'activity'" class="charts-grid">
        <div class="chart-card">
          <div class="chart-header">
            <span class="chart-title">{{ $t('status.opsPerSec') }}</span>
            <span class="chart-value">{{ latestStats.ops }}</span>
          </div>
          <canvas ref="chartOps" class="chart-canvas"></canvas>
        </div>
        <div class="chart-card">
          <div class="chart-header">
            <span class="chart-title">{{ $t('status.clients') }}</span>
            <span class="chart-value">{{ latestStats.clients }}</span>
          </div>
          <canvas ref="chartClients" class="chart-canvas"></canvas>
        </div>
        <div class="chart-card">
          <div class="chart-header">
            <span class="chart-title">{{ $t('status.memory') }}</span>
            <span class="chart-value">{{ latestStats.memoryMB }} MB</span>
          </div>
          <canvas ref="chartMemory" class="chart-canvas"></canvas>
        </div>
        <div class="chart-card">
          <div class="chart-header">
            <span class="chart-title">{{ $t('status.networkIn') }} / {{ $t('status.networkOut') }}</span>
            <span class="chart-value">{{ latestStats.inputKbps }} / {{ latestStats.outputKbps }} kbps</span>
          </div>
          <canvas ref="chartNetwork" class="chart-canvas"></canvas>
        </div>
      </div>

      <!-- 状态信息 Tab：左侧 section 切换 + 右侧表格 -->
      <div v-show="activeStatusTab === 'info'" class="info-layout">
        <div v-if="sectionNames.length" class="section-sidebar">
          <div
            v-for="name in sectionNames"
            :key="name"
            :class="['section-item', { active: activeSectionTab === name }]"
            @click="activeSectionTab = name"
          >{{ name }}</div>
        </div>
        <div class="section-content">
          <table v-if="activeSectionItems" class="detail-table">
            <tbody>
              <tr v-for="(val, key) in activeSectionItems" :key="key">
                <td class="detail-key">{{ key }}</td>
                <td class="detail-val">{{ val }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
    <div v-else-if="connState.status === 'connecting'" class="status-empty">
      <p>{{ $t('common.loading') }}</p>
    </div>
    <div v-else class="status-empty">
      <p>{{ $t('connection.notConnected') }}</p>
    </div>
    <!-- 客户端列表弹窗 -->
    <a-modal
      :visible="clientModalVisible"
      :title="$t('server.clients')"
      :footer="false"
      :width="900"
      :mask-closable="true"
      unmount-on-close
      @cancel="clientModalVisible = false"
    >
      <div class="client-modal-toolbar">
        <a-input
          v-model="clientSearch"
          :placeholder="$t('common.search')"
          size="small"
          allow-clear
          style="width: 240px"
        >
          <template #prefix><icon-search :size="14" /></template>
        </a-input>
        <a-button size="small" @click="loadClients">
          <template #icon><icon-refresh :size="14" /></template>
          {{ $t('common.refresh') }}
        </a-button>
      </div>
      <div class="client-table-wrap">
        <a-table
          :data="filteredClients"
          :pagination="false"
          :bordered="false"
          size="small"
          :scroll="{ y: 400 }"
          row-key="id"
        >
          <template #columns>
            <a-table-column :title="$t('server.clientId')" data-index="id" :width="80" :sortable="idSortable">
              <template #cell="{ record }"><span class="mono">{{ record.id }}</span></template>
            </a-table-column>
            <a-table-column :title="$t('server.clientAddr')" data-index="addr" :width="160">
              <template #cell="{ record }"><span class="mono">{{ record.addr }}</span></template>
            </a-table-column>
            <a-table-column :title="$t('server.clientDb')" data-index="db" :width="60" />
            <a-table-column :title="$t('server.clientCmd')" data-index="cmd" :width="120">
              <template #cell="{ record }"><span class="mono">{{ record.cmd }}</span></template>
            </a-table-column>
            <a-table-column :title="$t('server.clientAge')" data-index="age" :width="90" :sortable="ageSortable">
              <template #cell="{ record }">{{ formatUptime(record.age) }}</template>
            </a-table-column>
            <a-table-column :title="$t('server.clientIdle')" data-index="idle" :width="90" :sortable="idleSortable">
              <template #cell="{ record }">{{ formatUptime(record.idle) }}</template>
            </a-table-column>
          </template>
        </a-table>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, onUnmounted, nextTick } from 'vue'
import { IconSearch, IconRefresh, IconSync, IconLink } from '@arco-design/web-vue/es/icon'
import { request } from '../../utils/request'
import { showMessage } from '@/utils/platform'
import { useConnectionStore } from '../../stores/connection'
import { useI18n } from 'vue-i18n'

const STORAGE_KEY = 'easy_rdm_auto_refresh'
const MAX_HISTORY = 20
const intervalOptions = [3, 5, 10, 30, 60]

function loadRefreshSettings(): { enabled: boolean; interval: number } {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      return {
        enabled: typeof parsed.enabled === 'boolean' ? parsed.enabled : true,
        interval: intervalOptions.includes(parsed.interval) ? parsed.interval : 5,
      }
    }
  } catch (_e) { /* ignore */ }
  return { enabled: true, interval: 5 }
}

function saveRefreshSettings(enabled: boolean, interval: number) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ enabled, interval }))
}

const props = defineProps<{ connId: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const connState = computed(() => connectionStore.getConnState(props.connId))
const connName = computed(() => connectionStore.connections.find(c => c.id === props.connId)?.name || props.connId)

// ========== 客户端列表弹窗 ==========
const clientModalVisible = ref(false)
const clientList = ref<Record<string, string>[]>([])
const clientSearch = ref('')

const filteredClients = computed(() => {
  const q = clientSearch.value.toLowerCase()
  if (!q) return clientList.value
  return clientList.value.filter(c =>
    (c.addr || '').toLowerCase().includes(q)
  )
})

function openClientModal() {
  clientModalVisible.value = true
  loadClients()
}

async function loadClients() {
  try {
    const res = await request<Record<string, string>[]>('client_list', { params: { conn_id: props.connId } })
    clientList.value = res.data || []
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}



const numSorter = (field: string) => (a: Record<string, string>, b: Record<string, string>, extra: { direction: string }) => {
  const diff = (parseInt(a[field]) || 0) - (parseInt(b[field]) || 0)
  return extra.direction === 'descend' ? -diff : diff
}

const idSortable = { sortDirections: ['ascend', 'descend'] as ('ascend' | 'descend')[], defaultSortOrder: 'ascend' as const, sorter: numSorter('id') }
const ageSortable = { sortDirections: ['ascend', 'descend'] as ('ascend' | 'descend')[], sorter: numSorter('age') }
const idleSortable = { sortDirections: ['ascend', 'descend'] as ('ascend' | 'descend')[], sorter: numSorter('idle') }

const summary = ref<Record<string, any>>({})
const sections = ref<Record<string, Record<string, string>>>({})

// 顶层 Tab
const activeStatusTab = ref<'activity' | 'info'>('activity')
// 状态信息内的 section sub-tab
const activeSectionTab = ref('')
const sectionNames = computed(() => Object.keys(sections.value))
const activeSectionItems = computed(() => sections.value[activeSectionTab.value])

// 活动状态图表
interface StatsPoint {
  ts: number
  ops: number
  clients: number
  memoryMB: number
  inputKbps: number
  outputKbps: number
}
const statsHistory = ref<StatsPoint[]>([])
const chartOps = ref<HTMLCanvasElement>()
const chartClients = ref<HTMLCanvasElement>()
const chartMemory = ref<HTMLCanvasElement>()
const chartNetwork = ref<HTMLCanvasElement>()
const savedSettings = loadRefreshSettings()
const autoRefreshEnabled = ref(savedSettings.enabled)
const autoRefreshInterval = ref(savedSettings.interval)
let timer: ReturnType<typeof setInterval> | null = null

const latestStats = computed(() => {
  const h = statsHistory.value
  if (!h.length) return { ops: 0, clients: 0, memoryMB: 0, inputKbps: 0, outputKbps: 0 }
  const last = h[h.length - 1]
  return {
    ops: last.ops,
    clients: last.clients,
    memoryMB: last.memoryMB,
    inputKbps: last.inputKbps,
    outputKbps: last.outputKbps,
  }
})

// 防竞态：connId 变化时递增
let statusGeneration = 0

// connId 变化时保存旧数据、加载新数据
let prevConnId = ''
watch(() => props.connId, (newId) => {
  statusGeneration++
  if (prevConnId) {
    connectionStore.saveStatsCache(prevConnId, statsHistory.value)
  }
  prevConnId = newId
  summary.value = {}
  sections.value = {}
  activeSectionTab.value = ''
  // 从缓存恢复图表数据
  const cached = connectionStore.getStatsCache(newId)
  statsHistory.value = cached ? [...cached] : []
  loadStatus()
  restartAutoRefresh()
}, { immediate: true })

// 连接状态变为 connected 时立即加载（修复刷新页面后等 5 秒的 bug）
watch(() => connState.value.status, (status) => {
  if (status === 'connected') {
    loadStatus()
  }
})

async function loadStatus() {
  if (connState.value.status !== 'connected') return
  const gen = statusGeneration
  try {
    const res = await request<any>('get_server_status', {
      params: { conn_id: props.connId },
    })
    if (gen !== statusGeneration) return // connId 已切换，丢弃过期结果
    if (res.data?.summary) {
      summary.value = res.data.summary
      // 收集图表数据点
      const s = res.data.summary
      const point: StatsPoint = {
        ts: Date.now(),
        ops: parseFloat(s.ops_per_sec) || 0,
        clients: parseInt(s.connected_clients) || 0,
        memoryMB: Math.round((parseInt(s.used_memory) || 0) / 1048576 * 100) / 100,
        inputKbps: parseFloat(s.input_kbps) || 0,
        outputKbps: parseFloat(s.output_kbps) || 0,
      }
      statsHistory.value = [...statsHistory.value.slice(-(MAX_HISTORY - 1)), point]
      nextTick(renderCharts)
    }
    if (res.data?.sections) {
      sections.value = res.data.sections
      if (!activeSectionTab.value && sectionNames.value.length) {
        activeSectionTab.value = sectionNames.value[0]
      }
    }
  } catch (_e) { /* ignore */ }
}

function restartAutoRefresh() {
  stopAutoRefresh()
  if (autoRefreshEnabled.value) {
    timer = setInterval(loadStatus, autoRefreshInterval.value * 1000)
  }
}

function stopAutoRefresh() {
  if (timer) { clearInterval(timer); timer = null }
}

function onToggleAutoRefresh() {
  saveRefreshSettings(autoRefreshEnabled.value, autoRefreshInterval.value)
  restartAutoRefresh()
}

function onChangeInterval() {
  saveRefreshSettings(autoRefreshEnabled.value, autoRefreshInterval.value)
  restartAutoRefresh()
}

onMounted(() => {
  const cached = connectionStore.getStatsCache(props.connId)
  if (cached) statsHistory.value = [...cached]
  nextTick(renderCharts)
})

onBeforeUnmount(() => {
  connectionStore.saveStatsCache(props.connId, statsHistory.value)
})

onUnmounted(stopAutoRefresh)

// ========== 图表渲染 ==========
function formatTime(ts: number): string {
  const d = new Date(ts)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

function niceMax(v: number): number {
  if (v <= 0) return 1
  const mag = Math.pow(10, Math.floor(Math.log10(v)))
  const norm = v / mag
  if (norm <= 1) return mag
  if (norm <= 2) return 2 * mag
  if (norm <= 5) return 5 * mag
  return 10 * mag
}

function initCanvas(canvas: HTMLCanvasElement) {
  const dpr = window.devicePixelRatio || 1
  const rect = canvas.getBoundingClientRect()
  const w = rect.width
  const h = rect.height
  canvas.width = w * dpr
  canvas.height = h * dpr
  const ctx = canvas.getContext('2d')!
  ctx.scale(dpr, dpr)
  ctx.clearRect(0, 0, w, h)
  return { ctx, w, h }
}

function renderCharts() {
  const h = statsHistory.value
  const timestamps = h.map(p => p.ts)
  drawAreaChart(chartOps.value, h.map(p => p.ops), timestamps, '#4080ff', 'rgba(64,128,255,0.15)')
  drawAreaChart(chartClients.value, h.map(p => p.clients), timestamps, '#00b42a', 'rgba(0,180,42,0.15)')
  drawAreaChart(chartMemory.value, h.map(p => p.memoryMB), timestamps, '#722ed1', 'rgba(114,46,209,0.15)')
  drawDualAreaChart(
    chartNetwork.value,
    h.map(p => p.inputKbps), h.map(p => p.outputKbps), timestamps,
    '#4080ff', 'rgba(64,128,255,0.10)',
    '#f53f3f', 'rgba(245,63,63,0.10)',
  )
}

function drawAreaChart(canvas: HTMLCanvasElement | undefined, data: number[], timestamps: number[], strokeColor: string, fillColor: string) {
  if (!canvas) return
  const { ctx, w, h } = initCanvas(canvas)
  const textColor = getComputedStyle(canvas).getPropertyValue('--color-text-4') || 'rgba(128,128,128,0.6)'
  const gridColor = 'rgba(128,128,128,0.08)'
  const pad = { top: 12, bottom: 28, left: 50, right: 16 }
  const cw = w - pad.left - pad.right
  const ch = h - pad.top - pad.bottom

  const max = data.length ? niceMax(Math.max(...data)) : 1
  const tickCount = 4

  // Y 轴刻度
  ctx.font = '10px -apple-system, BlinkMacSystemFont, sans-serif'
  ctx.fillStyle = textColor
  ctx.textAlign = 'right'
  for (let i = 0; i <= tickCount; i++) {
    const y = pad.top + (ch / tickCount) * i
    const val = max - (max / tickCount) * i
    ctx.fillText(formatChartValue(val), pad.left - 8, y + 4)
    // 水平网格线
    ctx.beginPath()
    ctx.strokeStyle = gridColor
    ctx.lineWidth = 1
    ctx.setLineDash([3, 3])
    ctx.moveTo(pad.left, y)
    ctx.lineTo(w - pad.right, y)
    ctx.stroke()
    ctx.setLineDash([])
  }

  // X 轴时间标签（只显示首、中、尾）
  ctx.textAlign = 'center'
  ctx.fillStyle = textColor
  if (timestamps.length) {
    const indices = timestamps.length <= 3
      ? timestamps.map((_, i) => i)
      : [0, Math.floor(timestamps.length / 2), timestamps.length - 1]
    for (const i of indices) {
      const x = data.length > 1 ? pad.left + (i / (data.length - 1)) * cw : pad.left + cw / 2
      ctx.fillText(formatTime(timestamps[i]), x, h - 6)
    }
  }

  if (!data.length) return

  // 渐变填充
  const gradient = ctx.createLinearGradient(0, pad.top, 0, pad.top + ch)
  gradient.addColorStop(0, fillColor)
  gradient.addColorStop(1, 'rgba(255,255,255,0)')

  // 面积
  ctx.beginPath()
  ctx.moveTo(pad.left, pad.top + ch)
  for (let i = 0; i < data.length; i++) {
    const x = data.length > 1 ? pad.left + (i / (data.length - 1)) * cw : pad.left + cw / 2
    const y = pad.top + ch - (data[i] / max) * ch
    ctx.lineTo(x, y)
  }
  ctx.lineTo(data.length > 1 ? pad.left + cw : pad.left + cw / 2, pad.top + ch)
  ctx.closePath()
  ctx.fillStyle = gradient
  ctx.fill()

  // 平滑折线
  ctx.beginPath()
  for (let i = 0; i < data.length; i++) {
    const x = data.length > 1 ? pad.left + (i / (data.length - 1)) * cw : pad.left + cw / 2
    const y = pad.top + ch - (data[i] / max) * ch
    if (i === 0) ctx.moveTo(x, y)
    else ctx.lineTo(x, y)
  }
  ctx.strokeStyle = strokeColor
  ctx.lineWidth = 2
  ctx.lineJoin = 'round'
  ctx.lineCap = 'round'
  ctx.stroke()

  // 最新数据点高亮
  if (data.length) {
    const lastIdx = data.length - 1
    const lx = data.length > 1 ? pad.left + (lastIdx / (data.length - 1)) * cw : pad.left + cw / 2
    const ly = pad.top + ch - (data[lastIdx] / max) * ch
    // 光晕
    ctx.beginPath()
    ctx.arc(lx, ly, 6, 0, Math.PI * 2)
    ctx.fillStyle = fillColor
    ctx.fill()
    // 实心点
    ctx.beginPath()
    ctx.arc(lx, ly, 3, 0, Math.PI * 2)
    ctx.fillStyle = strokeColor
    ctx.fill()
  }
}

function drawDualAreaChart(
  canvas: HTMLCanvasElement | undefined,
  data1: number[], data2: number[], timestamps: number[],
  stroke1: string, fill1: string,
  stroke2: string, fill2: string,
) {
  if (!canvas) return
  const { ctx, w, h } = initCanvas(canvas)
  const textColor = getComputedStyle(canvas).getPropertyValue('--color-text-4') || 'rgba(128,128,128,0.6)'
  const gridColor = 'rgba(128,128,128,0.08)'
  const pad = { top: 12, bottom: 28, left: 50, right: 16 }
  const cw = w - pad.left - pad.right
  const ch = h - pad.top - pad.bottom

  const allMax = data1.length ? niceMax(Math.max(...data1, ...data2)) : 1
  const tickCount = 4

  // Y 轴刻度
  ctx.font = '10px -apple-system, BlinkMacSystemFont, sans-serif'
  ctx.fillStyle = textColor
  ctx.textAlign = 'right'
  for (let i = 0; i <= tickCount; i++) {
    const y = pad.top + (ch / tickCount) * i
    const val = allMax - (allMax / tickCount) * i
    ctx.fillText(formatChartValue(val), pad.left - 8, y + 4)
    ctx.beginPath()
    ctx.strokeStyle = gridColor
    ctx.lineWidth = 1
    ctx.setLineDash([3, 3])
    ctx.moveTo(pad.left, y)
    ctx.lineTo(w - pad.right, y)
    ctx.stroke()
    ctx.setLineDash([])
  }

  // X 轴时间标签
  ctx.textAlign = 'center'
  ctx.fillStyle = textColor
  if (timestamps.length) {
    const indices = timestamps.length <= 3
      ? timestamps.map((_, i) => i)
      : [0, Math.floor(timestamps.length / 2), timestamps.length - 1]
    for (const i of indices) {
      const x = data1.length > 1 ? pad.left + (i / (data1.length - 1)) * cw : pad.left + cw / 2
      ctx.fillText(formatTime(timestamps[i]), x, h - 6)
    }
  }

  if (!data1.length) return

  // 绘制两条数据线
  for (const [data, stroke, fill] of [[data1, stroke1, fill1], [data2, stroke2, fill2]] as [number[], string, string][]) {
    const gradient = ctx.createLinearGradient(0, pad.top, 0, pad.top + ch)
    gradient.addColorStop(0, fill)
    gradient.addColorStop(1, 'rgba(255,255,255,0)')

    ctx.beginPath()
    ctx.moveTo(pad.left, pad.top + ch)
    for (let i = 0; i < data.length; i++) {
      const x = data.length > 1 ? pad.left + (i / (data.length - 1)) * cw : pad.left + cw / 2
      const y = pad.top + ch - (data[i] / allMax) * ch
      ctx.lineTo(x, y)
    }
    ctx.lineTo(data.length > 1 ? pad.left + cw : pad.left + cw / 2, pad.top + ch)
    ctx.closePath()
    ctx.fillStyle = gradient
    ctx.fill()

    ctx.beginPath()
    for (let i = 0; i < data.length; i++) {
      const x = data.length > 1 ? pad.left + (i / (data.length - 1)) * cw : pad.left + cw / 2
      const y = pad.top + ch - (data[i] / allMax) * ch
      if (i === 0) ctx.moveTo(x, y)
      else ctx.lineTo(x, y)
    }
    ctx.strokeStyle = stroke
    ctx.lineWidth = 2
    ctx.lineJoin = 'round'
    ctx.lineCap = 'round'
    ctx.stroke()

    // 最新数据点
    if (data.length) {
      const lastIdx = data.length - 1
      const lx = data.length > 1 ? pad.left + (lastIdx / (data.length - 1)) * cw : pad.left + cw / 2
      const ly = pad.top + ch - (data[lastIdx] / allMax) * ch
      ctx.beginPath()
      ctx.arc(lx, ly, 6, 0, Math.PI * 2)
      ctx.fillStyle = fill
      ctx.fill()
      ctx.beginPath()
      ctx.arc(lx, ly, 3, 0, Math.PI * 2)
      ctx.fillStyle = stroke
      ctx.fill()
    }
  }
}

function formatChartValue(v: number): string {
  if (v >= 1000000) return (v / 1000000).toFixed(1) + 'M'
  if (v >= 1000) return (v / 1000).toFixed(1) + 'K'
  return v % 1 === 0 ? String(v) : v.toFixed(1)
}

function formatUptime(seconds: string | number | undefined): string {
  if (!seconds) return '-'
  const s = typeof seconds === 'string' ? parseInt(seconds, 10) : seconds
  if (isNaN(s)) return '-'
  const days = Math.floor(s / 86400)
  if (days >= 365) {
    const years = Math.floor(days / 365)
    const remainDays = days % 365
    return `${years}${t('status.years')} ${remainDays}${t('status.days')}`
  }
  if (days >= 1) return `${days}${t('status.days')}`
  const hours = Math.floor(s / 3600)
  if (hours >= 1) return `${hours}${t('status.hours')}`
  const minutes = Math.floor(s / 60)
  if (minutes >= 1) return `${minutes}${t('status.minutes')}`
  return `${s}${t('status.seconds')}`
}
</script>

<style scoped>
.status-view {
  padding: var(--spacing-lg);
  height: 100%;
  overflow-y: auto;
}

.status-section {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.status-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-3);
}

/* ===== 第一行：连接名 + 标签 + 刷新 ===== */
.info-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-md);
  margin-bottom: var(--spacing-sm);
}

.conn-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.info-tags {
  display: flex;
  gap: var(--spacing-xs);
  flex-shrink: 0;
}

.header-spacer {
  flex: 1;
}

/* ===== 第二行：运行时统计（4 项均分） ===== */
.info-stats {
  display: flex;
  align-items: center;
  padding: var(--spacing-md) var(--spacing-lg);
  background: var(--color-fill-1);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-1);
  margin-bottom: var(--spacing-md);
}

.stat-cell {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.stat-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  white-space: nowrap;
}

.stat-value {
  font-size: var(--font-size-xl);
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
  font-weight: 700;
  white-space: nowrap;
}

.client-link-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  margin-left: 4px;
  border-radius: var(--radius-sm);
  color: var(--color-text-4);
  cursor: pointer;
  vertical-align: middle;
  transition: all var(--transition-fast);
}

.client-link-btn:hover {
  color: var(--color-primary);
  background: var(--color-primary-bg);
}

/* ===== 自动刷新图标 ===== */
.refresh-icon-btn {
  flex-shrink: 0;
  border-radius: 50% !important;
  width: 32px !important;
  height: 32px !important;
  padding: 0 !important;
}

.refresh-icon-btn.spinning :deep(.arco-icon) {
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.refresh-popover {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 180px;
}

.refresh-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-md);
}

.refresh-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-2);
  white-space: nowrap;
}

/* ===== 主 Tab 切换 ===== */
.main-tabs {
  display: flex;
  gap: 0;
  margin-bottom: var(--spacing-md);
  background: var(--color-fill-2);
  border-radius: var(--radius-md);
  padding: 3px;
}

.main-tab {
  flex: 1;
  text-align: center;
  padding: 6px 0;
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--color-text-3);
  cursor: pointer;
  border-radius: calc(var(--radius-md) - 2px);
  transition: all var(--transition-fast);
  user-select: none;
}

.main-tab:hover {
  color: var(--color-text-1);
}

.main-tab.active {
  background: var(--color-bg-2);
  color: var(--color-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

/* ===== Activity Charts ===== */
.charts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-md);
}

.chart-card {
  display: flex;
  flex-direction: column;
  padding: var(--spacing-md);
  background: var(--color-fill-1);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-1);
  transition: box-shadow var(--transition-fast);
}

.chart-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.chart-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: var(--spacing-sm);
}

.chart-title {
  font-size: var(--font-size-sm);
  color: var(--color-text-2);
  font-weight: 500;
}

.chart-value {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
}

.chart-canvas {
  width: 100%;
  height: 160px;
}

/* ===== 状态信息 - 左侧切换布局 ===== */
.info-layout {
  display: flex;
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-md);
  overflow: hidden;
  flex: 1;
  min-height: 300px;
}

.section-sidebar {
  width: 160px;
  flex-shrink: 0;
  background: var(--color-fill-1);
  border-right: 1px solid var(--color-border-1);
  overflow-y: auto;
  scrollbar-width: thin;
}

.section-item {
  padding: 10px 16px;
  font-size: var(--font-size-sm);
  color: var(--color-text-3);
  cursor: pointer;
  border-left: 3px solid transparent;
  transition: all var(--transition-fast);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.section-item:hover {
  color: var(--color-text-1);
  background: var(--color-fill-2);
}

.section-item.active {
  color: var(--color-primary);
  background: var(--color-primary-bg);
  border-left-color: var(--color-primary);
  font-weight: 500;
}

.section-content {
  flex: 1;
  overflow-y: auto;
  scrollbar-width: thin;
  min-width: 0;
}

/* ===== Detail Table ===== */
.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-sm);
}

.detail-table tr:hover {
  background: var(--color-fill-1);
}

.detail-key {
  width: 40%;
  padding: 8px 16px;
  color: var(--color-text-2);
  font-family: var(--font-family-mono);
  border-bottom: 1px solid var(--color-border-1);
  white-space: nowrap;
}

.detail-val {
  padding: 8px 16px;
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
  border-bottom: 1px solid var(--color-border-1);
  word-break: break-all;
}

/* ===== Client Modal ===== */
.client-modal-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-md);
}

.client-table-wrap :deep(.arco-table) {
  font-size: var(--font-size-xs);
}

.client-table-wrap .mono {
  font-family: var(--font-family-mono);
}
</style>
