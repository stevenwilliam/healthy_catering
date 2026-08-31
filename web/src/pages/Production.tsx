import { useEffect, useMemo, useState } from 'react'
import { ApiFailure, request } from '../lib/api'
import { State } from '../components/ui'
import ExportCsv from '../components/ExportCsv'
import { useT } from '../lib/i18n'
import { longDateWIB, serviceDateWIB } from './AdminDashboard'

/** P1 — the kitchen production sheet, A4 (docs/10 §4.11).
 *
 * Printed on the BEIGE sheet with deep ink, never on the green ground: a
 * flood-filled A4 is about 90% ink coverage on a kitchen printer, and this
 * page exists to come out of one every morning.
 *
 * It renders outside the app shell (see App.tsx) so no masthead, fixed footer
 * or floating button lands on top of a table row.
 */

type Row = {
  service_date: string
  slot: string
  kitchen: string
  diet_type: string
  food_name: string
  item_role: string
  portions: number
}

export default function Production() {
  const t = useT()
  const [rows, setRows] = useState<Row[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [date, setDate] = useState(serviceDateWIB())

  useEffect(() => {
    setLoading(true)
    request<Row[]>(`/admin/reports/production?from=${date}&to=${date}`)
      .then(setRows)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('prod.load_failed')))
      .finally(() => setLoading(false))
  }, [date]) // eslint-disable-line react-hooks/exhaustive-deps

  // The slot columns, in time order, from the data.
  const slots = useMemo(
    () => [...new Set(rows.map((r) => r.slot))].sort((a, b) => a.localeCompare(b)),
    [rows],
  )

  // Table 1 — the dish that names the meal (the MAIN item), by slot.
  const dishes = useMemo(() => {
    const by = new Map<string, { name: string; diet: string; bySlot: Map<string, number> }>()
    for (const r of rows) {
      if (r.item_role !== 'MAIN') continue
      const key = `${r.food_name}|${r.diet_type}`
      let e = by.get(key)
      if (!e) { e = { name: r.food_name, diet: r.diet_type, bySlot: new Map() }; by.set(key, e) }
      e.bySlot.set(r.slot, (e.bySlot.get(r.slot) ?? 0) + r.portions)
    }
    return [...by.values()].sort((a, b) => a.name.localeCompare(b.name))
  }, [rows])

  // Table 2 — every component, summed across slots. This is what gets bought
  // and prepped, so it counts every role, not just the main.
  const components = useMemo(() => {
    const by = new Map<string, { name: string; role: string; portions: number }>()
    for (const r of rows) {
      const e = by.get(r.food_name)
      if (e) e.portions += r.portions
      else by.set(r.food_name, { name: r.food_name, role: r.item_role, portions: r.portions })
    }
    return [...by.values()].sort((a, b) => b.portions - a.portions)
  }, [rows])

  const totalPortions = useMemo(
    () => rows.filter((r) => r.item_role === 'MAIN').reduce((n, r) => n + r.portions, 0),
    [rows],
  )
  const kitchens = useMemo(() => [...new Set(rows.map((r) => r.kitchen))], [rows])

  return (
    <div className="min-h-screen bg-canvas py-8 print:bg-white print:py-0">
      {/* The controls are screen-only: they must not print. */}
      <div className="mx-auto mb-6 flex max-w-[794px] flex-wrap items-end justify-between gap-3 px-4 print:hidden">
        <div>
          <label className="label" htmlFor="prod-date">{t('price.order_date')}</label>
          <input id="prod-date" type="date" className="field" value={date}
                 onChange={(e) => setDate(e.target.value)} />
        </div>
        <div className="flex gap-3">
          <ExportCsv path="/admin/reports/production" params={{ from: date, to: date }}
                     filename={`production-${date}`} />
          <button className="btn-primary" onClick={() => window.print()}>{t('ui.print')}</button>
        </div>
      </div>

      <State loading={loading} error={error}>
        {/* The A4 sheet. 794×1123 is A4 at 96dpi, which is what the canvas
            draws and what the @page rule below prints to. */}
        <article className="sheet-print mx-auto flex w-[794px] max-w-full flex-col gap-6 p-12">
          <header className="flex items-start justify-between gap-6">
            <div>
              <img src="/images/evermore-wordmark-deep.png" alt="Evermore"
                   width={560} height={60} className="mb-2 block h-8 w-auto" />
              <div className="font-display text-2xl font-bold tracking-tight">{t('prod.title')}</div>
              <div className="mt-1 text-sm">
                {longDateWIB(date)}
                {kitchens.length > 0 && <> · {kitchens.join(' · ')}</>}
              </div>
            </div>
            <div className="text-right text-sm leading-relaxed">
              <div className="font-bold">{t('prod.printed')} {printedAt()}</div>
              <div className="opacity-70">{t('prod.snapshot')} 15.00</div>
            </div>
          </header>

          <div className="grid grid-cols-4 gap-3">
            <PrintStat label={t('prod.total_portions')} value={totalPortions} inverted />
            <PrintStat label={t('dash.kitchen')} value={kitchens.length} />
            <PrintStat label={t('prod.slots')} value={slots.length} />
            <PrintStat label={t('prod.components_short')} value={components.length} />
          </div>

          {/* ── Portions by dish and slot ─────────────────────────────────── */}
          <section>
            <h2 className="mb-2 font-display text-xl font-semibold">{t('prod.per_meal_slot')}</h2>
            <div className="overflow-hidden rounded-[12px] border border-nourish-deep">
              <div
                className="grid text-sm"
                style={{ gridTemplateColumns: `2.1fr 1fr repeat(${slots.length}, 0.7fr)` }}
              >
                <PrintHead>{t('prod.meal')}</PrintHead>
                <PrintHead>{t('cal.diet')}</PrintHead>
                {slots.map((s) => <PrintHead key={s}>{s}</PrintHead>)}
                {dishes.map((d, i) => (
                  <div key={`${d.name}-${d.diet}`} className="contents">
                    <PrintCell last={i === dishes.length - 1} bold>{d.name}</PrintCell>
                    <PrintCell last={i === dishes.length - 1}>{d.diet}</PrintCell>
                    {slots.map((s) => (
                      <PrintCell key={s} last={i === dishes.length - 1} bold={!!d.bySlot.get(s)}>
                        {d.bySlot.get(s) ?? '—'}
                      </PrintCell>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          </section>

          {/* ── Component requirements ────────────────────────────────────── */}
          <section>
            <h2 className="mb-2 font-display text-xl font-semibold">{t('prod.components')}</h2>
            <div className="overflow-hidden rounded-[12px] border border-nourish-deep">
              <div className="grid text-sm" style={{ gridTemplateColumns: '2.4fr 1fr 1.2fr' }}>
                <PrintHead>{t('prod.component')}</PrintHead>
                {/* Role, not "Meal": the cell renders item_role (MAIN / SIDE /
                    DRINK). The header said "Meal" and disagreed with it. */}
                <PrintHead>{t('prod.role')}</PrintHead>
                <PrintHead>{t('prod.portions')}</PrintHead>
                {components.map((c, i) => (
                  <div key={c.name} className="contents">
                    <PrintCell last={i === components.length - 1} bold>{c.name}</PrintCell>
                    <PrintCell last={i === components.length - 1}>{c.role}</PrintCell>
                    <PrintCell last={i === components.length - 1} bold>{c.portions}</PrintCell>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <footer className="mt-auto flex items-start gap-6 rounded-[12px] bg-nourish-deep p-4 text-beige">
            <div className="flex-1">
              <div className="kicker mb-1 opacity-70">{t('prod.special')}</div>
              {/* Says where the allergen detail actually lives. It used to
                  render "Allergens: Packing labels", which is the two message
                  keys concatenated and means nothing to a chef reading it at
                  05.10. The production report carries no per-line allergen
                  text — it aggregates portions — so this block points at the
                  artifact that does rather than inventing one. */}
              <div className="text-sm leading-relaxed">{t('prod.allergen_note')}</div>
            </div>
            <div className="text-right">
              <div className="kicker opacity-70">{t('prod.checked')}</div>
              <div className="mt-2 text-sm">___________________</div>
              <div className="mt-1 text-xs opacity-70">{t('prod.head_chef')}</div>
            </div>
          </footer>
        </article>
      </State>
    </div>
  )
}

function PrintStat({ label, value, inverted }: { label: string; value: number; inverted?: boolean }) {
  return (
    <div
      className={`rounded-[12px] p-4 ${
        inverted ? 'bg-nourish-deep text-beige' : 'border border-nourish-deep'
      }`}
    >
      <div className={`kicker ${inverted ? 'opacity-70' : 'opacity-60'}`}>{label}</div>
      <div className="font-display text-3xl font-bold">{value}</div>
    </div>
  )
}

/** The print table's header cell — INVERTED against the screen's (docs/10
 *  §4.6): deep fill, beige ink, because the sheet itself is beige. */
function PrintHead({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-nourish-deep px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-beige">
      {children}
    </div>
  )
}

function PrintCell({
  children, last, bold,
}: { children: React.ReactNode; last?: boolean; bold?: boolean }) {
  return (
    <div
      className={`px-3 py-2 ${bold ? 'font-semibold' : ''}`}
      style={last ? undefined : { borderBottom: '1px solid rgba(28,61,52,0.2)' }}
    >
      {children}
    </div>
  )
}

/** The wall-clock time in the operating timezone — the sheet says when it was
 *  taken, and a server-local or browser-local time would be a different hour
 *  for a kitchen in Jakarta (CLAUDE.md §10). */
export function printedAt(now = new Date()): string {
  return new Intl.DateTimeFormat('id-ID', {
    timeZone: 'Asia/Jakarta', hour: '2-digit', minute: '2-digit', hour12: false,
  }).format(now) + ' WIB'
}
