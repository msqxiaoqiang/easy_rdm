<template>
  <div class="cluster-view">
    <div class="cluster-header">
      <h3 class="cluster-title">{{ $t('cluster.title') }}</h3>
      <a-button size="small" @click="loadInfo" :disabled="loading">
        <template #icon>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
        </template>
      </a-button>
    </div>

    <div v-if="loading" class="cluster-loading">{{ $t('common.loading') }}</div>
    <div v-else-if="error" class="cluster-error">{{ error }}</div>
    <template v-else>
      <!-- 集群概览 -->
      <div class="info-section">
        <div class="info-section-title">{{ $t('cluster.overview') }}</div>
        <div class="info-table" v-if="infoEntries.length">
          <div class="info-row" v-for="entry in infoEntries" :key="entry.key">
            <span class="info-key">{{ entry.key }}</span>
            <span class="info-val">
              <span v-if="entry.key === 'cluster_state'" :class="['state-badge', entry.value === 'ok' ? 'ok' : 'fail']">{{ entry.value }}</span>
              <template v-else>{{ entry.value }}</template>
            </span>
          </div>
        </div>
        <div v-else class="info-empty">{{ $t('common.noData') }}</div>
      </div>

      <!-- 节点列表 -->
      <div class="info-section">
        <div class="info-section-title">{{ $t('cluster.nodes') }} ({{ nodes.length }})</div>
        <div class="node-list" v-if="nodes.length">
          <div class="node-card" v-for="(node, i) in nodes" :key="i">
            <div class="node-header">
              <span class="node-endpoint">{{ node.endpoint }}</span>
              <span :class="['role-badge', nodeRole(node.flags)]">{{ nodeRole(node.flags) }}</span>
              <span :class="['link-badge', node.link_state === 'connected' ? 'up' : 'down']">{{ node.link_state }}</span>
            </div>
            <div class="node-details">
              <div class="node-row">
                <span class="node-key">ID</span>
                <span class="node-val node-id">{{ node.id }}</span>
              </div>
              <div class="node-row">
                <span class="node-key">{{ $t('cluster.flags') }}</span>
                <span class="node-val">{{ node.flags }}</span>
              </div>
              <div class="node-row" v-if="nodeRole(node.flags) === 'slave' && node.master_id !== '-'">
                <span class="node-key">{{ $t('cluster.masterId') }}</span>
                <span class="node-val node-id">{{ node.master_id }}</span>
              </div>
              <div class="node-row" v-if="node.slots">
                <span class="node-key">{{ $t('cluster.slots') }}</span>
                <span class="node-val">{{ node.slots }}</span>
              </div>
              <div class="node-row">
                <span class="node-key">{{ $t('cluster.configEpoch') }}</span>
                <span class="node-val">{{ node.config_epoch }}</span>
              </div>
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
const infoEntries = ref<{ key: string; value: string }[]>([])
const nodes = ref<Record<string, string>[]>([])

function nodeRole(flags: string): string {
  if (flags.includes('master')) return 'master'
  if (flags.includes('slave')) return 'slave'
  return flags.split(',')[0] || 'unknown'
}

function parseClusterInfo(raw: string) {
  const entries: { key: string; value: string }[] = []
  for (const line of raw.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue
    const idx = trimmed.indexOf(':')
    if (idx > 0) {
      entries.push({ key: trimmed.slice(0, idx), value: trimmed.slice(idx + 1) })
    }
  }
  return entries
}

async function loadInfo() {
  loading.value = true
  error.value = ''
  try {
    const res = await request<any>('get_cluster_info', { params: { conn_id: props.connId } })
    const data = res.data
    infoEntries.value = data.cluster_info ? parseClusterInfo(data.cluster_info) : []
    nodes.value = data.nodes || []
  } catch (e: any) {
    error.value = e.message || t('cluster.loadFailed')
  } finally {
    loading.value = false
  }
}

onMounted(loadInfo)
</script>

<style scoped>
.cluster-view {
  padding: var(--spacing-lg);
  overflow-y: auto;
  height: 100%;
}

.cluster-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-lg);
}

.cluster-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-1);
}

.cluster-loading,
.cluster-error {
  padding: var(--spacing-xl);
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-sm);
}

.cluster-error {
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
  min-width: 160px;
  flex-shrink: 0;
}

.info-val {
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  word-break: break-all;
}

.state-badge {
  display: inline-block;
  padding: 1px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 600;
}

.state-badge.ok {
  background: var(--color-success-light, rgba(0, 180, 42, 0.1));
  color: var(--color-success, #00b42a);
}

.state-badge.fail {
  background: var(--color-error-light, rgba(245, 63, 63, 0.1));
  color: var(--color-error, #f53f3f);
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

.node-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-xs);
}

.node-endpoint {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-1);
}

.role-badge,
.link-badge {
  display: inline-block;
  padding: 0 6px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 500;
  line-height: 18px;
}

.role-badge.master {
  background: var(--color-primary-light, rgba(22, 93, 255, 0.1));
  color: var(--color-primary);
}

.role-badge.slave {
  background: var(--color-fill-3);
  color: var(--color-text-2);
}

.link-badge.up {
  background: var(--color-success-light, rgba(0, 180, 42, 0.1));
  color: var(--color-success, #00b42a);
}

.link-badge.down {
  background: var(--color-error-light, rgba(245, 63, 63, 0.1));
  color: var(--color-error, #f53f3f);
}

.node-details {
  padding-left: var(--spacing-xs);
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
  min-width: 100px;
  flex-shrink: 0;
}

.node-val {
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  word-break: break-all;
}

.node-id {
  font-family: var(--font-family-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}
</style>
