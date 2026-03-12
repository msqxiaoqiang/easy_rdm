/**
 * 平台适配层 — 统一封装三端差异化 API
 *
 * 根据 VITE_PLATFORM 环境变量（gmssh / desktop / web）自动选择实现。
 * 所有组件代码只调用本模块导出的函数，不直接使用 $gm。
 */

import { Message as arcoMessage } from '@arco-design/web-vue'

export type Platform = 'gmssh' | 'desktop' | 'web'

/** 当前运行平台 */
export function getPlatform(): Platform {
  return (import.meta.env.VITE_PLATFORM as Platform) || 'gmssh'
}

export function isGmssh(): boolean { return getPlatform() === 'gmssh' }
export function isDesktop(): boolean { return getPlatform() === 'desktop' }
export function isWeb(): boolean { return getPlatform() === 'web' }

/**
 * 是否需要原生文件对话框（gmssh / desktop 有，web 没有）
 * 用于替代原来的 `$gm._isMock` 判断
 */
export function hasNativeFileDialog(): boolean {
  return isGmssh() || isDesktop()
}

// ========== 消息提示 ==========

export type MessageType = 'success' | 'error' | 'warning' | 'info'

/**
 * 显示消息提示
 * - gmssh: $gm.message[type]
 * - desktop / web: Arco Design Message 组件
 */
export function showMessage(type: MessageType, text: string): void {
  if (isGmssh()) {
    ;(window as any).$gm?.message?.[type]?.(text)
  } else {
    // desktop / web 模式：使用 Arco Design Message
    arcoMessage[type]?.(text)
  }
}

// ========== 请求 ==========

import i18n from '../i18n'

interface BizResponse<T = any> {
  code: number
  data: T
  msg: string
}

/** 业务错误，携带后端返回的 code 和 data */
export class BizError extends Error {
  code: number
  data: any
  constructor(code: number, msg: string, data?: any) {
    super(msg)
    this.code = code
    this.data = data
  }
}

interface RequestOptions {
  params?: Record<string, any>
}

const APP_ORG = 'jm'
const APP_NAME = 'easy_rdm'

function buildGmsshUrl(path: string): string {
  return `/api/call/${APP_ORG}/${APP_NAME}/${path}`
}

/**
 * 统一请求函数
 * - gmssh: $gm.request → GA 双层响应
 * - desktop: window.go.main.App.Call(method, paramsJSON) → 直接 BizResponse
 * - web: fetch('/api/{method}') → 直接 BizResponse
 */
export async function request<T = any>(
  path: string,
  options: RequestOptions = {},
): Promise<BizResponse<T>> {
  const platform = getPlatform()

  if (platform === 'gmssh') {
    return requestGmssh<T>(path, options)
  } else if (platform === 'desktop') {
    return requestDesktop<T>(path, options)
  } else {
    return requestWeb<T>(path, options)
  }
}

/** GMSSH 模式：通过 $gm.request → GA 双层响应 */
async function requestGmssh<T>(path: string, options: RequestOptions): Promise<BizResponse<T>> {
  const gm = (window as any).$gm
  if (!gm?.request) {
    throw new Error('GMSSH SDK 未初始化')
  }
  const url = buildGmsshUrl(path)
  const res: any = await gm.request({
    url,
    method: 'POST',
    data: { params: options.params ?? {} },
  })

  // GA 外层响应 code === 200000 表示转发成功
  if (!res || res.code !== 200000) {
    throw new Error(res?.msg || i18n.global.t('common.failed'))
  }

  // 内层为插件业务响应
  const bizData = res.data as BizResponse<T>
  if (bizData.code !== 200) {
    throw new BizError(bizData.code, bizData.msg || i18n.global.t('common.failed'), bizData.data)
  }
  return bizData
}

/** Desktop 模式：通过 Wails binding 直接调用 */
async function requestDesktop<T>(path: string, options: RequestOptions): Promise<BizResponse<T>> {
  const paramsJSON = JSON.stringify(options.params ?? {})
  const resultJSON: string = await (window as any).go.main.App.Call(path, paramsJSON)
  let bizData: BizResponse<T>
  try {
    bizData = JSON.parse(resultJSON) as BizResponse<T>
  } catch {
    throw new Error(`响应解析失败: ${resultJSON?.substring(0, 100)}`)
  }
  if (bizData.code !== 200) {
    throw new BizError(bizData.code, bizData.msg || i18n.global.t('common.failed'), bizData.data)
  }
  return bizData
}

/** Web 模式：通过 fetch 直连后端 HTTP */
async function requestWeb<T>(path: string, options: RequestOptions): Promise<BizResponse<T>> {
  const baseUrl = import.meta.env.DEV ? '/dev-api' : '/api'
  const res = await fetch(`${baseUrl}/${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options.params ?? {}),
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(`HTTP ${res.status}: ${text || 'Request failed'}`)
  }
  const bizData = await res.json() as BizResponse<T>
  if (bizData.code !== 200) {
    throw new BizError(bizData.code, bizData.msg || i18n.global.t('common.failed'), bizData.data)
  }
  return bizData
}

// ========== 文件/目录选择 ==========

/**
 * 选择文件
 * - gmssh: $gm.chooseFile
 * - desktop: Wails OpenFileDialog binding
 * - web: <input type="file">
 */
export function chooseFile(
  callback: (filePath: string) => void,
  options?: { accept?: string; defaultPath?: string },
): void {
  const platform = getPlatform()

  if (platform === 'gmssh') {
    ;(window as any).$gm?.chooseFile?.((path: string) => {
      if (path) callback(path)
    }, options?.defaultPath || '')
  } else if (platform === 'desktop') {
    ;(window as any).go?.main?.App?.ChooseFile?.('', options?.accept || '')
      .then((path: string) => { if (path) callback(path) })
      .catch(() => {})
  } else {
    // web 模式：<input type="file">
    const input = document.createElement('input')
    input.type = 'file'
    if (options?.accept) input.accept = options.accept
    input.onchange = () => {
      const file = input.files?.[0]
      if (file) callback(file.name)
    }
    input.click()
  }
}

/**
 * 选择文件并获取 File 对象（web 模式专用，其他模式返回路径）
 * 返回 { path, file } — gmssh/desktop 只有 path，web 只有 file
 */
export function chooseFileWithData(
  callback: (result: { path?: string; file?: File }) => void,
  options?: { accept?: string; defaultPath?: string },
): void {
  const platform = getPlatform()

  if (platform === 'gmssh') {
    ;(window as any).$gm?.chooseFile?.((path: string) => {
      if (path) callback({ path })
    }, options?.defaultPath || '')
  } else if (platform === 'desktop') {
    ;(window as any).go?.main?.App?.ChooseFile?.('', options?.accept || '')
      .then((path: string) => { if (path) callback({ path }) })
      .catch(() => {})
  } else {
    const input = document.createElement('input')
    input.type = 'file'
    if (options?.accept) input.accept = options.accept
    input.onchange = () => {
      const file = input.files?.[0]
      if (file) callback({ file })
    }
    input.click()
  }
}

/**
 * 选择目录
 * - gmssh: $gm.chooseFolder
 * - desktop: Wails OpenDirectoryDialog binding
 * - web: 不支持（返回空）
 */
export function chooseFolder(
  callback: (folderPath: string) => void,
  defaultPath?: string,
): void {
  const platform = getPlatform()

  if (platform === 'gmssh') {
    ;(window as any).$gm?.chooseFolder?.((path: string) => {
      if (path) callback(path)
    }, defaultPath || '')
  } else if (platform === 'desktop') {
    ;(window as any).go?.main?.App?.ChooseFolder?.('')
      .then((path: string) => { if (path) callback(path) })
      .catch(() => {})
  }
  // web 模式不支持目录选择
}

// ========== HTTP 端点（web/desktop 用的文件上传下载） ==========

/**
 * 获取 HTTP 端点基础 URL
 * - 开发环境用 vite proxy /dev-api
 * - 生产环境 web 模式直接用 /api
 */
export function getHttpBaseUrl(): string {
  if (import.meta.env.DEV) return '/dev-api'
  return '/api'
}
