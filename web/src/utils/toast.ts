import { Message } from '@arco-design/web-vue'

/** 兼容旧接口：setToastRef 不再需要，保留空函数避免调用方报错 */
export function setToastRef(_instance: any) {
  // no-op: Arco Message 不需要组件实例
}

export function toast(message: string, type: 'success' | 'error' | 'warning' | 'info' = 'info', duration = 3000) {
  Message[type]({ content: message, duration })
}
