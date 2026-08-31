import { type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { useT } from '../lib/i18n'

/** The chrome artboards 01–06, M2 and M3 repeat (docs/10 §4.10).
 *
 * One thing runs through all of it: every string that sits on the mid-green
 * bar is 19px/700, because beige on #468973 is 3.93 — AA for large text and
 * nothing else (§4.1). That is why the bottom bar's kicker is not 12px and the
 * total is not 17px, which is what the artboards draw. The colour is the
 * canvas's, untouched; the size is what makes it legal.
 *
 * The artboards' 44px status strip ("09.41 · evermore.co.id") is the phone's
 * own status bar, not product UI, so it has no counterpart here.
 */

/** AppBar — the mid-green header with a round back button and a title. */
export function AppBar({
  title, back = true, trailing,
}: {
  title: string
  /** false on a ROOT screen. The artboards give those the wordmark instead of
   *  a back button and a title — there is nowhere to go back to, and the
   *  screen's own heading is already the first thing under the bar. */
  back?: boolean
  trailing?: ReactNode
}) {
  const nav = useNavigate()
  const t = useT()
  return (
    <header className="appbar justify-between rounded-t-frame">
      <div className="flex min-w-0 items-center gap-3">
        {back ? (
          <>
            <button
              type="button"
              className="btn-icon shrink-0"
              onClick={() => nav(-1)}
              aria-label={t('ui.back')}
            >
              {/* An arrow glyph, not an icon font: one character, no request,
                  and it inherits the button's ink. aria-hidden because the
                  button already has an accessible name. */}
              <span aria-hidden="true">←</span>
            </button>
            <h1 className="truncate text-onbar font-bold text-beige">{title}</h1>
          </>
        ) : (
          <img
            src="/images/evermore-wordmark-light.png"
            width={560}
            height={60}
            alt={title}
            className="block h-6 w-auto"
          />
        )}
      </div>
      {trailing}
    </header>
  )
}

/** BottomBar — the sticky action bar: running total left, action right. */
export function BottomBar({
  kicker, total, children,
}: {
  kicker: string
  total: ReactNode
  children: ReactNode
}) {
  return (
    <div className="bottombar rounded-b-frame">
      <div className="min-w-0">
        <div className="bottombar-kicker">{kicker}</div>
        <div className="bottombar-total">{total}</div>
      </div>
      <div className="flex shrink-0 items-center gap-2">{children}</div>
    </div>
  )
}

/** Stepper — the −/+ quantity control from 02 and 03.
 *
 * A group of two buttons and a live number rather than a number input: the
 * artboards draw it that way, and on a phone a numeric keyboard popping over
 * the sheet to change 2 into 3 is worse than two taps. The value is announced
 * politely so a screen-reader user hears the result of their own tap.
 */
export function Stepper({
  value, onChange, min = 0, max = 999, label, size = 'lg',
}: {
  value: number
  onChange: (n: number) => void
  min?: number
  max?: number
  label: string
  size?: 'lg' | 'sm'
}) {
  const t = useT()
  const box = size === 'lg' ? 'h-touch w-touch text-onbar' : 'h-9 w-9 text-base'
  return (
    <div
      className="inline-flex items-center rounded border-2 border-beige"
      role="group"
      aria-label={label}
    >
      <button
        type="button"
        className={`${box} font-bold text-beige disabled:opacity-40`}
        onClick={() => onChange(Math.max(min, value - 1))}
        disabled={value <= min}
        aria-label={t('ui.decrease')}
      >
        −
      </button>
      <span
        className={`min-w-[2rem] text-center font-bold ${size === 'lg' ? 'text-onbar' : 'text-base'}`}
        aria-live="polite"
      >
        {value}
      </span>
      <button
        type="button"
        className={`${box} font-bold text-beige disabled:opacity-40`}
        onClick={() => onChange(Math.min(max, value + 1))}
        disabled={value >= max}
        aria-label={t('ui.increase')}
      >
        +
      </button>
    </div>
  )
}

/** DayChips — the week strip from 01, M3 and 04.
 *
 * Radio semantics, not buttons: exactly one day is chosen, and a screen reader
 * should say "3 of 6" rather than reading six unrelated buttons.
 */
export function DayChips({
  days, value, onChange, disabled,
}: {
  days: string[]
  value: string
  onChange: (d: string) => void
  /** Dates with nothing published — shown, but not selectable. */
  disabled?: (d: string) => boolean
}) {
  return (
    <div className="flex gap-2" role="radiogroup">
      {days.map((d) => {
        const on = d === value
        const off = disabled?.(d) ?? false
        return (
          <button
            key={d}
            type="button"
            role="radio"
            aria-checked={on}
            disabled={off}
            onClick={() => onChange(d)}
            className={[
              'flex-1 rounded px-1 py-2 text-center transition-colors',
              on ? 'bg-beige text-nourish-deep' : 'border border-edge',
              off ? 'opacity-40' : '',
            ].join(' ')}
          >
            <div className={`text-xs font-bold uppercase tracking-wider ${on ? '' : 'text-beige-deep'}`}>
              {weekdayShort(d)}
            </div>
            <div className="font-display text-xl font-bold">{dayNum(d)}</div>
          </button>
        )
      })}
    </div>
  )
}

/** ChipRow — the diet filter from 01, M2 and S2. */
export function ChipRow({
  options, value, onChange, label,
}: {
  options: { id: string; name: string }[]
  value: string
  onChange: (id: string) => void
  label: string
}) {
  return (
    <div className="flex flex-wrap gap-2" role="radiogroup" aria-label={label}>
      {options.map((o) => (
        <button
          key={o.id}
          type="button"
          role="radio"
          aria-checked={o.id === value}
          className={o.id === value ? 'chip-on' : 'chip-off'}
          onClick={() => onChange(o.id)}
        >
          {o.name}
        </button>
      ))}
    </div>
  )
}

/** Phone — the 390px column the customer artboards are drawn in.
 *
 * Responsive, per Steven's call: exact at 390, and it simply stops growing
 * past its own width on a desktop rather than stretching a phone layout across
 * a monitor. The rounded frame and the ground behind it are the artboards'.
 */
export function Phone({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto flex min-h-[calc(100vh-8rem)] w-full max-w-[430px] flex-col overflow-hidden rounded-frame border border-edge">
      {children}
    </div>
  )
}

// ── Date helpers ────────────────────────────────────────────────────────────
// YYYY-MM-DD parsed at UTC noon. A bare date string is parsed as UTC midnight,
// which renders as the previous day in any negative-offset zone.
function at(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1, 12))
}
function weekdayShort(iso: string): string {
  return new Intl.DateTimeFormat('id-ID', { weekday: 'short', timeZone: 'UTC' }).format(at(iso))
}
function dayNum(iso: string): string {
  return new Intl.DateTimeFormat('id-ID', { day: 'numeric', timeZone: 'UTC' }).format(at(iso))
}

/** "Senin 1 Sep" — the artboards' own date format. */
export function dayLong(iso: string, locale = 'id-ID'): string {
  return new Intl.DateTimeFormat(locale, {
    weekday: 'long', day: 'numeric', month: 'short', timeZone: 'UTC',
  }).format(at(iso))
}

/** "4 jam 12 menit" — the cut-off countdown's own wording in 01 and 05. */
export function humanLeft(ms: number, locale = 'id-ID'): string {
  const total = Math.max(0, Math.floor(ms / 60000))
  const h = Math.floor(total / 60)
  const m = total % 60
  const unit = locale === 'id-ID'
    ? { h: 'jam', m: 'menit' }
    : locale === 'zh'
      ? { h: '小时', m: '分钟' }
      : { h: 'h', m: 'min' }
  const sep = locale === 'zh' ? '' : ' '
  return h > 0 ? `${h}${sep}${unit.h} ${m}${sep}${unit.m}` : `${m}${sep}${unit.m}`
}
