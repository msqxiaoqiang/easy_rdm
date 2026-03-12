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
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in members" :key="m" :class="{ selected: selected.includes(m) }">
            <td class="col-check"><input type="checkbox" :checked="selected.includes(m)" @change="toggleSelect(m)" /></td>
            <td class="mono"><span class="cell-text" v-ellipsis-tip>{{ m }}</span></td>
            <td class="col-actions">
              <button class="mini-btn" :data-tooltip="$t('common.copy')" data-tooltip-pos="bottom" @click="copyRow(m)"><IconCopy :size="14" /></button>
              <button class="mini-btn danger" :data-tooltip="$t('common.delete')" data-tooltip-pos="bottom" @click="removeMembers([m])"><IconDelete :size="14" /></button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!members.length && !loading" class="empty-hint">{{ $t('common.noData') }}</div>
    </div>

    <div class="collection-footer">
      <span class="footer-info">{{ members.length }} {{ $t('collection.membersLoaded') }}</span>
      <a-button v-if="cursor !== 0" size="mini" @click="loadMore">{{ $t('collection.loadMore') }}</a-button>
    </div>

    <!-- Add Dialog -->
    <a-modal :visible="showAdd" :title="$t('collection.addMember')" :width="400" unmount-on-close @cancel="showAdd = false">
      <label class="dialog-label">{{ $t('collection.member') }}</label>
      <a-textarea v-model="newMember" :placeholder="$t('collection.oneMemberPerLine')" :auto-size="{ minRows: 3, maxRows: 10 }" />
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
import { IconSearch, IconClose, IconCopy, IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

const members = ref<string[]>([])
const cursor = ref<number>(0)
const pattern = ref('')
const searchMode = ref(false)
const loading = ref(false)
const selected = ref<string[]>([])

// Add dialog
const showAdd = ref(false)
const newMember = ref('')

const allChecked = computed(() => members.value.length > 0 && selected.value.length === members.value.length)

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `set:${props.keyName}`, {
    members: [...members.value], cursor: cursor.value, selected: [...selected.value],
  })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `set:${oldKey}`, {
    members: [...members.value], cursor: cursor.value, selected: [...selected.value],
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `set:${newKey}`)
  if (cached) {
    members.value = cached.members; cursor.value = cached.cursor; selected.value = cached.selected
    return
  }
  resetAndLoad()
}, { immediate: true })

async function resetAndLoad() {
  members.value = []
  cursor.value = 0
  selected.value = []
  await loadMore()
}

async function loadMore() {
  loading.value = true
  try {
    const res = await request<any>('sscan_members', {
      params: { conn_id: props.connId, key: props.keyName, pattern: pattern.value || '*', cursor: cursor.value, count: 100 },
    })
    if (res.data) {
      const existing = new Set(members.value)
      const newMembers = (res.data.members || []).filter((m: string) => !existing.has(m))
      members.value.push(...newMembers)
      cursor.value = res.data.cursor ?? 0
    }
  } catch (_e) { /* 加载失败静默 */ }
  loading.value = false
}

function toggleAll(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  selected.value = checked ? [...members.value] : []
}

function toggleSelect(m: string) {
  const idx = selected.value.indexOf(m)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(m)
}

async function removeMembers(list: string[], skipConfirm = false) {
  if (!list.length) return
  if (!skipConfirm && !await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('srem_members', {
      params: { conn_id: props.connId, key: props.keyName, members: list },
    })
    members.value = members.value.filter(m => !list.includes(m))
    selected.value = selected.value.filter(s => !list.includes(s))
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
  newMember.value = ''
  showAdd.value = true
}

async function addMembers() {
  const list = newMember.value.split('\n').map(s => s.trim()).filter(Boolean)
  if (!list.length) return
  try {
    await request('sadd_members', {
      params: { conn_id: props.connId, key: props.keyName, members: list },
    })
    showAdd.value = false
    showMessage('success', t('common.success'))
    resetAndLoad()
  } catch (e) { showError(e) }
}
</script>

<style scoped>
@import './collection.css';
</style>
