import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ArcoVue from '@arco-design/web-vue'
import SettingsModal from '@/components/settings/SettingsModal.vue'
import zhCN from '@/i18n/locales/zh-CN'

// Mock request
vi.mock('@/utils/request', () => ({
  request: vi.fn(),
}))

// Mock theme
vi.mock('@/utils/theme', () => ({
  applyTheme: vi.fn(),
}))

// Mock i18n loadLanguage
vi.mock('@/i18n', () => ({
  default: {},
  loadLanguage: vi.fn(),
}))

// Mock dialog
vi.mock('@/utils/dialog', () => ({
  gmConfirm: vi.fn().mockResolvedValue(true),
}))

// Mock settingsStore — 使用共享实例以便断言
const mockSettingsStoreInstance = {
  settings: {
    theme: 'dark', language: 'zh-CN', scanCount: 200,
    autoReconnect: true, retryInterval: 5, maxRetries: 10, heartbeatInterval: 30,
    editorFontSize: 14, showLineNumbers: true, cliFontSize: 14,
    defaultExportFormat: 'json', bufferLimit: 5000, scanBatchSize: 1000,
    dangerousCommandConfirm: true,
  },
  updateFromSave: vi.fn(),
  applySettings: vi.fn(),
}
vi.mock('@/stores/settings', () => ({
  useSettingsStore: vi.fn(() => mockSettingsStoreInstance),
}))

// Mock useShortcuts
vi.mock('@/composables/useShortcuts', () => ({
  shortcutActions: [
    { id: 'newKey', labelKey: 'shortcut.newKey', defaultBinding: 'Ctrl+N' },
    { id: 'refresh', labelKey: 'shortcut.refresh', defaultBinding: 'Ctrl+R' },
  ],
  getBindings: vi.fn(() => ({ newKey: 'Ctrl+N', refresh: 'Ctrl+R' })),
  setBinding: vi.fn(),
  resetShortcutBindings: vi.fn(),
  eventToBinding: vi.fn(),
  detectConflict: vi.fn(),
  formatBinding: vi.fn((b: string) => b),
}))

import { request } from '@/utils/request'
import { loadLanguage } from '@/i18n'
import { setBinding, resetShortcutBindings } from '@/composables/useShortcuts'

const mockRequest = vi.mocked(request)
const mockLoadLanguage = vi.mocked(loadLanguage)
const mockSetBinding = vi.mocked(setBinding)
const mockResetShortcutBindings = vi.mocked(resetShortcutBindings)

let wrapper: VueWrapper<any>

function createWrapper() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: { 'zh-CN': zhCN },
  })
  wrapper = mount(SettingsModal, {
    attachTo: document.body,
    global: {
      plugins: [pinia, i18n, ArcoVue],
    },
  })
  return wrapper
}

/** Arco Modal teleport 到 body，用原生查询 */
function bodyFind(selector: string) {
  return document.querySelector(selector)
}
function bodyFindAll(selector: string) {
  return Array.from(document.querySelectorAll(selector))
}

describe('SettingsModal.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })
  })

  afterEach(() => {
    wrapper?.unmount()
    // 清理 Arco Modal teleport 残留
    document.body.innerHTML = ''
  })

  // ========== 渲染测试 ==========

  it('应渲染设置弹框标题', async () => {
    createWrapper()
    await flushPromises()
    const title = bodyFind('.arco-modal-title')
    expect(title).toBeTruthy()
    expect(title!.textContent).toBe(zhCN.settings.title)
  })

  it('应渲染 5 个左侧 tab', async () => {
    createWrapper()
    await flushPromises()
    const tabs = bodyFindAll('.settings-tabs button')
    expect(tabs.length).toBe(5)
    expect(tabs[0].textContent).toBe(zhCN.settings.general)
    expect(tabs[1].textContent).toBe(zhCN.settings.connection)
    expect(tabs[2].textContent).toBe(zhCN.settings.advanced)
    expect(tabs[3].textContent).toBe(zhCN.shortcut.title)
    expect(tabs[4].textContent).toBe(zhCN.decoder.title)
  })

  it('默认显示常规配置 tab，第一个 tab 有 active class', async () => {
    createWrapper()
    await flushPromises()
    const tabs = bodyFindAll('.settings-tabs button')
    expect(tabs[0].classList.contains('active')).toBe(true)
  })

  it('应渲染底部按钮（重置、取消、保存）', async () => {
    createWrapper()
    await flushPromises()
    const btns = bodyFindAll('.settings-footer button')
    expect(btns.length).toBe(3)
  })

  // ========== Tab 切换 ==========

  it('点击 tab 切换内容区', async () => {
    createWrapper()
    await flushPromises()
    const tabs = bodyFindAll('.settings-tabs button') as HTMLElement[]

    // 点击连接配置
    tabs[1].click()
    await wrapper.vm.$nextTick()

    expect(tabs[1].classList.contains('active')).toBe(true)

    // 连接配置区域可见（v-show）
    const sections = bodyFindAll('.settings-section') as HTMLElement[]
    expect(sections[1].style.display).not.toBe('none')
  })

  // ========== 数据加载 ==========

  it('挂载时调用 get_settings 加载设置', async () => {
    mockRequest.mockResolvedValueOnce({
      code: 200,
      data: { theme: 'light', scanCount: 500 },
      msg: 'OK',
    })

    createWrapper()
    await flushPromises()

    expect(mockRequest).toHaveBeenCalledWith('get_settings', { params: {} })
  })

  it('get_settings 失败时使用默认值', async () => {
    mockRequest.mockRejectedValueOnce(new Error('network error'))

    createWrapper()
    await flushPromises()

    // 不应崩溃，使用默认值
    const title = bodyFind('.arco-modal-title')
    expect(title!.textContent).toBe(zhCN.settings.title)
  })

  // ========== 保存 ==========

  it('点击保存调用 save_settings 并应用主题和语言', async () => {
    mockRequest
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' }) // get_settings
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' }) // save_settings

    createWrapper()
    await flushPromises()

    // 点击保存按钮（最后一个）
    const saveBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    saveBtns[saveBtns.length - 1].click()
    await flushPromises()

    // 验证调用
    expect(mockRequest).toHaveBeenCalledWith('save_settings', expect.objectContaining({
      params: expect.objectContaining({
        theme: 'dark',
        language: 'zh-CN',
      }),
    }))
    // settingsStore.updateFromSave 应被调用（内部会 applyTheme）
    expect(mockSettingsStoreInstance.updateFromSave).toHaveBeenCalled()
    expect(mockLoadLanguage).toHaveBeenCalledWith('zh-CN')
  })

  it('保存成功后触发 saved 和 close 事件', async () => {
    mockRequest
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' }) // get_settings
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' }) // save_settings

    createWrapper()
    await flushPromises()

    const saveBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    saveBtns[saveBtns.length - 1].click()
    await flushPromises()

    expect(wrapper.emitted('saved')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('保存失败不触发 close 事件', async () => {
    mockRequest
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' }) // get_settings
      .mockRejectedValueOnce(new Error('save failed')) // save_settings

    createWrapper()
    await flushPromises()

    const saveBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    saveBtns[saveBtns.length - 1].click()
    await flushPromises()

    expect(wrapper.emitted('saved')).toBeFalsy()
    expect(wrapper.emitted('close')).toBeFalsy()
  })

  // ========== 重置 ==========

  it('点击重置恢复默认值', async () => {
    mockRequest.mockResolvedValueOnce({
      code: 200,
      data: { theme: 'light' },
      msg: 'OK',
    })

    createWrapper()
    await flushPromises()

    // 点击重置按钮（第一个）
    const footerBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    footerBtns[0].click()
    await wrapper.vm.$nextTick()

    // 重置后再保存，应该是默认值
    mockRequest.mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
    footerBtns[footerBtns.length - 1].click()
    await flushPromises()

    // save_settings 应使用默认值
    const saveCall = mockRequest.mock.calls.find(c => c[0] === 'save_settings')
    expect(saveCall).toBeTruthy()
    const saveParams = (saveCall as any[])[1].params
    expect(saveParams.theme).toBe('dark')
  })

  // ========== 取消 ==========

  it('点击取消触发 close 事件', async () => {
    createWrapper()
    await flushPromises()

    const footerBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    // 取消是倒数第二个
    footerBtns[footerBtns.length - 2].click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('close')).toBeTruthy()
  })

  // ========== 解码器 Tab ==========

  it('切换到解码器 tab 时加载解码器列表', async () => {
    mockRequest
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' }) // get_settings
      .mockResolvedValueOnce({
        code: 200,
        data: [{ id: 'd1', name: 'Base64', type: 'builtin', command: '' }],
        msg: 'OK',
      }) // get_decoders

    createWrapper()
    await flushPromises()

    // 切换到解码器 tab
    const tabs = bodyFindAll('.settings-tabs button') as HTMLElement[]
    tabs[4].click()
    await flushPromises()

    expect(mockRequest).toHaveBeenCalledWith('get_decoders', { params: {} })
  })

  // ========== 快捷键 ==========

  it('应渲染快捷键列表', async () => {
    createWrapper()
    await flushPromises()

    // 切换到快捷键 tab
    const tabs = bodyFindAll('.settings-tabs button') as HTMLElement[]
    tabs[3].click()
    await wrapper.vm.$nextTick()

    const rows = bodyFindAll('.shortcut-row')
    expect(rows.length).toBeGreaterThan(0)
  })

  // ========== Bug 1: 快捷键取消回滚 ==========

  it('取消时恢复快捷键快照到全局（不持久化录制期间的新绑定）', async () => {
    createWrapper()
    await flushPromises()
    // 清除 onMounted 期间的调用记录
    mockSetBinding.mockClear()

    const footerBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    footerBtns[footerBtns.length - 2].click()
    await wrapper.vm.$nextTick()

    // handleClose 恢复快照，应调用 setBinding 恢复原始绑定
    expect(mockSetBinding).toHaveBeenCalledWith('newKey', 'Ctrl+N')
    expect(mockSetBinding).toHaveBeenCalledWith('refresh', 'Ctrl+R')
  })

  it('保存时调用 setBinding 将快捷键写入全局', async () => {
    mockRequest
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })

    createWrapper()
    await flushPromises()

    const saveBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    saveBtns[saveBtns.length - 1].click()
    await flushPromises()

    expect(mockSetBinding).toHaveBeenCalled()
  })

  // ========== Bug 3: 重置含快捷键 ==========

  it('点击重置同时重置快捷键', async () => {
    createWrapper()
    await flushPromises()

    const footerBtns = bodyFindAll('.settings-footer button') as HTMLElement[]
    footerBtns[0].click()
    await wrapper.vm.$nextTick()

    expect(mockResetShortcutBindings).toHaveBeenCalled()
  })

  // ========== Bug 5: 解码器缓存 ==========

  it('多次切换到解码器 tab 只加载一次', async () => {
    mockRequest
      .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
      .mockResolvedValueOnce({ code: 200, data: [], msg: 'OK' })

    createWrapper()
    await flushPromises()

    const tabs = bodyFindAll('.settings-tabs button') as HTMLElement[]
    tabs[4].click()
    await flushPromises()
    tabs[0].click()
    await wrapper.vm.$nextTick()
    tabs[4].click()
    await flushPromises()

    const decoderCalls = mockRequest.mock.calls.filter(c => c[0] === 'get_decoders')
    expect(decoderCalls.length).toBe(1)
  })
})
