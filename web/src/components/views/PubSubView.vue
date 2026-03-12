<template>
  <div class="pubsub-view">
    <!-- 订阅控制 -->
    <div class="pubsub-toolbar">
      <a-input
        v-model="channelInput"
        size="small"
        :placeholder="$t('pubsub.channelPlaceholder')"
        :disabled="subscribed"
        @keydown.enter="startSubscribe"
        style="flex:1;min-width:120px"
      />
      <a-button v-if="!subscribed" type="primary" size="small" :disabled="!channelInput.trim()" @click="startSubscribe">
        {{ $t('pubsub.subscribe') }}
      </a-button>
      <a-button v-else type="primary" status="danger" size="small" @click="stopSubscribe">
        {{ $t('pubsub.unsubscribe') }}
      </a-button>
      <div class="toolbar-sep"></div>
      <a-switch v-model="usePattern" size="small" :disabled="subscribed" />
      <span class="toggle-label">{{ $t('pubsub.patternMode') }}</span>
      <div class="toolbar-sep"></div>
      <span class="msg-count">{{ messages.length }} {{ $t('pubsub.messages') }}</span>
      <a-button size="small" :disabled="messages.length === 0" @click="clearMessages">{{ $t('pubsub.clear') }}</a-button>
    </div>

    <!-- 消息列表 -->
    <div class="pubsub-messages" ref="msgContainer">
      <div v-if="messages.length === 0" class="pubsub-empty">
        {{ subscribed ? $t('pubsub.waiting') : $t('pubsub.hint') }}
      </div>
      <div v-for="(msg, i) in messages" :key="i" class="msg-item">
        <span class="msg-time">{{ formatTime(msg.ts) }}</span>
        <span class="msg-channel">{{ msg.channel }}</span>
        <span class="msg-payload">{{ msg.payload }}</span>
      </div>
    </div>

    <!-- 发布区域 -->
    <div class="pubsub-publish">
      <a-input
        v-model="pubChannel"
        size="small"
        class="pub-channel"
        :placeholder="$t('pubsub.pubChannel')"
        @keydown.enter="pubMsgInput?.focus()"
      />
      <a-input
        ref="pubMsgInput"
        v-model="pubMessage"
        size="small"
        class="pub-message"
        :placeholder="$t('pubsub.pubMessage')"
        @keydown.enter="doPublish"
      />
      <a-button type="primary" size="small" :disabled="!pubChannel.trim() || !pubMessage.trim()" @click="doPublish">
        {{ $t('pubsub.publish') }}
      </a-button>
    </div>
  </div>
</template>

<script lang="ts">
import { Poller } from '../../utils/poller'
// 模块级：活跃的 Poller 实例，组件卸载后继续运行
const activePollers = new Map<string, Poller>()
</script>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { request } from '../../utils/request'
import { type PollEvent } from '../../utils/poller'
import { useI18n } from 'vue-i18n'
import { useConnectionStore } from '../../stores/connection'
import { showMessage } from '@/utils/platform'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()

interface Message {
  ts: number
  channel: string
  pattern: string
  payload: string
}

const channelInput = ref('*')
const usePattern = ref(true)
const subscribed = ref(false)
const messages = ref<Message[]>([])
const msgContainer = ref<HTMLElement>()
const pubChannel = ref('')
const pubMessage = ref('')
const pubMsgInput = ref<any>()
const MAX_MESSAGES = 5000

function handlePollData(events: PollEvent[]) {
  for (const evt of events) {
    const d = evt.data
    messages.value.push({
      ts: evt.time || Date.now(),
      channel: d.channel || '',
      pattern: d.pattern || '',
      payload: d.payload || '',
    })
  }
  if (messages.value.length > MAX_MESSAGES) {
    messages.value = messages.value.slice(-3000)
  }
  nextTick(() => {
    if (msgContainer.value) {
      msgContainer.value.scrollTop = msgContainer.value.scrollHeight
    }
  })
}

// 后台回调工厂：显式绑定 connId，避免闭包捕获响应式 props
function makeBackgroundCallback(connId: string) {
  return (events: PollEvent[]) => {
    const cache = connectionStore.getPubsubCache(connId)
    if (!cache) return
    for (const evt of events) {
      const d = evt.data
      cache.messages.push({
        ts: evt.time || Date.now(),
        channel: d.channel || '',
        pattern: d.pattern || '',
        payload: d.payload || '',
      })
    }
    if (cache.messages.length > MAX_MESSAGES) {
      cache.messages = cache.messages.slice(-3000)
    }
  }
}

async function startSubscribe() {
  const input = channelInput.value.trim()
  if (!input) return

  const channels = usePattern.value ? [] : input.split(',').map(s => s.trim()).filter(Boolean)
  const patterns = usePattern.value ? input.split(',').map(s => s.trim()).filter(Boolean) : []

  try {
    await request('pubsub_start', {
      params: {
        conn_id: props.connId,
        channels,
        patterns,
      },
    })
    subscribed.value = true
    messages.value = []

    // 停止旧 poller
    activePollers.get(props.connId)?.stop()

    const poller = new Poller({
      connId: props.connId,
      scene: 'pubsub',
      onData: handlePollData,
    })
    activePollers.set(props.connId, poller)
    poller.start()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function stopSubscribe() {
  const poller = activePollers.get(props.connId)
  poller?.stop()
  activePollers.delete(props.connId)
  try {
    await request('pubsub_stop', { params: { conn_id: props.connId } })
  } catch (_e) { /* ignore */ }
  subscribed.value = false
}

function clearMessages() {
  messages.value = []
}

async function doPublish() {
  if (!pubChannel.value.trim() || !pubMessage.value.trim()) return
  try {
    const res = await request<{ receivers: number }>('pubsub_publish', {
      params: {
        conn_id: props.connId,
        channel: pubChannel.value,
        message: pubMessage.value,
      },
    })
    const receivers = res.data?.receivers ?? 0
    showMessage('success', t('pubsub.published', { count: receivers }))
    pubMessage.value = ''
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function formatTime(ts: number): string {
  const d = new Date(ts)
  return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
    + '.' + String(d.getMilliseconds()).padStart(3, '0')
}

function saveToCache() {
  connectionStore.savePubsubCache(props.connId, {
    messages: messages.value,
    subscribed: subscribed.value,
    channelInput: channelInput.value,
    usePattern: usePattern.value,
  })
}

watch(() => props.connId, (_newId, oldId) => {
  // 切换连接时：保存旧连接状态，切换 poller 回调为后台模式
  if (oldId) {
    const oldPoller = activePollers.get(oldId)
    if (oldPoller?.running) {
      connectionStore.savePubsubCache(oldId, {
        messages: messages.value,
        subscribed: subscribed.value,
        channelInput: channelInput.value,
        usePattern: usePattern.value,
      })
      oldPoller.setCallbacks(makeBackgroundCallback(oldId))
    }
  }
  // 恢复新连接状态
  const cache = connectionStore.getPubsubCache(props.connId)
  if (cache) {
    messages.value = [...cache.messages]
    subscribed.value = cache.subscribed
    channelInput.value = cache.channelInput
    usePattern.value = cache.usePattern
  } else {
    messages.value = []
    subscribed.value = false
    channelInput.value = '*'
    usePattern.value = true
  }
  // 如果新连接有活跃 poller，切换回调为前台模式
  const newPoller = activePollers.get(props.connId)
  if (newPoller?.running) {
    newPoller.setCallbacks(handlePollData)
  }
})

// 监听断开连接：停止 poller
watch(() => connectionStore.getConnState(props.connId).status, (status) => {
  if (status === 'disconnected') {
    const poller = activePollers.get(props.connId)
    if (poller) {
      poller.stop()
      activePollers.delete(props.connId)
    }
    subscribed.value = false
    messages.value = []
  }
})

onMounted(() => {
  const cache = connectionStore.getPubsubCache(props.connId)
  if (cache) {
    messages.value = [...cache.messages]
    subscribed.value = cache.subscribed
    channelInput.value = cache.channelInput
    usePattern.value = cache.usePattern
  }
  // 恢复前台回调
  const poller = activePollers.get(props.connId)
  if (poller?.running) {
    poller.setCallbacks(handlePollData)
  }
})

onBeforeUnmount(() => {
  saveToCache()
  // 切换 poller 为后台模式（不停止）
  const poller = activePollers.get(props.connId)
  if (poller?.running) {
    poller.setCallbacks(makeBackgroundCallback(props.connId))
  }
})
</script>

<style scoped>
.pubsub-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.pubsub-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  flex-shrink: 0;
}

.toolbar-sep {
  width: 1px;
  height: 16px;
  background: var(--color-border-2);
  flex-shrink: 0;
}

.toggle-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  white-space: nowrap;
}

.msg-count {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  white-space: nowrap;
}

.pubsub-messages {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-sm);
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
}

.pubsub-empty {
  color: var(--color-text-4);
  text-align: center;
  padding: var(--spacing-xl);
}

.msg-item {
  display: flex;
  gap: var(--spacing-sm);
  padding: 3px 0;
  border-bottom: 1px solid var(--color-border-1);
  line-height: 1.5;
}

.msg-time {
  color: var(--color-text-4);
  flex-shrink: 0;
}

.msg-channel {
  color: var(--color-primary);
  flex-shrink: 0;
  font-weight: 500;
}

.msg-payload {
  color: var(--color-text-1);
  word-break: break-all;
}

.pubsub-publish {
  display: flex;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-top: 1px solid var(--color-border-1);
  background: var(--color-bg-2);
  flex-shrink: 0;
}

.pub-channel { width: 160px; flex-shrink: 0; }
.pub-message { flex: 1; }
</style>
