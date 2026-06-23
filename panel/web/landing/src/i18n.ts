import { ref } from 'vue'
import en from './i18n/en.json'
import fa from './i18n/fa.json'
import zh from './i18n/zh.json'
import ru from './i18n/ru.json'

export type Locale = 'en' | 'fa' | 'zh' | 'ru'

const translations: Record<Locale, Record<string, string>> = { en, fa, zh, ru }

/** Detect initial locale from URL param, localStorage, or browser preference */
function detectLocale(): Locale {
  // 1. URL param ?lang=
  const params = new URLSearchParams(window.location.search)
  const urlLang = params.get('lang')?.toLowerCase()
  if (urlLang && urlLang in translations) return urlLang as Locale

  // 2. localStorage
  const stored = localStorage.getItem('koris_locale')
  if (stored && stored in translations) return stored as Locale

  // 3. Browser preference
  const nav = navigator.language?.toLowerCase() || ''
  if (nav.startsWith('fa')) return 'fa'
  if (nav.startsWith('zh')) return 'zh'
  if (nav.startsWith('ru')) return 'ru'

  return 'en'
}

const locale = ref<Locale>(detectLocale())

const rtlLocales: Locale[] = ['fa']

/** Translate a key, with optional interpolation for {year} etc. */
function t(key: string, params?: Record<string, string | number>): string {
  let value = translations[locale.value]?.[key] || translations.en[key] || key
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      value = value.replace(`{${k}}`, String(v))
    }
  }
  return value
}

/** Change locale and persist */
function setLocale(code: string) {
  const normalized = code.toLowerCase() as Locale
  if (!(normalized in translations)) return
  locale.value = normalized
  localStorage.setItem('koris_locale', normalized)
  // Update document direction for RTL
  const isRtl = rtlLocales.includes(normalized)
  document.documentElement.dir = isRtl ? 'rtl' : 'ltr'
  document.documentElement.lang = normalized
}

// Set initial direction
if (typeof document !== 'undefined') {
  const isRtl = rtlLocales.includes(locale.value)
  document.documentElement.dir = isRtl ? 'rtl' : 'ltr'
  document.documentElement.lang = locale.value
}

export function useI18n() {
  return { t, locale, setLocale }
}
