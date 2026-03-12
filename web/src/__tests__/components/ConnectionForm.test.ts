import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { defineComponent } from 'vue'
import ArcoVue from '@arco-design/web-vue'
import ConnectionForm from '@/components/connection/ConnectionForm.vue'
import zhCN from '@/i18n/locales/zh-CN'

const ClientOnlyStub = defineComponent({
  name: 'ClientOnly',
  setup(_, { slots }) { return () => slots.default?.() },
})

vi.mock('@/utils/request', () => ({
  request: vi.fn(),
}))

import { request } from '@/utils/request'
const mockRequest = vi.mocked(request)

function createWrapper(props: Record<string, any> = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: { 'zh-CN': zhCN },
  })
  return mount(ConnectionForm, {
    props,
    global: {
      plugins: [pinia, i18n, ArcoVue],
      stubs: { ClientOnly: ClientOnlyStub, Teleport: true },
    },
  })
}

/** 找到 footer 区域的所有按钮 */
function footerBtns(wrapper: ReturnType<typeof createWrapper>) {
  return wrapper.findAll('.arco-modal-footer button.arco-btn')
}

describe('ConnectionForm.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ========== 渲染测试 ==========

  it('新建模式应显示"新建连接"标题', () => {
    const wrapper = createWrapper()
    expect(wrapper.find('.arco-modal-title').text()).toBe(zhCN.connection.new)
  })

  it('编辑模式应显示"编辑连接"标题', () => {
    const conn = {
      id: 'edit-1', name: 'Test', host: '10.0.0.1', port: 6380,
      db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10,
    }
    const wrapper = createWrapper({ connection: conn })
    expect(wrapper.find('.arco-modal-title').text()).toBe(zhCN.connection.edit)
  })

  it('应渲染所有基本表单字段', () => {
    const wrapper = createWrapper()
    // a-input renders input.arco-input; a-input-number also has input inside
    const inputs = wrapper.findAll('.arco-input')
    // name, host, password, username, group = 5 a-input + port, db = 2 a-input-number
    expect(inputs.length).toBeGreaterThanOrEqual(5)
  })

  it('默认值应为 127.0.0.1:6379', () => {
    const wrapper = createWrapper()
    const textInputs = wrapper.findAll('.conn-form-section .arco-input')
    // 顺序: name, host, password(inside input-password), username, group
    // host 是第二个 a-input
    const hostInput = textInputs[1]?.element as HTMLInputElement
    expect(hostInput?.value).toBe('127.0.0.1')
    // port 在 a-input-number 中
    const numInputs = wrapper.findAll('.arco-input-number input')
    expect((numInputs[0]?.element as HTMLInputElement)?.value).toBe('6379')
  })

  it('默认显示常规 Tab', () => {
    const wrapper = createWrapper()
    const activeTab = wrapper.find('.conn-form-tab.active')
    expect(activeTab.exists()).toBe(true)
    // 常规 Tab 应默认选中（第一个）
    const tabs = wrapper.findAll('.conn-form-tab')
    expect(tabs[0].classes()).toContain('active')
  })

  it('点击高级 Tab 应切换到高级配置', async () => {
    const wrapper = createWrapper()
    const tabs = wrapper.findAll('.conn-form-tab')
    // 高级是第 4 个 tab
    await tabs[3].trigger('click')
    expect(tabs[3].classes()).toContain('active')
  })

  // ========== 测试连接 ==========

  it('测试连接成功应显示成功提示', async () => {
    mockRequest.mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
    mockRequest.mockResolvedValue({ code: 200, data: [], msg: 'OK' })

    const mockSuccess = vi.fn()
    ;(window as any).$gm = { message: { success: mockSuccess, error: vi.fn() } }

    const wrapper = createWrapper()
    const btns = footerBtns(wrapper)
    const testBtn = btns[0]
    expect(testBtn.text()).toBe(zhCN.connection.test)

    await testBtn.trigger('click')
    await flushPromises()

    expect(mockSuccess).toHaveBeenCalledWith(zhCN.connection.testSuccess)
  })

  it('测试连接失败（code=500）应显示错误提示', async () => {
    mockRequest.mockRejectedValueOnce(new Error('连接失败: NOAUTH Authentication required'))

    const mockError = vi.fn()
    ;(window as any).$gm = { message: { success: vi.fn(), error: mockError } }

    const wrapper = createWrapper()
    await footerBtns(wrapper)[0].trigger('click')
    await flushPromises()

    expect(mockError).toHaveBeenCalled()
    expect(mockError.mock.calls[0][0]).toContain('NOAUTH')
  })

  it('测试连接失败（网络错误）应显示错误提示', async () => {
    mockRequest.mockRejectedValueOnce(new Error('Network Error'))

    const mockError = vi.fn()
    ;(window as any).$gm = { message: { success: vi.fn(), error: mockError } }

    const wrapper = createWrapper()
    await footerBtns(wrapper)[0].trigger('click')
    await flushPromises()

    expect(mockError).toHaveBeenCalled()
    expect(mockError.mock.calls[0][0]).toContain('Network Error')
  })

  it('测试中按钮应禁用并显示"测试中..."', async () => {
    let resolvePromise: Function
    mockRequest.mockReturnValueOnce(new Promise(resolve => { resolvePromise = resolve }))

    const wrapper = createWrapper()
    const testBtn = footerBtns(wrapper)[0]

    await testBtn.trigger('click')
    await wrapper.vm.$nextTick()

    // loading 状态下 Arco button 添加 loading class
    expect(testBtn.classes()).toContain('arco-btn-loading')

    resolvePromise!({ code: 200, data: null, msg: 'OK' })
    await flushPromises()
  })

  // ========== 保存 ==========

  it('保存应调用 save_connection 并触发 saved 事件', async () => {
    mockRequest.mockResolvedValue({ code: 200, data: [], msg: 'OK' })

    const wrapper = createWrapper()
    // 填写名称 - 第一个 a-input
    const nameInput = wrapper.find('.conn-form-section input.arco-input')
    await nameInput.setValue('My Redis')

    const saveBtn = wrapper.find('.arco-modal-footer button.arco-btn-primary')
    await saveBtn.trigger('click')
    await flushPromises()

    expect(mockRequest).toHaveBeenCalledWith(
      'save_connection',
      expect.objectContaining({ params: expect.objectContaining({ name: 'My Redis' }) }),
    )
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('名称为空时保存应自动填充 host:port', async () => {
    mockRequest.mockResolvedValue({ code: 200, data: [], msg: 'OK' })

    const wrapper = createWrapper()
    const saveBtn = wrapper.find('.arco-modal-footer button.arco-btn-primary')
    await saveBtn.trigger('click')
    await flushPromises()

    const savedCall = mockRequest.mock.calls.find(c => c[0] === 'save_connection')
    expect(savedCall).toBeTruthy()
    expect((savedCall as any)[1].params.name).toBe('127.0.0.1:6379')
  })

  // ========== 关闭 ==========

  it('点击关闭按钮应触发 close 事件', async () => {
    const wrapper = createWrapper()
    const closeBtn = wrapper.find('.arco-modal-close-btn')
    await closeBtn.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('点击遮罩层应触发 close 事件', async () => {
    const wrapper = createWrapper()
    const modal = wrapper.findComponent({ name: 'Modal' })
    await modal.vm.$emit('cancel')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('点击取消按钮应触发 close 事件', async () => {
    const wrapper = createWrapper()
    const btns = footerBtns(wrapper)
    // 按钮顺序: 测试, 取消, 保存
    const cancelBtn = btns[1]
    expect(cancelBtn.text()).toBe(zhCN.common.cancel)
    await cancelBtn.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  // ========== 编辑模式 ==========

  it('编辑模式应填充已有连接数据', async () => {
    const conn = {
      id: 'edit-1', name: 'Prod Redis', host: '10.0.0.1', port: 6380,
      db: 2, conn_type: 'tcp', conn_timeout: 30, exec_timeout: 30,
    }
    const wrapper = createWrapper({ connection: conn })
    await flushPromises()

    const textInputs = wrapper.findAll('.conn-form-section input.arco-input')
    expect((textInputs[0].element as HTMLInputElement).value).toBe('Prod Redis')
    expect((textInputs[1].element as HTMLInputElement).value).toBe('10.0.0.1')
    // port 在 a-input-number 中
    const numInputs = wrapper.findAll('.arco-input-number input')
    expect((numInputs[0].element as HTMLInputElement).value).toBe('6380')
  })

  it('编辑已加密密码的连接时密码字段应为空', async () => {
    const conn = {
      id: 'enc-1', name: 'Encrypted', host: '10.0.0.1', port: 6379,
      password: 'encrypted-blob', password_encrypted: true,
      db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10,
    }
    const wrapper = createWrapper({ connection: conn })
    await flushPromises()

    // password 在 a-input-password 中
    const pwdInput = wrapper.find('.arco-input-wrapper input[type="password"]')
    expect((pwdInput.element as HTMLInputElement).value).toBe('')
  })
})
