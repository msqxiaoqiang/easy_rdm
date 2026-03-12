import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { nextTick } from 'vue'
import ArcoVue from '@arco-design/web-vue'
import BatchDialog from '@/components/views/BatchDialog.vue'
import zhCN from '@/i18n/locales/zh-CN'

vi.mock('@/utils/request', () => ({ request: vi.fn() }))
import { request } from '@/utils/request'

function createWrapper(props?: Partial<{ isCluster: boolean }>) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: { 'zh-CN': zhCN },
  })
  return mount(BatchDialog, {
    props: {
      connId: 'test-conn',
      visible: true,
      isCluster: props?.isCluster ?? false,
    },
    global: {
      plugins: [pinia, i18n, ArcoVue],
      stubs: { teleport: true },
    },
  })
}

describe('BatchDialog.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('操作选项包含 delete', async () => {
    const wrapper = createWrapper()
    await nextTick()
    const select = wrapper.find('.arco-select')
    expect(select.exists()).toBe(true)
  })

  it('directExecute 勾选框默认未勾选（安全路径：先预览）', async () => {
    const wrapper = createWrapper()
    await nextTick()
    const checkbox = wrapper.find('.check-row.arco-checkbox')
    expect(checkbox.exists()).toBe(true)
    // 默认未勾选 — 安全路径
    expect(checkbox.classes()).not.toContain('arco-checkbox-checked')
  })

  it('默认模式先预览：点击按钮扫描后显示 key 列表', async () => {
    vi.mocked(request).mockResolvedValueOnce({
      code: 200,
      data: { keys: ['key1', 'key2', 'key3'], total: 3 },
      msg: 'OK',
    })
    const wrapper = createWrapper()
    await nextTick()
    const confirmBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    await confirmBtn.trigger('click')
    await flushPromises()
    expect(wrapper.find('.preview-list').exists()).toBe(true)
    expect(wrapper.findAll('.preview-item').length).toBe(3)
    // 仅调用 scan，未调用批量操作
    expect(request).toHaveBeenCalledTimes(1)
  })

  it('预览列表超过 100 个时显示省略提示', async () => {
    const manyKeys = Array.from({ length: 120 }, (_, i) => `key:${i}`)
    vi.mocked(request).mockResolvedValueOnce({
      code: 200,
      data: { keys: manyKeys, total: 120 },
      msg: 'OK',
    })
    const wrapper = createWrapper()
    await nextTick()
    const confirmBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    await confirmBtn.trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.preview-item').length).toBe(100)
    expect(wrapper.find('.preview-more').exists()).toBe(true)
  })

  it('预览后再次点击 footer 按钮才真正执行批量操作', async () => {
    vi.mocked(request)
      .mockResolvedValueOnce({
        code: 200,
        data: { keys: ['key1', 'key2'], total: 2 },
        msg: 'OK',
      })
      .mockResolvedValueOnce({
        code: 200,
        data: { success: 2, failed: 0 },
        msg: 'OK',
      })
    const wrapper = createWrapper()
    await nextTick()
    const confirmBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    // 第一次点击 — 预览
    await confirmBtn.trigger('click')
    await flushPromises()
    expect(wrapper.find('.preview-list').exists()).toBe(true)
    // 第二次点击 — 执行
    await confirmBtn.trigger('click')
    await flushPromises()
    expect(request).toHaveBeenCalledTimes(2)
  })

  it('勾选 directExecute 后直接执行 batch_delete_keys', async () => {
    vi.mocked(request)
      .mockResolvedValueOnce({
        code: 200,
        data: { keys: ['k1', 'k2'], total: 2 },
        msg: 'OK',
      })
      .mockResolvedValueOnce({
        code: 200,
        data: { success: 2, failed: 0 },
        msg: 'OK',
      })
    const wrapper = createWrapper()
    await nextTick()
    const vm = wrapper.vm as any
    vm.operation = 'delete'
    vm.directExecute = true
    await nextTick()
    const confirmBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    await confirmBtn.trigger('click')
    await flushPromises()
    expect(request).toHaveBeenCalledWith('batch_delete_keys', {
      params: { conn_id: 'test-conn', keys: ['k1', 'k2'] },
    })
  })
})
