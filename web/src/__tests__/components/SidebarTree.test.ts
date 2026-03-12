import { describe, it, expect } from 'vitest'
import { buildTreeData, GROUP_PREFIX, shouldAllowDrop, computeDropResult, idToGroupKey } from '@/utils/sidebar-tree'

describe('buildTreeData', () => {
  it('无数据时返回空数组', () => {
    expect(buildTreeData([], [], {})).toEqual([])
  })

  it('未分组连接作为顶层叶子节点', () => {
    const conns = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
      { id: 'c2', name: 'Redis-2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    const result = buildTreeData(conns, [], {})
    expect(result).toHaveLength(2)
    expect(result[0]).toMatchObject({ key: 'c1', title: 'Redis-1', isLeaf: true })
    expect(result[1]).toMatchObject({ key: 'c2', title: 'Redis-2', isLeaf: true })
  })

  it('分组连接在文件夹父节点下', () => {
    const conns = [
      { id: 'c1', name: 'Redis-1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
      { id: 'c2', name: 'Redis-Prod', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    const result = buildTreeData(conns, [], {})
    expect(result).toHaveLength(2)
    expect(result[0]).toMatchObject({ key: 'c1', isLeaf: true })
    expect(result[1]).toMatchObject({ key: `${GROUP_PREFIX}prod`, title: 'prod', isLeaf: false })
    expect(result[1].children).toHaveLength(1)
    expect(result[1].children[0]).toMatchObject({ key: 'c2', title: 'Redis-Prod', isLeaf: true })
  })

  it('空文件夹也应出现', () => {
    const result = buildTreeData([], [idToGroupKey('staging')], {})
    expect(result).toHaveLength(1)
    expect(result[0]).toMatchObject({ key: `${GROUP_PREFIX}staging`, title: 'staging', isLeaf: false })
    expect(result[0].children).toEqual([])
  })

  it('混合场景：未分组 + 有连接的分组 + 空分组', () => {
    const conns = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
      { id: 'c2', name: 'R2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    // displayOrder 包含所有顶层项目的 key
    const displayOrder = ['c1', idToGroupKey('prod'), idToGroupKey('staging')]
    const result = buildTreeData(conns, displayOrder, {})
    expect(result).toHaveLength(3)
    expect(result[0].key).toBe('c1')
    expect(result[1].key).toBe(`${GROUP_PREFIX}prod`)
    expect(result[1].children).toHaveLength(1)
    expect(result[2].key).toBe(`${GROUP_PREFIX}staging`)
    expect(result[2].children).toEqual([])
  })

  it('displayOrder 决定所有顶层项目的显示顺序', () => {
    const conns = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'alpha' },
      { id: 'c2', name: 'R2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'beta' },
    ]
    // displayOrder 指定 beta 排在 alpha 前面
    const displayOrder = [idToGroupKey('beta'), idToGroupKey('alpha')]
    const result = buildTreeData(conns, displayOrder, {})
    expect(result[0].key).toBe(`${GROUP_PREFIX}beta`)
    expect(result[0].children![0].key).toBe('c2')
    expect(result[1].key).toBe(`${GROUP_PREFIX}alpha`)
    expect(result[1].children![0].key).toBe('c1')
  })

  it('保持 connections 数组中子节点的顺序', () => {
    const conns = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
      { id: 'c2', name: 'R2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
      { id: 'c3', name: 'R3', host: '10.0.0.2', port: 6381, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    // displayOrder: group prod 在前，c2 在后
    const displayOrder = [idToGroupKey('prod'), 'c2']
    const result = buildTreeData(conns, displayOrder, {})
    expect(result[0].key).toBe(`${GROUP_PREFIX}prod`)
    expect(result[0].children[0].key).toBe('c1')
    expect(result[0].children[1].key).toBe('c3')
    expect(result[1].key).toBe('c2')
  })

  it('displayOrder 中重复的 key 不应创建重复节点', () => {
    const conns = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    const displayOrder = [idToGroupKey('prod'), idToGroupKey('staging')]
    const result = buildTreeData(conns, displayOrder, {})
    const groupKeys = result.filter((n: any) => !n.isLeaf).map((n: any) => n.key)
    expect(groupKeys).toEqual([`${GROUP_PREFIX}prod`, `${GROUP_PREFIX}staging`])
  })

  it('未分组连接可以在分组之间自由排序', () => {
    const conns = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
      { id: 'c2', name: 'R2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    ]
    // c1 排在 prod 后面
    const displayOrder = [idToGroupKey('prod'), 'c1']
    const result = buildTreeData(conns, displayOrder, {})
    expect(result[0].key).toBe(`${GROUP_PREFIX}prod`)
    expect(result[1].key).toBe('c1')
  })
})

describe('shouldAllowDrop', () => {
  it('禁止文件夹移入文件夹 (dropPosition=0)', () => {
    expect(shouldAllowDrop(`${GROUP_PREFIX}alpha`, `${GROUP_PREFIX}beta`, 0)).toBe(false)
  })

  it('禁止连接移入连接 (dropPosition=0)', () => {
    expect(shouldAllowDrop('c1', 'c2', 0)).toBe(false)
  })

  it('允许连接移入文件夹 (dropPosition=0)', () => {
    expect(shouldAllowDrop('c1', `${GROUP_PREFIX}prod`, 0)).toBe(true)
  })

  it('允许文件夹在文件夹前后排序', () => {
    expect(shouldAllowDrop(`${GROUP_PREFIX}alpha`, `${GROUP_PREFIX}beta`, -1)).toBe(true)
    expect(shouldAllowDrop(`${GROUP_PREFIX}alpha`, `${GROUP_PREFIX}beta`, 1)).toBe(true)
  })

  it('允许连接在连接前后排序', () => {
    expect(shouldAllowDrop('c1', 'c2', -1)).toBe(true)
    expect(shouldAllowDrop('c1', 'c2', 1)).toBe(true)
  })
})

describe('computeDropResult', () => {
  const baseConns = [
    { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    { id: 'c2', name: 'R2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
    { id: 'c3', name: 'R3', host: '10.0.0.2', port: 6381, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
  ]

  it('连接移入文件夹 (dropPosition=0)', () => {
    const treeData = buildTreeData(baseConns, [], {})
    const result = computeDropResult(treeData, 'c1', `${GROUP_PREFIX}prod`, 0)
    expect(result.find(i => i.id === 'c1')!.group).toBe('prod')
    expect(result.find(i => i.id === 'c2')!.group).toBe('prod')
  })

  it('连接在文件夹前插入 (dropPosition=-1)', () => {
    const treeData = buildTreeData(baseConns, [], {})
    const result = computeDropResult(treeData, 'c2', 'c1', -1)
    expect(result.find(i => i.id === 'c2')!.group).toBe('')
    const ids = result.map(i => i.id)
    expect(ids.indexOf('c2')).toBeLessThan(ids.indexOf('c1'))
  })

  it('连接在文件夹后插入 (dropPosition=1)', () => {
    const treeData = buildTreeData(baseConns, [], {})
    const result = computeDropResult(treeData, 'c1', `${GROUP_PREFIX}prod`, 1)
    expect(result.find(i => i.id === 'c1')!.group).toBe('')
  })

  it('连接在组内连接前后排序', () => {
    const treeData = buildTreeData(baseConns, [], {})
    const result = computeDropResult(treeData, 'c3', 'c2', -1)
    const prodItems = result.filter(i => i.group === 'prod')
    expect(prodItems[0].id).toBe('c3')
    expect(prodItems[1].id).toBe('c2')
  })

  it('文件夹排序 (dropPosition=-1 between groups)', () => {
    const conns = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'alpha' },
      { id: 'c2', name: 'R2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'beta' },
    ]
    const treeData = buildTreeData(conns, [], {})
    const result = computeDropResult(treeData, `${GROUP_PREFIX}beta`, `${GROUP_PREFIX}alpha`, -1)
    const ids = result.map(i => i.id)
    expect(ids.indexOf('c2')).toBeLessThan(ids.indexOf('c1'))
  })

  it('连接拖出文件夹到顶层', () => {
    const conns = [
      { id: 'c1', name: 'R1', host: '127.0.0.1', port: 6379, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10, group: 'prod' },
      { id: 'c2', name: 'R2', host: '10.0.0.1', port: 6380, db: 0, conn_type: 'tcp', conn_timeout: 10, exec_timeout: 10 },
    ]
    const treeData = buildTreeData(conns, [], {})
    const result = computeDropResult(treeData, 'c1', 'c2', 1)
    expect(result.find(i => i.id === 'c1')!.group).toBe('')
    const ids = result.map(i => i.id)
    expect(ids.indexOf('c1')).toBeGreaterThan(ids.indexOf('c2'))
  })
})
