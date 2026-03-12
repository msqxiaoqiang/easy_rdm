import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { nextTick } from 'vue'
import ArcoVue from '@arco-design/web-vue'
import ExportDialog from '@/components/views/ExportDialog.vue'
import zhCN from '@/i18n/locales/zh-CN'

vi.mock('@/utils/request', () => ({ request: vi.fn() }))
import { request } from '@/utils/request'
const mockRequest = vi.mocked(request)

// Mock platform 适配层，允许测试控制 hasNativeFileDialog
const mockHasNativeFileDialog = vi.fn(() => false)
vi.mock('@/utils/platform', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/platform')>()
  return { ...actual, hasNativeFileDialog: () => mockHasNativeFileDialog() }
})

const i18n = createI18n({ locale: 'zh-CN', messages: { 'zh-CN': zhCN }, legacy: false })

function createWrapper(props = {}) {
  return mount(ExportDialog, {
    props: { connId: 'test-conn', visible: true, selectedKeys: [], ...props },
    global: {
      plugins: [i18n, ArcoVue],
      stubs: { teleport: true },
    },
  })
}

describe('ExportDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(window as any).$gm = {
      _isMock: true,
      message: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
      chooseFolder: vi.fn(),
    }
  })

  it('renders scope options when no selectedKeys', async () => {
    const wrapper = createWrapper()
    await nextTick()
    const radios = wrapper.findAll('input[name="scope"]')
    expect(radios.length).toBe(2)
    expect(wrapper.text()).not.toContain('已选中')
  })

  it('shows key list and hides scope when selectedKeys provided', async () => {
    const wrapper = createWrapper({ selectedKeys: ['key1', 'key2', 'key3'] })
    await nextTick()
    const scopeRadios = wrapper.findAll('input[name="scope"]')
    expect(scopeRadios.length).toBe(0)
    expect(wrapper.text()).toContain('已选中 3 个键')
    const keyItems = wrapper.findAll('.selected-key-item')
    expect(keyItems.length).toBe(3)
    expect(keyItems[0].text()).toBe('key1')
  })

  it('footer only has cancel and export buttons', async () => {
    const wrapper = createWrapper()
    await nextTick()
    const footerBtns = wrapper.findAll('.arco-modal-footer .arco-btn')
    expect(footerBtns.length).toBe(2)
    expect(footerBtns[0].text()).toContain('取消')
    expect(footerBtns[1].text()).toContain('导出')
  })

  it('non-GM env uses fetch to download file on export click', async () => {
    const mockBlob = new Blob(['test'], { type: 'application/json' })
    const mockResponse = {
      blob: vi.fn().mockResolvedValue(mockBlob),
      headers: { get: vi.fn().mockReturnValue('attachment; filename="test.json"') },
    }
    const mockFetch = vi.fn().mockResolvedValue(mockResponse)
    globalThis.fetch = mockFetch

    const createObjectURL = vi.fn().mockReturnValue('blob:test')
    const revokeObjectURL = vi.fn()
    globalThis.URL.createObjectURL = createObjectURL
    globalThis.URL.revokeObjectURL = revokeObjectURL

    const wrapper = createWrapper()
    await nextTick()
    const exportBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    await exportBtn.trigger('click')
    await flushPromises()

    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('download_keys_export'), expect.objectContaining({
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    }))
    expect(createObjectURL).toHaveBeenCalled()
    expect(revokeObjectURL).toHaveBeenCalled()
  })

  it('does not show export-dir input in non-GM env', async () => {
    const wrapper = createWrapper()
    await nextTick()
    expect(wrapper.find('.export-dir-row').exists()).toBe(false)
  })

  it('shows export-dir input in GM env', async () => {
    mockHasNativeFileDialog.mockReturnValue(true)
    const wrapper = createWrapper()
    await nextTick()
    expect(wrapper.find('.export-dir-row').exists()).toBe(true)
  })

  it('GM env shows error when export-dir is empty', async () => {
    mockHasNativeFileDialog.mockReturnValue(true)
    const errorFn = vi.fn()
    ;(window as any).$gm = {
      message: { success: vi.fn(), error: errorFn, info: vi.fn() },
      chooseFolder: vi.fn(),
    }
    const wrapper = createWrapper()
    await nextTick()
    const exportBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    await exportBtn.trigger('click')
    await flushPromises()
    expect(errorFn).toHaveBeenCalledWith('请输入导出目录')
    expect(mockRequest).not.toHaveBeenCalled()
  })

  it('GM env calls export_keys_file with dir path', async () => {
    mockHasNativeFileDialog.mockReturnValue(true)
    mockRequest.mockResolvedValueOnce({
      code: 200, msg: 'OK',
      data: { path: '/tmp/test.json', count: 5 },
    } as any)
    ;(window as any).$gm = {
      message: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
      chooseFolder: vi.fn(),
    }
    const wrapper = createWrapper()
    await nextTick()

    // 设置导出目录
    const dirInput = wrapper.find('.export-dir-row input')
    await dirInput.setValue('/tmp/export')

    const exportBtn = wrapper.find('.arco-modal-footer .arco-btn-primary')
    await exportBtn.trigger('click')
    await flushPromises()

    expect(mockRequest).toHaveBeenCalledWith('export_keys_file', {
      params: expect.objectContaining({
        conn_id: 'test-conn',
        format: 'json',
        file_path: '/tmp/export',
      }),
    })
  })
})
