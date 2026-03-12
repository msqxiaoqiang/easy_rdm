<template>
  <a-modal
    :visible="visible"
    :title="connection ? $t('connection.edit') : $t('connection.new')"
    :width="640"
    :mask-closable="true"
    unmount-on-close
    @cancel="$emit('close')"
  >
    <div class="conn-form-body">
      <!-- 左侧导航 -->
      <div class="conn-form-tabs">
        <a-button
          v-for="tab in tabs"
          :key="tab.key"
          :class="['conn-form-tab', { active: activeTab === tab.key }]"
          type="text"
          long
          @click="activeTab = tab.key"
        >{{ tab.label }}</a-button>
      </div>

      <!-- 右侧内容 -->
      <div class="conn-form-content">
        <!-- 常规 -->
        <div v-show="activeTab === 'general'" class="conn-form-section">
          <div class="form-row">
            <label class="form-label">{{ $t('connection.name') }}</label>
            <a-input v-model="form.name" :placeholder="$t('connection.name')" />
          </div>
          <div class="form-row two-col">
            <div class="form-col">
              <label class="form-label">{{ $t('connection.host') }}</label>
              <a-input v-model="form.host" placeholder="127.0.0.1" />
            </div>
            <div class="form-col" style="max-width: 120px">
              <label class="form-label">{{ $t('connection.port') }}</label>
              <a-input-number v-model="form.port" :min="1" placeholder="6379" style="width: 100%" />
            </div>
          </div>
          <div class="form-row">
            <label class="form-label">{{ $t('connection.password') }}</label>
            <a-input-password v-model="form.password" :placeholder="connection?.password_encrypted ? $t('connection.passwordKeepHint') : $t('connection.passwordPlaceholder')" />
          </div>
          <div class="form-row two-col">
            <div class="form-col">
              <label class="form-label">{{ $t('connection.username') }}</label>
              <a-input v-model="form.username" :placeholder="$t('connection.usernamePlaceholder')" />
            </div>
            <div class="form-col" style="max-width: 120px">
              <label class="form-label">DB</label>
              <a-input-number v-model="form.db" :min="0" placeholder="0" style="width: 100%" />
            </div>
          </div>
          <div class="form-row">
            <label class="form-label">{{ $t('connection.group') }}</label>
            <a-select v-model="form.group" :placeholder="$t('connection.noGroup')" allow-clear>
              <a-option v-for="opt in groupOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-option>
            </a-select>
          </div>
        </div>

        <!-- SSH -->
        <div v-show="activeTab === 'ssh'" class="conn-form-section">
          <div class="form-row">
            <a-checkbox v-model="form.use_ssh">{{ $t('connection.enableSSH') }}</a-checkbox>
          </div>
          <template v-if="form.use_ssh">
            <div class="form-row two-col">
              <div class="form-col">
                <label class="form-label">{{ $t('connection.sshHost') }}</label>
                <a-input v-model="form.ssh_host" placeholder="127.0.0.1" />
              </div>
              <div class="form-col" style="max-width: 120px">
                <label class="form-label">{{ $t('connection.sshPort') }}</label>
                <a-input-number v-model="form.ssh_port" :min="1" placeholder="22" style="width: 100%" />
              </div>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.sshUsername') }}</label>
              <a-input v-model="form.ssh_username" placeholder="root" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.sshPassword') }}</label>
              <a-input-password v-model="form.ssh_password" :placeholder="connection?.ssh_password_encrypted ? $t('connection.passwordKeepHint') : $t('connection.sshPasswordPlaceholder')" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.sshPrivateKey') }}</label>
              <a-input v-model="form.ssh_private_key" :placeholder="$t('connection.sshPrivateKeyPlaceholder')" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.sshPassphrase') }}</label>
              <a-input-password v-model="form.ssh_passphrase" :placeholder="$t('connection.sshPassphrasePlaceholder')" />
            </div>
          </template>
        </div>

        <!-- TLS/SSL -->
        <div v-show="activeTab === 'tls'" class="conn-form-section">
          <div class="form-row">
            <a-checkbox v-model="form.use_tls">{{ $t('connection.enableTLS') }}</a-checkbox>
          </div>
          <template v-if="form.use_tls">
            <div class="form-row">
              <label class="form-label">{{ $t('connection.tlsCA') }}</label>
              <a-input v-model="form.tls_ca_file" :placeholder="$t('connection.tlsCAPlaceholder')" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.tlsCert') }}</label>
              <a-input v-model="form.tls_cert_file" :placeholder="$t('connection.tlsCertPlaceholder')" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.tlsKey') }}</label>
              <a-input v-model="form.tls_key_file" :placeholder="$t('connection.tlsKeyPlaceholder')" />
            </div>
            <div class="form-row">
              <a-checkbox v-model="form.tls_skip_verify">{{ $t('connection.tlsSkipVerify') }}</a-checkbox>
            </div>
          </template>
        </div>

        <!-- 高级 -->
        <div v-show="activeTab === 'advanced'" class="conn-form-section">
          <div class="form-row two-col">
            <div class="form-col">
              <label class="form-label">{{ $t('connection.keySeparator') }}</label>
              <a-input v-model="form.key_separator" placeholder=":" />
            </div>
            <div class="form-col">
              <label class="form-label">{{ $t('connection.scanCount') }}</label>
              <a-input-number v-model="form.scan_count" :min="10" placeholder="200" style="width: 100%" />
            </div>
          </div>
          <div class="form-row">
            <label class="form-label">{{ $t('connection.keyFilter') }}</label>
            <a-input v-model="form.key_filter" :placeholder="$t('connection.keyFilterPlaceholder')" />
          </div>
          <div class="form-row two-col">
            <div class="form-col">
              <label class="form-label">{{ $t('connection.defaultView') }}</label>
              <a-select v-model="form.default_view">
                <a-option value="tree">{{ $t('common.treeView') }}</a-option>
                <a-option value="flat">{{ $t('common.flatView') }}</a-option>
              </a-select>
            </div>
            <div class="form-col">
              <label class="form-label">{{ $t('connection.connTimeout') }}</label>
              <a-input-number v-model="form.conn_timeout" :min="1" placeholder="10" style="width: 100%" />
            </div>
          </div>
          <div class="form-row two-col">
            <div class="form-col">
              <label class="form-label">{{ $t('connection.execTimeout') }}</label>
              <a-input-number v-model="form.exec_timeout" :min="1" placeholder="10" style="width: 100%" />
            </div>
            <div class="form-col">
              <label class="form-label">{{ $t('connection.dbFilterMode') }}</label>
              <a-select v-model="form.db_filter_mode">
                <a-option value="all">{{ $t('connection.dbFilterAll') }}</a-option>
                <a-option value="include">{{ $t('connection.dbFilterInclude') }}</a-option>
                <a-option value="exclude">{{ $t('connection.dbFilterExclude') }}</a-option>
              </a-select>
            </div>
          </div>
          <div v-if="form.db_filter_mode !== 'all'" class="form-row">
            <label class="form-label">{{ $t('connection.dbFilterList') }}</label>
            <a-input v-model="dbFilterListStr" :placeholder="$t('connection.dbFilterListPlaceholder')" />
          </div>
        </div>

        <!-- 哨兵模式 -->
        <div v-show="activeTab === 'sentinel'" class="conn-form-section">
          <div class="form-row">
            <a-checkbox v-model="form.use_sentinel" :disabled="form.use_cluster">{{ $t('connection.enableSentinel') }}</a-checkbox>
          </div>
          <template v-if="form.use_sentinel">
            <div v-if="form.use_ssh" class="form-row">
              <div class="sentinel-warning">{{ $t('connection.sshSentinelWarning') }}</div>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.sentinelAddrs') }}</label>
              <a-input v-model="form.sentinel_addrs" :placeholder="$t('connection.sentinelAddrsPlaceholder')" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.sentinelMasterName') }}</label>
              <a-input v-model="form.sentinel_master_name" :placeholder="$t('connection.sentinelMasterNamePlaceholder')" />
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.sentinelPassword') }}</label>
              <a-input-password v-model="form.sentinel_password" :placeholder="connection?.sentinel_password_encrypted ? $t('connection.passwordKeepHint') : $t('connection.sentinelPasswordPlaceholder')" />
            </div>
          </template>
        </div>

        <!-- 集群模式 -->
        <div v-show="activeTab === 'cluster'" class="conn-form-section">
          <div class="form-row">
            <a-checkbox v-model="form.use_cluster" :disabled="form.use_sentinel">{{ $t('connection.enableCluster') }}</a-checkbox>
          </div>
          <template v-if="form.use_cluster">
            <div v-if="form.use_ssh" class="form-row">
              <div class="sentinel-warning">{{ $t('connection.sshClusterWarning') }}</div>
            </div>
            <div class="form-row">
              <label class="form-label">{{ $t('connection.clusterAddrs') }}</label>
              <a-input v-model="form.cluster_addrs" :placeholder="$t('connection.clusterAddrsPlaceholder')" />
            </div>
            <div class="form-row">
              <div class="cluster-hint">{{ $t('connection.clusterHostHint') }}</div>
            </div>
          </template>
        </div>

        <!-- 网络代理 -->
        <div v-show="activeTab === 'proxy'" class="conn-form-section">
          <div class="form-row">
            <a-checkbox v-model="form.use_proxy">{{ $t('connection.enableProxy') }}</a-checkbox>
          </div>
          <template v-if="form.use_proxy">
            <div class="form-row">
              <label class="form-label">{{ $t('connection.proxyType') }}</label>
              <a-select v-model="form.proxy_type">
                <a-option value="socks5">SOCKS5</a-option>
                <a-option value="socks5h">SOCKS5H</a-option>
                <a-option value="http">HTTP</a-option>
                <a-option value="https">HTTPS</a-option>
              </a-select>
            </div>
            <div class="form-row two-col">
              <div class="form-col">
                <label class="form-label">{{ $t('connection.proxyHost') }}</label>
                <a-input v-model="form.proxy_host" placeholder="127.0.0.1" />
              </div>
              <div class="form-col" style="max-width: 120px">
                <label class="form-label">{{ $t('connection.proxyPort') }}</label>
                <a-input-number v-model="form.proxy_port" :min="1" placeholder="1080" style="width: 100%" />
              </div>
            </div>
            <div class="form-row two-col">
              <div class="form-col">
                <label class="form-label">{{ $t('connection.proxyUsername') }}</label>
                <a-input v-model="form.proxy_username" :placeholder="$t('connection.proxyUsernamePlaceholder')" />
              </div>
              <div class="form-col">
                <label class="form-label">{{ $t('connection.proxyPassword') }}</label>
                <a-input-password v-model="form.proxy_password" :placeholder="$t('connection.proxyPasswordPlaceholder')" />
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="conn-form-footer">
        <a-button @click="handleTest" :loading="testing">{{ testing ? $t('connection.testing') : $t('connection.test') }}</a-button>
        <div style="flex:1"></div>
        <a-button @click="$emit('close')">{{ $t('common.cancel') }}</a-button>
        <a-button type="primary" @click="handleSave" :loading="saving">{{ $t('common.save') }}</a-button>
      </div>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useConnectionStore, type Connection } from '../../stores/connection'
import { useI18n } from 'vue-i18n'
import { showMessage } from '@/utils/platform'

const props = withDefaults(defineProps<{ connection?: Connection; visible?: boolean }>(), { visible: true })
const emit = defineEmits<{ close: []; saved: [] }>()

const { t } = useI18n()
const connectionStore = useConnectionStore()

const activeTab = ref('general')
const testing = ref(false)
const saving = ref(false)

// 分组下拉选项：从 store 的 groupMeta 构建
const groupOptions = computed(() => {
  return Object.entries(connectionStore.groupMeta).map(([id, name]) => ({
    value: id,
    label: name,
  }))
})

const tabs = computed(() => [
  { key: 'general', label: t('settings.general') },
  { key: 'advanced', label: t('connection.advanced') },
  { key: 'ssh', label: 'SSH' },
  { key: 'tls', label: 'TLS/SSL' },
  { key: 'sentinel', label: t('connection.sentinel') },
  { key: 'cluster', label: t('connection.cluster') },
  { key: 'proxy', label: t('connection.proxy') },
])

const form = reactive<Connection>({
  id: '',
  name: '',
  host: '127.0.0.1',
  port: 6379,
  password: '',
  username: '',
  db: 0,
  conn_type: 'tcp',
  conn_timeout: 10,
  exec_timeout: 10,
  group: '',
  key_separator: ':',
  key_filter: '',
  scan_count: 200,
  default_view: 'tree',
  db_filter_mode: 'all',
  db_filter_list: [],
  use_tls: false,
  tls_cert_file: '',
  tls_key_file: '',
  tls_ca_file: '',
  tls_skip_verify: false,
  use_ssh: false,
  ssh_host: '',
  ssh_port: 22,
  ssh_username: '',
  ssh_password: '',
  ssh_private_key: '',
  ssh_passphrase: '',
  use_proxy: false,
  proxy_type: 'socks5',
  proxy_host: '',
  proxy_port: 1080,
  proxy_username: '',
  proxy_password: '',
  use_sentinel: false,
  sentinel_addrs: '',
  sentinel_master_name: '',
  sentinel_password: '',
  use_cluster: false,
  cluster_addrs: '',
})

const dbFilterListStr = computed({
  get: () => (form.db_filter_list || []).join(','),
  set: (val: string) => {
    form.db_filter_list = val
      .split(',')
      .map(s => parseInt(s.trim(), 10))
      .filter(n => !isNaN(n) && n >= 0)
  },
})

// 哨兵/集群互斥
watch(() => form.use_sentinel, (val) => {
  if (val) form.use_cluster = false
})
watch(() => form.use_cluster, (val) => {
  if (val) form.use_sentinel = false
})

onMounted(() => {
  if (props.connection) {
    Object.assign(form, props.connection)
    // 编辑模式不显示已加密的密码
    if (props.connection.password_encrypted) {
      form.password = ''
    }
    if ((props.connection as any).ssh_password_encrypted) {
      form.ssh_password = ''
    }
    if (props.connection.sentinel_password_encrypted) {
      form.sentinel_password = ''
    }
  }
})

async function handleTest() {
  testing.value = true
  try {
    const timeout = new Promise<never>((_, reject) =>
      setTimeout(() => reject(new Error(t('connection.testFailed') + ' (timeout 10s)')), 10000)
    )
    const testParams: Partial<Connection> = {
      host: form.host,
      port: form.port,
      password: form.password,
      username: form.username,
      db: form.db,
      conn_type: form.conn_type,
      conn_timeout: form.conn_timeout,
      use_tls: form.use_tls,
      tls_cert_file: form.tls_cert_file,
      tls_key_file: form.tls_key_file,
      tls_ca_file: form.tls_ca_file,
      tls_skip_verify: form.tls_skip_verify,
      use_ssh: form.use_ssh,
      ssh_host: form.ssh_host,
      ssh_port: form.ssh_port,
      ssh_username: form.ssh_username,
      ssh_password: form.ssh_password,
      ssh_private_key: form.ssh_private_key,
      ssh_passphrase: form.ssh_passphrase,
      use_proxy: form.use_proxy,
      proxy_type: form.proxy_type,
      proxy_host: form.proxy_host,
      proxy_port: form.proxy_port,
      proxy_username: form.proxy_username,
      proxy_password: form.proxy_password,
      use_sentinel: form.use_sentinel,
      sentinel_addrs: form.sentinel_addrs,
      sentinel_master_name: form.sentinel_master_name,
      sentinel_password: form.sentinel_password,
      use_cluster: form.use_cluster,
      cluster_addrs: form.cluster_addrs,
    }
    // 编辑模式下密码为空 = 未修改，传原始连接 ID 让后端查找已存储的密码
    if (props.connection && !form.password) {
      testParams.id = props.connection.id
      testParams.password_encrypted = true
    }
    if (props.connection && !form.ssh_password) {
      testParams.id = props.connection.id
      ;(testParams as any).ssh_password_encrypted = true
    }
    if (props.connection && !form.sentinel_password) {
      testParams.id = props.connection.id
      ;(testParams as any).sentinel_password_encrypted = true
    }
    await Promise.race([
      connectionStore.testConnection(testParams),
      timeout,
    ])
    showMessage('success', t('connection.testSuccess'))
  } catch (e: any) {
    showMessage('error', e.message || t('connection.testFailed'))
  } finally {
    testing.value = false
  }
}

async function handleSave() {
  if (!form.name.trim()) {
    form.name = `${form.host}:${form.port}`
  }
  saving.value = true
  try {
    const data: Connection = { ...form }
    if (!data.id) {
      data.id = connectionStore.generateId()
    }
    // form.group 已经是 group ID 或空字符串，无需额外解析
    if (!data.group) {
      data.group = ''
    }
    // 编辑模式下处理密码
    if (props.connection) {
      if (data.password) {
        data.password_encrypted = false
      } else {
        data.password = props.connection.password
        data.password_encrypted = props.connection.password_encrypted
      }
      // SSH 密码同理
      if ((data as any).ssh_password) {
        ;(data as any).ssh_password_encrypted = false
      } else {
        ;(data as any).ssh_password = (props.connection as any).ssh_password
        ;(data as any).ssh_password_encrypted = (props.connection as any).ssh_password_encrypted
      }
      // 哨兵密码同理
      if (data.sentinel_password) {
        data.sentinel_password_encrypted = false
      } else {
        data.sentinel_password = props.connection.sentinel_password
        data.sentinel_password_encrypted = props.connection.sentinel_password_encrypted
      }
    }
    await connectionStore.saveConnection(data)
    emit('saved')
  } catch (e: any) {
    showMessage('error', e?.message || t('common.failed'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.conn-form-body {
  display: flex;
  overflow: hidden;
  min-height: 380px;
}

.conn-form-tabs {
  width: 100px;
  flex-shrink: 0;
  border-right: 1px solid var(--color-border-1);
  padding: var(--spacing-sm) 0;
}

.conn-form-tab {
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

.conn-form-tab:hover {
  color: var(--color-text-1) !important;
  background: var(--color-fill-1) !important;
}

.conn-form-tab.active {
  color: var(--color-primary) !important;
  border-left-color: var(--color-primary);
  background: var(--color-primary-bg) !important;
}

.conn-form-content {
  flex: 1;
  padding: var(--spacing-lg);
  overflow-y: auto;
}

.conn-form-section {
  display: flex;
  flex-direction: column;
}

.section-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-3);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding-bottom: var(--spacing-xs);
  border-bottom: 1px solid var(--color-border-1);
  margin-top: var(--spacing-lg);
  margin-bottom: var(--spacing-md);
}

.form-row.two-col {
  display: flex;
  gap: var(--spacing-md);
}

.form-col {
  flex: 1;
}


.sentinel-warning {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-warning-bg);
  border: 1px solid rgba(255, 125, 0, 0.3);
  border-radius: var(--radius-md);
  color: var(--color-warning);
  font-size: var(--font-size-xs);
  line-height: 1.5;
}

.cluster-hint {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--color-fill-1);
  border-radius: var(--radius-md);
  color: var(--color-text-3);
  font-size: var(--font-size-xs);
  line-height: 1.5;
}

.conn-form-footer {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  width: 100%;
}
</style>
