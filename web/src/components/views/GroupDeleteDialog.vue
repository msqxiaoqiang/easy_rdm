<template>
  <a-modal
    :visible="visible"
    :title="$t('groupDelete.title')"
    :width="520"
    unmount-on-close
    @cancel="close"
  >
    <label class="dialog-label">{{ $t('groupDelete.pattern') }}</label>
    <a-input
      v-model="pattern"
      :placeholder="$t('groupDelete.patternPlaceholder')"
      @keydown.enter="handleAction"
    />

    <a-checkbox v-model="directDelete" class="check-row">
      {{ $t('groupDelete.directDelete') }}
    </a-checkbox>

    <!-- 预览结果 -->
    <template v-if="previewKeys.length > 0">
      <div class="preview-header">
        {{ $t('groupDelete.affectedKeys', { count: previewKeys.length }) }}
        <span v-if="previewTruncated" class="preview-truncated">{{ $t('groupDelete.truncated') }}</span>
      </div>
      <div class="preview-list">
        <div v-for="k in previewKeys" :key="k" class="preview-item" v-ellipsis-tip>{{ k }}</div>
      </div>
    </template>

    <template #footer>
      <a-button @click="close">{{ $t('common.cancel') }}</a-button>
      <a-button
        type="primary"
        status="danger"
        :disabled="loading"
        @click="handleAction"
      >
        {{ actionLabel }}
      </a-button>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { request } from '../../utils/request'
import { showMessage } from '@/utils/platform'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  connId: string
  visible: boolean
  groupPrefix: string
  separator: string
}>()

const emit = defineEmits<{ close: []; deleted: [] }>()
const { t } = useI18n()

const pattern = ref('')
const directDelete = ref(false)
const loading = ref(false)
const previewKeys = ref<string[]>([])
const previewTruncated = ref(false)
const confirmed = ref(false)

watch(() => props.visible, (v) => {
  if (v) {
    pattern.value = props.groupPrefix + '*'
    directDelete.value = false
    previewKeys.value = []
    previewTruncated.value = false
    confirmed.value = false
    loading.value = false
  }
})

const actionLabel = computed(() => {
  if (loading.value) return t('common.loading')
  if (directDelete.value) return t('common.delete')
  if (confirmed.value) return t('groupDelete.confirmDelete', { count: previewKeys.value.length })
  return t('groupDelete.viewAffected')
})

async function handleAction() {
  if (loading.value) return
  if (!pattern.value.trim()) return

  if (directDelete.value) {
    await doDelete()
  } else if (!confirmed.value) {
    await doPreview()
  } else {
    await doDelete()
  }
}

async function doPreview() {
  loading.value = true
  try {
    const res = await request<{ keys: string[]; total: number }>('scan_pattern_keys', {
      params: { conn_id: props.connId, pattern: pattern.value, limit: 10000 },
    })
    if (res.data) {
      previewKeys.value = res.data.keys || []
      previewTruncated.value = (res.data.total || 0) >= 10000
      confirmed.value = true
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function doDelete() {
  loading.value = true
  try {
    // 如果是直接删除模式，先扫描所有匹配的 key
    let keysToDelete = previewKeys.value
    if (directDelete.value && keysToDelete.length === 0) {
      const res = await request<{ keys: string[]; total: number }>('scan_pattern_keys', {
        params: { conn_id: props.connId, pattern: pattern.value, limit: 10000 },
      })
      keysToDelete = res.data?.keys || []
    }

    if (keysToDelete.length === 0) {
      showMessage('info', t('groupDelete.noKeys'))
      return
    }

    // 分批删除（每批 500）
    for (let i = 0; i < keysToDelete.length; i += 500) {
      const batch = keysToDelete.slice(i, i + 500)
      await request('delete_keys', {
        params: { conn_id: props.connId, keys: batch },
      })
    }

    showMessage('success', t('groupDelete.deleteSuccess', { count: keysToDelete.length }))
    emit('deleted')
    emit('close')
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

function close() { emit('close') }
</script>

<style scoped>
.check-row {
  margin-top: var(--spacing-md);
}

.preview-header {
  margin-top: var(--spacing-md);
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  font-weight: 600;
}

.preview-truncated {
  color: var(--color-warning);
  margin-left: var(--spacing-xs);
}

.preview-list {
  margin-top: var(--spacing-xs);
  max-height: 240px;
  overflow-y: auto;
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  padding: var(--spacing-xs);
}

.preview-item {
  padding: 3px var(--spacing-sm);
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border-radius: var(--radius-sm);
  transition: background var(--transition-fast);
}

.preview-item:hover {
  background: var(--color-fill-2);
}
</style>
