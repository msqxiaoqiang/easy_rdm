<template>
  <div class="collection-detail">
    <div class="collection-toolbar">
      <a-button size="mini" type="primary" @click="openAdd"><IconPlus :size="12" /> {{ $t('stream.addMessage') }}</a-button>
      <a-button size="mini" @click="openTrim">{{ $t('stream.trim') }}</a-button>
      <div class="toolbar-spacer"></div>
      <a-button size="mini" :class="{ active: showGroups }" @click="showGroups = !showGroups">
        {{ $t('stream.consumerGroups') }}
      </a-button>
      <a-button size="mini" status="danger" @click="deleteSelected" :disabled="!selected.length">
        {{ $t('common.delete') }} ({{ selected.length }})
      </a-button>
    </div>

    <div class="collection-table-wrap">
      <table class="collection-table">
        <thead>
          <tr>
            <th class="col-check"><input type="checkbox" @change="toggleAll" :checked="allChecked" /></th>
            <th class="col-stream-id">ID</th>
            <th>{{ $t('stream.fields') }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="msg in messages" :key="msg.id" :class="{ selected: selected.includes(msg.id) }">
            <td class="col-check"><input type="checkbox" :checked="selected.includes(msg.id)" @change="toggleSelect(msg.id)" /></td>
            <td class="col-stream-id mono">{{ msg.id }}</td>
            <td class="mono">
              <span class="cell-text" v-ellipsis-tip>{{ formatFields(msg.fields) }}</span>
            </td>
            <td class="col-actions">
              <button class="mini-btn" :data-tooltip="$t('common.copy')" data-tooltip-pos="bottom" @click="copyRow(msg)"><IconCopy :size="14" /></button>
              <button class="mini-btn danger" :data-tooltip="$t('common.delete')" data-tooltip-pos="bottom" @click="deleteMessages([msg.id])"><IconDelete :size="14" /></button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!messages.length && !loading" class="empty-hint">{{ $t('common.noData') }}</div>
    </div>

    <div class="collection-footer">
      <span class="footer-info">{{ messages.length }} / {{ total }}</span>
      <div class="page-nav">
        <a-button size="mini" :disabled="!cursorStack.length" @click="prevPage">‹</a-button>
        <span>{{ cursorStack.length + 1 }}</span>
        <a-button size="mini" :disabled="!hasMore" @click="nextPage">›</a-button>
      </div>
    </div>

    <!-- Consumer Groups Panel -->
    <div v-if="showGroups" class="groups-panel">
      <div class="groups-header">
        <span class="groups-title">{{ $t('stream.consumerGroups') }}</span>
        <a-button size="mini" type="primary" @click="openCreateGroup"><IconPlus :size="12" /> {{ $t('stream.createGroup') }}</a-button>
      </div>
      <table class="collection-table" v-if="groups.length">
        <thead>
          <tr>
            <th>{{ $t('stream.groupName') }}</th>
            <th>{{ $t('stream.consumers') }}</th>
            <th>{{ $t('stream.pending') }}</th>
            <th>{{ $t('stream.lastDeliveredId') }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="g in groups" :key="g.name">
            <td class="mono">{{ g.name }}</td>
            <td>{{ g.consumers }}</td>
            <td>{{ g.pending }}</td>
            <td class="mono">{{ g.last_delivered_id }}</td>
            <td class="col-actions">
              <button class="mini-btn danger" :data-tooltip="$t('common.delete')" data-tooltip-pos="bottom" @click="destroyGroup(g.name)"><IconDelete :size="14" /></button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty-hint">{{ $t('common.noData') }}</div>
    </div>

    <!-- Add Message Dialog -->
    <a-modal :visible="showAdd" :title="$t('stream.addMessage')" :width="520" unmount-on-close @cancel="showAdd = false">
      <div class="stream-add-body">
        <label class="dialog-label">ID ({{ $t('stream.autoId') }})</label>
        <a-input v-model="newId" placeholder="*" />
        <label class="dialog-label" style="margin-top:var(--spacing-md)">{{ $t('stream.fields') }}</label>
        <div class="stream-fields-list">
          <div v-for="(pair, idx) in newFields" :key="idx" class="stream-field-item">
            <span class="stream-field-idx">{{ idx + 1 }}</span>
            <a-input v-model="pair.key" :placeholder="$t('collection.field')" />
            <a-input v-model="pair.value" :placeholder="$t('collection.value')" />
            <button v-if="newFields.length > 1" class="stream-field-remove" @click="newFields.splice(idx, 1)"><IconClose :size="12" /></button>
          </div>
        </div>
        <button class="stream-add-field-btn" @click="newFields.push({ key: '', value: '' })"><IconPlus :size="12" /> {{ $t('collection.addField') }}</button>
      </div>
      <template #footer>
        <a-button @click="showAdd = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="addMessage">{{ $t('common.confirm') }}</a-button>
      </template>
    </a-modal>

    <!-- Trim Dialog -->
    <a-modal :visible="showTrim" :title="$t('stream.trim')" :width="400" unmount-on-close @cancel="showTrim = false">
      <label class="dialog-label">MAXLEN</label>
      <a-input v-model="trimMaxLen" type="number" :min="0" />
      <template #footer>
        <a-button @click="showTrim = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="trimStream">{{ $t('common.confirm') }}</a-button>
      </template>
    </a-modal>

    <!-- Create Group Dialog -->
    <a-modal :visible="showCreateGroup" :title="$t('stream.createGroup')" :width="400" unmount-on-close @cancel="showCreateGroup = false">
      <label class="dialog-label">{{ $t('stream.groupName') }}</label>
      <a-input v-model="newGroupName" />
      <label class="dialog-label">{{ $t('stream.startId') }}</label>
      <a-input v-model="newGroupStart" placeholder="$" />
      <template #footer>
        <a-button @click="showCreateGroup = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="createGroup">{{ $t('common.confirm') }}</a-button>
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
import { IconCopy, IconDelete, IconClose, IconPlus } from '@arco-design/web-vue/es/icon'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

interface StreamMessage { id: string; fields: Record<string, string> }
interface GroupInfo { name: string; consumers: number; pending: number; last_delivered_id: string }

const messages = ref<StreamMessage[]>([])
const total = ref(0)
const cursorStack = ref<string[]>([])  // 历史页的 start ID
const currentStart = ref('-')
const hasMore = ref(false)
const pageSize = 100
const loading = ref(false)
const selected = ref<string[]>([])

const showGroups = ref(false)
const groups = ref<GroupInfo[]>([])

const showAdd = ref(false)
const newId = ref('*')
const newFields = ref<{ key: string; value: string }[]>([{ key: '', value: '' }])

const showTrim = ref(false)
const trimMaxLen = ref('1000')

const showCreateGroup = ref(false)
const newGroupName = ref('')
const newGroupStart = ref('$')

const allChecked = computed(() => messages.value.length > 0 && selected.value.length === messages.value.length)

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `stream:${props.keyName}`, {
    messages: [...messages.value], total: total.value, selected: [...selected.value],
    cursorStack: [...cursorStack.value], currentStart: currentStart.value, hasMore: hasMore.value,
  })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `stream:${oldKey}`, {
    messages: [...messages.value], total: total.value, selected: [...selected.value],
    cursorStack: [...cursorStack.value], currentStart: currentStart.value, hasMore: hasMore.value,
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `stream:${newKey}`)
  if (cached) {
    messages.value = cached.messages; total.value = cached.total; selected.value = cached.selected
    cursorStack.value = cached.cursorStack; currentStart.value = cached.currentStart; hasMore.value = cached.hasMore
    return
  }
  resetAndLoad()
}, { immediate: true })
watch(showGroups, (v) => { if (v) loadGroups() })

function formatFields(fields: Record<string, string>): string {
  return Object.entries(fields).map(([k, v]) => `${k}: ${v}`).join(', ')
}

async function resetAndLoad() {
  messages.value = []
  selected.value = []
  cursorStack.value = []
  currentStart.value = '-'
  hasMore.value = false
  await loadPage()
}

async function loadPage() {
  loading.value = true
  try {
    const res = await request<any>('xrange_messages', {
      params: { conn_id: props.connId, key: props.keyName, start: currentStart.value, end: '+', count: pageSize, rev: false },
    })
    if (res.data) {
      messages.value = res.data.messages || []
      total.value = res.data.total ?? 0
      hasMore.value = res.data.has_more ?? false
    }
  } catch (_e) { /* ignore */ }
  loading.value = false
}

function nextStreamId(id: string): string {
  const parts = id.split('-')
  if (parts.length === 2) {
    return parts[0] + '-' + (parseInt(parts[1], 10) + 1)
  }
  return id + '-1'
}

function nextPage() {
  if (!messages.value.length) return
  const lastId = messages.value[messages.value.length - 1].id
  cursorStack.value.push(currentStart.value)
  currentStart.value = nextStreamId(lastId)
  loadPage()
}

function prevPage() {
  if (!cursorStack.value.length) return
  currentStart.value = cursorStack.value.pop()!
  loadPage()
}

function toggleAll(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  selected.value = checked ? messages.value.map(m => m.id) : []
}

function toggleSelect(id: string) {
  const idx = selected.value.indexOf(id)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(id)
}

async function deleteMessages(ids: string[], skipConfirm = false) {
  if (!ids.length) return
  if (!skipConfirm && !await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('xdel_messages', { params: { conn_id: props.connId, key: props.keyName, ids } })
    messages.value = messages.value.filter(m => !ids.includes(m.id))
    selected.value = selected.value.filter(s => !ids.includes(s))
    total.value = Math.max(0, total.value - ids.length)
  } catch (e) { showError(e) }
}

function deleteSelected() { deleteMessages([...selected.value]) }

async function copyRow(msg: StreamMessage) {
  try {
    const text = Object.entries(msg.fields).map(([k, v]) => `${k}: ${v}`).join('\n')
    await navigator.clipboard.writeText(text)
    showMessage('success', t('common.copySuccess'))
  } catch (_e) { /* ignore */ }
}

function openAdd() {
  newId.value = '*'
  newFields.value = [{ key: '', value: '' }]
  showAdd.value = true
}

async function addMessage() {
  const fields: Record<string, string> = {}
  for (const p of newFields.value) {
    if (p.key) fields[p.key] = p.value
  }
  if (!Object.keys(fields).length) return
  try {
    await request('xadd_message', {
      params: { conn_id: props.connId, key: props.keyName, id: newId.value || '*', fields },
    })
    showAdd.value = false
    showMessage('success', t('common.success'))
    resetAndLoad()
  } catch (e) { showError(e) }
}

function openTrim() {
  trimMaxLen.value = '1000'
  showTrim.value = true
}

async function trimStream() {
  try {
    await request('xtrim_stream', {
      params: { conn_id: props.connId, key: props.keyName, max_len: parseInt(trimMaxLen.value) || 0 },
    })
    showTrim.value = false
    resetAndLoad()
  } catch (e) { showError(e) }
}

async function loadGroups() {
  try {
    const res = await request<any>('xinfo_groups', { params: { conn_id: props.connId, key: props.keyName } })
    groups.value = res.data || []
  } catch (_e) { groups.value = [] }
}

function openCreateGroup() {
  newGroupName.value = ''
  newGroupStart.value = '$'
  showCreateGroup.value = true
}

async function createGroup() {
  if (!newGroupName.value) return
  try {
    await request('xgroup_create', {
      params: { conn_id: props.connId, key: props.keyName, group: newGroupName.value, start: newGroupStart.value || '$' },
    })
    showCreateGroup.value = false
    loadGroups()
  } catch (e) { showError(e) }
}

async function destroyGroup(name: string) {
  if (!await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('xgroup_destroy', { params: { conn_id: props.connId, key: props.keyName, group: name } })
    groups.value = groups.value.filter(g => g.name !== name)
  } catch (e) { showError(e) }
}
</script>

<style scoped>
@import './collection.css';

.col-stream-id {
  width: 200px;
  font-family: var(--font-family-mono);
}

.groups-panel {
  border-top: 2px solid var(--color-border-1);
  max-height: 200px;
  overflow: auto;
}

.groups-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-xs) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
}

.groups-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-2);
  flex: 1;
}

.stream-add-body {
  max-height: 50vh;
  overflow-y: auto;
  padding-bottom: var(--spacing-sm);
}

.stream-fields-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stream-field-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: var(--color-fill-1);
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-1);
}

.stream-field-idx {
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

.stream-field-item :deep(.arco-input-wrapper) {
  flex: 1;
  min-width: 0;
}

.stream-field-remove {
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

.stream-field-remove:hover {
  background: var(--color-error-bg);
  color: var(--color-error);
}

.stream-add-field-btn {
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

.stream-add-field-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-primary-bg);
}
</style>
