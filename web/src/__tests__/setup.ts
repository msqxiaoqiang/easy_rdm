/** 测试环境初始化：mock $gm SDK */
import { vi } from 'vitest'

const mockGm = {
  token: 'test-token',
  data: null,
  fileId: 'test-file-id',
  name: 'easy_rdm',
  webURL: 'http://localhost',
  host: 'localhost',
  lang: 'zh-CN',
  themeCss: '',
  version: '1.0.0',
  communicationType: 'http',
  init: vi.fn().mockResolvedValue(undefined),
  request: vi.fn().mockResolvedValue({ data: { code: 200000, data: { code: 200, data: null, msg: 'OK' } } }),
  message: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
    loading: vi.fn(),
    destroyAll: vi.fn(),
  },
  dialog: {
    warning: vi.fn(),
    info: vi.fn(),
    success: vi.fn(),
    error: vi.fn(),
    create: vi.fn(),
    destroyAll: vi.fn(),
  },
  closeApp: vi.fn(),
  closeOtherApp: vi.fn(),
  chooseFolder: vi.fn(),
  chooseFile: vi.fn(),
  openFolder: vi.fn(),
  getRectSize: vi.fn().mockReturnValue({ width: 1024, height: 768 }),
  setAppRectStyle: vi.fn(),
  childRectListener: vi.fn(),
  childDestroyedListener: vi.fn(),
  execShell: vi.fn(),
  apiForward: vi.fn(),
}

// 挂载到 window
;(globalThis as any).$gm = mockGm
;(window as any).$gm = mockGm

export { mockGm }
