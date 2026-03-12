<template>
  <div class="monitor-view">
    <!-- 控制栏 -->
    <div class="monitor-toolbar">
      <a-button v-if="!running" type="primary" size="small" @click="startMonitor">
        {{ $t('monitor.start') }}
      </a-button>
      <a-button v-else type="primary" status="danger" size="small" @click="stopMonitor">
        {{ $t('monitor.stop') }}
      </a-button>
      <div class="toolbar-sep"></div>
      <a-input
        v-model="filter"
        size="small"
        :placeholder="$t('monitor.filterPlaceholder')"
        style="flex:1;min-width:100px;max-width:300px"
      />
      <div class="toolbar-sep"></div>
      <a-switch v-model="autoScroll" size="small" />
      <span class="toggle-label">{{ $t('monitor.autoScroll') }}</span>
      <span class="cmd-count">{{ filteredCommands.length }} {{ $t('monitor.commands') }}</span>
      <a-button size="small" :disabled="commands.length === 0" @click="clearCommands">{{ $t('pubsub.clear') }}</a-button>
    </div>

    <!-- 命令流 -->
    <div class="table-wrap" ref="cmdContainer">
      <a-table
        :data="filteredCommands"
        :pagination="false"
        :bordered="false"
        size="small"
        :sticky-header="true"
        row-key="_idx"
      >
        <template #columns>
          <a-table-column :title="$t('monitor.time')" data-index="timestamp" :width="230">
            <template #cell="{ record }"><span class="mono" style="white-space:nowrap">{{ formatTimestamp(record.timestamp) }}</span></template>
          </a-table-column>
          <a-table-column title="DB" data-index="db" :width="55">
            <template #cell="{ record }"><span class="mono cmd-db-cell">{{ record.db }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('monitor.client')" data-index="addr" :width="170">
            <template #cell="{ record }"><span class="mono" style="white-space:nowrap">{{ record.addr }}</span></template>
          </a-table-column>
          <a-table-column :title="$t('monitor.command')" data-index="command">
            <template #cell="{ record }"><span class="mono cell-ellipsis" v-ellipsis-tip>{{ record.command }}</span></template>
          </a-table-column>
        </template>
        <template #empty>
          <div class="empty-state">{{ running ? $t('monitor.waiting') : $t('monitor.hint') }}</div>
        </template>
      </a-table>
    </div>
  </div>
</template>

<script lang="ts">
import { Poller } from '../../utils/poller'
// 模块级：活跃的 Poller 实例，组件卸载后继续运行
const activeMonitorPollers = new Map<string, Poller>()
</script>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { request } from '../../utils/request'
import { type PollEvent } from '../../utils/poller'
import { useI18n } from 'vue-i18n'
import { useConnectionStore } from '../../stores/connection'
import { gmConfirm } from '../../utils/dialog'
import { showMessage } from '@/utils/platform'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()

interface MonitorCommand {
  timestamp: string
  db: string
  addr: string
  command: string
  raw: string
}

const running = ref(false)
const commands = ref<MonitorCommand[]>([])
const filter = ref('')
const autoScroll = ref(true)
const cmdContainer = ref<HTMLElement>()
const MAX_COMMANDS = 5000

const filteredCommands = computed(() => {
  if (!filter.value.trim()) return commands.value
  const f = filter.value.toLowerCase()
  return commands.value.filter(c =>
    c.command.toLowerCase().includes(f) ||
    c.addr.toLowerCase().includes(f)
  )
})

function handleMonitorData(events: PollEvent[]) {
  for (const evt of events) {
    const d = evt.data
    commands.value.push({
      timestamp: d.timestamp || '',
      db: d.db || '0',
      addr: d.addr || '',
      command: d.command || d.raw || '',
      raw: d.raw || '',
    })
  }
  if (commands.value.length > MAX_COMMANDS) {
    commands.value = commands.value.slice(-3000)
  }
  if (autoScroll.value) {
    nextTick(() => {
      const scrollBody = cmdContainer.value?.querySelector('.arco-table-body')
      if (scrollBody) {
        scrollBody.scrollTop = scrollBody.scrollHeight
      }
    })
  }
}

function makeBackgroundCallback(connId: string) {
  return (events: PollEvent[]) => {
    const cache = connectionStore.getMonitorCache(connId)
    if (!cache) return
    for (const evt of events) {
      const d = evt.data
      cache.commands.push({
        timestamp: d.timestamp || '',
        db: d.db || '0',
        addr: d.addr || '',
        command: d.command || d.raw || '',
        raw: d.raw || '',
      })
    }
    if (cache.commands.length > MAX_COMMANDS) {
      cache.commands = cache.commands.slice(-3000)
    }
  }
}

async function startMonitor() {
  if (!await gmConfirm(t('monitor.warning'))) return
  try {
    await request('monitor_start', {
      params: { conn_id: props.connId },
    })
    running.value = true
    commands.value = []

    activeMonitorPollers.get(props.connId)?.stop()

    const poller = new Poller({
      connId: props.connId,
      scene: 'monitor',
      onData: handleMonitorData,
    })
    activeMonitorPollers.set(props.connId, poller)
    poller.start()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function stopMonitor() {
  const poller = activeMonitorPollers.get(props.connId)
  poller?.stop()
  activeMonitorPollers.delete(props.connId)
  try {
    await request('monitor_stop', { params: { conn_id: props.connId } })
  } catch (_e) { /* ignore */ }
  running.value = false
}

function formatTimestamp(ts: string): string {
  if (!ts) return '-'
  const parts = ts.split('.')
  const sec = parseInt(parts[0], 10)
  const micro = parts[1] || '000000'
  if (isNaN(sec)) return ts
  const d = new Date(sec * 1000)
  const y = d.getFullYear()
  const M = String(d.getMonth() + 1).padStart(2, '0')
  const D = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  return `${y}-${M}-${D} ${hh}:${mm}:${ss}.${micro.padEnd(6, '0')}`
}

function clearCommands() {
  commands.value = []
}

function saveToCache() {
  connectionStore.saveMonitorCache(props.connId, {
    commands: commands.value,
    running: running.value,
    filter: filter.value,
  })
}

watch(() => props.connId, (_newId, oldId) => {
  if (oldId) {
    const oldPoller = activeMonitorPollers.get(oldId)
    if (oldPoller?.running) {
      connectionStore.saveMonitorCache(oldId, {
        commands: commands.value,
        running: running.value,
        filter: filter.value,
      })
      oldPoller.setCallbacks(makeBackgroundCallback(oldId))
    }
  }
  const cache = connectionStore.getMonitorCache(props.connId)
  if (cache) {
    commands.value = [...cache.commands]
    running.value = cache.running
    filter.value = cache.filter
  } else {
    commands.value = []
    running.value = false
    filter.value = ''
  }
  const newPoller = activeMonitorPollers.get(props.connId)
  if (newPoller?.running) {
    newPoller.setCallbacks(handleMonitorData)
  }
})

watch(() => connectionStore.getConnState(props.connId).status, (status) => {
  if (status === 'disconnected') {
    const poller = activeMonitorPollers.get(props.connId)
    if (poller) {
      poller.stop()
      activeMonitorPollers.delete(props.connId)
    }
    running.value = false
    commands.value = []
  }
})

onMounted(() => {
  const cache = connectionStore.getMonitorCache(props.connId)
  if (cache) {
    commands.value = [...cache.commands]
    running.value = cache.running
    filter.value = cache.filter
  }
  const poller = activeMonitorPollers.get(props.connId)
  if (poller?.running) {
    poller.setCallbacks(handleMonitorData)
  }
})

onBeforeUnmount(() => {
  saveToCache()
  const poller = activeMonitorPollers.get(props.connId)
  if (poller?.running) {
    poller.setCallbacks(makeBackgroundCallback(props.connId))
  }
})
</script>

<style scoped>
.monitor-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.monitor-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  flex-shrink: 0;
}

.toolbar-sep {
  width: 1px;
  height: 16px;
  background: var(--color-border-2);
  flex-shrink: 0;
}

.toggle-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  white-space: nowrap;
}

.cmd-count {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  white-space: nowrap;
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

.cmd-db-cell {
  color: var(--color-warning);
}

.empty-state {
  padding: var(--spacing-xl);
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
}
</style>
