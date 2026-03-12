import type { Directive } from 'vue'

let tipEl: HTMLDivElement | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null

function ensureTipEl(): HTMLDivElement {
  if (!tipEl) {
    tipEl = document.createElement('div')
    tipEl.className = 'body-tooltip-popup'
    document.body.appendChild(tipEl)
  }
  return tipEl
}

function showTip(el: HTMLElement) {
  const text = (el as any)._bodyTooltipText || el.getAttribute('data-body-tooltip') || ''
  if (!text) return

  const tip = ensureTipEl()
  if (hideTimer) { clearTimeout(hideTimer); hideTimer = null }

  tip.textContent = text
  tip.style.display = 'block'
  tip.style.opacity = '0'

  // 需要一帧让浏览器计算 tip 尺寸
  requestAnimationFrame(() => {
    const rect = el.getBoundingClientRect()
    const tipRect = tip.getBoundingClientRect()

    // 默认显示在上方，空间不够则下方
    let top = rect.top - tipRect.height - 6
    if (top < 4) top = rect.bottom + 6

    let left = rect.left + (rect.width - tipRect.width) / 2
    if (left < 4) left = 4
    if (left + tipRect.width > window.innerWidth - 4) {
      left = window.innerWidth - tipRect.width - 4
    }

    tip.style.left = left + 'px'
    tip.style.top = top + 'px'
    tip.style.opacity = '1'
  })
}

function scheduleHide() {
  if (hideTimer) clearTimeout(hideTimer)
  hideTimer = setTimeout(() => {
    if (tipEl) {
      tipEl.style.opacity = '0'
      setTimeout(() => { if (tipEl) tipEl.style.display = 'none' }, 150)
    }
  }, 80)
}

export const vBodyTooltip: Directive = {
  mounted(el: HTMLElement, binding) {
    const text = binding.value || ''
    ;(el as any)._bodyTooltipText = text
    el.setAttribute('data-body-tooltip', text)
    const onEnter = () => showTip(el)
    const onLeave = scheduleHide
    el.addEventListener('mouseenter', onEnter)
    el.addEventListener('mouseleave', onLeave)
    ;(el as any)._bodyTooltipHandlers = { onEnter, onLeave }
  },
  updated(el: HTMLElement, binding) {
    const text = binding.value || ''
    ;(el as any)._bodyTooltipText = text
    el.setAttribute('data-body-tooltip', text)
  },
  beforeUnmount(el: HTMLElement) {
    const h = (el as any)._bodyTooltipHandlers
    if (h) {
      el.removeEventListener('mouseenter', h.onEnter)
      el.removeEventListener('mouseleave', h.onLeave)
    }
  },
}
