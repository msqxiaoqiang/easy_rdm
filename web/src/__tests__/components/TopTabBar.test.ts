import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import TopTabBar from '@/components/layout/TopTabBar.vue'
import { useConnectionStore } from '@/stores/connection'
import zhCN from '@/i18n/locales/zh-CN'

// Mock request module
vi.mock('@/utils/request', () => ({
  request: vi.fn().mockResolvedValue({ code: 200, data: null, msg: 'OK' }),
}))

function createWrapper() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: { 'zh-CN': zhCN },
  })
  return {
    wrapper: mount(TopTabBar, {
      global: { plugins: [pinia, i18n] },
    }),
    store: useConnectionStore(),
  }
}

describe('TopTabBar.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('无 Tab 时应渲染空 Tab 列表', () => {
    const { wrapper } = createWrapper()
    expect(wrapper.findAll('.tab-item')).toHaveLength(0)
  })

  it('应渲染所有 Tab', () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false },
      { id: 't2', name: 'Redis-2', host: '10.0.0.1', port: 6380, pinned: false },
    ]

    // Vue reactivity needs a tick
    return wrapper.vm.$nextTick().then(() => {
      const tabs = wrapper.findAll('.tab-item')
      expect(tabs).toHaveLength(2)
      expect(tabs[0].find('.tab-name').text()).toBe('Redis-1')
      expect(tabs[1].find('.tab-name').text()).toBe('Redis-2')
    })
  })

  it('Tab 不应显示地址信息', async () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false },
    ]
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.tab-addr').exists()).toBe(false)
  })

  it('激活的 Tab 应有 active 样式', async () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false },
      { id: 't2', name: 'Redis-2', host: '10.0.0.1', port: 6380, pinned: false },
    ]
    store.activeTabId = 't2'
    await wrapper.vm.$nextTick()

    const tabs = wrapper.findAll('.tab-item')
    expect(tabs[0].classes()).not.toContain('active')
    expect(tabs[1].classes()).toContain('active')
  })

  it('点击 Tab 应切换激活状态', async () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false },
      { id: 't2', name: 'Redis-2', host: '10.0.0.1', port: 6380, pinned: false },
    ]
    store.activeTabId = 't1'
    await wrapper.vm.$nextTick()

    await wrapper.findAll('.tab-item')[1].trigger('click')
    expect(store.activeTabId).toBe('t2')
  })

  it('未固定的 Tab 应显示关闭按钮', async () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false },
    ]
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.tab-close').exists()).toBe(true)
    expect(wrapper.find('.tab-pin').exists()).toBe(false)
  })

  it('固定的 Tab 应显示图钉而非关闭按钮', async () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: true },
    ]
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.tab-close').exists()).toBe(false)
    expect(wrapper.find('.tab-pin').exists()).toBe(true)
  })

  it('点击关闭按钮应移除 Tab', async () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false },
      { id: 't2', name: 'Redis-2', host: '10.0.0.1', port: 6380, pinned: false },
    ]
    store.activeTabId = 't1'
    await wrapper.vm.$nextTick()

    await wrapper.findAll('.tab-close')[0].trigger('click')
    await flushPromises()

    expect(store.tabs).toHaveLength(1)
    expect(store.tabs[0].id).toBe('t2')
  })

  it('已连接状态的 Tab 应显示 connected 图标', async () => {
    const { wrapper, store } = createWrapper()
    store.tabs = [
      { id: 't1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false },
    ]
    store.setConnState('t1', { status: 'connected', currentDb: 0, cliDb: 0, redisVersion: '7.0.0' })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.tab-icon').classes()).toContain('connected')
  })

  // 设置和日志按钮已移至 Sidebar，TopTabBar 不再包含这些按钮
})
