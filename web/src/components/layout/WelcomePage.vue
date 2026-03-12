<template>
  <div class="welcome-page">
    <div class="welcome-content">
      <!-- Redis logo illustration -->
      <div class="welcome-icon">
        <svg viewBox="0 0 128 128" width="72" height="72">
          <defs>
            <linearGradient id="redis-grad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" style="stop-color:var(--color-primary);stop-opacity:0.9" />
              <stop offset="100%" style="stop-color:var(--color-primary-dark);stop-opacity:1" />
            </linearGradient>
          </defs>
          <!-- Stylized database/Redis stack icon -->
          <ellipse cx="64" cy="28" rx="40" ry="14" fill="url(#redis-grad)" opacity="0.85"/>
          <path d="M24 28v18c0 7.7 17.9 14 40 14s40-6.3 40-14V28" fill="none" stroke="url(#redis-grad)" stroke-width="3" opacity="0.7"/>
          <ellipse cx="64" cy="46" rx="40" ry="14" fill="url(#redis-grad)" opacity="0.15"/>
          <path d="M24 46v18c0 7.7 17.9 14 40 14s40-6.3 40-14V46" fill="none" stroke="url(#redis-grad)" stroke-width="3" opacity="0.55"/>
          <ellipse cx="64" cy="64" rx="40" ry="14" fill="url(#redis-grad)" opacity="0.1"/>
          <path d="M24 64v18c0 7.7 17.9 14 40 14s40-6.3 40-14V64" fill="none" stroke="url(#redis-grad)" stroke-width="3" opacity="0.4"/>
          <ellipse cx="64" cy="82" rx="40" ry="14" fill="url(#redis-grad)" opacity="0.08"/>
          <!-- Key symbol -->
          <circle cx="64" cy="28" r="4" fill="#fff" opacity="0.9"/>
          <rect x="62" y="32" width="4" height="10" rx="2" fill="#fff" opacity="0.6"/>
        </svg>
      </div>

      <h2 class="welcome-title">Easy RDM</h2>
      <p class="welcome-desc">{{ $t('app.welcomeDesc') }}</p>

      <div class="welcome-actions">
        <a-button type="primary" size="large" @click="$emit('newConnection')">
          <template #icon><IconPlus /></template>
          {{ $t('connection.new') }}
        </a-button>
      </div>

      <p class="welcome-tip">{{ $t('app.welcomeTip') }}</p>

      <!-- Keyboard shortcut hints -->
      <div class="shortcut-hints">
        <div class="shortcut-item">
          <kbd>{{ isMac ? '⌘' : 'Ctrl' }}</kbd><kbd>N</kbd>
          <span>{{ $t('shortcut.newKey') }}</span>
        </div>
        <div class="shortcut-item">
          <kbd>{{ isMac ? '⌘' : 'Ctrl' }}</kbd><kbd>R</kbd>
          <span>{{ $t('shortcut.refresh') }}</span>
        </div>
        <div class="shortcut-item">
          <kbd>{{ isMac ? '⌘' : 'Ctrl' }}</kbd><kbd>F</kbd>
          <span>{{ $t('shortcut.search') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { IconPlus } from '@arco-design/web-vue/es/icon'

defineEmits<{ newConnection: [] }>()

const isMac = navigator.platform.toUpperCase().includes('MAC')
</script>

<style scoped>
.welcome-page {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  background: var(--color-bg-1);
}

.welcome-content {
  text-align: center;
  max-width: 420px;
  padding: var(--spacing-xl);
}

.welcome-icon {
  margin-bottom: var(--spacing-lg);
  opacity: 0.9;
}

.welcome-title {
  font-size: 26px;
  font-weight: 700;
  color: var(--color-text-1);
  margin: 0 0 var(--spacing-sm);
  letter-spacing: -0.5px;
}

.welcome-desc {
  color: var(--color-text-3);
  font-size: var(--font-size-md);
  margin: 0 0 var(--spacing-xl);
  line-height: 1.5;
}

.welcome-actions {
  margin-bottom: var(--spacing-lg);
}

.welcome-tip {
  color: var(--color-text-4);
  font-size: var(--font-size-xs);
  margin: 0 0 var(--spacing-xl);
}

.shortcut-hints {
  display: flex;
  justify-content: center;
  gap: var(--spacing-lg);
  flex-wrap: wrap;
}

.shortcut-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-size-xs);
  color: var(--color-text-4);
}

.shortcut-item span {
  margin-left: 4px;
}

kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  background: var(--color-fill-2);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  line-height: 1;
}
</style>
