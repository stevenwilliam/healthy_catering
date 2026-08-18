/** The language selector.
 *
 * Same design as the public pages' one, and the same reasoning:
 *
 *  - Inline SVG flags, never the flag emoji. Windows ships no glyphs for
 *    regional-indicator pairs, so 🇮🇩 🇬🇧 🇨🇳 render as the letter boxes
 *    "ID" "GB" "CN" on the browser most customers here are using.
 *  - The flags are aria-hidden decoration; the language NAME is the label. A
 *    flag is a country, not a language, and English is not the flag of one.
 *  - Every option is written in its own language, so someone who has landed in
 *    a language they cannot read can still find their way out.
 *  - Closed, it is the current language's flag alone (Steven, 2026-08-18); the
 *    name took too much of the bar. aria-label still names the control, so
 *    what a screen reader announces is unchanged — only the sighted label is
 *    gone, and the open menu still spells every language out.
 */
import { useEffect, useRef, useState } from 'react'
import { LOCALES, LOCALE_INFO, useI18n, type Locale } from '../lib/i18n'

const FLAGS: Record<Locale, JSX.Element> = {
  id: (
    <svg viewBox="0 0 6 4" aria-hidden="true" focusable="false" className="flag">
      <rect width="6" height="2" fill="#CE1126" />
      <rect y="2" width="6" height="2" fill="#F5F5F5" />
    </svg>
  ),
  en: (
    <svg viewBox="0 0 60 40" aria-hidden="true" focusable="false" className="flag">
      <rect width="60" height="40" fill="#012169" />
      <path d="M0,0 60,40 M60,0 0,40" stroke="#F5F5F5" strokeWidth="8" />
      <path d="M0,0 60,40 M60,0 0,40" stroke="#C8102E" strokeWidth="4" />
      <path d="M30,0 V40 M0,20 H60" stroke="#F5F5F5" strokeWidth="13" />
      <path d="M30,0 V40 M0,20 H60" stroke="#C8102E" strokeWidth="8" />
    </svg>
  ),
  zh: (
    <svg viewBox="0 0 30 20" aria-hidden="true" focusable="false" className="flag">
      <rect width="30" height="20" fill="#EE1C25" />
      <g fill="#FFDE00">
        {[
          'translate(5,5) scale(3)',
          'translate(10,2) scale(1) rotate(23)',
          'translate(12,4) scale(1) rotate(46)',
          'translate(12,7) scale(1) rotate(70)',
          'translate(10,9) scale(1) rotate(21)',
        ].map((transform) => (
          <path
            key={transform}
            transform={transform}
            d="M0,-1 .225,-.309 .951,-.309 .363,.118 .588,.809 0,.382 -.588,.809 -.363,.118 -.951,-.309 -.225,-.309Z"
          />
        ))}
      </g>
    </svg>
  ),
}

export default function LanguageSelector() {
  const { locale, setLocale, t } = useI18n()
  const [open, setOpen] = useState(false)
  const box = useRef<HTMLDivElement>(null)

  // Close on an outside click or on Escape. Without the Escape handler a
  // keyboard user who opens the menu has no way to dismiss it without
  // choosing something.
  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div className="relative" ref={box}>
      <button
        type="button"
        className="langpick-trigger"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={t('lang.choose')}
        onClick={() => setOpen((v) => !v)}
      >
        {FLAGS[locale]}
        <svg viewBox="0 0 10 6" aria-hidden="true" focusable="false"
             className={`h-1.5 w-2.5 transition-transform ${open ? 'rotate-180' : ''}`}>
          <path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.6"
                strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>

      {open && (
        <ul className="langpick-list" role="menu">
          {LOCALES.map((l) => (
            <li key={l} role="none">
              <button
                type="button"
                role="menuitemradio"
                aria-checked={l === locale}
                lang={LOCALE_INFO[l].tag}
                className="langpick-item"
                onClick={() => {
                  setLocale(l)
                  setOpen(false)
                }}
              >
                {FLAGS[l]}
                <span>{LOCALE_INFO[l].endonym}</span>
                {/* The tick is a second signal beside aria-checked — the
                    current language is never indicated by colour alone. */}
                {l === locale && (
                  <svg viewBox="0 0 12 12" aria-hidden="true" focusable="false"
                       className="ml-auto h-3.5 w-3.5">
                    <path d="M2 6.5l2.8 2.8L10 3.5" fill="none" stroke="currentColor"
                          strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
