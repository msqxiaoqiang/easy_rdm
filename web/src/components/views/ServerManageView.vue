<template>
  <div class="server-manage">
    <!-- 内部 Tab 栏 -->
    <div class="manage-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['manage-tab', { active: activeTab === tab.key }]"
        @click="activeTab = tab.key"
      >{{ $t(tab.label) }}</button>
    </div>

    <!-- Config 配置管理 -->
    <div v-if="activeTab === 'config'" class="manage-panel">
      <div class="panel-toolbar">
        <input
          v-model="configSearch"
          class="panel-search"
          :placeholder="$t('server.configSearch')"
        />
        <button class="panel-btn" @click="loadConfig">{{ $t('common.refresh') }}</button>
        <button class="panel-btn primary" @click="configRewrite">{{ $t('server.configRewrite') }}</button>
      </div>
      <div class="table-wrap">
        <a-table
          :data="filteredConfig"
          :pagination="false"
          :bordered="false"
          size="medium"
          :sticky-header="true"
          row-key="key"
        >
          <template #columns>
            <a-table-column title="Key" data-index="key" :width="'40%'">
              <template #cell="{ record }"><span class="mono">{{ record.key }}</span></template>
            </a-table-column>
            <a-table-column title="Value" data-index="value">
              <template #cell="{ record }">
                <template v-if="editingConfig === record.key">
                  <input
                    v-model="editingConfigValue"
                    class="inline-input"
                    @keydown.enter="saveConfigItem(record.key)"
                    @keydown.escape="editingConfig = ''"
                  />
                </template>
                <span v-else class="mono">{{ record.value }}</span>
              </template>
            </a-table-column>
            <a-table-column :width="60">
              <template #cell="{ record }">
                <button
                  v-if="editingConfig === record.key"
                  class="mini-btn save"
                  @click="saveConfigItem(record.key)"
                  :data-tooltip="$t('common.save')" data-tooltip-pos="left"
                ><IconCheck :size="14" /></button>
                <button
                  v-else
                  class="mini-btn"
                  @click="startEditConfig(record)"
                  :data-tooltip="$t('common.edit')" data-tooltip-pos="left"
                ><IconEdit :size="14" /></button>
              </template>
            </a-table-column>
          </template>
          <template #empty>
            <div class="empty-state">{{ $t('common.noData') }}</div>
          </template>
        </a-table>
      </div>
    </div>

    <!-- Persistence 持久化 -->
    <div v-if="activeTab === 'persistence'" class="manage-panel">
      <div class="panel-toolbar">
        <button class="panel-btn" @click="loadPersistence">{{ $t('common.refresh') }}</button>
        <button class="panel-btn primary" @click="triggerBgsave">{{ $t('server.bgsave') }}</button>
        <button class="panel-btn primary" @click="triggerBgrewriteaof">{{ $t('server.bgrewriteaof') }}</button>
      </div>
      <div class="persistence-grid">
        <!-- RDB -->
        <div class="persist-card">
          <h4>RDB</h4>
          <div class="persist-items">
            <div class="persist-row">
              <span class="persist-label">{{ $t('server.lastSaveTime') }}</span>
              <span class="persist-value">{{ persistInfo.rdb_last_save_time ? formatTimestamp(Number(persistInfo.rdb_last_save_time)) : '-' }}</span>
            </div>
            <div class="persist-row">
              <span class="persist-label">{{ $t('server.lastSaveStatus') }}</span>
              <span :class="['persist-value', persistInfo.rdb_last_bgsave_status === 'ok' ? 'ok' : 'err']">
                {{ persistInfo.rdb_last_bgsave_status || '-' }}
              </span>
            </div>
            <div class="persist-row">
              <span class="persist-label">{{ $t('server.rdbSaving') }}</span>
              <span class="persist-value">{{ persistInfo.rdb_bgsave_in_progress === '1' ? '✓' : '-' }}</span>
            </div>
            <div class="persist-row">
              <span class="persist-label">rdb_changes_since_last_save</span>
              <span class="persist-value mono">{{ persistInfo.rdb_changes_since_last_save || '0' }}</span>
            </div>
            <div class="persist-row">
              <span class="persist-label">rdb_current_bgsave_time_sec</span>
              <span class="persist-value mono">{{ persistInfo.rdb_current_bgsave_time_sec || '-' }}</span>
            </div>
          </div>
        </div>
        <!-- AOF -->
        <div class="persist-card">
          <h4>AOF</h4>
          <div class="persist-items">
            <div class="persist-row">
              <span class="persist-label">aof_enabled</span>
              <span :class="['persist-value', persistInfo.aof_enabled === '1' ? 'ok' : '']">
                {{ persistInfo.aof_enabled === '1' ? $t('server.aofEnabled') : $t('server.aofDisabled') }}
              </span>
            </div>
            <div class="persist-row">
              <span class="persist-label">{{ $t('server.aofRewriting') }}</span>
              <span class="persist-value">{{ persistInfo.aof_rewrite_in_progress === '1' ? '✓' : '-' }}</span>
            </div>
            <div class="persist-row">
              <span class="persist-label">aof_last_bgrewrite_status</span>
              <span :class="['persist-value', persistInfo.aof_last_bgrewrite_status === 'ok' ? 'ok' : 'err']">
                {{ persistInfo.aof_last_bgrewrite_status || '-' }}
              </span>
            </div>
            <div class="persist-row">
              <span class="persist-label">aof_current_size</span>
              <span class="persist-value mono">{{ persistInfo.aof_current_size || '-' }}</span>
            </div>
            <div class="persist-row">
              <span class="persist-label">aof_base_size</span>
              <span class="persist-value mono">{{ persistInfo.aof_base_size || '-' }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ACL 权限管理 -->
    <div v-if="activeTab === 'acl'" class="manage-panel">
      <div class="panel-toolbar">
        <span class="toolbar-info" v-if="aclCurrentUser">
          {{ $t('server.aclCurrentUser') }}: <strong>{{ aclCurrentUser }}</strong>
        </span>
        <div style="flex:1"></div>
        <button class="panel-btn" @click="loadACL">{{ $t('common.refresh') }}</button>
        <button class="panel-btn primary" @click="showAddUser = true">{{ $t('server.aclAddUser') }}</button>
      </div>
      <!-- Add/Edit user form -->
      <div v-if="showAddUser || editingUser" class="acl-form">
        <div class="acl-form-row">
          <label>{{ $t('connection.username') }}</label>
          <input
            v-model="aclForm.username"
            class="panel-search"
            :disabled="!!editingUser"
            :placeholder="$t('server.aclUsernamePlaceholder')"
          />
        </div>
        <div class="acl-form-row">
          <label>{{ $t('server.aclRules') }}</label>
          <input
            v-model="aclForm.rules"
            class="panel-search"
            style="flex:1"
            :placeholder="$t('server.aclRulesPlaceholder')"
          />
        </div>
        <div class="acl-form-row">
          <button class="panel-btn primary" @click="saveACLUser">{{ $t('common.save') }}</button>
          <button class="panel-btn" @click="cancelACLForm">{{ $t('common.cancel') }}</button>
        </div>
      </div>
      <div class="table-wrap">
        <a-table
          :data="aclUsers"
          :pagination="false"
          :bordered="false"
          size="medium"
          :sticky-header="true"
          row-key="username"
        >
          <template #columns>
            <a-table-column :title="$t('connection.username')" data-index="username">
              <template #cell="{ record }">
                <span class="mono">{{ record.username }}</span>
                <span v-if="record.is_current" class="acl-current-badge">{{ $t('server.aclCurrentUser') }}</span>
              </template>
            </a-table-column>
            <a-table-column :title="$t('server.aclFlags')" data-index="flags">
              <template #cell="{ record }">
                <span
                  v-for="flag in (record.flags || [])"
                  :key="flag"
                  :class="['acl-flag', flag === 'on' ? 'on' : flag === 'off' ? 'off' : '']"
                >{{ flag }}</span>
              </template>
            </a-table-column>
            <a-table-column :title="$t('server.aclCommands')" data-index="commands">
              <template #cell="{ record }"><span class="mono cell-ellipsis" v-ellipsis-tip>{{ record.commands || '-' }}</span></template>
            </a-table-column>
            <a-table-column :title="$t('server.aclKeys')" data-index="keys">
              <template #cell="{ record }"><span class="mono cell-ellipsis" v-ellipsis-tip>{{ record.keys || '-' }}</span></template>
            </a-table-column>
            <a-table-column :width="80">
              <template #cell="{ record }">
                <button
                  class="mini-btn"
                  @click="startEditUser(record)"
                  :data-tooltip="$t('common.edit')" data-tooltip-pos="left"
                ><IconEdit :size="14" /></button>
                <button
                  v-if="!record.is_current && record.username !== 'default'"
                  class="mini-btn danger"
                  @click="deleteACLUser(record)"
                  :data-tooltip="$t('common.delete')" data-tooltip-pos="left"
                ><IconDelete :size="14" /></button>
              </template>
            </a-table-column>
          </template>
          <template #empty>
            <template v-if="aclError">
              <div class="empty-state" style="color:var(--color-error,#f53f3f)">{{ aclError }}</div>
            </template>
            <template v-else>
              <div class="empty-state">{{ $t('common.noData') }}</div>
            </template>
          </template>
        </a-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { request } from '../../utils/request'
import { useI18n } from 'vue-i18n'
import { gmConfirm } from '../../utils/dialog'
import { useConnectionStore } from '../../stores/connection'
import { versionGte } from '../../utils/version'
import { showMessage } from '@/utils/platform'
import { IconCheck, IconEdit, IconDelete } from '@arco-design/web-vue/es/icon'

const props = defineProps<{ connId: string }>()
const { t } = useI18n()
const connectionStore = useConnectionStore()
const redisVersion = computed(() => connectionStore.getConnState(props.connId).redisVersion)

const allTabs: { key: string; label: string; minVersion?: string }[] = [
  // { key: 'config', label: 'server.config' },
  // { key: 'persistence', label: 'server.persistence' },
  // { key: 'acl', label: 'server.acl', minVersion: '6.0.0' },
]

const tabs = computed(() => allTabs.filter(t => !t.minVersion || versionGte(redisVersion.value, t.minVersion)))

const activeTab = ref('config')

// ========== Config ==========
const configItems = ref<{ key: string; value: string }[]>([])
const configSearch = ref('')
const editingConfig = ref('')
const editingConfigValue = ref('')

const filteredConfig = computed(() => {
  const q = configSearch.value.toLowerCase()
  if (!q) return configItems.value
  return configItems.value.filter(i => i.key.toLowerCase().includes(q) || i.value.toLowerCase().includes(q))
})

async function loadConfig() {
  try {
    const res = await request<{ key: string; value: string }[]>('config_get', { params: { conn_id: props.connId, pattern: '*' } })
    configItems.value = (res.data || []).sort((a, b) => a.key.localeCompare(b.key))
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function startEditConfig(item: { key: string; value: string }) {
  editingConfig.value = item.key
  editingConfigValue.value = item.value
  nextTick(() => {
    const inputs = document.querySelectorAll('.inline-input')
    if (inputs.length) (inputs[inputs.length - 1] as HTMLInputElement).focus()
  })
}

async function saveConfigItem(key: string) {
  try {
    await request('config_set', { params: { conn_id: props.connId, key, value: editingConfigValue.value } })
    showMessage('success', t('common.success'))
    const item = configItems.value.find(i => i.key === key)
    if (item) item.value = editingConfigValue.value
    editingConfig.value = ''
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function configRewrite() {
  if (!await gmConfirm(t('server.configRewriteConfirm'))) return
  try {
    await request('config_rewrite', { params: { conn_id: props.connId } })
    showMessage('success', t('server.configRewriteSuccess'))
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

function formatTimestamp(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}

// ========== Persistence ==========
const persistInfo = ref<Record<string, string>>({})

async function loadPersistence() {
  try {
    const res = await request<Record<string, string>>('persistence_info', { params: { conn_id: props.connId } })
    persistInfo.value = res.data || {}
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function triggerBgsave() {
  try {
    await request('bgsave', { params: { conn_id: props.connId } })
    showMessage('success', t('server.bgsaveSuccess'))
    setTimeout(loadPersistence, 1000)
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function triggerBgrewriteaof() {
  try {
    await request('bgrewriteaof', { params: { conn_id: props.connId } })
    showMessage('success', t('server.bgrewriteaofSuccess'))
    setTimeout(loadPersistence, 1000)
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

// ========== ACL ==========
interface ACLUser {
  username: string
  flags?: string[]
  password_count?: number
  commands?: string
  keys?: string
  channels?: string
  is_current?: boolean
  error?: string
}

const aclUsers = ref<ACLUser[]>([])
const aclCurrentUser = ref('')
const aclError = ref('')
const showAddUser = ref(false)
const editingUser = ref('')
const aclForm = ref({ username: '', rules: '' })

async function loadACL() {
  aclError.value = ''
  try {
    const res = await request<ACLUser[]>('acl_list', { params: { conn_id: props.connId } })
    aclUsers.value = res.data || []
    const current = aclUsers.value.find(u => u.is_current)
    aclCurrentUser.value = current?.username || ''
  } catch (e: any) {
    const msg = e?.message || ''
    if (msg.includes('ERR') && (msg.includes('unknown command') || msg.includes('no such command'))) {
      aclError.value = t('server.aclNotSupported')
    } else {
      aclError.value = msg || t('common.failed')
    }
    aclUsers.value = []
  }
}

function startEditUser(u: ACLUser) {
  editingUser.value = u.username
  showAddUser.value = false
  aclForm.value = { username: u.username, rules: '' }
}

function cancelACLForm() {
  showAddUser.value = false
  editingUser.value = ''
  aclForm.value = { username: '', rules: '' }
}

async function saveACLUser() {
  const username = aclForm.value.username.trim()
  if (!username) return
  const rulesStr = aclForm.value.rules.trim()
  const rules = rulesStr ? rulesStr.split(/\s+/) : []
  try {
    await request('acl_setuser', { params: { conn_id: props.connId, username, rules } })
    showMessage('success', t('common.success'))
    cancelACLForm()
    loadACL()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

async function deleteACLUser(u: ACLUser) {
  if (!await gmConfirm(t('server.aclDelConfirm', { username: u.username }))) return
  try {
    await request('acl_deluser', { params: { conn_id: props.connId, username: u.username } })
    showMessage('success', t('common.success'))
    loadACL()
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  }
}

// ========== Tab 切换自动加载 ==========
// 如果当前 tab 因版本不支持而被隐藏，回退到第一个可见 tab
watch(tabs, (visible) => {
  if (!visible.some(t => t.key === activeTab.value)) {
    activeTab.value = visible[0]?.key || 'config'
  }
})

watch(activeTab, (tab) => {
  if (tab === 'config') loadConfig()
  else if (tab === 'persistence') loadPersistence()
  else if (tab === 'acl') loadACL()
}, { immediate: true })

watch(() => props.connId, () => {
  if (activeTab.value === 'config') loadConfig()
  else if (activeTab.value === 'persistence') loadPersistence()
  else if (activeTab.value === 'acl') loadACL()
})
</script>

<style scoped>
.server-manage {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.manage-tabs {
  display: flex;
  gap: 0;
  padding: 0 var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-bg-1);
}

.manage-tab {
  padding: var(--spacing-xs) var(--spacing-lg);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.manage-tab:hover {
  color: var(--color-text-1);
}

.manage-tab.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
  font-weight: 500;
}

.manage-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
}

.panel-search {
  width: 220px;
  height: 28px;
  padding: 2px var(--spacing-sm);
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  outline: none;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.panel-search:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary-bg);
}

.panel-btn {
  height: 28px;
  padding: 0 var(--spacing-md);
  background: var(--color-fill-2);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.panel-btn:hover {
  background: var(--color-fill-3);
}

.panel-btn.primary {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: #fff;
}

.panel-btn.primary:hover {
  opacity: 0.9;
}

.panel-btn.danger {
  color: var(--color-error, #f53f3f);
}

.panel-btn.danger:hover {
  background: rgba(245, 63, 63, 0.08);
}

.toolbar-info {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}

.table-wrap {
  flex: 1;
  overflow: auto;
}

.mono {
  font-family: var(--font-family-mono);
}

.cell-ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inline-input {
  width: 100%;
  height: 24px;
  padding: 2px var(--spacing-xs);
  background: var(--color-fill-1);
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-sm);
  color: var(--color-text-1);
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  outline: none;
}

.mini-btn {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  color: var(--color-text-3);
  cursor: pointer;
  font-size: var(--font-size-xs);
  transition: all var(--transition-fast);
}

.mini-btn:hover {
  background: var(--color-fill-2);
  color: var(--color-text-1);
}

.mini-btn.save {
  color: var(--color-primary);
}

.mini-btn.danger:hover {
  color: var(--color-error, #f53f3f);
  background: rgba(245, 63, 63, 0.08);
}

.empty-state {
  padding: var(--spacing-xl);
  text-align: center;
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
}

/* Persistence */
.persistence-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg);
  overflow: auto;
}

.persist-card {
  background: var(--color-bg-2);
  border: 1px solid var(--color-border-1);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
}

.persist-card h4 {
  margin: 0 0 var(--spacing-sm) 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-1);
  font-weight: 600;
}

.persist-items {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xs);
}

.persist-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.persist-label {
  font-size: var(--font-size-xs);
  color: var(--color-text-3);
}

.persist-value {
  font-size: var(--font-size-xs);
  color: var(--color-text-1);
}

.persist-value.ok {
  color: var(--color-success, #00b42a);
}

.persist-value.err {
  color: var(--color-error, #f53f3f);
}

/* ACL */
.acl-form {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  border-bottom: 1px solid var(--color-border-1);
  background: var(--color-fill-1);
  align-items: center;
}

.acl-form-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
}

.acl-form-row label {
  font-size: var(--font-size-xs);
  color: var(--color-text-2);
  white-space: nowrap;
}

.acl-flag {
  display: inline-block;
  padding: 1px 6px;
  margin-right: 4px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  background: var(--color-fill-2);
  color: var(--color-text-2);
}

.acl-flag.on {
  background: rgba(0, 180, 42, 0.12);
  color: var(--color-success, #00b42a);
}

.acl-flag.off {
  background: rgba(245, 63, 63, 0.08);
  color: var(--color-error, #f53f3f);
}

.acl-current-badge {
  display: inline-block;
  margin-left: 6px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  background: rgba(22, 93, 255, 0.1);
  color: var(--color-primary);
}
</style>
