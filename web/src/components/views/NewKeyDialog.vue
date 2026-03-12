<template>
  <a-modal
    :visible="visible"
    :title="$t('newKey.title')"
    :width="520"
    unmount-on-close
    @cancel="close"
  >
    <div class="new-key-form">
      <label class="dialog-label">{{ $t('newKey.keyName') }}</label>
      <a-input
        v-model="keyName"
        :placeholder="$t('newKey.keyNamePlaceholder')"
        @blur="checkExists"
      />
      <span v-if="existsWarning" class="exists-warning">{{ $t('newKey.keyExists') }}</span>

      <label class="dialog-label">{{ $t('newKey.keyType') }}</label>
      <a-select v-model="keyType" class="type-select">
        <template #label="{ data }">
          <span class="type-option-inner">
            <span class="type-dot" :style="{ background: typeColors[data?.value] }"></span>
            <span>{{ data?.label }}</span>
          </span>
        </template>
        <a-option
          v-for="opt in allTypeOptions"
          :key="opt.value"
          :value="opt.value"
          :disabled="isTypeDisabled(opt)"
        >
          <span class="type-option-inner">
            <span class="type-dot" :style="{ background: typeColors[opt.value] }"></span>
            <span>{{ opt.label }}</span>
            <span v-if="isTypeDisabled(opt)" class="type-version-hint">Redis {{ opt.minVersion }}+</span>
          </span>
        </a-option>
      </a-select>

      <template v-if="keyType === 'string'">
        <label class="dialog-label">{{ $t('collection.value') }}</label>
        <a-textarea v-model="stringValue" :auto-size="{ minRows: 3, maxRows: 8 }" :placeholder="$t('newKey.valuePlaceholder')" />
      </template>

      <template v-else-if="keyType === 'hash'">
        <label class="dialog-label">{{ $t('collection.field') }} / {{ $t('collection.value') }}</label>
        <div ref="dynamicRowsRef" class="dynamic-rows">
          <div v-for="(row, idx) in hashFields" :key="idx" class="dynamic-row">
            <a-input v-model="row.field" class="flex-1" :placeholder="$t('collection.field')" />
            <a-input v-model="row.value" class="flex-1" :placeholder="$t('collection.value')" />
            <a-button size="small" :disabled="hashFields.length <= 1" @click="removeRow(hashFields, idx)">✕</a-button>
          </div>
        </div>
        <a-button size="small" type="dashed" long @click="addRow(hashFields, { field: '', value: '' })">+ {{ $t('newKey.addRow') }}</a-button>
      </template>

      <template v-else-if="keyType === 'list'">
        <label class="dialog-label">{{ $t('collection.value') }}</label>
        <div ref="dynamicRowsRef" class="dynamic-rows">
          <div v-for="(_, idx) in listValues" :key="idx" class="dynamic-row">
            <a-input v-model="listValues[idx]" class="flex-1" :placeholder="$t('collection.value')" />
            <a-button size="small" :disabled="listValues.length <= 1" @click="removeRow(listValues, idx)">✕</a-button>
          </div>
        </div>
        <a-button size="small" type="dashed" long @click="addRow(listValues, '')">+ {{ $t('newKey.addRow') }}</a-button>
      </template>

      <template v-else-if="keyType === 'set'">
        <label class="dialog-label">{{ $t('collection.member') }}</label>
        <div ref="dynamicRowsRef" class="dynamic-rows">
          <div v-for="(_, idx) in setMembers" :key="idx" class="dynamic-row">
            <a-input v-model="setMembers[idx]" class="flex-1" :placeholder="$t('collection.member')" />
            <a-button size="small" :disabled="setMembers.length <= 1" @click="removeRow(setMembers, idx)">✕</a-button>
          </div>
        </div>
        <a-button size="small" type="dashed" long @click="addRow(setMembers, '')">+ {{ $t('newKey.addRow') }}</a-button>
      </template>

      <template v-else-if="keyType === 'zset'">
        <label class="dialog-label">{{ $t('collection.member') }} / {{ $t('collection.score') }}</label>
        <div ref="dynamicRowsRef" class="dynamic-rows">
          <div v-for="(row, idx) in zsetMembers" :key="idx" class="dynamic-row">
            <a-input v-model="row.member" class="flex-1" :placeholder="$t('collection.member')" />
            <a-input-number v-model="row.score" class="score-input" :placeholder="$t('collection.score')" />
            <a-button size="small" :disabled="zsetMembers.length <= 1" @click="removeRow(zsetMembers, idx)">✕</a-button>
          </div>
        </div>
        <a-button size="small" type="dashed" long @click="addRow(zsetMembers, { member: '', score: 0 })">+ {{ $t('newKey.addRow') }}</a-button>
      </template>

      <template v-else-if="keyType === 'stream'">
        <label class="dialog-label">ID</label>
        <a-input v-model="streamId" placeholder="*" />
        <span class="stream-id-hint">{{ $t('newKey.streamIdHint') }}</span>
        <label class="dialog-label">{{ $t('stream.fields') }}</label>
        <div ref="dynamicRowsRef" class="dynamic-rows">
          <div v-for="(row, idx) in streamFields" :key="idx" class="dynamic-row">
            <a-input v-model="row.field" class="flex-1" :placeholder="$t('collection.field')" />
            <a-input v-model="row.value" class="flex-1" :placeholder="$t('collection.value')" />
            <a-button size="small" :disabled="streamFields.length <= 1" @click="removeRow(streamFields, idx)">✕</a-button>
          </div>
        </div>
        <a-button size="small" type="dashed" long @click="addRow(streamFields, { field: '', value: '' })">+ {{ $t('newKey.addRow') }}</a-button>
      </template>

      <template v-else-if="keyType === 'bitmap'">
        <label class="dialog-label">{{ $t('bitmap.offset') }}</label>
        <a-input-number v-model="bitmapOffset" :min="0" placeholder="0" />
      </template>

      <template v-else-if="keyType === 'hll'">
        <label class="dialog-label">{{ $t('hll.addElements') }}</label>
        <div ref="dynamicRowsRef" class="dynamic-rows">
          <div v-for="(_, idx) in hllElements" :key="idx" class="dynamic-row">
            <a-input v-model="hllElements[idx]" class="flex-1" placeholder="element" />
            <a-button size="small" :disabled="hllElements.length <= 1" @click="removeRow(hllElements, idx)">✕</a-button>
          </div>
        </div>
        <a-button size="small" type="dashed" long @click="addRow(hllElements, '')">+ {{ $t('newKey.addRow') }}</a-button>
      </template>

      <template v-else-if="keyType === 'geo'">
        <label class="dialog-label">{{ $t('geo.addMember') }}</label>
        <div ref="dynamicRowsRef" class="dynamic-rows">
          <div v-for="(row, idx) in geoMembers" :key="idx" class="dynamic-row">
            <a-input-number v-model="row.longitude" class="geo-coord" :step="0.000001" :placeholder="$t('geo.longitude')" />
            <a-input-number v-model="row.latitude" class="geo-coord" :step="0.000001" :placeholder="$t('geo.latitude')" />
            <a-input v-model="row.member" class="flex-1" :placeholder="$t('collection.member')" />
            <a-button size="small" :disabled="geoMembers.length <= 1" @click="removeRow(geoMembers, idx)">✕</a-button>
          </div>
        </div>
        <a-button size="small" type="dashed" long @click="addRow(geoMembers, { longitude: 0, latitude: 0, member: '' })">+ {{ $t('newKey.addRow') }}</a-button>
      </template>

      <label class="dialog-label">TTL</label>
      <div class="ttl-row">
        <a-input-number v-model="ttlValue" class="ttl-input" :min="-1" placeholder="-1" />
        <span class="ttl-hint">{{ $t('newKey.ttlHint') }}</span>
      </div>
    </div>

    <template #footer>
      <a-button @click="close">{{ $t('common.cancel') }}</a-button>
      <a-button type="primary" @click="create" :disabled="!keyName.trim() || existsWarning">{{ $t('common.confirm') }}</a-button>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { request } from '../../utils/request'
import { showMessage } from '@/utils/platform'
import { useI18n } from 'vue-i18n'
import { useConnectionStore } from '../../stores/connection'
import { versionGte } from '../../utils/version'

const props = defineProps<{ connId: string; visible: boolean; defaultKeyName?: string }>()
const emit = defineEmits<{ close: []; created: [key: string, type: string] }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const redisVersion = computed(() => connectionStore.getConnState(props.connId).redisVersion)

const keyName = ref('')
const keyType = ref('string')
const stringValue = ref('')
const ttlValue = ref(-1)
const existsWarning = ref(false)
let checkTimer: ReturnType<typeof setTimeout> | null = null
const dynamicRowsRef = ref<HTMLElement | null>(null)

const hashFields = ref([{ field: '', value: '' }])
const listValues = ref([''])
const setMembers = ref([''])
const zsetMembers = ref([{ member: '', score: 0 }])
const streamId = ref('')
const streamFields = ref([{ field: '', value: '' }])
const bitmapOffset = ref(0)
const hllElements = ref([''])
const geoMembers = ref([{ longitude: 0, latitude: 0, member: '' }])

const typeColors: Record<string, string> = {
  string: '#4080ff',
  hash: '#722ed1',
  list: '#00b42a',
  set: '#ff7d00',
  zset: '#f53f3f',
  stream: '#0fc6c2',
  bitmap: '#eb2f96',
  hll: '#9254de',
  geo: '#13c2c2',
}

const allTypeOptions = [
  { value: 'string', label: 'String', minVersion: '' },
  { value: 'hash', label: 'Hash', minVersion: '' },
  { value: 'list', label: 'List', minVersion: '' },
  { value: 'set', label: 'Set', minVersion: '' },
  { value: 'zset', label: 'ZSet', minVersion: '' },
  { value: 'stream', label: 'Stream', minVersion: '5.0.0' },
  { value: 'bitmap', label: 'Bitmap', minVersion: '' },
  { value: 'hll', label: 'HyperLogLog', minVersion: '' },
  { value: 'geo', label: 'Geo', minVersion: '3.2.0' },
]

function isTypeDisabled(opt: typeof allTypeOptions[0]): boolean {
  if (!opt.minVersion) return false
  return !versionGte(redisVersion.value, opt.minVersion)
}

watch(() => props.visible, (v) => {
  if (v) {
    keyName.value = props.defaultKeyName || ''
    keyType.value = 'string'
    stringValue.value = ''
    ttlValue.value = -1
    existsWarning.value = false
    hashFields.value = [{ field: '', value: '' }]
    listValues.value = ['']
    setMembers.value = ['']
    zsetMembers.value = [{ member: '', score: 0 }]
    streamId.value = ''
    streamFields.value = [{ field: '', value: '' }]
    bitmapOffset.value = 0
    hllElements.value = ['']
    geoMembers.value = [{ longitude: 0, latitude: 0, member: '' }]
  }
})

function addRow(arr: any[], template: any) {
  arr.push(typeof template === 'object' ? { ...template } : template)
  nextTick(() => {
    dynamicRowsRef.value?.scrollTo({ top: dynamicRowsRef.value.scrollHeight, behavior: 'smooth' })
  })
}

function removeRow(arr: any[], idx: number) {
  if (arr.length > 1) arr.splice(idx, 1)
}

async function checkExists() {
  if (!keyName.value.trim()) { existsWarning.value = false; return }
  if (checkTimer) clearTimeout(checkTimer)
  checkTimer = setTimeout(async () => {
    try {
      const res = await request<any>('check_key_exists', {
        params: { conn_id: props.connId, key: keyName.value },
      })
      existsWarning.value = res.data?.exists === true
    } catch (_e) { existsWarning.value = false }
  }, 200)
}

async function create() {
  if (!keyName.value.trim()) return
  const ttl = ttlValue.value ?? -1
  const params: any = {
    conn_id: props.connId,
    key: keyName.value,
    type: keyType.value,
    ttl,
  }
  switch (keyType.value) {
    case 'string':
      params.value = stringValue.value
      break
    case 'hash':
      params.hash_fields = hashFields.value.filter(f => f.field.trim())
      break
    case 'list':
      params.list_values = listValues.value.filter(v => v.trim())
      break
    case 'set':
      params.set_members = setMembers.value.filter(m => m.trim())
      break
    case 'zset':
      params.zset_members = zsetMembers.value.filter(m => m.member.trim())
      break
    case 'stream':
      params.stream_id = streamId.value
      params.stream_fields = streamFields.value.filter(f => f.field.trim())
      break
    case 'bitmap':
      params.bitmap_offset = bitmapOffset.value || 0
      break
    case 'hll':
      params.hll_elements = hllElements.value.filter(e => e.trim())
      break
    case 'geo':
      params.geo_members = geoMembers.value.filter(m => m.member.trim())
      break
  }
  try {
    await request('create_key', { params })
    showMessage('success', t('common.success'))
    emit('created', keyName.value, keyType.value)
    emit('close')
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function close() { emit('close') }
</script>

<style scoped>
.new-key-form :deep(.arco-input-wrapper),
.new-key-form :deep(.arco-textarea-wrapper),
.new-key-form :deep(.arco-input-number) {
  margin-bottom: var(--spacing-xs);
}

.type-select {
  margin-bottom: var(--spacing-sm);
}

.type-option-inner {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.type-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  flex-shrink: 0;
}

.type-version-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-4);
  margin-left: auto;
}

.ttl-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}

.ttl-input {
  width: 120px;
  flex: none;
}

.ttl-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}

.exists-warning {
  display: block;
  margin-top: 2px;
  font-size: var(--font-size-xs);
  color: var(--color-error);
}

.dynamic-rows {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 160px;
  overflow-y: auto;
  margin-bottom: 6px;
}

.dynamic-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.dynamic-row > .arco-btn:last-child {
  width: 32px;
  flex-shrink: 0;
}

.flex-1 {
  flex: 1;
  min-width: 0;
}

.score-input {
  width: 90px;
  flex: none;
}

.geo-coord {
  width: 100px;
  flex: none;
}

.stream-id-hint {
  display: block;
  margin-top: 2px;
  margin-bottom: var(--spacing-xs);
  font-size: var(--font-size-xs);
  color: var(--color-text-4);
}
</style>
