<template>
  <div class="cli-view" ref="scrollRef" @click="focusInput">
    <div class="cli-output">
      <div v-for="(line, i) in history" :key="i" :class="['cli-line', line.type]">
        <span v-if="line.type === 'cmd'" class="cli-prompt">{{ prompt }}</span>
        <!-- Multi-line result with collapse -->
        <template v-if="(line.type === 'result' || line.type === 'error') && line.lineCount && line.lineCount > collapseThreshold">
          <div class="cli-result-block">
            <pre class="cli-text">{{ line.collapsed ? truncateText(line.text) : line.text }}</pre>
            <a-button size="mini" class="cli-toggle" @click.stop="line.collapsed = !line.collapsed">
              {{ line.collapsed ? `▼ ${t('cli.showAll', { count: line.lineCount })}` : `▲ ${t('cli.collapse')}` }}
            </a-button>
          </div>
        </template>
        <template v-else>
          <pre class="cli-text">{{ line.text }}</pre>
        </template>
      </div>
    </div>
    <div class="cli-input-row">
      <span class="cli-prompt">{{ prompt }}</span>
      <div class="cli-input-wrap">
        <input
          ref="inputRef"
          v-model="command"
          class="cli-input"
          :placeholder="$t('cli.placeholder')"
          @keydown="handleKeydown"
          @input="handleInput"
          spellcheck="false"
          autocomplete="off"
        />
      </div>
    </div>
    <!-- Autocomplete dropdown (teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div v-if="suggestions.length" class="cli-suggestions" :style="suggestionsStyle">
        <div
          v-for="(s, idx) in suggestions"
          :key="s"
          :class="['cli-suggestion', { active: suggestionIdx === idx }]"
          @mousedown.prevent="applySuggestion(s)"
        >{{ s }}</div>
      </div>
    </Teleport>
    <div ref="bottomAnchorRef"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, computed, onMounted, onBeforeUnmount, reactive } from 'vue'
import { useConnectionStore } from '../../stores/connection'
import { useI18n } from 'vue-i18n'
import { request } from '../../utils/request'

interface CliLine {
  type: 'cmd' | 'result' | 'error'
  text: string
  collapsed: boolean
  lineCount: number
}

const props = defineProps<{ connId: string }>()
const connectionStore = useConnectionStore()
const { t } = useI18n()

const command = ref('')
const history = ref<CliLine[]>([])
const cmdHistory = ref<string[]>([])
const cmdHistoryIdx = ref(-1)
const scrollRef = ref<HTMLElement>()
const bottomAnchorRef = ref<HTMLElement>()
const inputRef = ref<HTMLElement>()
const suggestions = ref<string[]>([])
const suggestionIdx = ref(-1)
const collapseThreshold = 5

const suggestionsStyle = computed(() => {
  const el = inputRef.value as HTMLInputElement | undefined
  if (!el) return {}
  const rect = el.getBoundingClientRect()
  const maxH = 200
  const spaceAbove = rect.top
  const style: Record<string, string> = {
    position: 'fixed',
    left: rect.left + 'px',
    minWidth: '180px',
    maxHeight: maxH + 'px',
  }
  if (spaceAbove >= maxH) {
    // 上方空间足够，向上弹出
    style.bottom = (window.innerHeight - rect.top + 4) + 'px'
  } else {
    // 上方空间不够，向下弹出
    style.top = (rect.bottom + 4) + 'px'
  }
  return style
})

const connState = computed(() => connectionStore.getConnState(props.connId))
const prompt = computed(() => `redis db${connState.value.cliDb}> `)

// Common Redis commands for autocomplete
const redisCommands = [
  'APPEND', 'AUTH', 'BGSAVE', 'BITCOUNT', 'BITFIELD', 'BITOP', 'BITPOS',
  'BLPOP', 'BRPOP', 'CLIENT', 'CLUSTER', 'COMMAND', 'CONFIG',
  'COPY', 'DBSIZE', 'DECR', 'DECRBY', 'DEL', 'DUMP',
  'ECHO', 'EVAL', 'EVALSHA', 'EXEC', 'EXISTS', 'EXPIRE', 'EXPIREAT',
  'FLUSHALL', 'FLUSHDB',
  'GEOADD', 'GEODIST', 'GEOHASH', 'GEOPOS', 'GEOSEARCH',
  'GET', 'GETDEL', 'GETEX', 'GETRANGE', 'GETSET',
  'HDEL', 'HEXISTS', 'HGET', 'HGETALL', 'HINCRBY', 'HINCRBYFLOAT',
  'HKEYS', 'HLEN', 'HMGET', 'HMSET', 'HRANDFIELD', 'HSCAN', 'HSET', 'HSETNX', 'HVALS',
  'INCR', 'INCRBY', 'INCRBYFLOAT', 'INFO',
  'KEYS', 'LASTSAVE', 'LINDEX', 'LINSERT', 'LLEN', 'LMOVE',
  'LPOP', 'LPOS', 'LPUSH', 'LPUSHX', 'LRANGE', 'LREM', 'LSET', 'LTRIM',
  'MGET', 'MONITOR', 'MOVE', 'MSET', 'MSETNX', 'MULTI',
  'OBJECT', 'PERSIST', 'PEXPIRE', 'PEXPIREAT', 'PFADD', 'PFCOUNT', 'PFMERGE',
  'PING', 'PSETEX', 'PTTL', 'PUBLISH',
  'RANDOMKEY', 'RENAME', 'RENAMENX', 'RESTORE', 'ROLE',
  'RPOP', 'RPOPLPUSH', 'RPUSH', 'RPUSHX',
  'SADD', 'SAVE', 'SCAN', 'SCARD', 'SDIFF', 'SDIFFSTORE', 'SELECT',
  'SET', 'SETEX', 'SETNX', 'SETRANGE', 'SINTER', 'SINTERSTORE',
  'SISMEMBER', 'SMEMBERS', 'SMOVE', 'SORT', 'SPOP', 'SRANDMEMBER',
  'SREM', 'SSCAN', 'STRLEN', 'SUBSCRIBE',
  'SUNION', 'SUNIONSTORE', 'SWAPDB',
  'TIME', 'TOUCH', 'TTL', 'TYPE', 'UNLINK', 'UNWATCH',
  'WAIT', 'WATCH',
  'XACK', 'XADD', 'XCLAIM', 'XDEL', 'XGROUP', 'XINFO', 'XLEN',
  'XPENDING', 'XRANGE', 'XREAD', 'XREADGROUP', 'XREVRANGE', 'XTRIM',
  'ZADD', 'ZCARD', 'ZCOUNT', 'ZDIFF', 'ZDIFFSTORE', 'ZINCRBY',
  'ZINTER', 'ZINTERSTORE', 'ZLEXCOUNT', 'ZMPOP', 'ZMSCORE',
  'ZPOPMAX', 'ZPOPMIN', 'ZRANDMEMBER', 'ZRANGE', 'ZRANGEBYLEX',
  'ZRANGEBYSCORE', 'ZRANGESTORE', 'ZRANK', 'ZREM', 'ZREMRANGEBYLEX',
  'ZREMRANGEBYRANK', 'ZREMRANGEBYSCORE', 'ZREVRANGE', 'ZREVRANGEBYLEX',
  'ZREVRANGEBYSCORE', 'ZREVRANK', 'ZSCAN', 'ZSCORE', 'ZUNION', 'ZUNIONSTORE',
]

function scrollToBottom() {
  nextTick(() => {
    if (typeof bottomAnchorRef.value?.scrollIntoView === 'function') {
      bottomAnchorRef.value.scrollIntoView({ block: 'end' })
    }
  })
}

function focusInput() {
  (inputRef.value as HTMLInputElement)?.focus()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Tab') {
    e.preventDefault()
    handleTab()
    return
  }
  if (e.key === 'Escape') {
    closeSuggestions()
    return
  }
  if (suggestions.value.length) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      suggestionIdx.value = Math.min(suggestionIdx.value + 1, suggestions.value.length - 1)
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      suggestionIdx.value = Math.max(suggestionIdx.value - 1, 0)
      return
    }
    if (e.key === 'Enter' && suggestionIdx.value >= 0) {
      e.preventDefault()
      applySuggestion(suggestions.value[suggestionIdx.value])
      return
    }
  }

  if (e.key === 'Enter') {
    closeSuggestions()
    executeCommand()
    return
  }
  // 无 suggestions 时上下翻历史
  if (!suggestions.value.length) {
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      historyUp()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      historyDown()
      return
    }
  }
}

/** 输入时实时匹配命令 */
function handleInput() {
  const input = command.value.trim()
  if (!input) {
    closeSuggestions()
    return
  }
  // 只在输入第一个单词（命令部分）时匹配
  const parts = input.split(/\s+/)
  if (parts.length > 1) {
    // 已有参数，不再提示命令
    closeSuggestions()
    return
  }
  const prefix = parts[0].toUpperCase()
  const matches = redisCommands.filter(c => c.startsWith(prefix) && c !== prefix)
  if (matches.length > 0) {
    suggestions.value = matches.slice(0, 10)
    suggestionIdx.value = 0
  } else {
    closeSuggestions()
  }
}

function handleTab() {
  const input = command.value.trim()
  if (!input) return

  // Extract the first word (command part)
  const parts = input.split(/\s+/)
  const prefix = parts[0].toUpperCase()

  if (suggestions.value.length && suggestionIdx.value >= 0) {
    applySuggestion(suggestions.value[suggestionIdx.value])
    return
  }

  const matches = redisCommands.filter(c => c.startsWith(prefix) && c !== prefix)
  if (matches.length === 1) {
    // Single match: apply directly
    parts[0] = matches[0]
    command.value = parts.join(' ') + (parts.length === 1 ? ' ' : '')
    closeSuggestions()
  } else if (matches.length > 1) {
    suggestions.value = matches.slice(0, 10)
    suggestionIdx.value = 0
  }
}

function applySuggestion(cmd: string) {
  const parts = command.value.trim().split(/\s+/)
  parts[0] = cmd
  command.value = parts.join(' ') + (parts.length === 1 ? ' ' : '')
  closeSuggestions()
  focusInput()
}

function closeSuggestions() {
  suggestions.value = []
  suggestionIdx.value = -1
}

function pushLine(type: CliLine['type'], text: string) {
  const lines = text.split('\n')
  const lineCount = lines.length
  history.value.push(reactive({
    type,
    text,
    collapsed: lineCount > collapseThreshold,
    lineCount,
  }))
}

function truncateText(text: string): string {
  const lines = text.split('\n')
  return lines.slice(0, collapseThreshold).join('\n') + '\n...'
}

/** Format Redis result for display */
function formatResult(raw: string): string {
  if (!raw) return '(nil)'
  const lines = raw.split('\n')
  // Detect numbered array pattern: "1) ...\n2) ..."
  const isNumberedArray = lines.length >= 2 && lines.every(l => !l || /^\d+\)\s/.test(l))
  if (isNumberedArray) {
    // Check if it's a hash-like result (even count, alternating key/value)
    if (lines.length >= 4 && lines.length % 2 === 0) {
      const isHashLike = lines.every((l, i) => {
        if (!l) return true
        const val = l.replace(/^\d+\)\s*/, '')
        // Even indices (0-based) are keys, odd are values
        return i % 2 === 0 ? !val.startsWith('"') || val.length < 100 : true
      })
      if (isHashLike) {
        const formatted: string[] = []
        for (let i = 0; i < lines.length; i += 2) {
          const key = lines[i].replace(/^\d+\)\s*/, '').replace(/^"|"$/g, '')
          const val = lines[i + 1]?.replace(/^\d+\)\s*/, '') || ''
          formatted.push(`  ${key} → ${val}`)
        }
        return formatted.join('\n')
      }
    }
    // Regular array: indent items
    return lines.map(l => l ? `  ${l}` : l).join('\n')
  }
  return raw
}

async function executeCommand() {
  const cmd = command.value.trim()
  if (!cmd) return

  pushLine('cmd', cmd)
  cmdHistory.value.unshift(cmd)
  cmdHistoryIdx.value = -1
  command.value = ''
  closeSuggestions()

  // Local commands
  const lower = cmd.toLowerCase()
  if (lower === 'clear' || lower === 'cls') {
    history.value = []
    return
  }

  try {
    const res = await request<{ result: string }>('execute_command', {
      params: { conn_id: props.connId, command: cmd },
    })
    const raw = res.data?.result || '(nil)'
    pushLine('result', formatResult(raw))
    // SELECT 命令成功后更新 CLI 独立的 db
    const selectMatch = cmd.match(/^\s*select\s+(\d+)\s*$/i)
    if (selectMatch && raw.replace(/^"|"$/g, '').toUpperCase() === 'OK') {
      const db = parseInt(selectMatch[1], 10)
      connectionStore.setCliDb(props.connId, db)
    }
  } catch (e: any) {
    pushLine('error', e.message || t('common.failed'))
  }

  scrollToBottom()
}

function historyUp() {
  if (cmdHistory.value.length === 0) return
  cmdHistoryIdx.value = Math.min(cmdHistoryIdx.value + 1, cmdHistory.value.length - 1)
  command.value = cmdHistory.value[cmdHistoryIdx.value]
}

function historyDown() {
  if (cmdHistoryIdx.value <= 0) {
    cmdHistoryIdx.value = -1
    command.value = ''
    return
  }
  cmdHistoryIdx.value--
  command.value = cmdHistory.value[cmdHistoryIdx.value]
}

onMounted(() => {
  const cache = connectionStore.getCliCache(props.connId)
  if (cache) {
    history.value = cache.history
    cmdHistory.value = cache.cmdHistory
  } else {
    // 插入欢迎信息
    const conn = connectionStore.connections.find(c => c.id === props.connId)
    const state = connectionStore.getConnState(props.connId)
    const connName = conn?.name || props.connId
    const host = conn?.host || '127.0.0.1'
    const port = conn?.port || 6379
    const version = state.redisVersion || 'unknown'
    const db = state.cliDb ?? 0
    const welcomeText = `Connected to ${connName} (${host}:${port})\nRedis version: ${version} | DB: ${db}\nType commands below. Use TAB for autocomplete.`
    history.value.unshift(reactive({
      type: 'result' as const,
      text: welcomeText,
      collapsed: false,
      lineCount: 3,
    }))
  }
  ;(inputRef.value as HTMLInputElement)?.focus()
  scrollToBottom()
})

onBeforeUnmount(() => {
  connectionStore.saveCliCache(props.connId, history.value, cmdHistory.value)
})
</script>

<style scoped>
.cli-view {
  height: 100%;
  overflow-y: auto;
  background: var(--color-bg-1);
  font-family: var(--font-family-mono);
  font-size: var(--app-cli-font-size, var(--font-size-sm));
}

.cli-output {
  padding: var(--spacing-sm) var(--spacing-md) 0;
}

.cli-line {
  line-height: 1.6;
  word-break: break-all;
}

.cli-line.error .cli-text {
  color: var(--color-error);
}

.cli-line.result .cli-text {
  color: var(--color-text-2);
}

.cli-prompt {
  color: var(--color-primary);
  user-select: none;
}

.cli-text {
  color: var(--color-text-1);
  margin: 0;
  font-family: inherit;
  font-size: inherit;
  white-space: pre-wrap;
  word-break: break-all;
}

.cli-result-block {
  display: inline;
}

.cli-toggle {
  display: inline-block;
  margin-left: var(--spacing-sm);
  font-size: var(--font-size-xs);
  vertical-align: middle;
}

.cli-input-row {
  display: flex;
  align-items: center;
  padding: var(--spacing-xs) var(--spacing-md);
}

.cli-input-wrap {
  flex: 1;
  position: relative;
}

.cli-input {
  width: 100%;
  background: none;
  border: none;
  outline: none;
  color: var(--color-text-1);
  font-family: var(--font-family-mono);
  font-size: var(--app-cli-font-size, var(--font-size-sm));
  line-height: 1.6;
}

</style>

<style>
/* Unscoped because suggestions are teleported to body */
.cli-suggestions {
  max-height: 200px;
  overflow-y: auto;
  background: var(--color-bg-popup);
  border: 1px solid var(--color-border-2);
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-md);
  z-index: 9000;
}

.cli-suggestion {
  padding: 3px 8px;
  font-size: var(--font-size-xs);
  font-family: var(--font-family-mono);
  color: var(--color-text-2);
  cursor: pointer;
}

.cli-suggestion:hover,
.cli-suggestion.active {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}
</style>
