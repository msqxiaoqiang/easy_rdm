<template>
  <div
    :class="['side-panel', { collapsed }]"
    :style="{ width: collapsed ? '0px' : panelWidth + 'px' }"
  >
    <div v-show="!collapsed" class="panel-inner">
      <div class="panel-header">
        <span class="panel-title">{{ title }}</span>
        <a-tooltip :content="$t('common.collapse')" position="right" mini>
          <button class="panel-collapse-btn" @click="$emit('collapse')">
            <icon-left :size="14" />
          </button>
        </a-tooltip>
      </div>
      <div class="panel-body">
        <slot />
      </div>
    </div>
    <!-- 收起时的展开按钮 -->
    <div v-if="collapsed" class="panel-expand-btn" @click="$emit('expand')">
      <icon-right :size="12" />
    </div>
    <div
      v-show="!collapsed"
      class="panel-resize-handle"
      @mousedown="startResize"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { IconLeft, IconRight } from '@arco-design/web-vue/es/icon'

const props = defineProps<{
  collapsed: boolean
  title: string
  width?: number
}>()

defineEmits<{ collapse: []; expand: [] }>()

const panelWidth = ref(props.width ?? 280)

function startResize(e: MouseEvent) {
  e.preventDefault()
  const startX = e.clientX
  const startW = panelWidth.value
  function onMove(ev: MouseEvent) {
    panelWidth.value = Math.max(220, Math.min(startW + (ev.clientX - startX), 400))
  }
  function onUp() {
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
</script>

<style scoped>
.side-panel {
  height: 100vh;
  display: flex;
  flex-shrink: 0;
  overflow: hidden;
  transition: width 0.25s ease-out;
  position: relative;
}

.side-panel.collapsed {
  overflow: visible;
}

.panel-inner {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border-right: 1px solid var(--color-border-1);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  flex-shrink: 0;
}

.panel-title {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.panel-collapse-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--color-text-3);
  cursor: pointer;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.panel-collapse-btn:hover {
  background: var(--color-fill-2);
  color: var(--color-text-1);
}

.panel-body {
  flex: 1;
  overflow: hidden;
}

.panel-resize-handle {
  width: 4px;
  flex-shrink: 0;
  cursor: col-resize;
  background: transparent;
  transition: background var(--transition-fast);
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  z-index: 10;
}

.panel-resize-handle:hover {
  background: var(--color-primary);
}

.panel-expand-btn {
  position: absolute;
  top: var(--spacing-sm);
  left: 0;
  width: 16px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--color-text-4);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-1);
  border-left: none;
  transition: all var(--transition-fast);
  z-index: 10;
}

.panel-expand-btn:hover {
  color: var(--color-primary);
  background: var(--color-fill-1);
}

</style>
