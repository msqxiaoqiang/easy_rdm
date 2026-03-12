<template>
  <div class="latency-view">
    <!-- 控制栏 -->
    <div class="latency-toolbar">
      <label class="latency-label">{{ $t('latency.pingCount') }}</label>
      <input v-model.number="pingCount" type="number" min="5" max="100" class="latency-num-input" />
      <button class="tool-btn primary" :disabled="testing" @click="runTest">
        {{ testing ? $t('common.loading') : $t('latency.runTest') }}
      </button>
    </div>

    <!-- 结果 -->
    <div class="latency-body" v-if="result">
      <!-- 统计卡片 -->
      <div class="stats-grid">
        <div class="stat-card">
          <span class="stat-label">{{ $t('latency.min') }}</span>
          <span class="stat-value">{{ result.min_ms.toFixed(2) }} ms</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">{{ $t('latency.avg') }}</span>
          <span class="stat-value">{{ result.avg_ms.toFixed(2) }} ms</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">{{ $t('latency.max') }}</span>
          <span class="stat-value">{{ result.max_ms.toFixed(2) }} ms</span>
        </div>
        <div class="stat-card">
          <span class="stat-label">{{ $t('latency.samples') }}</span>
          <span class="stat-value">{{ result.count }} / {{ pingCount }}</span>
        </div>
      </div>

      <!-- 延迟图表 -->
      <div class="chart-section" v-if="result.samples && result.samples.length > 1">
        <span class="chart-title">{{ $t('latency.distribution') }}</span>
        <canvas ref="chartCanvas" class="latency-chart"></canvas>
      </div>

      <!-- LATENCY LATEST -->
      <div class="latency-section" v-if="result.latency">
        <span class="section-title">LATENCY LATEST</span>
        <pre class="latency-pre">{{ formatLatency(result.latency) }}</pre>
      </div>
    </div>

    <div v-else class="latency-empty">
      {{ $t('latency.hint') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { showMessage } from '@/utils/platform'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()

const pingCount = ref(20)
const testing = ref(false)
const result = ref<any>(null)
const chartCanvas = ref<HTMLCanvasElement>()

async function runTest() {
  if (testing.value) return
  testing.value = true
  result.value = null
  try {
    const res = await request<any>('latency_test', {
      params: {
        conn_id: props.connId,
        count: Math.round(Number(pingCount.value)) || 20,
      },
    })
    result.value = res.data
    nextTick(() => drawChart())
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    testing.value = false
  }
}

function drawChart() {
  const canvas = chartCanvas.value
  if (!canvas || !result.value?.samples) return

  const samples: number[] = result.value.samples
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const dpr = window.devicePixelRatio || 1
  const rect = canvas.getBoundingClientRect()
  canvas.width = rect.width * dpr
  canvas.height = rect.height * dpr
  ctx.scale(dpr, dpr)

  const w = rect.width
  const h = rect.height
  const pad = { top: 10, right: 10, bottom: 20, left: 40 }
  const plotW = w - pad.left - pad.right
  const plotH = h - pad.top - pad.bottom

  const maxVal = Math.max(...samples) * 1.1 || 1
  const minVal = 0

  ctx.clearRect(0, 0, w, h)

  // 网格线
  ctx.strokeStyle = getComputedStyle(canvas).getPropertyValue('--color-border-1') || '#e5e5e5'
  ctx.lineWidth = 0.5
  for (let i = 0; i <= 4; i++) {
    const y = pad.top + (plotH / 4) * i
    ctx.beginPath()
    ctx.moveTo(pad.left, y)
    ctx.lineTo(w - pad.right, y)
    ctx.stroke()

    // Y 轴标签
    const val = maxVal - (maxVal - minVal) * (i / 4)
    ctx.fillStyle = getComputedStyle(canvas).getPropertyValue('--color-text-3') || '#999'
    ctx.font = '10px monospace'
    ctx.textAlign = 'right'
    ctx.fillText(val.toFixed(1), pad.left - 4, y + 3)
  }

  // 折线
  if (samples.length < 2) return
  const stepX = plotW / (samples.length - 1)

  ctx.beginPath()
  ctx.strokeStyle = getComputedStyle(canvas).getPropertyValue('--color-primary') || '#4080ff'
  ctx.lineWidth = 1.5
  for (let i = 0; i < samples.length; i++) {
    const x = pad.left + stepX * i
    const y = pad.top + plotH * (1 - (samples[i] - minVal) / (maxVal - minVal))
    if (i === 0) ctx.moveTo(x, y)
    else ctx.lineTo(x, y)
  }
  ctx.stroke()

  // 数据点
  ctx.fillStyle = getComputedStyle(canvas).getPropertyValue('--color-primary') || '#4080ff'
  for (let i = 0; i < samples.length; i++) {
    const x = pad.left + stepX * i
    const y = pad.top + plotH * (1 - (samples[i] - minVal) / (maxVal - minVal))
    ctx.beginPath()
    ctx.arc(x, y, 2, 0, Math.PI * 2)
    ctx.fill()
  }
}

function formatLatency(data: any): string {
  if (!data) return '(empty)'
  if (Array.isArray(data) && data.length === 0) return '(no latency events recorded)'
  try {
    return JSON.stringify(data, null, 2)
  } catch {
    return String(data)
  }
}

watch(() => props.connId, () => {
  result.value = null
})
</script>

<style scoped>
.latency-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.latency-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  flex-shrink: 0;
}

.latency-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  white-space: nowrap;
}

.latency-num-input {
  width: 60px;
  height: 28px;
  padding: 0 var(--spacing-xs);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  background: var(--color-bg-1);
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  text-align: center;
  outline: none;
}

.latency-num-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary-light);
}

.latency-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-md);
}

.latency-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-4);
  font-size: var(--font-size-sm);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--spacing-md);
  margin-bottom: var(--spacing-lg);
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-md);
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-md);
}

.stat-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}

.stat-value {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
}

.chart-section {
  margin-bottom: var(--spacing-lg);
}

.chart-title {
  display: block;
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  font-weight: 600;
  margin-bottom: var(--spacing-sm);
}

.latency-chart {
  width: 100%;
  height: 160px;
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-sm);
  background: var(--color-bg-1);
}

.latency-section {
  margin-bottom: var(--spacing-md);
}

.section-title {
  display: block;
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  font-weight: 600;
  margin-bottom: var(--spacing-sm);
}

.latency-pre {
  padding: var(--spacing-sm);
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-sm);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  overflow-x: auto;
  white-space: pre-wrap;
  margin: 0;
}
</style>
