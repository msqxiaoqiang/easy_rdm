import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { request } from '../utils/request'
import { isWeb } from '../utils/platform'
import i18n from '../i18n'
import { isGroupKey, idToGroupKey } from '../utils/sidebar-tree'

/** 连接配置 */
export interface Connection {
  id: string
  name: string
  host: string
  port: number
  password?: string
  username?: string
  db: number
  conn_type: string // tcp | unix
  unix_socket?: string
  conn_timeout: number
  exec_timeout: number
  group?: string
  // 高级配置
  key_filter?: string
  key_separator?: string
  default_view?: string // tree | flat
  scan_count?: number
  db_filter_mode?: string // all | include | exclude
  db_filter_list?: number[]
  // TLS/SSL
  use_tls?: boolean
  tls_cert_file?: string
  tls_key_file?: string
  tls_ca_file?: string
  tls_skip_verify?: boolean
  // SSH 隧道
  use_ssh?: boolean
  ssh_host?: string
  ssh_port?: number
  ssh_username?: string
  ssh_password?: string
  ssh_private_key?: string
  ssh_passphrase?: string
  // 网络代理
  use_proxy?: boolean
  proxy_type?: string
  proxy_host?: string
  proxy_port?: number
  proxy_username?: string
  proxy_password?: string
  // 哨兵模式
  use_sentinel?: boolean
  sentinel_addrs?: string
  sentinel_master_name?: string
  sentinel_password?: string
  sentinel_password_encrypted?: boolean
  // 集群模式
  use_cluster?: boolean
  cluster_addrs?: string
  // 状态（前端维护）
  password_encrypted?: boolean
  ssh_password_encrypted?: boolean
}

/** 打开的 Tab */
export interface ConnectionTab {
  id: string       // connection id
  name: string
  host: string
  port: number
  pinned: boolean
}

/** 连接运行时状态 */
export interface ConnState {
  status: 'disconnected' | 'connecting' | 'connected' | 'error'
  redisVersion?: string
  redisMode?: string
  currentDb: number
  cliDb: number
  error?: string
}

export const useConnectionStore = defineStore('connection', () => {
  // 所有保存的连接配置
  const connections = ref<Connection[]>([])
  // 显示顺序（分组用 __group__<groupId>，未分组连接用连接 ID）
  const groups = ref<string[]>([])
  // 分组元数据（groupId → 显示名称）
  const groupMeta = ref<Record<string, string>>({})
  // 打开的 Tab 列表
  const tabs = ref<ConnectionTab[]>([])
  // 当前激活的 Tab ID
  const activeTabId = ref<string>('')
  // 各连接的运行时状态
  const connStates = ref<Record<string, ConnState>>({})
  // 各连接的 Key 列表视图模式
  const viewModes = ref<Record<string, 'flat' | 'tree'>>({})
  // 每个连接最后一次切换的 db，用于下次连接时恢复
  const dbPreferences = ref<Record<string, number>>({})
  // CLI 独立的 db 偏好（与键列表 db 互不影响）
  const cliDbPreferences = ref<Record<string, number>>({})
  // CLI 历史缓存（纯 JS Map，不走响应式，断开连接时清空）
  const cliCaches = new Map<string, { history: any[]; cmdHistory: string[] }>()
  // 图表数据缓存（纯 JS Map，断开连接时清空）
  const statsCaches = new Map<string, any[]>()
  // PubSub 会话缓存（切 tab 保留，断开时清空）
  const pubsubCaches = new Map<string, { messages: any[]; subscribed: boolean; channelInput: string; usePattern: boolean }>()
  // Monitor 会话缓存（切 tab 保留，断开时清空）
  const monitorCaches = new Map<string, { commands: any[]; running: boolean; filter: string }>()
  // Key 详情缓存（切 tab 保留，断开时清空）—— connId → keyName → 任意组件状态
  const keyDetailCaches = new Map<string, Map<string, any>>()
  // 每个连接最后选中的 key（切 tab 保留，断开时清空）
  const selectedKeyCaches = new Map<string, { key: string; type: string }>()
  // Key 列表状态缓存（切连接保留，断开时清空）
  const keyListStateCaches = new Map<string, any>()
  // 每个连接最后选中的子 Tab（切连接保留，断开时清空）
  const subTabCaches = new Map<string, string>()
  const activeTab = computed(() => tabs.value.find(t => t.id === activeTabId.value))
  const activeConnState = computed(() => connStates.value[activeTabId.value])

  // ========== 连接配置 CRUD ==========

  async function loadConnections() {
    const res = await request<Connection[]>('get_connections', { params: {} })
    connections.value = res.data || []

    // 加载显示顺序和分组元数据
    const savedRes = await request<string[]>('get_groups', { params: {} })
    const saved = savedRes.data || []
    const metaRes = await request<Record<string, string>>('get_group_meta', { params: {} })
    groupMeta.value = metaRes.data || {}

    // 从连接推导出需要出现的 key
    const neededKeys = new Set<string>()
    for (const c of connections.value) {
      if (c.group) {
        neededKeys.add(idToGroupKey(c.group))
      } else {
        neededKeys.add(c.id)
      }
    }

    // 已知连接 ID 集合（用于过滤无效条目）
    const connIds = new Set(connections.value.map(c => c.id))

    // 以 saved 为基础，追加新出现的 key（过滤无效条目）
    const result: string[] = []
    const seen = new Set<string>()
    for (const key of saved) {
      if (seen.has(key)) continue
      if (!isGroupKey(key) && !connIds.has(key)) continue
      result.push(key)
      seen.add(key)
    }
    for (const key of neededKeys) {
      if (!seen.has(key)) {
        result.push(key)
        seen.add(key)
      }
    }
    groups.value = result
  }

  async function saveGroups() {
    await request('save_groups', { params: { groups: groups.value } })
    await request('save_group_meta', { params: { group_meta: groupMeta.value } })
  }

  async function saveConnection(conn: Connection) {
    await request('save_connection', { params: conn })
    await loadConnections()
    await saveGroups()
  }

  async function deleteConnection(id: string) {
    await request('delete_connection', { params: { id } })
    // 关闭对应 Tab
    closeTab(id)
    await loadConnections()
  }

  async function copyConnection(conn: Connection) {
    const newConn: Connection = {
      ...conn,
      id: generateId(),
      name: conn.name + ' (copy)',
      password_encrypted: false,
    }
    // 复制时密码不带过来（加密的无法复制）
    delete newConn.password
    await request('save_connection', { params: newConn })
    await loadConnections()
  }

  // ========== 连接操作 ==========

  async function connect(id: string, dbOverride?: number) {
    const conn = connections.value.find(c => c.id === id)
    if (!conn) return

    const targetDb = dbOverride ?? dbPreferences.value[id] ?? conn.db ?? 0
    const targetCliDb = cliDbPreferences.value[id] ?? targetDb
    setConnState(id, { status: 'connecting', currentDb: targetDb, cliDb: targetCliDb })

    try {
      const res = await request<{ info: string }>('connect', { params: { id, db: targetDb } })
      const info = parseServerInfo(res.data?.info || '')
      setConnState(id, {
        status: 'connected',
        currentDb: targetDb,
        cliDb: targetCliDb,
        redisVersion: info.redis_version,
        redisMode: info.redis_mode,
      })
      // 添加 Tab
      addTab(conn)
    } catch (e: any) {
      setConnState(id, { status: 'error', currentDb: targetDb, cliDb: targetCliDb, error: e.message })
      throw e
    }
  }

  async function disconnect(id: string) {
    await request('disconnect', { params: { id } })
    clearCliCache(id)
    clearStatsCache(id)
    clearPubsubCache(id)
    clearMonitorCache(id)
    clearKeyDetailCache(id)
    clearSelectedKeyCache(id)
    clearSubTabCache(id)
    clearKeyListStateCache(id)
    setConnState(id, { status: 'disconnected', currentDb: 0, cliDb: 0 })
  }

  async function testConnection(conn: Partial<Connection>) {
    return await request('test_connection', { params: conn })
  }

  // ========== Tab 管理 ==========

  function addTab(conn: Connection) {
    if (tabs.value.find(t => t.id === conn.id)) {
      activeTabId.value = conn.id
      return
    }
    if (tabs.value.length >= 20) {
      throw new Error(i18n.global.t('connection.maxTabs'))
    }
    tabs.value.push({
      id: conn.id,
      name: conn.name,
      host: conn.host,
      port: conn.port,
      pinned: false,
    })
    activeTabId.value = conn.id
  }

  function closeTab(id: string) {
    const idx = tabs.value.findIndex(t => t.id === id)
    if (idx === -1) return
    if (tabs.value[idx].pinned) return
    tabs.value.splice(idx, 1)
    if (activeTabId.value === id) {
      activeTabId.value = tabs.value[Math.min(idx, tabs.value.length - 1)]?.id || ''
    }
  }

  function pinTab(id: string) {
    const tab = tabs.value.find(t => t.id === id)
    if (tab) tab.pinned = !tab.pinned
    // 固定的 Tab 靠左排列
    tabs.value.sort((a, b) => (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0))
  }

  function setActiveTab(id: string) {
    activeTabId.value = id
  }

  async function deleteGroup(groupId: string) {
    const affected = connections.value.filter(c => c.group === groupId)
    // 逐个删除组内连接
    for (const c of affected) {
      await request('delete_connection', { params: { id: c.id } })
      closeTab(c.id)
    }
    // 从 displayOrder 中移除分组 key
    const groupKey = idToGroupKey(groupId)
    groups.value = groups.value.filter(k => k !== groupKey)
    // 从 meta 中移除
    delete groupMeta.value[groupId]
    saveGroups()
    await loadConnections()
  }

  function generateGroupId(): string {
    return 'grp_' + Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
  }

  // ========== 视图模式 ==========

  function getViewMode(connId: string): 'flat' | 'tree' {
    return viewModes.value[connId] || 'flat'
  }

  function setViewMode(connId: string, mode: 'flat' | 'tree') {
    viewModes.value[connId] = mode
  }

  // ========== 会话保存/恢复 ==========

  async function saveSession() {
    try {
      const sessionData: Record<string, any> = {
        tabs: tabs.value,
        activeTabId: activeTabId.value,
        viewModes: viewModes.value,
        dbPreferences: dbPreferences.value,
        cliDbPreferences: cliDbPreferences.value,
      }
      // 仅 Web 模式保存连接状态（GM/Desktop 关闭后需要重新连接）
      if (isWeb()) {
        const connectedIds = Object.entries(connStates.value)
          .filter(([_, s]) => s.status === 'connected')
          .map(([id]) => id)
        const dbStates: Record<string, number> = {}
        for (const [id, s] of Object.entries(connStates.value)) {
          if (s.status === 'connected') {
            dbStates[id] = s.currentDb
          }
        }
        sessionData.connectedIds = connectedIds
        sessionData.dbStates = dbStates
      }
      // Web 模式持久化子Tab偏好（Desktop/GMSSH 关闭即重置，不需要持久化）
      if (isWeb()) {
        const subTabPrefs: Record<string, string> = {}
        subTabCaches.forEach((v, k) => { subTabPrefs[k] = v })
        sessionData.subTabPreferences = subTabPrefs
      }
      await request('save_session', { params: sessionData })
    } catch (_e) { /* ignore */ }
  }

  async function restoreSession() {
    try {
      const res = await request<any>('get_session')
      if (!res.data) return
      const savedTabs = res.data.tabs as ConnectionTab[] | undefined
      const savedActive = res.data.activeTabId as string | undefined
      const savedConnectedIds = res.data.connectedIds as string[] | undefined
      const savedViewModes = res.data.viewModes as Record<string, 'flat' | 'tree'> | undefined
      const savedDbStates = res.data.dbStates as Record<string, number> | undefined
      // Web 模式恢复 tabs 和选中状态；GM/Desktop 不恢复（关闭即重置）
      if (isWeb() && savedTabs?.length) {
        tabs.value = savedTabs
        activeTabId.value = savedActive || savedTabs[0]?.id || ''
      }
      if (savedViewModes) {
        viewModes.value = savedViewModes
      }
      const savedDbPreferences = res.data.dbPreferences as Record<string, number> | undefined
      if (savedDbPreferences) {
        dbPreferences.value = savedDbPreferences
      }
      const savedCliDbPreferences = res.data.cliDbPreferences as Record<string, number> | undefined
      if (savedCliDbPreferences) {
        cliDbPreferences.value = savedCliDbPreferences
      }
      // Web 模式恢复子Tab偏好
      if (isWeb()) {
        const savedSubTabPrefs = res.data.subTabPreferences as Record<string, string> | undefined
        if (savedSubTabPrefs) {
          for (const [k, v] of Object.entries(savedSubTabPrefs)) {
            subTabCaches.set(k, v)
          }
        }
      }
      // 仅 Web 模式自动重连（GM/Desktop 关闭后需要重新连接）
      if (isWeb() && savedConnectedIds?.length) {
        for (const id of savedConnectedIds) {
          const conn = connections.value.find(c => c.id === id)
          if (conn) {
            const targetDb = savedDbStates?.[id] ?? conn.db ?? 0
            const targetCliDb = cliDbPreferences.value[id] ?? targetDb
            setConnState(id, { status: 'connecting', currentDb: targetDb, cliDb: targetCliDb })
          }
        }
        for (const id of savedConnectedIds) {
          const conn = connections.value.find(c => c.id === id)
          const targetDb = savedDbStates?.[id] ?? conn?.db ?? 0
          connect(id, targetDb).catch(() => { /* 静默失败 */ })
        }
      }
    } catch (_e) { /* ignore */ }
  }

  // ========== 辅助 ==========

  function setConnState(id: string, state: ConnState) {
    connStates.value[id] = state
  }

  function getConnState(id: string): ConnState {
    return connStates.value[id] || { status: 'disconnected', currentDb: 0, cliDb: 0 }
  }

  function setCurrentDb(connId: string, db: number) {
    const state = getConnState(connId)
    setConnState(connId, { ...state, currentDb: db })
    dbPreferences.value[connId] = db
  }

  function setCliDb(connId: string, db: number) {
    const state = getConnState(connId)
    setConnState(connId, { ...state, cliDb: db })
    cliDbPreferences.value[connId] = db
  }

  function saveCliCache(connId: string, history: any[], cmdHistory: string[]) {
    cliCaches.set(connId, { history, cmdHistory })
  }

  function getCliCache(connId: string) {
    return cliCaches.get(connId)
  }

  function clearCliCache(connId: string) {
    cliCaches.delete(connId)
  }

  function saveStatsCache(connId: string, points: any[]) {
    statsCaches.set(connId, points)
  }

  function getStatsCache(connId: string): any[] | undefined {
    return statsCaches.get(connId)
  }

  function clearStatsCache(connId: string) {
    statsCaches.delete(connId)
  }

  function savePubsubCache(connId: string, data: { messages: any[]; subscribed: boolean; channelInput: string; usePattern: boolean }) {
    pubsubCaches.set(connId, { ...data, messages: [...data.messages] })
  }

  function getPubsubCache(connId: string) {
    return pubsubCaches.get(connId)
  }

  function clearPubsubCache(connId: string) {
    pubsubCaches.delete(connId)
  }

  function saveMonitorCache(connId: string, data: { commands: any[]; running: boolean; filter: string }) {
    monitorCaches.set(connId, { ...data, commands: [...data.commands] })
  }

  function getMonitorCache(connId: string) {
    return monitorCaches.get(connId)
  }

  function clearMonitorCache(connId: string) {
    monitorCaches.delete(connId)
  }

  function saveKeyDetailCache(connId: string, keyName: string, data: any) {
    if (!keyDetailCaches.has(connId)) keyDetailCaches.set(connId, new Map())
    keyDetailCaches.get(connId)!.set(keyName, data)
  }

  function getKeyDetailCache(connId: string, keyName: string): any | undefined {
    return keyDetailCaches.get(connId)?.get(keyName)
  }

  function clearKeyDetailCache(connId: string) {
    keyDetailCaches.delete(connId)
  }

  function saveSelectedKeyCache(connId: string, key: string, type: string) {
    selectedKeyCaches.set(connId, { key, type })
  }

  function getSelectedKeyCache(connId: string) {
    return selectedKeyCaches.get(connId)
  }

  function clearSelectedKeyCache(connId: string) {
    selectedKeyCaches.delete(connId)
  }

  function setSubTabCache(connId: string, tabKey: string) {
    subTabCaches.set(connId, tabKey)
  }

  function getSubTabCache(connId: string): string | undefined {
    return subTabCaches.get(connId)
  }

  function clearSubTabCache(connId: string) {
    subTabCaches.delete(connId)
  }

  function saveKeyListStateCache(connId: string, state: any) {
    keyListStateCaches.set(connId, state)
  }

  function getKeyListStateCache(connId: string): any | undefined {
    return keyListStateCaches.get(connId)
  }

  function clearKeyListStateCache(connId: string) {
    keyListStateCaches.delete(connId)
  }

  function generateId(): string {
    return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
  }

  function parseServerInfo(raw: string): Record<string, string> {
    const result: Record<string, string> = {}
    raw.split('\n').forEach(line => {
      if (line.includes(':')) {
        const [key, val] = line.split(':')
        result[key.trim()] = val?.trim() || ''
      }
    })
    return result
  }

  async function reorderConnections(items: { id: string; group: string }[]) {
    // 1. 立即更新本地数据（同步，匹配 DOM 状态）
    const connMap = new Map(connections.value.map(c => [c.id, c]))
    const reordered: Connection[] = []
    const seen = new Set<string>()
    for (const item of items) {
      const conn = connMap.get(item.id)
      if (conn) {
        conn.group = item.group
        reordered.push(conn)
        seen.add(item.id)
      }
    }
    for (const conn of connections.value) {
      if (!seen.has(conn.id)) reordered.push(conn)
    }
    connections.value = reordered

    // 2. 后端持久化（不 reload）
    request('reorder_connections', { params: { items } })
  }

  return {
    connections, groups, groupMeta, tabs, activeTabId,
    connStates, viewModes, dbPreferences, cliDbPreferences, activeTab, activeConnState,
    loadConnections, saveConnection, deleteConnection, copyConnection,
    connect, disconnect, testConnection,
    addTab, closeTab, pinTab, setActiveTab, deleteGroup,
    getConnState, setConnState, setCurrentDb, setCliDb, getViewMode, setViewMode,
    saveCliCache, getCliCache, clearCliCache,
    saveStatsCache, getStatsCache, clearStatsCache,
    savePubsubCache, getPubsubCache, clearPubsubCache,
    saveMonitorCache, getMonitorCache, clearMonitorCache,
    saveKeyDetailCache, getKeyDetailCache, clearKeyDetailCache,
    saveSelectedKeyCache, getSelectedKeyCache, clearSelectedKeyCache,
    setSubTabCache, getSubTabCache, clearSubTabCache,
    saveKeyListStateCache, getKeyListStateCache, clearKeyListStateCache,
    generateId, generateGroupId, parseServerInfo,
    saveSession, restoreSession, reorderConnections, saveGroups,
  }
})
