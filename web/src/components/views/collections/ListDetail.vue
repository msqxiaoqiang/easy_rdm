<template>
  <div class="collection-detail">
    <div class="collection-toolbar">
      <span class="footer-info">{{ $t('collection.total') }}: {{ total }}</span>
      <div class="toolbar-spacer"></div>
      <a-button size="mini" type="primary" @click="openPush"><IconPlus :size="12" /> {{ $t('collection.push') }}</a-button>
    </div>

    <div class="list-body">
      <!-- Left: data table -->
      <div class="list-table-pane" :class="{ 'full-width': selectedIdx < 0 }">
        <div class="collection-table-wrap">
          <table class="collection-table">
            <thead>
              <tr>
                <th class="col-idx">#</th>
                <th>{{ $t('collection.value') }}</th>
                <th class="col-actions"></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(val, i) in values"
                :key="page * pageSize + i"
                :class="{ selected: selectedIdx === page * pageSize + i }"
              >
                <td class="col-idx">{{ page * pageSize + i }}</td>
                <td class="mono"><span class="cell-text" v-ellipsis-tip>{{ val }}</span></td>
                <td class="col-actions">
                  <button class="mini-btn" :data-tooltip="$t('common.edit')" data-tooltip-pos="bottom" @click="selectRow(page * pageSize + i, val)"><IconEdit :size="14" /></button>
                  <button class="mini-btn" :data-tooltip="$t('common.copy')" data-tooltip-pos="bottom" @click="copyRow(val)"><IconCopy :size="14" /></button>
                  <button class="mini-btn danger" :data-tooltip="$t('common.delete')" data-tooltip-pos="bottom" @click="removeItem(page * pageSize + i)"><IconDelete :size="14" /></button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!values.length && !loading" class="empty-hint">{{ $t('common.noData') }}</div>
        </div>
      </div>

      <!-- Right: editor panel (only visible when editing) -->
      <div v-if="selectedIdx >= 0" class="list-editor-pane">
        <template v-if="editorLoading">
          <div class="editor-loading">
            <span class="loading-spinner"></span>
          </div>
        </template>
        <template v-else>
          <div class="value-toolbar">
            <a-select v-model="viewAs" class="view-select" size="mini" :style="{ width: '110px' }">
              <a-option value="text">{{ $t('format.text') }}</a-option>
              <a-option value="json">{{ $t('format.json') }}</a-option>
              <a-option value="hex">{{ $t('format.hex') }}</a-option>
            </a-select>
            <a-select v-if="decoders.length" v-model="selectedDecoder" class="view-select" size="mini" :style="{ width: '110px' }" @change="onDecoderChange">
              <a-option value="">{{ $t('decoder.none') }}</a-option>
              <a-option v-for="d in decoders" :key="d.id" :value="d.id">{{ d.name }}</a-option>
            </a-select>
            <span class="editor-index">#{{ selectedIdx }}</span>
            <div class="toolbar-spacer"></div>
            <button v-if="!selectedDecoder" class="save-btn" @click="saveEdit" :disabled="!modified">{{ $t('common.save') }}</button>
            <button class="mini-btn" :data-tooltip="$t('common.close')" data-tooltip-pos="bottom" @click="closeEditor"><IconClose :size="14" /></button>
          </div>
          <div v-if="selectedDecoder && decodedValue !== null" class="value-editor mono decoded-view">{{ decodedValue }}</div>
          <JsonCodeEditor
            v-else-if="viewAs === 'json'"
            :model-value="editValue"
            @update:model-value="editValue = $event; modified = true"
          />
          <textarea
            v-else
            v-model="editValue"
            class="value-editor"
            :class="{ 'mono': viewAs !== 'text' }"
            spellcheck="false"
            @input="modified = true"
          ></textarea>
        </template>
      </div>
    </div>

    <div class="collection-footer">
      <span class="footer-info">{{ values.length }} / {{ total }}</span>
      <div class="page-nav">
        <a-button size="mini" :disabled="page === 0" @click="page--; loadPage()">‹</a-button>
        <span>{{ page + 1 }} / {{ totalPages }}</span>
        <a-button size="mini" :disabled="page >= totalPages - 1" @click="page++; loadPage()">›</a-button>
      </div>
    </div>

    <!-- Push Dialog -->
    <a-modal :visible="showPush" :title="$t('collection.push')" :width="520" unmount-on-close @cancel="showPush = false">
      <div class="push-body">
        <!-- Position toggle -->
        <div class="push-section">
          <label class="push-label">{{ $t('collection.position') }}</label>
          <div class="push-pos-toggle">
            <button :class="['pos-btn', { active: pushPos === 'tail' }]" @click="pushPos = 'tail'">
              {{ $t('collection.tail') }} (RPush)
            </button>
            <button :class="['pos-btn', { active: pushPos === 'head' }]" @click="pushPos = 'head'">
              {{ $t('collection.head') }} (LPush)
            </button>
          </div>
        </div>

        <!-- Values -->
        <div class="push-section">
          <label class="push-label">{{ $t('collection.value') }}</label>
          <div class="push-value-list">
            <div
              v-for="(_, i) in pushValues"
              :key="i"
              :class="['push-value-item', { dragging: dragIdx === i, 'drop-before': dropIdx === i && dragIdx !== i && dragIdx !== i - 1, 'drop-after': dropIdx === pushValues.length && i === pushValues.length - 1 && dragIdx !== i }]"
              @dragover.prevent="onDragOver($event, i)"
              @dragleave="onDragLeave"
              @drop.prevent="onDrop"
            >
              <span
                class="push-value-handle"
                draggable="true"
                @dragstart="onDragStart(i)"
                @dragend="onDragEnd"
              ><IconDragDotVertical :size="14" /></span>
              <span class="push-value-idx">{{ i + 1 }}</span>
              <textarea
                v-model="pushValues[i]"
                class="push-value-input"
                rows="2"
                spellcheck="false"
                :placeholder="$t('collection.value') + ' ' + (i + 1)"
              ></textarea>
              <button
                v-if="pushValues.length > 1"
                class="push-value-remove"
                @click="pushValues.splice(i, 1)"
              ><IconClose :size="12" /></button>
            </div>
            <button class="push-add-btn" @click="pushValues.push('')">
              <IconPlus :size="14" />
              {{ $t('common.add') }}
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <a-button @click="showPush = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="doPush" :disabled="!pushValues.some(v => v.trim())">{{ $t('common.confirm') }}</a-button>
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
import { useDecoder } from '../../../composables/useDecoder'
import JsonCodeEditor from '../../common/JsonCodeEditor.vue'
import { IconEdit, IconCopy, IconDelete, IconClose, IconPlus, IconDragDotVertical } from '@arco-design/web-vue/es/icon'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

// Decoder
const { decoders, selectedDecoder, decodedValue, applyDecoder, resetDecoder } = useDecoder()

function onDecoderChange() {
  applyDecoder(rawSelectedValue.value)
}

const values = ref<string[]>([])
const total = ref(0)
const page = ref(0)
const pageSize = 100
const loading = ref(false)

// Editor state
const selectedIdx = ref(-1)
const rawSelectedValue = ref('')
const editValue = ref('')
const viewAs = ref('text')
const modified = ref(false)
const suppressViewAsWatch = ref(false)
const editorLoading = ref(false)

// Push dialog
const showPush = ref(false)
const pushPos = ref('tail')
const pushValues = ref<string[]>([''])
const dragIdx = ref(-1)
const dropIdx = ref(-1)

function onDragStart(i: number) {
  dragIdx.value = i
}

function onDragOver(e: DragEvent, i: number) {
  if (dragIdx.value < 0) return
  const el = (e.currentTarget as HTMLElement)
  const rect = el.getBoundingClientRect()
  const midY = rect.top + rect.height / 2
  dropIdx.value = e.clientY < midY ? i : i + 1
}

function onDragLeave() {
  // only clear if leaving the list entirely (child transitions don't count)
}

function onDrop() {
  if (dragIdx.value < 0 || dropIdx.value < 0) return
  const from = dragIdx.value
  let to = dropIdx.value
  if (to === from || to === from + 1) { dragIdx.value = -1; dropIdx.value = -1; return }
  const item = pushValues.value.splice(from, 1)[0]
  if (to > from) to--
  pushValues.value.splice(to, 0, item)
  dragIdx.value = -1
  dropIdx.value = -1
}

function onDragEnd() {
  dragIdx.value = -1
  dropIdx.value = -1
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `list:${props.keyName}`, {
    values: [...values.value], total: total.value, page: page.value,
    selectedIdx: selectedIdx.value, rawSelectedValue: rawSelectedValue.value,
    editValue: editValue.value, viewAs: viewAs.value, modified: modified.value,
  })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `list:${oldKey}`, {
    values: [...values.value], total: total.value, page: page.value,
    selectedIdx: selectedIdx.value, rawSelectedValue: rawSelectedValue.value,
    editValue: editValue.value, viewAs: viewAs.value, modified: modified.value,
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `list:${newKey}`)
  if (cached) {
    values.value = cached.values; total.value = cached.total; page.value = cached.page
    selectedIdx.value = cached.selectedIdx; rawSelectedValue.value = cached.rawSelectedValue
    editValue.value = cached.editValue; viewAs.value = cached.viewAs; modified.value = cached.modified
    return
  }
  page.value = 0; selectedIdx.value = -1; loadPage()
}, { immediate: true })

async function loadPage() {
  loading.value = true
  const start = page.value * pageSize
  const stop = start + pageSize - 1
  try {
    const res = await request<any>('lrange_values', {
      params: { conn_id: props.connId, key: props.keyName, start, stop },
    })
    if (res.data) {
      values.value = res.data.values || []
      total.value = res.data.total ?? 0
    }
  } catch (_e) { /* silent */ }
  loading.value = false
}

function selectRow(idx: number, val: string) {
  selectedIdx.value = idx
  editorLoading.value = true
  resetDecoder()
  setTimeout(() => {
    rawSelectedValue.value = val
    modified.value = false
    suppressViewAsWatch.value = true
    // 自动识别 JSON 格式：仅对象/数组（以 { 或 [ 开头）才切换 JSON 视图
    const trimmed = val.trim()
    if ((trimmed.startsWith('{') || trimmed.startsWith('[')) && (() => { try { JSON.parse(trimmed); return true } catch { return false } })()) {
      viewAs.value = 'json'
      editValue.value = JSON.stringify(JSON.parse(trimmed), null, 2)
    } else {
      viewAs.value = 'text'
      editValue.value = val
    }
    suppressViewAsWatch.value = false
    editorLoading.value = false
  }, 150)
}

// Format conversion on viewAs change
watch(viewAs, (mode, oldMode) => {
  if (suppressViewAsWatch.value) return

  // Restore to raw string from current format
  let raw: string
  if (oldMode === 'hex') {
    try {
      const bytes = editValue.value.trim().split(/\s+/).map(h => parseInt(h, 16))
      raw = new TextDecoder().decode(new Uint8Array(bytes))
    } catch (_e) { raw = rawSelectedValue.value }
  } else if (oldMode === 'json') {
    try { raw = JSON.stringify(JSON.parse(editValue.value)) } catch (_e) { raw = editValue.value }
  } else {
    raw = editValue.value
  }

  // Format to target
  if (mode === 'json') {
    try { editValue.value = JSON.stringify(JSON.parse(raw), null, 2) } catch (_e) { editValue.value = raw }
  } else if (mode === 'hex') {
    editValue.value = Array.from(new TextEncoder().encode(raw))
      .map(b => b.toString(16).padStart(2, '0'))
      .join(' ')
  } else {
    editValue.value = raw
  }
})

async function saveEdit() {
  if (selectedIdx.value < 0) return
  // Restore to raw string before saving
  let saveValue = editValue.value
  if (viewAs.value === 'hex') {
    try {
      const bytes = saveValue.trim().split(/\s+/).map(h => parseInt(h, 16))
      saveValue = new TextDecoder().decode(new Uint8Array(bytes))
    } catch (_e) { /* save as-is */ }
  } else if (viewAs.value === 'json') {
    try { saveValue = JSON.stringify(JSON.parse(saveValue)) } catch (_e) { /* save as-is */ }
  }
  try {
    await request('lset_value', {
      params: { conn_id: props.connId, key: props.keyName, index: selectedIdx.value, value: saveValue },
    })
    // Update local list
    const localIdx = selectedIdx.value - page.value * pageSize
    if (localIdx >= 0 && localIdx < values.value.length) {
      values.value[localIdx] = saveValue
    }
    rawSelectedValue.value = saveValue
    modified.value = false
    selectedIdx.value = -1
  } catch (e) { showError(e) }
}

function closeEditor() {
  selectedIdx.value = -1
}

async function copyRow(val: string) {
  try {
    await navigator.clipboard.writeText(val)
    showMessage('success', t('common.copySuccess'))
  } catch (_e) { /* ignore */ }
}

async function removeItem(idx: number) {
  if (!await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('list_remove', {
      params: { conn_id: props.connId, key: props.keyName, index: idx },
    })
    if (selectedIdx.value === idx) selectedIdx.value = -1
    const newTotal = Math.max(0, total.value - 1)
    const maxPage = Math.max(0, Math.ceil(newTotal / pageSize) - 1)
    if (page.value > maxPage) page.value = maxPage
    await loadPage()
  } catch (e) { showError(e) }
}

function openPush() {
  pushValues.value = ['']
  pushPos.value = 'tail'
  showPush.value = true
}

async function doPush() {
  const vals = pushValues.value.filter(v => v.trim())
  if (!vals.length) return
  try {
    await request('list_push', {
      params: { conn_id: props.connId, key: props.keyName, values: vals, position: pushPos.value },
    })
    showPush.value = false
    showMessage('success', t('common.success'))
    await loadPage()
  } catch (e) { showError(e) }
}
</script>

<style scoped>
@import './collection.css';

.list-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.list-table-pane {
  width: 50%;
  min-width: 240px;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--color-border-1);
}

.list-table-pane.full-width {
  width: 100%;
  border-right: none;
}

.list-table-pane .collection-table-wrap {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.list-table-pane .collection-table {
  table-layout: fixed;
  width: 100%;
}

.list-table-pane .col-actions {
  width: 72px;
}

.list-editor-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.editor-index {
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  color: var(--color-text-3);
}

.decoded-view {
  flex: 1;
  overflow: auto;
  padding: var(--spacing-md);
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--color-bg-1);
}

.push-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-md) var(--spacing-lg);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.push-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.push-label {
  font-size: var(--font-size-xs);
  font-weight: 500;
  color: var(--color-text-3);
}

.push-pos-toggle {
  display: flex;
  background: var(--color-fill-1);
  border-radius: var(--radius-sm);
  padding: 2px;
  gap: 2px;
}

.pos-btn {
  flex: 1;
  padding: 6px 0;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  cursor: pointer;
  transition: all 0.15s ease;
}

.pos-btn:hover {
  color: var(--color-text-1);
}

.pos-btn.active {
  background: var(--color-bg-1);
  color: var(--color-primary);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.push-value-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.push-value-item {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-xs);
  position: relative;
  transition: opacity 0.15s ease;
}

.push-value-item.dragging {
  opacity: 0.3;
}

.push-value-item.drop-before::before {
  content: '';
  position: absolute;
  top: -5px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--color-primary);
  border-radius: 1px;
}

.push-value-item.drop-after::after {
  content: '';
  position: absolute;
  bottom: -5px;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--color-primary);
  border-radius: 1px;
}

.push-value-handle {
  width: 20px;
  height: 20px;
  margin-top: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  cursor: grab;
  border-radius: var(--radius-sm);
  user-select: none;
}

.push-value-handle:hover {
  background: var(--color-fill-2);
  color: var(--color-text-2);
}

.push-value-handle:active {
  cursor: grabbing;
}

.push-value-idx {
  width: 20px;
  height: 20px;
  margin-top: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: var(--color-fill-2);
  border-radius: 50%;
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-3);
}

.push-value-input {
  flex: 1;
  min-height: 48px;
  padding: var(--spacing-xs) var(--spacing-sm);
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-sm);
  line-height: 1.5;
  outline: none;
  resize: vertical;
  box-sizing: border-box;
  transition: border-color 0.15s ease;
}

.push-value-input:focus {
  border-color: var(--color-primary);
  background: var(--color-bg-1);
}

.push-value-remove {
  width: 24px;
  height: 24px;
  margin-top: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: all 0.15s ease;
}

.push-value-remove:hover {
  background: var(--color-error-bg);
  color: var(--color-error);
}

.push-add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: var(--spacing-xs) 0;
  background: none;
  border: 1px dashed var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: all 0.15s ease;
}

.push-add-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-primary-bg);
}

.push-add-icon {
  font-size: var(--font-size-sm);
  font-weight: 600;
  line-height: 1;
}
</style>
