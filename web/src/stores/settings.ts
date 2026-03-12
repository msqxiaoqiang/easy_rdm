import { reactive } from 'vue'
import { defineStore } from 'pinia'
import { request } from '../utils/request'
import { applyTheme } from '../utils/theme'
import { loadLanguage } from '../i18n'
import { loadShortcutBindings } from '../composables/useShortcuts'

export const defaults = {
  theme: 'dark',
  language: 'zh-CN',
  scanCount: 200,
  autoReconnect: true,
  retryInterval: 5,
  maxRetries: 10,
  heartbeatInterval: 30,
  editorFontSize: 14,
  showLineNumbers: true,
  cliFontSize: 14,
  defaultExportFormat: 'json',
  bufferLimit: 5000,
  scanBatchSize: 1000,
  dangerousCommandConfirm: true,
}

export type AppSettings = typeof defaults

function applyCssVars(s: AppSettings) {
  const root = document.documentElement.style
  root.setProperty('--app-cli-font-size', s.cliFontSize + 'px')
  root.setProperty('--app-editor-font-size', s.editorFontSize + 'px')
}

export const useSettingsStore = defineStore('settings', () => {
  const settings = reactive<AppSettings>({ ...defaults })

  async function load() {
    try {
      const res = await request<Record<string, any>>('get_settings', { params: {} })
      if (res.data) {
        const { shortcuts, ...rest } = res.data
        Object.assign(settings, { ...defaults, ...rest })
        if (shortcuts) {
          loadShortcutBindings(shortcuts)
        }
      }
    } catch (_e) {
      // 使用默认值
    }
    applySettings()
    await loadLanguage(settings.language)
  }

  function applySettings() {
    applyTheme(settings.theme)
    applyCssVars(settings)
  }

  function updateFromSave(data: Partial<AppSettings>) {
    Object.assign(settings, data)
    applySettings()
  }

  return { settings, load, applySettings, updateFromSave }
})
