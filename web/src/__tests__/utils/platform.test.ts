import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mockGm } from '../setup'
import {
  getPlatform,
  isGmssh,
  isWeb,
  isDesktop,
  hasNativeFileDialog,
  showMessage,
  request,
  BizError,
} from '@/utils/platform'

describe('platform.ts', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ========== 模式判断 ==========

  describe('模式判断', () => {
    it('getPlatform() 默认返回 gmssh', () => {
      expect(getPlatform()).toBe('gmssh')
    })

    it('isGmssh() 在 gmssh 模式返回 true', () => {
      expect(isGmssh()).toBe(true)
    })

    it('isWeb() 在 gmssh 模式返回 false', () => {
      expect(isWeb()).toBe(false)
    })

    it('isDesktop() 在 gmssh 模式返回 false', () => {
      expect(isDesktop()).toBe(false)
    })

    it('hasNativeFileDialog() 在 gmssh 模式返回 true', () => {
      expect(hasNativeFileDialog()).toBe(true)
    })
  })

  // ========== showMessage ==========

  describe('showMessage (gmssh 模式)', () => {
    it('success 类型调用 $gm.message.success', () => {
      showMessage('success', '操作成功')
      expect(mockGm.message.success).toHaveBeenCalledWith('操作成功')
    })

    it('error 类型调用 $gm.message.error', () => {
      showMessage('error', '操作失败')
      expect(mockGm.message.error).toHaveBeenCalledWith('操作失败')
    })

    it('warning 类型调用 $gm.message.warning', () => {
      showMessage('warning', '请注意')
      expect(mockGm.message.warning).toHaveBeenCalledWith('请注意')
    })

    it('info 类型调用 $gm.message.info', () => {
      showMessage('info', '提示信息')
      expect(mockGm.message.info).toHaveBeenCalledWith('提示信息')
    })
  })

  // ========== request ==========

  describe('request (gmssh 模式)', () => {
    beforeEach(() => {
      vi.clearAllMocks()
    })

    it('成功请求：GA 外层 code=200000，内层 code=200，返回 bizData', async () => {
      mockGm.request.mockResolvedValueOnce({
        code: 200000,
        data: { code: 200, data: { key: 'value' }, msg: 'OK' },
        msg: 'Successful operation',
      })

      const res = await request('test_method')

      expect(res.code).toBe(200)
      expect(res.data).toEqual({ key: 'value' })
      expect(res.msg).toBe('OK')
    })

    it('GA 外层失败：code !== 200000 抛出 Error', async () => {
      mockGm.request.mockResolvedValueOnce({
        code: 500000,
        data: null,
        msg: 'Internal Error',
      })

      await expect(request('test_method')).rejects.toThrow('Internal Error')
    })

    it('业务失败：内层 code !== 200 抛出 BizError', async () => {
      mockGm.request.mockResolvedValueOnce({
        code: 200000,
        data: { code: 500, data: { detail: 'err' }, msg: '连接失败' },
        msg: 'Successful operation',
      })

      try {
        await request('test_method')
        expect.unreachable('应该抛出异常')
      } catch (err) {
        expect(err).toBeInstanceOf(BizError)
        const bizErr = err as BizError
        expect(bizErr.code).toBe(500)
        expect(bizErr.message).toBe('连接失败')
        expect(bizErr.data).toEqual({ detail: 'err' })
      }
    })

    it('请求参数格式正确：url 包含 /api/call/jm/easy_rdm/{path}，method 为 POST', async () => {
      mockGm.request.mockResolvedValueOnce({
        code: 200000,
        data: { code: 200, data: null, msg: 'OK' },
      })

      await request('some_path', { params: { foo: 'bar' } })

      const callArgs = mockGm.request.mock.calls[0][0]
      expect(callArgs.url).toBe('/api/call/jm/easy_rdm/some_path')
      expect(callArgs.method).toBe('POST')
      expect(callArgs.data).toEqual({ params: { foo: 'bar' } })
    })

    it('无 params 时默认空对象', async () => {
      mockGm.request.mockResolvedValueOnce({
        code: 200000,
        data: { code: 200, data: null, msg: 'OK' },
      })

      await request('no_params')

      const callArgs = mockGm.request.mock.calls[0][0]
      expect(callArgs.data.params).toEqual({})
    })
  })

  // ========== BizError ==========

  describe('BizError', () => {
    it('继承 Error', () => {
      const err = new BizError(500, '业务错误')
      expect(err).toBeInstanceOf(Error)
      expect(err).toBeInstanceOf(BizError)
    })

    it('有 code、message、data 属性', () => {
      const err = new BizError(403, '无权限', { role: 'guest' })
      expect(err.code).toBe(403)
      expect(err.message).toBe('无权限')
      expect(err.data).toEqual({ role: 'guest' })
    })

    it('data 可选，默认 undefined', () => {
      const err = new BizError(500, '服务异常')
      expect(err.code).toBe(500)
      expect(err.message).toBe('服务异常')
      expect(err.data).toBeUndefined()
    })
  })
})
