<template>
  <div class="activity-bar">
    <div class="activity-top">
      <a-tooltip
        v-for="item in topItems"
        :key="item.key"
        :content="$t(item.label)"
        position="right"
        mini
      >
        <div
          :class="['activity-item', { active: activePanel === item.key }]"
          @click="handlePanelClick(item.key)"
        >
          <component :is="item.icon" :size="20" />
        </div>
      </a-tooltip>
    </div>
    <div class="activity-bottom">
      <a-tooltip :content="$t('settings.title')" position="right" mini>
        <div
          class="activity-item"
          @click="$emit('openSettings')"
        >
          <icon-settings :size="20" />
        </div>
      </a-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { IconComputer, IconHistory, IconSettings } from '@arco-design/web-vue/es/icon'
import { markRaw, type Component } from 'vue'

const props = defineProps<{ activePanel: string; panelCollapsed: boolean }>()

const emit = defineEmits<{
  'update:activePanel': [panel: string]
  togglePanel: []
  openSettings: []
}>()

const topItems: { key: string; label: string; icon: Component }[] = [
  { key: 'connections', label: 'connection.title', icon: markRaw(IconComputer) },
  { key: 'log', label: 'opLog.title', icon: markRaw(IconHistory) },
]

function handlePanelClick(key: string) {
  if (key === 'connections') {
    if (props.activePanel === key) {
      // 同一图标再次点击 → 切换折叠
      emit('togglePanel')
    } else {
      // 从其他面板切回 → 仅切换面板，保留折叠状态
      emit('update:activePanel', key)
    }
  } else {
    // 非连接管理面板（如日志）：直接切换，无侧面板折叠逻辑
    if (props.activePanel === key) {
      // 再次点击同一个 → 回到连接管理
      emit('update:activePanel', 'connections')
    } else {
      emit('update:activePanel', key)
    }
  }
}
</script>

<style scoped>
.activity-bar {
  width: 48px;
  min-width: 48px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-3);
  border-right: 1px solid var(--color-border-2);
  flex-shrink: 0;
}

.activity-top {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: var(--spacing-sm);
  gap: 2px;
}

.activity-bottom {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-bottom: var(--spacing-sm);
}

.activity-item {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--color-text-3);
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
}

.activity-item:hover {
  color: var(--color-text-1);
  background: var(--color-fill-2);
  transform: scale(1.08);
}

.activity-item.active {
  color: var(--color-primary);
  background: var(--color-primary-bg);
}

.activity-item.active::before {
  content: '';
  position: absolute;
  left: -4px;
  top: 8px;
  bottom: 8px;
  width: 2px;
  background: var(--color-primary);
  border-radius: 1px;
}
</style>
