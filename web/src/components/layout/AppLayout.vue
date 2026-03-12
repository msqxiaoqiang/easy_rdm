<template>
  <div class="app-layout">
    <template v-if="ready">
      <!-- Activity Bar -->
      <ActivityBar
        v-model:activePanel="activePanel"
        :panel-collapsed="panelCollapsed"
        @togglePanel="panelCollapsed = !panelCollapsed"
        @openSettings="showSettings = true"
      />
      <!-- Side Panel (仅连接管理模式) -->
      <SidePanel
        v-show="activePanel === 'connections'"
        :collapsed="panelCollapsed"
        :title="panelTitle"
        @collapse="panelCollapsed = true"
        @expand="panelCollapsed = false"
      >
        <Sidebar ref="sidebarRef" />
      </SidePanel>
      <!-- Main Content -->
      <div class="app-main">
        <div v-show="activePanel === 'log'" class="app-log-wrap">
          <OpLogView :active="activePanel === 'log'" />
        </div>
        <div v-show="activePanel !== 'log'" class="app-conn-wrap">
          <TopTabBar class="app-topbar" />
          <div class="app-content">
            <transition name="fade" mode="out-in">
              <WelcomePage
                v-if="!connectionStore.activeTabId"
                @newConnection="sidebarRef?.openNewConnection()"
              />
              <ContentArea v-else :conn-id="connectionStore.activeTabId" />
            </transition>
          </div>
        </div>
      </div>

      <!-- 设置弹窗（直接从 Activity Bar 打开） -->
      <SettingsModal v-if="showSettings" :visible="showSettings" @close="showSettings = false" @saved="showSettings = false" />

      <!-- 全局弹框（confirm / prompt） -->
      <GlobalDialogs />
    </template>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConnectionStore } from '../../stores/connection'
import ActivityBar from './ActivityBar.vue'
import SidePanel from './SidePanel.vue'
import Sidebar from './Sidebar.vue'
import TopTabBar from './TopTabBar.vue'
import ContentArea from './ContentArea.vue'
import WelcomePage from './WelcomePage.vue'
import SettingsModal from '../settings/SettingsModal.vue'
import GlobalDialogs from '../common/GlobalDialogs.vue'
import OpLogView from '../views/OpLogView.vue'
import { useHeartbeat } from '../../composables/useHeartbeat'
import { matchesAction } from '../../composables/useShortcuts'
import { useSettingsStore } from '../../stores/settings'

const props = defineProps<{ skipInit?: boolean }>()

const { t } = useI18n()
const connectionStore = useConnectionStore()
const settingsStore = useSettingsStore()
const ready = ref(false)
const sidebarRef = ref<InstanceType<typeof Sidebar> | null>(null)
const showSettings = ref(false)

// Activity Bar + Side Panel 状态
const activePanel = ref<string>('connections')
const panelCollapsed = ref(false)
const panelTitle = computed(() => {
  const map: Record<string, string> = {
    connections: t('connection.title'),
    log: t('opLog.title'),
  }
  return map[activePanel.value] || ''
})

// 初始化心跳
useHeartbeat()

// 自动保存会话（tabs / activeTabId / connStates 变化时）
let saveTimer: ReturnType<typeof setTimeout> | null = null
watch(
  () => [connectionStore.tabs, connectionStore.activeTabId, connectionStore.connStates, connectionStore.dbPreferences, connectionStore.cliDbPreferences],
  () => {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => connectionStore.saveSession(), 1000)
  },
  { deep: true },
)

// 页面刷新/关闭前立即保存，防止防抖未执行导致丢失
function handleBeforeUnload() {
  if (saveTimer) clearTimeout(saveTimer)
  connectionStore.saveSession()
}

// ========== 全局快捷键 ==========
function handleGlobalKeydown(e: KeyboardEvent) {
  const target = e.target as HTMLElement
  const isInput = target.tagName === 'INPUT' || target.tagName === 'SELECT' || target.tagName === 'TEXTAREA'

  // 设置（任何时候都可触发）
  if (matchesAction(e, 'settings')) {
    e.preventDefault()
    showSettings.value = true
    return
  }

  // 关闭当前 Tab
  if (matchesAction(e, 'closeTab')) {
    e.preventDefault()
    const activeId = connectionStore.activeTabId
    if (!activeId) return
    const tab = connectionStore.tabs.find(t => t.id === activeId)
    if (tab?.pinned) return
    if (connectionStore.getConnState(activeId).status === 'connected') {
      connectionStore.disconnect(activeId).then(() => connectionStore.closeTab(activeId))
    } else {
      connectionStore.closeTab(activeId)
    }
    return
  }

  // Ctrl+Tab / Ctrl+Shift+Tab → 切换 Tab（保留硬编码，因为 Tab 键特殊）
  if (e.ctrlKey && e.key === 'Tab') {
    e.preventDefault()
    const tabs = connectionStore.tabs
    if (tabs.length < 2) return
    const idx = tabs.findIndex(t => t.id === connectionStore.activeTabId)
    const next = e.shiftKey
      ? (idx - 1 + tabs.length) % tabs.length
      : (idx + 1) % tabs.length
    connectionStore.setActiveTab(tabs[next].id)
    return
  }

  // Ctrl/Cmd+1~9 → 切换到第 N 个 Tab
  const mod = e.ctrlKey || e.metaKey
  if (mod && e.key >= '1' && e.key <= '9') {
    e.preventDefault()
    const idx = parseInt(e.key) - 1
    if (idx < connectionStore.tabs.length) {
      connectionStore.setActiveTab(connectionStore.tabs[idx].id)
    }
    return
  }

  // 以下快捷键在输入框中不触发
  if (isInput) return

  // 新建 Key
  if (matchesAction(e, 'newKey')) {
    e.preventDefault()
    document.dispatchEvent(new CustomEvent('shortcut:newKey'))
    return
  }

  // 刷新 Key 列表
  if (matchesAction(e, 'refresh') || e.key === 'F5') {
    e.preventDefault()
    document.dispatchEvent(new CustomEvent('shortcut:refresh'))
    return
  }

  // 保存
  if (matchesAction(e, 'save')) {
    e.preventDefault()
    document.dispatchEvent(new CustomEvent('shortcut:save'))
    return
  }

  // 搜索
  if (matchesAction(e, 'search')) {
    e.preventDefault()
    document.dispatchEvent(new CustomEvent('shortcut:search'))
    return
  }
}

onMounted(async () => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  window.addEventListener('keydown', handleGlobalKeydown)
  if (props.skipInit) {
    // App.vue 已经完成数据初始化，直接就绪
    ready.value = true
  } else {
    // 独立使用时，自行加载数据
    await Promise.all([
      connectionStore.loadConnections(),
      settingsStore.load(),
    ])
    await connectionStore.restoreSession()
    ready.value = true
  }
})

onUnmounted(() => {
  if (saveTimer) clearTimeout(saveTimer)
  window.removeEventListener('beforeunload', handleBeforeUnload)
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--color-bg-1);
  color: var(--color-text-1);
  font-family: var(--font-family);
  font-size: var(--font-size-md);
}

.app-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.app-log-wrap,
.app-conn-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.app-topbar {
  height: var(--topbar-height);
  min-height: var(--topbar-height);
  border-bottom: 1px solid var(--color-border-2);
  background: var(--color-bg-2);
}

.app-content {
  flex: 1;
  overflow: hidden;
}

/* fade 过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
