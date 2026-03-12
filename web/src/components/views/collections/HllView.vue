<template>
  <div class="hll-view">
    <div class="collection-toolbar">
      <span class="footer-info">{{ $t('hll.cardinality') }}: <strong>{{ cardinality }}</strong></span>
      <a-button size="small" @click="loadCount">{{ $t('common.refresh') }}</a-button>
      <div class="toolbar-spacer"></div>
    </div>

    <div class="hll-content">
      <div class="hll-section">
        <label class="dialog-label">{{ $t('hll.addElements') }}</label>
        <a-textarea v-model="elementsText" :placeholder="$t('hll.addPlaceholder')" :auto-size="{ minRows: 3 }" />
        <a-button size="small" type="primary" @click="addElements" style="margin-top:var(--spacing-xs)">{{ $t('hll.add') }}</a-button>
      </div>

      <div class="hll-section" style="margin-top:var(--spacing-lg)">
        <label class="dialog-label">{{ $t('hll.mergeFrom') }}</label>
        <div class="merge-row">
          <a-input v-model="mergeKey" size="small" :placeholder="$t('hll.mergeKeyPlaceholder')" @press-enter="doMerge" />
          <a-button size="small" type="primary" @click="doMerge" :disabled="!mergeKey.trim()">{{ $t('hll.merge') }}</a-button>
        </div>
      </div>
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

const cardinality = ref(0)
const elementsText = ref('')
const mergeKey = ref('')

onBeforeUnmount(() => {
  connectionStore.saveKeyDetailCache(props.connId, `hll:${props.keyName}`, { cardinality: cardinality.value })
})

watch(() => props.keyName, (newKey, oldKey) => {
  if (oldKey) connectionStore.saveKeyDetailCache(props.connId, `hll:${oldKey}`, { cardinality: cardinality.value })
  const cached = connectionStore.getKeyDetailCache(props.connId, `hll:${newKey}`)
  if (cached) {
    cardinality.value = cached.cardinality
    return
  }
  loadCount()
}, { immediate: true })

async function loadCount() {
  try {
    const res = await request<any>('pfcount', {
      params: { conn_id: props.connId, key: props.keyName },
    })
    cardinality.value = res.data?.count ?? 0
  } catch (_e) { /* ignore */ }
}

async function addElements() {
  const elements = elementsText.value.split('\n').map(s => s.trim()).filter(Boolean)
  if (!elements.length) return
  try {
    await request('pfadd', {
      params: { conn_id: props.connId, key: props.keyName, elements },
    })
    elementsText.value = ''
    showMessage('success', t('common.success'))
    loadCount()
  } catch (e) { showError(e) }
}

async function doMerge() {
  const src = mergeKey.value.trim()
  if (!src) return
  try {
    await request('pfmerge', {
      params: { conn_id: props.connId, key: props.keyName, source_keys: [src] },
    })
    mergeKey.value = ''
    showMessage('success', t('common.success'))
    loadCount()
  } catch (e) { showError(e) }
}
</script>

<style scoped>
@import './collection.css';

.hll-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.hll-content {
  flex: 1;
  padding: var(--spacing-md);
  overflow: auto;
}

.hll-section {
  max-width: 500px;
}

.merge-row {
  display: flex;
  gap: var(--spacing-xs);
  align-items: center;
}

.merge-row .dialog-input {
  flex: 1;
}
</style>
