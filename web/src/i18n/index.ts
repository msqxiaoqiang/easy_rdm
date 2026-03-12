import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import en from './locales/en'

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'en',
  messages: {
    'zh-CN': zhCN,
    en,
  },
})

/** Vite 兼容的动态语言包加载映射 */
const localeLoaders: Record<string, () => Promise<{ default: Record<string, any> }>> = {
  'zh-TW': () => import('./locales/zh-TW'),
  ja: () => import('./locales/ja'),
  ko: () => import('./locales/ko'),
  ru: () => import('./locales/ru'),
  fr: () => import('./locales/fr'),
}

const loadedLanguages = ['zh-CN', 'en']

export async function loadLanguage(lang: string) {
  if (loadedLanguages.includes(lang)) {
    i18n.global.locale.value = lang as any
    return
  }
  const loader = localeLoaders[lang]
  if (!loader) return
  const messages = await loader()
  i18n.global.setLocaleMessage(lang, messages.default as any)
  loadedLanguages.push(lang)
  i18n.global.locale.value = lang as any
}

export default i18n
