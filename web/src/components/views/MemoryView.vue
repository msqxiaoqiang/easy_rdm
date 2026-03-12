<template>
  <div class="memory-view">
    <!-- 控制栏 -->
    <div class="memory-toolbar">
      <div class="toolbar-group">
        <a-button size="medium" type="primary" :disabled="scanning" @click="runScan">
          {{ scanning ? $t('common.loading') : $t('memory.scanBigKeys') }}
        </a-button>
        <a-button size="medium" :disabled="scanning" @click="runDistribution">
          {{ $t('memory.distribution') }}
        </a-button>
      </div>
      <div class="toolbar-sep"></div>
      <label class="toolbar-label">{{ $t('memory.pattern') }}</label>
      <a-input v-model="pattern" size="medium" class="toolbar-input" placeholder="*" :disabled="scanning" />
      <label class="toolbar-label">{{ $t('memory.scanLimit') }}</label>
      <a-input-number v-model="scanLimit" size="medium" class="toolbar-num" :min="100" :max="100000" :disabled="scanning" hide-button />
      <label class="toolbar-label">Top</label>
      <a-input-number v-model="topN" size="medium" class="toolbar-num" :min="10" :max="500" :disabled="scanning" hide-button />
    </div>

    <div class="memory-body">
      <!-- 大 Key 扫描结果 -->
      <template v-if="scanResult">
        <div class="result-header">
          {{ $t('memory.scannedKeys', { count: scanResult.scanned }) }}
          ·
          {{ $t('memory.totalMemory') }}: {{ formatBytes(scanResult.total_memory) }}
        </div>
        <div class="key-table-wrap">
          <a-table
            :data="scanResult.keys"
            :pagination="false"
            :bordered="false"
            size="small"
            :sticky-header="true"
          >
            <template #columns>
              <a-table-column title="Key" data-index="key">
                <template #cell="{ record }"><span class="mono cell-ellipsis" v-ellipsis-tip>{{ record.key }}</span></template>
              </a-table-column>
              <a-table-column :title="$t('memory.type')" data-index="type" :width="90">
                <template #cell="{ record }"><span :class="'type-tag type-' + record.type">{{ record.type }}</span></template>
              </a-table-column>
              <a-table-column :title="$t('memory.memoryUsage')" data-index="memory" :width="110">
                <template #cell="{ record }"><span class="mono" style="text-align:right;display:block">{{ formatBytes(record.memory) }}</span></template>
              </a-table-column>
              <a-table-column title="TTL" data-index="ttl" :width="80">
                <template #cell="{ record }"><span class="mono" style="text-align:right;display:block">{{ record.ttl === -1 ? '∞' : record.ttl + 's' }}</span></template>
              </a-table-column>
            </template>
            <template #empty>
              <div class="empty-state">{{ $t('common.noData') }}</div>
            </template>
          </a-table>
        </div>
      </template>

      <!-- 内存分布 -->
      <template v-if="distResult">
        <div class="dist-section">
          <h4 class="dist-title">{{ $t('memory.byType') }}</h4>
          <div class="dist-bars">
            <div v-for="item in distResult.by_type" :key="item.type" class="bar-row">
              <span class="bar-label">{{ item.type }}</span>
              <div class="bar-track">
                <div class="bar-fill" :style="{ width: typePercent(item.memory) + '%' }"></div>
              </div>
              <span class="bar-value">{{ formatBytes(item.memory) }} ({{ item.count }})</span>
            </div>
          </div>
        </div>

        <div class="dist-section">
          <h4 class="dist-title">{{ $t('memory.byPrefix') }}</h4>
          <div class="dist-bars">
            <div v-for="item in distResult.by_prefix" :key="item.prefix" class="bar-row">
              <span class="bar-label">{{ item.prefix }}</span>
              <div class="bar-track">
                <div class="bar-fill bar-prefix" :style="{ width: prefixPercent(item.memory) + '%' }"></div>
              </div>
              <span class="bar-value">{{ formatBytes(item.memory) }} ({{ item.count }})</span>
            </div>
          </div>
        </div>
      </template>

      <!-- 空状态 -->
      <div v-if="!scanResult && !distResult && !scanning" class="memory-empty">
        {{ $t('memory.hint') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { showMessage } from '@/utils/platform'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()

const pattern = ref('*')
const scanLimit = ref(10000)
const topN = ref(100)
const scanning = ref(false)
const scanResult = ref<any>(null)
const distResult = ref<any>(null)

async function runScan() {
  if (scanning.value) return
  scanning.value = true
  scanResult.value = null
  try {
    const res = await request<any>('memory_scan', {
      params: {
        conn_id: props.connId,
        pattern: pattern.value,
        limit: scanLimit.value,
        top_n: topN.value,
      },
    })
    scanResult.value = res.data
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    scanning.value = false
  }
}

async function runDistribution() {
  if (scanning.value) return
  scanning.value = true
  distResult.value = null
  try {
    const res = await request<any>('memory_distribution', {
      params: {
        conn_id: props.connId,
        scan_limit: scanLimit.value,
      },
    })
    distResult.value = res.data
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    scanning.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

function typePercent(memory: number): number {
  if (!distResult.value?.by_type?.length) return 0
  const max = distResult.value.by_type[0].memory
  return max > 0 ? (memory / max) * 100 : 0
}

function prefixPercent(memory: number): number {
  if (!distResult.value?.by_prefix?.length) return 0
  const max = distResult.value.by_prefix[0].memory
  return max > 0 ? (memory / max) * 100 : 0
}

watch(() => props.connId, () => {
  scanResult.value = null
  distResult.value = null
})
</script>

<style scoped>
.memory-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.memory-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  flex-shrink: 0;
  flex-wrap: wrap;
}

.toolbar-group {
  display: flex;
  gap: var(--spacing-xs);
}

.toolbar-sep {
  width: 1px;
  height: 16px;
  background: var(--color-border-2);
  flex-shrink: 0;
}

.toolbar-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  white-space: nowrap;
}

.toolbar-input {
  width: 100px;
}

.toolbar-num {
  width: 70px;
}

.memory-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-md);
}

.memory-empty {
  text-align: center;
  color: var(--color-text-4);
  padding: var(--spacing-xl);
}

.result-header {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  font-weight: 600;
  margin-bottom: var(--spacing-sm);
}

.key-table-wrap {
  margin-bottom: var(--spacing-lg);
}

.mono {
  font-family: var(--font-family-mono);
}

.cell-ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-state {
  padding: var(--spacing-xl);
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
}

.type-tag {
  display: inline-block;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 500;
}

.type-string { background: var(--color-type-string-bg); color: var(--color-type-string); }
.type-hash { background: var(--color-type-hash-bg); color: var(--color-type-hash); }
.type-list { background: var(--color-type-list-bg); color: var(--color-type-list); }
.type-set { background: var(--color-type-set-bg); color: var(--color-type-set); }
.type-zset { background: var(--color-type-zset-bg); color: var(--color-type-zset); }
.type-stream { background: var(--color-type-stream-bg); color: var(--color-type-stream); }

.dist-section {
  margin-bottom: var(--spacing-lg);
}

.dist-title {
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  font-weight: 600;
  margin: 0 0 var(--spacing-sm) 0;
}

.dist-bars {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.bar-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.bar-label {
  width: 80px;
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  text-align: right;
  flex-shrink: 0;
  font-family: var(--font-family-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bar-track {
  flex: 1;
  height: 16px;
  background: var(--color-bg-3);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: var(--radius-sm);
  min-width: 2px;
  transition: width 0.3s ease;
}

.bar-prefix {
  background: var(--color-success);
}

.bar-value {
  width: 140px;
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  font-family: var(--font-family-mono);
  flex-shrink: 0;
}
</style>
