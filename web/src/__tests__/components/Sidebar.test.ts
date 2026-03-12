import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ArcoVue from '@arco-design/web-vue'
import Sidebar from '@/components/layout/Sidebar.vue'
import { useConnectionStore } from '@/stores/connection'
import zhCN from '@/i18n/locales/zh-CN'
import { idToGroupKey } from '@/utils/sidebar-tree'

// Mock request module
const mockRequest = vi.fn().mockResolvedValue({ code: 200, data: null, msg: 'OK' })
vi.mock('@/utils/request', () => ({
  request: (...args: any[]) => mockRequest(...args),
}))

// Mock dialog module
const mockGmConfirm = vi.fn().mockResolvedValue(false)
const mockGmPrompt = vi.fn().mockResolvedValue(null)
vi.mock('@/utils/dialog', () => ({
  gmConfirm: (...args: any[]) => mockGmConfirm(...args),
  gmPrompt: (...args: any[]) => mockGmPrompt(...args),
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
    wrapper: mount(Sidebar, {
      global: {
        plugins: [pinia, i18n, ArcoVue],
        stubs: {
          // Teleport 在 jsdom 中需要 stub
          Teleport: true,
        },
      },
    }),
    store: useConnectionStore(),
  }
}

describe('Sidebar.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ========== 空状态 ==========

  it('无连接时应显示空状态', () => {
    const { wrapper } = createWrapper()
    const emptyState = wrapper.find('.empty-state')
    expect(emptyState.exists()).toBe(true)
    expect(emptyState.text()).toContain(zhCN.common.noData)
  })

  it('空状态应有新建连接按钮', () => {
    const { wrapper } = createWrapper()
    const emptyState = wrapper.find('.empty-state')
    // Arco Button 渲染为 <button class="arco-btn arco-btn-primary ...">
    const btn = emptyState.find('.arco-btn')
    expect(btn.exists()).toBe(true)
    expect(emptyState.text()).toContain(zhCN.connection.new)
  })

  it('有分组但无连接时不应显示空状态', async () => {
    const { wrapper, store } = createWrapper()
    store.groups = [idToGroupKey('staging')]
    await wrapper.vm.$nextTick()

    const emptyState = wrapper.find('.empty-state')
    expect(emptyState.exists()).toBe(false)
  })

  // ========== 工具栏 ==========

  it('应渲染 4 个工具栏按钮', () => {
    const { wrapper } = createWrapper()
    const toolbar = wrapper.find('.sidebar-toolbar')
    const btns = toolbar.findAll('.arco-btn')
    expect(btns).toHaveLength(4)
  })

  it('点击新建按钮应显示连接表单', async () => {
    const { wrapper } = createWrapper()
    const toolbar = wrapper.find('.sidebar-toolbar')
    const addBtn = toolbar.findAll('.arco-btn')[0]
    await addBtn.trigger('click')
    // ConnectionForm 应该被渲染
    expect(wrapper.findComponent({ name: 'ConnectionForm' }).exists()).toBe(true)
  })

  // ========== 工具栏布局 ==========

  it('不应有搜索筛选输入框', () => {
    const { wrapper } = createWrapper()
    const searchBox = wrapper.find('.sidebar-search')
    expect(searchBox.exists()).toBe(false)
  })

  // ========== 连接列表 ==========

  it('应渲染连接列表', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
      { id: 'c2', name: 'Redis-2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    await wrapper.vm.$nextTick()

    const items = wrapper.findAll('.connection-item')
    expect(items).toHaveLength(2)
    expect(items[0].find('.conn-name').text()).toBe('Redis-1')
    expect(items[1].find('.conn-name').text()).toBe('Redis-2')
  })

  it('已连接的连接项应有 connected 样式', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    store.setConnState('c1', { status: 'connected', currentDb: 0, cliDb: 0 })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.connection-item').classes()).toContain('connected')
    expect(wrapper.find('.status-dot').classes()).toContain('connected')
  })

  // ========== 分组 ==========

  it('应按分组展示连接', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
      { id: 'c2', name: 'Redis-2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    await wrapper.vm.$nextTick()

    const groupHeaders = wrapper.findAll('.group-header')
    expect(groupHeaders).toHaveLength(1)
    expect(groupHeaders[0].find('.group-name').text()).toBe('prod')
    expect(groupHeaders[0].find('.group-count').text()).toBe('(1)')
  })

  it('Arco Tree 应默认展开所有文件夹', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-Prod', host: '10.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    await wrapper.vm.$nextTick()

    // 初始展开 — 连接应可见
    expect(wrapper.findAll('.connection-item')).toHaveLength(1)
    // 文件夹头部应存在
    expect(wrapper.findAll('.group-header')).toHaveLength(1)
  })

  // ========== 空分组显示 ==========

  it('新建空分组后应在列表中显示分组头', async () => {
    const { wrapper, store } = createWrapper()
    // 模拟 displayOrder 中有一个空分组
    store.groups = [idToGroupKey('staging')]
    await wrapper.vm.$nextTick()

    const groupHeaders = wrapper.findAll('.group-header')
    expect(groupHeaders).toHaveLength(1)
    expect(groupHeaders[0].find('.group-name').text()).toBe('staging')
    expect(groupHeaders[0].find('.group-count').text()).toBe('(0)')
  })

  it('空分组和有连接的分组应同时显示', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    store.groups = [idToGroupKey('staging')]
    await wrapper.vm.$nextTick()

    const groupHeaders = wrapper.findAll('.group-header')
    expect(groupHeaders).toHaveLength(2)
    const names = groupHeaders.map(h => h.find('.group-name').text())
    expect(names).toContain('prod')
    expect(names).toContain('staging')
  })

  it('空分组不应重复已有连接的分组', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    // displayOrder 中也包含 prod，不应重复
    store.groups = [idToGroupKey('prod'), idToGroupKey('staging')]
    await wrapper.vm.$nextTick()

    const groupHeaders = wrapper.findAll('.group-header')
    expect(groupHeaders).toHaveLength(2)
  })

  // ========== 连接交互 ==========

  it('单击连接项只高亮不切换 activeTab', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    await wrapper.vm.$nextTick()

    await wrapper.find('.connection-item').trigger('click')
    // 单击不应设置 activeTabId
    expect(store.activeTabId).toBe('')
    // 应该有 highlighted class
    expect(wrapper.find('.connection-item').classes()).toContain('highlighted')
  })

  it('右键连接项应显示上下文菜单', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    await wrapper.vm.$nextTick()

    await wrapper.find('.connection-item').trigger('contextmenu', {
      clientX: 100,
      clientY: 200,
    })
    await wrapper.vm.$nextTick()

    // ContextMenu 组件应被渲染（被 stub 为 Teleport）
    expect(wrapper.findComponent({ name: 'ContextMenu' }).exists()).toBe(true)
  })

  // ========== 分组操作按钮 ==========

  it('分组头 hover 时显示操作按钮', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    await wrapper.vm.$nextTick()
    const groupHeader = wrapper.find('.group-header')
    expect(groupHeader.exists()).toBe(true)
    expect(groupHeader.find('.group-actions').exists()).toBe(true)
    expect(groupHeader.findAll('.group-action-btn')).toHaveLength(3)
  })

  // ========== 分组头样式与图标 ==========

  it('分组头应包含文件夹图标', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.group-header .group-folder-icon').exists()).toBe(true)
  })

  it('分组头按钮顺序：新建连接、编辑、删除', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    await wrapper.vm.$nextTick()
    const btns = wrapper.findAll('.group-action-btn')
    expect(btns).toHaveLength(3)
    // 第3个按钮应有 danger class（删除）
    expect(btns[2].classes()).toContain('danger')
    // 前两个不应有 danger
    expect(btns[0].classes()).not.toContain('danger')
    expect(btns[1].classes()).not.toContain('danger')
  })

  // ========== 重命名分组 ==========

  it('点击编辑按钮应调用 gmPrompt 并更新分组显示名', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    store.groups = [idToGroupKey('prod')]
    await wrapper.vm.$nextTick()

    mockGmPrompt.mockResolvedValueOnce('production')

    const editBtn = wrapper.findAll('.group-action-btn')[1]
    await editBtn.trigger('click')
    await new Promise(r => setTimeout(r, 50))

    expect(mockGmPrompt).toHaveBeenCalled()
    // groupMeta 应被更新
    expect(store.groupMeta['prod']).toBe('production')
    // 连接的 group 字段不变（存的是 ID）
    expect(store.connections[0].group).toBe('prod')
    // groups 中的 key 不变
    expect(store.groups).toContain(idToGroupKey('prod'))
    // 应调用 saveGroups 持久化（save_groups + save_group_meta）
    expect(mockRequest).toHaveBeenCalledWith('save_groups', expect.anything())
    expect(mockRequest).toHaveBeenCalledWith('save_group_meta', expect.anything())
    // 不应调用 save_connection
    expect(mockRequest).not.toHaveBeenCalledWith('save_connection', expect.anything())
  })

  it('重命名分组时输入空或相同名称不应生效', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    store.groups = [idToGroupKey('prod')]
    await wrapper.vm.$nextTick()

    mockGmPrompt.mockResolvedValueOnce('prod') // 相同名称
    mockRequest.mockClear()

    const editBtn = wrapper.findAll('.group-action-btn')[1]
    await editBtn.trigger('click')
    await new Promise(r => setTimeout(r, 50))

    expect(store.connections[0].group).toBe('prod')
    // 不应调用任何保存接口
    expect(mockRequest).not.toHaveBeenCalledWith('save_groups', expect.anything())
    expect(mockRequest).not.toHaveBeenCalledWith('save_group_meta', expect.anything())
  })

  // ========== 删除分组 ==========

  it('删除分组确认文案应包含连接数量', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
      { id: 'c2', name: 'R2', host: '127.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    store.groups = [idToGroupKey('prod')]
    await wrapper.vm.$nextTick()

    mockGmConfirm.mockResolvedValueOnce(false)

    const deleteBtn = wrapper.findAll('.group-action-btn.danger')[0]
    await deleteBtn.trigger('click')
    await new Promise(r => setTimeout(r, 50))

    // 确认框文案应包含数量 2
    expect(mockGmConfirm).toHaveBeenCalledWith(expect.stringContaining('2'))
  })

  it('确认删除分组应真删除组内连接', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    store.groups = [idToGroupKey('prod')]
    await wrapper.vm.$nextTick()

    mockGmConfirm.mockResolvedValueOnce(true)
    const deleteGroupSpy = vi.spyOn(store, 'deleteGroup').mockResolvedValue(undefined)

    const deleteBtn = wrapper.findAll('.group-action-btn.danger')[0]
    await deleteBtn.trigger('click')
    await new Promise(r => setTimeout(r, 50))

    expect(deleteGroupSpy).toHaveBeenCalledWith('prod')
  })

  // ========== 断开连接关闭 tab ==========

  it('断开连接后应关闭对应 tab', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    store.setConnState('c1', { status: 'connected', currentDb: 0, cliDb: 0 })
    store.tabs = [{ id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, pinned: false }]
    store.activeTabId = 'c1'
    await wrapper.vm.$nextTick()

    // 触发右键菜单
    await wrapper.find('.connection-item').trigger('contextmenu', { clientX: 100, clientY: 200 })
    await wrapper.vm.$nextTick()

    // spy disconnect 和 closeTab
    const disconnectSpy = vi.spyOn(store, 'disconnect').mockResolvedValue(undefined)
    const closeTabSpy = vi.spyOn(store, 'closeTab')

    // 模拟选择 disconnect
    const ctxMenuComp = wrapper.findComponent({ name: 'ContextMenu' })
    ctxMenuComp.vm.$emit('select', 'disconnect')
    await wrapper.vm.$nextTick()
    // 等待 async handler
    await new Promise(r => setTimeout(r, 50))

    expect(disconnectSpy).toHaveBeenCalledWith('c1')
    expect(closeTabSpy).toHaveBeenCalledWith('c1')
  })

  // ========== 拖拽排序结构 ==========

  it('连接项应有 data-conn-id 属性', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    await wrapper.vm.$nextTick()

    const item = wrapper.find('.connection-item')
    expect(item.attributes('data-conn-id')).toBe('c1')
  })

  it('分组连接应在 Arco Tree 中渲染', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    await wrapper.vm.$nextTick()

    // 文件夹头部应存在
    expect(wrapper.find('.group-header').exists()).toBe(true)
    expect(wrapper.find('.group-header .group-name').text()).toBe('prod')
    // 连接项应有 data-conn-id
    expect(wrapper.find('.connection-item').attributes('data-conn-id')).toBe('c1')
  })

  it('未分组连接应在 Arco Tree 中渲染', async () => {
    const { wrapper, store } = createWrapper()
    store.connections = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    await wrapper.vm.$nextTick()

    // 连接项应直接渲染为叶子节点
    const item = wrapper.find('.connection-item')
    expect(item.exists()).toBe(true)
    expect(item.find('.conn-name').text()).toBe('Redis-1')
  })
})
