<template>
  <a-modal :visible="visible" :title="$t('migrate.title')" :width="520" unmount-on-close @cancel="$emit('close')">
    <!-- 目标连接 -->
    <div class="form-row">
      <label class="form-label">{{ $t('migrate.targetConn') }}</label>
      <a-select v-model="targetConnId" :placeholder="$t('migrate.selectTarget')">
        <a-option v-for="c in availableConns" :key="c.id" :value="c.id">
          {{ c.name }} ({{ c.host }}:{{ c.port }})
        </a-option>
      </a-select>
    </div>

    <!-- 键来源 -->
    <div class="form-row">
      <label class="form-label">{{ $t('migrate.keySource') }}</label>
      <a-select v-model="keySource">
        <a-option value="selected">{{ $t('batch.fromSelected') }} ({{ selectedKeys.length }})</a-option>
        <a-option value="pattern">{{ $t('batch.fromPattern') }}</a-option>
      </a-select>
    </div>

    <div v-if="keySource === 'pattern'" class="form-row">
      <label class="form-label">{{ $t('batch.pattern') }}</label>
      <a-input v-model="pattern" placeholder="user:*" />
    </div>

    <!-- 冲突处理 -->
    <div class="form-row">
      <a-checkbox v-model="replace">{{ $t('migrate.replaceExisting') }}</a-checkbox>
    </div>

    <!-- 版本提示 -->
    <div v-if="result && result.source_ver && result.target_ver" class="version-info">
      {{ $t('migrate.versionInfo') }}: {{ result.source_ver }} → {{ result.target_ver }}
    </div>

    <!-- 结果 -->
    <div v-if="result" class="result-box" :class="{ error: result.failed > 0 }">
      <div>{{ $t('migrate.migrated', { count: result.success }) }}</div>
      <div v-if="result.failed > 0">{{ $t('common.failed') }}: {{ result.failed }}</div>
      <div v-if="result.skipped > 0">{{ $t('migrate.skipped') }}: {{ result.skipped }}</div>
      <div v-if="result.errors?.length" class="error-list">
        <div v-for="(e, i) in result.errors" :key="i" class="error-item">{{ e }}</div>
      </div>
    </div>

    <template #footer>
      <a-button @click="$emit('close')">{{ $t('common.cancel') }}</a-button>
      <a-button type="primary" :disabled="executing || !canExecute" :loading="executing" @click="execute">
        {{ $t('migrate.start') }}
      </a-button>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { useConnectionStore } from '../../stores/connection'
import { showMessage } from '@/utils/platform'

const props = defineProps<{
  connId: string
  visible: boolean
  selectedKeys: string[]
}>()

const emit = defineEmits<{ close: []; done: [] }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()

const targetConnId = ref('')
const keySource = ref<'selected' | 'pattern'>('selected')
const pattern = ref('*')
const replace = ref(false)
const executing = ref(false)
const result = ref<{
  success: number; failed: number; skipped: number; total: number
  source_ver: string; target_ver: string; errors?: string[]
} | null>(null)

// 已连接的其他连接
const availableConns = computed(() => {
  return connectionStore.connections.filter(c => {
    if (c.id === props.connId) return false
    const state = connectionStore.getConnState(c.id)
    return state.status === 'connected'
  })
})

const canExecute = computed(() => {
  if (!targetConnId.value) return false
  if (keySource.value === 'selected' && props.selectedKeys.length === 0) return false
  if (keySource.value === 'pattern' && !pattern.value.trim()) return false
  return true
})

watch(() => props.visible, (v) => {
  if (v) {
    result.value = null
    targetConnId.value = ''
    if (props.selectedKeys.length > 0) keySource.value = 'selected'
  }
})

async function execute() {
  executing.value = true
  result.value = null

  try {
    const params: Record<string, any> = {
      source_conn_id: props.connId,
      target_conn_id: targetConnId.value,
      replace: replace.value,
    }

    if (keySource.value === 'selected') {
      params.keys = props.selectedKeys
    } else {
      params.pattern = pattern.value
      params.limit = 50000
    }

    const res = await request<typeof result.value>('migrate_keys', { params })
    result.value = res.data || { success: 0, failed: 0, skipped: 0, total: 0, source_ver: '', target_ver: '' }

    if (result.value && result.value.success > 0) {
      showMessage('success', t('migrate.migrated', { count: result.value.success }))
      emit('done')
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    executing.value = false
  }
}
</script>

<style scoped>
.version-info {
  padding: 6px 10px;
  background: var(--color-fill-1);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  margin-bottom: 12px;
}
.result-box {
  padding: 10px 12px;
  background: var(--color-success-bg, rgba(0, 180, 42, 0.08));
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  color: var(--color-text-1);
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
</style>
