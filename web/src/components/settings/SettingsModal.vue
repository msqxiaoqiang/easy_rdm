<template>
  <a-modal
    :visible="visible"
    :title="$t('settings.title')"
    :width="640"
    :footer="false"
    :mask-closable="true"
    unmount-on-close
    @cancel="handleClose"
  >
      <div class="settings-body">
        <!-- 左侧 Tab -->
        <div class="settings-tabs">
          <a-button
            v-for="tab in tabs"
            :key="tab.key"
            :class="['settings-tab', { active: activeTab === tab.key }]"
            type="text"
            long
            @click="activeTab = tab.key"
          >{{ $t(tab.label) }}</a-button>
        </div>

        <!-- 右侧内容 -->
        <div class="settings-content">
          <!-- 常规配置 -->
          <div v-show="activeTab === 'general'" class="settings-section">
            <div class="form-row">
              <label class="form-label">{{ $t('settings.theme') }}</label>
              <a-radio-group v-model="form.theme">
                <a-radio v-for="opt in themeOptions" :key="opt.value" :value="opt.value">
                  {{ $t(opt.label) }}
                </a-radio>
              </a-radio-group>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.language') }}</label>
              <a-select v-model="form.language" style="width: 160px">
                <a-option v-for="lang in languages" :key="lang.value" :value="lang.value">
                  {{ lang.label }}
                </a-option>
              </a-select>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.scanCount') }}</label>
              <a-input-number v-model="form.scanCount" :min="10" :max="10000" style="width: 120px" />
            </div>
          </div>

          <!-- 连接配置 -->
          <div v-show="activeTab === 'connection'" class="settings-section">
            <div class="form-row">
              <label class="form-label">
                <a-switch v-model="form.autoReconnect" size="small" />
                {{ $t('settings.autoReconnect') }}
              </label>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.retryInterval') }}</label>
              <a-input-number v-model="form.retryInterval" :min="1" :max="60" style="width: 120px" />
              <span class="form-hint">{{ $t('settings.seconds') }}</span>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.maxRetries') }}</label>
              <a-input-number v-model="form.maxRetries" :min="0" :max="100" style="width: 120px" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.heartbeatInterval') }}</label>
              <a-input-number v-model="form.heartbeatInterval" :min="5" :max="300" style="width: 120px" />
              <span class="form-hint">{{ $t('settings.seconds') }}</span>
            </div>
          </div>

          <!-- 高级配置 -->
          <div v-show="activeTab === 'advanced'" class="settings-section">
            <div class="section-title">{{ $t('connection.cli') }}</div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.cliFontSize') }}</label>
              <a-input-number v-model="form.cliFontSize" :min="10" :max="24" :step="1" style="width: 100px" />
            </div>
            <div class="section-title">{{ $t('connection.keyDetail') }}</div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.editorFontSize') }}</label>
              <a-input-number v-model="form.editorFontSize" :min="10" :max="24" :step="1" style="width: 100px" />
            </div>
            <div class="form-row">
              <label class="form-label">
                <a-switch v-model="form.showLineNumbers" size="small" />
                {{ $t('settings.showLineNumbers') }}
              </label>
            </div>
            <div class="section-title">{{ $t('common.export') }} / {{ $t('common.import') }}</div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.defaultExportFormat') }}</label>
              <a-select v-model="form.defaultExportFormat" style="width: 120px">
                <a-option value="json">JSON</a-option>
                <a-option value="csv">CSV</a-option>
                <a-option value="redis">{{ $t('format.redisCmd') }}</a-option>
              </a-select>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.bufferLimit') }}</label>
              <a-input-number v-model="form.bufferLimit" :min="100" :max="50000" style="width: 120px" />
              <span class="form-hint">{{ $t('settings.bufferLimitHint') }}</span>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('settings.scanBatchSize') }}</label>
              <a-input-number v-model="form.scanBatchSize" :min="100" :max="10000" style="width: 120px" />
              <span class="form-hint">{{ $t('settings.scanBatchSizeHint') }}</span>
            </div>
            <div class="form-row">
              <label class="form-label">
                <a-switch v-model="form.dangerousCommandConfirm" size="small" />
                {{ $t('settings.dangerousCommandConfirm') }}
              </label>
            </div>
          </div>

          <!-- 快捷键 -->
          <div v-show="activeTab === 'shortcut'" class="settings-section">
            <div class="shortcut-list">
              <div v-for="action in shortcutActions" :key="action.id" class="shortcut-row">
                <span class="shortcut-label">{{ $t(action.labelKey) }}</span>
                <button
                  :class="['shortcut-binding', { recording: recordingId === action.id }]"
                  @click="startRecording(action.id)"
                  @keydown.prevent.stop="handleRecordKey($event, action.id)"
                  @blur="cancelRecording"
                >
                  {{ recordingId === action.id ? $t('shortcut.recording') : formatBinding(shortcutBindings[action.id] || action.defaultBinding) }}
                </button>
                <span v-if="conflictMsg[action.id]" class="shortcut-conflict">{{ conflictMsg[action.id] }}</span>
              </div>
            </div>
            <div class="shortcut-footer">
              <a-button size="small" @click="handleResetShortcuts">{{ $t('shortcut.resetAll') }}</a-button>
            </div>
          </div>

          <!-- 自定义解码器 -->
          <div v-show="activeTab === 'decoder'" class="settings-section">
            <div class="decoder-toolbar">
              <a-button type="primary" size="small" @click="startAddDecoder"><IconPlus :size="12" /> {{ $t('decoder.addDecoder') }}</a-button>
            </div>

            <!-- 添加/编辑表单 -->
            <div v-if="decoderForm.visible" class="decoder-form">
              <div class="form-row">
                <label class="form-label">{{ $t('decoder.name') }}</label>
                <a-input v-model="decoderForm.name" style="width:200px" :placeholder="$t('decoder.namePlaceholder')" />
              </div>
              <div class="form-row">
                <label class="form-label">{{ $t('decoder.command') }}</label>
                <a-input v-model="decoderForm.command" style="width:280px" :placeholder="$t('decoder.commandPlaceholder')" />
              </div>
              <div class="form-row">
                <span class="form-hint">{{ $t('decoder.commandHint') }}</span>
              </div>
              <div class="form-row" style="gap:var(--spacing-sm)">
                <a-button type="primary" size="small" @click="saveDecoder" :disabled="!decoderForm.name || !decoderForm.command">{{ $t('common.save') }}</a-button>
                <a-button size="small" @click="decoderForm.visible = false">{{ $t('common.cancel') }}</a-button>
              </div>
            </div>

            <!-- 解码器列表 -->
            <table v-if="decoderList.length" class="decoder-table">
              <thead>
                <tr>
                  <th>{{ $t('decoder.name') }}</th>
                  <th>{{ $t('bitfield.type') }}</th>
                  <th>{{ $t('decoder.command') }}</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="d in decoderList" :key="d.id">
                  <td>{{ d.name }}</td>
                  <td><span :class="['decoder-type-badge', d.type]">{{ d.type === 'builtin' ? $t('decoder.builtin') : $t('decoder.custom') }}</span></td>
                  <td class="mono-cell">{{ d.command || '—' }}</td>
                  <td class="action-cell">
                    <template v-if="d.type !== 'builtin'">
                      <a-tooltip :content="$t('common.edit')" mini><a-button type="text" size="mini" @click="startEditDecoder(d)"><template #icon><IconEdit :size="14" /></template></a-button></a-tooltip>
                      <a-tooltip :content="$t('common.delete')" mini><a-button type="text" size="mini" status="danger" @click="deleteDecoder(d.id)"><template #icon><IconDelete :size="14" /></template></a-button></a-tooltip>
                    </template>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="settings-footer">
        <a-button @click="handleReset">{{ $t('common.reset') }}</a-button>
        <div class="footer-spacer"></div>
        <a-button @click="handleClose">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="handleSave" :disabled="saving">{{ $t('common.save') }}</a-button>
      </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { request } from '../../utils/request'
import { loadLanguage } from '../../i18n'
import { gmConfirm } from '../../utils/dialog'
import { IconEdit, IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'
import {
  shortcutActions,
  getBindings,
  setBinding,
  resetShortcutBindings,
  eventToBinding,
  detectConflict,
  formatBinding,
} from '../../composables/useShortcuts'
import { useSettingsStore } from '../../stores/settings'
import { showMessage } from '@/utils/platform'

withDefaults(defineProps<{ visible?: boolean }>(), { visible: true })
const emit = defineEmits<{ close: []; saved: [] }>()
const { t } = useI18n()
const settingsStore = useSettingsStore()

const activeTab = ref('general')
const saving = ref(false)

const tabs = [
  { key: 'general', label: 'settings.general' },
  { key: 'connection', label: 'settings.connection' },
  { key: 'advanced', label: 'settings.advanced' },
  { key: 'shortcut', label: 'shortcut.title' },
  { key: 'decoder', label: 'decoder.title' },
]

const themeOptions = [
  { value: 'light', label: 'settings.themeLight' },
  { value: 'dark', label: 'settings.themeDark' },
]

const languages = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en', label: 'English' },
  { value: 'zh-TW', label: '繁體中文' },
  { value: 'ja', label: '日本語' },
  { value: 'ko', label: '한국어' },
  { value: 'ru', label: 'Русский' },
  { value: 'fr', label: 'Français' },
]


const defaults = {
  theme: 'dark',
  language: 'zh-CN',
  scanCount: 200,
  autoReconnect: true,
  retryInterval: 5,
  maxRetries: 10,
  heartbeatInterval: 30,
  editorFontSize: 14,
  showLineNumbers: true,
  cliFontSize: 14,
  defaultExportFormat: 'json',
  bufferLimit: 5000,
  scanBatchSize: 1000,
  dangerousCommandConfirm: true,
}

const form = reactive({ ...defaults })

onMounted(async () => {
  // 从 settingsStore 恢复当前值
  const s = settingsStore.settings
  Object.assign(form, {
    theme: s.theme, language: s.language,
    scanCount: s.scanCount, autoReconnect: s.autoReconnect,
    retryInterval: s.retryInterval, maxRetries: s.maxRetries,
    heartbeatInterval: s.heartbeatInterval, editorFontSize: s.editorFontSize,
    showLineNumbers: s.showLineNumbers, cliFontSize: s.cliFontSize,
    defaultExportFormat: s.defaultExportFormat, bufferLimit: s.bufferLimit,
    scanBatchSize: s.scanBatchSize, dangerousCommandConfirm: s.dangerousCommandConfirm,
  })

  // 加载后端完整数据
  try {
    const res = await request<Record<string, any>>('get_settings', { params: {} })
    if (res.data) {
      const { shortcuts, ...rest } = res.data
      Object.assign(form, { ...defaults, ...rest })
      if (shortcuts && typeof shortcuts === 'object') {
        for (const [id, binding] of Object.entries(shortcuts)) {
          if (typeof binding === 'string' && shortcutActions.some(a => a.id === id)) {
            shortcutBindings[id] = binding
          }
        }
      }
    }
  } catch (_e) {
    // 使用默认值
  }
  // 保存快捷键快照用于取消回滚
  Object.assign(savedBindings, getBindings())
})

const decoderLoaded = ref(false)
watch(activeTab, (tab) => {
  if (tab === 'decoder' && !decoderLoaded.value) {
    loadDecoders()
    decoderLoaded.value = true
  }
})

function handleReset() {
  Object.assign(form, defaults)
  handleResetShortcuts()
}

// ========== 解码器管理 ==========
interface DecoderConfig { id: string; name: string; type: string; command: string }
const decoderList = ref<DecoderConfig[]>([])
const decoderForm = reactive({ visible: false, id: '', name: '', command: '' })

async function loadDecoders() {
  try {
    const res = await request<DecoderConfig[]>('get_decoders', { params: {} })
    if (res.data) decoderList.value = res.data
  } catch (_e) { /* ignore */ }
}

function startAddDecoder() {
  decoderForm.visible = true
  decoderForm.id = ''
  decoderForm.name = ''
  decoderForm.command = ''
}

function startEditDecoder(d: DecoderConfig) {
  decoderForm.visible = true
  decoderForm.id = d.id
  decoderForm.name = d.name
  decoderForm.command = d.command
}

async function saveDecoder() {
  try {
    await request('save_decoder', {
      params: { id: decoderForm.id || undefined, name: decoderForm.name, type: 'command', command: decoderForm.command },
    })
    decoderForm.visible = false
    await loadDecoders()
    showMessage('success', t('common.success'))
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function deleteDecoder(id: string) {
  if (!await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('delete_decoder', { params: { id } })
    await loadDecoders()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

// ========== 快捷键管理 ==========
const shortcutBindings = reactive<Record<string, string>>({ ...getBindings() })
const savedBindings: Record<string, string> = {}
const recordingId = ref('')
const conflictMsg = reactive<Record<string, string>>({})

function startRecording(actionId: string) {
  recordingId.value = actionId
  delete conflictMsg[actionId]
}

function cancelRecording() {
  recordingId.value = ''
}

function handleRecordKey(e: KeyboardEvent, actionId: string) {
  if (recordingId.value !== actionId) return
  const binding = eventToBinding(e)
  if (!binding) return // modifier-only press
  if (e.key === 'Escape') {
    recordingId.value = ''
    return
  }
  const conflict = detectConflict(binding, actionId)
  if (conflict) {
    const conflictAction = shortcutActions.find(a => a.id === conflict)
    conflictMsg[actionId] = t('shortcut.conflict', { action: conflictAction ? t(conflictAction.labelKey) : conflict })
    return
  }
  delete conflictMsg[actionId]
  shortcutBindings[actionId] = binding
  recordingId.value = ''
}

function handleClose() {
  // 恢复快捷键为打开时的快照（取消回滚）
  for (const [id, binding] of Object.entries(savedBindings)) {
    setBinding(id, binding)
  }
  emit('close')
}

function handleResetShortcuts() {
  resetShortcutBindings()
  Object.assign(shortcutBindings, getBindings())
  for (const key of Object.keys(conflictMsg)) delete conflictMsg[key]
}

async function handleSave() {
  saving.value = true
  try {
    await request('save_settings', { params: { ...form, shortcuts: { ...shortcutBindings } } })
    // 批量写入全局快捷键绑定
    for (const [id, binding] of Object.entries(shortcutBindings)) {
      setBinding(id, binding)
    }
    // 更新 settingsStore（会自动 applyTheme + applyCssVars）
    settingsStore.updateFromSave({ ...form })
    // 应用语言
    await loadLanguage(form.language)
    emit('saved')
    emit('close')
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.settings-body {
  display: flex;
  overflow: hidden;
  height: 450px;
}

.settings-tabs {
  width: 140px;
  flex-shrink: 0;
  border-right: 1px solid var(--color-border-1);
  padding: var(--spacing-sm) 0;
}

.settings-tab {
  display: flex !important;
  justify-content: center !important;
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md) !important;
  background: none !important;
  border: none;
  border-left: 2px solid transparent;
  color: var(--color-text-3) !important;
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.settings-tab:hover {
  color: var(--color-text-1) !important;
  background: var(--color-fill-1) !important;
}

.settings-tab.active {
  color: var(--color-primary) !important;
  border-left-color: var(--color-primary);
  background: var(--color-primary-bg) !important;
}

.settings-content {
  flex: 1;
  padding: var(--spacing-lg);
  overflow-y: auto;
}

.settings-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
}

.section-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-3);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding-bottom: var(--spacing-xs);
  border-bottom: 1px solid var(--color-border-1);
  margin-top: var(--spacing-xs);
}

.section-title:first-child {
  margin-top: 0;
}

/* 覆盖全局 .form-row/.form-label 为横向布局 */
.form-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: 0;
}

.form-label {
  min-width: 120px;
  font-size: var(--font-size-sm);
  color: var(--color-text-2);
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.settings-footer {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-md) var(--spacing-lg);
  border-top: 1px solid var(--color-border-1);
}

.footer-spacer {
  flex: 1;
}

.decoder-toolbar {
  display: flex;
  justify-content: flex-end;
}

.decoder-form {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  background: var(--color-fill-1);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-1);
}

.decoder-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-sm);
}

.decoder-table th {
  text-align: left;
  padding: var(--spacing-xs) var(--spacing-sm);
  color: var(--color-text-3);
  font-weight: 500;
  font-size: var(--font-size-xs);
  border-bottom: 1px solid var(--color-border-1);
}

.decoder-table td {
  padding: var(--spacing-xs) var(--spacing-sm);
  color: var(--color-text-1);
  border-bottom: 1px solid var(--color-border-1);
}

.decoder-table .mono-cell {
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}

.decoder-table .action-cell {
  text-align: right;
  white-space: nowrap;
}

.decoder-type-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
}

.decoder-type-badge.builtin {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}

.decoder-type-badge.command {
  background: var(--color-fill-2);
  color: var(--color-text-2);
}

.shortcut-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.shortcut-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-xs) 0;
}

.shortcut-label {
  min-width: 120px;
  font-size: var(--font-size-sm);
  color: var(--color-text-2);
}

.shortcut-binding {
  min-width: 140px;
  height: 28px;
  padding: 0 var(--spacing-sm);
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-md);
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  cursor: pointer;
  text-align: center;
  transition: all var(--transition-fast);
}

.shortcut-binding:hover {
  border-color: var(--color-primary);
}

.shortcut-binding:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary-bg);
  outline: none;
}

.shortcut-binding.recording {
  border-color: var(--color-warning);
  background: var(--color-warning-bg, rgba(255, 165, 0, 0.08));
  color: var(--color-warning);
  font-family: var(--font-family);
}

.shortcut-conflict {
  font-size: var(--font-size-xs);
  color: var(--color-error);
}

.shortcut-footer {
  margin-top: var(--spacing-md);
  display: flex;
  justify-content: flex-end;
}
</style>
