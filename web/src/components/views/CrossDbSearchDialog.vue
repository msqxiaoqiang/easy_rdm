<template>
  <a-modal
    :visible="visible"
    :title="$t('crossDbSearch.title')"
    :width="520"
    unmount-on-close
    :body-style="{ maxHeight: '60vh', overflow: 'auto' }"
    @cancel="close"
  >
    <div class="search-row">
      <a-input
        v-model="pattern"
        :placeholder="$t('crossDbSearch.patternPlaceholder')"
        @keydown.enter="doSearch"
      />
      <a-button type="primary" :disabled="loading" @click="doSearch">
        {{ loading ? $t('common.loading') : $t('common.search') }}
      </a-button>
    </div>

    <!-- 结果区域 -->
    <div class="results-area" v-if="results.length > 0 || searched">
      <div class="results-summary" v-if="searched">
        {{ $t('crossDbSearch.scannedDbs', { count: scannedDbs }) }}，{{ $t('crossDbSearch.foundKeys', { count: totalFound }) }}
      </div>

      <div v-if="results.length === 0 && searched && !loading" class="no-results">
        {{ $t('common.noData') }}
      </div>

      <div v-for="group in results" :key="group.db" class="db-group">
        <div class="db-group-header" @click="toggleGroup(group.db)">
          <span class="tree-arrow" :class="{ collapsed: collapsedGroups.has(group.db) }"><IconRight :size="12" /></span>
          <span class="db-label">db{{ group.db }}</span>
          <span class="db-count">({{ group.keys.length }}{{ group.has_more ? '+' : '' }} / {{ group.total }})</span>
        </div>
        <div v-if="!collapsedGroups.has(group.db)" class="db-group-keys">
          <div
            v-for="item in group.keys"
            :key="item.key"
            class="key-row"
            @click="jumpToKey(group.db, item.key, item.type)"
          >
            <span :class="['type-badge', item.type]">{{ typeLabel(item.type) }}</span>
            <span class="key-name" v-ellipsis-tip>{{ item.key }}</span>
            <span :class="['key-ttl', ttlClass(item.ttl)]">{{ formatTTL(item.ttl) }}</span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <a-button @click="close">{{ $t('common.close') }}</a-button>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { IconRight } from '@arco-design/web-vue/es/icon'
import { showMessage } from '@/utils/platform'

interface SearchKeyItem {
  key: string
  type: string
  ttl: number
}

interface DBResult {
  db: number
  keys: SearchKeyItem[]
  total: number
  has_more: boolean
}

const props = defineProps<{ connId: string; visible: boolean }>()
const emit = defineEmits<{ close: []; jump: [db: number, key: string, type: string] }>()

const { t } = useI18n()
const pattern = ref('')
const loading = ref(false)
const searched = ref(false)
const results = ref<DBResult[]>([])
const scannedDbs = ref(0)
const collapsedGroups = ref(new Set<number>())

const totalFound = ref(0)

async function doSearch() {
  if (loading.value) return
  const p = pattern.value.trim()
  if (!p || p === '*') {
    showMessage('error', t('crossDbSearch.patternRequired'))
    return
  }
  loading.value = true
  searched.value = true
  results.value = []
  collapsedGroups.value = new Set()
  try {
    const res = await request<{ results: DBResult[]; scanned_dbs: number }>('cross_db_search', {
      params: { conn_id: props.connId, pattern: p, max_per_db: 100 },
    })
    if (res.data) {
      results.value = res.data.results || []
      scannedDbs.value = res.data.scanned_dbs
      totalFound.value = results.value.reduce((sum, g) => sum + g.keys.length, 0)
    }
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    loading.value = false
  }
}

function toggleGroup(db: number) {
  const s = new Set(collapsedGroups.value)
  if (s.has(db)) s.delete(db)
  else s.add(db)
  collapsedGroups.value = s
}

function jumpToKey(db: number, key: string, type: string) {
  emit('jump', db, key, type)
  close()
}

function close() {
  emit('close')
}

function typeLabel(type: string): string {
  const map: Record<string, string> = {
    string: 'S', list: 'L', set: 'E', zset: 'Z',
    hash: 'H', stream: 'X', geo: 'G', none: '?',
  }
  return map[type] || type[0]?.toUpperCase() || '?'
}

function ttlClass(ttl: number): string {
  if (ttl === -1) return 'permanent'
  if (ttl <= 60) return 'danger'
  if (ttl <= 3600) return 'warning'
  return 'safe'
}

function formatTTL(ttl: number): string {
  if (ttl === -1) return '∞'
  if (ttl === -2) return '-'
  if (ttl < 60) return ttl + 's'
  if (ttl < 3600) return Math.floor(ttl / 60) + 'm'
  if (ttl < 86400) return Math.floor(ttl / 3600) + 'h'
  return Math.floor(ttl / 86400) + 'd'
}
</script>

<style scoped>
.search-row {
  display: flex;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-sm);
}

.results-area {
  min-height: 0;
}

.results-summary {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
  margin-bottom: var(--spacing-sm);
}

.no-results {
  text-align: center;
  color: var(--color-text-4);
  padding: var(--spacing-lg) 0;
  font-size: var(--font-size-sm);
}

.db-group {
  margin-bottom: var(--spacing-xs);
}

.db-group-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: 4px var(--spacing-xs);
  cursor: pointer;
  font-size: var(--font-size-sm);
  color: var(--color-text-2);
  user-select: none;
  border-radius: var(--radius-sm);
}

.db-group-header:hover {
  background: var(--color-fill-1);
}

.tree-arrow {
  display: inline-block;
  font-size: var(--font-size-xs);
  transition: transform 0.15s;
  transform: rotate(90deg);
}

.tree-arrow.collapsed {
  transform: rotate(0deg);
}

.db-label {
  font-weight: 600;
  color: var(--color-text-1);
}

.db-count {
  font-size: var(--font-size-xs);
  color: var(--color-text-4);
}

.db-group-keys {
  padding-left: var(--spacing-md);
}

.key-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: 3px var(--spacing-xs);
  cursor: pointer;
  font-size: var(--font-size-xs);
  border-radius: var(--radius-sm);
  transition: background var(--transition-fast);
}

.key-row:hover {
  background: var(--color-primary-bg);
}

.type-badge {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 600;
  flex-shrink: 0;
  color: #fff;
}

.type-badge.string { background: #4080ff; }
.type-badge.list { background: #00b42a; }
.type-badge.set { background: #ff7d00; }
.type-badge.zset { background: #f53f3f; }
.type-badge.hash { background: #722ed1; }
.type-badge.stream { background: #0fc6c2; }
.type-badge.geo { background: #13c2c2; }
.type-badge.ReJSON-RL { background: #e8590c; }

.key-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
}

.key-ttl {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
}

.key-ttl.permanent { color: var(--color-ttl-permanent); }
.key-ttl.danger { color: var(--color-ttl-danger); }
.key-ttl.warning { color: var(--color-ttl-warning); }
.key-ttl.safe { color: var(--color-ttl-safe); }
</style>
