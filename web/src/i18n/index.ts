import { ref } from 'vue'

export type LocaleCode = 'zh-CN' | 'en-US' | 'ja-JP' | 'fr-FR'

const LOCALE_KEY = 'anox-locale'

export const locale = ref<LocaleCode>(
  (localStorage.getItem(LOCALE_KEY) as LocaleCode) || 'zh-CN'
)
export const messages = ref<Record<string, unknown>>({})

export interface LocaleMeta {
  code: LocaleCode
  name: string
}

export const availableLocales = ref<LocaleMeta[]>([])

function getNested(obj: Record<string, unknown>, path: string): string {
  const keys = path.split('.')
  let cur: unknown = obj
  for (const key of keys) {
    if (cur && typeof cur === 'object' && key in (cur as Record<string, unknown>)) {
      cur = (cur as Record<string, unknown>)[key]
    } else {
      return path
    }
  }
  return typeof cur === 'string' ? cur : path
}

export function t(key: string, params?: Record<string, string | number>): string {
  let text = getNested(messages.value as Record<string, unknown>, key)
  if (params) {
    for (const [k, v] of Object.entries(params)) {
      text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v))
    }
  }
  return text
}

export async function loadLocale(code: LocaleCode): Promise<void> {
  const res = await fetch(`/locales/${code}.json`)
  if (!res.ok) {
    throw new Error(`Failed to load locale: ${code}`)
  }
  messages.value = await res.json()
  locale.value = code
  localStorage.setItem(LOCALE_KEY, code)
  document.documentElement.lang = code
  const title = getNested(messages.value as Record<string, unknown>, 'app.title')
  if (title && title !== 'app.title') {
    document.title = title
  }
}

export async function initI18n(): Promise<void> {
  try {
    const metaRes = await fetch('/locales/locales.json')
    if (metaRes.ok) {
      availableLocales.value = await metaRes.json()
    }
  } catch {
    availableLocales.value = [
      { code: 'zh-CN', name: '中文' },
      { code: 'en-US', name: 'English' },
      { code: 'ja-JP', name: '日本語' },
      { code: 'fr-FR', name: 'Français' },
    ]
  }
  await loadLocale(locale.value)
}

export function useI18n() {
  return {
    locale,
    messages,
    t,
    loadLocale,
    availableLocales,
  }
}
