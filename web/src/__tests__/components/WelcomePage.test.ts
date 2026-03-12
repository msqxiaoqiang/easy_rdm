import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ArcoVue from '@arco-design/web-vue'
import WelcomePage from '@/components/layout/WelcomePage.vue'
import zhCN from '@/i18n/locales/zh-CN'

function createWrapper() {
  const i18n = createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: { 'zh-CN': zhCN },
  })
  return mount(WelcomePage, {
    global: { plugins: [i18n, ArcoVue] },
  })
}

describe('WelcomePage.vue', () => {
  it('应渲染标题和描述', () => {
    const wrapper = createWrapper()
    expect(wrapper.find('.welcome-title').text()).toBe('Easy RDM')
    expect(wrapper.find('.welcome-desc').text()).toBe(zhCN.app.welcomeDesc)
  })

  it('应渲染新建连接按钮', () => {
    const wrapper = createWrapper()
    const btn = wrapper.find('.welcome-actions button.arco-btn-primary')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain(zhCN.connection.new)
  })

  it('点击新建连接按钮应触发 newConnection 事件', async () => {
    const wrapper = createWrapper()
    await wrapper.find('.welcome-actions button.arco-btn-primary').trigger('click')
    expect(wrapper.emitted('newConnection')).toHaveLength(1)
  })

  it('应渲染使用提示', () => {
    const wrapper = createWrapper()
    expect(wrapper.find('.welcome-tip').text()).toBe(zhCN.app.welcomeTip)
  })
})
