import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/utils/request', () => ({
  request: vi.fn(),
}))
vi.mock('@/utils/theme', () => ({
  applyTheme: vi.fn(),
}))
vi.mock('@/i18n', () => ({
  default: {},
  loadLanguage: vi.fn(),
}))
vi.mock('@/composables/useShortcuts', () => ({
  loadShortcutBindings: vi.fn(),
}))

import { request } from '@/utils/request'
import { applyTheme } from '@/utils/theme'
import { useSettingsStore } from '@/stores/settings'

const mockRequest = vi.mocked(request)
const mockApplyTheme = vi.mocked(applyTheme)

describe('settingsStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setActivePinia(createPinia())
  })

  it('load 从后端加载设置并应用主题和语言', async () => {
    mockRequest.mockResolvedValueOnce({
      code: 200,
      data: { theme: 'light', language: 'en' },
      msg: 'OK',
    })

    const store = useSettingsStore()
    await store.load()

    expect(mockRequest).toHaveBeenCalledWith('get_settings', { params: {} })
    expect(store.settings.theme).toBe('light')
    expect(mockApplyTheme).toHaveBeenCalledWith('light')
  })

  it('load 失败时使用默认值', async () => {
    mockRequest.mockRejectedValueOnce(new Error('fail'))

    const store = useSettingsStore()
    await store.load()

    expect(store.settings.theme).toBe('dark')
    expect(mockApplyTheme).toHaveBeenCalledWith('dark')
  })

  it('applySettings 注入 CSS 变量', () => {
    const store = useSettingsStore()
    store.settings.cliFontSize = 15
    store.settings.editorFontSize = 13

    store.applySettings()

    const root = document.documentElement.style
    expect(root.getPropertyValue('--app-cli-font-size')).toBe('15px')
    expect(root.getPropertyValue('--app-editor-font-size')).toBe('13px')
  })

  it('updateFromSave 更新 store 并应用设置', () => {
    const store = useSettingsStore()
    store.updateFromSave({ theme: 'light', cliFontSize: 15, editorFontSize: 13 })

    expect(store.settings.theme).toBe('light')
    expect(mockApplyTheme).toHaveBeenCalledWith('light')
  })
})
