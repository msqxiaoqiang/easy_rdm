<template>
  <!-- Confirm Dialog -->
  <a-modal
    :visible="confirmState.visible"
    :width="400"
    :mask-closable="false"
    unmount-on-close
    class="confirm-modal"
    @cancel="resolveConfirm(false)"
  >
    <template #title>
      <span class="confirm-title">
        <icon-exclamation-circle-fill class="confirm-icon" />
        <span>{{ confirmState.title || $t('common.confirm') }}</span>
      </span>
    </template>
    <div class="confirm-content">{{ confirmState.content }}</div>
    <template #footer>
      <a-button @click="resolveConfirm(false)">{{ $t('common.cancel') }}</a-button>
      <a-button type="primary" @click="resolveConfirm(true)">{{ $t('common.confirm') }}</a-button>
    </template>
  </a-modal>

  <!-- Prompt Dialog -->
  <a-modal
    :visible="promptState.visible"
    :width="400"
    :mask-closable="false"
    unmount-on-close
    @cancel="resolvePrompt(null)"
  >
    <template #title>
      <span>{{ promptState.title }}</span>
    </template>
    <a-input
      v-model="promptState.value"
      :placeholder="promptState.placeholder"
      allow-clear
      @keydown.enter="resolvePrompt(promptState.value)"
    />
    <template #footer>
      <a-button @click="resolvePrompt(null)">{{ $t('common.cancel') }}</a-button>
      <a-button type="primary" @click="resolvePrompt(promptState.value)">{{ $t('common.confirm') }}</a-button>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { IconExclamationCircleFill } from '@arco-design/web-vue/es/icon'
import { confirmState, resolveConfirm, promptState, resolvePrompt } from '../../utils/dialog'
</script>

<style scoped>
:deep(.confirm-modal .arco-modal-header) {
  text-align: left;
}

.confirm-title {
  display: inline-flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.confirm-icon {
  color: var(--color-warning);
  font-size: 18px;
}

.confirm-content {
  font-size: var(--font-size-md);
  color: var(--color-text-1);
  line-height: 1.6;
}
</style>
