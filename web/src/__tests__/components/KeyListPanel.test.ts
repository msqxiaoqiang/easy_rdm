import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { nextTick, toRaw } from 'vue'
import { useConnectionStore } from '@/stores/connection'
import KeyListPanel from '@/components/views/KeyListPanel.vue'
import zhCN from '@/i18n/locales/zh-CN'
import { request } from '@/utils/request'

vi.mock('@/utils/request', () => ({
  request: vi.fn().mockResolvedValue({ code: 200, data: null, msg: 'OK' }),
}))
vi.mock('@/utils/dialog', () => ({
  gmConfirm: vi.fn().mockResolvedValue(true),
  gmPrompt: vi.fn().mockResolvedValue(''),
}))
vi.mock('@/components/views/NewKeyDialog.vue', () => ({ default: { name: 'NewKeyDialog', template: '<div />' } }))
vi.mock('@/components/views/GroupDeleteDialog.vue', () => ({ default: { name: 'GroupDeleteDialog', template: '<div />' } }))
vi.mock('@/components/views/ExportDialog.vue', () => ({ default: { name: 'ExportDialog', template: '<div />' } }))
vi.mock('@/components/views/ImportDialog.vue', () => ({ default: { name: 'ImportDialog', template: '<div />' } }))
vi.mock('@/components/views/BatchDialog.vue', () => ({ default: { name: 'BatchDialog', template: '<div />' } }))
vi.mock('@/components/views/MigrateDialog.vue', () => ({ default: { name: 'MigrateDialog', template: '<div />' } }))
vi.mock('@/components/views/CrossDbSearchDialog.vue', () => ({ default: { name: 'CrossDbSearchDialog', template: '<div />' } }))
vi.mock('@/components/views/KeyBackupDialog.vue', () => ({ default: { name: 'KeyBackupDialog', template: '<div />' } }))

const mockedRequest = vi.mocked(request)

const TEST_KEYS = [
  { key: 'user:1', type: 'string', ttl: -1 },
  { key: 'user:2', type: 'string', ttl: -1 },
  { key: 'order:1', type: 'hash', ttl: 3600 },
]

function setupMock(favorites: string[], cursor = 0) {
  mockedRequest.mockImplementation((method: string) => {
    if (method === 'get_favorites') {
      return Promise.resolve({ code: 200, data: favorites, msg: 'OK' } as any)
    }
    if (method === 'scan_keys') {
      return Promise.resolve({ code: 200, data: { keys: TEST_KEYS, cursor }, msg: 'OK' } as any)
    }
    if (method === 'db_list') {
      return Promise.resolve({ code: 200, data: [{ db: 0, keys: 3 }], msg: 'OK' } as any)
    }
    return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
  })
}

function createWrapper() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const i18n = createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': zhCN } })
  const store = useConnectionStore()
  store.connections = [
    { id: 'tc', name: 'Test', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 } as any,
  ]
  store.setConnState('tc', { status: 'connected', currentDb: 0, cliDb: 0 })
  const wrapper = mount(KeyListPanel, {
    props: { connId: 'tc' },
    global: {
      plugins: [pinia, i18n],
      directives: { 'ellipsis-tip': { mounted() {} }, 'body-tooltip': { mounted() {} } },
    },
  })
  return { wrapper, store }
}

function getState(wrapper: any) {
  return toRaw(wrapper.vm.$.setupState)
}

describe('KeyListPanel - 纯前端树构建', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('切换到树形视图应从本地 keys 构建分组（默认折叠）', async () => {
    setupMock([])
    const { wrapper, store } = createWrapper()
    await flushPromises()
    await nextTick()

    store.setViewMode('tc', 'tree')
    await nextTick()

    const state = getState(wrapper)
    const treeNodes = state.treeNodes.value
    // 默认折叠：只有分组节点可见，无叶子
    const groupNodes = treeNodes.filter((n: any) => n.isGroup)
    const leafNodes = treeNodes.filter((n: any) => !n.isGroup)
    expect(groupNodes).toHaveLength(2)
    expect(groupNodes.map((n: any) => n.label).sort()).toEqual(['order', 'user'])
    expect(leafNodes).toHaveLength(0)
  })

  it('不调用 scan_tree_level，树完全从本地构建', async () => {
    setupMock([])
    const { store } = createWrapper()
    await flushPromises()
    await nextTick()

    store.setViewMode('tc', 'tree')
    await flushPromises()
    await nextTick()

    const treeCalls = mockedRequest.mock.calls.filter(c => c[0] === 'scan_tree_level')
    expect(treeCalls).toHaveLength(0)
  })

  it('展开分组应显示叶子 key，无后端请求', async () => {
    setupMock([])
    const { wrapper, store } = createWrapper()
    await flushPromises()
    await nextTick()

    store.setViewMode('tc', 'tree')
    await nextTick()

    const state = getState(wrapper)
    // 默认折叠，展开 user: 分组
    state.toggleTreeNode('user:')
    await nextTick()

    const treeNodes = state.treeNodes.value
    const userLeaves = treeNodes.filter((n: any) => !n.isGroup && n.fullKey?.startsWith('user:'))
    expect(userLeaves).toHaveLength(2)

    // 确认没有 scan_tree_level 调用
    const treeCalls = mockedRequest.mock.calls.filter(c => c[0] === 'scan_tree_level')
    expect(treeCalls).toHaveLength(0)
  })

  it('showFavoritesOnly 在树形视图下应过滤非收藏 key', async () => {
    setupMock(['user:1'])
    const { wrapper, store } = createWrapper()
    await flushPromises()
    await nextTick()

    store.setViewMode('tc', 'tree')
    await nextTick()

    const state = getState(wrapper)

    // 展开 user: 分组并开启收藏筛选
    state.toggleTreeNode('user:')
    state.showFavoritesOnly.value = true
    await nextTick()

    const treeNodes = state.treeNodes.value
    const leafNodes = treeNodes.filter((n: any) => !n.isGroup && n.fullKey)
    // 只应显示 user:1，不显示 user:2 和 order:1
    expect(leafNodes).toHaveLength(1)
    expect(leafNodes[0].fullKey).toBe('user:1')
  })

  it('loadMore 应追加新 key 并更新树', async () => {
    let callCount = 0
    mockedRequest.mockImplementation((method: string) => {
      if (method === 'get_favorites') {
        return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
      }
      if (method === 'scan_keys') {
        callCount++
        if (callCount <= 1) {
          return Promise.resolve({ code: 200, data: { keys: [TEST_KEYS[0]], cursor: 5 }, msg: 'OK' } as any)
        }
        return Promise.resolve({ code: 200, data: { keys: [TEST_KEYS[1], TEST_KEYS[2]], cursor: 0 }, msg: 'OK' } as any)
      }
      return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
    })

    const { wrapper, store } = createWrapper()
    await flushPromises()
    await nextTick()

    const state = getState(wrapper)
    expect(state.keys.value).toHaveLength(1)
    expect(state.hasMore.value).toBe(true)

    // 加载更多
    await state.loadMore()
    await flushPromises()

    expect(state.keys.value).toHaveLength(3)
    expect(state.hasMore.value).toBe(false)

    // 树也应更新（分组默认折叠）
    store.setViewMode('tc', 'tree')
    await nextTick()
    const groups = state.treeNodes.value.filter((n: any) => n.isGroup)
    expect(groups).toHaveLength(2)
  })

  it('勾选折叠的分组应选中所有已加载的子 key', async () => {
    setupMock([])
    const { wrapper, store } = createWrapper()
    await flushPromises()
    await nextTick()

    store.setViewMode('tc', 'tree')
    await nextTick()

    const state = getState(wrapper)
    state.checkMode.value = true
    await nextTick()

    // 分组默认折叠，直接勾选 user: 分组
    state.toggleGroupCheck('user:')
    await nextTick()

    // 应选中 user:1 和 user:2
    expect(state.multiSelected.value.has('user:1')).toBe(true)
    expect(state.multiSelected.value.has('user:2')).toBe(true)
    expect(state.multiSelected.value.has('order:1')).toBe(false)
  })

  it('刷新后已展开的分组保持展开，消失的分组被清理', async () => {
    setupMock([])
    const { wrapper, store } = createWrapper()
    await flushPromises()
    await nextTick()

    store.setViewMode('tc', 'tree')
    await nextTick()

    const state = getState(wrapper)
    // 展开 user: 分组
    state.toggleTreeNode('user:')
    await nextTick()

    // 确认 user: 已展开（不在 collapsedNodes 中）
    expect(state.collapsedNodes.value.has('user:')).toBe(false)
    expect(state.collapsedNodes.value.has('order:')).toBe(true)

    // 刷新
    await state.refreshKeys()
    await flushPromises()
    await nextTick()

    // user: 应保持展开，order: 仍折叠
    expect(state.collapsedNodes.value.has('user:')).toBe(false)
    expect(state.collapsedNodes.value.has('order:')).toBe(true)
  })

  it('onKeyCreated 直接插入 key 而不全量刷新', async () => {
    setupMock([])
    const { wrapper, store } = createWrapper()
    await flushPromises()
    await nextTick()

    const state = getState(wrapper)
    const scanCallsBefore = mockedRequest.mock.calls.filter(c => c[0] === 'scan_keys').length

    // 新建一个 key
    state.onKeyCreated('product:1', 'string')
    await nextTick()

    // 不应触发新的 scan_keys 请求
    const scanCallsAfter = mockedRequest.mock.calls.filter(c => c[0] === 'scan_keys').length
    expect(scanCallsAfter).toBe(scanCallsBefore)

    // key 应已插入到列表
    expect(state.keys.value.find((k: any) => k.key === 'product:1')).toBeTruthy()

    // 树形视图下新 key 的分组应自动展开
    store.setViewMode('tc', 'tree')
    await nextTick()
    const groups = state.treeNodes.value.filter((n: any) => n.isGroup)
    expect(groups.map((n: any) => n.label)).toContain('product')
    // product: 分组应展开（不在 collapsedNodes 中）
    expect(state.collapsedNodes.value.has('product:')).toBe(false)
  })
})
