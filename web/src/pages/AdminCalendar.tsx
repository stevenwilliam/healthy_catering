import { useEffect, useMemo, useState } from 'react'
import { ApiFailure, Page, request } from '../lib/api'
import { SearchBox, State, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'
import { serviceDateWIB } from './AdminDashboard'

/** S2 — the menu schedule calendar (docs/10 §4.10).
 *
 * One global calendar across kitchens: the same meal is published for every
 * kitchen on a date and slot, which is why this screen has no kitchen filter
 * and the capacity shown is the sum across them (docs/01, and the artboard's
 * own "kalender global lintas dapur" note).
 */

type Meal = {
  id: string
  service_date: string
  diet_type_id: string
  diet_type: string
  slot_id: string
  slot: string
  name?: string
  qty_capacity?: number
  qty_reserved: number
  status: string
  items: { food_name: string }[]
  nutrition: { calories_kcal: number; complete: boolean }
}

type DietType = { ID: string; Name: string }

const DAY_MS = 86_400_000

export default function AdminCalendar() {
  const t = useT()
  const [weekStart, setWeekStart] = useState(() => mondayOf(serviceDateWIB()))
  const [meals, setMeals] = useState<Meal[]>([])
  const [diets, setDiets] = useState<DietType[]>([])
  const [diet, setDiet] = useState('')
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [publishing, setPublishing] = useState(false)

  const days = useMemo(
    () => Array.from({ length: 6 }, (_, i) => addDays(weekStart, i)),
    [weekStart],
  )
  const from = days[0]!
  const to = days[days.length - 1]!

  function load() {
    setLoading(true)
    Promise.all([
      request<Meal[]>(
        `/admin/calendar?from=${from}&to=${to}` +
        `${diet ? `&diet_type_id=${diet}` : ''}${q ? `&q=${encodeURIComponent(q)}` : ''}`,
      ),
      request<Page<DietType>>('/admin/diet-types'),
    ])
      .then(([m, d]) => { setMeals(m); setDiets(d.items) })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('cal.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [from, to, diet, q]) // eslint-disable-line react-hooks/exhaustive-deps

  // Rows are the slots actually scheduled this week, in time order — not a
  // hard-coded 11.30/18.00, because slots are configurable.
  const slotRows = useMemo(() => {
    const seen = new Map<string, string>()
    for (const m of meals) if (!seen.has(m.slot_id)) seen.set(m.slot_id, m.slot)
    return [...seen.entries()].map(([id, label]) => ({ id, label }))
      .sort((a, b) => a.label.localeCompare(b.label))
  }, [meals])

  const drafts = useMemo(() => meals.filter((m) => m.status !== 'PUBLISHED'), [meals])

  async function publishWeek() {
    if (drafts.length === 0) return
    setPublishing(true)
    setError(null)
    try {
      await request('/admin/calendar/publish', {
        method: 'POST',
        body: { ids: drafts.map((d) => d.id) },
      })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('cal.publish_failed'))
    } finally {
      setPublishing(false)
    }
  }

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-center justify-between gap-4">
        <h1>{t('cal.title')}</h1>
        <div className="flex flex-wrap items-center gap-3">
          <button className="btn-ghost" onClick={() => setWeekStart(addDays(weekStart, -7))}>
            {t('cal.prev_week')}
          </button>
          <span className="text-sm font-semibold">{rangeLabel(from, to)}</span>
          <button className="btn-ghost" onClick={() => setWeekStart(addDays(weekStart, 7))}>
            {t('cal.next_week')}
          </button>
          {/* A disabled control explains itself rather than being a dead box
              (99 §8): with nothing in draft there is nothing to publish, and
              the count says so. */}
          <SubmitButton
            pending={publishing}
            type="button"
            onClick={publishWeek}
            disabled={drafts.length === 0}
          >
            {t('cal.publish_week')}{drafts.length > 0 ? ` · ${drafts.length}` : ''}
          </SubmitButton>
        </div>
      </div>

      {/* Diet chips (docs/10 §4.9) and the legend. */}
      <div className="mb-4 flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="kicker mr-1">{t('cal.diet')}</span>
          <button className={diet === '' ? 'chip-on' : 'chip-off'} onClick={() => setDiet('')}>
            {t('ui.all')}
          </button>
          {diets.map((d) => (
            <button
              key={d.ID}
              className={diet === d.ID ? 'chip-on' : 'chip-off'}
              onClick={() => setDiet(d.ID)}
            >
              {d.Name}
            </button>
          ))}
        </div>
        {/* Each legend entry carries a WORD as well as a swatch — the state is
            never the colour alone (docs/10 §2.4 rule 4). */}
        <div className="flex flex-wrap items-center gap-4 text-xs text-beige-deep">
          <span className="flex items-center gap-2">
            <span className="h-3 w-3 rounded-full bg-beige" />{t('cal.published')}
          </span>
          <span className="flex items-center gap-2">
            <span className="h-3 w-3 rounded-full border-2 border-ember-light" />{t('cal.draft')}
          </span>
          <span className="flex items-center gap-2">
            <span className="h-3 w-3 rounded-full bg-berry-deep ring-1 ring-berry-light" />
            {t('cal.at_capacity')}
          </span>
          <span>{t('cal.global_note')}</span>
        </div>
      </div>

      <SearchBox value={q} onChange={setQ} resultCount={meals.length} />

      <State loading={loading} error={error}>
        {/* Horizontal scroll on its own container: a six-day grid does not fit
            a phone, and the page body must never scroll sideways. */}
        <div className="overflow-x-auto">
          <div
            className="grid min-w-[64rem] gap-3"
            style={{ gridTemplateColumns: '86px repeat(6, 1fr)' }}
          >
            <div />
            {days.map((d) => (
              <div key={d} className="text-center">
                <div className="kicker">{weekdayLabel(d)}</div>
                <div className="font-display text-xl font-bold">{dayLabel(d)}</div>
              </div>
            ))}

            {slotRows.map((slot) => (
              <div key={slot.id} className="contents">
                <div className="flex flex-col justify-center">
                  <span className="font-display text-xl font-bold">{slot.label}</span>
                </div>
                {days.map((d) => {
                  const m = meals.find((x) => x.service_date === d && x.slot_id === slot.id)
                  if (!m) {
                    return (
                      <div
                        key={d}
                        className="flex items-center justify-center rounded-card border border-rule p-4"
                      >
                        <button className="btn-quiet">{t('cal.schedule')}</button>
                      </div>
                    )
                  }
                  return <MealCell key={d} meal={m} onChanged={load} />
                })}
              </div>
            ))}

            {slotRows.length === 0 && (
              <p className="col-span-7 py-10 text-sm text-beige-deep">{t('ui.empty')}</p>
            )}
          </div>
        </div>
      </State>
    </div>
  )
}

/** One cell of the calendar.
 *
 * Published is a beige card with deep ink; draft is a dashed ember outline on
 * the ground. Both states carry the word as well as the treatment.
 */
function MealCell({ meal, onChanged }: { meal: Meal; onChanged: () => void }) {
  const t = useT()
  const [busy, setBusy] = useState(false)
  const published = meal.status === 'PUBLISHED'
  const full = meal.qty_capacity !== undefined && meal.qty_reserved >= (meal.qty_capacity ?? 0)

  async function publish() {
    setBusy(true)
    try {
      await request('/admin/calendar/publish', { method: 'POST', body: { ids: [meal.id] } })
      onChanged()
    } finally {
      setBusy(false)
    }
  }

  if (published) {
    return (
      <div className="on-sheet flex flex-col gap-2 rounded-card bg-sheet p-4 text-ink">
        <span className="text-xs font-bold uppercase tracking-wider">{t('cal.published')}</span>
        <span className="font-display text-lg font-semibold leading-tight">
          {meal.name ?? meal.items[0]?.food_name}
        </span>
        <span className="text-xs">
          {meal.items.length} {t('cal.components')} · {meal.nutrition.calories_kcal} kkal
        </span>
        <span className="mt-auto pt-1">
          {full ? (
            // docs/10 §4.1 #1 — berry DEEP with a berry-light ring.
            <span className="pill-full">
              {meal.qty_reserved} / {meal.qty_capacity} {t('cal.full')}
            </span>
          ) : (
            <span className="text-xs font-semibold">
              {meal.qty_reserved} / {meal.qty_capacity ?? '∞'} {t('cal.filled')}
            </span>
          )}
        </span>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2 rounded-card border-2 border-dashed border-ember-light p-4">
      <span className="kicker-emph">{t('cal.draft')}</span>
      <span className="font-display text-lg font-semibold leading-tight">
        {meal.name ?? meal.items[0]?.food_name}
      </span>
      <span className="text-xs text-beige-deep">
        {meal.items.length} {t('cal.components')}
        {!meal.nutrition.complete && ` · ${t('menu.estimated')}`}
      </span>
      <SubmitButton pending={busy} type="button" onClick={publish} className="btn-primary mt-auto self-start">
        {t('cal.publish')}
      </SubmitButton>
    </div>
  )
}

// ── Date helpers ────────────────────────────────────────────────────────────
// All of these work on YYYY-MM-DD strings parsed at UTC noon. A bare date
// string parsed by `new Date()` is UTC midnight, which renders as the previous
// day in any negative-offset zone — the classic off-by-one in a calendar.

function at(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1, 12))
}
function iso(d: Date): string { return d.toISOString().slice(0, 10) }

export function addDays(isoDate: string, n: number): string {
  return iso(new Date(at(isoDate).getTime() + n * DAY_MS))
}

/** The Monday of the week containing `isoDate`. */
export function mondayOf(isoDate: string): string {
  const d = at(isoDate)
  // getUTCDay: 0 = Sunday. Monday-based offset, so Sunday steps back six days
  // rather than forward one.
  const back = (d.getUTCDay() + 6) % 7
  return addDays(isoDate, -back)
}

function weekdayLabel(isoDate: string): string {
  return new Intl.DateTimeFormat('id-ID', { weekday: 'short', timeZone: 'UTC' }).format(at(isoDate))
}
function dayLabel(isoDate: string): string {
  return new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', timeZone: 'UTC' })
    .format(at(isoDate))
}
function rangeLabel(from: string, to: string): string {
  return `${dayLabel(from)} – ${dayLabel(to)} ${at(to).getUTCFullYear()}`
}
