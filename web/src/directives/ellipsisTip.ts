import type { Directive } from 'vue'

let tipEl: HTMLDivElement | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null
let tipInited = false

function ensureTipEl(): HTMLDivElement {
  if (!tipEl) {
    tipEl = document.createElement('div')
    tipEl.className = 'ellipsis-tip-popup'
    document.body.appendChild(tipEl)
  }
  if (!tipInited) {
    tipInited = true
    tipEl.addEventListener('mouseenter', () => {
      if (hideTimer) { clearTimeout(hideTimer); hideTimer = null }
    })
    tipEl.addEventListener('mouseleave', scheduleHide)
  }
  return tipEl
}

function showTip(el: HTMLElement) {
  if (el.scrollWidth <= el.clientWidth) return

  const tip = ensureTipEl()
  if (hideTimer) { clearTimeout(hideTimer); hideTimer = null }

  tip.textContent = el.textContent?.trim() || ''
  tip.style.opacity = '0'
  tip.style.display = 'block'

  // measure after content is set
  const rect = el.getBoundingClientRect()
  const tipRect = tip.getBoundingClientRect()
  let left = rect.left + (rect.width - tipRect.width) / 2

  // default: above the element; fallback: below if not enough space above
  let top = rect.top - tipRect.height - 6
  if (top < 4) {
    top = rect.bottom + 6
  }

  // keep within horizontal viewport
  if (left < 4) left = 4
  if (left + tipRect.width > window.innerWidth - 4) left = window.innerWidth - tipRect.width - 4

  tip.style.left = left + 'px'
  tip.style.top = top + 'px'
  tip.style.opacity = '1'
}

function scheduleHide() {
  if (hideTimer) clearTimeout(hideTimer)
  hideTimer = setTimeout(() => {
    if (tipEl) {
      tipEl.style.opacity = '0'
      setTimeout(() => { if (tipEl) tipEl.style.display = 'none' }, 150)
    }
  }, 100)
}

export const vEllipsisTip: Directive = {
  mounted(el: HTMLElement) {
    const onEnter = () => showTip(el)
    const onLeave = scheduleHide
    el.addEventListener('mouseenter', onEnter)
    el.addEventListener('mouseleave', onLeave)
    ;(el as any)._ellipsisTipHandlers = { onEnter, onLeave }
  },
  beforeUnmount(el: HTMLElement) {
    const h = (el as any)._ellipsisTipHandlers
    if (h) {
      el.removeEventListener('mouseenter', h.onEnter)
      el.removeEventListener('mouseleave', h.onLeave)
    }
  },
}

// Global delegation: auto tooltip for truncated Arco select values & options
let delegationInited = false
export function initSelectEllipsisTip() {
  if (delegationInited) return
  delegationInited = true
  document.addEventListener('mouseenter', (e: Event) => {
    const target = e.target as HTMLElement
    // Match select selected-value display or option content
    const el = target.closest?.('.arco-select-view-value, .arco-select-option-content') as HTMLElement | null
    if (el && el.scrollWidth > el.clientWidth) {
      showTip(el)
    }
  }, true)
  document.addEventListener('mouseleave', (e: Event) => {
    const target = e.target as HTMLElement
    if (target.closest?.('.arco-select-view-value, .arco-select-option-content')) {
      scheduleHide()
    }
  }, true)
}
