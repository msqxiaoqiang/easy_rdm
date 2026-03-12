<template>
  <div class="content-area">
    <!-- 子 Tab 栏（Arco Tabs line 风格） -->
    <a-tabs
      v-model:active-key="activeSubTab"
      type="line"
      size="small"
      class="sub-tabs"
      :hide-content="true"
    >
      <a-tab-pane
        v-for="tab in visibleSubTabs"
        :key="tab.key"
        :title="$t(tab.label)"
      >
        <template #title>
          <span class="sub-tab-title">
            <component :is="tab.icon" :size="14" />
            <span>{{ $t(tab.label) }}</span>
          </span>
        </template>
      </a-tab-pane>
    </a-tabs>

    <!-- 内容区：Key 列表 + 详情 -->
    <div class="content-body">
      <!-- Key 列表面板（仅在 keyDetail tab 时显示） -->
      <KeyListPanel
        v-show="activeSubTab === 'keyDetail'"
        ref="keyListRef"
        class="key-list-pane"
        :style="{ width: keyListWidth + 'px' }"
        :conn-id="connId"
        @selectKey="handleSelectKey"
        @deleted="handleKeyDeleted"
        @refreshKey="handleRefreshKey"
      />
      <!-- 拖拽分隔条 -->
      <div
        v-show="activeSubTab === 'keyDetail'"
        class="resize-handle"
        @mousedown="startResize"
      ></div>
      <!-- 右侧详情 -->
      <div class="detail-pane">
        <component
          :is="currentComponent"
          :conn-id="connId"
          v-bind="currentProps"
          @deleted="handleKeyDeleted"
          @notFound="handleKeyNotFound"
          @renamed="handleKeyRenamed"
          @ttlChanged="handleTTLChanged"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount, markRaw, nextTick, type Component } from 'vue'
import {
  IconDashboard, IconStorage, IconCode,
  IconSend, IconEye, IconCommon, IconFile, IconHistory,
  IconSafe, IconApps,
} from '@arco-design/web-vue/es/icon'
import StatusView from '../views/StatusView.vue'
import KeyDetailView from '../views/KeyDetailView.vue'
import CliView from '../views/CliView.vue'
import PubSubView from '../views/PubSubView.vue'
import MonitorView from '../views/MonitorView.vue'

import MemoryView from '../views/MemoryView.vue'
import LuaScriptView from '../views/LuaScriptView.vue'
import SlowLogView from '../views/SlowLogView.vue'
import SentinelInfoView from '../views/SentinelInfoView.vue'
import ClusterInfoView from '../views/ClusterInfoView.vue'
import KeyListPanel from '../views/KeyListPanel.vue'
import { useConnectionStore } from '../../stores/connection'
import { versionGte } from '../../utils/version'
import { request } from '../../utils/request'

const props = defineProps<{ connId: string }>()
const connectionStore = useConnectionStore()

const redisVersion = computed(() => connectionStore.getConnState(props.connId).redisVersion)
const isSentinel = computed(() => connectionStore.connections.find(c => c.id === props.connId)?.use_sentinel ?? false)
const isCluster = computed(() => connectionStore.connections.find(c => c.id === props.connId)?.use_cluster ?? false)

interface SubTab {
  key: string
  label: string
  icon: Component
  component: Component
  visible?: () => boolean
}

const subTabDefs: SubTab[] = [
  { key: 'status', label: 'connection.status', icon: markRaw(IconDashboard), component: markRaw(StatusView) },
  { key: 'sentinel', label: 'sentinel.title', icon: markRaw(IconSafe), component: markRaw(SentinelInfoView), visible: () => isSentinel.value },
  { key: 'cluster', label: 'cluster.title', icon: markRaw(IconApps), component: markRaw(ClusterInfoView), visible: () => isCluster.value },
  { key: 'keyDetail', label: 'connection.keyDetail', icon: markRaw(IconStorage), component: markRaw(KeyDetailView) },
  { key: 'cli', label: 'connection.cli', icon: markRaw(IconCode), component: markRaw(CliView) },
  { key: 'pubsub', label: 'pubsub.title', icon: markRaw(IconSend), component: markRaw(PubSubView) },
  { key: 'monitor', label: 'monitor.title', icon: markRaw(IconEye), component: markRaw(MonitorView) },
  { key: 'memory', label: 'memory.title', icon: markRaw(IconCommon), component: markRaw(MemoryView), visible: () => versionGte(redisVersion.value, '4.0.0') },
  { key: 'lua', label: 'lua.title', icon: markRaw(IconFile), component: markRaw(LuaScriptView) },
  { key: 'slowlog', label: 'server.slowlog', icon: markRaw(IconHistory), component: markRaw(SlowLogView) },
]

const activeSubTab = ref('status')
const selectedKey = ref('')
const selectedKeyType = ref('')
const keyListRef = ref<InstanceType<typeof KeyListPanel> | null>(null)
const keyListWidth = ref(280)
let resizing = false

function startResize(e: MouseEvent) {
  e.preventDefault()
  resizing = true
  const startX = e.clientX
  const startW = keyListWidth.value
  function onMove(ev: MouseEvent) {
    if (!resizing) return
    const w = startW + (ev.clientX - startX)
    keyListWidth.value = Math.max(180, Math.min(w, 600))
  }
  function onUp() {
    resizing = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

onBeforeUnmount(() => { resizing = false })

const visibleSubTabs = computed(() =>
  subTabDefs.filter(t => !t.visible || t.visible())
)

const currentComponent = computed(() => {
  const tab = subTabDefs.find(t => t.key === activeSubTab.value)
  return tab?.component || StatusView
})

const currentProps = computed(() => {
  if (activeSubTab.value === 'keyDetail') {
    // DB 未就绪时不传 keyName，避免在错误 DB 上查询导致 "Key 不存在"
    return { keyName: dbReady.value ? selectedKey.value : '', keyType: selectedKeyType.value }
  }
  return {}
})

function handleSelectKey(key: string, type: string) {
  selectedKey.value = key
  selectedKeyType.value = type
  connectionStore.saveSelectedKeyCache(props.connId, key, type)
}

function handleKeyDeleted(key?: string) {
  selectedKey.value = ''
  selectedKeyType.value = ''
  connectionStore.clearSelectedKeyCache(props.connId)
  if (key) {
    keyListRef.value?.removeKey(key)
  }
}

function handleKeyNotFound() {
  selectedKey.value = ''
  selectedKeyType.value = ''
  connectionStore.clearSelectedKeyCache(props.connId)
}

function handleKeyRenamed(_oldKey: string, newKey: string) {
  selectedKey.value = newKey
  keyListRef.value?.refreshKeys()
}

async function handleRefreshKey(key: string) {
  // 强制触发 KeyDetailView watcher：先清空再设回
  const type = selectedKeyType.value
  selectedKey.value = ''
  await nextTick()
  selectedKey.value = key
  selectedKeyType.value = type
}

function handleTTLChanged(key: string, ttl: number) {
  keyListRef.value?.updateKeyTTL(key, ttl)
}

// DB 切换就绪标记：离开 CLI 时需要先恢复 DB，防止 KeyDetailView 在错误 DB 上查询
const dbReady = ref(true)

watch(activeSubTab, async (newTab, oldTab) => {
  const state = connectionStore.getConnState(props.connId)
  if (state.cliDb === state.currentDb) return

  if (oldTab === 'cli' && newTab !== 'cli') {
    // 离开 CLI → 恢复到 currentDb，必须等待完成后再让详情组件加载
    dbReady.value = false
    try {
      await request('select_db', { params: { conn_id: props.connId, db: state.currentDb } })
    } finally {
      dbReady.value = true
    }
  } else if (newTab === 'cli' && oldTab !== 'cli') {
    // 进入 CLI → 切换到 cliDb
    await request('select_db', { params: { conn_id: props.connId, db: state.cliDb } })
  }
})

// 子Tab变化时保存到 store
watch(activeSubTab, (newTab) => {
  if (props.connId) {
    connectionStore.setSubTabCache(props.connId, newTab)
  }
})

watch(() => props.connId, (newId, oldId) => {
  // 保存旧连接的选中 key
  if (oldId && selectedKey.value) {
    connectionStore.saveSelectedKeyCache(oldId, selectedKey.value, selectedKeyType.value)
  }
  // 恢复新连接的子Tab（默认 status）
  activeSubTab.value = connectionStore.getSubTabCache(newId) || 'status'
  // 恢复新连接的选中 key
  const cached = connectionStore.getSelectedKeyCache(newId)
  if (cached) {
    selectedKey.value = cached.key
    selectedKeyType.value = cached.type
  } else {
    selectedKey.value = ''
    selectedKeyType.value = ''
  }
})
</script>

<style scoped>
.content-area {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.sub-tabs {
  background: var(--color-bg-2);
  flex-shrink: 0;
  border-bottom: 1px solid var(--color-border-2);
}

.sub-tabs :deep(.arco-tabs-nav) {
  padding: 0 var(--spacing-md);
}

.sub-tabs :deep(.arco-tabs-nav::before) {
  display: none;
}

/* 确保 sub-tab 选中线和文字为红色主色调 */
.sub-tabs :deep(.arco-tabs-nav-ink) {
  background-color: var(--color-primary);
}

.sub-tabs :deep(.arco-tabs-tab-active) {
  color: var(--color-primary);
}

.sub-tabs :deep(.arco-tabs-tab:hover) {
  color: var(--color-primary);
}

.sub-tab-title {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.content-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.key-list-pane {
  min-width: 180px;
  max-width: 600px;
  flex-shrink: 0;
  border-right: 1px solid var(--color-border-1);
}

.resize-handle {
  width: 4px;
  flex-shrink: 0;
  cursor: col-resize;
  background: transparent;
  transition: background var(--transition-fast);
}

.resize-handle:hover {
  background: var(--color-primary);
}

.detail-pane {
  flex: 1;
  overflow: hidden;
}
</style>
