import { reactive } from 'vue'

/** 快捷键动作定义 */
export interface ShortcutAction {
  id: string
  labelKey: string // i18n key
  defaultBinding: string // e.g. "Ctrl+N", "Cmd+S"
}

/** 所有可自定义的快捷键动作 */
export const shortcutActions: ShortcutAction[] = [
  { id: 'newKey', labelKey: 'shortcut.newKey', defaultBinding: 'Ctrl+N' },
  { id: 'refresh', labelKey: 'shortcut.refresh', defaultBinding: 'Ctrl+R' },
  { id: 'deleteKey', labelKey: 'shortcut.deleteKey', defaultBinding: 'Delete' },
  { id: 'save', labelKey: 'shortcut.save', defaultBinding: 'Ctrl+S' },
  { id: 'search', labelKey: 'shortcut.search', defaultBinding: 'Ctrl+F' },
  { id: 'closeTab', labelKey: 'shortcut.closeTab', defaultBinding: 'Ctrl+W' },
  { id: 'settings', labelKey: 'shortcut.settings', defaultBinding: 'Ctrl+,' },
  { id: 'nextTab', labelKey: 'shortcut.nextTab', defaultBinding: 'Ctrl+Tab' },
  { id: 'prevTab', labelKey: 'shortcut.prevTab', defaultBinding: 'Ctrl+Shift+Tab' },
]

/** 当前快捷键绑定（action id → binding string） */
const bindings = reactive<Record<string, string>>({})

// 初始化默认绑定
for (const action of shortcutActions) {
  bindings[action.id] = action.defaultBinding
}

/** 加载自定义绑定（从 settings 中读取） */
export function loadShortcutBindings(saved: Record<string, string> | undefined) {
  if (!saved) return
  for (const [id, binding] of Object.entries(saved)) {
    if (shortcutActions.some(a => a.id === id) && binding) {
      bindings[id] = binding
    }
  }
}

/** 重置为默认绑定 */
export function resetShortcutBindings() {
  for (const action of shortcutActions) {
    bindings[action.id] = action.defaultBinding
  }
}

/** 获取当前绑定 */
export function getBindings(): Record<string, string> {
  return { ...bindings }
}

/** 设置单个绑定 */
export function setBinding(actionId: string, binding: string) {
  bindings[actionId] = binding
}

/** 将 KeyboardEvent 转换为绑定字符串 */
export function eventToBinding(e: KeyboardEvent): string {
  const parts: string[] = []
  if (e.ctrlKey || e.metaKey) parts.push('Ctrl')
  if (e.shiftKey) parts.push('Shift')
  if (e.altKey) parts.push('Alt')

  const key = e.key
  // 忽略单独的修饰键
  if (['Control', 'Meta', 'Shift', 'Alt'].includes(key)) return ''

  // 标准化 key 名称
  if (key === ' ') parts.push('Space')
  else if (key === 'Tab') parts.push('Tab')
  else if (key === 'Escape') parts.push('Escape')
  else if (key === 'Delete') parts.push('Delete')
  else if (key === 'Backspace') parts.push('Backspace')
  else if (key === 'Enter') parts.push('Enter')
  else if (key.startsWith('Arrow')) parts.push(key)
  else if (key.startsWith('F') && key.length <= 3) parts.push(key) // F1-F12
  else if (key === ',') parts.push(',')
  else parts.push(key.toUpperCase())

  return parts.join('+')
}

/** 检查 KeyboardEvent 是否匹配指定动作 */
export function matchesAction(e: KeyboardEvent, actionId: string): boolean {
  const binding = bindings[actionId]
  if (!binding) return false
  const eventBinding = eventToBinding(e)
  return eventBinding === binding
}

/** 检测冲突：返回与给定 binding 冲突的 action id（排除自身） */
export function detectConflict(binding: string, excludeId: string): string | null {
  for (const [id, b] of Object.entries(bindings)) {
    if (id !== excludeId && b === binding) return id
  }
  return null
}

/** 格式化绑定字符串用于显示（macOS 用符号） */
export function formatBinding(binding: string): string {
  const isMac = navigator.platform.includes('Mac')
  if (!isMac) return binding
  return binding
    .replace(/Ctrl\+/g, '⌘')
    .replace(/Shift\+/g, '⇧')
    .replace(/Alt\+/g, '⌥')
}
