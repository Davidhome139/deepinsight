import { createI18n } from 'vue-i18n'
import messages from '../locales'

const getBrowserLanguage = (): string => {
  const browserLang = navigator.language
  if (browserLang.startsWith('zh')) {
    return browserLang.startsWith('zh-TW') || browserLang.startsWith('zh-HK') ? 'zh-TW' : 'zh-CN'
  }
  return 'en'
}

const savedLanguage = localStorage.getItem('language') || getBrowserLanguage()

const i18n = createI18n({
  legacy: false,
  locale: savedLanguage,
  fallbackLocale: 'en',
  messages,
  silentTranslationWarn: true,
  silentFallbackWarn: true
})

export default i18n
