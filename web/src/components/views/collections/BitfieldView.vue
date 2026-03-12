<template>
  <div class="bitfield-view">
    <div class="collection-toolbar">
      <a-button size="small" type="primary" @click="addFieldDef">+ {{ $t('bitfield.addField') }}</a-button>
      <a-button size="small" @click="loadValues" :disabled="!fieldDefs.length">{{ $t('common.refresh') }}</a-button>
      <div class="toolbar-spacer"></div>
    </div>

    <div class="collection-table-wrap">
      <table class="collection-table">
        <thead>
          <tr>
            <th>{{ $t('bitfield.type') }}</th>
            <th>{{ $t('bitmap.offset') }}</th>
            <th>{{ $t('collection.value') }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(fd, idx) in fieldDefs" :key="idx">
            <td>
              <a-select v-model="fd.type" size="small" class="inline-edit" @change="loadValues">
                <a-option v-for="t in typeOptions" :key="t" :value="t">{{ t }}</a-option>
              </a-select>
            </td>
            <td>
              <a-input v-model="fd.offset" size="small" class="inline-edit" style="width:80px" @change="loadValues" />
            </td>
            <td class="mono">
              <template v-if="editingIdx === idx">
                <a-input-number v-model="editingValue" size="small" class="inline-edit" @keyup.enter="saveEdit(idx)" @keyup.escape="editingIdx = -1" hide-button />
              </template>
              <template v-else>
                <span class="cell-text" v-ellipsis-tip @dblclick="startEdit(idx)">{{ fd.value ?? '—' }}</span>
              </template>
            </td>
            <td class="col-actions">
              <template v-if="editingIdx === idx">
                <a-button size="mini" @click="saveEdit(idx)">&#x2713;</a-button>
                <a-button size="mini" @click="editingIdx = -1">&#x2715;</a-button>
              </template>
              <template v-else>
                <a-button size="mini" @click="startEdit(idx)">&#x270E;</a-button>
                <a-button size="mini" @click="incrField(idx)">+1</a-button>
                <a-button size="mini" status="danger" @click="fieldDefs.splice(idx, 1); loadValues()">&#x2715;</a-button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!fieldDefs.length" class="empty-hint">{{ $t('bitfield.noFields') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { request } from '../../../utils/request'
import { useConnectionStore } from '../../../stores/connection'
import { useI18n } from 'vue-i18n'
import { showMessage } from '../../../utils/platform'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

interface FieldDef { type: string; offset: string; value: number | null }

const typeOptions = ['u8', 'u16', 'u32', 'u64', 'i8', 'i16', 'i32', 'i64']
const fieldDefs = ref<FieldDef[]>([])
const editingIdx = ref(-1)
const editingValue = ref('0')

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `bitfield:${props.keyName}`, {
    fieldDefs: fieldDefs.value.map(f => ({ ...f })),
  })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `bitfield:${oldKey}`, {
    fieldDefs: fieldDefs.value.map(f => ({ ...f })),
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `bitfield:${newKey}`)
  if (cached) {
    fieldDefs.value = cached.fieldDefs
    return
  }
  if (!fieldDefs.value.length) {
    fieldDefs.value = [{ type: 'u8', offset: '0', value: null }]
  }
  loadValues()
}, { immediate: true })

function addFieldDef() {
  const lastOffset = fieldDefs.value.length ? parseInt(fieldDefs.value[fieldDefs.value.length - 1].offset) || 0 : 0
  fieldDefs.value.push({ type: 'u8', offset: String(lastOffset + 8), value: null })
  loadValues()
}

async function loadValues() {
  if (!fieldDefs.value.length) return
  try {
    const fields = fieldDefs.value.map(f => ({ type: f.type, offset: f.offset }))
    const res = await request<any>('bitfield_get', {
      params: { conn_id: props.connId, key: props.keyName, fields },
    })
    const values: number[] = res.data?.values || []
    for (let i = 0; i < fieldDefs.value.length; i++) {
      fieldDefs.value[i].value = i < values.length ? values[i] : null
    }
  } catch (_e) { /* ignore */ }
}

function startEdit(idx: number) {
  editingIdx.value = idx
  editingValue.value = String(fieldDefs.value[idx].value ?? 0)
}

async function saveEdit(idx: number) {
  const fd = fieldDefs.value[idx]
  try {
    await request('bitfield_set', {
      params: { conn_id: props.connId, key: props.keyName, type: fd.type, offset: fd.offset, value: parseInt(editingValue.value) || 0 },
    })
    editingIdx.value = -1
    showMessage('success', t('common.success'))
    loadValues()
  } catch (e) { showError(e) }
}

async function incrField(idx: number) {
  const fd = fieldDefs.value[idx]
  try {
    await request('bitfield_incrby', {
      params: { conn_id: props.connId, key: props.keyName, type: fd.type, offset: fd.offset, increment: 1 },
    })
    loadValues()
  } catch (e) { showError(e) }
}
</script>

<style scoped>
@import './collection.css';

.bitfield-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
