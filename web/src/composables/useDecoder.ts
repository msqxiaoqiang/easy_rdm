import { ref } from 'vue'
import { request } from '../utils/request'
import { useI18n } from 'vue-i18n'

export interface DecoderItem { id: string; name: string; type: string }

// 全局缓存解码器列表，避免每个组件重复请求
let cachedDecoders: DecoderItem[] | null = null
let loadingPromise: Promise<void> | null = null

export function useDecoder() {
  const { t } = useI18n()
  const decoders = ref<DecoderItem[]>(cachedDecoders || [])
  const selectedDecoder = ref('')
  const decodedValue = ref<string | null>(null)

  async function loadDecoders() {
    if (cachedDecoders) {
      decoders.value = cachedDecoders
      return
    }
    if (loadingPromise) {
      await loadingPromise
      decoders.value = cachedDecoders || []
      return
    }
    loadingPromise = (async () => {
      try {
        const res = await request<DecoderItem[]>('get_decoders', { params: {} })
        if (res.data) cachedDecoders = res.data
      } catch (_e) { /* ignore */ }
    })()
    await loadingPromise
    loadingPromise = null
    decoders.value = cachedDecoders || []
  }

  async function applyDecoder(rawValue: string) {
    if (!selectedDecoder.value) {
      decodedValue.value = null
      return
    }
    try {
      const res = await request<string>('decode_value', {
        params: { decoder_id: selectedDecoder.value, value: rawValue },
      })
      decodedValue.value = res.data ?? ''
    } catch (e: any) {
      decodedValue.value = `[${t('decoder.decodeError')}] ${e.message || ''}`
    }
  }

  function resetDecoder() {
    selectedDecoder.value = ''
    decodedValue.value = null
  }

  loadDecoders()

  return {
    decoders,
    selectedDecoder,
    decodedValue,
    applyDecoder,
    resetDecoder,
  }
}
