<template>
  <div class="topbar">
    <!-- Tab 列表 -->
    <div class="tab-list" ref="tabListRef">
      <div
        v-for="tab in connectionStore.tabs"
        :key="tab.id"
        :class="['tab-item', { active: connectionStore.activeTabId === tab.id, pinned: tab.pinned }]"
        @click="connectionStore.setActiveTab(tab.id)"
        @contextmenu.prevent="showTabMenu($event, tab)"
      >
        <!-- Redis 图标（颜色反映连接状态） -->
        <span :class="['tab-icon', connectionStore.getConnState(tab.id).status]">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <ellipse cx="12" cy="6" rx="8" ry="3" opacity="0.9"/>
            <path d="M4 6v4c0 1.7 3.6 3 8 3s8-1.3 8-3V6" opacity="0.6"/>
            <path d="M4 10v4c0 1.7 3.6 3 8 3s8-1.3 8-3V10" opacity="0.4"/>
            <path d="M4 14v4c0 1.7 3.6 3 8 3s8-1.3 8-3V14" opacity="0.25"/>
          </svg>
        </span>
        <span class="tab-name" v-ellipsis-tip>{{ tab.name }}</span>
        <button
          v-if="!tab.pinned"
          class="tab-close"
          @click.stop="handleCloseTab(tab.id)"
        >
          <icon-close :size="12" />
        </button>
        <span v-else class="tab-pin">
          <icon-pushpin :size="10" />
        </span>
      </div>
    </div>

    <!-- Tab 右键菜单 -->
    <ContextMenu
      v-if="ctxMenu"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :items="ctxMenu.items"
      @close="ctxMenu = null"
      @select="handleTabCtxAction"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useConnectionStore, type ConnectionTab } from '../../stores/connection'
import { useI18n } from 'vue-i18n'
import { IconClose, IconPushpin } from '@arco-design/web-vue/es/icon'
import ContextMenu, { type MenuItem } from '../common/ContextMenu.vue'

const { t } = useI18n()

const connectionStore = useConnectionStore()
const tabListRef = ref<HTMLElement>()
const ctxMenu = ref<{ x: number; y: number; tab: ConnectionTab; items: MenuItem[] } | null>(null)

async function handleCloseTab(id: string) {
  const state = connectionStore.getConnState(id)
  if (state.status === 'connected') {
    await connectionStore.disconnect(id)
  }
  connectionStore.closeTab(id)
}

function showTabMenu(event: MouseEvent, tab: ConnectionTab) {
  const items: MenuItem[] = [
    { key: 'pin', label: tab.pinned ? t('connection.unpin') : t('connection.pin') },
    { key: 'close', label: t('common.close'), disabled: tab.pinned },
    { key: 'closeOthers', label: t('connection.closeOthers') },
  ]
  ctxMenu.value = { x: event.clientX, y: event.clientY, tab, items }
}

async function handleTabCtxAction(key: string) {
  const tab = ctxMenu.value?.tab
  if (!tab) return
  ctxMenu.value = null

  switch (key) {
    case 'pin':
      connectionStore.pinTab(tab.id)
      break
    case 'close':
      await handleCloseTab(tab.id)
      break
    case 'closeOthers':
      for (const t of [...connectionStore.tabs]) {
        if (t.id !== tab.id && !t.pinned) {
          await handleCloseTab(t.id)
        }
      }
      break
  }
}
</script>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  height: 100%;
  padding: 0 var(--spacing-sm);
  gap: var(--spacing-xs);
}

.tab-list {
  display: flex;
  flex: 1;
  overflow-x: auto;
  gap: 0;
  scrollbar-width: none;
  align-items: stretch;
  height: 100%;
}

.tab-list::-webkit-scrollbar {
  display: none;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: 0 var(--spacing-sm);
  height: 100%;
  background: transparent;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
  max-width: 160px;
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  transition: all var(--transition-fast);
  border-bottom: 2px solid transparent;
  box-sizing: border-box;
}

.tab-item:hover {
  color: var(--color-text-1);
  background: var(--color-fill-1);
}

.tab-item.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
  background: transparent;
}

.tab-icon {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  color: var(--color-text-4);
  opacity: 0.7;
}

.tab-icon.connected {
  color: var(--color-primary);
  opacity: 1;
}

.tab-icon.connecting {
  color: var(--color-warning);
  opacity: 1;
}

.tab-icon.error {
  color: var(--color-error);
  opacity: 1;
}

.tab-item.active .tab-icon {
  opacity: 1;
}

.tab-name {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tab-close {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--color-text-4);
  cursor: pointer;
  flex-shrink: 0;
  transition: all var(--transition-fast);
}

.tab-close:hover {
  background: var(--color-error-bg);
  color: var(--color-error);
}

.tab-pin {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  color: var(--color-text-4);
}
</style>
