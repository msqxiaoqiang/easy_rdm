import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useConnectionStore } from '@/stores/connection'
import { idToGroupKey } from '@/utils/sidebar-tree'

// Mock request module
vi.mock('@/utils/request', () => ({
  request: vi.fn(),
}))

import { request } from '@/utils/request'
const mockRequest = vi.mocked(request)

describe('useConnectionStore', () => {
  let store: ReturnType<typeof useConnectionStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    store = useConnectionStore()
    vi.clearAllMocks()
  })

  // ========== 连接配置 CRUD ==========

  describe('loadConnections', () => {
    it('应加载连接列表', async () => {
      const conns = [
        { id: '1', name: 'Redis-1', host: '127.0.0.1', port: 6379 },
        { id: '2', name: 'Redis-2', host: '10.0.0.1', port: 6380, group: 'prod' },
      ]
      mockRequest
        .mockResolvedValueOnce({ code: 200, data: conns, msg: 'OK' })
        .mockResolvedValueOnce({ code: 200, data: [], msg: 'OK' })
        .mockResolvedValueOnce({ code: 200, data: {}, msg: 'OK' })

      await store.loadConnections()

      expect(store.connections).toHaveLength(2)
      expect(store.connections[0].name).toBe('Redis-1')
      expect(store.groups).toContain(idToGroupKey('prod'))
    })

    it('空列表时 connections 为空数组', async () => {
      mockRequest
        .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
        .mockResolvedValueOnce({ code: 200, data: [], msg: 'OK' })
        .mockResolvedValueOnce({ code: 200, data: {}, msg: 'OK' })

      await store.loadConnections()

      expect(store.connections).toHaveLength(0)
    })

    it('应合并连接中的分组和 groups.json 中的分组', async () => {
      const conns = [
        { id: '1', name: 'Redis-1', host: '127.0.0.1', port: 6379, group: 'prod' },
      ]
      mockRequest
        .mockResolvedValueOnce({ code: 200, data: conns, msg: 'OK' })     // get_connections
        .mockResolvedValueOnce({ code: 200, data: [idToGroupKey('staging')], msg: 'OK' }) // get_groups (displayOrder 格式)
        .mockResolvedValueOnce({ code: 200, data: {}, msg: 'OK' })        // get_group_meta

      await store.loadConnections()

      expect(store.groups).toContain(idToGroupKey('prod'))
      expect(store.groups).toContain(idToGroupKey('staging'))
    })

    it('get_groups 返回 null 时不报错', async () => {
      const conns = [
        { id: '1', name: 'Redis-1', host: '127.0.0.1', port: 6379, group: 'prod' },
      ]
      mockRequest
        .mockResolvedValueOnce({ code: 200, data: conns, msg: 'OK' })     // get_connections
        .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })       // get_groups
        .mockResolvedValueOnce({ code: 200, data: {}, msg: 'OK' })        // get_group_meta

      await store.loadConnections()

      expect(store.groups).toContain(idToGroupKey('prod'))
    })
  })

  describe('saveConnection', () => {
    it('应保存连接并刷新列表', async () => {
      mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })

      const conn = {
        id: 'new-1', name: 'New', host: '127.0.0.1', port: 6379,
        db: 0, conn_type: 'tcp', conn_timeout: 60, exec_timeout: 60,
      }
      await store.saveConnection(conn)

      expect(mockRequest).toHaveBeenCalledWith('save_connection', { params: conn })
    })
  })

  describe('deleteConnection', () => {
    it('应删除连接并关闭对应 Tab', async () => {
      mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })

      // 先添加一个 Tab
      store.tabs = [{ id: 'del-1', name: 'Test', host: '127.0.0.1', port: 6379, pinned: false }]
      store.activeTabId = 'del-1'

      await store.deleteConnection('del-1')

      expect(store.tabs).toHaveLength(0)
      expect(mockRequest).toHaveBeenCalledWith('delete_connection', { params: { id: 'del-1' } })
    })
  })

  describe('copyConnection', () => {
    it('应复制连接并追加 (copy) 后缀', async () => {
      mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })

      const original = {
        id: 'orig', name: 'Original', host: '127.0.0.1', port: 6379,
        db: 0, conn_type: 'tcp', conn_timeout: 60, exec_timeout: 60,
      }
      await store.copyConnection(original)

      const savedCall = mockRequest.mock.calls.find(c => c[0] === 'save_connection')
      expect(savedCall).toBeTruthy()
      const savedConn = (savedCall as any)[1].params
      expect(savedConn.name).toBe('Original (copy)')
      expect(savedConn.id).not.toBe('orig')
    })
  })

  // ========== Tab 管理 ==========

  describe('addTab', () => {
    it('应添加新 Tab 并激活', () => {
      const conn = {
        id: 't1', name: 'Tab1', host: '127.0.0.1', port: 6379,
        db: 0, conn_type: 'tcp', conn_timeout: 60, exec_timeout: 60,
      }
      store.addTab(conn)

      expect(store.tabs).toHaveLength(1)
      expect(store.activeTabId).toBe('t1')
    })

    it('重复添加应切换到已有 Tab', () => {
      const conn = {
        id: 't1', name: 'Tab1', host: '127.0.0.1', port: 6379,
        db: 0, conn_type: 'tcp', conn_timeout: 60, exec_timeout: 60,
      }
      store.addTab(conn)
      store.addTab(conn)

      expect(store.tabs).toHaveLength(1)
    })

    it('超过 20 个 Tab 应抛出错误', () => {
      for (let i = 0; i < 20; i++) {
        store.tabs.push({ id: `t${i}`, name: `Tab${i}`, host: '127.0.0.1', port: 6379, pinned: false })
      }

      const conn = {
        id: 'overflow', name: 'Overflow', host: '127.0.0.1', port: 6379,
        db: 0, conn_type: 'tcp', conn_timeout: 60, exec_timeout: 60,
      }
      expect(() => store.addTab(conn)).toThrow('已达到最大标签页数')
    })
  })

  describe('closeTab', () => {
    beforeEach(() => {
      store.tabs = [
        { id: 't1', name: 'Tab1', host: '127.0.0.1', port: 6379, pinned: false },
        { id: 't2', name: 'Tab2', host: '127.0.0.1', port: 6380, pinned: false },
        { id: 't3', name: 'Tab3', host: '127.0.0.1', port: 6381, pinned: false },
      ]
      store.activeTabId = 't2'
    })

    it('应关闭指定 Tab', () => {
      store.closeTab('t2')
      expect(store.tabs).toHaveLength(2)
      expect(store.tabs.find(t => t.id === 't2')).toBeUndefined()
    })

    it('关闭当前激活 Tab 应切换到相邻 Tab', () => {
      store.closeTab('t2')
      expect(store.activeTabId).toBe('t3')
    })

    it('不应关闭固定的 Tab', () => {
      store.tabs[1].pinned = true
      store.closeTab('t2')
      expect(store.tabs).toHaveLength(3)
    })

    it('关闭不存在的 Tab 不报错', () => {
      store.closeTab('nonexistent')
      expect(store.tabs).toHaveLength(3)
    })
  })

  describe('pinTab', () => {
    it('应切换固定状态', () => {
      store.tabs = [
        { id: 't1', name: 'Tab1', host: '127.0.0.1', port: 6379, pinned: false },
        { id: 't2', name: 'Tab2', host: '127.0.0.1', port: 6380, pinned: false },
      ]

      store.pinTab('t2')
      expect(store.tabs[0].id).toBe('t2') // 固定的靠左
      expect(store.tabs[0].pinned).toBe(true)
    })
  })

  // ========== 分组操作 ==========

  describe('deleteGroup', () => {
    it('应逐个删除组内连接并从 groups 移除', async () => {
      store.connections = [
        { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' } as any,
        { id: 'c2', name: 'R2', host: '127.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' } as any,
        { id: 'c3', name: 'R3', host: '127.0.0.1', port: 6381, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 } as any,
      ]
      store.groups = [idToGroupKey('prod')]
      store.tabs = [
        { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, pinned: false },
      ]
      store.activeTabId = 'c1'

      // delete_connection 成功，saveGroups (save_groups + save_group_meta)，loadConnections (get_connections + get_groups + get_group_meta)
      mockRequest.mockImplementation(async (method: string) => {
        if (method === 'delete_connection') return { code: 200, data: null, msg: 'OK' }
        if (method === 'get_connections') return { code: 200, data: [store.connections[2]], msg: 'OK' }
        if (method === 'save_groups') return { code: 200, data: null, msg: 'OK' }
        if (method === 'save_group_meta') return { code: 200, data: null, msg: 'OK' }
        if (method === 'get_groups') return { code: 200, data: null, msg: 'OK' }
        if (method === 'get_group_meta') return { code: 200, data: {}, msg: 'OK' }
        return { code: 200, data: null, msg: 'OK' }
      })

      await store.deleteGroup('prod')

      // 应调用 delete_connection 两次（c1, c2）
      const deleteCalls = mockRequest.mock.calls.filter(c => c[0] === 'delete_connection')
      expect(deleteCalls).toHaveLength(2)
      expect(deleteCalls[0][1]).toEqual({ params: { id: 'c1' } })
      expect(deleteCalls[1][1]).toEqual({ params: { id: 'c2' } })

      // groups 中不应再有 prod
      expect(store.groups).not.toContain(idToGroupKey('prod'))
    })

    it('空分组删除不应调用 delete_connection', async () => {
      store.connections = []
      store.groups = [idToGroupKey('empty-group')]
      mockRequest.mockImplementation(async (method: string) => {
        if (method === 'get_connections') return { code: 200, data: [], msg: 'OK' }
        if (method === 'save_groups') return { code: 200, data: null, msg: 'OK' }
        if (method === 'save_group_meta') return { code: 200, data: null, msg: 'OK' }
        if (method === 'get_groups') return { code: 200, data: [], msg: 'OK' }
        if (method === 'get_group_meta') return { code: 200, data: {}, msg: 'OK' }
        return { code: 200, data: null, msg: 'OK' }
      })

      await store.deleteGroup('empty-group')

      const deleteCalls = mockRequest.mock.calls.filter(c => c[0] === 'delete_connection')
      expect(deleteCalls).toHaveLength(0)
      expect(store.groups).not.toContain(idToGroupKey('empty-group'))
    })
  })

  // ========== 连接状态 ==========

  describe('connState', () => {
    it('默认状态为 disconnected', () => {
      const state = store.getConnState('unknown')
      expect(state.status).toBe('disconnected')
      expect(state.currentDb).toBe(0)
    })

    it('setConnState 应更新状态', () => {
      store.setConnState('c1', { status: 'connected', currentDb: 3, cliDb: 0, redisVersion: '7.0.0' })
      const state = store.getConnState('c1')
      expect(state.status).toBe('connected')
      expect(state.currentDb).toBe(3)
      expect(state.redisVersion).toBe('7.0.0')
    })
  })

  // ========== 辅助函数 ==========

  describe('generateId', () => {
    it('应生成唯一 ID', () => {
      const id1 = store.generateId()
      const id2 = store.generateId()
      expect(id1).not.toBe(id2)
      expect(id1.length).toBeGreaterThan(5)
    })
  })

  describe('parseServerInfo', () => {
    it('应解析 INFO 输出', () => {
      const raw = 'redis_version:7.0.0\nredis_mode:standalone\nrole:master\n'
      const result = store.parseServerInfo(raw)
      expect(result.redis_version).toBe('7.0.0')
      expect(result.redis_mode).toBe('standalone')
      expect(result.role).toBe('master')
    })

    it('空字符串返回空对象', () => {
      const result = store.parseServerInfo('')
      expect(Object.keys(result)).toHaveLength(0)
    })
  })

  // ========== 会话 ==========

  describe('session', () => {
    it('saveSession 应调用 request', async () => {
      mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })
      store.tabs = [{ id: 't1', name: 'Tab1', host: '127.0.0.1', port: 6379, pinned: false }]
      store.activeTabId = 't1'

      await store.saveSession()

      expect(mockRequest).toHaveBeenCalledWith('save_session', {
        params: expect.objectContaining({ tabs: store.tabs, activeTabId: 't1' }),
      })
    })

    it('restoreSession 非 Web 模式不恢复 Tab（GM/Desktop 关闭即重置）', async () => {
      const savedTabs = [{ id: 's1', name: 'Saved', host: '127.0.0.1', port: 6379, pinned: false }]
      mockRequest.mockResolvedValueOnce({
        code: 200,
        data: { tabs: savedTabs, activeTabId: 's1' },
        msg: 'OK',
      })

      await store.restoreSession()

      // 默认测试环境为 gmssh 模式，不恢复 tabs
      expect(store.tabs).toHaveLength(0)
      expect(store.activeTabId).toBe('')
    })

    it('restoreSession 无数据时不报错', async () => {
      mockRequest.mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
      await store.restoreSession()
      expect(store.tabs).toHaveLength(0)
    })
  })

  // ========== db 持久化 ==========

  describe('db 持久化 - dbPreferences', () => {
    it('setCurrentDb 应同步更新 dbPreferences', () => {
      store.setCurrentDb('c1', 5)
      expect(store.dbPreferences['c1']).toBe(5)
    })

    it('connect 应优先使用 dbPreferences 中的 db', async () => {
      store.connections = [
        {
          id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379,
          db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10,
        } as any,
      ]
      store.dbPreferences = { c1: 3 }
      mockRequest.mockResolvedValueOnce({ code: 200, data: { info: '' }, msg: 'OK' })

      await store.connect('c1')

      expect(mockRequest).toHaveBeenCalledWith('connect', { params: { id: 'c1', db: 3 } })
    })

    it('saveSession 应包含 dbPreferences', async () => {
      store.dbPreferences = { c1: 5 }
      mockRequest.mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })

      await store.saveSession()

      expect(mockRequest).toHaveBeenCalledWith('save_session', expect.objectContaining({
        params: expect.objectContaining({ dbPreferences: { c1: 5 } }),
      }))
    })

    it('restoreSession 应恢复 dbPreferences', async () => {
      mockRequest.mockResolvedValueOnce({
        code: 200,
        data: { tabs: [], connectedIds: [], dbPreferences: { c1: 7 } },
        msg: 'OK',
      })

      await store.restoreSession()

      expect(store.dbPreferences).toEqual({ c1: 7 })
    })
  })

  // ========== 拖拽排序 ==========

  describe('reorderConnections', () => {
    it('应本地更新连接顺序和分组，并调用后端持久化', async () => {
      store.connections = [
        { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: '' } as any,
        { id: 'c2', name: 'R2', host: '127.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: '' } as any,
      ]
      mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })

      const items = [
        { id: 'c2', group: 'prod' },
        { id: 'c1', group: '' },
      ]
      await store.reorderConnections(items)

      // 本地数据应立即更新
      expect(store.connections[0].id).toBe('c2')
      expect(store.connections[0].group).toBe('prod')
      expect(store.connections[1].id).toBe('c1')
      // 应调用后端持久化
      expect(mockRequest).toHaveBeenCalledWith('reorder_connections', { params: { items } })
      // 不应调用 loadConnections（不再 reload）
      expect(mockRequest).not.toHaveBeenCalledWith('get_connections', expect.anything())
    })

    it('items 中缺失的连接应追加到末尾', async () => {
      store.connections = [
        { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 } as any,
        { id: 'c2', name: 'R2', host: '127.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 } as any,
        { id: 'c3', name: 'R3', host: '127.0.0.1', port: 6381, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 } as any,
      ]
      mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })

      await store.reorderConnections([{ id: 'c3', group: 'g1' }])

      expect(store.connections[0].id).toBe('c3')
      expect(store.connections[1].id).toBe('c1')
      expect(store.connections[2].id).toBe('c2')
    })

    it('空 items 也应正常调用', async () => {
      store.connections = [
        { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 } as any,
      ]
      mockRequest.mockResolvedValue({ code: 200, data: null, msg: 'OK' })

      await store.reorderConnections([])

      expect(mockRequest).toHaveBeenCalledWith('reorder_connections', { params: { items: [] } })
      // 原始连接仍在
      expect(store.connections).toHaveLength(1)
    })
  })

  describe('saveGroups', () => {
    it('应调用 save_groups 和 save_group_meta 接口', async () => {
      store.groups = ['prod', 'staging']
      store.groupMeta = { grp1: 'Production' }
      mockRequest
        .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
        .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })

      await store.saveGroups()

      expect(mockRequest).toHaveBeenCalledWith('save_groups', { params: { groups: ['prod', 'staging'] } })
      expect(mockRequest).toHaveBeenCalledWith('save_group_meta', { params: { group_meta: { grp1: 'Production' } } })
    })

    it('空分组列表也应正常保存', async () => {
      store.groups = []
      store.groupMeta = {}
      mockRequest
        .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })
        .mockResolvedValueOnce({ code: 200, data: null, msg: 'OK' })

      await store.saveGroups()

      expect(mockRequest).toHaveBeenCalledWith('save_groups', { params: { groups: [] } })
      expect(mockRequest).toHaveBeenCalledWith('save_group_meta', { params: { group_meta: {} } })
    })
  })
})
