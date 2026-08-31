import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiFailure, loadSession, request } from '../lib/api'
import { Money as MoneyView, SearchBox, State } from '../components/ui'
import { AppBar, BottomBar, ChipRow, DayChips, Phone, dayLong, humanLeft } from '../components/mobile'
import { useCart } from '../lib/cart'
import { useI18n, useT } from '../lib/i18n'
import { MealPhoto } from '../components/MealPhoto'
import LanguageSelector from '../components/LanguageSelector'

/** Artboard 01 — the published menu calendar.
 *
 * Two things on this screen are rules rather than decoration, and both are
 * computed in Asia/Jakarta rather than the browser's zone (CLAUDE.md §10):
 * the 15.00 cut-off countdown, and which dates are orderable at all.
 *
 * The card price is a QUOTE at the cart's current size, not a stored number.
 * The tier is worked out on the whole basket (docs/02 B-11), so a price shown
 * per card is only true for the basket it was quoted against — which is why it
 * re-quotes when the cart changes rather than being cached per meal.
 */

export type Meal = {
  id: string
  service_date: string
  diet_type: string
  diet_type_id: string
  slot: string
  slot_id: string
  name?: string
  qty_capacity?: number
  qty_reserved: number
  hero_photo_key?: string
  items: { food_name: string; item_role: string; portion_size: string }[]
  nutrition: {
    calories_kcal: number
    protein_mg: number
    fat_mg: number
    carbohydrate_mg: number
    fibre_mg: number
    sodium_mg: number
    complete: boolean
  }
  /** Union across the components, never null. Both stored names travel; the
   *  allergen table has no Chinese column, so zh reads the English one. */
  allergens: { code: string; name_id: string; name_en: string }[]
}

type Quote = { unit_price: string; line_total: string; tier: string }

export default function Menu() {
  const t = useT()
  const { locale } = useI18n()
  const cart = useCart()
  const [meals, setMeals] = useState<Meal[]>([])
  const [q, setQ] = useState('')
  const [diet, setDiet] = useState('')
  const [day, setDay] = useState('')
  const [prices, setPrices] = useState<Record<string, Quote>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const from = serviceDateWIB()
    const to = addDays(from, 7)
    request<Meal[]>(`/menu?from=${from}&to=${to}${q ? `&q=${encodeURIComponent(q)}` : ''}`)
      .then(setMeals)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('menu.load_failed')))
      .finally(() => setLoading(false))
  }, [q]) // eslint-disable-line react-hooks/exhaustive-deps

  // The diet chips come from what is actually PUBLISHED, not from the diet
  // table: a chip for a diet with no meal this week is a dead filter.
  const diets = useMemo(() => {
    const seen = new Map<string, string>()
    for (const m of meals) if (!seen.has(m.diet_type_id)) seen.set(m.diet_type_id, m.diet_type)
    return [...seen.entries()].map(([id, name]) => ({ id, name }))
  }, [meals])

  const days = useMemo(
    () => [...new Set(meals.map((m) => m.service_date))].sort(),
    [meals],
  )

  useEffect(() => {
    if (days.length && !days.includes(day)) setDay(days[0]!)
  }, [days, day])

  // One quote per diet type, at the cart's CURRENT size, so the price on a
  // card is the price this basket would actually pay.
  useEffect(() => {
    if (diets.length === 0) return
    const qty = Math.max(1, cart.totalMeals)
    const date = day || days[0] || serviceDateWIB()
    let live = true
    Promise.all(diets.map(async (d) => {
      try {
        const qt = await request<Quote>(
          `/quote?diet_type_id=${d.id}&qty=${qty}&date=${date}`)
        return [d.id, qt] as const
      } catch {
        return null // a missing price hides the figure; it never blocks the menu
      }
    })).then((rs) => {
      if (!live) return
      setPrices(Object.fromEntries(rs.filter((r): r is readonly [string, Quote] => !!r)))
    })
    return () => { live = false }
  }, [diets, cart.totalMeals, day]) // eslint-disable-line react-hooks/exhaustive-deps

  const shown = useMemo(
    () => meals.filter((m) =>
      (!day || m.service_date === day) && (!diet || m.diet_type_id === diet)),
    [meals, day, diet],
  )

  const session = loadSession()
  const unverified = session !== null && !session.email_verified

  return (
    <Phone>
      <AppBar back={false} title="Evermore" trailing={<LanguageSelector />} />

      <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
        <div>
          <h2 className="text-2xl leading-tight">{t('c01.title')}</h2>
          {days.length > 0 && (
            <p className="text-sm text-beige-deep">
              {t('c01.published_until', dayLong(days[days.length - 1]!, localeTag(locale)))}
            </p>
          )}
        </div>

        {unverified && (
          <div className="note-emph" role="status">
            {t('menu.verify_email')} <strong>{session.email}</strong>.
          </div>
        )}

        {diets.length > 1 && (
          <ChipRow
            label={t('cal.diet')}
            options={[{ id: '', name: t('ui.all') }, ...diets]}
            value={diet}
            onChange={setDiet}
          />
        )}

        {days.length > 0 && <DayChips days={days} value={day} onChange={setDay} />}

        <CutoffNote />

        <SearchBox value={q} onChange={setQ} resultCount={shown.length}
                   placeholder={t('menu.search_placeholder')} />

        <State loading={loading} error={error} empty={shown.length === 0}
               emptyText={t('menu.empty')}>
          <div className="flex flex-col gap-3">
            {shown.map((m) => (
              <MealCard key={m.id} meal={m} price={prices[m.diet_type_id]} />
            ))}
          </div>
        </State>
      </div>

      {cart.totalMeals > 0 && (
        <BottomBar
          kicker={t('c03.subtotal', cart.totalMeals)}
          total={<CartTotal />}
        >
          <Link to="/cart" className="btn-primary">{t('c01.cart')}</Link>
        </BottomBar>
      )}
    </Phone>
  )
}

/** The card from artboard 01: photo band, slot kicker, dish, nutrition line,
 *  then price and action on one baseline. */
function MealCard({ meal, price }: { meal: Meal; price?: Quote }) {
  const t = useT()
  const cart = useCart()
  const soldOut = meal.qty_capacity !== undefined && meal.qty_reserved >= meal.qty_capacity
  const qty = cart.qtyOf(meal.id)
  const title = meal.name ?? meal.items[0]?.food_name ?? ''

  const snapshot = {
    mealID: meal.id, name: title, slot: meal.slot,
    serviceDate: meal.service_date, dietType: meal.diet_type,
    dietTypeID: meal.diet_type_id, photoKey: meal.hero_photo_key,
  }

  return (
    <article className="overflow-hidden rounded border border-rule">
      <Link to={`/menu/${meal.id}`} className="block no-underline">
        <MealPhoto photoKey={meal.hero_photo_key} diet={meal.diet_type}
                   alt={title} className="h-24" />
      </Link>
      <div className="flex flex-col gap-2 p-4">
        <span className="kicker-emph">{meal.slot} · {meal.diet_type}</span>
        <Link to={`/menu/${meal.id}`} className="no-underline">
          <h3 className="text-xl leading-tight text-beige">{title}</h3>
        </Link>
        <p className="text-sm text-beige-deep">
          {meal.items.length} {t('cal.components')} · {meal.nutrition.calories_kcal} kkal
          {' · '}{Math.round(meal.nutrition.protein_mg / 1000)} g {t('c02.protein').toLowerCase()}
        </p>
        <div className="flex flex-wrap items-end justify-between gap-3 pt-1">
          <div>
            {price ? (
              <>
                <div className="font-display text-xl font-bold">
                  <MoneyView formatted={price.unit_price} />
                </div>
                <div className="text-xs text-beige-deep">{t('c01.incl_tax')}</div>
              </>
            ) : (
              <div className="text-xs text-beige-deep">—</div>
            )}
          </div>
          {soldOut ? (
            <span className="pill-full">{t('menu.sold_out')}</span>
          ) : qty > 0 ? (
            <Link to="/cart" className="btn-ghost">{qty} · {t('c01.cart')}</Link>
          ) : (
            <button type="button" className="btn-primary"
                    onClick={() => cart.add(snapshot)}>
              {t('c01.add')}
            </button>
          )}
        </div>
      </div>
    </article>
  )
}

/** The running total in the bottom bar, quoted on the whole basket. */
function CartTotal() {
  const cart = useCart()
  const [total, setTotal] = useState<string | null>(null)

  useEffect(() => {
    const first = cart.lines[0]
    if (!first) { setTotal(null); return }
    let live = true
    request<Quote>(
      `/quote?diet_type_id=${first.dietTypeID}&qty=${cart.totalMeals}&date=${first.serviceDate}`,
    ).then((q) => { if (live) setTotal(q.line_total) }).catch(() => setTotal(null))
    return () => { live = false }
  }, [cart.totalMeals, cart.lines])

  return <>{total ? <MoneyView formatted={total} /> : '—'}</>
}

/** The cut-off callout from 01. Counts to 15.00 WIB, not to 15.00 local. */
function CutoffNote() {
  const t = useT()
  const { locale } = useI18n()
  const [left, setLeft] = useState(() => msToCutoffWIB())

  useEffect(() => {
    const id = window.setInterval(() => setLeft(msToCutoffWIB()), 30_000)
    return () => window.clearInterval(id)
  }, [])

  return (
    <div className="note-emph flex items-center gap-3">
      <span aria-hidden="true" className="text-lg text-ember-light">◷</span>
      <p className="m-0">{t('c01.cutoff', humanLeft(left, localeTag(locale)))}</p>
    </div>
  )
}

// ── Time, in the operating zone ─────────────────────────────────────────────

export function serviceDateWIB(now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Jakarta', year: 'numeric', month: '2-digit', day: '2-digit',
  }).format(now)
}

export function msToCutoffWIB(now = new Date()): number {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: 'Asia/Jakarta', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  }).formatToParts(now)
  const get = (k: string) => Number(parts.find((p) => p.type === k)?.value ?? 0)
  const secs = get('hour') * 3600 + get('minute') * 60 + get('second')
  const d = 15 * 3600 - secs
  return (d > 0 ? d : d + 86400) * 1000
}

export function addDays(iso: string, n: number): string {
  const [y, m, d] = iso.split('-').map(Number)
  const at = new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1, 12) + n * 86400000)
  return at.toISOString().slice(0, 10)
}

export function localeTag(l: string): string {
  return l === 'id' ? 'id-ID' : l === 'zh' ? 'zh-Hans' : 'en'
}
