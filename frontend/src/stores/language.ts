import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import i18n from '../i18n'

type Locale = 'zh-CN' | 'zh-TW' | 'en'

export const useLanguageStore = defineStore('language', () => {
  const currentLocale = ref<Locale>(i18n.global.locale.value as Locale)

  const setLocale = (locale: Locale) => {
    currentLocale.value = locale
    i18n.global.locale.value = locale
    localStorage.setItem('language', locale)
  }

  watch(currentLocale, (newLocale) => {
    if (i18n.global.locale.value !== newLocale) {
      i18n.global.locale.value = newLocale
      localStorage.setItem('language', newLocale)
    }
  })

  return {
    currentLocale,
    setLocale
  }
})
