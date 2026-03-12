import { createApp } from 'vue'
import { createPinia } from 'pinia'
import i18n from './i18n'
import App from './App.vue'
import '@arco-design/web-vue/dist/arco.css'
import './assets/styles/variables.css'
import './assets/styles/global.css'
import { vEllipsisTip, initSelectEllipsisTip } from './directives/ellipsisTip'
import { vBodyTooltip } from './directives/bodyTooltip'
import { isGmssh } from './utils/platform'

;(async () => {
  if (isGmssh()) {
    // GMSSH 模式：加载平台 SDK（同步注册 $gm 到 window）
    await import('gm-app-sdk')
  }

  const app = createApp(App)
  app.use(createPinia())
  app.use(i18n)
  app.directive('ellipsis-tip', vEllipsisTip)
  app.directive('body-tooltip', vBodyTooltip)
  app.mount('#app')
  initSelectEllipsisTip()

  // Global event delegation for [data-tooltip] elements
  // Renders tooltip as body-appended popup (avoids overflow:hidden clipping)
  initDataTooltip()
})()

function initDataTooltip() {
  let tipEl: HTMLDivElement | null = null
  let hideTimer: ReturnType<typeof setTimeout> | null = null
  let currentTarget: HTMLElement | null = null

  function ensureTip(): HTMLDivElement {
    if (!tipEl) {
      tipEl = document.createElement('div')
      tipEl.className = 'body-tooltip-popup'
      document.body.appendChild(tipEl)
    }
    return tipEl
  }

  function show(el: HTMLElement) {
    const text = el.getAttribute('data-tooltip')
    if (!text) return
    currentTarget = el

    const tip = ensureTip()
    if (hideTimer) { clearTimeout(hideTimer); hideTimer = null }

    tip.textContent = text
    tip.style.display = 'block'
    tip.style.opacity = '0'

    requestAnimationFrame(() => {
      const rect = el.getBoundingClientRect()
      const tipRect = tip.getBoundingClientRect()
      const pos = el.getAttribute('data-tooltip-pos') || 'top'

      let top: number, left: number

      if (pos === 'bottom') {
        top = rect.bottom + 6
        left = rect.left + (rect.width - tipRect.width) / 2
      } else if (pos === 'right') {
        top = rect.top + (rect.height - tipRect.height) / 2
        left = rect.right + 6
      } else if (pos === 'left') {
        top = rect.top + (rect.height - tipRect.height) / 2
        left = rect.left - tipRect.width - 6
      } else {
        // top (default)
        top = rect.top - tipRect.height - 6
        if (top < 4) top = rect.bottom + 6 // fallback to bottom
        left = rect.left + (rect.width - tipRect.width) / 2
      }

      if (left < 4) left = 4
      if (left + tipRect.width > window.innerWidth - 4) left = window.innerWidth - tipRect.width - 4

      tip.style.left = left + 'px'
      tip.style.top = top + 'px'
      tip.style.opacity = '1'
    })
  }

  function hide() {
    if (hideTimer) clearTimeout(hideTimer)
    hideTimer = setTimeout(() => {
      if (tipEl) {
        tipEl.style.opacity = '0'
        setTimeout(() => { if (tipEl) tipEl.style.display = 'none' }, 150)
      }
      currentTarget = null
    }, 80)
  }

  document.addEventListener('mouseover', (e) => {
    const target = (e.target as HTMLElement).closest?.('[data-tooltip]') as HTMLElement | null
    if (target && target !== currentTarget) {
      show(target)
    }
  })

  document.addEventListener('mouseout', (e) => {
    const target = (e.target as HTMLElement).closest?.('[data-tooltip]') as HTMLElement | null
    const related = (e.relatedTarget as HTMLElement)?.closest?.('[data-tooltip]') as HTMLElement | null
    if (target && target !== related) {
      hide()
    }
  })
}
