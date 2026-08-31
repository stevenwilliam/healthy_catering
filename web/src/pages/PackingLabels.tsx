import { useEffect, useState } from 'react'
import { ApiFailure, request } from '../lib/api'
import { State } from '../components/ui'
import ExportCsv from '../components/ExportCsv'
import { useT } from '../lib/i18n'
import { serviceDateWIB } from './AdminDashboard'

/** P2 — packing labels, 100×150 mm with a 100×50 mm compact variant
 *  (docs/10 §4.11).
 *
 * One label per delivery LINE. The server prints from the order SNAPSHOT, not
 * the live menu, so a substitution made after the order does not change a
 * label that has already been printed — that is the repository's rule and this
 * screen must not undo it by re-deriving anything.
 *
 * Renders outside the app shell so nothing overlays a label, and each label is
 * `break-inside: avoid` so one never straddles a page.
 */

type Label = {
  delivery_id: string
  delivery_code: string
  service_date: string
  slot: string
  kitchen_code: string
  customer_name: string
  phone: string
  address_line: string
  district: string
  diet_type: string
  qty: number
  foods: string
  allergens: string
  driver_note: string
}

export default function PackingLabels() {
  const t = useT()
  const [rows, setRows] = useState<Label[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [date, setDate] = useState(serviceDateWIB())
  const [compact, setCompact] = useState(false)

  useEffect(() => {
    setLoading(true)
    request<Label[]>(`/admin/reports/packing-labels?from=${date}&to=${date}`)
      .then(setRows)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('label.load_failed')))
      .finally(() => setLoading(false))
  }, [date]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="min-h-screen bg-canvas py-8 print:bg-white print:py-0">
      <div className="mx-auto mb-6 flex max-w-[1100px] flex-wrap items-end justify-between gap-3 px-4 print:hidden">
        <div className="flex flex-wrap items-end gap-3">
          <div>
            <label className="label" htmlFor="lab-date">{t('price.order_date')}</label>
            <input id="lab-date" type="date" className="field" value={date}
                   onChange={(e) => setDate(e.target.value)} />
          </div>
          {/* Two sizes, chosen not guessed — the compact label is what goes on
              a drink or a second box. */}
          <div className="flex gap-2">
            <button className={compact ? 'chip-off' : 'chip-on'} onClick={() => setCompact(false)}>
              100 × 150 mm
            </button>
            <button className={compact ? 'chip-on' : 'chip-off'} onClick={() => setCompact(true)}>
              {t('label.compact')} · 100 × 50 mm
            </button>
          </div>
        </div>
        <div className="flex gap-3">
          <ExportCsv path="/admin/reports/packing-labels" params={{ from: date, to: date }}
                     filename={`packing-labels-${date}`} />
          <button className="btn-primary" onClick={() => window.print()}>{t('ui.print')}</button>
        </div>
      </div>

      <State loading={loading} error={error} empty={rows.length === 0}>
        <div className="mx-auto flex max-w-[1100px] flex-wrap justify-center gap-5 px-4 print:max-w-none print:gap-2 print:px-0">
          {rows.map((r) => (compact
            ? <CompactLabel key={r.delivery_id + r.delivery_code} row={r} />
            : <FullLabel key={r.delivery_id + r.delivery_code} row={r} />))}
        </div>
      </State>
    </div>
  )
}

/** 100×150 mm at 96dpi ≈ 378×567 CSS px — the canvas's own artboard size. */
function FullLabel({ row }: { row: Label }) {
  const t = useT()
  return (
    <article className="sheet-print label-100x150 flex flex-col gap-4 p-6">
      <div className="flex items-start justify-between gap-4">
        <img src="/images/evermore-wordmark-deep.png" alt="Evermore"
             width={560} height={60} className="block h-6 w-auto" />
        <span className="rounded-full bg-nourish-deep px-3 py-1 text-xs font-bold uppercase tracking-wider text-beige">
          {row.diet_type}
        </span>
      </div>
      <div className="h-px bg-nourish-deep opacity-25" />

      <div>
        <div className="kicker opacity-60">{t('label.for')}</div>
        <div className="font-display text-3xl font-bold leading-tight tracking-tight">
          {row.customer_name}
        </div>
        <div className="mt-1 text-sm leading-snug">{row.address_line}</div>
        {row.district && <div className="text-sm leading-snug">{row.district}</div>}
        <div className="mt-1 text-sm font-semibold">{row.phone}</div>
      </div>

      <div className="flex gap-3">
        <div className="flex-1 rounded-[12px] bg-nourish-deep p-3 text-beige">
          <div className="kicker opacity-70">{t('label.deliver')}</div>
          <div className="font-display text-xl font-bold">{row.slot}</div>
        </div>
        <div className="flex-1 rounded-[12px] border border-nourish-deep p-3">
          <div className="kicker opacity-60">{t('label.kitchen')}</div>
          <div className="font-display text-xl font-bold">{row.kitchen_code}</div>
        </div>
      </div>

      <div>
        <div className="kicker mb-1 opacity-60">{t('label.contents')}</div>
        <div className="font-display text-lg font-semibold leading-snug">{row.foods}</div>
        {row.qty > 1 && <div className="mt-1 text-sm">× {row.qty}</div>}
        {/* Allergens are bold and spelled out. This is a regulated claim on a
            food package, not a decoration, and it must survive a monochrome
            printer — so it is weight and words, never a colour. */}
        {row.allergens && (
          <div className="mt-2 text-sm font-bold">
            {t('label.allergens')}: {row.allergens}
          </div>
        )}
        {row.driver_note && (
          <div className="mt-1 text-xs leading-snug opacity-80">{row.driver_note}</div>
        )}
      </div>

      <div className="mt-auto flex items-end justify-between gap-4">
        <div>
          <div className="kicker opacity-60">{t('label.order')}</div>
          <div className="font-display text-lg font-bold">{row.delivery_code}</div>
          <div className="mt-1 text-xs">{t('label.keep_cold')}</div>
        </div>
        {/* The QR block is a reserved area, not a rendered code: generating one
            needs the tracking URL scheme settled, and printing a square that
            scans to nothing is worse than printing a labelled placeholder. */}
        <div className="flex h-20 w-20 items-center justify-center rounded-[12px] bg-nourish-deep text-center text-[10px] font-bold leading-tight tracking-wider text-beige">
          {t('label.track')}
        </div>
      </div>
    </article>
  )
}

/** 100×50 mm ≈ 378×189 CSS px. */
function CompactLabel({ row }: { row: Label }) {
  const t = useT()
  return (
    <article className="sheet-print label-100x50 flex items-center justify-between gap-4 p-5">
      <div className="min-w-0">
        <div className="kicker opacity-60">{t('label.compact')}</div>
        <div className="truncate font-display text-xl font-bold leading-tight">
          {row.customer_name} · {row.slot}
        </div>
        <div className="truncate text-sm">{row.foods} · {row.diet_type}</div>
        {row.allergens && (
          <div className="truncate text-xs font-bold">
            {t('label.allergens')}: {row.allergens}
          </div>
        )}
      </div>
      <div className="h-14 w-14 shrink-0 rounded-[10px] bg-nourish-deep" aria-hidden="true" />
    </article>
  )
}
