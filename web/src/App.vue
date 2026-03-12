<template>
  <AppLayout v-if="ready" :skip-init="true" />
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppLayout from './components/layout/AppLayout.vue'
import { isGmssh } from './utils/platform'
import { useSettingsStore } from './stores/settings'
import { useConnectionStore } from './stores/connection'

const ready = ref(false)

function hideLoading() {
  const el = document.getElementById('app-loading')
  if (!el) return
  el.classList.add('hidden')
  setTimeout(() => el.remove(), 250)
}

onMounted(async () => {
  try {
    // 1. GMSSH 模式：等待 SDK 初始化
    if (isGmssh()) {
      await (window as any).$gm.init()
    }

    // 2. 加载设置和连接数据
    const settingsStore = useSettingsStore()
    const connectionStore = useConnectionStore()
    await Promise.all([
      connectionStore.loadConnections(),
      settingsStore.load(),
    ])
    await connectionStore.restoreSession()
  } catch (e) {
    console.error('App init error:', e)
  } finally {
    // 即使初始化部分失败，也显示界面
    hideLoading()
    ready.value = true
  }
})
</script>
