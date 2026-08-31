import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiFailure, Page, request } from '../lib/api'
import { State } from '../components/ui'
import { useT } from '../lib/i18n'

/** S1 — the daily dashboard (docs/10 §4.10).
 *
 * Every figure on this screen is for ONE service date in Asia/Jakarta, and the
 * date is computed in that zone rather than read off the browser: a staff
 * member opening this from a laptop still on UTC would otherwise see
 * yesterday's kitchen between 00:00 and 07:00 WIB (CLAUDE.md §10).
 */

type SlotLoad = {
  slot_id: string
  alias: string
  slot_time: string
  quota: number
  used: number
  available: boolean
}

type Kitchen = {
  id: string
  code: string
  name: string
  is_active: boolean
  slots: SlotLoad[]
}

type QueueItem = {
  payment_id: string
  order_code: string
  waiting_minutes: number
}

type CoverageRow = { attempts: number }
type SalesRow = { gross: string; gross_idr: number; meals: number; orders: number }
type Delivery = { id: string; status: string }

/** The service date in the operating timezone, as YYYY-MM-DD.
 *
 * `en-CA` because it formats as ISO — the one locale that does — so this needs
 * no month-name table and no zero-padding of its own.
 */
export function serviceDateWIB(now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Jakarta',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(now)
}

export default function AdminDashboard() {
  const t = useT()
  const [kitchens, setKitchens] = useState<Kitchen[]>([])
  const [queue, setQueue] = useState<QueueItem[]>([])
  const [gaps, setGaps] = useState<CoverageRow[]>([])
  const [sales, setSales] = useState<SalesRow[]>([])
  const [deliveries, setDeliveries] = useState<Delivery[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const date = serviceDateWIB()

  useEffect(() => {
    Promise.all([
      request<Kitchen[]>(`/admin/kitchens?date=${date}`),
      // The queue drives both the "needs verification" tile and the action
      // list, so it is fetched once rather than counted twice.
      request<Page<QueueItem>>('/admin/payments?status=SUBMITTED'),
      request<CoverageRow[]>('/admin/reports/coverage'),
      // Revenue needs PermReportFinancial, which not every staff role has.
      // Its failure must cost the TILE, not the dashboard — an ops manager
      // without finance rights still needs the capacity grid. Same for the
      // delivery counts, which need PermDeliveryRead.
      request<SalesRow[]>(`/admin/reports/sales?from=${date}&to=${date}`).catch(() => []),
      request<Page<Delivery>>(`/admin/deliveries?from=${date}&to=${date}`)
        .catch(() => ({ items: [], total: 0, page: 1, page_size: 0 })),
    ])
      .then(([k, p, c, sl, dl]) => {
        setKitchens(k)
        setQueue(p.items)
        setGaps(c)
        setSales(Array.isArray(sl) ? sl : [])
        setDeliveries(Array.isArray(dl?.items) ? dl.items : [])
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('dash.load_failed')))
      .finally(() => setLoading(false))
  }, [date]) // eslint-disable-line react-hooks/exhaustive-deps

  // Every slot column in the grid, in the order the kitchens list them. Built
  // from the data rather than hard-coded to 07.00/11.30/12.00/18.00: slots are
  // a configurable table, and a fifth one must appear here without a deploy.
  const slotCols = useMemo(() => {
    const seen = new Map<string, string>()
    for (const k of kitchens) {
      for (const s of k.slots) if (!seen.has(s.slot_id)) seen.set(s.slot_id, s.slot_time)
    }
    return [...seen.entries()].map(([id, time]) => ({ id, time })).sort(
      (a, b) => a.time.localeCompare(b.time),
    )
  }, [kitchens])

  const mealsToday = useMemo(
    () => kitchens.reduce((n, k) => n + k.slots.reduce((m, s) => m + s.used, 0), 0),
    [kitchens],
  )
  const oldest = useMemo(
    () => queue.reduce((max, q) => Math.max(max, q.waiting_minutes), 0),
    [queue],
  )
  const oldestRef = useMemo(
    () => queue.find((q) => q.waiting_minutes === oldest)?.order_code ?? '',
    [queue, oldest],
  )
  const outOfRange = useMemo(() => gaps.reduce((n, g) => n + g.attempts, 0), [gaps])

  // Summed in INTEGER rupiah and formatted once at the end. No float touches
  // this (CLAUDE.md §4), and the "jt" abbreviation the artboard uses is a
  // display transform of that integer, never a rounded value carried forward.
  const grossIDR = useMemo(
    () => sales.reduce((n, r) => n + (r.gross_idr ?? 0), 0), [sales])
  const delivered = useMemo(
    () => deliveries.filter((d) => d.status === 'DELIVERED').length, [deliveries])
  const enRoute = useMemo(
    () => deliveries.filter((d) => d.status === 'OUT_FOR_DELIVERY' || d.status === 'PICKED_UP').length,
    [deliveries])
  const fullSlots = useMemo(
    () => kitchens.flatMap((k) => k.slots.filter((s) => s.available && s.used >= s.quota)
      .map((s) => ({ kitchen: k.name, slot: s.slot_time }))),
    [kitchens],
  )

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-6 border-b border-rule pb-5">
        <div>
          <h1 className="mb-1">{longDateWIB(date)}</h1>
          <p className="text-sm text-beige-deep">{t('dash.subtitle')}</p>
        </div>
        {/* The cut-off countdown is emphasis, not alarm: it wears the ember
            ring and the ember kicker together, so the state is never carried
            by colour alone (docs/10 §2.4 rule 4). */}
        <CutoffCountdown />
      </div>

      <State loading={loading} error={error}>
        <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
          <div className="stat">
            <div className="stat-label">{t('dash.meals_today')}</div>
            <div className="stat-value">{mealsToday}</div>
            <div className="stat-sub">{t('dash.meals_sub', kitchens.length, slotCols.length)}</div>
          </div>
          <div className="stat">
            <div className="stat-label">{t('dash.deliveries')}</div>
            <div className="stat-value">{deliveries.length}</div>
            <div className="stat-sub">{t('dash.deliveries_sub', delivered, enRoute)}</div>
          </div>
          <div className={queue.length > 0 ? 'stat-emph' : 'stat'}>
            <div className="stat-label">{t('dash.needs_verify')}</div>
            <div className="stat-value">{queue.length}</div>
            <div className="stat-sub">
              {queue.length > 0 ? `${t('dash.action_oldest')} ${oldest} ${t('pay.minutes')}` : '—'}
            </div>
          </div>
          <div className="stat">
            <div className="stat-label">{t('dash.revenue')}</div>
            <div className="stat-value">{shortIDR(grossIDR)}</div>
            <div className="stat-sub">{t('dash.revenue_sub')}</div>
          </div>
          <div className={outOfRange > 0 ? 'stat-danger' : 'stat'}>
            <div className="stat-label">{t('dash.out_of_range')}</div>
            <div className="stat-value">{outOfRange}</div>
            <div className="stat-sub">{t('dash.out_of_range_sub')}</div>
          </div>
        </div>

        <div className="grid gap-6 2xl:grid-cols-[1.6fr_1fr]">
          {/* ── Capacity per kitchen and slot ───────────────────────────────
              A framed grid (docs/10 §4.6). The column count comes from the
              data, so the template string is built rather than written. */}
          <section className="gtable min-w-0">
            <div className="flex items-baseline justify-between gap-4 border-b border-edge px-4 py-3">
              <h2 className="text-xl">{t('dash.capacity')}</h2>
              <span className="text-xs text-beige-deep">{t('dash.capacity_legend')}</span>
            </div>
            {/* .gtable hides its overflow so the frame's corners stay round,
                which silently CLIPPED the last slot column once a fourth slot
                existed. The scroll lives here, on its own container, so the
                page body still never scrolls sideways. */}
            <div className="overflow-x-auto">
            <div
              className="grid min-w-[34rem]"
              style={{ gridTemplateColumns: `1.5fr repeat(${slotCols.length}, minmax(5rem, 1fr))` }}
            >
              <div className="gtable-head">{t('dash.kitchen')}</div>
              {slotCols.map((s) => (
                <div key={s.id} className="gtable-head">{s.time}</div>
              ))}
              {kitchens.map((k, ki) => {
                const last = ki === kitchens.length - 1
                const cell = `gtable-cell ${last ? 'is-last' : ''}`
                return (
                  <div key={k.id} className="contents">
                    <div className={`${cell} font-semibold`}>
                      {k.name} <span className="font-medium text-beige-deep">{k.code}</span>
                    </div>
                    {slotCols.map((col) => {
                      const s = k.slots.find((x) => x.slot_id === col.id)
                      if (!s || !s.available) {
                        return (
                          <div key={col.id} className={`${cell} text-beige-deep`}>
                            {t('dash.closed')}
                          </div>
                        )
                      }
                      const full = s.used >= s.quota
                      return (
                        <div key={col.id} className={cell}>
                          {full ? (
                            // docs/10 §4.1 #1 — the capacity pill is berry
                            // DEEP with a berry-light ring, not a berry-light
                            // fill: 7.89 rather than 3.40.
                            <span className="pill-full">{s.used} / {s.quota}</span>
                          ) : (
                            <>{s.used} / {s.quota}</>
                          )}
                        </div>
                      )
                    })}
                  </div>
                )
              })}
            </div>
            </div>
          </section>

          {/* ── Needs action ─────────────────────────────────────────────── */}
          <section className="panel min-w-0">
            <div className="panel-head"><h2 className="text-xl">{t('dash.actions')}</h2></div>
            <div className="px-5 py-2">
              {queue.length === 0 && fullSlots.length === 0 ? (
                <p className="py-6 text-sm text-beige-deep">{t('ui.empty')}</p>
              ) : (
                <>
                  {queue.length > 0 && (
                    <ActionRow
                      title={`${queue.length} ${t('dash.action_proofs')}`}
                      detail={`${t('dash.action_oldest')} ${oldestRef} · ${oldest} ${t('pay.minutes')}`}
                      to="/admin/payments"
                      cta={t('dash.verify')}
                      primary
                    />
                  )}
                  {fullSlots.map((f) => (
                    <ActionRow
                      key={`${f.kitchen}-${f.slot}`}
                      title={`${t('cal.at_capacity')} — ${f.kitchen} ${f.slot}`}
                      detail={t('cov.manual_note')}
                      to="/admin/coverage"
                      cta={t('bo.coverage')}
                    />
                  ))}
                </>
              )}
            </div>
          </section>
        </div>
      </State>
    </div>
  )
}

function ActionRow({
  title, detail, to, cta, primary,
}: {
  title: string; detail: string; to: string; cta: string; primary?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-rule py-3 last:border-b-0">
      <div className="min-w-0">
        <div className="text-sm font-semibold">{title}</div>
        <div className="truncate text-xs text-beige-deep">{detail}</div>
      </div>
      <Link to={to} className={primary ? 'btn-primary' : 'btn-ghost'}>{cta}</Link>
    </div>
  )
}

/** The countdown to tomorrow's order cut-off.
 *
 * 15.00 WIB is the rule (docs/02). Computed against Asia/Jakarta rather than
 * the browser's zone, and re-rendered once a second — cleared on unmount, so
 * navigating away does not leave a timer running against a dead component.
 */
function CutoffCountdown() {
  const t = useT()
  const [left, setLeft] = useState(() => msToCutoff())

  useEffect(() => {
    const id = window.setInterval(() => setLeft(msToCutoff()), 1000)
    return () => window.clearInterval(id)
  }, [])

  return (
    <div className="flex items-center gap-4 rounded-full border border-ember-light px-5 py-2">
      <span className="kicker-emph">{t('dash.cutoff')}</span>
      {/* A live region would announce every second, which is unusable; the
          value is polite-off and the label carries the meaning. */}
      <span className="font-display text-xl font-bold tabular-nums">{hhmmss(left)}</span>
    </div>
  )
}

/** Milliseconds until the next 15:00 in Asia/Jakarta. */
export function msToCutoff(now = new Date()): number {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Asia/Jakarta', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  }).formatToParts(now)
  const get = (k: string) => Number(parts.find((p) => p.type === k)?.value ?? 0)
  const secsNow = get('hour') * 3600 + get('minute') * 60 + get('second')
  const cutoff = 15 * 3600
  const delta = cutoff - secsNow
  return (delta > 0 ? delta : delta + 24 * 3600) * 1000
}

function hhmmss(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000))
  const p = (n: number) => String(n).padStart(2, '0')
  return `${p(Math.floor(s / 3600))}:${p(Math.floor((s % 3600) / 60))}:${p(s % 60)}`
}

/** "Senin, 1 September 2026" in the reader's locale, from a YYYY-MM-DD date.
 *
 * Parsed as UTC noon rather than `new Date(s)`: a bare date string is parsed
 * as UTC midnight, which in a negative-offset zone renders as the day before.
 */
export function longDateWIB(iso: string, locale = 'id-ID'): string {
  const [y, m, d] = iso.split('-').map(Number)
  const at = new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1, 12))
  return new Intl.DateTimeFormat(locale, {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC',
  }).format(at)
}

/** "24,1 jt" — the artboard's abbreviated rupiah for a stat tile.
 *
 * A DISPLAY transform of an integer, applied once at the edge. The integer is
 * what was summed and what any export carries; this never feeds arithmetic
 * (CLAUDE.md §4). Indonesian uses a comma as the decimal separator.
 */
export function shortIDR(v: number): string {
  if (v >= 1_000_000_000) return `${(v / 1_000_000_000).toFixed(1).replace('.', ',')} m`
  if (v >= 1_000_000) return `${(v / 1_000_000).toFixed(1).replace('.', ',')} jt`
  if (v >= 1_000) return `${Math.round(v / 1_000)} rb`
  return String(v)
}
