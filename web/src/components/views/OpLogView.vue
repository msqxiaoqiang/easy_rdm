<template>
  <div class="oplog-view">
    <!-- 工具栏 -->
    <div class="panel-toolbar">
      <a-select v-model="filterConnId" size="medium" style="width: 140px">
        <a-option value="">{{ $t('opLog.allConnections') }}</a-option>
        <a-option v-for="conn in connectionStore.connections" :key="conn.id" :value="conn.id">
          {{ conn.name }}
        </a-option>
      </a-select>
      <span class="toolbar-info">{{ filteredLogs.length }} {{ $t('opLog.entries') }}</span>
      <div style="flex:1"></div>
      <button class="panel-btn" :class="{ 'refresh-spinning': loading }" @click="() => loadLogs()">{{ $t('common.refresh') }}</button>
      <button class="panel-btn danger" @click="clearLogs" :disabled="logs.length === 0">{{ $t('opLog.clear') }}</button>
    </div>

    <!-- 日志表格 -->
    <div class="table-wrap">
      <a-table
        :data="filteredLogs"
        :pagination="false"
        :bordered="false"
        size="small"
        :sticky-header="true"
        row-key="seq"
      >
        <template #columns>
          <a-table-column :title="$t('opLog.time')" data-index="time" :width="180" :sortable="timeSortable">
            <template #cell="{ record }"><span class="mono" style="white-space:nowrap">{{ formatDateTime(record.time) }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('opLog.connection')" data-index="conn_id" :width="180">
            <template #cell="{ record }"><span class="cell-ellipsis" v-ellipsis-tip>{{ getConnName(record.conn_id) }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('opLog.action')" data-index="action" :width="70">
            <template #cell="{ record }"><span :class="['action-tag', actionClass(record.action)]">{{ record.action }}</span></template>
          </a-table-column>
          <a-table-column title="Key" data-index="key" :width="200">
            <template #cell="{ record }"><span class="mono cell-ellipsis" v-ellipsis-tip>{{ record.key }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('opLog.detail')" data-index="detail" :width="180">
            <template #cell="{ record }"><span class="cell-ellipsis" v-ellipsis-tip>{{ record.detail }}</span></template>
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
import { ref, computed, watch } from 'vue'
import { request } from '../../utils/request'
import { useConnectionStore } from '../../stores/connection'
import { useI18n } from 'vue-i18n'
import { showMessage } from '@/utils/platform'
import { gmConfirm } from '@/utils/dialog'

interface LogEntry {
  seq: number
  time: number
  conn_id: string
  action: string
  key: string
  detail: string
}

const props = defineProps<{ active?: boolean }>()

const { t } = useI18n()
const connectionStore = useConnectionStore()

const timeSortable = {
  sortDirections: ['ascend', 'descend'] as ('ascend' | 'descend')[],
  sorter: (a: LogEntry, b: LogEntry, extra: { direction: string }) => {
    const diff = a.time - b.time
    return extra.direction === 'descend' ? -diff : diff
  },
}

const logs = ref<LogEntry[]>([])
const filterConnId = ref('')
const loading = ref(false)
let lastSeq = 0 // 增量加载：记住最后一条的 seq

const filteredLogs = computed(() => {
  if (!filterConnId.value) return logs.value
  return logs.value.filter(l => l.conn_id === filterConnId.value)
})

function getConnName(connId: string): string {
  if (!connId) return '-'
  const conn = connectionStore.connections.find(c => c.id === connId)
  return conn?.name || connId.slice(0, 8)
}

async function loadLogs() {
  loading.value = true
  try {
    const params: any = { after: lastSeq, limit: 500 }
    const res = await request<LogEntry[]>('get_op_log', { params })
    const newEntries = res.data || []
    if (newEntries.length > 0) {
      // 新条目倒序插入到列表头部，最多保留 500 条
      const reversed = [...newEntries].reverse()
      const merged = [...reversed, ...logs.value]
      logs.value = merged.length > 500 ? merged.slice(0, 500) : merged
      lastSeq = newEntries[newEntries.length - 1].seq
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function clearLogs() {
  if (!await gmConfirm(t('opLog.clearConfirm'))) return
  try {
    await request('clear_op_log', { params: {} })
    logs.value = []
    lastSeq = 0
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function formatDateTime(ts: number): string {
  const d = new Date(ts)
  const y = d.getFullYear()
  const M = String(d.getMonth() + 1).padStart(2, '0')
  const D = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  return `${y}-${M}-${D} ${hh}:${mm}:${ss}`
}

function actionClass(action: string): string {
  if (action === 'DELETE' || action === 'HDEL' || action === 'SREM' || action === 'ZREM' || action === 'LREM') return 'delete'
  if (action === 'CREATE') return 'create'
  if (action === 'RENAME') return 'rename'
  return 'modify'
}

// 每次切换到操作日志面板时自动加载增量日志
watch(() => props.active, (val) => {
  if (val) loadLogs()
}, { immediate: true })
</script>

<style scoped>
.oplog-view {
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
  flex-shrink: 0;
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

.panel-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.refresh-spinning {
  animation: oplog-spin 0.8s linear infinite;
}

@keyframes oplog-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
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

/* 操作标签 */
.action-tag {
  display: inline-block;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  white-space: nowrap;
}

.action-tag.delete {
  background: var(--color-error-bg, rgba(245, 63, 63, 0.1));
  color: var(--color-error);
}

.action-tag.create {
  background: var(--color-success-bg, rgba(0, 180, 42, 0.1));
  color: var(--color-success);
}

.action-tag.rename {
  background: var(--color-purple-bg, rgba(114, 46, 209, 0.1));
  color: var(--color-purple, #722ed1);
}

.action-tag.modify {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}

.empty-state {
  padding: var(--spacing-xl);
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
}
</style>
