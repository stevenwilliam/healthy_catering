/** Locale handling for the app.
 *
 * The public pages carry the language in the URL (/en/, /zh/) because they are
 * indexed and shared, and a cookie serving three different bodies at one
 * address breaks both. The app is noindex and behind a login, so the opposite
 * trade is right here: the choice is a stored PREFERENCE that follows the
 * customer between screens and survives a reload, and no route changes.
 *
 * The catalogue lives in ./messages.ts. Same rule as the server side: a
 * missing translation falls back to Indonesian rather than rendering an empty
 * label, and an unknown key echoes itself so it is loud in review.
 */
import {
  createContext, useCallback, useContext, useEffect, useMemo, useState,
  type ReactNode,
} from 'react'
import { messages, type MessageKey } from './messages'

export const LOCALES = ['id', 'en', 'zh'] as const
export type Locale = (typeof LOCALES)[number]
export const DEFAULT_LOCALE: Locale = 'id'

export type LocaleInfo = {
  locale: Locale
  /** BCP-47, for <html lang>. Chinese carries the script: "zh" alone does not
   *  say Simplified or Traditional. */
  tag: string
  /** The language's name IN that language — someone who has landed in a
   *  language they cannot read still has to find their own in the list. */
  endonym: string
}

export const LOCALE_INFO: Record<Locale, LocaleInfo> = {
  id: { locale: 'id', tag: 'id-ID', endonym: 'Bahasa Indonesia' },
  en: { locale: 'en', tag: 'en', endonym: 'English' },
  zh: { locale: 'zh', tag: 'zh-Hans', endonym: '中文' },
}

const STORAGE_KEY = 'evermore.lang'

function isLocale(v: string | null | undefined): v is Locale {
  return !!v && (LOCALES as readonly string[]).includes(v)
}

/** Match a browser tag such as "en-GB" or "zh-TW" on its primary subtag, so a
 *  Traditional-Chinese browser still lands on Chinese rather than falling
 *  through to Indonesian. */
function fromTag(tag: string): Locale | null {
  const primary = tag.toLowerCase().split(/[-_]/)[0]
  return isLocale(primary) ? primary : null
}

export function detectLocale(): Locale {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (isLocale(stored)) return stored
  } catch {
    // Private mode or blocked storage: fall through to the browser's own
    // preference rather than failing to render.
  }
  for (const tag of navigator.languages ?? [navigator.language]) {
    const hit = fromTag(tag ?? '')
    if (hit) return hit
  }
  return DEFAULT_LOCALE
}

type Ctx = {
  locale: Locale
  setLocale: (l: Locale) => void
  t: (key: MessageKey) => string
}

const I18nContext = createContext<Ctx | null>(null)

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => detectLocale())

  // Keep <html lang> in step. Screen readers switch voice on it, and it is the
  // signal browsers use to offer (or not offer) to translate the page.
  useEffect(() => {
    document.documentElement.lang = LOCALE_INFO[locale].tag
  }, [locale])

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l)
    try {
      localStorage.setItem(STORAGE_KEY, l)
    } catch {
      // Not persisting is survivable; not switching would not be.
    }
  }, [])

  const t = useCallback(
    (key: MessageKey) => {
      const entry = messages[key]
      if (!entry) return key
      return entry[locale] || entry[DEFAULT_LOCALE] || key
    },
    [locale],
  )

  const value = useMemo(() => ({ locale, setLocale, t }), [locale, setLocale, t])
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): Ctx {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n outside I18nProvider')
  return ctx
}

/** Shorthand for the common case: `const t = useT()`. */
export function useT() {
  return useI18n().t
}
