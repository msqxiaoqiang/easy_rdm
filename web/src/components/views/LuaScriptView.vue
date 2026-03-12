<template>
  <div class="lua-view">
    <!-- 工具栏 -->
    <div class="lua-toolbar">
      <a-select
        v-model="selectedScriptId"
        size="small"
        style="width: 200px"
        :placeholder="$t('lua.scriptName')"
        allow-clear
        @change="loadSelected"
      >
        <a-option v-for="s in savedScripts" :key="s.id" :value="s.id">{{ s.name }}</a-option>
      </a-select>
      <div style="flex:1"></div>
      <a-button size="small" @click="handleSave" :disabled="!script.trim()">
        {{ $t('common.save') }}
      </a-button>
      <a-button size="small" v-if="selectedScriptId" @click="deleteScript">
        {{ $t('common.delete') }}
      </a-button>
    </div>

    <!-- 保存脚本弹框（仅首次保存时弹出，编辑已有脚本直接覆盖） -->
    <a-modal
      :visible="saveDialogVisible"
      :title="$t('lua.saveScript')"
      :ok-text="$t('common.confirm')"
      :cancel-text="$t('common.cancel')"
      :ok-button-props="{ disabled: !saveScriptName.trim() }"
      :width="400"
      :mask-closable="true"
      unmount-on-close
      @ok="confirmSave"
      @cancel="saveDialogVisible = false"
    >
      <a-input
        v-model="saveScriptName"
        :placeholder="$t('lua.scriptName')"
        @keydown.enter="saveScriptName.trim() && confirmSave()"
      />
    </a-modal>

    <div class="lua-body">
      <!-- 脚本编辑器 -->
      <div class="editor-section">
        <label class="section-label">{{ $t('lua.script') }}</label>
        <a-textarea
          v-model="script"
          class="lua-editor"
          :placeholder="$t('lua.scriptPlaceholder')"
          :auto-size="{ minRows: 4 }"
        />
      </div>

      <!-- KEYS 和 ARGV + 执行 -->
      <div class="params-row">
        <div class="param-group">
          <label class="section-label">KEYS <span class="param-hint">{{ $t('lua.keysHint') }}</span></label>
          <a-input v-model="keysInput" size="small" placeholder="key1, key2, ..." />
        </div>
        <div class="param-group">
          <label class="section-label">ARGV <span class="param-hint">{{ $t('lua.argsHint') }}</span></label>
          <a-input v-model="argsInput" size="small" placeholder="arg1, arg2, ..." />
        </div>
        <div class="param-action">
          <a-button type="primary" :disabled="executing || !script.trim()" @click="executeScript">
            {{ executing ? $t('common.loading') : $t('lua.execute') }}
          </a-button>
        </div>
      </div>

      <!-- 执行结果 -->
      <div class="result-section" v-if="result !== null">
        <label class="section-label">{{ $t('lua.result') }}</label>
        <pre class="result-pre" :class="{ 'result-error': resultError }">{{ resultText }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { gmConfirm } from '../../utils/dialog'
import { showMessage } from '@/utils/platform'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()

interface SavedScript {
  id: string
  name: string
  script: string
  keys: string
  args: string
  updated: number
}

const script = ref('')
const scriptName = ref('')
const keysInput = ref('')
const argsInput = ref('')
const executing = ref(false)
const result = ref<any>(null)
const resultError = ref(false)
const resultText = ref('')
const savedScripts = ref<SavedScript[]>([])
const selectedScriptId = ref('')
const saveDialogVisible = ref(false)
const saveScriptName = ref('')

function handleSave() {
  if (!script.value.trim()) return
  if (selectedScriptId.value) {
    // 已选中保存的脚本 → 直接覆盖更新
    saveScript()
  } else {
    // 未选中 → 弹框输入名字
    saveScriptName.value = ''
    saveDialogVisible.value = true
  }
}

function confirmSave() {
  const name = saveScriptName.value.trim()
  if (!name) return
  saveDialogVisible.value = false
  scriptName.value = name
  saveScript()
}

function parseCSV(input: string): string[] {
  return input.split(',').map(s => s.trim()).filter(Boolean)
}

async function executeScript() {
  if (executing.value || !script.value.trim()) return
  executing.value = true
  result.value = null
  resultError.value = false
  try {
    const res = await request<{ result: any; error: string | null }>('lua_eval', {
      params: {
        conn_id: props.connId,
        script: script.value,
        keys: parseCSV(keysInput.value),
        args: parseCSV(argsInput.value),
      },
    })
    const data = res.data
    if (data?.error) {
      resultError.value = true
      resultText.value = data.error
    } else {
      resultText.value = formatResult(data?.result)
    }
    result.value = data
  } catch (e: any) {
    resultError.value = true
    resultText.value = e?.message || t('common.failed')
    result.value = { error: resultText.value }
  } finally {
    executing.value = false
  }
}

function formatResult(val: any): string {
  if (val === null || val === undefined) return '(nil)'
  if (typeof val === 'string') return val
  try {
    return JSON.stringify(val, null, 2)
  } catch {
    return String(val)
  }
}

async function loadScripts() {
  try {
    const res = await request<SavedScript[]>('lua_scripts_list', { params: {} })
    savedScripts.value = res.data || []
  } catch { /* ignore */ }
}

function loadSelected(val?: string) {
  // allow-clear 触发时 val 为 undefined
  if (!val) {
    selectedScriptId.value = ''
    scriptName.value = ''
    script.value = ''
    keysInput.value = ''
    argsInput.value = ''
    result.value = null
    return
  }
  const s = savedScripts.value.find(s => s.id === val)
  if (s) {
    scriptName.value = s.name
    script.value = s.script
    keysInput.value = s.keys || ''
    argsInput.value = s.args || ''
  }
  result.value = null
}

async function saveScript() {
  if (!scriptName.value.trim() || !script.value.trim()) return
  const id = selectedScriptId.value || ('lua_' + Date.now())
  try {
    await request('lua_script_save', {
      params: {
        id,
        name: scriptName.value,
        script: script.value,
        keys: keysInput.value,
        args: argsInput.value,
      },
    })
    showMessage('success', t('common.success'))
    selectedScriptId.value = id
    await loadScripts()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function deleteScript() {
  if (!selectedScriptId.value) return
  if (!await gmConfirm(t('lua.deleteConfirm'))) return
  try {
    await request('lua_script_delete', {
      params: { id: selectedScriptId.value },
    })
    selectedScriptId.value = ''
    loadSelected()
    await loadScripts()
    showMessage('success', t('common.success'))
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

onMounted(() => loadScripts())

watch(() => props.connId, () => {
  result.value = null
})
</script>

<style scoped>
.lua-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.lua-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  flex-shrink: 0;
}

.lua-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-md);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.section-label {
  display: block;
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  font-weight: 600;
  margin-bottom: var(--spacing-xs);
}

.param-hint {
  font-weight: 400;
  color: var(--color-text-4);
}

.editor-section {
  flex: 1;
  min-height: 120px;
  display: flex;
  flex-direction: column;
}

.lua-editor {
  flex: 1;
  min-height: 100px;
  font-family: var(--font-family-mono);
  font-size: var(--app-editor-font-size, var(--font-size-sm));
  line-height: 1.6;
}

.params-row {
  display: flex;
  gap: var(--spacing-md);
}

.param-group {
  flex: 1;
}

.param-action {
  flex-shrink: 0;
  display: flex;
  align-items: flex-end;
}

.result-section {
  flex-shrink: 0;
}

.result-pre {
  padding: var(--spacing-sm);
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-sm);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-1);
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  max-height: 200px;
  overflow-y: auto;
}

.result-error {
  color: var(--color-error);
  border-color: var(--color-error);
}
</style>
