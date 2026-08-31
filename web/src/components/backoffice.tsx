import { type ReactNode } from 'react'

/** The back-office chrome S2–S5 share (docs/10 §4.10).
 *
 * S1 is drawn with the 228px rail; S2–S5 are drawn with a TOP BAR instead,
 * because each of them needs the full width — a week of calendar, five columns
 * of price table, a queue beside a detail panel, a map beside a form. Both
 * exist in the canvas and they are not interchangeable.
 *
 * Everything sitting ON the bar is 19px/700: beige on #468973 is 3.93, which
 * is AA for large text and nothing else (§4.1). Long supporting sentences —
 * S3's tax note, S5's polygon rule — are NOT put on the bar at 19px, where
 * they would dominate what they are subordinate to. They keep the canvas's
 * exact 14px #CCBDAA and drop one row onto the deep ground, where that
 * measures 6.47. `note` is that row.
 */
export function TopBar({
  title, actions, note,
}: {
  title: string
  /** Right-hand controls — week navigation, a publish button, a count pill. */
  actions?: ReactNode
  /** The supporting sentence. Rendered UNDER the bar, not on it. */
  note?: ReactNode
}) {
  return (
    <>
      <header className="appbar flex-wrap justify-between gap-4 rounded-t-lg px-6 py-4">
        <div className="flex min-w-0 items-center gap-5">
          <img
            src="/images/evermore-wordmark-light.png"
            width={560}
            height={60}
            alt="Evermore"
            className="block h-5 w-auto shrink-0"
          />
          <h1 className="truncate text-onbar font-bold text-beige">{title}</h1>
        </div>
        {actions && <div className="flex flex-wrap items-center gap-3">{actions}</div>}
      </header>
      {note && (
        <p className="m-0 border-b border-rule px-6 py-3 text-sm text-beige-deep">{note}</p>
      )}
    </>
  )
}

/** The framed board every S-artboard sits in: one rounded panel on the ground
 *  with the bar across its top. */
export function Board({ children }: { children: ReactNode }) {
  return (
    <section className="overflow-hidden rounded-lg border border-edge">
      {children}
    </section>
  )
}

/** Tabs — the queue filters in S4 and the four price tables in S3.
 *
 * Radio semantics: these select one view of one dataset, they are not
 * navigation, and a screen reader should say "2 of 4" rather than reading four
 * unrelated buttons.
 */
export function Tabs({
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

/** A legend entry: a swatch AND a word, never a swatch alone (§2.4 rule 4). */
export function LegendItem({ swatch, children }: { swatch: ReactNode; children: ReactNode }) {
  return (
    <span className="flex items-center gap-2 text-xs text-beige-deep">
      {swatch}
      {children}
    </span>
  )
}
