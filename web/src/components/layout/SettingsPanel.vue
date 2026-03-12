<template>
  <div class="settings-panel">
    <div class="settings-placeholder">
      <icon-settings :size="32" style="color: var(--color-text-4)" />
      <span class="settings-hint">{{ $t('settings.title') }}</span>
      <a-button type="primary" size="small" @click="visible = true">
        {{ $t('settings.title') }}
      </a-button>
    </div>
    <SettingsModal v-if="visible" @close="visible = false" @saved="visible = false" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { IconSettings } from '@arco-design/web-vue/es/icon'
import SettingsModal from '../settings/SettingsModal.vue'

const visible = ref(false)
</script>

<style scoped>
.settings-panel {
  height: 100%;
  overflow-y: auto;
}

.settings-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-md);
  height: 100%;
  padding: var(--spacing-xl);
}

.settings-hint {
  font-size: var(--font-size-sm);
  color: var(--color-text-3);
}
</style>
