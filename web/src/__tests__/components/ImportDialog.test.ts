import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { nextTick } from 'vue'
import ArcoVue from '@arco-design/web-vue'
import ImportDialog from '@/components/views/ImportDialog.vue'
import zhCN from '@/i18n/locales/zh-CN'

vi.mock('@/utils/request', () => ({ request: vi.fn() }))

const i18n = createI18n({ locale: 'zh-CN', messages: { 'zh-CN': zhCN }, legacy: false })

function createWrapper(props = {}) {
  return mount(ImportDialog, {
    props: { connId: 'test-conn', visible: true, ...props },
    global: {
      plugins: [i18n, ArcoVue],
      stubs: { teleport: true },
    },
  })
}

describe('ImportDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(window as any).$gm = {
      _isMock: true,
      message: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
      chooseFile: vi.fn(),
    }
  })

  it('renders format and conflict radio options', async () => {
    const wrapper = createWrapper()
    await nextTick()
    const formatRadios = wrapper.findAll('input[name="format"]')
    expect(formatRadios.length).toBe(3)
    const conflictRadios = wrapper.findAll('input[name="conflict"]')
    expect(conflictRadios.length).toBe(2)
  })

  it('renders file select row with placeholder', async () => {
    const wrapper = createWrapper()
    await nextTick()
    expect(wrapper.find('.file-select-row').exists()).toBe(true)
  })

  it('does not render textarea', async () => {
    const wrapper = createWrapper()
    await nextTick()
    expect(wrapper.find('textarea').exists()).toBe(false)
  })

  it('import button is disabled when no file selected', async () => {
    const wrapper = createWrapper()
    await nextTick()
    const importBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    expect(importBtn.attributes('disabled')).toBeDefined()
  })

  it('footer only has cancel and import buttons', async () => {
    const wrapper = createWrapper()
    await nextTick()
    const footerBtns = wrapper.findAll('.arco-modal-footer .arco-btn')
    expect(footerBtns.length).toBe(2)
    expect(footerBtns[0].text()).toContain('取消')
    expect(footerBtns[1].text()).toContain('导入')
  })
})
