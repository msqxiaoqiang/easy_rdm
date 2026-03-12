import type { TreeNodeData } from '@arco-design/web-vue'
import type { Connection } from '../stores/connection'

/** 分组节点 key 前缀 */
export const GROUP_PREFIX = '__group__'

/** 判断 key 是否为分组节点 */
export function isGroupKey(key: string): boolean {
  return key.startsWith(GROUP_PREFIX)
}

/** 从分组 key 提取分组 ID（__group__ 后的部分） */
export function groupKeyToId(key: string): string {
  return key.slice(GROUP_PREFIX.length)
}

/** 将分组 ID 转为分组 key */
export function idToGroupKey(id: string): string {
  return GROUP_PREFIX + id
}

/** 从 displayOrder 数组中提取分组 ID 列表 */
export function extractGroupIds(displayOrder: string[]): string[] {
  return displayOrder.filter(k => isGroupKey(k)).map(k => groupKeyToId(k))
}

/**
 * 将连接列表和显示顺序构建为树节点数据
 *
 * displayOrder 包含所有顶层项目的 key（分组用 __group__<groupId>，未分组连接用连接 ID）
 * groupMeta 映射分组 ID → 显示名称
 * 按 displayOrder 的顺序渲染，不在 displayOrder 中的项目追加到末尾
 */
export function buildTreeData(
  connections: Connection[],
  displayOrder: string[],
  groupMeta: Record<string, string>,
): TreeNodeData[] {
  // 构建分组ID → 子连接映射（conn.group 存的是分组 ID）
  const groupChildrenMap = new Map<string, TreeNodeData[]>()
  const ungroupedMap = new Map<string, TreeNodeData>()

  for (const conn of connections) {
    if (conn.group) {
      const groupId = conn.group
      if (!groupChildrenMap.has(groupId)) {
        groupChildrenMap.set(groupId, [])
      }
      groupChildrenMap.get(groupId)!.push({
        key: conn.id,
        title: conn.name,
        isLeaf: true,
      })
    } else {
      ungroupedMap.set(conn.id, {
        key: conn.id,
        title: conn.name,
        isLeaf: true,
      })
    }
  }

  // 按 displayOrder 构建结果
  const result: TreeNodeData[] = []
  const usedGroupIds = new Set<string>()
  const usedConns = new Set<string>()

  for (const key of displayOrder) {
    if (isGroupKey(key)) {
      const groupId = groupKeyToId(key)
      usedGroupIds.add(groupId)
      const displayName = groupMeta[groupId] || groupId
      result.push({
        key,
        title: displayName,
        isLeaf: false,
        children: groupChildrenMap.get(groupId) || [],
      })
    } else {
      const node = ungroupedMap.get(key)
      if (node) {
        usedConns.add(key)
        result.push(node)
      }
    }
  }

  // 追加不在 displayOrder 中的项目（按 connections 数组顺序）
  const appendedGroupIds = new Set<string>()
  for (const conn of connections) {
    if (conn.group) {
      if (!usedGroupIds.has(conn.group) && !appendedGroupIds.has(conn.group)) {
        appendedGroupIds.add(conn.group)
        const displayName = groupMeta[conn.group] || conn.group
        result.push({
          key: idToGroupKey(conn.group),
          title: displayName,
          isLeaf: false,
          children: groupChildrenMap.get(conn.group) || [],
        })
      }
    } else {
      if (!usedConns.has(conn.id)) {
        const node = ungroupedMap.get(conn.id)
        if (node) result.push(node)
      }
    }
  }

  return result
}

/**
 * 判断拖拽是否允许放置
 *
 * - dropPosition === 0 时：只允许非 group 拖入 group（连接移入文件夹）
 * - dropPosition === -1/1 时：总是允许
 */
export function shouldAllowDrop(
  dragKey: string,
  dropKey: string,
  dropPosition: -1 | 0 | 1,
): boolean {
  if (dropPosition === 0) {
    // 只允许连接（非 group）拖入文件夹（group）
    return !isGroupKey(dragKey) && isGroupKey(dropKey)
  }
  return true
}

/**
 * 执行拖拽操作：深拷贝 treeData，移除 dragNode，插入到 dropKey 指定位置
 * 返回操作后的新树
 */
function applyDrop(
  treeData: TreeNodeData[],
  dragKey: string | number,
  dropKey: string | number,
  dropPosition: -1 | 0 | 1,
): TreeNodeData[] {
  const tree: TreeNodeData[] = JSON.parse(JSON.stringify(treeData))

  let dragNode: TreeNodeData | null = null

  for (let i = tree.length - 1; i >= 0; i--) {
    const node = tree[i]
    if (String(node.key) === String(dragKey)) {
      dragNode = node
      tree.splice(i, 1)
      break
    }
    if (node.children) {
      for (let j = node.children.length - 1; j >= 0; j--) {
        if (String(node.children[j].key) === String(dragKey)) {
          dragNode = node.children[j]
          node.children.splice(j, 1)
          break
        }
      }
      if (dragNode) break
    }
  }

  if (!dragNode) return tree

  if (dropPosition === 0) {
    for (const node of tree) {
      if (String(node.key) === String(dropKey)) {
        if (!node.children) node.children = []
        node.children.push(dragNode)
        break
      }
    }
  } else {
    insertAtPosition(tree, dragNode, dropKey, dropPosition)
  }

  return tree
}

/**
 * 计算拖拽后的连接排序结果
 *
 * 返回展平后的 { id, group }[] 数组，表示每个连接的 id 和所属分组（分组 ID）
 */
export function computeDropResult(
  treeData: TreeNodeData[],
  dragKey: string | number,
  dropKey: string | number,
  dropPosition: -1 | 0 | 1,
): { id: string; group: string }[] {
  return flattenTree(applyDrop(treeData, dragKey, dropKey, dropPosition))
}

/**
 * 计算拖拽后的完整显示顺序
 *
 * 返回所有顶层节点的 key 数组（分组用 __group__<groupId>，未分组连接用 ID）
 */
export function computeDropDisplayOrder(
  treeData: TreeNodeData[],
  dragKey: string | number,
  dropKey: string | number,
  dropPosition: -1 | 0 | 1,
): string[] {
  const tree = applyDrop(treeData, dragKey, dropKey, dropPosition)
  return tree.map(n => String(n.key))
}

/**
 * 在树中找到 dropKey 对应的节点，在其前/后插入 dragNode
 */
function insertAtPosition(
  tree: TreeNodeData[],
  dragNode: TreeNodeData,
  dropKey: string | number,
  dropPosition: -1 | 1,
): void {
  // 先在顶层查找
  for (let i = 0; i < tree.length; i++) {
    if (String(tree[i].key) === String(dropKey)) {
      const insertIndex = dropPosition === -1 ? i : i + 1
      tree.splice(insertIndex, 0, dragNode)
      return
    }
    // 在子节点中查找
    if (tree[i].children) {
      for (let j = 0; j < tree[i].children!.length; j++) {
        if (String(tree[i].children![j].key) === String(dropKey)) {
          const insertIndex = dropPosition === -1 ? j : j + 1
          tree[i].children!.splice(insertIndex, 0, dragNode)
          return
        }
      }
    }
  }
}

/**
 * 展平树结构为 { id, group }[] 数组
 * - 顶层叶子节点 group 为 ''
 * - 文件夹内叶子节点 group 为分组 ID（__group__ 后的部分）
 */
function flattenTree(tree: TreeNodeData[]): { id: string; group: string }[] {
  const result: { id: string; group: string }[] = []

  for (const node of tree) {
    if (node.isLeaf) {
      result.push({ id: String(node.key), group: '' })
    } else {
      // 文件夹节点，展开其子节点
      const groupId = groupKeyToId(String(node.key))
      if (node.children) {
        for (const child of node.children) {
          result.push({ id: String(child.key), group: groupId })
        }
      }
    }
  }

  return result
}
