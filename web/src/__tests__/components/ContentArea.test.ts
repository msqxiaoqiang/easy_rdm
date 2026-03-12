import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ContentArea from '@/components/layout/ContentArea.vue'
import { useConnectionStore } from '@/stores/connection'
import zhCN from '@/i18n/locales/zh-CN'

vi.mock('@/utils/request', () => ({
  request: vi.fn().mockResolvedValue({ code: 200, data: null, msg: 'OK' }),
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
  store.connections = [{ id: 'c1', name: 'test', host: '127.0.0.1', port: 6379 }] as any
  store.setConnState('c1', { status: 'connected', currentDb: 0, cliDb: 0 })
  const wrapper = mount(ContentArea, {
    props: { connId: 'c1' },
    global: { plugins: [pinia, i18n] },
    shallow: true,
  })
  return { wrapper, store }
}

describe('ContentArea — 子 tab 切换 DB 恢复', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('从 CLI 切到其他 tab 时，cliDb !== currentDb 则调 select_db 恢复', async () => {
    const { wrapper, store } = createWrapper()
    store.setCliDb('c1', 3)
    // 先切到 CLI tab
    ;(wrapper.vm as any).activeSubTab = 'cli'
    await flushPromises()
    vi.clearAllMocks()

    // 从 CLI 切到 keyDetail
    ;(wrapper.vm as any).activeSubTab = 'keyDetail'
    await flushPromises()

    expect(request).toHaveBeenCalledWith('select_db', {
      params: { conn_id: 'c1', db: 0 },
    })
  })

  it('从其他 tab 切到 CLI 时，cliDb !== currentDb 则调 select_db 切换', async () => {
    const { wrapper, store } = createWrapper()
    store.setCliDb('c1', 5)
    await wrapper.vm.$nextTick()
    vi.clearAllMocks()

    // 从 status 切到 cli
    ;(wrapper.vm as any).activeSubTab = 'cli'
    await flushPromises()

    expect(request).toHaveBeenCalledWith('select_db', {
      params: { conn_id: 'c1', db: 5 },
    })
  })

  it('cliDb === currentDb 时切换 tab 不发 select_db 请求', async () => {
    const { wrapper } = createWrapper()
    // cliDb=0, currentDb=0, 相同
    ;(wrapper.vm as any).activeSubTab = 'cli'
    await flushPromises()
    // 检查没有 select_db 调用
    const selectDbCalls = vi.mocked(request).mock.calls.filter(
      c => c[0] === 'select_db'
    )
    expect(selectDbCalls).toHaveLength(0)
  })
})
