<template>
  <div class="key-list-panel" ref="panelRef" tabindex="-1" @keydown="handleKeydown">
    <!-- 工具栏 -->
    <div class="key-toolbar">
      <a-tooltip :content="viewMode === 'flat' ? $t('common.treeView') : $t('common.flatView')" mini>
        <a-button size="mini" @click="toggleView">
          <icon-mind-mapping v-if="viewMode === 'flat'" :size="14" />
          <icon-list v-else :size="14" />
        </a-button>
      </a-tooltip>
      <a-tooltip :content="$t('common.refresh')" mini>
        <a-button size="mini" @click="refreshKeys">
          <icon-refresh :size="14" />
        </a-button>
      </a-tooltip>
      <a-tooltip :content="$t('newKey.title')" mini>
        <a-button size="mini" @click="newKeyDefaultName = ''; showNewKey = true">
          <icon-plus :size="14" />
        </a-button>
      </a-tooltip>
      <div class="toolbar-spacer"></div>
      <!-- More actions dropdown -->
      <div class="more-dropdown" ref="moreDropdownRef">
        <a-tooltip :content="$t('common.more')" mini :disabled="showMoreMenu">
          <a-button size="mini" @click="showMoreMenu = !showMoreMenu">
            <icon-more :size="14" />
          </a-button>
        </a-tooltip>
        <div v-if="showMoreMenu" class="more-menu">
          <!-- 排序 -->
          <div class="more-menu-group">
            <span class="more-menu-label">{{ $t('key.sortBy') }}</span>
            <div class="sort-row">
              <a-select v-model="sortMode" class="more-menu-select" size="small" @change="showMoreMenu = false">
                <a-option value="name">{{ $t('key.sortByName') }}</a-option>
                <a-option value="type">{{ $t('key.sortByType') }}</a-option>
                <a-option value="ttl">{{ $t('key.sortByTTL') }}</a-option>
                <a-option value="memory" :disabled="!supportsMemory">{{ $t('key.sortByMemory') }}</a-option>
              </a-select>
              <button class="sort-order-btn" @click="sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'" :title="sortOrder === 'asc' ? $t('key.sortAsc') : $t('key.sortDesc')">
                <IconArrowUp v-if="sortOrder === 'asc'" :size="12" />
                <IconArrowDown v-else :size="12" />
              </button>
            </div>
          </div>
          <!-- 收藏筛选 -->
          <button :class="['more-menu-item', { active: showFavoritesOnly }]" @click="showFavoritesOnly = !showFavoritesOnly; showMoreMenu = false">
            <icon-star-fill v-if="showFavoritesOnly" :size="14" style="color: var(--color-warning)" />
            <icon-star v-else :size="14" />
            <span>{{ $t('key.showFavoritesOnly') }}</span>
          </button>
          <div class="more-menu-divider"></div>
          <button class="more-menu-item" @click="showExport = true; showMoreMenu = false">
            <icon-upload :size="14" />
            <span>{{ $t('common.export') }}</span>
          </button>
          <button class="more-menu-item" @click="showImport = true; showMoreMenu = false">
            <icon-download :size="14" />
            <span>{{ $t('common.import') }}</span>
          </button>
          <div class="more-menu-divider"></div>
          <button class="more-menu-item" @click="showBatch = true; showMoreMenu = false">
            <icon-apps :size="14" />
            <span>{{ $t('batch.title') }}</span>
          </button>
          <button class="more-menu-item" @click="showMigrate = true; showMoreMenu = false">
            <icon-swap :size="14" />
            <span>{{ $t('migrate.title') }}</span>
          </button>
          <button v-if="!isCluster" class="more-menu-item" @click="showCrossDbSearch = true; showMoreMenu = false">
            <icon-search :size="14" />
            <span>{{ $t('crossDbSearch.title') }}</span>
          </button>
          <button class="more-menu-item danger" @click="handleFlushDb(); showMoreMenu = false">
            <icon-delete :size="14" />
            <span>{{ $t('key.flushDb') }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="key-search">
      <a-input
        ref="searchInputRef"
        v-model="searchPattern"
        size="small"
        :placeholder="$t('common.search') + ' (e.g. user:*)'"
        allow-clear
        @keydown.enter="refreshKeys"
        @clear="searchPattern = '*'; refreshKeys()"
      />
    </div>

    <!-- Key 列表（虚拟滚动） -->
    <div class="key-list" ref="listRef" @scroll="handleScroll">
      <div class="virtual-spacer" :style="{ height: totalHeight + 'px' }">
        <div class="virtual-window" :style="{ transform: 'translateY(' + offsetY + 'px)' }">
          <!-- 平铺视图 -->
          <template v-if="viewMode === 'flat'">
            <div
              v-for="item in visibleFlatItems"
              :key="item.key"
              :class="['key-item', { active: selectedKey === item.key, 'multi-selected': multiSelected.has(item.key) }]"
              @click="handleItemClick($event, item)"
              @contextmenu.prevent="openContextMenu($event, item)"
            >
              <input v-if="checkMode" type="checkbox" class="key-checkbox" :checked="multiSelected.has(item.key)" @click.stop="toggleCheck(item.key)" />
              <span :class="['type-badge', item.type]">{{ typeLabel(item.type) }}</span>
              <span class="key-name" v-ellipsis-tip>{{ item.key }}</span>
              <span :class="['fav-star', { active: favorites.has(item.key) }]" @click.stop="toggleFavorite(item.key)"><icon-star-fill v-if="favorites.has(item.key)" :size="14" /><icon-star v-else :size="14" /></span>
              <span :class="['key-ttl', ttlClass(item.ttl)]">{{ formatTTL(item.ttl) }}</span>
            </div>
          </template>

          <!-- 树形视图 -->
          <template v-else>
            <template v-for="node in visibleTreeItems" :key="node.path">
              <div
                v-if="node.isGroup"
                class="tree-group"
                :style="{ paddingLeft: (node.depth * 16 + 8) + 'px' }"
                @click="toggleTreeNode(node.path)"
                @contextmenu.prevent="openGroupContextMenu($event, node)"
              >
                <input v-if="checkMode" type="checkbox" class="key-checkbox" :checked="isGroupAllChecked(node.path)" :indeterminate="isGroupPartialChecked(node.path)" @click.stop="toggleGroupCheck(node.path)" />
                <span class="tree-arrow" :class="{ collapsed: collapsedNodes.has(node.path) }"><IconRight :size="12" /></span>
                <span class="tree-group-name" v-ellipsis-tip>{{ node.label }}</span>
                <span class="tree-count">({{ node.count }})</span>
                <span class="tree-group-actions" @click.stop>
                  <button class="tree-action-btn" @click="handleGroupHoverAction('refresh', node.path)" v-body-tooltip="$t('common.refresh')">↻</button>
                  <button class="tree-action-btn" @click="handleGroupHoverAction('addKey', node.path)" v-body-tooltip="$t('common.add')">+</button>
                  <button class="tree-action-btn danger" @click="handleGroupHoverAction('delete', node.path)" v-body-tooltip="$t('common.delete')"><IconDelete :size="12" /></button>
                </span>
              </div>
              <div
                v-else
                :class="['key-item', { active: selectedKey === node.fullKey, 'multi-selected': multiSelected.has(node.fullKey!) }]"
                :style="{ paddingLeft: (node.depth * 16 + 8) + 'px' }"
                @click="handleItemClick($event, { key: node.fullKey!, type: node.type!, ttl: node.ttl! })"
                @contextmenu.prevent="openContextMenu($event, { key: node.fullKey!, type: node.type!, ttl: node.ttl! })"
              >
                <input v-if="checkMode" type="checkbox" class="key-checkbox" :checked="multiSelected.has(node.fullKey!)" @click.stop="toggleCheck(node.fullKey!)" />
                <span :class="['type-badge', node.type]">{{ typeLabel(node.type!) }}</span>
                <span class="key-name" v-ellipsis-tip>{{ node.label }}</span>
                <span :class="['fav-star', { active: favorites.has(node.fullKey!) }]" @click.stop="toggleFavorite(node.fullKey!)"><icon-star-fill v-if="favorites.has(node.fullKey!)" :size="14" /><icon-star v-else :size="14" /></span>
                <span :class="['key-ttl', ttlClass(node.ttl!)]">{{ formatTTL(node.ttl!) }}</span>
              </div>
            </template>
          </template>
        </div>
      </div>

      <!-- 加载中 -->
      <div v-if="loading" class="key-loading">{{ $t('common.loading') }}</div>

      <!-- 空状态 -->
      <div v-if="!loading && keys.length === 0" class="key-empty">
        {{ $t('common.noData') }}
      </div>
    </div>

    <!-- 底部栏 -->
    <div v-if="checkMode" class="check-action-bar">
      <span class="check-count">{{ $t('key.checkedCount', { count: multiSelected.size }) }}</span>
      <div class="check-actions">
        <a-tooltip :content="$t('common.export')" mini>
          <a-button size="mini" :disabled="multiSelected.size === 0" @click="showCheckExport = true"><icon-download :size="14" /></a-button>
        </a-tooltip>
        <a-tooltip :content="$t('key.setTTL')" mini>
          <a-button size="mini" :disabled="multiSelected.size === 0" @click="batchSetTTLChecked"><icon-clock-circle :size="14" /></a-button>
        </a-tooltip>
        <a-tooltip :content="$t('common.delete')" mini>
          <a-button size="mini" status="danger" :disabled="multiSelected.size === 0" @click="batchDeleteChecked"><icon-delete :size="14" /></a-button>
        </a-tooltip>
        <a-tooltip :content="$t('common.cancel')" mini>
          <a-button size="mini" @click="toggleCheckMode"><icon-close :size="14" /></a-button>
        </a-tooltip>
      </div>
    </div>
    <div v-else class="key-status-bar">
      <a-select
        v-if="!isCluster"
        v-model="currentDb"
        class="db-select-bottom"
        size="mini"
        :popup-visible="dbPopupVisible"
        @change="handleDbChange"
        @popup-visible-change="handleDbPopupVisibleChange"
      >
        <a-option v-for="db in dbList" :key="db.db" :value="db.db">
          db{{ db.db }} ({{ db.db === currentDb ? keys.length + '/' : '' }}{{ db.keys }})
        </a-option>
      </a-select>
      <span v-else class="cluster-label-bottom">Cluster</span>
      <div class="status-bar-spacer"></div>
      <a-tooltip :content="$t('key.loadMore')" mini>
        <a-button size="mini" @click="loadMore" :disabled="!hasMore || loading"><icon-arrow-down :size="14" /></a-button>
      </a-tooltip>
      <a-tooltip :content="loadingAll ? $t('common.loading') : $t('key.loadAll')" mini>
        <a-button size="mini" @click="loadAllKeys" :disabled="!hasMore || loading || loadingAll">
          <icon-double-down :size="14" :spin="loadingAll" />
        </a-button>
      </a-tooltip>
      <a-tooltip :content="$t('key.checkMode')" mini>
        <a-button size="mini" @click="toggleCheckMode"><icon-check-square :size="14" /></a-button>
      </a-tooltip>
    </div>

    <!-- 新建 Key 弹窗 -->
    <NewKeyDialog
      :conn-id="connId"
      :visible="showNewKey"
      :default-key-name="newKeyDefaultName"
      @close="showNewKey = false; newKeyDefaultName = ''"
      @created="onKeyCreated"
    />

    <!-- 右键菜单 -->
    <ContextMenu
      v-if="ctxMenu.visible"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :items="ctxMenuItems"
      @select="handleCtxAction"
      @close="ctxMenu.visible = false"
    />

    <!-- 分组右键菜单 -->
    <ContextMenu
      v-if="groupCtxMenu.visible"
      :x="groupCtxMenu.x"
      :y="groupCtxMenu.y"
      :items="groupCtxMenuItems"
      @select="handleGroupCtxAction"
      @close="groupCtxMenu.visible = false"
    />

    <!-- 分组删除弹窗 -->
    <GroupDeleteDialog
      :conn-id="connId"
      :visible="showGroupDelete"
      :group-prefix="groupDeletePrefix"
      :separator="separator"
      @close="showGroupDelete = false"
      @deleted="refreshKeys"
    />

    <!-- 导出弹窗（普通导出，不传选中键） -->
    <ExportDialog
      :conn-id="connId"
      :visible="showExport"
      :selected-keys="[]"
      @close="showExport = false"
    />
    <!-- 导出弹窗（勾选模式导出，传选中键） -->
    <ExportDialog
      :conn-id="connId"
      :visible="showCheckExport"
      :selected-keys="[...multiSelected]"
      @close="showCheckExport = false"
    />

    <!-- 导入弹窗 -->
    <ImportDialog
      :conn-id="connId"
      :visible="showImport"
      @close="showImport = false"
      @imported="refreshKeys"
    />

    <!-- 批量操作弹窗 -->
    <BatchDialog
      :conn-id="connId"
      :visible="showBatch"
      :is-cluster="isCluster"
      @close="showBatch = false"
      @done="refreshKeys"
    />

    <!-- 数据迁移弹窗 -->
    <MigrateDialog
      :conn-id="connId"
      :visible="showMigrate"
      :selected-keys="[...multiSelected]"
      @close="showMigrate = false"
      @done="refreshKeys"
    />

    <!-- 跨库搜索弹窗 -->
    <CrossDbSearchDialog
      :conn-id="connId"
      :visible="showCrossDbSearch"
      @close="showCrossDbSearch = false"
      @jump="handleCrossDbJump"
    />

    <!-- 批量删除确认弹窗 -->
    <a-modal
      :visible="showBatchDeleteConfirm"
      :title="$t('groupDelete.confirmDelete', { count: multiSelected.size })"
      :width="520"
      unmount-on-close
      @cancel="showBatchDeleteConfirm = false"
    >
      <div class="preview-header">{{ $t('groupDelete.affectedKeys', { count: multiSelected.size }) }}</div>
      <div class="preview-list">
        <div v-for="k in [...multiSelected].sort()" :key="k" class="preview-item" v-ellipsis-tip>{{ k }}</div>
      </div>
      <template #footer>
        <a-button @click="showBatchDeleteConfirm = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" status="danger" :disabled="batchDeleteLoading" :loading="batchDeleteLoading" @click="confirmBatchDelete">
          {{ $t('groupDelete.confirmDelete', { count: multiSelected.size }) }}
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, onBeforeUnmount, nextTick } from 'vue'
import { request } from '../../utils/request'
import { useConnectionStore } from '../../stores/connection'
import { useSettingsStore } from '../../stores/settings'
import { useI18n } from 'vue-i18n'
import { gmConfirm, gmPrompt } from '../../utils/dialog'
import { showMessage } from '@/utils/platform'
import { matchesAction } from '../../composables/useShortcuts'
import { versionGte } from '../../utils/version'
import NewKeyDialog from './NewKeyDialog.vue'
import GroupDeleteDialog from './GroupDeleteDialog.vue'
import ExportDialog from './ExportDialog.vue'
import ImportDialog from './ImportDialog.vue'
import BatchDialog from './BatchDialog.vue'
import MigrateDialog from './MigrateDialog.vue'
import CrossDbSearchDialog from './CrossDbSearchDialog.vue'
import ContextMenu from '../common/ContextMenu.vue'
import type { MenuItem } from '../common/ContextMenu.vue'
import { IconMindMapping, IconList, IconRefresh, IconPlus, IconCheckSquare, IconMore, IconUpload, IconDownload, IconApps, IconSwap, IconSearch, IconDelete, IconStar, IconStarFill, IconClockCircle, IconClose, IconArrowDown, IconArrowUp, IconDoubleDown, IconRight } from '@arco-design/web-vue/es/icon'

interface KeyItem {
  key: string
  type: string
  ttl: number
}

interface TreeNode {
  path: string
  label: string
  depth: number
  isGroup: boolean
  count?: number
  fullKey?: string
  type?: string
  ttl?: number
}

const props = defineProps<{ connId: string }>()
const emit = defineEmits<{ selectKey: [key: string, type: string]; deleted: [key: string]; refreshKey: [key: string] }>()

const { t } = useI18n()
const connectionStore = useConnectionStore()
const settingsStore = useSettingsStore()
const connState = computed(() => connectionStore.getConnState(props.connId))
const isCluster = computed(() => connectionStore.connections.find(c => c.id === props.connId)?.use_cluster ?? false)
const supportsMemory = computed(() => versionGte(connState.value.redisVersion, '4.0.0'))

// 防竞态：connId 变化时递增，异步请求返回后检查是否过期
let connGeneration = 0

const keys = ref<KeyItem[]>([])
const dbList = ref<{ db: number; keys: number }[]>([])
const currentDb = ref(0)
const dbPopupVisible = ref(false)
const searchPattern = ref('*')
const viewMode = computed<'flat' | 'tree'>({
  get: () => connectionStore.getViewMode(props.connId),
  set: (v) => connectionStore.setViewMode(props.connId, v),
})
const selectedKey = ref('')
const loading = ref(false)
const cursor = ref<number>(0)
const hasMore = ref(false)
const collapsedNodes = ref(new Set<string>())
const listRef = ref<HTMLElement>()
const panelRef = ref<HTMLElement>()
const searchInputRef = ref<HTMLInputElement>()
const separator = ref(':')
const showNewKey = ref(false)
const newKeyDefaultName = ref('')
const multiSelected = ref(new Set<string>())
const checkMode = ref(false)
const keySet = new Set<string>() // 持久化去重集合，避免每次 loadKeys 重建

// 每连接 key 列表状态缓存（存储在 store 中，disconnect 时自动清除）
interface KeyListState {
  keys: KeyItem[]
  keySetArr: string[]
  cursor: number
  hasMore: boolean
  selectedKey: string
  searchPattern: string
  currentDb: number
  collapsedNodesArr: string[]
  favoritesArr: string[]
  showFavoritesOnly: boolean
  memoryMap: Record<string, number>
  sortMode: 'name' | 'type' | 'ttl' | 'memory'
  sortOrder: 'asc' | 'desc'
  dbList: { db: number; keys: number }[]
  separator: string
}

function saveKeyListState(connId: string) {
  const state: KeyListState = {
    keys: [...keys.value],
    keySetArr: [...keySet],
    cursor: cursor.value,
    hasMore: hasMore.value,
    selectedKey: selectedKey.value,
    searchPattern: searchPattern.value,
    currentDb: currentDb.value,
    collapsedNodesArr: [...collapsedNodes.value],
    favoritesArr: [...favorites.value],
    showFavoritesOnly: showFavoritesOnly.value,
    memoryMap: { ...memoryMap.value },
    sortMode: sortMode.value,
    sortOrder: sortOrder.value,
    dbList: [...dbList.value],
    separator: separator.value,
  }
  connectionStore.saveKeyListStateCache(connId, state)
}

function restoreKeyListState(connId: string): boolean {
  const cached = connectionStore.getKeyListStateCache(connId) as KeyListState | undefined
  if (!cached) return false
  keys.value = cached.keys
  keySet.clear()
  cached.keySetArr.forEach(k => keySet.add(k))
  cursor.value = cached.cursor
  hasMore.value = cached.hasMore
  selectedKey.value = cached.selectedKey
  searchPattern.value = cached.searchPattern
  currentDb.value = cached.currentDb
  collapsedNodes.value = new Set(cached.collapsedNodesArr)
  favorites.value = new Set(cached.favoritesArr)
  showFavoritesOnly.value = cached.showFavoritesOnly
  memoryMap.value = cached.memoryMap
  sortMode.value = cached.sortMode
  sortOrder.value = cached.sortOrder
  dbList.value = cached.dbList
  separator.value = cached.separator
  return true
}

// 加载全部状态
const loadingAll = ref(false)
const lastClickedKey = ref('')
const ctxMenu = ref({ visible: false, x: 0, y: 0, key: '' })
const groupCtxMenu = ref({ visible: false, x: 0, y: 0, path: '' })
const showGroupDelete = ref(false)
const groupDeletePrefix = ref('')
const showExport = ref(false)
const showCheckExport = ref(false)
const showImport = ref(false)
const showBatch = ref(false)
const showMigrate = ref(false)
const showCrossDbSearch = ref(false)
const showBatchDeleteConfirm = ref(false)
const batchDeleteLoading = ref(false)
const sortMode = ref<'name' | 'type' | 'ttl' | 'memory'>('name')
const sortOrder = ref<'asc' | 'desc'>('asc')
const memoryMap = ref<Record<string, number>>({})
const favorites = ref(new Set<string>())
const showFavoritesOnly = ref(false)
const showMoreMenu = ref(false)
const moreDropdownRef = ref<HTMLElement>()

// Close more menu on outside click
function handleClickOutsideMore(e: MouseEvent) {
  if (showMoreMenu.value && moreDropdownRef.value && !moreDropdownRef.value.contains(e.target as Node)) {
    showMoreMenu.value = false
  }
}

// Sorted keys (with optional favorites filter)
const sortedKeys = computed(() => {
  let arr = [...keys.value]
  if (showFavoritesOnly.value) {
    arr = arr.filter(k => favorites.value.has(k.key))
  }
  const dir = sortOrder.value === 'asc' ? 1 : -1
  switch (sortMode.value) {
    case 'type':
      arr.sort((a, b) => dir * (a.type.localeCompare(b.type) || a.key.localeCompare(b.key)))
      break
    case 'ttl':
      arr.sort((a, b) => dir * (a.ttl - b.ttl) || a.key.localeCompare(b.key))
      break
    case 'memory':
      arr.sort((a, b) => dir * ((memoryMap.value[a.key] ?? 0) - (memoryMap.value[b.key] ?? 0)) || a.key.localeCompare(b.key))
      break
    default:
      arr.sort((a, b) => dir * a.key.localeCompare(b.key))
  }
  return arr
})

// Virtual scroll
const ITEM_HEIGHT = 32
const BUFFER = 10
const scrollTop = ref(0)
const containerHeight = ref(400)

const totalHeight = computed(() => {
  const count = viewMode.value === 'flat' ? sortedKeys.value.length : treeNodes.value.length
  return count * ITEM_HEIGHT
})

const visibleRange = computed(() => {
  const start = Math.max(0, Math.floor(scrollTop.value / ITEM_HEIGHT) - BUFFER)
  const visibleCount = Math.ceil(containerHeight.value / ITEM_HEIGHT) + BUFFER * 2
  const total = viewMode.value === 'flat' ? sortedKeys.value.length : treeNodes.value.length
  const end = Math.min(total, start + visibleCount)
  return { start, end }
})

const offsetY = computed(() => visibleRange.value.start * ITEM_HEIGHT)

const visibleFlatItems = computed(() => {
  const { start, end } = visibleRange.value
  return sortedKeys.value.slice(start, end)
})

const visibleTreeItems = computed(() => {
  const { start, end } = visibleRange.value
  return treeNodes.value.slice(start, end)
})

const ctxMenuItems = computed<MenuItem[]>(() => {
  const count = multiSelected.value.size
  const isFav = count <= 1 && favorites.value.has(ctxMenu.value.key)
  return [
    { key: 'favorite', label: isFav ? t('key.removeFavorite') : t('key.addFavorite'), icon: isFav ? '★' : '☆', disabled: count > 1 },
    { key: 'copy', label: t('common.copy'), icon: '⧉' },
    { key: 'copyAsCommand', label: t('key.copyAsCommand'), icon: '⌘', disabled: count > 1 },
    { key: 'rename', label: t('key.rename'), icon: '✎', disabled: count > 1 },
    { key: 'ttl', label: t('key.setTTL'), icon: '⏱', disabled: count > 1 },
    { key: 'delete', label: count > 1 ? `${t('common.delete')} (${count})` : t('common.delete'), icon: '✕', danger: true },
  ]
})

const groupCtxMenuItems = computed<MenuItem[]>(() => {
  const path = groupCtxMenu.value.path
  const isCollapsed = collapsedNodes.value.has(path)
  return [
    { key: 'toggle', label: isCollapsed ? t('key.expandGroup') : t('key.collapseGroup'), icon: isCollapsed ? '▼' : '▶' },
    { key: 'refresh', label: t('common.refresh'), icon: '↻' },
    { key: 'addKey', label: t('common.add'), icon: '+' },
    { key: 'export', label: t('key.exportGroup'), icon: '↗' },
    { key: 'delete', label: t('groupDelete.title'), icon: '✕', danger: true },
  ]
})

// 树形视图节点：从本地 keys 数组纯前端构建
interface GroupNode {
  children: Map<string, GroupNode>
  keys: KeyItem[]
  prefix: string
}

function buildGroupTree(items: KeyItem[], sep: string): GroupNode {
  const root: GroupNode = { children: new Map(), keys: [], prefix: '' }
  for (const item of items) {
    const parts = item.key.split(sep)
    if (parts.length <= 1) {
      root.keys.push(item)
    } else {
      let node = root
      for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i]
        if (!node.children.has(part)) {
          const childPrefix = node.prefix + part + sep
          node.children.set(part, { children: new Map(), keys: [], prefix: childPrefix })
        }
        node = node.children.get(part)!
      }
      node.keys.push(item)
    }
  }
  return root
}

function countGroupKeys(group: GroupNode): number {
  let count = group.keys.length
  for (const child of group.children.values()) count += countGroupKeys(child)
  return count
}

function flattenGroup(group: GroupNode, depth: number, nodes: TreeNode[]) {
  const dir = sortOrder.value === 'asc' ? 1 : -1
  // 分组在前，按 label 排序（跟随排序方向）
  const sortedGroups = [...group.children.entries()].sort((a, b) => dir * a[0].localeCompare(b[0]))
  for (const [label, child] of sortedGroups) {
    const count = countGroupKeys(child)
    if (count === 0) continue
    nodes.push({ path: child.prefix, label, depth, isGroup: true, count })
    if (!collapsedNodes.value.has(child.prefix)) {
      flattenGroup(child, depth + 1, nodes)
    }
  }
  // 叶子 key 在后；收藏筛选时过滤；跟随排序模式和方向
  let leaves = [...group.keys]
  if (showFavoritesOnly.value) {
    leaves = leaves.filter(k => favorites.value.has(k.key))
  }
  switch (sortMode.value) {
    case 'type':
      leaves.sort((a, b) => dir * (a.type.localeCompare(b.type) || a.key.localeCompare(b.key)))
      break
    case 'ttl':
      leaves.sort((a, b) => dir * (a.ttl - b.ttl) || a.key.localeCompare(b.key))
      break
    case 'memory':
      leaves.sort((a, b) => dir * ((memoryMap.value[a.key] ?? 0) - (memoryMap.value[b.key] ?? 0)) || a.key.localeCompare(b.key))
      break
    default:
      leaves.sort((a, b) => dir * a.key.localeCompare(b.key))
  }
  const prefixLen = group.prefix.length
  for (const item of leaves) {
    nodes.push({
      path: item.key,
      label: item.key.slice(prefixLen) || item.key,
      depth,
      isGroup: false,
      fullKey: item.key,
      type: item.type,
      ttl: item.ttl,
    })
  }
}

const treeNodes = computed(() => {
  if (viewMode.value !== 'tree') return []
  const root = buildGroupTree(sortedKeys.value, separator.value)
  const nodes: TreeNode[] = []
  flattenGroup(root, 0, nodes)
  return nodes
})

watch(() => props.connId, (_newId, oldId) => {
  connGeneration++
  // 保存旧连接的列表状态（仅当连接仍活跃时，避免 disconnect 清除缓存后又被存回）
  if (oldId && connectionStore.getConnState(oldId).status === 'connected') {
    saveKeyListState(oldId)
  }
  // 尝试从缓存恢复（校验 DB 一致性：断开重连后 DB 可能变化）
  const expectedDb = connState.value.currentDb ?? 0
  const cached = connectionStore.getKeyListStateCache(props.connId) as KeyListState | undefined
  if (cached && cached.currentDb === expectedDb && restoreKeyListState(props.connId)) {
    // 恢复成功，不需要重新 SCAN
    return
  }
  // 无缓存，正常初始化
  currentDb.value = connState.value.currentDb ?? 0
  keys.value = []
  keySet.clear()
  selectedKey.value = ''
  cursor.value = 0
  memoryMap.value = {}
  favorites.value = new Set()
  showFavoritesOnly.value = false
  collapsedNodes.value = new Set()

  loadDbList()
  refreshKeys()
  loadFavorites()
}, { immediate: true })


watch(sortMode, (mode) => {
  if (mode === 'memory') fetchMemoryUsage()
})

async function loadDbList() {
  const gen = connGeneration
  try {
    const res = await request<{ db: number; keys: number }[]>('get_db_list', {
      params: { conn_id: props.connId },
    })
    if (gen !== connGeneration) return
    dbList.value = res.data || []
  } catch (_e) {
    if (gen !== connGeneration) return
    // 默认 16 个 DB
    dbList.value = Array.from({ length: 16 }, (_, i) => ({ db: i, keys: 0 }))
  }
}

function handleDbPopupVisibleChange(visible: boolean) {
  dbPopupVisible.value = visible
  if (visible) {
    // 下拉展开后滚动到当前选中的 DB
    nextTick(() => {
      setTimeout(() => {
        const popup = document.querySelector('.arco-select-dropdown')
        const selected = popup?.querySelector('.arco-select-option-selected') as HTMLElement | null
        if (selected && popup) {
          selected.scrollIntoView({ block: 'center' })
        }
      }, 50)
    })
  }
}

async function handleDbChange() {
  try {
    await request('select_db', { params: { conn_id: props.connId, db: currentDb.value } })
    connectionStore.setCurrentDb(props.connId, currentDb.value)
    keys.value = []
    keySet.clear()
    collapsedNodes.value = new Set()
    cursor.value = 0
    await refreshKeys()
    loadFavorites()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

// 收集当前 keys 中所有分组前缀，用于默认折叠
function collectGroupPrefixes(): Set<string> {
  const sep = separator.value
  const prefixes = new Set<string>()
  for (const k of keys.value) {
    const parts = k.key.split(sep)
    if (parts.length > 1) {
      let prefix = ''
      for (let i = 0; i < parts.length - 1; i++) {
        prefix += parts[i] + sep
        prefixes.add(prefix)
      }
    }
  }
  return prefixes
}

async function refreshKeys() {
  loading.value = true
  // 记住刷新前已展开的分组（即不在 collapsedNodes 中的），必须在清空 keys 之前
  const prevExpanded = new Set<string>()
  for (const prefix of collectGroupPrefixes()) {
    if (!collapsedNodes.value.has(prefix)) prevExpanded.add(prefix)
  }
  cursor.value = 0
  keys.value = []
  keySet.clear()
  multiSelected.value = new Set()
  try {
    await loadKeys()
    // 刷新后：所有新分组默认折叠，但保留之前已展开的
    const allPrefixes = collectGroupPrefixes()
    const newCollapsed = new Set<string>()
    for (const prefix of allPrefixes) {
      if (!prevExpanded.has(prefix)) newCollapsed.add(prefix)
    }
    collapsedNodes.value = newCollapsed
    loadDbList()
  } finally {
    loading.value = false
  }
}

async function loadKeys() {
  const gen = connGeneration
  try {
    const connCfg = connectionStore.connections.find(c => c.id === props.connId)
    const scanCount = connCfg?.scan_count || settingsStore.settings.scanCount || 200
    const res = await request<{ keys: KeyItem[]; cursor: number }>('scan_keys', {
      params: {
        conn_id: props.connId,
        pattern: searchPattern.value || '*',
        count: scanCount,
        cursor: cursor.value,
      },
    })
    if (gen !== connGeneration) return // connId 已切换，丢弃过期结果
    if (res.data) {
      const newKeys = (res.data.keys || []).filter((k: KeyItem) => !keySet.has(k.key))
      for (const k of newKeys) keySet.add(k.key)
      keys.value = [...keys.value, ...newKeys]
      cursor.value = res.data.cursor
      hasMore.value = res.data.cursor !== 0
    }
  } catch (e: any) {
    if (gen !== connGeneration) return
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function loadMore() {
  if (!hasMore.value || loading.value) return
  loading.value = true
  // 记住当前已展开的分组
  const prevExpanded = new Set<string>()
  for (const prefix of collectGroupPrefixes()) {
    if (!collapsedNodes.value.has(prefix)) prevExpanded.add(prefix)
  }
  try {
    await loadKeys()
    // 新分组默认折叠，保留之前已展开的
    const allPrefixes = collectGroupPrefixes()
    const newCollapsed = new Set<string>()
    for (const prefix of allPrefixes) {
      if (!prevExpanded.has(prefix)) newCollapsed.add(prefix)
    }
    collapsedNodes.value = newCollapsed
  } finally {
    loading.value = false
  }
}

async function loadAllKeys() {
  if (!hasMore.value || loadingAll.value) return
  loadingAll.value = true
  loading.value = true
  // 记住当前已展开的分组
  const prevExpanded = new Set<string>()
  for (const prefix of collectGroupPrefixes()) {
    if (!collapsedNodes.value.has(prefix)) prevExpanded.add(prefix)
  }
  try {
    while (hasMore.value) {
      await loadKeys()
      // 每批加载后立即折叠新分组，避免抖动
      const allPrefixes = collectGroupPrefixes()
      const updated = new Set<string>()
      for (const prefix of allPrefixes) {
        if (!prevExpanded.has(prefix)) updated.add(prefix)
      }
      collapsedNodes.value = updated
    }
  } finally {
    loadingAll.value = false
    loading.value = false
  }
}

async function refreshGroup(prefix: string) {
  loading.value = true
  try {
    // 移除旧的属于该分组的键
    for (const k of keys.value) {
      if (k.key.startsWith(prefix)) keySet.delete(k.key)
    }
    const otherKeys = keys.value.filter(k => !k.key.startsWith(prefix))

    // 用 scan_keys 循环扫描该前缀下的所有键（含 type/ttl）
    const connCfg = connectionStore.connections.find(c => c.id === props.connId)
    const scanCount = connCfg?.scan_count || settingsStore.settings.scanCount || 200
    const newKeyItems: KeyItem[] = []
    let groupCursor = 0
    do {
      const res = await request<{ keys: KeyItem[]; cursor: number }>('scan_keys', {
        params: {
          conn_id: props.connId,
          pattern: prefix + '*',
          count: scanCount,
          cursor: groupCursor,
        },
      })
      if (res.data) {
        for (const k of res.data.keys || []) {
          if (!keySet.has(k.key)) {
            keySet.add(k.key)
            newKeyItems.push(k)
          }
        }
        groupCursor = res.data.cursor
      } else {
        break
      }
    } while (groupCursor !== 0)

    keys.value = [...otherKeys, ...newKeyItems]
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function fetchMemoryUsage() {
  if (keys.value.length === 0) return
  loading.value = true
  try {
    const keyNames = keys.value.map(k => k.key)
    const res = await request<Record<string, number>>('get_keys_memory', {
      params: { conn_id: props.connId, keys: keyNames },
    })
    if (res.data) memoryMap.value = res.data
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function loadFavorites() {
  const gen = connGeneration
  try {
    const res = await request<string[]>('get_favorites', {
      params: { conn_id: props.connId, db: currentDb.value },
    })
    if (gen !== connGeneration) return
    favorites.value = new Set(res.data || [])
  } catch (_e) {
    if (gen !== connGeneration) return
    favorites.value = new Set()
  }
}

async function toggleFavorite(key: string) {
  try {
    const res = await request<{ added: boolean; favorites: string[] }>('toggle_favorite', {
      params: { conn_id: props.connId, db: currentDb.value, key },
    })
    if (res.data) {
      favorites.value = new Set(res.data.favorites)
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function handleScroll() {
  if (!listRef.value) return
  scrollTop.value = listRef.value.scrollTop
  containerHeight.value = listRef.value.clientHeight
}

function scrollToIndex(index: number) {
  if (!listRef.value) return
  const top = index * ITEM_HEIGHT
  const bottom = top + ITEM_HEIGHT
  const viewTop = listRef.value.scrollTop
  const viewBottom = viewTop + listRef.value.clientHeight
  if (top < viewTop) {
    listRef.value.scrollTop = top
  } else if (bottom > viewBottom) {
    listRef.value.scrollTop = bottom - listRef.value.clientHeight
  }
}

function selectKey(item: KeyItem) {
  selectedKey.value = item.key
  emit('selectKey', item.key, item.type)
}

function handleItemClick(e: MouseEvent, item: KeyItem) {
  if (e.shiftKey && lastClickedKey.value) {
    // Shift+Click: range select
    const allKeys = sortedKeys.value.map(k => k.key)
    const fromIdx = allKeys.indexOf(lastClickedKey.value)
    const toIdx = allKeys.indexOf(item.key)
    if (fromIdx !== -1 && toIdx !== -1) {
      const start = Math.min(fromIdx, toIdx)
      const end = Math.max(fromIdx, toIdx)
      const s = new Set(multiSelected.value)
      for (let i = start; i <= end; i++) s.add(allKeys[i])
      multiSelected.value = s
    }
    return
  }
  if (e.ctrlKey || e.metaKey) {
    // Ctrl/Cmd+click: toggle multi-select
    const s = new Set(multiSelected.value)
    if (s.has(item.key)) s.delete(item.key)
    else s.add(item.key)
    multiSelected.value = s
  } else {
    multiSelected.value = new Set()
    selectKey(item)
  }
  lastClickedKey.value = item.key
}

function openContextMenu(e: MouseEvent, item: KeyItem) {
  // If right-clicked item is not in multi-select, select it alone
  if (!multiSelected.value.has(item.key)) {
    multiSelected.value = new Set([item.key])
  }
  groupCtxMenu.value.visible = false
  ctxMenu.value = { visible: true, x: e.clientX, y: e.clientY, key: item.key }
  selectKey(item)
}

function openGroupContextMenu(e: MouseEvent, node: TreeNode) {
  ctxMenu.value.visible = false
  groupCtxMenu.value = { visible: true, x: e.clientX, y: e.clientY, path: node.path }
}

async function handleGroupCtxAction(action: string) {
  const path = groupCtxMenu.value.path
  await doGroupAction(action, path)
  groupCtxMenu.value.visible = false
}

function handleGroupHoverAction(action: string, path: string) {
  doGroupAction(action, path)
}

async function doGroupAction(action: string, path: string) {
  if (action === 'toggle') {
    toggleTreeNode(path)
  } else if (action === 'refresh') {
    refreshGroup(path)
  } else if (action === 'addKey') {
    newKeyDefaultName.value = path
    showNewKey.value = true
  } else if (action === 'export') {
    const groupKeys = keys.value.filter(k => k.key.startsWith(path)).map(k => k.key)
    multiSelected.value = new Set(groupKeys)
    showCheckExport.value = true
  } else if (action === 'delete') {
    groupDeletePrefix.value = path
    showGroupDelete.value = true
  }
}

async function handleCtxAction(action: string) {
  const targetKeys = multiSelected.value.size > 0 ? [...multiSelected.value] : [ctxMenu.value.key]
  const firstKey = targetKeys[0]

  switch (action) {
    case 'favorite':
      await toggleFavorite(firstKey)
      break
    case 'copy':
      try {
        await navigator.clipboard.writeText(targetKeys.join('\n'))
        showMessage('success', t('common.copySuccess'))
      } catch (_e) { /* ignore */ }
      break
    case 'rename': {
      const newName = await gmPrompt(t('key.renamePrompt'), firstKey)
      if (!newName || newName === firstKey) return
      try {
        await request('rename_key', { params: { conn_id: props.connId, key: firstKey, new_key: newName } })
        showMessage('success', t('common.success'))
        refreshKeys()
      } catch (e: any) { showMessage('error', e?.message || t('common.failed')) }
      break
    }
    case 'ttl': {
      const input = await gmPrompt(t('key.ttlPrompt'), '-1')
      if (input === null) return
      const ttl = parseInt(input, 10)
      if (isNaN(ttl)) return
      try {
        await request('set_ttl', { params: { conn_id: props.connId, key: firstKey, ttl } })
        showMessage('success', t('common.success'))
        refreshKeys()
      } catch (e: any) { showMessage('error', e?.message || t('common.failed')) }
      break
    }
    case 'copyAsCommand': {
      try {
        const res = await request<string>('copy_as_command', { params: { conn_id: props.connId, key: firstKey } })
        if (res.data) {
          await navigator.clipboard.writeText(res.data)
          showMessage('success', t('common.copySuccess'))
        }
      } catch (e: any) { showMessage('error', e?.message || t('common.failed')) }
      break
    }
    case 'delete':
      if (!await gmConfirm(t('key.deleteConfirm'))) return
      try {
        await request('delete_keys', { params: { conn_id: props.connId, keys: targetKeys } })
        showMessage('success', t('common.success'))
        keys.value = keys.value.filter(k => !targetKeys.includes(k.key))
        for (const k of targetKeys) keySet.delete(k)
        multiSelected.value = new Set()
        if (targetKeys.includes(selectedKey.value)) {
          selectedKey.value = ''
          emit('deleted', targetKeys[0])
        }
        loadDbList()
      } catch (e: any) { showMessage('error', e?.message || t('common.failed')) }
      break
  }
}

async function handleFlushDb() {
  const input = await gmPrompt(t('key.flushDbConfirm'), '', 'FLUSHDB')
  if (input === null) return // 用户取消
  if (input !== 'FLUSHDB') {
    showMessage('error', t('key.flushDbInputError'))
    return
  }
  try {
    await request('flush_db', { params: { conn_id: props.connId } })
    showMessage('success', t('key.flushDbSuccess'))
    keys.value = []
    keySet.clear()
    selectedKey.value = ''
    multiSelected.value = new Set()
    loadDbList()
  } catch (e: any) { showMessage('error', e?.message || t('common.failed')) }
}

function toggleCheckMode() {
  checkMode.value = !checkMode.value
  if (!checkMode.value) multiSelected.value = new Set()
}

function toggleCheck(key: string) {
  const s = new Set(multiSelected.value)
  if (s.has(key)) s.delete(key)
  else s.add(key)
  multiSelected.value = s
}

function getGroupKeys(groupPath: string): string[] {
  return keys.value
    .filter(k => k.key.startsWith(groupPath))
    .map(k => k.key)
}

function isGroupAllChecked(groupPath: string): boolean {
  const groupKeys = getGroupKeys(groupPath)
  return groupKeys.length > 0 && groupKeys.every(k => multiSelected.value.has(k))
}

function isGroupPartialChecked(groupPath: string): boolean {
  const groupKeys = getGroupKeys(groupPath)
  const checkedCount = groupKeys.filter(k => multiSelected.value.has(k)).length
  return checkedCount > 0 && checkedCount < groupKeys.length
}

function toggleGroupCheck(groupPath: string) {
  const groupKeys = getGroupKeys(groupPath)
  const s = new Set(multiSelected.value)
  if (isGroupAllChecked(groupPath)) {
    groupKeys.forEach(k => s.delete(k))
  } else {
    groupKeys.forEach(k => s.add(k))
  }
  multiSelected.value = s
}


async function batchSetTTLChecked() {
  const targetKeys = [...multiSelected.value]
  if (!targetKeys.length) return
  const input = await gmPrompt(t('key.ttlPrompt'), '-1')
  if (input === null || input === undefined) return
  const ttl = parseInt(String(input), 10)
  if (isNaN(ttl)) return
  try {
    await request('batch_set_ttl', { params: { conn_id: props.connId, keys: targetKeys, ttl } })
    showMessage('success', t('common.success'))
    // 更新本地 key 的 ttl
    for (const k of keys.value) {
      if (targetKeys.includes(k.key)) k.ttl = ttl
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function batchDeleteChecked() {
  const targetKeys = [...multiSelected.value]
  if (!targetKeys.length) return
  showBatchDeleteConfirm.value = true
}

async function confirmBatchDelete() {
  const targetKeys = [...multiSelected.value]
  if (!targetKeys.length) return
  batchDeleteLoading.value = true
  try {
    await request('delete_keys', { params: { conn_id: props.connId, keys: targetKeys } })
    showMessage('success', t('common.success'))
    keys.value = keys.value.filter(k => !targetKeys.includes(k.key))
    for (const k of targetKeys) keySet.delete(k)
    multiSelected.value = new Set()
    if (targetKeys.includes(selectedKey.value)) {
      selectedKey.value = ''
      emit('deleted', targetKeys[0])
    }
    loadDbList()
    showBatchDeleteConfirm.value = false
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    batchDeleteLoading.value = false
  }
}

function toggleView() {
  viewMode.value = viewMode.value === 'flat' ? 'tree' : 'flat'
}

function toggleTreeNode(path: string) {
  if (collapsedNodes.value.has(path)) {
    collapsedNodes.value.delete(path)
  } else {
    collapsedNodes.value.add(path)
  }
}

function typeLabel(type: string): string {
  const map: Record<string, string> = {
    string: 'S', list: 'L', set: 'E', zset: 'Z',
    hash: 'H', stream: 'X', geo: 'G', 'ReJSON-RL': 'J', none: '?',
  }
  return map[type] || type[0]?.toUpperCase() || '?'
}

function ttlClass(ttl: number): string {
  if (ttl === -1) return 'permanent'
  if (ttl <= 60) return 'danger'
  if (ttl <= 3600) return 'warning'
  return 'safe'
}

function formatTTL(ttl: number): string {
  if (ttl === -1) return '∞'
  if (ttl === -2) return '-'
  if (ttl < 60) return ttl + 's'
  if (ttl < 3600) return Math.floor(ttl / 60) + 'm'
  if (ttl < 86400) return Math.floor(ttl / 3600) + 'h'
  return Math.floor(ttl / 86400) + 'd'
}

function handleKeydown(e: KeyboardEvent) {
  const target = e.target as HTMLElement
  // Don't intercept when typing in input/select
  if (target.tagName === 'INPUT' || target.tagName === 'SELECT' || target.tagName === 'TEXTAREA') return

  if (matchesAction(e, 'deleteKey')) {
    if (!selectedKey.value) return
    e.preventDefault()
    const targetKeys = multiSelected.value.size > 0 ? [...multiSelected.value] : [selectedKey.value]
    gmConfirm(t('key.deleteConfirm')).then(ok => {
      if (!ok) return
      request('delete_keys', { params: { conn_id: props.connId, keys: targetKeys } }).then(() => {
        keys.value = keys.value.filter(k => !targetKeys.includes(k.key))
        multiSelected.value = new Set()
        if (targetKeys.includes(selectedKey.value)) {
          selectedKey.value = ''
          emit('deleted', targetKeys[0])
        }
      })
    })
    return
  }
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (viewMode.value === 'flat') {
      const list = sortedKeys.value
      if (!list.length) return
      const currentIdx = list.findIndex(k => k.key === selectedKey.value)
      let nextIdx: number
      if (e.key === 'ArrowDown') {
        nextIdx = currentIdx < list.length - 1 ? currentIdx + 1 : 0
      } else {
        nextIdx = currentIdx > 0 ? currentIdx - 1 : list.length - 1
      }
      selectKey(list[nextIdx])
      scrollToIndex(nextIdx)
    } else {
      // 树形视图：只在叶子节点间导航
      const leaves = treeNodes.value
        .map((n, i) => ({ node: n, index: i }))
        .filter(x => !x.node.isGroup)
      if (!leaves.length) return
      const curLeafIdx = leaves.findIndex(x => x.node.fullKey === selectedKey.value)
      let nextLeafIdx: number
      if (e.key === 'ArrowDown') {
        nextLeafIdx = curLeafIdx < leaves.length - 1 ? curLeafIdx + 1 : 0
      } else {
        nextLeafIdx = curLeafIdx > 0 ? curLeafIdx - 1 : leaves.length - 1
      }
      const leaf = leaves[nextLeafIdx].node
      selectKey({ key: leaf.fullKey!, type: leaf.type!, ttl: leaf.ttl! })
      scrollToIndex(leaves[nextLeafIdx].index)
    }
    return
  }
}

// 检查面板是否可见（v-show 控制）
function isPanelVisible() {
  return panelRef.value?.offsetParent !== null
}

// 监听全局快捷键 Ctrl+F → 聚焦搜索框
function onShortcutSearch() {
  if (!isPanelVisible()) return
  searchInputRef.value?.focus()
  searchInputRef.value?.select()
}

// 监听全局快捷键 Ctrl+N → 新建 Key
function onShortcutNewKey() {
  if (!isPanelVisible()) return
  showNewKey.value = true
}

// 监听全局快捷键 Ctrl+R / F5 → 刷新列表
function onShortcutRefresh() {
  if (!isPanelVisible()) return
  refreshKeys()
}

onMounted(() => {
  document.addEventListener('shortcut:search', onShortcutSearch)
  document.addEventListener('shortcut:newKey', onShortcutNewKey)
  document.addEventListener('shortcut:refresh', onShortcutRefresh)
  document.addEventListener('click', handleClickOutsideMore)
})
onBeforeUnmount(() => {
  // 无需额外操作，keys 数据在 store 中持久化
})
onUnmounted(() => {
  document.removeEventListener('shortcut:search', onShortcutSearch)
  document.removeEventListener('shortcut:newKey', onShortcutNewKey)
  document.removeEventListener('shortcut:refresh', onShortcutRefresh)
  document.removeEventListener('click', handleClickOutsideMore)
})

function updateKeyTTL(key: string, ttl: number) {
  const item = keys.value.find(k => k.key === key)
  if (item) item.ttl = ttl
}

function removeKey(key: string) {
  keys.value = keys.value.filter(k => k.key !== key)
  keySet.delete(key)
  if (selectedKey.value === key) {
    selectedKey.value = ''
  }
  loadDbList()
}

defineExpose({ refreshKeys, updateKeyTTL, removeKey })

async function handleCrossDbJump(db: number, key: string, type: string) {
  if (db !== currentDb.value) {
    currentDb.value = db
    await handleDbChange()
  }
  // 等列表加载后选中目标 key
  selectedKey.value = key
  emit('selectKey', key, type)
}

function onKeyCreated(key: string, type: string) {
  // 直接插入到本地列表，无需全量刷新
  const redisType: Record<string, string> = { bitmap: 'string', hll: 'string', geo: 'zset' }
  const mappedType = redisType[type] || type
  const newItem: KeyItem = { key, type: mappedType, ttl: -1 }
  if (!keySet.has(key)) {
    keySet.add(key)
    keys.value = [...keys.value, newItem]
  }
  // 树形视图下确保新 key 所在分组展开
  if (viewMode.value === 'tree') {
    const sep = separator.value
    const parts = key.split(sep)
    if (parts.length > 1) {
      let prefix = ''
      for (let i = 0; i < parts.length - 1; i++) {
        prefix += parts[i] + sep
        collapsedNodes.value.delete(prefix)
      }
    }
  }
  loadDbList()
  // 如果新建的 key 与当前选中的同名，清缓存并强制刷新详情
  if (selectedKey.value === key) {
    connectionStore.saveKeyDetailCache(props.connId, key, null)
    emit('refreshKey', key)
  } else {
    selectKey(newItem)
  }
}

</script>

<style scoped>
@import './collections/collection.css';

.preview-header {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  font-weight: 600;
  margin-bottom: var(--spacing-xs);
}

.preview-list {
  max-height: 240px;
  overflow-y: auto;
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  padding: var(--spacing-xs);
}

.preview-item {
  padding: 3px var(--spacing-sm);
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border-radius: var(--radius-sm);
  transition: background var(--transition-fast);
}

.preview-item:hover {
  background: var(--color-fill-2);
}




.key-list-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  border-right: 1px solid var(--color-border-2);
  background: var(--color-bg-1);
  outline: none;
}

.key-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-sm);
  border-bottom: 1px solid var(--color-border-1);
}

.db-select-bottom {
  max-width: 120px;
  font-size: var(--font-size-xs);
}

.cluster-label-bottom {
  padding: 0 var(--spacing-xs);
  color: var(--color-primary);
  font-size: var(--font-size-xs);
  font-weight: 600;
}

.toolbar-spacer {
  flex: 1;
}

.toolbar-separator {
  width: 1px;
  height: 16px;
  background: var(--color-border-2);
  flex-shrink: 0;
}

.more-dropdown {
  position: relative;
  flex-shrink: 0;
}

.more-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 4px;
  min-width: 180px;
  background: var(--color-bg-popup);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: var(--spacing-xs) 0;
  z-index: 100;
}

.more-menu-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md);
  background: none;
  border: none;
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: background var(--transition-fast);
  white-space: nowrap;
}

.more-menu-item:hover {
  background: var(--color-fill-1);
}

.more-menu-item.danger {
  color: var(--color-error);
}

.more-menu-item.danger:hover {
  background: var(--color-error-bg);
}

.more-menu-divider {
  height: 1px;
  margin: var(--spacing-xs) 0;
  background: var(--color-border-1);
}

.more-menu-group {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
}

.more-menu-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  white-space: nowrap;
}

.sort-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.more-menu-select {
  flex: 1;
  height: 26px;
  padding: 0 var(--spacing-xs);
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  outline: none;
}

.sort-order-btn {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-1);
  font-size: var(--font-size-md);
  font-weight: 600;
  cursor: pointer;
  flex-shrink: 0;
}

.sort-order-btn:hover {
  background: var(--color-fill-2);
  color: var(--color-primary);
}

.more-menu-item.active {
  color: var(--color-warning);
}

.key-toolbar .arco-btn {
  color: var(--color-text-3);
}
.key-toolbar .arco-btn:hover {
  color: var(--color-text-1);
}

.key-search {
  padding: var(--spacing-xs) var(--spacing-sm);
  border-bottom: 1px solid var(--color-border-1);
}


.key-list {
  flex: 1;
  overflow-y: auto;
}

.virtual-spacer {
  position: relative;
}

.virtual-window {
  position: absolute;
  left: 0;
  right: 0;
}

.key-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: 7px var(--spacing-sm);
  cursor: pointer;
  font-size: var(--font-size-sm);
  transition: background var(--transition-fast);
}

.key-item:hover {
  background: var(--color-fill-1);
}

.key-item.active {
  background: var(--color-primary-bg);
}

.key-item.multi-selected {
  background: var(--color-fill-2);
}

.key-item.multi-selected.active {
  background: var(--color-primary-bg);
}

.type-badge {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 600;
  flex-shrink: 0;
  color: #fff;
}

.type-badge.string { background: #4080ff; }
.type-badge.list { background: #00b42a; }
.type-badge.set { background: #ff7d00; }
.type-badge.zset { background: #f53f3f; }
.type-badge.hash { background: #722ed1; }
.type-badge.stream { background: #0fc6c2; }
.type-badge.geo { background: #13c2c2; }
.type-badge.ReJSON-RL { background: #e8590c; }

.key-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
}

.key-ttl {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
}

.key-ttl.permanent { color: var(--color-ttl-permanent); }
.key-ttl.danger { color: var(--color-ttl-danger); }
.key-ttl.warning { color: var(--color-ttl-warning); }
.key-ttl.safe { color: var(--color-ttl-safe); }

.fav-star {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  cursor: pointer;
  color: var(--color-text-4);
  opacity: 0;
  transition: opacity var(--transition-fast), color var(--transition-fast);
}

.fav-star.active {
  color: var(--color-warning);
  opacity: 1;
}

.key-item:hover .fav-star {
  opacity: 1;
}

.key-toolbar .arco-btn.active {
  background: var(--color-primary-bg);
  color: var(--color-warning);
  border-color: var(--color-border-2);
}

.tree-group {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: 7px var(--spacing-sm);
  cursor: pointer;
  font-size: var(--font-size-sm);
  color: var(--color-text-2);
  user-select: none;
}

.tree-group:hover {
  background: var(--color-fill-1);
}

.tree-arrow {
  font-size: var(--font-size-xs);
  transition: transform var(--transition-fast);
  transform: rotate(90deg) scale(0.75);
  transform-origin: center center;
}

.tree-arrow.collapsed {
  transform: rotate(0deg) scale(0.75);
}

.tree-group-name {
  font-weight: 500;
}

.tree-count {
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
}

.status-bar-spacer {
  flex: 1;
}

.tree-group-actions {
  display: none;
  margin-left: auto;
  gap: 2px;
  flex-shrink: 0;
}

.tree-group:hover .tree-group-actions {
  display: flex;
}

.tree-action-btn {
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-fill-2);
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  line-height: 18px;
  text-align: center;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tree-action-btn:hover {
  background: var(--color-fill-3);
  color: var(--color-text-1);
}

.tree-action-btn.danger:hover {
  background: var(--color-danger-light, rgba(220, 53, 69, 0.12));
  color: var(--color-danger);
}

.key-loading,
.key-empty {
  padding: var(--spacing-lg);
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
}

.key-status-bar {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: 2px var(--spacing-sm);
  border-top: 1px solid var(--color-border-1);
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}




.key-checkbox {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  cursor: pointer;
  accent-color: var(--color-primary);
}

.check-action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px var(--spacing-sm);
  border-top: 1px solid var(--color-border-1);
  background: var(--color-fill-1);
  flex-shrink: 0;
}

.check-count {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
}

.check-actions {
  display: flex;
  gap: 4px;
}


</style>

<style>
/* DB selector dropdown: enhance selected option visibility */
.db-select-bottom .arco-select-dropdown .arco-select-option-selected,
.arco-select-dropdown .arco-select-option-selected {
  color: var(--color-primary) !important;
  font-weight: 600 !important;
  background-color: var(--color-primary-bg) !important;
}
</style>
