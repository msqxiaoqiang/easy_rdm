export function applyTheme(theme: string) {
  // 缓存到 localStorage，供 index.html 预设背景色
  localStorage.setItem('app-theme', theme)
  let resolved = theme
  if (theme === 'auto') {
    resolved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  document.documentElement.setAttribute('data-theme', resolved)
  // Arco Design dark mode
  if (resolved === 'dark') {
    document.body.setAttribute('arco-theme', 'dark')
  } else {
    document.body.removeAttribute('arco-theme')
  }
}
