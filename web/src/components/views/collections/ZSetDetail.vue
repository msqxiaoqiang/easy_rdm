<template>
  <div class="collection-detail">
    <div class="collection-toolbar">
      <input
        v-model="pattern"
        class="filter-input"
        :placeholder="$t('collection.filterMembers')"
        @keyup.enter="searchMode = true; resetAndLoad()"
      />
      <a-button size="mini" @click="searchMode = true; resetAndLoad()"><template #icon><IconSearch :size="14" /></template></a-button>
      <a-button v-if="searchMode" size="mini" @click="searchMode = false; pattern = ''; resetAndLoad()"><IconClose :size="12" /> {{ $t('common.reset') }}</a-button>
      <div class="toolbar-spacer"></div>
      <div class="zset-order-toggle">
        <button class="order-btn" :class="{ active: !rev }" @click="rev = false; resetAndLoad()">ASC</button>
        <button class="order-btn" :class="{ active: rev }" @click="rev = true; resetAndLoad()">DESC</button>
      </div>
      <a-button size="mini" type="primary" @click="openAdd"><IconPlus :size="12" /> {{ $t('collection.addMember') }}</a-button>
      <a-button size="mini" status="danger" @click="removeSelected" :disabled="!selected.length">
        {{ $t('common.delete') }} ({{ selected.length }})
      </a-button>
    </div>

    <div class="collection-table-wrap">
      <table class="collection-table">
        <thead>
          <tr>
            <th class="col-check"><input type="checkbox" @change="toggleAll" :checked="allChecked" /></th>
            <th>{{ $t('collection.member') }}</th>
            <th class="col-score">{{ $t('collection.score') }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.member" :class="{ selected: selected.includes(item.member) }">
            <td class="col-check"><input type="checkbox" :checked="selected.includes(item.member)" @change="toggleSelect(item.member)" /></td>
            <td class="mono"><span class="cell-text" v-ellipsis-tip>{{ item.member }}</span></td>
            <td class="col-score">
              <template v-if="editingMember === item.member">
                <input v-model="editingScore" class="inline-edit" type="number" step="any" @keyup.enter="saveScore(item)" @keyup.escape="editingMember = ''" />
              </template>
              <template v-else>
                <span class="cell-text score-cell" v-ellipsis-tip @dblclick="startEditScore(item)">{{ item.score }}</span>
              </template>
            </td>
            <td class="col-actions">
              <template v-if="editingMember === item.member">
                <button class="mini-btn" :data-tooltip="$t('common.confirm')" data-tooltip-pos="bottom" @click="saveScore(item)"><IconCheck :size="14" /></button>
                <button class="mini-btn" :data-tooltip="$t('common.cancel')" data-tooltip-pos="bottom" @click="editingMember = ''"><IconClose :size="14" /></button>
              </template>
              <template v-else>
                <button class="mini-btn" :data-tooltip="$t('common.copy')" data-tooltip-pos="bottom" @click="copyRow(item.member)"><IconCopy :size="14" /></button>
                <button class="mini-btn danger" :data-tooltip="$t('common.delete')" data-tooltip-pos="bottom" @click="removeMembers([item.member])"><IconDelete :size="14" /></button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!items.length && !loading" class="empty-hint">{{ $t('common.noData') }}</div>
    </div>

    <div class="collection-footer">
      <span class="footer-info">{{ items.length }} / {{ total }}</span>
      <template v-if="searchMode">
        <a-button v-if="scanCursor !== 0" size="mini" @click="loadMoreScan">{{ $t('collection.loadMore') }}</a-button>
      </template>
      <template v-else>
        <div class="page-nav">
          <a-button size="mini" :disabled="page === 0" @click="page--; loadPage()">‹</a-button>
          <span>{{ page + 1 }} / {{ totalPages }}</span>
          <a-button size="mini" :disabled="page >= totalPages - 1" @click="page++; loadPage()">›</a-button>
        </div>
      </template>
    </div>

    <!-- Add Dialog -->
    <a-modal :visible="showAdd" :title="$t('collection.addMember')" :width="520" unmount-on-close @cancel="showAdd = false">
      <div class="zset-add-body">
        <div class="zset-mode-toggle">
          <span class="zset-mode-label">{{ $t('collection.existMode') }}</span>
          <div class="zset-mode-options">
            <button :class="['mode-btn', { active: addMode === '' }]" @click="addMode = ''">{{ $t('collection.overwrite') }}</button>
            <button :class="['mode-btn', { active: addMode === 'nx' }]" @click="addMode = 'nx'">{{ $t('collection.ignore') }}</button>
          </div>
        </div>
        <div class="zset-add-list">
          <div v-for="(row, idx) in newMembers" :key="idx" class="zset-add-item">
            <span class="zset-add-idx">{{ idx + 1 }}</span>
            <a-input v-model="row.member" :placeholder="$t('collection.member')" />
            <a-input v-model="row.score" class="zset-score-input" type="number" step="any" :placeholder="$t('collection.score')" />
            <button v-if="newMembers.length > 1" class="zset-add-remove" @click="newMembers.splice(idx, 1)"><IconClose :size="12" /></button>
          </div>
        </div>
        <button class="zset-add-more-btn" @click="newMembers.push({ member: '', score: '0' })"><IconPlus :size="12" /> {{ $t('collection.addMember') }}</button>
      </div>
      <template #footer>
        <a-button @click="showAdd = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="addMembers">{{ $t('common.confirm') }}</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onBeforeUnmount } from 'vue'
import { request } from '../../../utils/request'
import { useConnectionStore } from '../../../stores/connection'
import { useI18n } from 'vue-i18n'
import { gmConfirm } from '../../../utils/dialog'
import { showMessage } from '../../../utils/platform'
import { IconSearch, IconClose, IconCopy, IconDelete, IconCheck, IconPlus } from '@arco-design/web-vue/es/icon'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

interface ZItem { member: string; score: number }

const items = ref<ZItem[]>([])
const total = ref(0)
const page = ref(0)
const pageSize = 100
const rev = ref(false)
const loading = ref(false)
const selected = ref<string[]>([])

// Search mode (ZSCAN)
const searchMode = ref(false)
const pattern = ref('')
const scanCursor = ref<number>(0)

// Score inline edit
const editingMember = ref('')
const editingScore = ref('0')

// Add dialog
const showAdd = ref(false)
const addMode = ref('')  // '' = overwrite, 'nx' = ignore existing
const newMembers = ref<{ member: string; score: string }[]>([{ member: '', score: '0' }])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const allChecked = computed(() => items.value.length > 0 && selected.value.length === items.value.length)

function saveZsetCache() {
  connectionStore.saveKeyDetailCache(props.connId, `zset:${props.keyName}`, {
    items: [...items.value], total: total.value, page: page.value,
    rev: rev.value, selected: [...selected.value], searchMode: searchMode.value,
    pattern: pattern.value, scanCursor: scanCursor.value,
  })
}

onBeforeUnmount(() => { saveZsetCache() })

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `zset:${oldKey}`, {
    items: [...items.value], total: total.value, page: page.value,
    rev: rev.value, selected: [...selected.value], searchMode: searchMode.value,
    pattern: pattern.value, scanCursor: scanCursor.value,
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `zset:${newKey}`)
  if (cached) {
    items.value = cached.items; total.value = cached.total; page.value = cached.page
    rev.value = cached.rev; selected.value = cached.selected
    searchMode.value = cached.searchMode; pattern.value = cached.pattern
    scanCursor.value = cached.scanCursor
    return
  }
  resetAndLoad()
}, { immediate: true })

async function resetAndLoad() {
  items.value = []
  selected.value = []
  page.value = 0
  scanCursor.value = 0
  if (searchMode.value && pattern.value) {
    await loadMoreScan()
  } else {
    searchMode.value = false
    await loadPage()
  }
}

async function loadPage() {
  loading.value = true
  const start = page.value * pageSize
  const stop = start + pageSize - 1
  try {
    const res = await request<any>('zrange_members', {
      params: { conn_id: props.connId, key: props.keyName, start, stop, rev: rev.value },
    })
    if (res.data) {
      items.value = res.data.members || []
      total.value = res.data.total ?? 0
    }
  } catch (_e) { /* 加载失败静默 */ }
  loading.value = false
}

async function loadMoreScan() {
  loading.value = true
  try {
    const res = await request<any>('zscan_members', {
      params: { conn_id: props.connId, key: props.keyName, pattern: pattern.value || '*', cursor: scanCursor.value, count: 100 },
    })
    if (res.data) {
      const existing = new Set(items.value.map(i => i.member))
      const newItems = (res.data.members || []).filter((i: ZItem) => !existing.has(i.member))
      items.value.push(...newItems)
      scanCursor.value = res.data.cursor ?? 0
      total.value = items.value.length
    }
  } catch (_e) { /* 加载失败静默 */ }
  loading.value = false
}

function toggleAll(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  selected.value = checked ? items.value.map(i => i.member) : []
}

function toggleSelect(m: string) {
  const idx = selected.value.indexOf(m)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(m)
}

function startEditScore(item: ZItem) {
  editingMember.value = item.member
  editingScore.value = String(item.score)
}

async function saveScore(item: ZItem) {
  const score = parseFloat(editingScore.value) || 0
  try {
    await request('zadd_members', {
      params: { conn_id: props.connId, key: props.keyName, members: [{ member: item.member, score }] },
    })
    item.score = score
    editingMember.value = ''
  } catch (e) { showError(e) }
}

async function removeMembers(list: string[], skipConfirm = false) {
  if (!list.length) return
  if (!skipConfirm && !await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('zrem_members', {
      params: { conn_id: props.connId, key: props.keyName, members: list },
    })
    items.value = items.value.filter(i => !list.includes(i.member))
    selected.value = selected.value.filter(s => !list.includes(s))
    total.value = Math.max(0, total.value - list.length)
  } catch (e) { showError(e) }
}

function removeSelected() {
  removeMembers([...selected.value])
}

async function copyRow(val: string) {
  try {
    await navigator.clipboard.writeText(val)
    showMessage('success', t('common.copySuccess'))
  } catch (_e) { /* ignore */ }
}

function openAdd() {
  newMembers.value = [{ member: '', score: '0' }]
  addMode.value = ''
  showAdd.value = true
}

async function addMembers() {
  const members = newMembers.value
    .filter(r => r.member.trim())
    .map(r => ({ member: r.member.trim(), score: parseFloat(r.score) || 0 }))
  if (!members.length) return
  try {
    await request('zadd_members', {
      params: { conn_id: props.connId, key: props.keyName, members, mode: addMode.value || undefined },
    })
    showAdd.value = false
    showMessage('success', t('common.success'))
    resetAndLoad()
  } catch (e) { showError(e) }
}
</script>

<style scoped>
@import './collection.css';

.zset-order-toggle {
  display: flex;
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.order-btn {
  padding: 2px 10px;
  background: var(--color-fill-1);
  border: none;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.order-btn:not(:last-child) {
  border-right: 1px solid var(--color-border-2);
}

.order-btn.active {
  background: var(--color-primary);
  color: #fff;
}

.order-btn:not(.active):hover {
  background: var(--color-fill-3);
}

.score-cell {
  cursor: pointer;
}

.score-cell:hover {
  color: var(--color-primary);
}

.zset-mode-toggle {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-sm);
}

.zset-mode-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  white-space: nowrap;
}

.zset-mode-options {
  display: flex;
  background: var(--color-fill-1);
  border-radius: var(--radius-sm);
  padding: 2px;
  gap: 2px;
}

.mode-btn {
  padding: 3px 12px;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  cursor: pointer;
  transition: all 0.15s ease;
}

.mode-btn:hover {
  color: var(--color-text-1);
}

.mode-btn.active {
  background: var(--color-bg-1);
  color: var(--color-primary);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.zset-add-body {
  max-height: 50vh;
  overflow-y: auto;
  padding-bottom: var(--spacing-sm);
}

.zset-add-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.zset-add-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: var(--color-fill-1);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-1);
}

.zset-add-idx {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary);
  color: #fff;
  border-radius: 50%;
  font-size: var(--font-size-xs);
  font-weight: 600;
  flex-shrink: 0;
}

.zset-add-item :deep(.arco-input-wrapper) {
  flex: 1;
  min-width: 0;
}

.zset-score-input :deep(.arco-input-wrapper) {
  width: 100px !important;
  flex: none !important;
}

.zset-add-remove {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  cursor: pointer;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}

.zset-add-remove:hover {
  background: var(--color-error-bg);
  color: var(--color-error);
}

.zset-add-more-btn {
  width: 100%;
  margin-top: 6px;
  padding: 6px;
  background: none;
  border: 1px dashed var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: all 0.15s;
}

.zset-add-more-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-primary-bg);
}
</style>
