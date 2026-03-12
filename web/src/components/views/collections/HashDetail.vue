<template>
  <div class="collection-detail">
    <div class="collection-toolbar">
      <input
        v-model="pattern"
        class="filter-input"
        :placeholder="$t('collection.filterFields')"
        @keyup.enter="searchMode = true; resetAndLoad()"
      />
      <a-button size="mini" @click="searchMode = true; resetAndLoad()"><template #icon><IconSearch :size="14" /></template></a-button>
      <a-button v-if="searchMode" size="mini" @click="searchMode = false; pattern = ''; resetAndLoad()"><IconClose :size="12" /> {{ $t('common.reset') }}</a-button>
      <div class="toolbar-spacer"></div>
      <a-button size="mini" type="primary" @click="openAdd"><IconPlus :size="12" /> {{ $t('collection.addField') }}</a-button>
      <a-button size="mini" status="danger" @click="deleteSelected" :disabled="!selected.length">
        {{ $t('common.delete') }} ({{ selected.length }})
      </a-button>
    </div>

    <div class="hash-body">
      <!-- Left: field list -->
      <div class="hash-table-pane" :class="{ 'full-width': !editingField }" :style="editingField ? { width: tablePaneWidth + 'px' } : {}">
        <div class="collection-table-wrap">
          <table class="collection-table">
            <thead>
              <tr>
                <th class="col-check"><input type="checkbox" @change="toggleAll" :checked="allChecked" /></th>
                <th>{{ $t('collection.field') }}</th>
                <th>{{ $t('collection.value') }}</th>
                <th class="col-actions"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in fields" :key="item.field" :class="{ selected: selected.includes(item.field), active: editingField === item.field }">
                <td class="col-check"><input type="checkbox" :checked="selected.includes(item.field)" @change="toggleSelect(item.field)" /></td>
                <td class="mono"><span class="cell-text" v-ellipsis-tip>{{ item.field }}</span></td>
                <td class="mono"><span class="cell-text" v-ellipsis-tip>{{ item.value }}</span></td>
                <td class="col-actions">
                  <button class="mini-btn" :data-tooltip="$t('common.edit')" data-tooltip-pos="bottom" @click="selectField(item)"><IconEdit :size="14" /></button>
                  <button class="mini-btn" :data-tooltip="$t('common.copy')" data-tooltip-pos="bottom" @click="copyRow(item.value)"><IconCopy :size="14" /></button>
                  <button class="mini-btn danger" :data-tooltip="$t('common.delete')" data-tooltip-pos="bottom" @click="deleteFields([item.field])"><IconDelete :size="14" /></button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!fields.length && !loading" class="empty-hint">{{ $t('common.noData') }}</div>
        </div>
      </div>

      <!-- Resize handle -->
      <div v-if="editingField" class="hash-resize-handle" @mousedown="startPaneResize"></div>

      <!-- Right: editor panel -->
      <div v-if="editingField" class="hash-editor-pane">
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
            <span class="editor-field-name mono" v-ellipsis-tip>{{ editingField }}</span>
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
      <span class="footer-info">{{ fields.length }} {{ $t('collection.fieldsLoaded') }}</span>
      <a-button v-if="cursor !== 0" size="mini" @click="loadMore">{{ $t('collection.loadMore') }}</a-button>
    </div>

    <!-- Add Dialog -->
    <a-modal :visible="showAdd" :title="$t('collection.addField')" :width="400" unmount-on-close @cancel="showAdd = false">
      <label class="dialog-label">{{ $t('collection.field') }}</label>
      <a-input v-model="newField" />
      <label class="dialog-label">{{ $t('collection.value') }}</label>
      <a-textarea v-model="newValue" :auto-size="{ minRows: 3, maxRows: 8 }" />
      <template #footer>
        <a-button @click="showAdd = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="addField">{{ $t('common.confirm') }}</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onBeforeUnmount } from 'vue'
import { request } from '../../../utils/request'
import { useConnectionStore } from '../../../stores/connection'
import JsonCodeEditor from '../../common/JsonCodeEditor.vue'
import { useI18n } from 'vue-i18n'
import { IconSearch, IconClose, IconEdit, IconCopy, IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'
import { gmConfirm } from '../../../utils/dialog'
import { showMessage } from '../../../utils/platform'
import { useDecoder } from '../../../composables/useDecoder'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

// Decoder
const { decoders, selectedDecoder, decodedValue, applyDecoder, resetDecoder } = useDecoder()

function onDecoderChange() {
  applyDecoder(rawEditValue.value)
}

interface FieldItem { field: string; value: string }

const fields = ref<FieldItem[]>([])
const cursor = ref<number>(0)
const pattern = ref('')
const searchMode = ref(false)
const loading = ref(false)
const selected = ref<string[]>([])

// Editor state
const editingField = ref('')
const rawEditValue = ref('')
const editValue = ref('')
const viewAs = ref('text')
const modified = ref(false)
const suppressViewAsWatch = ref(false)
const editorLoading = ref(false)

// Add dialog
const showAdd = ref(false)
const newField = ref('')
const newValue = ref('')

const allChecked = computed(() => fields.value.length > 0 && selected.value.length === fields.value.length)

// Pane resize
const tablePaneWidth = ref(400)
let paneResizing = false

function startPaneResize(e: MouseEvent) {
  e.preventDefault()
  paneResizing = true
  const startX = e.clientX
  const startW = tablePaneWidth.value
  function onMove(ev: MouseEvent) {
    if (!paneResizing) return
    const w = startW + (ev.clientX - startX)
    tablePaneWidth.value = Math.max(200, Math.min(w, 800))
  }
  function onUp() {
    paneResizing = false
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

onBeforeUnmount(() => {
  paneResizing = false
  saveHashCache()
})

function saveHashCache() {
  connectionStore.saveKeyDetailCache(props.connId, `hash:${props.keyName}`, {
    fields: [...fields.value], cursor: cursor.value, selected: [...selected.value],
    editingField: editingField.value, rawEditValue: rawEditValue.value,
    editValue: editValue.value, viewAs: viewAs.value, modified: modified.value,
  })
}

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `hash:${oldKey}`, {
    fields: [...fields.value], cursor: cursor.value, selected: [...selected.value],
    editingField: editingField.value, rawEditValue: rawEditValue.value,
    editValue: editValue.value, viewAs: viewAs.value, modified: modified.value,
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `hash:${newKey}`)
  if (cached) {
    fields.value = cached.fields; cursor.value = cached.cursor; selected.value = cached.selected
    editingField.value = cached.editingField; rawEditValue.value = cached.rawEditValue
    editValue.value = cached.editValue; viewAs.value = cached.viewAs; modified.value = cached.modified
    return
  }
  resetAndLoad()
}, { immediate: true })

async function resetAndLoad() {
  fields.value = []
  cursor.value = 0
  selected.value = []
  editingField.value = ''
  await loadMore()
}

async function loadMore() {
  loading.value = true
  try {
    const res = await request<any>('hscan_fields', {
      params: { conn_id: props.connId, key: props.keyName, pattern: pattern.value || '*', cursor: cursor.value, count: 100 },
    })
    if (res.data) {
      const existing = new Set(fields.value.map(f => f.field))
      const newFields = (res.data.fields || []).filter((f: FieldItem) => !existing.has(f.field))
      fields.value.push(...newFields)
      cursor.value = res.data.cursor ?? 0
    }
  } catch (_e) { /* silent */ }
  loading.value = false
}

function toggleAll(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  selected.value = checked ? fields.value.map(f => f.field) : []
}

function toggleSelect(field: string) {
  const idx = selected.value.indexOf(field)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(field)
}

function selectField(item: FieldItem) {
  editingField.value = item.field
  editorLoading.value = true
  resetDecoder()
  setTimeout(() => {
    rawEditValue.value = item.value
    modified.value = false
    suppressViewAsWatch.value = true
    // 自动识别 JSON 格式：仅对象/数组（以 { 或 [ 开头）才切换 JSON 视图
    const trimmed = item.value.trim()
    if ((trimmed.startsWith('{') || trimmed.startsWith('[')) && (() => { try { JSON.parse(trimmed); return true } catch { return false } })()) {
      viewAs.value = 'json'
      editValue.value = JSON.stringify(JSON.parse(trimmed), null, 2)
    } else {
      viewAs.value = 'text'
      editValue.value = item.value
    }
    suppressViewAsWatch.value = false
    editorLoading.value = false
  }, 150)
}

watch(viewAs, (mode, oldMode) => {
  if (suppressViewAsWatch.value) return
  let raw: string
  if (oldMode === 'hex') {
    try {
      const bytes = editValue.value.trim().split(/\s+/).map(h => parseInt(h, 16))
      raw = new TextDecoder().decode(new Uint8Array(bytes))
    } catch (_e) { raw = rawEditValue.value }
  } else if (oldMode === 'json') {
    try { raw = JSON.stringify(JSON.parse(editValue.value)) } catch (_e) { raw = editValue.value }
  } else {
    raw = editValue.value
  }
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
  if (!editingField.value) return
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
    await request('hset_field', {
      params: { conn_id: props.connId, key: props.keyName, field: editingField.value, value: saveValue },
    })
    const item = fields.value.find(f => f.field === editingField.value)
    if (item) item.value = saveValue
    rawEditValue.value = saveValue
    modified.value = false
    editingField.value = ''
  } catch (e) { showError(e) }
}

function closeEditor() {
  editingField.value = ''
}

async function copyRow(val: string) {
  try {
    await navigator.clipboard.writeText(val)
    showMessage('success', t('common.copySuccess'))
  } catch (_e) { /* ignore */ }
}

async function deleteFields(fieldList: string[], skipConfirm = false) {
  if (!fieldList.length) return
  if (!skipConfirm && !await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('hdel_fields', {
      params: { conn_id: props.connId, key: props.keyName, fields: fieldList },
    })
    fields.value = fields.value.filter(f => !fieldList.includes(f.field))
    selected.value = selected.value.filter(s => !fieldList.includes(s))
    if (fieldList.includes(editingField.value)) editingField.value = ''
  } catch (e) { showError(e) }
}

function deleteSelected() {
  deleteFields([...selected.value])
}

function openAdd() {
  newField.value = ''
  newValue.value = ''
  showAdd.value = true
}

async function addField() {
  if (!newField.value) return
  try {
    await request('hset_field', {
      params: { conn_id: props.connId, key: props.keyName, field: newField.value, value: newValue.value },
    })
    showAdd.value = false
    showMessage('success', t('common.success'))
    resetAndLoad()
  } catch (e) { showError(e) }
}
</script>

<style scoped>
@import './collection.css';

.hash-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.hash-table-pane {
  min-width: 200px;
  max-width: 800px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--color-border-1);
}

.hash-resize-handle {
  width: 4px;
  flex-shrink: 0;
  cursor: col-resize;
  background: transparent;
  transition: background var(--transition-fast);
}

.hash-resize-handle:hover {
  background: var(--color-primary);
}

.hash-table-pane.full-width {
  width: 100%;
  max-width: none;
  border-right: none;
}

.hash-table-pane .collection-table-wrap {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.hash-table-pane .collection-table {
  table-layout: fixed;
  width: 100%;
}

.hash-table-pane .col-actions {
  width: 72px;
}

.collection-table tr.active td {
  background: var(--color-primary-bg);
}

.hash-editor-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.editor-field-name {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.decoded-view {
  flex: 1;
  overflow: auto;
  padding: var(--spacing-md);
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--color-bg-1);
}
</style>
