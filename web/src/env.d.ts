/// <reference types="vite/client" />

interface GMRequestOptions {
  url: string
  method?: string
  data?: any
}

interface GMMessage {
  success: (msg: string) => void
  error: (msg: string) => void
  warning: (msg: string) => void
  info: (msg: string) => void
}

interface GMDialog {
  warning: (opts: any) => void
  info: (opts: any) => void
  success: (opts: any) => void
  error: (opts: any) => void
  create: (opts: any) => void
  destroyAll: () => void
}

interface GMProps {
  communicationType: string
  fileId?: string
  name?: string
  version?: string
  themeCss?: string
  request: (opts: GMRequestOptions) => Promise<any>
  init: () => Promise<any>
  message: GMMessage
  dialog: GMDialog
  closeApp: () => void
  emitParent: (msg: any) => void
  getRectSize: () => { width: number; height: number }
  setAppRectStyle: (style?: any) => void
  childRectListener: (cb: (size: { width: number; height: number }) => void) => () => void
  mainGMCListener: (cb: (payload: any) => void) => void
  serveActiveListener: (cb: (payload: any) => void) => void
  extAppStatusListener?: (cb: (payload: any) => void) => void
}

declare global {
  interface Window {
    $gm: GMProps
  }
  const $gm: GMProps
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

export {}
