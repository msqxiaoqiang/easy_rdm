<template>
  <div class="collection-detail">
    <div class="collection-toolbar">
      <a-button size="mini" type="primary" @click="openAdd"><IconPlus :size="12" /> {{ $t('geo.addMember') }}</a-button>
      <a-button size="mini" @click="openDist">{{ $t('geo.calcDist') }}</a-button>
      <a-button size="mini" @click="openSearch">{{ $t('geo.searchNearby') }}</a-button>
      <div class="toolbar-spacer"></div>
      <label class="view-toggle">
        <a-checkbox v-model="mapMode" /> {{ $t('geo.mapView') }}
      </label>
      <a-button size="mini" status="danger" @click="removeSelected" :disabled="!selected.length">
        {{ $t('common.delete') }} ({{ selected.length }})
      </a-button>
    </div>

    <!-- Table / Map toggle area -->
    <div v-if="!mapMode" class="collection-table-wrap">
      <table class="collection-table">
        <thead>
          <tr>
            <th class="col-check"><input type="checkbox" @change="toggleAll" :checked="allChecked" /></th>
            <th>{{ $t('collection.member') }}</th>
            <th class="col-coord">{{ $t('geo.longitude') }}</th>
            <th class="col-coord">{{ $t('geo.latitude') }}</th>
            <th class="col-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.member" :class="{ selected: selected.includes(item.member) }">
            <td class="col-check"><input type="checkbox" :checked="selected.includes(item.member)" @change="toggleSelect(item.member)" /></td>
            <td class="mono"><span class="cell-text" v-ellipsis-tip>{{ item.member }}</span></td>
            <td class="col-coord mono">{{ item.longitude.toFixed(6) }}</td>
            <td class="col-coord mono">{{ item.latitude.toFixed(6) }}</td>
            <td class="col-actions">
              <button class="mini-btn" :data-tooltip="$t('common.copy')" data-tooltip-pos="bottom" @click="copyRow(item.member)"><IconCopy :size="14" /></button>
              <button class="mini-btn danger" :data-tooltip="$t('common.delete')" data-tooltip-pos="bottom" @click="removeMembers([item.member])"><IconDelete :size="14" /></button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!items.length && !loading" class="empty-hint">{{ $t('common.noData') }}</div>
    </div>

    <!-- Mini-map scatter view -->
    <div v-else class="geo-map-wrap">
      <svg v-if="items.length" class="geo-map" viewBox="0 0 400 300" preserveAspectRatio="xMidYMid meet">
        <!-- grid lines -->
        <line x1="0" y1="150" x2="400" y2="150" class="map-axis" />
        <line x1="200" y1="0" x2="200" y2="300" class="map-axis" />
        <!-- axis labels -->
        <text x="396" y="145" class="map-label" text-anchor="end">{{ mapBounds.maxLon.toFixed(2) }}</text>
        <text x="4" y="145" class="map-label" text-anchor="start">{{ mapBounds.minLon.toFixed(2) }}</text>
        <text x="204" y="12" class="map-label" text-anchor="start">{{ mapBounds.maxLat.toFixed(2) }}</text>
        <text x="204" y="296" class="map-label" text-anchor="start">{{ mapBounds.minLat.toFixed(2) }}</text>
        <!-- points -->
        <g v-for="pt in mapPoints" :key="pt.member">
          <circle :cx="pt.x" :cy="pt.y" r="5" class="map-point" @mouseenter="hoverMember = pt.member" @mouseleave="hoverMember = ''" />
          <text v-if="hoverMember === pt.member || items.length <= 20" :x="pt.x + 7" :y="pt.y + 4" class="map-point-label">{{ pt.member }}</text>
        </g>
      </svg>
      <div v-else class="empty-hint">{{ $t('common.noData') }}</div>
      <!-- hover info -->
      <div v-if="hoverMember" class="map-hover-info">
        {{ hoverMember }} — {{ hoverItem?.longitude.toFixed(6) }}, {{ hoverItem?.latitude.toFixed(6) }}
      </div>
    </div>

    <div class="collection-footer">
      <span class="footer-info">{{ items.length }} / {{ total }}</span>
      <div class="page-nav">
        <a-button size="mini" :disabled="page === 0" @click="page--; loadPage()">‹</a-button>
        <span>{{ page + 1 }} / {{ totalPages }}</span>
        <a-button size="mini" :disabled="page >= totalPages - 1" @click="page++; loadPage()">›</a-button>
      </div>
    </div>

    <!-- Add Dialog -->
    <a-modal :visible="showAdd" :title="$t('geo.addMember')" :width="400" unmount-on-close @cancel="showAdd = false">
      <label class="dialog-label">{{ $t('collection.member') }}</label>
      <a-input v-model="newName" />
      <label class="dialog-label">{{ $t('geo.longitude') }}</label>
      <a-input v-model="newLon" type="number" step="any" />
      <label class="dialog-label">{{ $t('geo.latitude') }}</label>
      <a-input v-model="newLat" type="number" step="any" />
      <template #footer>
        <a-button @click="showAdd = false">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="addMember">{{ $t('common.confirm') }}</a-button>
      </template>
    </a-modal>

    <!-- Distance Dialog -->
    <a-modal :visible="showDist" :title="$t('geo.calcDist')" :width="400" unmount-on-close @cancel="showDist = false">
      <label class="dialog-label">{{ $t('geo.member1') }}</label>
      <a-input v-model="distMember1" />
      <label class="dialog-label">{{ $t('geo.member2') }}</label>
      <a-input v-model="distMember2" />
      <label class="dialog-label">{{ $t('geo.unit') }}</label>
      <a-select v-model="distUnit">
        <a-option value="m">m</a-option>
        <a-option value="km">km</a-option>
        <a-option value="mi">mi</a-option>
        <a-option value="ft">ft</a-option>
      </a-select>
      <div v-if="distResult !== null" class="dist-result">
        {{ $t('geo.distance') }}: {{ distResult }} {{ distUnit }}
      </div>
      <template #footer>
        <a-button @click="showDist = false">{{ $t('common.close') }}</a-button>
        <a-button type="primary" @click="calcDist">{{ $t('geo.calculate') }}</a-button>
      </template>
    </a-modal>

    <!-- Search Dialog -->
    <a-modal :visible="showSearch" :title="$t('geo.searchNearby')" :width="400" unmount-on-close @cancel="showSearch = false">
      <label class="dialog-label">{{ $t('geo.longitude') }}</label>
      <a-input v-model="searchLon" type="number" step="any" />
      <label class="dialog-label">{{ $t('geo.latitude') }}</label>
      <a-input v-model="searchLat" type="number" step="any" />
      <label class="dialog-label">{{ $t('geo.radius') }}</label>
      <div class="dialog-row">
        <a-input v-model="searchRadius" type="number" step="any" />
        <a-select v-model="searchUnit" style="width:80px;flex:none">
          <a-option value="m">m</a-option>
          <a-option value="km">km</a-option>
          <a-option value="mi">mi</a-option>
        </a-select>
      </div>
      <div v-if="searchResults.length" class="search-results">
        <div v-for="r in searchResults" :key="r.name" class="search-result-item">
          {{ r.name }} — {{ r.dist.toFixed(2) }} {{ searchUnit }}
        </div>
      </div>
      <template #footer>
        <a-button @click="showSearch = false">{{ $t('common.close') }}</a-button>
        <a-button type="primary" @click="doSearch">{{ $t('common.search') }}</a-button>
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
import { IconCopy, IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

interface GeoItem { member: string; longitude: number; latitude: number }
interface GeoSearchResult { name: string; dist: number; longitude: number; latitude: number }

const items = ref<GeoItem[]>([])
const total = ref(0)
const page = ref(0)
const pageSize = 100
const loading = ref(false)
const selected = ref<string[]>([])

const showAdd = ref(false)
const newName = ref('')
const newLon = ref('0')
const newLat = ref('0')

const showDist = ref(false)
const distMember1 = ref('')
const distMember2 = ref('')
const distUnit = ref('m')
const distResult = ref<number | null>(null)

const showSearch = ref(false)
const searchLon = ref('0')
const searchLat = ref('0')
const searchRadius = ref('1000')
const searchUnit = ref('m')
const searchResults = ref<GeoSearchResult[]>([])

const mapMode = ref(false)
const hoverMember = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const allChecked = computed(() => items.value.length > 0 && selected.value.length === items.value.length)

const hoverItem = computed(() => items.value.find(i => i.member === hoverMember.value))

const mapBounds = computed(() => {
  if (!items.value.length) return { minLon: -180, maxLon: 180, minLat: -90, maxLat: 90 }
  let minLon = Infinity, maxLon = -Infinity, minLat = Infinity, maxLat = -Infinity
  for (const i of items.value) {
    if (i.longitude < minLon) minLon = i.longitude
    if (i.longitude > maxLon) maxLon = i.longitude
    if (i.latitude < minLat) minLat = i.latitude
    if (i.latitude > maxLat) maxLat = i.latitude
  }
  // add padding
  const lonPad = Math.max((maxLon - minLon) * 0.15, 0.001)
  const latPad = Math.max((maxLat - minLat) * 0.15, 0.001)
  return { minLon: minLon - lonPad, maxLon: maxLon + lonPad, minLat: minLat - latPad, maxLat: maxLat + latPad }
})

const mapPoints = computed(() => {
  const b = mapBounds.value
  const w = 400, h = 300
  return items.value.map(i => ({
    member: i.member,
    x: ((i.longitude - b.minLon) / (b.maxLon - b.minLon)) * w,
    y: h - ((i.latitude - b.minLat) / (b.maxLat - b.minLat)) * h,
  }))
})

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `geo:${props.keyName}`, {
    items: [...items.value], total: total.value, page: page.value, selected: [...selected.value],
  })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `geo:${oldKey}`, {
    items: [...items.value], total: total.value, page: page.value, selected: [...selected.value],
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `geo:${newKey}`)
  if (cached) {
    items.value = cached.items; total.value = cached.total; page.value = cached.page
    selected.value = cached.selected
    return
  }
  resetAndLoad()
}, { immediate: true })

async function resetAndLoad() {
  items.value = []
  selected.value = []
  page.value = 0
  await loadPage()
}

async function loadPage() {
  loading.value = true
  const start = page.value * pageSize
  const stop = start + pageSize - 1
  try {
    const res = await request<any>('geo_members', {
      params: { conn_id: props.connId, key: props.keyName, start, stop },
    })
    if (res.data) {
      items.value = res.data.members || []
      total.value = res.data.total ?? 0
    }
  } catch (_e) { /* ignore */ }
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

function removeSelected() { removeMembers([...selected.value]) }

async function copyRow(val: string) {
  try {
    await navigator.clipboard.writeText(val)
    showMessage('success', t('common.copySuccess'))
  } catch (_e) { /* ignore */ }
}

function openAdd() {
  newName.value = ''
  newLon.value = '0'
  newLat.value = '0'
  showAdd.value = true
}

async function addMember() {
  if (!newName.value) return
  try {
    await request('geo_add', {
      params: {
        conn_id: props.connId, key: props.keyName,
        members: [{ name: newName.value, longitude: parseFloat(newLon.value) || 0, latitude: parseFloat(newLat.value) || 0 }],
      },
    })
    showAdd.value = false
    showMessage('success', t('common.success'))
    resetAndLoad()
  } catch (e) { showError(e) }
}

function openDist() {
  distMember1.value = ''
  distMember2.value = ''
  distResult.value = null
  showDist.value = true
}

async function calcDist() {
  if (!distMember1.value || !distMember2.value) return
  try {
    const res = await request<any>('geo_dist', {
      params: { conn_id: props.connId, key: props.keyName, member1: distMember1.value, member2: distMember2.value, unit: distUnit.value },
    })
    distResult.value = res.data?.distance ?? null
  } catch (_e) { distResult.value = null }
}

function openSearch() {
  searchResults.value = []
  showSearch.value = true
}

async function doSearch() {
  try {
    const res = await request<any>('geo_search', {
      params: {
        conn_id: props.connId, key: props.keyName,
        longitude: parseFloat(searchLon.value) || 0,
        latitude: parseFloat(searchLat.value) || 0,
        radius: parseFloat(searchRadius.value) || 1000,
        unit: searchUnit.value,
      },
    })
    searchResults.value = res.data?.results || []
  } catch (_e) { searchResults.value = [] }
}
</script>

<style scoped>
@import './collection.css';

.col-coord {
  width: 130px;
  font-family: var(--font-family-mono);
}

.dist-result {
  margin-top: var(--spacing-sm);
  padding: var(--spacing-sm);
  background: var(--color-fill-1);
  border-radius: var(--radius-sm);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-sm);
  color: var(--color-primary);
}

.search-results {
  margin-top: var(--spacing-sm);
  max-height: 200px;
  overflow: auto;
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-sm);
}

.search-result-item {
  padding: 4px var(--spacing-sm);
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  border-bottom: 1px solid var(--color-border-1);
}

.search-result-item:last-child {
  border-bottom: none;
}

.view-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  cursor: pointer;
  flex-shrink: 0;
}

.geo-map-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-md);
  overflow: auto;
  position: relative;
}

.geo-map {
  width: 100%;
  max-width: 600px;
  max-height: 100%;
  background: var(--color-bg-1);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-sm);
}

.map-axis {
  stroke: var(--color-border-2);
  stroke-width: 0.5;
  stroke-dasharray: 4 2;
}

.map-label {
  font-size: var(--font-size-xs);
  fill: var(--color-text-3);
  font-family: var(--font-family-mono);
}

.map-point {
  fill: var(--color-primary);
  stroke: #fff;
  stroke-width: 1.5;
  cursor: pointer;
  transition: r 0.15s;
}

.map-point:hover {
  r: 7;
}

.map-point-label {
  font-size: var(--font-size-xs);
  fill: var(--color-text-2);
}

.map-hover-info {
  position: absolute;
  bottom: var(--spacing-md);
  left: 50%;
  transform: translateX(-50%);
  padding: 4px var(--spacing-sm);
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  color: var(--color-text-1);
  white-space: nowrap;
}
</style>
