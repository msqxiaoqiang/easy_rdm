<template>
  <Teleport to="body">
    <div class="ctx-menu-overlay" @click="$emit('close')" @contextmenu.prevent="$emit('close')">
      <div class="ctx-menu" :style="{ left: x + 'px', top: y + 'px' }" ref="menuRef">
        <button
          v-for="item in items"
          :key="item.key"
          :class="['ctx-menu-item', { danger: item.danger, disabled: item.disabled }]"
          :disabled="item.disabled"
          @click="handleClick(item)"
        >
          <!-- SECURITY: v-html 安全 — icon 内容全部来自硬编码 SVG，无用户输入 -->
          <span class="ctx-icon" v-if="item.icon" v-html="item.icon"></span>
          <span class="ctx-label">{{ item.label }}</span>
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'

export interface MenuItem {
  key: string
  label: string
  icon?: string
  danger?: boolean
  disabled?: boolean
}

const props = defineProps<{
  x: number
  y: number
  items: MenuItem[]
}>()

const emit = defineEmits<{
  close: []
  select: [key: string]
}>()

const menuRef = ref<HTMLElement>()

function handleClick(item: MenuItem) {
  if (item.disabled) return
  emit('select', item.key)
  emit('close')
}

onMounted(async () => {
  await nextTick()
  // 边界检测，防止菜单超出视口
  if (menuRef.value) {
    const rect = menuRef.value.getBoundingClientRect()
    if (rect.right > window.innerWidth) {
      menuRef.value.style.left = (props.x - rect.width) + 'px'
    }
    if (rect.bottom > window.innerHeight) {
      menuRef.value.style.top = (props.y - rect.height) + 'px'
    }
  }
})
</script>

<style scoped>
.ctx-menu-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
}

.ctx-menu {
  position: fixed;
  min-width: 160px;
  background: var(--color-bg-popup);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  padding: var(--spacing-xs) 0;
  z-index: 2001;
}

.ctx-menu-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  width: 100%;
  padding: var(--spacing-xs) var(--spacing-md);
  background: none;
  border: none;
  color: var(--color-text-1);
  font-size: var(--font-size-sm);
  cursor: pointer;
  text-align: left;
  transition: background var(--transition-fast);
}

.ctx-menu-item:hover:not(:disabled) {
  background: var(--color-fill-2);
}

.ctx-menu-item.danger {
  color: var(--color-error);
}

.ctx-menu-item.danger:hover:not(:disabled) {
  background: var(--color-error-bg);
}

.ctx-menu-item.disabled {
  color: var(--color-text-disabled);
  cursor: not-allowed;
}

.ctx-icon {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.ctx-icon :deep(svg) {
  width: 14px;
  height: 14px;
}
</style>
