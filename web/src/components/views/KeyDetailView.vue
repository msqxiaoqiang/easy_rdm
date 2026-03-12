<template>
  <div class="key-detail-view">
    <!-- 未选择 Key -->
    <div v-if="!keyName" class="empty-state">
      <p>{{ $t('key.selectKey') }}</p>
    </div>

    <!-- Loading -->
    <div v-else-if="detailLoading" class="loading-state">
      <span class="loading-spinner"></span>
    </div>

    <!-- Key 详情 -->
    <template v-else>
      <!-- 顶部信息栏 -->
      <div class="key-info-bar">
        <span :class="['type-badge', keyType]">{{ keyType }}</span>
        <span class="key-name-display" v-ellipsis-tip>{{ keyName }}</span>
        <span :class="['ttl-display', ttlClass]">TTL: {{ formatTTL(keyInfo.ttl) }}</span>
        <div class="info-actions">
          <a-tooltip :content="$t('common.refresh')" mini>
            <a-button size="mini" class="action-btn" @click="refreshValue">
              <template #icon><IconRefresh /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip :content="$t('common.copy')" mini>
            <a-button size="mini" class="action-btn" @click="copyValue">
              <template #icon><IconCopy /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip :content="$t('key.rename')" mini>
            <a-button size="mini" class="action-btn" @click="handleRename">
              <template #icon><IconEdit /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip :content="$t('key.setTTL')" mini>
            <a-button size="mini" class="action-btn" @click="handleSetTTL">
              <template #icon><IconClockCircle /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip :content="$t('common.delete')" mini>
            <a-button size="mini" status="danger" class="action-btn" @click="handleDelete">
              <template #icon><IconDelete /></template>
            </a-button>
          </a-tooltip>
        </div>
      </div>

      <!-- 值编辑区 -->
      <div class="key-value-area">
        <!-- String 类型 -->
        <template v-if="keyType === 'string'">
          <div v-if="valueTruncated" class="truncation-warning">{{ $t('key.valueTruncated') }}</div>
          <!-- String sub-type views -->
          <BitmapView v-if="viewAs === 'bitmap'" :conn-id="connId" :key-name="keyName!" />
          <HllView v-else-if="viewAs === 'hll'" :conn-id="connId" :key-name="keyName!" />
          <BitfieldView v-else-if="viewAs === 'bitfield'" :conn-id="connId" :key-name="keyName!" />
          <div v-else-if="selectedDecoder && decodedValue !== null" class="value-editor mono decoded-view">{{ decodedValue }}</div>
          <JsonCodeEditor
            v-else-if="viewAs === 'json'"
            :model-value="editValue"
            :readonly="valueTruncated"
            @update:model-value="editValue = $event; modified = true"
          />
          <a-textarea
            v-else
            v-model="editValue"
            class="value-editor"
            :class="{ 'mono': viewAs !== 'text' }"
            spellcheck="false"
            auto-size
            @input="modified = true"
          />
        </template>

        <!-- Hash -->
        <HashDetail v-else-if="keyType === 'hash'" :conn-id="connId" :key-name="keyName!" />
        <!-- List -->
        <ListDetail v-else-if="keyType === 'list'" :conn-id="connId" :key-name="keyName!" />
        <!-- Set -->
        <SetDetail v-else-if="keyType === 'set'" :conn-id="connId" :key-name="keyName!" />
        <!-- ZSet / Geo -->
        <template v-else-if="keyType === 'zset' || keyType === 'geo'">
          <div class="value-toolbar">
            <label class="view-toggle">
              <a-switch v-model="geoView" size="small" /> {{ $t('geo.geoView') }}
            </label>
          </div>
          <GeoDetail v-if="geoView || keyType === 'geo'" :conn-id="connId" :key-name="keyName!" />
          <ZSetDetail v-else :conn-id="connId" :key-name="keyName!" />
        </template>
        <!-- Stream -->
        <StreamDetail v-else-if="keyType === 'stream'" :conn-id="connId" :key-name="keyName!" />
        <!-- RedisJSON -->
        <JsonDetail v-else-if="keyType === 'ReJSON-RL'" :conn-id="connId" :key-name="keyName!" />

        <!-- 其他类型 -->
        <template v-else>
          <pre class="value-readonly">{{ JSON.stringify(rawValue, null, 2) }}</pre>
        </template>
      </div>

      <!-- 底部状态栏（Tiny RDM 风格） -->
      <div class="detail-footer">
        <div class="footer-left">
          <span class="footer-info">{{ lengthDisplay }}</span>
          <span v-if="keyInfo.memoryUsage > 0" class="footer-info footer-sep">{{ $t('key.memoryUsage', { size: formatBytes(keyInfo.memoryUsage) }) }}</span>
        </div>
        <div class="footer-right">
          <!-- String 类型：查看方式 + 解码方式 + 保存 -->
          <template v-if="keyType === 'string'">
            <a-select v-model="viewAs" class="footer-select" size="mini" :style="{ width: '110px' }">
              <a-option value="text">{{ $t('format.text') }}</a-option>
              <a-option value="json">{{ $t('format.json') }}</a-option>
              <a-option value="hex">{{ $t('format.hex') }}</a-option>
              <a-option value="binary">{{ $t('format.binary') }}</a-option>
              <a-option value="bitmap">{{ $t('format.bitmap') }}</a-option>
              <a-option value="bitfield">{{ $t('format.bitfield') }}</a-option>
              <a-option value="hll">{{ $t('format.hyperloglog') }}</a-option>
            </a-select>
            <a-select v-if="showDecoderSelect" v-model="selectedDecoder" class="footer-select" size="mini" :style="{ width: '110px' }" @change="applySelectedDecoder">
              <a-option value="">{{ $t('decoder.none') }}</a-option>
              <a-option v-for="d in decoders" :key="d.id" :value="d.id">{{ d.name }}</a-option>
            </a-select>
            <a-button v-if="showDecoderSelect && !selectedDecoder" type="primary" size="mini" @click="handleSave" :disabled="!modified || valueTruncated">
              {{ $t('common.save') }}
            </a-button>
          </template>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, nextTick, onMounted, onUnmounted, onBeforeUnmount } from 'vue'
import { request, BizError } from '../../utils/request'
import { useConnectionStore } from '../../stores/connection'
import { useI18n } from 'vue-i18n'
import { gmConfirm, gmPrompt } from '../../utils/dialog'
import { showMessage } from '@/utils/platform'
import HashDetail from './collections/HashDetail.vue'
import ListDetail from './collections/ListDetail.vue'
import SetDetail from './collections/SetDetail.vue'
import ZSetDetail from './collections/ZSetDetail.vue'
import StreamDetail from './collections/StreamDetail.vue'
import GeoDetail from './collections/GeoDetail.vue'
import BitmapView from './collections/BitmapView.vue'
import HllView from './collections/HllView.vue'
import BitfieldView from './collections/BitfieldView.vue'
import JsonDetail from './collections/JsonDetail.vue'
import JsonCodeEditor from '../common/JsonCodeEditor.vue'
import { IconRefresh, IconCopy, IconEdit, IconClockCircle, IconDelete } from '@arco-design/web-vue/es/icon'

const props = defineProps<{
  connId: string
  keyName?: string
  keyType?: string
}>()

const emit = defineEmits<{ deleted: [key: string]; notFound: [key: string]; renamed: [oldKey: string, newKey: string]; ttlChanged: [key: string, ttl: number] }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()

function saveDetailCache(keyName: string) {
  if (!keyName) return
  connectionStore.saveKeyDetailCache(props.connId, keyName, {
    keyInfo: { ...keyInfo.value },
    rawValue: rawValue.value,
    editValue: editValue.value,
    viewAs: viewAs.value,
    modified: modified.value,
    valueTruncated: valueTruncated.value,
    geoView: geoView.value,
  })
}

function restoreDetailCache(keyName: string): boolean {
  const cached = connectionStore.getKeyDetailCache(props.connId, keyName)
  if (!cached) return false
  keyInfo.value = { ...cached.keyInfo }
  rawValue.value = cached.rawValue
  editValue.value = cached.editValue
  viewAs.value = cached.viewAs
  modified.value = cached.modified
  valueTruncated.value = cached.valueTruncated
  geoView.value = cached.geoView ?? false
  return true
}

const rawValue = ref<any>(null)
const editValue = ref('')
const viewAs = ref('text')
const modified = ref(false)
const suppressViewAsWatch = ref(false)
const keyInfo = ref({ ttl: -1, encoding: '', length: 0, memoryUsage: 0 })
const detailLoading = ref(false)
const geoView = ref(false)
const valueTruncated = ref(false)
// Decoder
interface DecoderItem { id: string; name: string; type: string }
const decoders = ref<DecoderItem[]>([])
const selectedDecoder = ref('')
const decodedValue = ref<string | null>(null)

const showDecoderSelect = computed(() => {
  return viewAs.value === 'text' || viewAs.value === 'json' || viewAs.value === 'hex' || viewAs.value === 'binary'
})

// 根据类型显示不同的长度信息
const lengthDisplay = computed(() => {
  const len = keyInfo.value.length
  switch (props.keyType) {
    case 'string': return t('key.lengthChars', { count: len })
    case 'list': return t('key.lengthItems', { count: len })
    case 'hash': return t('key.lengthFields', { count: len })
    case 'set': case 'zset': case 'geo': return t('key.lengthMembers', { count: len })
    case 'stream': return t('key.lengthMessages', { count: len })
    default: return t('key.lengthItems', { count: len })
  }
})

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const val = bytes / Math.pow(1024, i)
  return (i === 0 ? val : val.toFixed(1)) + ' ' + units[i]
}

async function loadDecoders() {
  try {
    const res = await request<DecoderItem[]>('get_decoders', { params: {} })
    if (res.data) decoders.value = res.data
  } catch (_e) { /* ignore */ }
}
loadDecoders()

async function applySelectedDecoder() {
  if (!selectedDecoder.value) {
    decodedValue.value = null
    return
  }
  const val = typeof rawValue.value === 'string' ? rawValue.value : ''
  try {
    const res = await request<string>('decode_value', {
      params: { decoder_id: selectedDecoder.value, value: val },
    })
    decodedValue.value = res.data ?? ''
  } catch (e: any) {
    decodedValue.value = `[${t('decoder.decodeError')}] ${e.message || ''}`
  }
}

const ttlClass = computed(() => {
  const ttl = keyInfo.value.ttl
  if (ttl === -1) return 'permanent'
  if (ttl <= 60) return 'danger'
  if (ttl <= 3600) return 'warning'
  return 'safe'
})

watch(() => props.keyName, async (newKey, oldKey) => {
  // 保存旧 key 的状态到缓存
  if (oldKey) saveDetailCache(oldKey)

  selectedDecoder.value = ''
  decodedValue.value = null
  if (newKey) {
    // 过期检查：异步操作完成后 keyName 已变化则丢弃结果
    const isStale = () => props.keyName !== newKey

    // 优先从缓存恢复（立即展示），然后后台静默刷新
    const hasCached = restoreDetailCache(newKey)
    if (hasCached) {
      detailLoading.value = false
      // 后台静默刷新：如果 key 无未保存修改，更新为最新值
      if (!modified.value) {
        const exists = await loadKeyInfo()
        if (isStale()) return
        if (!exists) {
          showMessage('warning', t('key.keyNotFound'))
          emit('notFound', newKey)
          return
        }
        await loadKeyValue()
        if (isStale()) return
        saveDetailCache(newKey)
      }
      return
    }
    // 无缓存，走网络请求
    geoView.value = false
    detailLoading.value = true
    const exists = await loadKeyInfo()
    if (isStale()) return
    if (!exists) {
      detailLoading.value = false
      showMessage('warning', t('key.keyNotFound'))
      emit('notFound', newKey)
      return
    }
    await loadKeyValue()
    if (isStale()) return
    detailLoading.value = false
    // 首次加载后写入缓存
    saveDetailCache(newKey)
  } else {
    rawValue.value = null
    editValue.value = ''
    modified.value = false
  }
}, { immediate: true })

/** 加载 key 信息，返回 false 表示 key 不存在 */
async function loadKeyInfo(): Promise<boolean> {
  if (!props.keyName) return false
  try {
    const res = await request<any>('get_key_info', {
      params: { conn_id: props.connId, key: props.keyName },
    })
    if (res.data) {
      if (res.data.type === 'none') return false
      keyInfo.value = {
        ttl: res.data.ttl ?? -1,
        encoding: res.data.encoding ?? '',
        length: res.data.length ?? 0,
        memoryUsage: res.data.memory_usage ?? 0,
      }
    }
    return true
  } catch (_e) { return false }
}

async function loadKeyValue() {
  if (!props.keyName) return
  modified.value = false
  // 非 string 类型由专用组件分页加载，跳过全量拉取
  if (props.keyType && props.keyType !== 'string') {
    rawValue.value = null
    editValue.value = ''
    return
  }
  try {
    const res = await request<any>('get_key_value', {
      params: { conn_id: props.connId, key: props.keyName },
    })
    if (res.data) {
      rawValue.value = res.data.value
      valueTruncated.value = !!res.data.truncated
      if (res.data.type === 'string') {
        const val = res.data.value ?? ''
        suppressViewAsWatch.value = true
        // 自动识别 JSON 格式：仅对象/数组（以 { 或 [ 开头）才切换 JSON 视图
        const trimmed = val.trim()
        if ((trimmed.startsWith('{') || trimmed.startsWith('[')) && (() => { try { JSON.parse(trimmed); return true } catch { return false } })()) {
          viewAs.value = 'json'
          editValue.value = JSON.stringify(JSON.parse(trimmed), null, 2)
        } else {
          viewAs.value = 'text'
          editValue.value = val
        }
        // 等 watcher 执行完再重置 flag（Vue watcher 是异步调度的）
        await nextTick()
        suppressViewAsWatch.value = false
      }
    }
  } catch (_e) { /* ignore */ }
}

// 切换 viewAs 时重新格式化（保留用户编辑内容）
watch(viewAs, (mode, oldMode) => {
  if (suppressViewAsWatch.value) return
  if (typeof rawValue.value !== 'string') return
  if (mode === 'bitmap' || mode === 'hll' || mode === 'bitfield') return
  // 从子视图切回时，重新加载最新数据（子视图可能修改了底层数据）
  if (oldMode === 'bitmap' || oldMode === 'hll' || oldMode === 'bitfield') {
    loadKeyValue()
    return
  }

  // 先从当前格式还原为原始字符串
  let raw: string
  if (oldMode === 'hex') {
    try {
      const bytes = editValue.value.trim().split(/\s+/).map(h => parseInt(h, 16))
      raw = new TextDecoder().decode(new Uint8Array(bytes))
    } catch (_e) { raw = rawValue.value ?? '' }
  } else if (oldMode === 'json') {
    try { raw = JSON.stringify(JSON.parse(editValue.value)) } catch (_e) { raw = editValue.value }
  } else if (oldMode === 'binary') {
    try {
      const bytes = editValue.value.trim().split(/\s+/).map(b => parseInt(b, 2))
      raw = new TextDecoder().decode(new Uint8Array(bytes))
    } catch (_e) { raw = rawValue.value ?? '' }
  } else {
    raw = editValue.value
  }

  // 再格式化为目标格式
  if (mode === 'json') {
    try { editValue.value = JSON.stringify(JSON.parse(raw), null, 2) } catch (_e) { editValue.value = raw }
  } else if (mode === 'hex') {
    editValue.value = Array.from(new TextEncoder().encode(raw))
      .map(b => b.toString(16).padStart(2, '0'))
      .join(' ')
  } else if (mode === 'binary') {
    editValue.value = Array.from(new TextEncoder().encode(raw))
      .map(b => b.toString(2).padStart(8, '0'))
      .join(' ')
  } else {
    editValue.value = raw
  }
})

async function refreshValue() {
  const exists = await loadKeyInfo()
  if (!exists) {
    showMessage('warning', t('key.keyNotFound'))
    emit('notFound', props.keyName!)
    return
  }
  await loadKeyValue()
  if (props.keyName) saveDetailCache(props.keyName)
}

async function handleSave() {
  if (!props.keyName) return
  // 根据当前格式还原为原始字符串再保存
  let saveValue = editValue.value
  if (viewAs.value === 'hex') {
    try {
      const bytes = saveValue.trim().split(/\s+/).map(h => parseInt(h, 16))
      saveValue = new TextDecoder().decode(new Uint8Array(bytes))
    } catch (_e) { /* 解码失败则原样保存 */ }
  } else if (viewAs.value === 'json') {
    try { saveValue = JSON.stringify(JSON.parse(saveValue)) } catch (_e) { /* 原样保存 */ }
  } else if (viewAs.value === 'binary') {
    try {
      const bytes = saveValue.trim().split(/\s+/).map(b => parseInt(b, 2))
      saveValue = new TextDecoder().decode(new Uint8Array(bytes))
    } catch (_e) { /* 解码失败则原样保存 */ }
  }
  try {
    await request('set_key_value', {
      params: {
        conn_id: props.connId,
        key: props.keyName,
        value: saveValue,
        ttl: -1, // 不修改 TTL
      },
    })
    rawValue.value = saveValue
    modified.value = false
    if (props.keyName) saveDetailCache(props.keyName)
    showMessage('success', t('common.success'))
  } catch (e: any) {
    if (e instanceof BizError && e.code === 409) {
      if (e.message === 'key_deleted') {
        showMessage('error', t('key.keyDeleted'))
      } else if (e.message === 'type_changed') {
        showMessage('error', t('key.typeChanged', { type: e.data }))
      } else {
        showMessage('error', t('key.saveConflict'))
      }
    } else {
      showMessage('error', e.message || t('common.failed'))
    }
  }
}

async function handleDelete() {
  if (!props.keyName) return
  if (!await gmConfirm(t('key.deleteConfirm'))) return
  try {
    await request('delete_keys', {
      params: { conn_id: props.connId, keys: [props.keyName] },
    })
    emit('deleted', props.keyName)
  } catch (e: any) {
    showMessage('error', e.message || t('common.failed'))
  }
}

async function handleRename() {
  if (!props.keyName) return
  const newName = await gmPrompt(t('key.renamePrompt'), props.keyName)
  if (!newName || newName === props.keyName) return
  try {
    await request('rename_key', {
      params: { conn_id: props.connId, key: props.keyName, new_key: newName },
    })
    emit('renamed', props.keyName, newName)
  } catch (e: any) {
    showMessage('error', e.message || t('common.failed'))
  }
}

async function handleSetTTL() {
  if (!props.keyName) return
  const input = await gmPrompt(t('key.ttlPrompt'), String(keyInfo.value.ttl))
  if (input === null) return
  const ttl = parseInt(input, 10)
  if (isNaN(ttl) || ttl < -1 || ttl === 0) return
  try {
    await request('set_ttl', {
      params: { conn_id: props.connId, key: props.keyName, ttl },
    })
    await loadKeyInfo()
    emit('ttlChanged', props.keyName, keyInfo.value.ttl)
  } catch (e: any) {
    showMessage('error', e.message || t('common.failed'))
  }
}

async function copyValue() {
  try {
    let text: string
    if (typeof rawValue.value === 'string' && showDecoderSelect.value) {
      text = editValue.value
    } else {
      text = typeof rawValue.value === 'string' ? rawValue.value : JSON.stringify(rawValue.value)
    }
    await navigator.clipboard.writeText(text)
    showMessage('success', t('common.success'))
  } catch (_e) { /* ignore */ }
}

function formatTTL(ttl: number): string {
  if (ttl === -1) return '∞'
  if (ttl === -2) return '-'
  if (ttl < 60) return ttl + 's'
  if (ttl < 3600) return Math.floor(ttl / 60) + 'm ' + (ttl % 60) + 's'
  if (ttl < 86400) return Math.floor(ttl / 3600) + 'h ' + Math.floor((ttl % 3600) / 60) + 'm'
  return Math.floor(ttl / 86400) + 'd ' + Math.floor((ttl % 86400) / 3600) + 'h'
}

// 监听全局快捷键 Ctrl+S → 保存
function onShortcutSave() {
  if (props.keyType === 'string' && modified.value && !selectedDecoder.value) {
    handleSave()
  }
}
onMounted(() => document.addEventListener('shortcut:save', onShortcutSave))
onUnmounted(() => document.removeEventListener('shortcut:save', onShortcutSave))

// 卸载前保存当前 key 的缓存
onBeforeUnmount(() => {
  if (props.keyName) saveDetailCache(props.keyName)
})
</script>

<style scoped>
.key-detail-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-3);
  font-size: var(--font-size-md);
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.loading-spinner {
  width: 28px;
  height: 28px;
  border: 2.5px solid var(--color-border-2);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.key-info-bar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  min-width: 0;
}

.type-badge {
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: #fff;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.type-badge.string { background: var(--color-type-string); }
.type-badge.list { background: var(--color-type-list); }
.type-badge.set { background: var(--color-type-set); }
.type-badge.zset { background: var(--color-type-zset); }
.type-badge.hash { background: var(--color-type-hash); }
.type-badge.stream { background: var(--color-type-stream); }
.type-badge.geo { background: var(--color-type-geo); }
.type-badge.ReJSON-RL { background: var(--color-type-rejson); }

.key-name-display {
  flex: 1;
  font-family: var(--font-family-mono);
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ttl-display {
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  flex-shrink: 0;
}

.ttl-display.permanent { color: var(--color-ttl-permanent); }
.ttl-display.danger { color: var(--color-ttl-danger); }
.ttl-display.warning { color: var(--color-ttl-warning); }
.ttl-display.safe { color: var(--color-ttl-safe); }

.info-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.action-btn {
  color: var(--color-text-3);
}
.action-btn:hover {
  color: var(--color-text-1);
}

.key-value-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.value-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
}

.value-editor {
  flex: 1;
  width: 100%;
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-bg-1);
  border: none;
  color: var(--color-text-1);
  font-size: var(--app-editor-font-size, var(--font-size-sm));
  line-height: 1.6;
  resize: none;
  outline: none;
}

/* 去掉 Arco textarea 的 hover/focus 灰色遮罩 */
.key-value-area :deep(.arco-textarea-wrapper) {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
}

/* 让 Arco textarea 内部元素继承编辑器字体大小设置 */
.key-value-area :deep(.arco-textarea) {
  font-size: var(--app-editor-font-size, var(--font-size-sm));
}

.value-editor.mono {
  font-family: var(--font-family-mono);
}

.decoded-view {
  white-space: pre-wrap;
  word-break: break-all;
  overflow: auto;
  cursor: default;
  user-select: text;
}

.view-toggle {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  user-select: none;
}

.value-readonly {
  flex: 1;
  padding: var(--spacing-sm) var(--spacing-md);
  margin: 0;
  overflow: auto;
  background: var(--color-bg-1);
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-sm);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

.truncation-warning {
  padding: 6px var(--spacing-md);
  background: var(--color-warning-bg, #fff7e6);
  color: var(--color-warning, #d46b08);
  font-size: var(--font-size-xs);
  border-bottom: 1px solid var(--color-border-1);
}

/* ========== 底部状态栏（Tiny RDM 风格） ========== */
.detail-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px var(--spacing-md);
  border-top: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  flex-shrink: 0;
  min-height: 32px;
  gap: var(--spacing-md);
}

.footer-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  min-width: 0;
}

.footer-info {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  font-family: var(--font-family-mono);
  white-space: nowrap;
}

.footer-sep::before {
  content: '|';
  margin-right: var(--spacing-sm);
  color: var(--color-border-2);
}

.footer-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  flex-shrink: 0;
}

.footer-select {
  width: 110px;
}

.footer-select :deep(.arco-select-view-single) {
  height: 24px;
  line-height: 24px;
  font-size: var(--font-size-xs);
}

</style>
