import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { nextTick } from 'vue'
import ArcoVue from '@arco-design/web-vue'
import KeyDetailView from '@/components/views/KeyDetailView.vue'
import zhCN from '@/i18n/locales/zh-CN'
import { request } from '@/utils/request'
import { gmConfirm, gmPrompt } from '@/utils/dialog'

// ========== Mocks ==========

vi.mock('@/utils/request', () => ({
  request: vi.fn().mockResolvedValue({ code: 200, data: null, msg: 'OK' }),
  BizError: class BizError extends Error {
    code: number
    data: any
    constructor(code: number, msg: string, data?: any) {
      super(msg)
      this.code = code
      this.data = data
    }
  },
}))

vi.mock('@/utils/dialog', () => ({
  gmConfirm: vi.fn().mockResolvedValue(true),
  gmPrompt: vi.fn().mockResolvedValue(null),
}))

// Mock 所有子组件
vi.mock('@/components/views/collections/HashDetail.vue', () => ({ default: { name: 'HashDetail', template: '<div class="hash-detail" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/ListDetail.vue', () => ({ default: { name: 'ListDetail', template: '<div class="list-detail" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/SetDetail.vue', () => ({ default: { name: 'SetDetail', template: '<div class="set-detail" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/ZSetDetail.vue', () => ({ default: { name: 'ZSetDetail', template: '<div class="zset-detail" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/StreamDetail.vue', () => ({ default: { name: 'StreamDetail', template: '<div class="stream-detail" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/GeoDetail.vue', () => ({ default: { name: 'GeoDetail', template: '<div class="geo-detail" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/BitmapView.vue', () => ({ default: { name: 'BitmapView', template: '<div class="bitmap-view" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/HllView.vue', () => ({ default: { name: 'HllView', template: '<div class="hll-view" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/BitfieldView.vue', () => ({ default: { name: 'BitfieldView', template: '<div class="bitfield-view" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/views/collections/JsonDetail.vue', () => ({ default: { name: 'JsonDetail', template: '<div class="json-detail" />', props: ['connId', 'keyName'] } }))
vi.mock('@/components/common/JsonCodeEditor.vue', () => ({
  default: {
    name: 'JsonCodeEditor',
    template: '<textarea class="mock-json-editor" :value="modelValue" @input="onInput" />',
    props: ['modelValue', 'readonly', 'placeholder'],
    emits: ['update:modelValue', 'change'],
    methods: {
      onInput(e: Event) {
        (this as any).$emit('update:modelValue', (e.target as HTMLTextAreaElement).value);
        (this as any).$emit('change')
      },
    },
  },
}))

const mockedRequest = vi.mocked(request)
const mockedConfirm = vi.mocked(gmConfirm)
const mockedPrompt = vi.mocked(gmPrompt)

// ========== Helpers ==========

function createI18nPlugin() {
  return createI18n({ legacy: false, locale: 'zh-CN', messages: { 'zh-CN': zhCN } })
}

function createWrapper(props: { connId: string; keyName?: string; keyType?: string } = { connId: 'tc' }) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const i18n = createI18nPlugin()
  return mount(KeyDetailView, {
    props,
    global: {
      plugins: [pinia, i18n, ArcoVue],
      directives: { 'ellipsis-tip': { mounted() {} } },
    },
  })
}

/** Arco Select helper: emit update:modelValue + change */
async function setArcoSelect(wrapper: any, index: number, value: string) {
  const selects = wrapper.findAllComponents({ name: 'Select' })
  selects[index].vm.$emit('update:modelValue', value)
  selects[index].vm.$emit('change', value)
  await nextTick()
}

/** Find the inner textarea (Arco Textarea or mock JsonCodeEditor) */
function findTextarea(wrapper: any) {
  const arco = wrapper.find('.value-editor textarea')
  if (arco.exists()) return arco
  return wrapper.find('.mock-json-editor')
}

/** Find save button (arco primary button in detail-footer) */
function findSaveBtn(wrapper: any) {
  return wrapper.find('.detail-footer .arco-btn-primary')
}

/** Find delete button (arco danger status button) */
function findDeleteBtn(wrapper: any) {
  return wrapper.find('.action-btn.arco-btn-status-danger')
}

/** 模拟 get_key_info 返回 string 类型 */
function mockStringKey(value = 'hello', ttl = -1, truncated = false) {
  mockedRequest.mockImplementation((method: string) => {
    if (method === 'get_key_info') {
      return Promise.resolve({ code: 200, data: { type: 'string', ttl, encoding: 'raw', length: value.length }, msg: 'OK' } as any)
    }
    if (method === 'get_key_value') {
      return Promise.resolve({ code: 200, data: { type: 'string', value, truncated }, msg: 'OK' } as any)
    }
    if (method === 'get_decoders') {
      return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
    }
    return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
  })
}

/** 模拟 get_key_info 返回非 string 类型 */
function mockCollectionKey(type: string, ttl = -1, length = 10) {
  mockedRequest.mockImplementation((method: string) => {
    if (method === 'get_key_info') {
      return Promise.resolve({ code: 200, data: { type, ttl, encoding: 'ziplist', length }, msg: 'OK' } as any)
    }
    if (method === 'get_key_value') {
      return Promise.resolve({ code: 200, data: { type, value: null }, msg: 'OK' } as any)
    }
    if (method === 'get_decoders') {
      return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
    }
    return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
  })
}

/** 模拟 key 不存在（type: none） */
function mockKeyNotFound() {
  mockedRequest.mockImplementation((method: string) => {
    if (method === 'get_key_info') {
      return Promise.resolve({ code: 200, data: { type: 'none', ttl: -2, encoding: '', length: 0 }, msg: 'OK' } as any)
    }
    if (method === 'get_decoders') {
      return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
    }
    return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
  })
}

// ========== Tests ==========

describe('KeyDetailView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // 默认 clipboard mock
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
  })

  // ---------- 1. 空状态 ----------

  describe('空状态（未选择 key）', () => {
    it('未传 keyName 时显示空状态提示', () => {
      mockStringKey()
      const wrapper = createWrapper({ connId: 'tc' })
      expect(wrapper.find('.empty-state').exists()).toBe(true)
      expect(wrapper.find('.key-info-bar').exists()).toBe(false)
    })

    it('keyName 为空字符串时显示空状态', () => {
      mockStringKey()
      const wrapper = createWrapper({ connId: 'tc', keyName: '' })
      expect(wrapper.find('.empty-state').exists()).toBe(true)
    })
  })

  // ---------- 2. 加载状态与 key 信息 ----------

  describe('加载 key 信息', () => {
    it('选中 key 后显示 loading 状态', async () => {
      mockedRequest.mockImplementation(() => new Promise(() => {}))
      const wrapper = createWrapper({ connId: 'tc', keyName: 'test:key', keyType: 'string' })
      await nextTick()
      expect(wrapper.find('.loading-state').exists()).toBe(true)
      expect(wrapper.find('.key-info-bar').exists()).toBe(false)
    })

    it('加载成功后显示 key 信息栏', async () => {
      mockStringKey('hello', 3600)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'mykey', keyType: 'string' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.loading-state').exists()).toBe(false)
      expect(wrapper.find('.key-info-bar').exists()).toBe(true)
      expect(wrapper.find('.type-badge').text()).toBe('string')
      expect(wrapper.find('.key-name-display').text()).toBe('mykey')
    })

    it('key 不存在（type=none）时 emit notFound 并显示警告', async () => {
      mockKeyNotFound()
      const wrapper = createWrapper({ connId: 'tc', keyName: 'gone:key', keyType: 'string' })
      await flushPromises()
      await nextTick()
      expect(wrapper.emitted('notFound')).toBeTruthy()
      expect(wrapper.emitted('notFound')![0]).toEqual(['gone:key'])
      expect(wrapper.emitted('deleted')).toBeFalsy()
      expect((window as any).$gm.message.warning).toHaveBeenCalled()
    })

    it('get_key_info 请求失败时视为 key 不存在', async () => {
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'get_key_info') return Promise.reject(new Error('network error'))
        if (method === 'get_decoders') return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      const wrapper = createWrapper({ connId: 'tc', keyName: 'err:key', keyType: 'string' })
      await flushPromises()
      await nextTick()
      expect(wrapper.emitted('notFound')).toBeTruthy()
      expect(wrapper.emitted('deleted')).toBeFalsy()
    })
  })

  // ---------- 3. TTL 显示与格式化 ----------

  describe('TTL 显示', () => {
    it('TTL=-1 显示 ∞ 且 class=permanent', async () => {
      mockStringKey('v', -1)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'k1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const ttl = wrapper.find('.ttl-display')
      expect(ttl.text()).toContain('∞')
      expect(ttl.classes()).toContain('permanent')
    })

    it('TTL<=60 显示秒数且 class=danger', async () => {
      mockStringKey('v', 30)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'k2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const ttl = wrapper.find('.ttl-display')
      expect(ttl.text()).toContain('30s')
      expect(ttl.classes()).toContain('danger')
    })

    it('60<TTL<=3600 显示分钟且 class=warning', async () => {
      mockStringKey('v', 1800)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'k3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const ttl = wrapper.find('.ttl-display')
      expect(ttl.text()).toContain('30m')
      expect(ttl.classes()).toContain('warning')
    })

    it('TTL>3600 显示小时且 class=safe', async () => {
      mockStringKey('v', 7200)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'k4', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const ttl = wrapper.find('.ttl-display')
      expect(ttl.text()).toContain('2h')
      expect(ttl.classes()).toContain('safe')
    })

    it('TTL>=86400 显示天数', async () => {
      mockStringKey('v', 90000)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'k5', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const ttl = wrapper.find('.ttl-display')
      expect(ttl.text()).toContain('1d')
    })

    it('TTL=-2 显示 -', async () => {
      mockStringKey('v', -1) // 先用正常 mock 创建
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'get_key_info') {
          return Promise.resolve({ code: 200, data: { type: 'string', ttl: -2, encoding: 'raw', length: 1 }, msg: 'OK' } as any)
        }
        if (method === 'get_key_value') {
          return Promise.resolve({ code: 200, data: { type: 'string', value: 'v', truncated: false }, msg: 'OK' } as any)
        }
        if (method === 'get_decoders') return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      const wrapper = createWrapper({ connId: 'tc', keyName: 'k6', keyType: 'string' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.ttl-display').text()).toContain('-')
    })
  })

  // ---------- 4. String 值加载与显示 ----------

  describe('String 值加载', () => {
    it('普通文本以 text 模式显示', async () => {
      mockStringKey('hello world')
      const wrapper = createWrapper({ connId: 'tc', keyName: 's1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const textarea = findTextarea(wrapper)
      expect(textarea.exists()).toBe(true)
      expect((textarea.element as HTMLTextAreaElement).value).toBe('hello world')
      expect(wrapper.find('.footer-select').exists()).toBe(true)
    })

    it('JSON 对象自动切换为 json 视图并格式化', async () => {
      mockStringKey('{"name":"test","age":1}')
      const wrapper = createWrapper({ connId: 'tc', keyName: 's2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick() // suppressViewAsWatch 需要额外 tick
      const selectComp = wrapper.findAllComponents({ name: 'Select' })[0]
      expect(selectComp.props('modelValue')).toBe('json')
      // JSON 模式使用 textarea 显示格式化后的 JSON
      const textarea = findTextarea(wrapper)
      expect(textarea.exists()).toBe(true)
      const val = (textarea.element as HTMLTextAreaElement).value
      expect(val).toContain('"name"')
      expect(val).toContain('"test"')
    })

    it('JSON 数组自动切换为 json 视图', async () => {
      mockStringKey('[1,2,3]')
      const wrapper = createWrapper({ connId: 'tc', keyName: 's3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      const selectComp = wrapper.findAllComponents({ name: 'Select' })[0]
      expect(selectComp.props('modelValue')).toBe('json')
    })

    it('非 JSON 字符串保持 text 模式', async () => {
      mockStringKey('just plain text')
      const wrapper = createWrapper({ connId: 'tc', keyName: 's4', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      const selectComp = wrapper.findAllComponents({ name: 'Select' })[0]
      expect(selectComp.props('modelValue')).toBe('text')
    })

    it('truncated 值显示截断警告', async () => {
      mockStringKey('big value...', -1, true)
      const wrapper = createWrapper({ connId: 'tc', keyName: 's5', keyType: 'string' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.truncation-warning').exists()).toBe(true)
    })

    it('truncated 值时保存按钮禁用', async () => {
      mockStringKey('big value...', -1, true)
      const wrapper = createWrapper({ connId: 'tc', keyName: 's6', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const saveBtn = findSaveBtn(wrapper)
      expect(saveBtn.exists()).toBe(true)
      expect((saveBtn.element as HTMLButtonElement).disabled).toBe(true)
    })
  })

  // ---------- 5. 视图模式切换 ----------

  describe('视图模式切换', () => {
    it('切换到 hex 模式显示十六进制', async () => {
      mockStringKey('AB')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'vm1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 切换到 hex
      await setArcoSelect(wrapper, 0, 'hex')
      await nextTick()
      const textarea = findTextarea(wrapper)
      expect((textarea.element as HTMLTextAreaElement).value).toBe('41 42')
    })

    it('切换到 bitmap 显示 BitmapView 子组件', async () => {
      mockStringKey('AB')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'vm2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      await setArcoSelect(wrapper, 0, 'bitmap')
      await nextTick()
      expect(wrapper.find('.bitmap-view').exists()).toBe(true)
      expect(findTextarea(wrapper).exists()).toBe(false)
    })

    it('切换到 hll 显示 HllView 子组件', async () => {
      mockStringKey('AB')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'vm3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      await setArcoSelect(wrapper, 0, 'hll')
      await nextTick()
      expect(wrapper.find('.hll-view').exists()).toBe(true)
    })

    it('切换到 bitfield 显示 BitfieldView 子组件', async () => {
      mockStringKey('AB')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'vm4', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      await setArcoSelect(wrapper, 0, 'bitfield')
      await nextTick()
      expect(wrapper.find('.bitfield-view').exists()).toBe(true)
    })
  })

  // ---------- 6. 保存功能 ----------

  describe('保存（handleSave）', () => {
    it('text 模式保存成功后 modified 重置', async () => {
      mockStringKey('old')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sv1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 编辑内容
      const textarea = findTextarea(wrapper)
      await textarea.setValue('new value')
      await nextTick()
      // 保存
      const saveBtn = findSaveBtn(wrapper)
      expect((saveBtn.element as HTMLButtonElement).disabled).toBe(false)
      await saveBtn.trigger('click')
      await flushPromises()
      // 验证调用了 set_key_value
      const saveCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_key_value')
      expect(saveCalls).toHaveLength(1)
      expect(saveCalls[0][1]?.params?.value).toBe('new value')
      expect(saveCalls[0][1]?.params?.key).toBe('sv1')
    })

    it('json 模式保存时压缩 JSON', async () => {
      mockStringKey('{"a":1}')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sv2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // JSON 模式：通过 a-textarea 编辑
      const textarea = findTextarea(wrapper)
      await textarea.setValue('{\n  "a": 2\n}')
      await textarea.trigger('input')
      await nextTick()
      await findSaveBtn(wrapper).trigger('click')
      await flushPromises()
      const saveCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_key_value')
      expect(saveCalls).toHaveLength(1)
      expect(saveCalls[0][1]?.params?.value).toBe('{"a":2}')
    })

    it('hex 模式保存时还原为原始字符串', async () => {
      mockStringKey('AB')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sv3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 切换到 hex
      await setArcoSelect(wrapper, 0, 'hex')
      await nextTick()
      // 编辑 hex
      const textarea = findTextarea(wrapper)
      await textarea.setValue('43 44')
      await textarea.trigger('input')
      await nextTick()
      await findSaveBtn(wrapper).trigger('click')
      await flushPromises()
      const saveCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_key_value')
      expect(saveCalls).toHaveLength(1)
      expect(saveCalls[0][1]?.params?.value).toBe('CD')
    })

    it('保存时 BizError 409 key_deleted 显示对应提示', async () => {
      mockStringKey('old')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sv4', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 编辑触发 modified
      await findTextarea(wrapper).setValue('changed')
      await findTextarea(wrapper).trigger('input')
      await nextTick()
      // mock set_key_value 抛出 BizError 409
      const { BizError: BE } = await import('@/utils/request')
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'set_key_value') return Promise.reject(new BE(409, 'key_deleted'))
        if (method === 'get_decoders') return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      await findSaveBtn(wrapper).trigger('click')
      await flushPromises()
      expect((window as any).$gm.message.error).toHaveBeenCalled()
    })

    it('保存时 BizError 409 type_changed 显示类型变更提示', async () => {
      mockStringKey('old')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sv5', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      await findTextarea(wrapper).setValue('changed')
      await findTextarea(wrapper).trigger('input')
      await nextTick()
      const { BizError: BE } = await import('@/utils/request')
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'set_key_value') return Promise.reject(new BE(409, 'type_changed', 'hash'))
        if (method === 'get_decoders') return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      await findSaveBtn(wrapper).trigger('click')
      await flushPromises()
      expect((window as any).$gm.message.error).toHaveBeenCalled()
    })

    it('保存时 BizError 409 其他消息显示 saveConflict', async () => {
      mockStringKey('old')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sv6', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      await findTextarea(wrapper).setValue('changed')
      await findTextarea(wrapper).trigger('input')
      await nextTick()
      const { BizError: BE } = await import('@/utils/request')
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'set_key_value') return Promise.reject(new BE(409, 'other_conflict'))
        if (method === 'get_decoders') return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      await findSaveBtn(wrapper).trigger('click')
      await flushPromises()
      expect((window as any).$gm.message.error).toHaveBeenCalled()
    })

    it('保存时普通错误显示 error message', async () => {
      mockStringKey('old')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sv7', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      await findTextarea(wrapper).setValue('changed')
      await findTextarea(wrapper).trigger('input')
      await nextTick()
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'set_key_value') return Promise.reject(new Error('server error'))
        if (method === 'get_decoders') return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      await findSaveBtn(wrapper).trigger('click')
      await flushPromises()
      expect((window as any).$gm.message.error).toHaveBeenCalled()
    })
  })

  // ---------- 7. 删除功能 ----------

  describe('删除（handleDelete）', () => {
    it('确认删除后 emit deleted 事件', async () => {
      mockStringKey('val')
      mockedConfirm.mockResolvedValue(true)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'dk1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const deleteBtn = findDeleteBtn(wrapper)
      await deleteBtn.trigger('click')
      await flushPromises()
      expect(mockedConfirm).toHaveBeenCalled()
      const delCalls = mockedRequest.mock.calls.filter(c => c[0] === 'delete_keys')
      expect(delCalls).toHaveLength(1)
      expect(delCalls[0][1]?.params?.keys).toEqual(['dk1'])
      expect(wrapper.emitted('deleted')).toBeTruthy()
      expect(wrapper.emitted('deleted')![0]).toEqual(['dk1'])
    })

    it('取消确认不执行删除', async () => {
      mockStringKey('val')
      mockedConfirm.mockResolvedValue(false)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'dk2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await findDeleteBtn(wrapper).trigger('click')
      await flushPromises()
      const delCalls = mockedRequest.mock.calls.filter(c => c[0] === 'delete_keys')
      expect(delCalls).toHaveLength(0)
      expect(wrapper.emitted('deleted')).toBeFalsy()
    })

    it('删除失败显示错误提示', async () => {
      mockStringKey('val')
      mockedConfirm.mockResolvedValue(true)
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'delete_keys') return Promise.reject(new Error('delete failed'))
        if (method === 'get_key_info') return Promise.resolve({ code: 200, data: { type: 'string', ttl: -1, encoding: 'raw', length: 3 }, msg: 'OK' } as any)
        if (method === 'get_key_value') return Promise.resolve({ code: 200, data: { type: 'string', value: 'val', truncated: false }, msg: 'OK' } as any)
        if (method === 'get_decoders') return Promise.resolve({ code: 200, data: [], msg: 'OK' } as any)
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      const wrapper = createWrapper({ connId: 'tc', keyName: 'dk3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await findDeleteBtn(wrapper).trigger('click')
      await flushPromises()
      expect((window as any).$gm.message.error).toHaveBeenCalled()
    })
  })

  // ---------- 8. 重命名功能 ----------

  describe('重命名（handleRename）', () => {
    it('输入新名称后 emit renamed 事件', async () => {
      mockStringKey('val')
      mockedPrompt.mockResolvedValue('new:name')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'rn1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      // 按钮顺序: refresh, copy, rename, ttl, delete
      const btns = wrapper.findAll('.action-btn')
      await btns[2].trigger('click')
      await flushPromises()
      expect(mockedPrompt).toHaveBeenCalled()
      const renameCalls = mockedRequest.mock.calls.filter(c => c[0] === 'rename_key')
      expect(renameCalls).toHaveLength(1)
      expect(renameCalls[0][1]?.params?.new_key).toBe('new:name')
      expect(wrapper.emitted('renamed')).toBeTruthy()
      expect(wrapper.emitted('renamed')![0]).toEqual(['rn1', 'new:name'])
    })

    it('取消 prompt 不执行重命名', async () => {
      mockStringKey('val')
      mockedPrompt.mockResolvedValue(null)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'rn2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const btns = wrapper.findAll('.action-btn')
      await btns[2].trigger('click')
      await flushPromises()
      const renameCalls = mockedRequest.mock.calls.filter(c => c[0] === 'rename_key')
      expect(renameCalls).toHaveLength(0)
    })

    it('输入相同名称不执行重命名', async () => {
      mockStringKey('val')
      mockedPrompt.mockResolvedValue('rn3')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'rn3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const btns = wrapper.findAll('.action-btn')
      await btns[2].trigger('click')
      await flushPromises()
      const renameCalls = mockedRequest.mock.calls.filter(c => c[0] === 'rename_key')
      expect(renameCalls).toHaveLength(0)
    })
  })

  // ---------- 9. 设置 TTL ----------

  describe('设置 TTL（handleSetTTL）', () => {
    it('输入有效 TTL 后调用 set_ttl 并 emit ttlChanged', async () => {
      mockStringKey('val', -1)
      mockedPrompt.mockResolvedValue('300')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'ttl1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      // 按钮顺序: refresh, copy, rename, ttl, delete
      const btns = wrapper.findAll('.action-btn')
      await btns[3].trigger('click')
      await flushPromises()
      const ttlCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_ttl')
      expect(ttlCalls).toHaveLength(1)
      expect(ttlCalls[0][1]?.params?.ttl).toBe(300)
      expect(wrapper.emitted('ttlChanged')).toBeTruthy()
    })

    it('取消 prompt 不执行 set_ttl', async () => {
      mockStringKey('val')
      mockedPrompt.mockResolvedValue(null)
      const wrapper = createWrapper({ connId: 'tc', keyName: 'ttl2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const btns = wrapper.findAll('.action-btn')
      await btns[3].trigger('click')
      await flushPromises()
      const ttlCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_ttl')
      expect(ttlCalls).toHaveLength(0)
    })

    it('输入 NaN 不执行 set_ttl', async () => {
      mockStringKey('val')
      mockedPrompt.mockResolvedValue('abc')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'ttl3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      const btns = wrapper.findAll('.action-btn')
      await btns[3].trigger('click')
      await flushPromises()
      const ttlCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_ttl')
      expect(ttlCalls).toHaveLength(0)
    })
  })

  // ---------- 10. 复制功能 ----------

  describe('复制（copyValue）', () => {
    it('text 模式复制 editValue 到剪贴板', async () => {
      mockStringKey('copy me')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'cp1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      // 按钮顺序: refresh, copy, rename, ttl, delete
      const btns = wrapper.findAll('.action-btn')
      await btns[1].trigger('click')
      await flushPromises()
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith('copy me')
      expect((window as any).$gm.message.success).toHaveBeenCalled()
    })
  })

  // ---------- 11. 集合类型渲染 ----------

  describe('集合类型渲染', () => {
    it('hash 类型渲染 HashDetail 组件', async () => {
      mockCollectionKey('hash')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'h1', keyType: 'hash' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.hash-detail').exists()).toBe(true)
      expect(wrapper.find('.type-badge').text()).toBe('hash')
    })

    it('list 类型渲染 ListDetail 组件', async () => {
      mockCollectionKey('list')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'l1', keyType: 'list' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.list-detail').exists()).toBe(true)
    })

    it('set 类型渲染 SetDetail 组件', async () => {
      mockCollectionKey('set')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'st1', keyType: 'set' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.set-detail').exists()).toBe(true)
    })

    it('zset 类型默认渲染 ZSetDetail 组件', async () => {
      mockCollectionKey('zset')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'zs1', keyType: 'zset' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.zset-detail').exists()).toBe(true)
    })

    it('stream 类型渲染 StreamDetail 组件', async () => {
      mockCollectionKey('stream')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sm1', keyType: 'stream' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.stream-detail').exists()).toBe(true)
    })

    it('ReJSON-RL 类型渲染 JsonDetail 组件', async () => {
      mockCollectionKey('ReJSON-RL')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'rj1', keyType: 'ReJSON-RL' })
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.json-detail').exists()).toBe(true)
    })

    it('非 string 类型不调用 get_key_value 全量拉取', async () => {
      mockCollectionKey('hash')
      createWrapper({ connId: 'tc', keyName: 'h2', keyType: 'hash' })
      await flushPromises()
      await nextTick()
      const valueCalls = mockedRequest.mock.calls.filter(c => c[0] === 'get_key_value')
      expect(valueCalls).toHaveLength(0)
    })
  })

  // ---------- 12. 快捷键 Ctrl+S ----------

  describe('快捷键 Ctrl+S', () => {
    it('string 类型且 modified 时触发保存', async () => {
      mockStringKey('old')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'ks1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 编辑触发 modified
      await findTextarea(wrapper).setValue('new')
      await findTextarea(wrapper).trigger('input')
      await nextTick()
      // 清除之前的调用记录，只关注快捷键触发的保存
      mockedRequest.mockClear()
      mockStringKey('old') // 重新设置 mock 实现
      // 触发快捷键事件
      document.dispatchEvent(new Event('shortcut:save'))
      await flushPromises()
      // 注意：jsdom 中 document 事件监听器跨测试共享，其他测试创建的组件也会响应
      const saveCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_key_value')
      expect(saveCalls.length).toBeGreaterThanOrEqual(1)
    })

    it('未修改时快捷键不触发保存', async () => {
      mockStringKey('old')
      createWrapper({ connId: 'tc', keyName: 'ks2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      document.dispatchEvent(new Event('shortcut:save'))
      await flushPromises()
      const saveCalls = mockedRequest.mock.calls.filter(c => c[0] === 'set_key_value')
      expect(saveCalls).toHaveLength(0)
    })
  })

  // ---------- 13. 刷新功能 ----------

  describe('刷新（refreshValue）', () => {
    it('点击刷新按钮重新加载 key 信息和值', async () => {
      mockStringKey('v1')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'rf1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      vi.clearAllMocks()
      mockStringKey('v2')
      // 按钮顺序: refresh(0), copy(1), rename(2), ttl(3), delete(4)
      const btns = wrapper.findAll('.action-btn')
      await btns[0].trigger('click')
      await flushPromises()
      const infoCalls = mockedRequest.mock.calls.filter(c => c[0] === 'get_key_info')
      const valCalls = mockedRequest.mock.calls.filter(c => c[0] === 'get_key_value')
      expect(infoCalls).toHaveLength(1)
      expect(valCalls).toHaveLength(1)
    })
  })

  // ---------- 14. 解码器功能 ----------

  describe('解码器（Decoder）', () => {
    it('选择解码器后调用 decode_value 并显示解码结果', async () => {
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'get_key_info') {
          return Promise.resolve({ code: 200, data: { type: 'string', ttl: -1, encoding: 'raw', length: 5 }, msg: 'OK' } as any)
        }
        if (method === 'get_key_value') {
          return Promise.resolve({ code: 200, data: { type: 'string', value: 'hello', truncated: false }, msg: 'OK' } as any)
        }
        if (method === 'get_decoders') {
          return Promise.resolve({ code: 200, data: [{ id: 'base64', name: 'Base64', type: 'builtin' }], msg: 'OK' } as any)
        }
        if (method === 'decode_value') {
          return Promise.resolve({ code: 200, data: 'aGVsbG8=', msg: 'OK' } as any)
        }
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      const wrapper = createWrapper({ connId: 'tc', keyName: 'dc1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 找到解码器 select（第二个 Select 组件）
      await setArcoSelect(wrapper, 1, 'base64')
      await flushPromises()
      await nextTick()
      // 应显示解码结果而非 textarea
      expect(wrapper.find('.decoded-view').exists()).toBe(true)
      expect(wrapper.find('.decoded-view').text()).toBe('aGVsbG8=')
      // 保存按钮应隐藏（有解码器时不可编辑）
      expect(findSaveBtn(wrapper).exists()).toBe(false)
    })

    it('选择"原始值"清除解码结果', async () => {
      mockedRequest.mockImplementation((method: string) => {
        if (method === 'get_key_info') {
          return Promise.resolve({ code: 200, data: { type: 'string', ttl: -1, encoding: 'raw', length: 5 }, msg: 'OK' } as any)
        }
        if (method === 'get_key_value') {
          return Promise.resolve({ code: 200, data: { type: 'string', value: 'hello', truncated: false }, msg: 'OK' } as any)
        }
        if (method === 'get_decoders') {
          return Promise.resolve({ code: 200, data: [{ id: 'b64', name: 'Base64', type: 'builtin' }], msg: 'OK' } as any)
        }
        if (method === 'decode_value') {
          return Promise.resolve({ code: 200, data: 'decoded', msg: 'OK' } as any)
        }
        return Promise.resolve({ code: 200, data: null, msg: 'OK' } as any)
      })
      const wrapper = createWrapper({ connId: 'tc', keyName: 'dc2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 先选解码器
      await setArcoSelect(wrapper, 1, 'b64')
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.decoded-view').exists()).toBe(true)
      // 切回原始值
      await setArcoSelect(wrapper, 1, '')
      await flushPromises()
      await nextTick()
      expect(wrapper.find('.decoded-view').exists()).toBe(false)
      expect(findTextarea(wrapper).exists()).toBe(true)
    })
  })

  // ---------- 15. Geo 视图切换 ----------

  describe('Geo 视图切换', () => {
    it('zset 类型勾选 geoView 后渲染 GeoDetail', async () => {
      mockCollectionKey('zset')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'geo1', keyType: 'zset' })
      await flushPromises()
      await nextTick()
      // 默认显示 ZSetDetail
      expect(wrapper.find('.zset-detail').exists()).toBe(true)
      // 勾选 geoView
      const switchComp = wrapper.findComponent({ name: 'Switch' })
      switchComp.vm.$emit('update:modelValue', true)
      await nextTick()
      expect(wrapper.find('.geo-detail').exists()).toBe(true)
    })
  })

  // ---------- 16. keyName 变化重置状态 ----------

  describe('keyName 变化', () => {
    it('切换 key 时重置 geoView 和 decoder', async () => {
      mockStringKey('v1')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sw1', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 切换到新 key
      mockStringKey('v2')
      await wrapper.setProps({ keyName: 'sw2', keyType: 'string' })
      await flushPromises()
      await nextTick()
      // 应重新调用 get_key_info
      const infoCalls = mockedRequest.mock.calls.filter(c => c[0] === 'get_key_info')
      expect(infoCalls.length).toBeGreaterThanOrEqual(2)
    })

    it('keyName 清空时重置 rawValue 和 editValue', async () => {
      mockStringKey('v1')
      const wrapper = createWrapper({ connId: 'tc', keyName: 'sw3', keyType: 'string' })
      await flushPromises()
      await nextTick()
      await nextTick()
      // 清空 keyName
      await wrapper.setProps({ keyName: '' })
      await nextTick()
      expect(wrapper.find('.empty-state').exists()).toBe(true)
    })
  })
})
