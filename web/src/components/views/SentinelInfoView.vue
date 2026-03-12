<template>
  <div class="sentinel-view">
    <div class="sentinel-header">
      <h3 class="sentinel-title">{{ $t('sentinel.title') }}</h3>
      <a-button size="small" @click="loadInfo" :disabled="loading">
        <template #icon>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
        </template>
      </a-button>
    </div>

    <div v-if="loading" class="sentinel-loading">{{ $t('common.loading') }}</div>
    <div v-else-if="error" class="sentinel-error">{{ error }}</div>
    <template v-else>
      <!-- Master 信息 -->
      <div class="info-section">
        <div class="info-section-title">{{ $t('sentinel.master') }}</div>
        <div class="info-table" v-if="master">
          <div class="info-row">
            <span class="info-key">{{ $t('sentinel.masterName') }}</span>
            <span class="info-val">{{ masterName }}</span>
          </div>
          <div class="info-row">
            <span class="info-key">{{ $t('sentinel.address') }}</span>
            <span class="info-val">{{ master.host }}:{{ master.port }}</span>
          </div>
          <template v-if="masterInfo">
            <div class="info-row" v-for="key in masterDisplayKeys" :key="key">
              <span class="info-key">{{ key }}</span>
              <span class="info-val">{{ masterInfo[key] ?? '-' }}</span>
            </div>
          </template>
        </div>
        <div v-else class="info-empty">{{ $t('common.noData') }}</div>
      </div>

      <!-- Slave 列表 -->
      <div class="info-section">
        <div class="info-section-title">{{ $t('sentinel.slaves') }} ({{ slaves.length }})</div>
        <div class="node-list" v-if="slaves.length">
          <div class="node-card" v-for="(slave, i) in slaves" :key="i">
            <div class="node-row" v-for="key in slaveDisplayKeys" :key="key">
              <span class="node-key">{{ key }}</span>
              <span class="node-val">{{ slave[key] ?? '-' }}</span>
            </div>
          </div>
        </div>
        <div v-else class="info-empty">{{ $t('common.noData') }}</div>
      </div>

      <!-- Sentinel 节点列表 -->
      <div class="info-section">
        <div class="info-section-title">{{ $t('sentinel.sentinels') }} ({{ sentinels.length }})</div>
        <div class="node-list" v-if="sentinels.length">
          <div class="node-card" v-for="(s, i) in sentinels" :key="i">
            <div class="node-row" v-for="key in sentinelDisplayKeys" :key="key">
              <span class="node-key">{{ key }}</span>
              <span class="node-val">{{ s[key] ?? '-' }}</span>
            </div>
          </div>
        </div>
        <div v-else class="info-empty">{{ $t('common.noData') }}</div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()

const loading = ref(false)
const error = ref('')
const masterName = ref('')
const master = ref<Record<string, string> | null>(null)
const masterInfo = ref<Record<string, string> | null>(null)
const slaves = ref<Record<string, string>[]>([])
const sentinels = ref<Record<string, string>[]>([])

const masterDisplayKeys = ['flags', 'num-slaves', 'num-other-sentinels', 'quorum', 'failover-timeout', 'parallel-syncs']
const slaveDisplayKeys = ['name', 'ip', 'port', 'flags', 'master-host', 'master-port', 'slave-repl-offset']
const sentinelDisplayKeys = ['name', 'ip', 'port', 'flags', 'last-hello-message', 'voted-leader']

async function loadInfo() {
  loading.value = true
  error.value = ''
  try {
    const res = await request<any>('get_sentinel_info', { params: { conn_id: props.connId } })
    const data = res.data
    masterName.value = data.master_name || ''
    master.value = data.master || null
    masterInfo.value = data.master_info || null
    slaves.value = data.slaves || []
    sentinels.value = data.sentinels || []
  } catch (e: any) {
    error.value = e.message || t('sentinel.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(loadInfo)
</script>

<style scoped>
.sentinel-view {
  padding: var(--spacing-lg);
  overflow-y: auto;
  height: 100%;
}

.sentinel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-lg);
}

.sentinel-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-1);
}

.sentinel-loading,
.sentinel-error {
  padding: var(--spacing-xl);
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-sm);
}

.sentinel-error {
  color: var(--color-error);
}

.info-section {
  margin-bottom: var(--spacing-lg);
}

.info-section-title {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-2);
  margin-bottom: var(--spacing-sm);
  padding-bottom: var(--spacing-xs);
  border-bottom: 1px solid var(--color-border-1);
}

.info-table {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-xs) var(--spacing-lg);
}

.info-row {
  display: flex;
  align-items: baseline;
  gap: var(--spacing-sm);
}

.info-key {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  min-width: 140px;
  flex-shrink: 0;
}

.info-val {
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  word-break: break-all;
}

.info-empty {
  padding: var(--spacing-md);
  text-align: center;
  color: var(--color-text-4);
  font-size: var(--font-size-sm);
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.node-card {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-md);
}

.node-row {
  display: flex;
  align-items: baseline;
  gap: var(--spacing-sm);
  padding: 2px 0;
}

.node-key {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  min-width: 120px;
  flex-shrink: 0;
}

.node-val {
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  word-break: break-all;
}
</style>
