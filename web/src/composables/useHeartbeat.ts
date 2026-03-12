import { ref, onUnmounted, watch } from 'vue'
import { useConnectionStore } from '../stores/connection'
import { request } from '../utils/request'
import { toast } from '../utils/toast'

/**
 * 心跳 + 自动重连 composable
 * 在 AppLayout 中调用一次即可
 */
export function useHeartbeat() {
  const store = useConnectionStore()
  const timers = ref<Record<string, ReturnType<typeof setInterval>>>({})
  const retryCount = ref<Record<string, number>>({})

  const HEARTBEAT_INTERVAL = 30_000 // 30s
  const MAX_RETRIES = 5
  const BASE_RETRY_DELAY = 2_000 // 2s, exponential backoff

  // 哨兵故障转移轮询
  const sentinelTimers = ref<Record<string, ReturnType<typeof setInterval>>>({})
  const sentinelSeq = ref<Record<string, number>>({})
  const SENTINEL_POLL_INTERVAL = 5_000 // 5s

  function startHeartbeat(connId: string) {
    stopHeartbeat(connId)
    retryCount.value[connId] = 0

    timers.value[connId] = setInterval(async () => {
      const state = store.getConnState(connId)
      if (state.status !== 'connected') {
        stopHeartbeat(connId)
        return
      }

      try {
        const res = await request<string>('ping', { params: { conn_id: connId } })
        if (res.code !== 200) throw new Error(res.msg)
        retryCount.value[connId] = 0
      } catch (_e) {
        // Ping failed — attempt reconnect
        handleDisconnect(connId)
      }
    }, HEARTBEAT_INTERVAL)

    // 哨兵模式：启动故障转移轮询
    const conn = store.connections.find(c => c.id === connId)
    if (conn?.use_sentinel) {
      startSentinelPoll(connId)
    }
  }

  function stopHeartbeat(connId: string) {
    if (timers.value[connId]) {
      clearInterval(timers.value[connId])
      delete timers.value[connId]
    }
    stopSentinelPoll(connId)
  }

  function startSentinelPoll(connId: string) {
    stopSentinelPoll(connId)
    sentinelSeq.value[connId] = 0

    sentinelTimers.value[connId] = setInterval(async () => {
      try {
        const res = await request<any[]>('poll', {
          params: { conn_id: connId, scene: 'sentinel', after: sentinelSeq.value[connId] || 0 },
        })
        const events = res.data || []
        for (const evt of events) {
          sentinelSeq.value[connId] = evt.seq
          if (evt.type === 'switch-master') {
            const data = typeof evt.data === 'string' ? JSON.parse(evt.data) : evt.data
            const connName = store.connections.find(c => c.id === connId)?.name || connId
            const newAddr = data?.new_addr || ''
            toast(`Sentinel failover: ${connName} → ${newAddr}`, 'warning')
          }
        }
      } catch (_e) { /* ignore poll errors */ }
    }, SENTINEL_POLL_INTERVAL)
  }

  function stopSentinelPoll(connId: string) {
    if (sentinelTimers.value[connId]) {
      clearInterval(sentinelTimers.value[connId])
      delete sentinelTimers.value[connId]
    }
  }

  async function handleDisconnect(connId: string) {
    const count = (retryCount.value[connId] || 0) + 1
    retryCount.value[connId] = count

    if (count > MAX_RETRIES) {
      stopHeartbeat(connId)
      store.setConnState(connId, {
        ...store.getConnState(connId),
        status: 'error',
        error: 'Connection lost after max retries',
      })
      toast('Connection lost: ' + (store.connections.find(c => c.id === connId)?.name || connId), 'error')
      return
    }

    store.setConnState(connId, {
      ...store.getConnState(connId),
      status: 'connecting',
    })

    const delay = BASE_RETRY_DELAY * Math.pow(2, count - 1)
    await new Promise(r => setTimeout(r, delay))

    try {
      const savedDb = store.getConnState(connId).currentDb
      await store.connect(connId, savedDb)
      retryCount.value[connId] = 0
      toast('Reconnected: ' + (store.connections.find(c => c.id === connId)?.name || connId), 'success')
      startHeartbeat(connId)
    } catch (_e) {
      // Will retry on next interval or recursion
      handleDisconnect(connId)
    }
  }

  // Watch for new connections to start heartbeat
  watch(() => ({ ...store.connStates }), (states) => {
    for (const [id, state] of Object.entries(states)) {
      if (state.status === 'connected' && !timers.value[id]) {
        startHeartbeat(id)
      } else if (state.status === 'disconnected' && timers.value[id]) {
        stopHeartbeat(id)
      }
    }
  }, { deep: true })

  onUnmounted(() => {
    for (const id of Object.keys(timers.value)) {
      stopHeartbeat(id)
    }
    for (const id of Object.keys(sentinelTimers.value)) {
      stopSentinelPoll(id)
    }
  })

  return { startHeartbeat, stopHeartbeat }
}
