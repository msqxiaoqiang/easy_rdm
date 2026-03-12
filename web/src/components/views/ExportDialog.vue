<template>
  <a-modal
    :visible="visible"
    :title="$t('dataExport.title')"
    :width="520"
    unmount-on-close
    @cancel="close"
  >
    <!-- 选中键模式：显示 key 列表 -->
    <template v-if="hasSelectedKeys">
      <div class="selected-info">
        {{ $t('dataExport.selectedCount', { count: selectedKeys.length }) }}
      </div>
      <div class="selected-keys-list">
        <div v-for="key in selectedKeys" :key="key" class="selected-key-item">{{ key }}</div>
      </div>
    </template>
    <!-- 普通模式：显示范围选择 -->
    <template v-else>
      <label class="dialog-label">{{ $t('dataExport.scope') }}</label>
      <div class="scope-options">
        <label class="radio-option" v-for="opt in scopeOptions" :key="opt.value">
          <input type="radio" v-model="scope" :value="opt.value" name="scope" />
          <span>{{ opt.label }}</span>
        </label>
      </div>

      <template v-if="scope === 'pattern'">
        <label class="dialog-label">{{ $t('dataExport.pattern') }}</label>
        <a-input v-model="pattern" placeholder="e.g. user:*" />
      </template>
    </template>

    <label class="dialog-label">{{ $t('dataExport.format') }}</label>
    <div class="scope-options">
      <label class="radio-option" v-for="opt in formatOptions" :key="opt.value">
        <input type="radio" v-model="format" :value="opt.value" name="format" />
        <span>{{ opt.label }}</span>
      </label>
    </div>

    <!-- GM 环境：导出目录 -->
    <template v-if="gmEnv">
      <label class="dialog-label">{{ $t('dataExport.exportDir') }}</label>
      <div class="export-dir-row">
        <a-input v-model="exportDir" :placeholder="$t('dataExport.exportDirPlaceholder')" allow-clear />
        <a-button size="small" @click="chooseExportDir">{{ $t('dataExport.chooseDir') }}</a-button>
      </div>
    </template>

    <template #footer>
      <a-button @click="close">{{ $t('common.cancel') }}</a-button>
      <a-button type="primary" :disabled="loading" @click="doExport">
        {{ loading ? $t('common.loading') : $t('common.export') }}
      </a-button>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { showMessage, hasNativeFileDialog, chooseFolder, getHttpBaseUrl } from '@/utils/platform'

const props = defineProps<{
  connId: string
  visible: boolean
  selectedKeys: string[]
}>()

const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()

const scope = ref('all')
const pattern = ref('*')
const format = ref('json')
const loading = ref(false)
const exportDir = ref('')

const hasSelectedKeys = computed(() => props.selectedKeys.length > 0)

const gmEnv = computed(() => hasNativeFileDialog())

const scopeOptions = [
  { value: 'all', get label() { return t('dataExport.scopeAll') } },
  { value: 'pattern', get label() { return t('dataExport.scopePattern') } },
]

const formatOptions = [
  { value: 'json', label: 'JSON' },
  { value: 'csv', label: 'CSV' },
  { value: 'redis_cmd', get label() { return t('format.redisCmd') } },
]

watch(() => props.visible, (v) => {
  if (v) {
    scope.value = 'all'
    pattern.value = '*'
    format.value = 'json'
    loading.value = false
    exportDir.value = ''
  }
})

function buildExportParams(): Record<string, any> {
  if (hasSelectedKeys.value) {
    return {
      conn_id: props.connId,
      format: format.value,
      scope: 'selected',
      keys: props.selectedKeys,
    }
  }
  const params: Record<string, any> = {
    conn_id: props.connId,
    format: format.value,
    scope: scope.value,
  }
  if (scope.value === 'pattern') {
    params.pattern = pattern.value
  }
  return params
}

function chooseExportDir() {
  chooseFolder((path: string) => {
    exportDir.value = path
  }, exportDir.value)
}

async function doExport() {
  if (loading.value) return

  if (gmEnv.value) {
    if (!exportDir.value.trim()) {
      showMessage('error', t('dataExport.dirRequired'))
      return
    }
    loading.value = true
    try {
      const params = buildExportParams()
      params.file_path = exportDir.value.trim()
      const res = await request<{ path: string; count: number }>('export_keys_file', { params })
      showMessage('success', t('dataExport.exportSuccess', { count: res.data?.count || 0 }))
      close()
    } catch (e: any) {
      showMessage('error', e?.message || t('common.failed'))
    } finally {
      loading.value = false
    }
  } else {
    // 非 GM 环境：通过 HTTP 端点流式下载文件，避免在 JS 内存中持有全量内容
    loading.value = true
    try {
      const response = await fetch(`${getHttpBaseUrl()}/download_keys_export`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(buildExportParams()),
      })
      const blob = await response.blob()
      if (blob.size === 0) {
        showMessage('info', t('common.noData'))
        return
      }
      const disposition = response.headers.get('Content-Disposition') || ''
      const match = disposition.match(/filename="?([^"]+)"?/)
      const fileName = match?.[1] || 'export.json'
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = fileName
      a.click()
      URL.revokeObjectURL(url)
      showMessage('success', t('common.success'))
      close()
    } catch (e: any) {
      showMessage('error', e?.message || t('common.failed'))
    } finally {
      loading.value = false
    }
  }
}

function close() { emit('close') }
</script>

<style scoped>
.selected-info {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  padding: 6px 0;
  font-weight: 600;
}

.selected-keys-list {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--border-radius-sm, 4px);
  padding: 4px 0;
  margin-bottom: var(--spacing-sm);
}

.selected-key-item {
  padding: 2px 8px;
  font-size: var(--font-size-xs);
  color: var(--color-text-1);
  word-break: break-all;
  line-height: 1.6;
}

.selected-key-item:hover {
  background: var(--color-fill-2);
}

.scope-options {
  display: flex;
  gap: var(--spacing-md);
  margin-top: 4px;
}

.radio-option {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  cursor: pointer;
  user-select: none;
}

.radio-option input[type="radio"] {
  margin: 0;
  accent-color: var(--color-primary);
  cursor: pointer;
}

.export-dir-row {
  display: flex;
  gap: var(--spacing-xs);
  margin-top: 4px;
}

.export-dir-row .arco-input-wrapper {
  flex: 1;
}
</style>
