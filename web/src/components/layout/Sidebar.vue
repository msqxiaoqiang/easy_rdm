<template>
  <div class="sidebar">
    <!-- 连接列表面板 -->
    <div class="sidebar-panel">
      <!-- 工具栏 -->
      <div class="sidebar-toolbar">
        <a-tooltip :content="$t('connection.new')" position="bottom" mini>
          <a-button size="mini" class="toolbar-btn" @click="showConnectionForm = true">
            <template #icon><IconPlus /></template>
          </a-button>
        </a-tooltip>
        <a-tooltip :content="$t('connection.newGroup')" position="bottom" mini>
          <a-button size="mini" class="toolbar-btn" @click="handleNewGroup">
            <template #icon><IconFolderAdd /></template>
          </a-button>
        </a-tooltip>
        <a-tooltip :content="$t('common.import')" position="bottom" mini>
          <a-button size="mini" class="toolbar-btn" @click="importConnections">
            <template #icon><IconDownload /></template>
          </a-button>
        </a-tooltip>
        <a-tooltip :content="$t('common.export')" position="bottom" mini>
          <a-button size="mini" class="toolbar-btn" @click="exportConnections">
            <template #icon><IconUpload /></template>
          </a-button>
        </a-tooltip>
      </div>

      <!-- 连接列表 -->
      <div class="connection-list">
        <a-tree
          v-if="treeData.length > 0"
          :data="treeData"
          :draggable="true"
          :block-node="true"
          :show-line="false"
          :default-expand-all="true"
          v-model:expanded-keys="expandedKeys"
          :allow-drop="handleAllowDrop"
          @drag-start="handleDragStart"
          @drag-end="handleDragEnd"
          @drop="handleDrop"
        >
          <template #title="nodeData">
            <!-- 文件夹节点 -->
            <div
              v-if="isGroupNode(nodeData.key)"
              class="group-header"
              @contextmenu.prevent
            >
              <svg
                class="group-arrow"
                :class="{ expanded: expandedKeys.includes(String(nodeData.key)) }"
                viewBox="0 0 16 16"
                width="14"
                height="14"
                fill="currentColor"
                @click.stop="toggleGroup(String(nodeData.key))"
              >
                <path d="M6 4l4 4-4 4z" />
              </svg>
              <svg class="group-folder-icon" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
              <span class="group-name-wrap">
                <a-tooltip :content="String(nodeData.title)" :disabled="!groupNameOverflow[String(nodeData.key)]" position="top" mini>
                  <span class="group-name" :data-gk="String(nodeData.key)">{{ nodeData.title }}</span>
                </a-tooltip>
                <span class="group-count">({{ getGroupChildCount(String(nodeData.key)) }})</span>
              </span>
              <span class="group-actions" @click.stop @mousedown.stop>
                <a-tooltip :content="$t('connection.new')" position="top" mini>
                  <button class="group-action-btn" @click.stop="handleAddToGroup(groupKeyToId(String(nodeData.key)))">
                    <IconPlus :size="12" />
                  </button>
                </a-tooltip>
                <a-tooltip :content="$t('connection.renameGroup')" position="top" mini>
                  <button class="group-action-btn" @click.stop="handleRenameGroup(groupKeyToId(String(nodeData.key)))">
                    <IconEdit :size="12" />
                  </button>
                </a-tooltip>
                <a-tooltip :content="$t('common.delete')" position="top" mini>
                  <button class="group-action-btn danger" @click.stop="handleDeleteGroup(groupKeyToId(String(nodeData.key)))">
                    <IconDelete :size="12" />
                  </button>
                </a-tooltip>
              </span>
            </div>
            <!-- 连接节点 -->
            <div
              v-else
              :data-conn-id="String(nodeData.key)"
              :class="['connection-item', {
                active: connectionStore.activeTabId === String(nodeData.key),
                highlighted: highlightedId === String(nodeData.key),
                connected: getStatus(String(nodeData.key)) === 'connected',
                connecting: getStatus(String(nodeData.key)) === 'connecting',
              }]"
              @click="handleClickConnection(getConn(String(nodeData.key))!)"
              @dblclick="handleDblClickConnection(getConn(String(nodeData.key))!)"
              @contextmenu.prevent="showContextMenu($event, getConn(String(nodeData.key))!)"
            >
              <span :class="['status-dot', getStatus(String(nodeData.key))]"></span>
              <span class="conn-name" v-ellipsis-tip>{{ nodeData.title }}</span>
            </div>
          </template>
          <template #drag-icon> </template>
        </a-tree>

        <!-- 空状态 -->
        <div v-if="treeData.length === 0" class="empty-state">
          <p>{{ $t('common.noData') }}</p>
          <a-button type="primary" size="small" @click="showConnectionForm = true">
            {{ $t('connection.new') }}
          </a-button>
        </div>
      </div>
    </div>

    <!-- 连接表单弹窗 -->
    <ConnectionForm
      v-if="showConnectionForm"
      :visible="showConnectionForm"
      :connection="editingConnection"
      @close="showConnectionForm = false; editingConnection = undefined"
      @saved="onConnectionSaved"
    />

    <!-- 右键菜单 -->
    <ContextMenu
      v-if="ctxMenu"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :items="ctxMenu.items"
      @close="ctxMenu = null"
      @select="handleCtxAction"
    />

    <!-- 导出连接弹窗（选择是否包含密码） -->
    <a-modal
      :visible="showExportDialog"
      :title="$t('common.export')"
      :width="400"
      :mask-closable="true"
      unmount-on-close
      @cancel="showExportDialog = false"
    >
      <a-checkbox v-model="exportIncludePasswords">
        {{ $t('connection.exportIncludePasswords') }}
      </a-checkbox>
      <template #footer>
        <a-button @click="showExportDialog = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="doExport">{{ $t('common.export') }}</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, reactive, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useConnectionStore, type Connection } from '../../stores/connection'
import { useI18n } from 'vue-i18n'
import { request } from '../../utils/request'
import { gmConfirm, gmPrompt } from '../../utils/dialog'
import ConnectionForm from '../connection/ConnectionForm.vue'
import ContextMenu, { type MenuItem } from '../common/ContextMenu.vue'
import { IconPlus, IconFolderAdd, IconDownload, IconUpload, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'
import { buildTreeData, isGroupKey, groupKeyToId, idToGroupKey, computeDropResult, computeDropDisplayOrder } from '../../utils/sidebar-tree'
import { showMessage, hasNativeFileDialog, chooseFile as platformChooseFile, chooseFolder, getHttpBaseUrl } from '@/utils/platform'
import type { TreeNodeData } from '@arco-design/web-vue'

const connectionStore = useConnectionStore()
const { t } = useI18n()

const showConnectionForm = ref(false)

const showExportDialog = ref(false)
const exportIncludePasswords = ref(false)
const editingConnection = ref<Connection | undefined>()
const highlightedId = ref('')

// 右键菜单状态
const ctxMenu = ref<{ x: number; y: number; conn: Connection; items: MenuItem[] } | null>(null)

// Tree 数据
const treeData = computed(() => buildTreeData(connectionStore.connections, connectionStore.groups, connectionStore.groupMeta))

// 展开状态（持久化到 localStorage）
const EXPANDED_KEYS_STORAGE = 'sidebar_expanded_keys'
const expandedKeys = ref<string[]>(JSON.parse(localStorage.getItem(EXPANDED_KEYS_STORAGE) || '[]'))
let expandedInited = false

// 初始化时：如果没有存储记录则展开所有文件夹；之后新增文件夹自动展开
watch(treeData, (data) => {
  const allGroupKeys = data.filter(n => !n.isLeaf).map(n => String(n.key))
  if (!expandedInited) {
    expandedInited = true
    // 首次：如果没有持久化记录，展开全部
    if (!localStorage.getItem(EXPANDED_KEYS_STORAGE)) {
      expandedKeys.value = [...allGroupKeys]
    }
  } else {
    // 后续：仅自动展开新增的文件夹
    for (const k of allGroupKeys) {
      if (!expandedKeys.value.includes(k)) {
        expandedKeys.value.push(k)
      }
    }
  }
}, { immediate: true })

// 持久化展开状态
watch(expandedKeys, (keys) => {
  localStorage.setItem(EXPANDED_KEYS_STORAGE, JSON.stringify(keys))
}, { deep: true })

function isGroupNode(key: string | number): boolean {
  return isGroupKey(String(key))
}

function toggleGroup(key: string) {
  const idx = expandedKeys.value.indexOf(key)
  if (idx >= 0) {
    expandedKeys.value.splice(idx, 1)
  } else {
    expandedKeys.value.push(key)
  }
}

function getConn(id: string) {
  return connectionStore.connections.find(c => c.id === id)
}

function getGroupChildCount(key: string): number {
  const node = treeData.value.find(n => String(n.key) === key)
  return node?.children?.length ?? 0
}

// 跟踪当前拖拽的节点 key（allowDrop 回调不接收 dragNode）
const draggingKey = ref<string>('')

function handleDragStart(_ev: DragEvent, node: TreeNodeData) {
  draggingKey.value = String(node.key || '')
}

function handleDragEnd() {
  draggingKey.value = ''
}

function handleAllowDrop({ dropNode, dropPosition }: { dropNode: TreeNodeData; dropPosition: -1 | 0 | 1 }) {
  if (dropPosition === 0) {
    // 只允许连接（非文件夹）移入文件夹
    return !isGroupKey(draggingKey.value) && isGroupKey(String(dropNode.key))
  }
  return true
}

function handleDrop({ dragNode, dropNode, dropPosition }: {
  e: DragEvent
  dragNode: TreeNodeData
  dropNode: TreeNodeData
  dropPosition: -1 | 0 | 1
}) {
  const items = computeDropResult(
    treeData.value,
    dragNode.key!,
    dropNode.key!,
    dropPosition,
  )
  if (items.length) {
    connectionStore.reorderConnections(items)
  }

  // 更新完整的显示顺序（含所有顶层项目）
  const newDisplayOrder = computeDropDisplayOrder(
    treeData.value,
    dragNode.key!,
    dropNode.key!,
    dropPosition,
  )
  connectionStore.groups = newDisplayOrder
  connectionStore.saveGroups()
}

function getStatus(id: string) {
  return connectionStore.getConnState(id).status
}

async function handleClickConnection(conn: Connection) {
  highlightedId.value = conn.id
}

async function handleDblClickConnection(conn: Connection) {
  const state = connectionStore.getConnState(conn.id)
  if (state.status === 'connected') return
  try {
    await connectionStore.connect(conn.id)
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function showContextMenu(event: MouseEvent, conn: Connection) {
  const isConnected = getStatus(conn.id) === 'connected'
  const items: MenuItem[] = [
    { key: 'connect', label: t('connection.connect'), icon: '<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><polygon points="5,3 19,12 5,21"/></svg>', disabled: isConnected },
    { key: 'disconnect', label: t('connection.disconnect'), icon: '<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><rect x="4" y="4" width="16" height="16" rx="2"/></svg>', disabled: !isConnected },
    { key: 'edit', label: t('common.edit'), icon: '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>' },
    { key: 'copy', label: t('common.clone'), icon: '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>' },
    { key: 'delete', label: t('common.delete'), icon: '<svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>', danger: true },
  ]
  ctxMenu.value = { x: event.clientX, y: event.clientY, conn, items }
}

async function handleCtxAction(key: string) {
  const conn = ctxMenu.value?.conn
  if (!conn) return
  ctxMenu.value = null

  switch (key) {
    case 'connect':
      try { await connectionStore.connect(conn.id) } catch (e: any) {
        showMessage('error', e?.message || t('common.failed'))
      }
      break
    case 'disconnect':
      await connectionStore.disconnect(conn.id)
      connectionStore.closeTab(conn.id)
      break
    case 'edit':
      if (getStatus(conn.id) === 'connected') {
        if (await gmConfirm(t('connection.editNeedDisconnect'))) {
          await connectionStore.disconnect(conn.id)
          connectionStore.closeTab(conn.id)
        } else {
          break
        }
      }
      editingConnection.value = conn
      showConnectionForm.value = true
      break
    case 'copy':
      await connectionStore.copyConnection(conn)
      break
    case 'delete':
      if (await gmConfirm(t('connection.deleteConfirm'))) {
        await connectionStore.deleteConnection(conn.id)
      }
      break
  }
}

async function importConnections() {
  try {
    if (hasNativeFileDialog()) {
      // 原生文件选择（gmssh / desktop）
      platformChooseFile((filePath: string) => {
        doImport(filePath)
      }, { accept: '.zip' })
    } else {
      // 浏览器环境：用 input[type=file] 上传
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = '.zip'
      input.onchange = async () => {
        const file = input.files?.[0]
        if (!file) return
        try {
          const formData = new FormData()
          formData.append('file', file)
          const resp = await fetch(`${getHttpBaseUrl()}/upload_import`, { method: 'POST', body: formData })
          const result = await resp.json()
          if (result.code !== 200) throw new Error(result.msg || t('common.failed'))
          await connectionStore.loadConnections()
          const count = result.data?.imported ?? 0
          showMessage('success', `${t('common.success')} (${count})`)
        } catch (e: any) {
          showMessage('error', e?.message || t('common.failed'))
        }
      }
      input.click()
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function doImport(filePath: string) {
  try {
    const res = await request<any>('import_connections_zip', { params: { file_path: filePath } })
    await connectionStore.loadConnections()
    const count = res.data?.imported ?? 0
    showMessage('success', `${t('common.success')} (${count})`)
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function exportConnections() {
  exportIncludePasswords.value = false
  showExportDialog.value = true
}

async function doExport() {
  showExportDialog.value = false
  try {
    if (hasNativeFileDialog()) {
      // 原生目录选择（gmssh / desktop）
      chooseFolder((exportPath: string) => {
        doExportToPath(exportPath)
      }, '')
    } else {
      // 浏览器环境：直接下载文件
      const url = `${getHttpBaseUrl()}/download_export?include_passwords=${exportIncludePasswords.value}`
      const a = document.createElement('a')
      a.href = url
      const now = new Date()
      const ts = now.getFullYear().toString()
        + String(now.getMonth() + 1).padStart(2, '0')
        + String(now.getDate()).padStart(2, '0')
        + '_'
        + String(now.getHours()).padStart(2, '0')
        + String(now.getMinutes()).padStart(2, '0')
        + String(now.getSeconds()).padStart(2, '0')
      a.download = `easy_rdm_connections_${ts}.zip`
      a.click()
      showMessage('success', t('common.success'))
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function doExportToPath(exportPath: string) {
  try {
    await request<any>('export_connections_zip', {
      params: { include_passwords: exportIncludePasswords.value, export_path: exportPath },
    })
    showMessage('success', t('common.success'))
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function onConnectionSaved() {
  showConnectionForm.value = false
  editingConnection.value = undefined
}

function openNewConnection() {
  editingConnection.value = undefined
  showConnectionForm.value = true
}

function handleAddToGroup(groupId: string) {
  editingConnection.value = { group: groupId } as any
  showConnectionForm.value = true
}

async function handleRenameGroup(groupId: string) {
  const currentName = connectionStore.groupMeta[groupId] || groupId
  const newName = await gmPrompt(t('connection.renameGroupPrompt'), currentName)
  if (!newName?.trim() || newName.trim() === currentName) return
  // 只更新 meta 中的显示名，groupId 和连接的 group 字段不变
  connectionStore.groupMeta[groupId] = newName.trim()
  connectionStore.saveGroups()
}

async function handleDeleteGroup(groupId: string) {
  const count = connectionStore.connections.filter(c => c.group === groupId).length
  if (await gmConfirm(t('connection.deleteGroupConfirm', { count }))) {
    await connectionStore.deleteGroup(groupId)
  }
}

async function handleNewGroup() {
  const groupName = await gmPrompt(t('connection.newGroupPrompt'))
  if (groupName?.trim()) {
    const trimmed = groupName.trim()
    // 每次都生成新的唯一 ID，允许同名分组
    const groupId = connectionStore.generateGroupId()
    connectionStore.groupMeta[groupId] = trimmed
    connectionStore.groups.push(idToGroupKey(groupId))
    connectionStore.saveGroups()
    showMessage('success', t('common.success'))
  }
}

// 预计算分组名溢出状态（hover 时不修改 reactive 状态，避免重渲染导致 tooltip 箭头抖动）
const groupNameOverflow = reactive<Record<string, boolean>>({})

function scanGroupOverflows() {
  nextTick(() => {
    document.querySelectorAll<HTMLElement>('.group-name[data-gk]').forEach(el => {
      const key = el.dataset.gk!
      const isOverflow = el.scrollWidth > el.clientWidth
      if (groupNameOverflow[key] !== isOverflow) {
        groupNameOverflow[key] = isOverflow
      }
    })
  })
}

// tree 数据变化时重新扫描
watch(treeData, scanGroupOverflows)

// 挂载时扫描 + 监听侧栏宽度变化
let _resizeOb: ResizeObserver | null = null
onMounted(() => {
  scanGroupOverflows()
  const sidebar = document.querySelector('.sidebar')
  if (sidebar) {
    _resizeOb = new ResizeObserver(scanGroupOverflows)
    _resizeOb.observe(sidebar)
  }
})
onBeforeUnmount(() => {
  _resizeOb?.disconnect()
})

defineExpose({ openNewConnection })
</script>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.sidebar-toolbar {
  display: flex;
  gap: var(--spacing-xs);
  padding: var(--spacing-sm);
  border-bottom: 1px solid var(--color-border-1);
}

.toolbar-btn {
  color: var(--color-text-2);
}

.sidebar-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.connection-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-xs) 0;
}

.group-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-sm) var(--spacing-md);
  padding-right: 68px; /* 预留操作栏固定宽度：3×20px按钮 + 2×2px间距 + 4px右边距 */
  min-height: 32px;
  width: 100%;
  box-sizing: border-box;
  overflow: hidden;
  color: var(--color-text-3);
  font-size: var(--font-size-sm);
  cursor: grab;
  user-select: none;
  position: relative;
}

.group-arrow {
  color: var(--color-text-3);
  flex-shrink: 0;
  cursor: pointer;
  transition: transform 0.2s;
}

.group-arrow.expanded {
  transform: rotate(90deg);
}

.group-folder-icon {
  color: var(--color-text-3);
  flex-shrink: 0;
}

.group-name-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  overflow: hidden;
}

.group-name {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-count {
  flex-shrink: 0;
  white-space: nowrap;
}

.group-actions {
  position: absolute;
  right: var(--spacing-xs);
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity var(--transition-fast);
}

.group-header:hover .group-actions {
  opacity: 1;
}

.group-action-btn {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--color-text-3);
  cursor: pointer;
}

.group-action-btn:hover {
  background: var(--color-fill-2);
  color: var(--color-text-1);
}

.group-action-btn.danger:hover {
  color: var(--color-error);
}


.connection-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  min-height: 32px;
  width: 100%;
  box-sizing: border-box;
  border-radius: var(--radius-sm);
  cursor: grab;
  transition: all var(--transition-fast);
  user-select: none;
}

.connection-item:active {
  cursor: grabbing;
}

.connection-item:hover {
  background: var(--color-fill-1);
}

.connection-item.active {
  background: var(--color-primary-bg);
}

.connection-item.highlighted {
  background: var(--color-fill-2);
}

.connection-item.connected .conn-name {
  font-weight: 500;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background: var(--color-text-4);
}

.status-dot.connected {
  background: var(--color-success);
}

.status-dot.connecting {
  background: var(--color-warning);
}

.status-dot.error {
  background: var(--color-error);
}

.conn-name {
  flex: 1;
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}


.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-xxl) var(--spacing-lg);
  color: var(--color-text-3);
  font-size: var(--font-size-sm);
}

/* ===== Arco Tree 覆盖 ===== */

:deep(.arco-tree) {
  padding: var(--spacing-xs) 0;
}

:deep(.arco-tree-node) {
  padding-left: 0 !important;
}

:deep(.arco-tree-node-title) {
  min-width: 0 !important;
}

:deep(.arco-tree-node-switcher) {
  display: none !important;
}

:deep(.arco-tree) {
  padding: 0 var(--spacing-sm) 0 var(--spacing-xs);
}

:deep(.arco-tree-node-title) {
  padding: 0;
  margin-left: 0;
  background: none !important;
}

:deep(.arco-tree-node-title-text) {
  width: 100%;
  overflow: hidden;
}

:deep(.arco-tree-node-title:hover) {
  background: none !important;
}

:deep(.arco-tree-node-drag-icon) {
  display: none !important;
}

/* 拖拽插入线 — 上方 */
:deep(.arco-tree-node-title-gap-top::before) {
  height: 2px;
  background-color: var(--color-primary) !important;
}

/* 拖拽插入线 — 下方 */
:deep(.arco-tree-node-title-gap-bottom::after) {
  height: 2px;
  background-color: var(--color-primary) !important;
}

/* 拖拽高亮 — 移入文件夹效果 */
:deep(.arco-tree-node-title-highlight) {
  background-color: var(--color-primary-light-1, rgba(var(--primary-6), 0.08)) !important;
  border-radius: var(--radius-sm);
}

/* 拖拽中的原始项半透明 */
:deep(.arco-tree-node-title-dragging) {
  opacity: 0.5;
}

/* blockNode 让 title 占满宽度 */
:deep(.arco-tree-node-title-block) {
  flex: 1;
}

.export-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  line-height: 1.5;
  margin-top: var(--spacing-sm);
}
</style>
