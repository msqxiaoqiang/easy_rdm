import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import CliView from '@/components/views/CliView.vue'
import { useConnectionStore } from '@/stores/connection'
import zhCN from '@/i18n/locales/zh-CN'

vi.mock('@/utils/request', () => ({
  request: vi.fn(),
}))

import { request } from '@/utils/request'

function createWrapper() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: { 'zh-CN': zhCN },
  })
  const store = useConnectionStore()
  store.setConnState('test-conn', { status: 'connected', currentDb: 0, cliDb: 0 })
  const wrapper = mount(CliView, {
    props: { connId: 'test-conn' },
    global: { plugins: [pinia, i18n] },
  })
  return { wrapper, store }
}

describe('CliView.vue — SELECT 命令更新 db', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('SELECT 命令成功后更新 currentDb', async () => {
    vi.mocked(request).mockResolvedValueOnce({
      code: 200, data: { result: 'OK' }, msg: 'OK',
    })
    const { wrapper, store } = createWrapper()
    const input = wrapper.find('.cli-input')
    await input.setValue('SELECT 3')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    const state = store.getConnState('test-conn')
    expect(state.cliDb).toBe(3)
  })

  it('SELECT 后提示符显示新 db', async () => {
    vi.mocked(request).mockResolvedValueOnce({
      code: 200, data: { result: 'OK' }, msg: 'OK',
    })
    const { wrapper } = createWrapper()
    const input = wrapper.find('.cli-input')
    await input.setValue('select 5')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    // 输入行的 prompt 应该包含 db5
    const prompts = wrapper.findAll('.cli-prompt')
    const lastPrompt = prompts[prompts.length - 1]
    expect(lastPrompt.text()).toContain('db5')
  })

  it('SELECT 成功后只更新 cliDb，不影响 currentDb', async () => {
    vi.mocked(request).mockResolvedValueOnce({
      code: 200, data: { result: 'OK' }, msg: 'OK',
    })
    const { wrapper, store } = createWrapper()
    store.setCurrentDb('test-conn', 2)
    const input = wrapper.find('.cli-input')
    await input.setValue('SELECT 5')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    const state = store.getConnState('test-conn')
    expect(state.cliDb).toBe(5)
    expect(state.currentDb).toBe(2)
  })

  it('SELECT 失败不更新 currentDb', async () => {
    vi.mocked(request).mockResolvedValueOnce({
      code: 200, data: { result: 'ERR invalid DB index' }, msg: 'OK',
    })
    const { wrapper, store } = createWrapper()
    const input = wrapper.find('.cli-input')
    await input.setValue('SELECT 99')
    await input.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(store.getConnState('test-conn').cliDb).toBe(0)
  })
})
