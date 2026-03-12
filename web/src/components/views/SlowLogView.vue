<template>
  <div class="slowlog-view">
    <div class="panel-toolbar">
      <span class="toolbar-info" v-if="slowlogThreshold">
        {{ $t('server.slowlogThreshold') }}: {{ (parseInt(slowlogThreshold) / 1000).toFixed(1) }} ms
      </span>
      <span class="toolbar-info" v-if="slowlogTotal">
        {{ $t('collection.total') }}: {{ slowlogTotal }}
      </span>
      <div style="flex:1"></div>
      <button class="panel-btn" @click="loadSlowlog">{{ $t('common.refresh') }}</button>
      <button class="panel-btn danger" @click="resetSlowlog">{{ $t('server.slowlogReset') }}</button>
    </div>
    <div class="table-wrap">
      <a-table
        :data="slowlogEntries"
        :pagination="false"
        :bordered="false"
        size="small"
        :sticky-header="true"
        row-key="id"
      >
        <template #columns>
          <a-table-column :title="$t('server.slowlogTime')" data-index="timestamp" :width="200" :sortable="timeSortable">
            <template #cell="{ record }"><span class="mono" style="white-space:nowrap">{{ formatTimestamp(record.timestamp) }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('server.slowlogClient')" data-index="client_addr" :width="180">
            <template #cell="{ record }"><span class="mono" style="white-space:nowrap">{{ record.client_addr || '-' }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('server.slowlogCommand')" data-index="command">
            <template #cell="{ record }"><span class="mono cell-ellipsis" v-ellipsis-tip>{{ record.command }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('server.slowlogDuration')" data-index="duration" :width="120">
            <template #cell="{ record }"><span class="mono" style="white-space:nowrap">{{ (record.duration / 1000).toFixed(2) }} ms</span></template>
          </a-table-column>
        </template>
        <template #empty>
          <div class="empty-state">{{ $t('common.noData') }}</div>
        </template>
      </a-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { gmConfirm } from '../../utils/dialog'
import { showMessage } from '@/utils/platform'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()

interface SlowlogEntry {
  id: number
  timestamp: number
  duration: number
  command: string
  client_addr: string
  client_name: string
}

let slowlogGen = 0
const slowlogEntries = ref<SlowlogEntry[]>([])
const slowlogTotal = ref(0)
const slowlogThreshold = ref('')

const timeSortable = {
  sortDirections: ['ascend', 'descend'] as ('ascend' | 'descend')[],
  sorter: (a: SlowlogEntry, b: SlowlogEntry, extra: { direction: string }) => {
    const diff = a.timestamp - b.timestamp
    return extra.direction === 'descend' ? -diff : diff
  },
}

async function loadSlowlog() {
  const gen = slowlogGen
  try {
    const res = await request<{ entries: SlowlogEntry[]; total: number; threshold: string }>('slowlog_get', {
      params: { conn_id: props.connId, count: 128 },
    })
    if (gen !== slowlogGen) return
    if (res.data) {
      slowlogEntries.value = res.data.entries || []
      slowlogTotal.value = res.data.total || 0
      slowlogThreshold.value = res.data.threshold || ''
    }
  } catch (e: any) {
    if (gen !== slowlogGen) return
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function resetSlowlog() {
  if (!await gmConfirm(t('server.slowlogReset') + '?')) return
  try {
    await request('slowlog_reset', { params: { conn_id: props.connId } })
    showMessage('success', t('common.success'))
    loadSlowlog()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function formatTimestamp(ts: number): string {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  const y = d.getFullYear()
  const M = String(d.getMonth() + 1).padStart(2, '0')
  const D = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  return `${y}-${M}-${D} ${hh}:${mm}:${ss}`
}

// 初次加载 & connId 变化时重新加载
watch(() => props.connId, () => {
  slowlogGen++
  loadSlowlog()
}, { immediate: true })
</script>

<style scoped>
.slowlog-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
}

.toolbar-info {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}

.panel-btn {
  height: 28px;
  padding: 0 var(--spacing-md);
  background: var(--color-fill-2);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.panel-btn:hover {
  background: var(--color-fill-3);
}

.panel-btn.danger {
  color: var(--color-error, #f53f3f);
}

.panel-btn.danger:hover {
  background: rgba(245, 63, 63, 0.08);
}

.table-wrap {
  flex: 1;
  overflow: auto;
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
</style>
