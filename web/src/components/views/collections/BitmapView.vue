<template>
  <div class="bitmap-view">
    <div class="collection-toolbar">
      <label class="footer-info">{{ $t('bitmap.offset') }}:</label>
      <a-input-number v-model="startOffset" size="small" class="filter-input" :min="0" style="width:100px" hide-button @press-enter="loadBits" />
      <a-button size="small" @click="loadBits">{{ $t('common.refresh') }}</a-button>
      <div class="toolbar-spacer"></div>
      <span class="footer-info">BITCOUNT: {{ bitCount }}</span>
    </div>

    <div class="bitmap-grid-wrap">
      <div class="bitmap-grid">
        <div class="bitmap-row-header">
          <span class="bitmap-offset-label"></span>
          <span v-for="i in 8" :key="i" class="bitmap-col-label">{{ i - 1 }}</span>
        </div>
        <div v-for="(row, rowIdx) in bitRows" :key="rowIdx" class="bitmap-row">
          <span class="bitmap-offset-label">{{ startOffsetNum + rowIdx * 8 }}</span>
          <span
            v-for="(bit, colIdx) in row"
            :key="colIdx"
            class="bitmap-cell"
            :class="{ on: bit === 1 }"
            @click="toggleBit(rowIdx, colIdx)"
          >{{ bit }}</span>
        </div>
      </div>
      <div v-if="!bits.length && !loading" class="empty-hint">{{ $t('common.noData') }}</div>
    </div>

    <div class="collection-footer">
      <span class="footer-info">{{ $t('bitmap.showing') }} {{ bits.length }} {{ $t('bitmap.bits') }}</span>
      <div class="page-nav">
        <a-button size="small" :disabled="startOffsetNum === 0" @click="prevPage">&#x2039;</a-button>
        <a-button size="small" @click="nextPage">&#x203A;</a-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { request } from '../../../utils/request'
import { useConnectionStore } from '../../../stores/connection'
import { useI18n } from 'vue-i18n'
import { showMessage } from '../../../utils/platform'

const props = defineProps<{ connId: string; keyName: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const showError = (e: any) => { showMessage('error', e?.message || t('common.failed')) }

const bitsPerPage = 256
const startOffset = ref('0')
const bits = ref<number[]>([])
const bitCount = ref(0)
const loading = ref(false)

const startOffsetNum = computed(() => parseInt(startOffset.value) || 0)

const bitRows = computed(() => {
  const rows: number[][] = []
  for (let i = 0; i < bits.value.length; i += 8) {
    rows.push(bits.value.slice(i, i + 8))
  }
  return rows
})

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `bitmap:${props.keyName}`, {
    bits: [...bits.value], bitCount: bitCount.value, startOffset: startOffset.value,
  })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `bitmap:${oldKey}`, {
    bits: [...bits.value], bitCount: bitCount.value, startOffset: startOffset.value,
  })
  const cached = connectionStore.getKeyDetailCache(props.connId, `bitmap:${newKey}`)
  if (cached) {
    bits.value = cached.bits; bitCount.value = cached.bitCount; startOffset.value = cached.startOffset
    return
  }
  loadBits(); loadCount()
}, { immediate: true })

async function loadBits() {
  loading.value = true
  try {
    const res = await request<any>('bitmap_get_range', {
      params: { conn_id: props.connId, key: props.keyName, start: startOffsetNum.value, count: bitsPerPage },
    })
    bits.value = res.data?.bits || []
  } catch (_e) { bits.value = [] }
  loading.value = false
}

async function loadCount() {
  try {
    const res = await request<any>('bitmap_count', {
      params: { conn_id: props.connId, key: props.keyName },
    })
    bitCount.value = res.data?.count ?? 0
  } catch (_e) { /* ignore */ }
}

async function toggleBit(rowIdx: number, colIdx: number) {
  const offset = startOffsetNum.value + rowIdx * 8 + colIdx
  const current = bits.value[rowIdx * 8 + colIdx]
  const newVal = current === 1 ? 0 : 1
  try {
    await request('bitmap_set_bit', {
      params: { conn_id: props.connId, key: props.keyName, offset, value: newVal },
    })
    bits.value[rowIdx * 8 + colIdx] = newVal
    bits.value = [...bits.value]
    loadCount()
  } catch (e) { showError(e) }
}

function prevPage() {
  const v = Math.max(0, startOffsetNum.value - bitsPerPage)
  startOffset.value = String(v)
  loadBits()
}

function nextPage() {
  startOffset.value = String(startOffsetNum.value + bitsPerPage)
  loadBits()
}
</script>

<style scoped>
@import './collection.css';

.bitmap-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.bitmap-grid-wrap {
  flex: 1;
  overflow: auto;
  padding: var(--spacing-sm) var(--spacing-md);
}

.bitmap-grid {
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
}

.bitmap-row-header,
.bitmap-row {
  display: flex;
  gap: 2px;
  margin-bottom: 2px;
}

.bitmap-offset-label {
  width: 60px;
  text-align: right;
  padding-right: var(--spacing-xs);
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  line-height: 22px;
  flex-shrink: 0;
}

.bitmap-col-label {
  width: 22px;
  height: 18px;
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
}

.bitmap-cell {
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 2px;
  background: var(--color-fill-1);
  color: var(--color-text-3);
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
}

.bitmap-cell:hover {
  background: var(--color-fill-3);
}

.bitmap-cell.on {
  background: var(--color-primary);
  color: #fff;
}
</style>
