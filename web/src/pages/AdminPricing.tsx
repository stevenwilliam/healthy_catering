import { useEffect, useMemo, useState } from 'react'
import { ApiFailure, Page, request } from '../lib/api'
import { Money as MoneyView, SearchBox, State, SubmitButton } from '../components/ui'
import ExportCsv from '../components/ExportCsv'
import { useT } from '../lib/i18n'
import { serviceDateWIB } from './AdminDashboard'

/** S3 — the four price tables, and the resolver you can point at them.
 *
 * The artboard's whole argument is that a price is not a number but a
 * RESOLUTION: scope, table, tier and row. The right-hand panel exists so the
 * question "why did this customer pay that" is answered from the record rather
 * than by reading the resolver's source (docs/01, docs/10 §4).
 *
 * Prices are integer rupiah end to end. Nothing on this screen does arithmetic
 * on them — the base/tax split is displayed exactly as the server computed it
 * (CLAUDE.md §4).
 */

type PriceRow = {
  id: string
  scope_key: string
  scope: string
  diet_type_id?: string
  diet_type?: string
  tier_id?: string
  tier?: string
  package?: string
  price_idr: number
  price: string
  promo_label?: string
  valid_from: string
  valid_to?: string
  is_active: boolean
}

type Tier = { ID: string; Label: string; MinQty: number; MaxQty: number | null; Active: boolean }
type DietType = { ID: string; Name: string }
type Quote = {
  unit_price: string
  line_total: string
  normal_price: string
  is_promo: boolean
  promo_label?: string
  tier: string
  savings?: string
}

/** The four tables, in the artboard's order. The keys are the API's `:table`
 *  path segment, so this list is also the route. */
const TABLES = [
  { key: 'meal_normal', label: 'price.meal_normal' },
  { key: 'meal_promo', label: 'price.meal_promo' },
  { key: 'package_normal', label: 'price.pkg_normal' },
  { key: 'package_promo', label: 'price.pkg_promo' },
] as const

export default function AdminPricing() {
  const t = useT()
  const [table, setTable] = useState<string>(TABLES[0].key)
  const [rows, setRows] = useState<PriceRow[]>([])
  const [tiers, setTiers] = useState<Tier[]>([])
  const [diets, setDiets] = useState<DietType[]>([])
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    Promise.all([
      request<Page<PriceRow>>(`/admin/prices/${table}?q=${encodeURIComponent(q)}`),
      request<Tier[]>('/admin/price-tiers'),
      request<Page<DietType>>('/admin/diet-types'),
    ])
      .then(([p, ti, d]) => { setRows(p.items); setTiers(ti); setDiets(d.items) })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('price.load_failed')))
      .finally(() => setLoading(false))
  }, [table, q]) // eslint-disable-line react-hooks/exhaustive-deps

  // The tier ladder must cover 1..∞ with no gap and no overlap. Checked here
  // as well as on the server because the screen is where it gets fixed, and a
  // gap is invisible in a list of rows.
  const ladder = useMemo(() => checkTiers(tiers), [tiers])

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-end justify-between gap-4">
        <h1>{t('price.title')}</h1>
        <p className="max-w-xl text-sm text-beige-deep">{t('price.tax_note')}</p>
      </div>

      {/* The four tables as chips (docs/10 §4.9). */}
      <div className="mb-5 flex flex-wrap gap-2">
        {TABLES.map((tb) => (
          <button
            key={tb.key}
            className={table === tb.key ? 'chip-on' : 'chip-off'}
            onClick={() => setTable(tb.key)}
          >
            {t(tb.label)}
          </button>
        ))}
      </div>

      <div className="grid gap-6 xl:grid-cols-[1fr_400px] xl:items-start">
        <div>
          <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
            <div className="min-w-[16rem] flex-1"><SearchBox value={q} onChange={setQ} resultCount={rows.length} /></div>
            {/* Every data grid exports, pipe-delimited, exactly what is on
                screen — filters and search included (CLAUDE.md §7). */}
            <ExportCsv path={`/admin/prices/${table}`} params={{ q }} filename={`prices-${table}`} />
          </div>

          <State loading={loading} error={error} empty={rows.length === 0}>
            <div className="overflow-x-auto">
              <div className="gtable min-w-[46rem]">
                <div
                  className="grid"
                  style={{ gridTemplateColumns: '1.1fr 0.9fr 1.2fr 1.4fr 0.8fr' }}
                >
                  <div className="gtable-head">{t('price.tier')}</div>
                  <div className="gtable-head">{t('price.range')}</div>
                  <div className="gtable-head">{t('price.incl_tax')}</div>
                  {/* This column used to be headed "Base + tax" while the
                      cell under it rendered the promo label — the header and
                      the data disagreed. PriceRow carries no base/tax split:
                      that division is integer arithmetic the server performs
                      when an ORDER is priced, not a property of a price row,
                      which is exactly what the note above the table says. So
                      the column now says what it shows. */}
                  <div className="gtable-head">{t('price.promo')}</div>
                  <div className="gtable-head">{t('price.active')}</div>
                  {rows.map((r, i) => {
                    const last = i === rows.length - 1
                    const cell = `gtable-cell ${last ? 'is-last' : ''}`
                    const tier = tiers.find((x) => x.ID === r.tier_id)
                    return (
                      <div key={r.id} className="contents">
                        <div className={`${cell} font-semibold ${r.is_active ? '' : 'text-beige-deep'}`}>
                          {r.tier ?? '—'}
                        </div>
                        <div className={`${cell} ${r.is_active ? '' : 'text-beige-deep'}`}>
                          {tier ? rangeOf(tier) : '—'}
                        </div>
                        <div className={cell}>
                          <span className="font-display text-lg font-bold">
                            <MoneyView formatted={r.price} amount={r.price_idr} />
                          </span>
                          {r.valid_to && (
                            <span className="ml-2 text-xs text-beige-deep">
                              · {t('price.valid')} → {r.valid_to}
                            </span>
                          )}
                        </div>
                        <div className={`${cell} text-beige-deep`}>{r.promo_label || '—'}</div>
                        <div className={cell}>
                          {r.is_active
                            ? <span className="pill-ok">{t('price.yes')}</span>
                            : <span className="pill-archived">{t('price.archived')}</span>}
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>

            {/* Tier ladder validation (docs/10 §4.5) — info when it is sound,
                warning when there is a hole. */}
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <div className={ladder.ok ? 'note-info' : 'note-warn'}>
                <div className={ladder.ok ? 'kicker-info mb-1' : 'kicker-warn mb-1'}>
                  {ladder.ok ? t('price.tier_ok') : t('price.tier_gap')}
                </div>
                <p className="m-0">{ladder.message}</p>
              </div>
              <div className="note-info">
                <div className="kicker-info mb-1">{t('price.trace')}</div>
                <p className="m-0">{t('price.trace_note')}</p>
              </div>
            </div>
          </State>
        </div>

        {/* ── The resolver ─────────────────────────────────────────────────
            Runs the REAL /quote endpoint, not a local reimplementation. A
            second copy of the ladder in TypeScript would drift from the
            server's, and the whole point of the panel is to show what the
            server will actually charge. */}
        <Resolver diets={diets} />
      </div>
    </div>
  )
}

function Resolver({ diets }: { diets: DietType[] }) {
  const t = useT()
  const [diet, setDiet] = useState('')
  const [qty, setQty] = useState(6)
  const [date, setDate] = useState(serviceDateWIB())
  const [quote, setQuote] = useState<Quote | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => { if (!diet && diets.length) setDiet(diets[0]!.ID) }, [diets, diet])

  async function run() {
    setRunning(true)
    setError(null)
    setQuote(null)
    try {
      setQuote(await request<Quote>(
        `/quote?diet_type_id=${diet}&qty=${qty}&date=${date}`,
      ))
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('price.load_failed'))
    } finally {
      setRunning(false)
    }
  }

  return (
    <section className="panel">
      <div className="panel-head"><h2 className="text-xl">{t('price.resolver')}</h2></div>
      <div className="flex flex-col gap-4 p-5">
        <div>
          <label className="label" htmlFor="res-diet">{t('cal.diet')}</label>
          <select id="res-diet" className="field" value={diet} onChange={(e) => setDiet(e.target.value)}>
            {diets.map((d) => <option key={d.ID} value={d.ID}>{d.Name}</option>)}
          </select>
        </div>
        <div>
          <label className="label" htmlFor="res-qty">{t('price.qty')}</label>
          <input
            id="res-qty" type="number" min={1} max={999} className="field"
            value={qty}
            onChange={(e) => setQty(Math.max(1, Number(e.target.value) || 1))}
          />
        </div>
        <div>
          <label className="label" htmlFor="res-date">{t('price.order_date')}</label>
          <input id="res-date" type="date" className="field" value={date}
                 onChange={(e) => setDate(e.target.value)} />
        </div>
        <SubmitButton pending={running} type="button" onClick={run} className="btn-ghost">
          {t('price.run_resolver')}
        </SubmitButton>

        {error && <p className="error" role="alert">{error}</p>}

        {quote && (
          <>
            <div className="h-px bg-rule" />
            <div>
              <div className="kicker mb-1">{t('price.result')}</div>
              <div className="font-display text-2xl font-bold">
                <MoneyView formatted={quote.unit_price} />
              </div>
              <div className="text-sm text-beige-deep">
                {t('price.per_meal')} · {t('prod.total')} <MoneyView formatted={quote.line_total} />
              </div>
            </div>
            {/* The trace. Mid green as a FILL here, and every string on it is
                at least 14px — this is a panel, not a bar, so the 19px bar
                rule does not apply, but the tint is chosen so beige stays
                comfortable (docs/10 §2.7). */}
            <div className="flex flex-col gap-2 rounded-card border border-rule bg-beige/5 p-4 text-sm">
              <div className="kicker">{t('price.trace')}</div>
              <TraceRow label={t('price.tier')} value={quote.tier} />
              {quote.is_promo && <TraceRow label={t('price.meal_promo')} value={quote.promo_label ?? '—'} />}
              {!quote.is_promo && <TraceRow label={t('price.meal_promo')} value={t('ui.none')} />}
              {quote.savings && <TraceRow label={t('menu.savings')} value={quote.savings} />}
            </div>
          </>
        )}
      </div>
    </section>
  )
}

function TraceRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-3">
      <span className="text-beige-deep">{label}</span>
      <span className="font-semibold">{value}</span>
    </div>
  )
}

/** rangeOf renders a tier band the way the artboard does — "10 – ∞". */
function rangeOf(t: Tier): string {
  return `${t.MinQty} – ${t.MaxQty ?? '∞'}`
}

/** checkTiers looks for a gap or an overlap in the active tier ladder.
 *
 * The server refuses a bad ladder, but this screen is where it gets repaired,
 * and "tier 3 starts at 11 while tier 2 ends at 9" is invisible when you are
 * reading rows one at a time.
 */
export function checkTiers(tiers: Tier[]): { ok: boolean; message: string } {
  const active = tiers.filter((t) => t.Active).sort((a, b) => a.MinQty - b.MinQty)
  if (active.length === 0) return { ok: false, message: 'No active tier covers any quantity.' }
  if (active[0]!.MinQty !== 1) {
    return { ok: false, message: `The ladder starts at ${active[0]!.MinQty}, not 1.` }
  }
  for (let i = 1; i < active.length; i++) {
    const prev = active[i - 1]!
    const cur = active[i]!
    if (prev.MaxQty === null) {
      return { ok: false, message: `${prev.Label} is unbounded but ${cur.Label} follows it.` }
    }
    if (cur.MinQty <= prev.MaxQty) {
      return { ok: false, message: `${prev.Label} and ${cur.Label} overlap at ${cur.MinQty}.` }
    }
    if (cur.MinQty > prev.MaxQty + 1) {
      return { ok: false, message: `Nothing covers ${prev.MaxQty + 1}–${cur.MinQty - 1}.` }
    }
  }
  if (active[active.length - 1]!.MaxQty !== null) {
    return { ok: false, message: 'The top tier is bounded, so large orders resolve to nothing.' }
  }
  return { ok: true, message: 'No gap and no overlap from 1 upward. Tiers count meals, not dishes.' }
}
