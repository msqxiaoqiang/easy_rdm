<template>
  <a-modal :visible="visible" :title="$t('batch.title')" :width="520" :render-to-body="false" unmount-on-close @cancel="$emit('close')">
    <!-- 操作类型选择 -->
    <div class="form-row">
      <label class="form-label">{{ $t('batch.operation') }}</label>
      <a-select v-model="operation">
        <a-option value="setTTL">{{ $t('batch.setTTL') }}</a-option>
        <a-option v-if="!isCluster" value="moveDB">{{ $t('batch.moveDB') }}</a-option>
        <a-option value="delete">{{ $t('batch.delete') }}</a-option>
      </a-select>
    </div>

    <!-- 键匹配 -->
    <div class="form-row">
      <label class="form-label">{{ $t('batch.pattern') }}</label>
      <a-input v-model="pattern" placeholder="user:*" @keydown.enter="handleAction" />
    </div>

    <a-checkbox v-model="directExecute" class="check-row">
      {{ $t('batch.directExecute') }}
    </a-checkbox>

    <!-- TTL 设置 -->
    <div v-if="operation === 'setTTL'" class="form-row">
      <label class="form-label">TTL</label>
      <div class="input-with-hint">
        <a-input-number v-model="ttlValue" :min="-1" class="flex-1" />
        <span class="input-hint">{{ $t('newKey.ttlHint') }}</span>
      </div>
    </div>

    <!-- 目标 DB -->
    <div v-if="operation === 'moveDB'" class="form-row">
      <label class="form-label">{{ $t('batch.targetDB') }}</label>
      <a-select v-model="targetDB">
        <a-option v-for="i in 16" :key="i - 1" :value="i - 1">db{{ i - 1 }}</a-option>
      </a-select>
    </div>

    <div v-if="operation === 'moveDB'" class="form-row">
      <a-checkbox v-model="replace">{{ $t('batch.replaceExisting') }}</a-checkbox>
    </div>

    <!-- 预览列表 -->
    <template v-if="previewKeys.length > 0">
      <div class="preview-header">
        {{ $t('batch.matchedKeys') }} ({{ previewKeys.length }})
        <span v-if="previewTruncated" class="preview-truncated">{{ $t('groupDelete.truncated') }}</span>
      </div>
      <div class="preview-list">
        <div v-for="k in previewKeys.slice(0, 100)" :key="k" class="preview-item" v-ellipsis-tip>{{ k }}</div>
        <div v-if="previewKeys.length > 100" class="preview-more">
          ... {{ $t('batch.andMore', { count: previewKeys.length - 100 }) }}
        </div>
      </div>
    </template>

    <!-- 结果 -->
    <div v-if="result" class="result-box" :class="{ error: result.failed > 0 }">
      <div>{{ $t('batch.completed') }}: {{ result.success }} {{ $t('common.success') }}</div>
      <div v-if="result.failed > 0">{{ result.failed }} {{ $t('common.failed') }}</div>
      <div v-if="(result.skipped ?? 0) > 0">{{ $t('batch.skipped') }}: {{ result.skipped }}</div>
      <div v-if="result.errors?.length" class="error-list">
        <div v-for="(e, i) in result.errors" :key="i" class="error-item">{{ e }}</div>
      </div>
    </div>

    <template #footer>
      <a-button @click="$emit('close')">{{ $t('common.cancel') }}</a-button>
      <a-button
        type="primary"
        :status="operation === 'delete' ? 'danger' : undefined"
        :disabled="loading || !canExecute"
        :loading="loading"
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
  isCluster?: boolean
}>()

const emit = defineEmits<{ close: []; done: [] }>()
const { t } = useI18n()

const operation = ref<'setTTL' | 'moveDB' | 'delete'>('setTTL')
const pattern = ref('*')
const ttlValue = ref(3600)
const targetDB = ref(0)
const replace = ref(false)
const loading = ref(false)
const result = ref<{ success: number; failed: number; skipped?: number; errors?: string[] } | null>(null)
const directExecute = ref(false)
const previewKeys = ref<string[]>([])
const previewTruncated = ref(false)
const confirmed = ref(false)

const canExecute = computed(() => {
  if (!pattern.value.trim()) return false
  if (operation.value === 'setTTL' && (ttlValue.value === null || ttlValue.value === undefined || (ttlValue.value < 1 && ttlValue.value !== -1))) return false
  return true
})

const actionLabel = computed(() => {
  if (loading.value) return t('common.loading')
  if (directExecute.value) return t('batch.executeNow')
  if (confirmed.value) return t('batch.confirmExecute', { count: previewKeys.value.length })
  return t('batch.previewFirst')
})

watch(() => props.visible, (v) => {
  if (v) {
    result.value = null
    directExecute.value = false
    previewKeys.value = []
    previewTruncated.value = false
    confirmed.value = false
    loading.value = false
  }
})

// 表单数据变更时重置确认状态
watch([operation, pattern, ttlValue, targetDB, replace, directExecute], () => {
  if (confirmed.value) {
    confirmed.value = false
    previewKeys.value = []
    previewTruncated.value = false
    result.value = null
  }
})

async function handleAction() {
  if (loading.value) return
  if (!pattern.value.trim()) return

  if (directExecute.value) {
    await doScanAndExecute()
  } else if (!confirmed.value) {
    await doPreview()
  } else {
    await doExecutePreviewedKeys()
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
      if (previewKeys.value.length === 0) {
        showMessage('info', t('groupDelete.noKeys'))
      } else {
        confirmed.value = true
      }
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function doScanAndExecute() {
  loading.value = true
  result.value = null
  try {
    const scanRes = await request<{ keys: string[]; total: number }>('scan_pattern_keys', {
      params: { conn_id: props.connId, pattern: pattern.value, limit: 10000 },
    })
    const keys = scanRes.data?.keys || []
    if (keys.length === 0) {
      showMessage('info', t('groupDelete.noKeys'))
      return
    }
    await doExecute(keys)
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function doExecutePreviewedKeys() {
  loading.value = true
  result.value = null
  try {
    await doExecute(previewKeys.value)
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

async function doExecute(keys: string[]) {
  if (operation.value === 'delete') {
    const res = await request<{ success: number; failed: number }>('batch_delete_keys', {
      params: { conn_id: props.connId, keys },
    })
    result.value = res.data || { success: 0, failed: 0 }
  } else if (operation.value === 'setTTL') {
    const res = await request<{ success: number; failed: number }>('batch_set_ttl', {
      params: {
        conn_id: props.connId,
        keys,
        ttl: ttlValue.value,
      },
    })
    result.value = res.data || { success: 0, failed: 0 }
  } else if (operation.value === 'moveDB') {
    const res = await request<{ success: number; failed: number; errors?: string[] }>('batch_move_db', {
      params: {
        conn_id: props.connId,
        keys,
        target_db: targetDB.value,
        replace: replace.value,
      },
    })
    result.value = res.data || { success: 0, failed: 0 }
  }

  if (result.value && result.value.success > 0) {
    showMessage('success', t('common.success'))
    emit('done')
  }
}
</script>

<style scoped>
.check-row {
  margin-top: var(--spacing-md);
  margin-bottom: var(--spacing-md);
}
.input-with-hint {
  display: flex;
  align-items: center;
  gap: 8px;
}
.flex-1 { flex: 1; }
.input-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  white-space: nowrap;
}
.result-box {
  padding: 10px 12px;
  background: var(--color-success-bg, rgba(0, 180, 42, 0.08));
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-1);
  margin-top: var(--spacing-md);
}
.result-box.error {
  background: var(--color-error-bg, rgba(245, 63, 63, 0.08));
}
.error-list {
  margin-top: 6px;
  max-height: 80px;
  overflow-y: auto;
}
.error-item {
  font-size: var(--font-size-xs);
  color: var(--color-error);
  font-family: var(--font-family-mono);
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
.preview-more {
  padding: 3px var(--spacing-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  font-style: italic;
}
</style>
