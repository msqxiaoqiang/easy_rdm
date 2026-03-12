<template>
  <a-modal
    :visible="visible"
    :title="$t('dataImport.title')"
    :width="520"
    unmount-on-close
    @cancel="close"
  >
    <label class="dialog-label">{{ $t('dataImport.format') }}</label>
    <div class="scope-options">
      <label class="radio-option" v-for="opt in formatOptions" :key="opt.value">
        <input type="radio" v-model="format" :value="opt.value" name="format" />
        <span>{{ opt.label }}</span>
      </label>
    </div>

    <label class="dialog-label">{{ $t('dataImport.conflict') }}</label>
    <div class="scope-options">
      <label class="radio-option" v-for="opt in conflictOptions" :key="opt.value">
        <input type="radio" v-model="conflictMode" :value="opt.value" name="conflict" />
        <span>{{ opt.label }}</span>
      </label>
    </div>

    <label class="dialog-label">{{ $t('dataImport.selectFile') }}</label>
    <div class="file-select-row">
      <a-input :model-value="fileName" readonly :placeholder="$t('dataImport.selectFilePlaceholder')" />
      <a-button size="small" @click="selectFile">{{ $t('dataExport.chooseDir') }}</a-button>
    </div>

    <!-- 导入结果 -->
    <div v-if="importResult" class="import-result">
      <span class="result-imported">{{ $t('dataImport.imported', { count: importResult.imported }) }}</span>
      <span v-if="importResult.skipped > 0" class="result-skipped">{{ $t('dataImport.skipped', { count: importResult.skipped }) }}</span>
      <span v-if="importResult.failed > 0" class="result-failed">{{ $t('dataImport.importFailed', { count: importResult.failed }) }}</span>
    </div>

    <template #footer>
      <a-button @click="close">{{ $t('common.cancel') }}</a-button>
      <a-button type="primary" :disabled="loading || !hasFile" @click="doImport">
        {{ loading ? $t('common.loading') : $t('common.import') }}
      </a-button>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { showMessage, hasNativeFileDialog, chooseFileWithData, getHttpBaseUrl } from '@/utils/platform'

const props = defineProps<{
  connId: string
  visible: boolean
}>()

const emit = defineEmits<{ close: []; imported: [] }>()
const { t } = useI18n()

const format = ref('json')
const conflictMode = ref('skip')
const loading = ref(false)
const importResult = ref<{ imported: number; skipped: number; failed: number } | null>(null)

// GM 环境存储文件路径，非 GM 环境存储 File 对象
const gmFilePath = ref('')
const browserFile = ref<File | null>(null)
const fileName = ref('')

const hasFile = computed(() => !!gmFilePath.value || !!browserFile.value)

const gmEnv = computed(() => hasNativeFileDialog())

const formatOptions = [
  { value: 'json', label: 'JSON' },
  { value: 'csv', label: 'CSV' },
  { value: 'redis_cmd', get label() { return t('format.redisCmd') } },
]

const conflictOptions = [
  { value: 'skip', get label() { return t('dataImport.conflictSkip') } },
  { value: 'overwrite', get label() { return t('dataImport.conflictOverwrite') } },
]

watch(() => props.visible, (v) => {
  if (v) {
    format.value = 'json'
    conflictMode.value = 'skip'
    importResult.value = null
    loading.value = false
    gmFilePath.value = ''
    browserFile.value = null
    fileName.value = ''
  }
})

function selectFile() {
  chooseFileWithData((result) => {
    if (result.path) {
      gmFilePath.value = result.path
      fileName.value = result.path.split('/').pop() || result.path
    } else if (result.file) {
      browserFile.value = result.file
      fileName.value = result.file.name
    }
  }, { accept: '.json,.csv,.txt' })
}

async function doImport() {
  if (loading.value || !hasFile.value) return
  loading.value = true
  importResult.value = null

  try {
    if (gmEnv.value) {
      // GM 环境：通过 RPC 传文件路径
      const res = await request<{ imported: number; skipped: number; failed: number }>('import_keys_file', {
        params: {
          conn_id: props.connId,
          format: format.value,
          file_path: gmFilePath.value,
          conflict_mode: conflictMode.value,
        },
      })
      importResult.value = res.data || { imported: 0, skipped: 0, failed: 0 }
    } else {
      // 非 GM 环境：通过 HTTP 上传文件，避免在 JS 内存中持有全量内容
      const formData = new FormData()
      formData.append('file', browserFile.value!)
      formData.append('conn_id', props.connId)
      formData.append('format', format.value)
      formData.append('conflict_mode', conflictMode.value)
      const controller = new AbortController()
      const timer = setTimeout(() => controller.abort(), 120000) // 120秒超时
      try {
        const response = await fetch(`${getHttpBaseUrl()}/upload_keys_import`, {
          method: 'POST',
          body: formData,
          signal: controller.signal,
        })
        clearTimeout(timer)
        const result = await response.json()
        if (result.code !== 200) {
          throw new Error(result.msg || t('common.failed'))
        }
        importResult.value = result.data || { imported: 0, skipped: 0, failed: 0 }
      } catch (e: any) {
        clearTimeout(timer)
        if (e.name === 'AbortError') {
          throw new Error('导入超时，请检查文件大小或格式')
        }
        throw e
      }
    }

    if (importResult.value && importResult.value.imported > 0) {
      showMessage('success', t('dataImport.imported', { count: importResult.value.imported }))
      emit('imported')
    } else if (importResult.value && importResult.value.imported === 0 && importResult.value.failed === 0) {
      showMessage('info', t('dataImport.skipped', { count: importResult.value.skipped || 0 }))
    } else if (importResult.value && importResult.value.failed > 0) {
      showMessage('error', t('dataImport.importFailed', { count: importResult.value.failed }))
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

function close() { emit('close') }
</script>

<style scoped>
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

.file-select-row {
  display: flex;
  gap: var(--spacing-xs);
  margin-top: 4px;
}

.file-select-row .arco-input-wrapper {
  flex: 1;
}

.import-result {
  display: flex;
  gap: var(--spacing-md);
  margin-top: var(--spacing-sm);
  font-size: var(--font-size-xs);
}

.result-imported {
  color: var(--color-success);
  font-weight: 600;
}

.result-skipped {
  color: var(--color-warning);
}

.result-failed {
  color: var(--color-error);
}
</style>
