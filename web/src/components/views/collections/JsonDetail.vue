<template>
  <div class="collection-detail">
    <div class="collection-toolbar">
      <a-button size="mini" @click="loadData" :data-tooltip="$t('common.refresh')">↻</a-button>
      <div class="toolbar-spacer"></div>
      <a-button size="mini" @click="expandAll">{{ $t('json.expandAll') }}</a-button>
      <a-button size="mini" @click="collapseAll">{{ $t('json.collapseAll') }}</a-button>
      <a-button size="mini" type="primary" @click="openAddRoot">+ {{ $t('json.addKey') }}</a-button>
      <a-button size="mini" @click="editRaw = !editRaw">{{ editRaw ? $t('json.treeMode') : $t('json.rawMode') }}</a-button>
    </div>

    <!-- Raw editor mode -->
    <template v-if="editRaw">
      <div class="json-raw-area">
        <JsonCodeEditor
          :model-value="rawText"
          @update:model-value="rawText = $event; rawModified = true"
        />
        <div class="json-raw-actions">
          <button class="save-btn" @click="saveRaw" :disabled="!rawModified">{{ $t('common.save') }}</button>
        </div>
      </div>
    </template>

    <!-- Tree mode -->
    <template v-else>
      <div class="json-tree-wrap" v-if="rootData !== undefined">
        <div class="json-tree">
          <JsonNode
            :data="rootData"
            path="."
            :depth="0"
            :expanded-set="expandedPaths"
            @toggle="toggleExpand"
            @edit="handleEdit"
            @delete="handleDelete"
            @add="handleAdd"
          />
        </div>
      </div>
      <div v-else-if="!loading" class="empty-hint">{{ $t('common.noData') }}</div>
    </template>

    <div v-if="loading" class="json-loading">
      <span class="loading-spinner"></span>
    </div>

    <!-- Edit dialog -->
    <a-modal :visible="showEditDialog" :title="editDialogIsAdd ? $t('json.addKey') : $t('json.editValue')" :width="400" unmount-on-close @cancel="showEditDialog = false">
      <label v-if="editDialogIsAdd" class="dialog-label">{{ $t('json.keyName') }}</label>
      <a-input v-if="editDialogIsAdd" v-model="editDialogKey" class="mono" :placeholder="$t('json.keyNamePlaceholder')" />
      <label class="dialog-label">{{ editDialogIsAdd ? $t('json.valueType') : $t('json.path') }}</label>
      <template v-if="editDialogIsAdd">
        <a-select v-model="editDialogType">
          <a-option value="string">String</a-option>
          <a-option value="number">Number</a-option>
          <a-option value="boolean">Boolean</a-option>
          <a-option value="null">Null</a-option>
          <a-option value="object">Object {}</a-option>
          <a-option value="array">Array []</a-option>
        </a-select>
      </template>
      <template v-else>
        <a-input :model-value="editDialogPath" class="mono" disabled />
      </template>
      <label v-if="editDialogType !== 'null' && editDialogType !== 'object' && editDialogType !== 'array'" class="dialog-label">{{ $t('collection.value') }}</label>
      <a-textarea
        v-if="editDialogType !== 'null' && editDialogType !== 'object' && editDialogType !== 'array'"
        v-model="editDialogValue"
        class="mono"
        :auto-size="{ minRows: 4, maxRows: 8 }"
        spellcheck="false"
      />
      <template #footer>
        <a-button @click="showEditDialog = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="confirmEdit">{{ $t('common.confirm') }}</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, defineComponent, h, onBeforeUnmount } from 'vue'
import { request } from '../../../utils/request'
import { useConnectionStore } from '../../../stores/connection'
import { useI18n } from 'vue-i18n'
import { gmConfirm } from '../../../utils/dialog'
import { showMessage } from '../../../utils/platform'
import JsonCodeEditor from '../../common/JsonCodeEditor.vue'

const props = defineProps<{
  connId: string
  keyName: string
}>()

const { t } = useI18n()
const connectionStore = useConnectionStore()

const rootData = ref<any>(undefined)
const loading = ref(false)
const expandedPaths = ref(new Set<string>(['.']))
const editRaw = ref(false)
const rawText = ref('')
const rawModified = ref(false)

// Edit dialog state
const showEditDialog = ref(false)
const editDialogIsAdd = ref(false)
const editDialogPath = ref('')
const editDialogKey = ref('')
const editDialogValue = ref('')
const editDialogType = ref('string')

async function loadData() {
  loading.value = true
  try {
    const res = await request<any>('json_get', {
      params: { conn_id: props.connId, key: props.keyName, path: '.' },
    })
    if (res.data?.value) {
      rootData.value = JSON.parse(res.data.value)
      rawText.value = JSON.stringify(rootData.value, null, 2)
      rawModified.value = false
    }
  } catch (e: any) {
    showMessage('error', e.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function saveRaw() {
  try {
    // Validate JSON
    JSON.parse(rawText.value)
    await request('json_set', {
      params: { conn_id: props.connId, key: props.keyName, path: '.', value: rawText.value },
    })
    rawModified.value = false
    await loadData()
    showMessage('success', t('common.success'))
  } catch (e: any) {
    showMessage('error', e.message || t('json.invalidJson'))
  }
}

function toggleExpand(path: string) {
  if (expandedPaths.value.has(path)) {
    expandedPaths.value.delete(path)
  } else {
    expandedPaths.value.add(path)
  }
  expandedPaths.value = new Set(expandedPaths.value) // trigger reactivity
}

function expandAll() {
  const paths = new Set<string>()
  function walk(data: any, path: string) {
    if (data && typeof data === 'object') {
      paths.add(path)
      if (Array.isArray(data)) {
        data.forEach((v, i) => walk(v, `${path}[${i}]`))
      } else {
        Object.keys(data).forEach(k => walk(data[k], path === '.' ? `.${k}` : `${path}.${k}`))
      }
    }
  }
  walk(rootData.value, '.')
  expandedPaths.value = paths
}

function collapseAll() {
  expandedPaths.value = new Set(['.'])
}

function handleEdit(path: string, currentValue: any) {
  editDialogIsAdd.value = false
  editDialogPath.value = path
  editDialogKey.value = ''
  if (currentValue === null) {
    editDialogType.value = 'null'
    editDialogValue.value = ''
  } else if (typeof currentValue === 'boolean') {
    editDialogType.value = 'boolean'
    editDialogValue.value = String(currentValue)
  } else if (typeof currentValue === 'number') {
    editDialogType.value = 'number'
    editDialogValue.value = String(currentValue)
  } else if (typeof currentValue === 'string') {
    editDialogType.value = 'string'
    editDialogValue.value = currentValue
  } else {
    // object/array - edit as raw JSON
    editDialogType.value = 'string'
    editDialogValue.value = JSON.stringify(currentValue, null, 2)
  }
  showEditDialog.value = true
}

async function handleDelete(path: string) {
  if (!await gmConfirm(t('json.deleteConfirm', { path }))) return
  try {
    await request('json_del', {
      params: { conn_id: props.connId, key: props.keyName, path },
    })
    await loadData()
  } catch (e: any) {
    showMessage('error', e.message || t('common.failed'))
  }
}

function handleAdd(parentPath: string) {
  editDialogIsAdd.value = true
  editDialogPath.value = parentPath
  editDialogKey.value = ''
  editDialogValue.value = ''
  editDialogType.value = 'string'
  showEditDialog.value = true
}

function openAddRoot() {
  handleAdd('.')
}

function buildJsonValue(): string {
  switch (editDialogType.value) {
    case 'null': return 'null'
    case 'boolean': return editDialogValue.value === 'true' ? 'true' : 'false'
    case 'number': return String(Number(editDialogValue.value) || 0)
    case 'object': return '{}'
    case 'array': return '[]'
    default: return JSON.stringify(editDialogValue.value) // string - properly escaped
  }
}

async function confirmEdit() {
  try {
    let path: string
    let value: string

    if (editDialogIsAdd.value) {
      // Adding a new key
      const parentPath = editDialogPath.value
      const key = editDialogKey.value.trim()
      if (!key) {
        showMessage('error', t('json.keyRequired'))
        return
      }
      // Check if parent is array (key should be numeric index or use ARRAPPEND)
      path = parentPath === '.' ? `.${key}` : `${parentPath}.${key}`
      value = buildJsonValue()
    } else {
      // Editing existing value
      path = editDialogPath.value
      value = buildJsonValue()
    }

    await request('json_set', {
      params: { conn_id: props.connId, key: props.keyName, path, value },
    })
    showEditDialog.value = false
    await loadData()
  } catch (e: any) {
    showMessage('error', e.message || t('common.failed'))
  }
}

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `json:${props.keyName}`, {
    rootData: rootData.value, expandedPaths: new Set(expandedPaths.value), editRaw: editRaw.value,
  })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `json:${oldKey}`, {
    rootData: rootData.value, expandedPaths: new Set(expandedPaths.value), editRaw: editRaw.value,
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `json:${newKey}`)
  if (cached) {
    rootData.value = cached.rootData; expandedPaths.value = cached.expandedPaths; editRaw.value = cached.editRaw
    return
  }
  rootData.value = undefined
  expandedPaths.value = new Set(['.'])
  editRaw.value = false
  loadData()
}, { immediate: true })

// ========== JsonNode recursive component ==========

const JsonNode = defineComponent({
  name: 'JsonNode',
  props: {
    data: { required: true },
    path: { type: String, required: true },
    depth: { type: Number, default: 0 },
    expandedSet: { type: Set, required: true },
    parentKey: { type: String, default: '' },
  },
  emits: ['toggle', 'edit', 'delete', 'add'],
  setup(props, { emit }) {
    const isExpanded = () => (props.expandedSet as Set<string>).has(props.path)
    const isObject = () => props.data !== null && typeof props.data === 'object' && !Array.isArray(props.data)
    const isArray = () => Array.isArray(props.data)
    const isExpandable = () => isObject() || isArray()

    function renderValue(val: any): string {
      if (val === null) return 'null'
      if (typeof val === 'boolean') return String(val)
      if (typeof val === 'number') return String(val)
      if (typeof val === 'string') {
        return val.length > 120 ? `"${val.substring(0, 120)}..."` : `"${val}"`
      }
      if (Array.isArray(val)) return `Array[${val.length}]`
      if (typeof val === 'object') return `Object{${Object.keys(val).length}}`
      return String(val)
    }

    function valueClass(val: any): string {
      if (val === null) return 'json-null'
      if (typeof val === 'boolean') return 'json-boolean'
      if (typeof val === 'number') return 'json-number'
      if (typeof val === 'string') return 'json-string'
      return ''
    }

    return () => {
      const children: any[] = []
      const indent = { paddingLeft: `${props.depth * 16}px` }

      // Key label
      const keyLabel = props.parentKey !== '' ? props.parentKey : (props.path === '.' ? '(root)' : props.path)

      if (isExpandable()) {
        const expanded = isExpanded()
        // Toggle row
        children.push(h('div', { class: 'json-row', style: indent }, [
          h('span', {
            class: ['json-toggle', expanded ? 'expanded' : 'collapsed'],
            onClick: () => emit('toggle', props.path),
          }, expanded ? '▼' : '▶'),
          h('span', { class: 'json-key' }, keyLabel),
          h('span', { class: 'json-type-hint' }, isArray() ? `Array[${(props.data as any[]).length}]` : `Object{${Object.keys(props.data as object).length}}`),
          h('span', { class: 'json-row-actions' }, [
            h('button', { class: 'mini-btn', 'data-tooltip': t('common.add'), onClick: () => emit('add', props.path) }, '+'),
            props.path !== '.' ? h('button', { class: 'mini-btn danger', 'data-tooltip': t('common.delete'), onClick: () => emit('delete', props.path) }, '✕') : null,
          ]),
        ]))

        // Children
        if (expanded) {
          if (isArray()) {
            ;(props.data as any[]).forEach((item, i) => {
              const childPath = `${props.path}[${i}]`
              children.push(h(JsonNode, {
                data: item,
                path: childPath,
                depth: props.depth + 1,
                expandedSet: props.expandedSet,
                parentKey: String(i),
                onToggle: (p: string) => emit('toggle', p),
                onEdit: (p: string, v: any) => emit('edit', p, v),
                onDelete: (p: string) => emit('delete', p),
                onAdd: (p: string) => emit('add', p),
              }))
            })
          } else {
            Object.entries(props.data as object).forEach(([key, val]) => {
              const childPath = props.path === '.' ? `.${key}` : `${props.path}.${key}`
              children.push(h(JsonNode, {
                data: val,
                path: childPath,
                depth: props.depth + 1,
                expandedSet: props.expandedSet,
                parentKey: key,
                onToggle: (p: string) => emit('toggle', p),
                onEdit: (p: string, v: any) => emit('edit', p, v),
                onDelete: (p: string) => emit('delete', p),
                onAdd: (p: string) => emit('add', p),
              }))
            })
          }
        }
      } else {
        // Leaf value
        children.push(h('div', { class: 'json-row json-leaf', style: indent }, [
          h('span', { class: 'json-toggle-placeholder' }),
          h('span', { class: 'json-key' }, keyLabel),
          h('span', { class: 'json-colon' }, ':'),
          h('span', {
            class: ['json-value', valueClass(props.data)],
            onClick: () => emit('edit', props.path, props.data),
            title: String(props.data),
          }, renderValue(props.data)),
          h('span', { class: 'json-row-actions' }, [
            h('button', { class: 'mini-btn', 'data-tooltip': t('common.edit'), onClick: () => emit('edit', props.path, props.data) }, '✎'),
            h('button', { class: 'mini-btn', 'data-tooltip': t('common.copy'), onClick: () => {
              navigator.clipboard.writeText(typeof props.data === 'string' ? props.data : JSON.stringify(props.data))
            }}, '⧉'),
            h('button', { class: 'mini-btn danger', 'data-tooltip': t('common.delete'), onClick: () => emit('delete', props.path) }, '✕'),
          ]),
        ]))
      }

      return h('div', { class: 'json-node' }, children)
    }
  },
})
</script>

<style src="./collection.css"></style>
<style scoped>
.json-tree-wrap {
  flex: 1;
  overflow: auto;
  padding: var(--spacing-sm) 0;
}

.json-tree {
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
  line-height: 1.8;
}

.json-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 1px var(--spacing-md);
  cursor: default;
  min-height: 26px;
}

.json-row:hover {
  background: var(--color-fill-1);
}

.json-row:hover .json-row-actions {
  opacity: 1;
}

.json-toggle {
  width: 16px;
  text-align: center;
  cursor: pointer;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  flex-shrink: 0;
  user-select: none;
}

.json-toggle:hover {
  color: var(--color-primary);
}

.json-toggle-placeholder {
  width: 16px;
  flex-shrink: 0;
}

.json-key {
  color: var(--color-text-1);
  font-weight: 500;
  flex-shrink: 0;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.json-colon {
  color: var(--color-text-3);
  flex-shrink: 0;
}

.json-type-hint {
  color: var(--color-text-4);
  font-size: var(--font-size-xs);
  font-style: italic;
}

.json-value {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
  padding: 0 4px;
  border-radius: var(--radius-sm);
}

.json-value:hover {
  background: var(--color-primary-bg);
}

.json-string { color: var(--color-json-string); }
.json-number { color: var(--color-json-number); }
.json-boolean { color: var(--color-json-boolean); }
.json-null { color: var(--color-json-null); font-style: italic; }

.json-row-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity var(--transition-fast);
  flex-shrink: 0;
  margin-left: auto;
}

.json-raw-area {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.json-raw-actions {
  display: flex;
  justify-content: flex-end;
  padding: var(--spacing-sm) var(--spacing-md);
  border-top: 1px solid var(--color-border-1);
}

.json-loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-overlay);
}
</style>
