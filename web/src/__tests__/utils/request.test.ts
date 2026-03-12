import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mockGm } from '../setup'

// 直接测试 request 模块的逻辑
describe('request utility', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('GA 响应 code !== 200000 应抛出错误', async () => {
    mockGm.request.mockResolvedValueOnce({
      code: 500000, data: null, msg: 'Internal Error',
    })

    const { request } = await import('@/utils/request')

    await expect(request('test')).rejects.toThrow('Internal Error')
  })

  it('GA 响应成功应返回内层业务数据', async () => {
    mockGm.request.mockResolvedValueOnce({
      code: 200000,
      data: { code: 200, data: { key: 'value' }, msg: 'OK' },
      msg: 'Successful operation',
    })

    const { request } = await import('@/utils/request')
    const res = await request('test', { params: { foo: 'bar' } })

    expect(res.code).toBe(200)
    expect(res.data).toEqual({ key: 'value' })
  })

  it('$gm.request 应传对象参数 { url, method, data }', async () => {
    mockGm.request.mockResolvedValueOnce({
      code: 200000, data: { code: 200, data: null, msg: 'OK' },
    })

    const { request } = await import('@/utils/request')
    await request('hello', { params: { key: 'val' } })

    const callArgs = mockGm.request.mock.calls[0]
    // 对象格式调用
    expect(callArgs[0]).toMatchObject({
      url: expect.stringContaining('/hello'),
      method: 'POST',
      data: { params: { key: 'val' } },
    })
  })

  it('业务响应 code !== 200 应抛出错误（bug fix 验证）', async () => {
    mockGm.request.mockResolvedValueOnce({
      code: 200000,
      data: { code: 500, data: null, msg: '连接失败: NOAUTH Authentication required' },
      msg: 'Successful operation',
    })

    const { request } = await import('@/utils/request')

    await expect(request('test_connection')).rejects.toThrow('NOAUTH')
  })

  it('业务响应 code !== 200 且无 msg 应使用默认错误信息', async () => {
    mockGm.request.mockResolvedValueOnce({
      code: 200000,
      data: { code: 500, data: null, msg: '' },
      msg: 'Successful operation',
    })

    const { request } = await import('@/utils/request')

    await expect(request('test')).rejects.toThrow('操作失败')
  })

  it('params 默认为空对象', async () => {
    mockGm.request.mockResolvedValueOnce({
      code: 200000, data: { code: 200, data: null, msg: 'OK' },
    })

    const { request } = await import('@/utils/request')
    await request('no_params')

    const callArgs = mockGm.request.mock.calls[0]
    expect(callArgs[0].data.params).toEqual({})
  })
})
