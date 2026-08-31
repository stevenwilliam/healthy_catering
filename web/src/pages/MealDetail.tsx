import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ApiFailure, request } from '../lib/api'
import { Money as MoneyView, State } from '../components/ui'
import { AppBar, BottomBar, Phone, Stepper, dayLong } from '../components/mobile'
import { MealPhoto } from '../components/MealPhoto'
import { useCart } from '../lib/cart'
import { useI18n, useT } from '../lib/i18n'
import type { MessageKey } from '../lib/messages'
import { addDays, localeTag, serviceDateWIB, type Meal } from './Menu'

/** Artboard 02 — one meal, in full.
 *
 * The nutrition panel is the SUM of the components, and it says so. That is
 * not a caption, it is the honest description of where the numbers come from:
 * a meal has no panel of its own, so an incomplete component makes the whole
 * total an estimate — which the panel then says out loud rather than quietly
 * under-reporting (docs/01, and nutrition.Facts.Complete).
 *
 * Allergens are words, never a colour or an icon alone. This is a regulated
 * claim on food, and the one place where "colour is never the only signal"
 * stops being a style rule and starts being a safety one.
 */

type Quote = { unit_price: string; line_total: string; tier: string }

export default function MealDetail() {
  const { id = '' } = useParams()
  const t = useT()
  const { locale } = useI18n()
  const nav = useNavigate()
  const cart = useCart()
  const [meal, setMeal] = useState<Meal | null>(null)
  const [qty, setQty] = useState(1)
  const [quote, setQuote] = useState<Quote | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Read from the same published window the menu uses rather than a per-meal
  // endpoint: it is one request either way, and it guarantees the detail page
  // can only ever show a meal the menu would have offered.
  useEffect(() => {
    const from = serviceDateWIB()
    request<Meal[]>(`/menu?from=${from}&to=${addDays(from, 7)}`)
      .then((ms) => {
        const found = ms.find((m) => m.id === id) ?? null
        setMeal(found)
        if (!found) setError(t('menu.empty'))
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('menu.load_failed')))
      .finally(() => setLoading(false))
  }, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => { setQty(Math.max(1, cart.qtyOf(id))) }, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Quoted on what the basket would become, because the tier is worked out on
  // the whole order — the price of "this meal" alone is not a real number.
  useEffect(() => {
    if (!meal) return
    const basket = cart.totalMeals - cart.qtyOf(id) + qty
    let live = true
    request<Quote>(
      `/quote?diet_type_id=${meal.diet_type_id}&qty=${Math.max(1, basket)}&date=${meal.service_date}`,
    ).then((q) => { if (live) setQuote(q) }).catch(() => setQuote(null))
    return () => { live = false }
  }, [meal, qty]) // eslint-disable-line react-hooks/exhaustive-deps

  const grouped = useMemo(() => {
    if (!meal) return []
    const order = ['MAIN', 'SIDE', 'DESSERT', 'DRINK']
    return [...meal.items].sort(
      (a, b) => order.indexOf(a.item_role) - order.indexOf(b.item_role))
  }, [meal])

  const title = meal?.name ?? meal?.items[0]?.food_name ?? ''
  const soldOut = !!meal && meal.qty_capacity !== undefined
    && meal.qty_reserved >= meal.qty_capacity

  return (
    <Phone>
      <AppBar title={title || t('c01.title')} />
      <State loading={loading} error={error}>
        {meal && (
          <>
            <MealPhoto photoKey={meal.hero_photo_key} diet={meal.diet_type}
                       alt={title} className="h-40 shrink-0" />

            <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
              <div>
                <span className="kicker-emph">
                  {meal.diet_type} · {dayLong(meal.service_date, localeTag(locale))} · {meal.slot}
                </span>
                <h2 className="mt-2 text-2xl leading-tight">{title}</h2>
              </div>

              {/* ── Isi meal ─────────────────────────────────────────────── */}
              <section>
                <h3 className="kicker mb-2">{t('c02.contents')}</h3>
                <dl className="grid grid-cols-[6rem_1fr] gap-x-3 gap-y-2 text-sm">
                  {grouped.map((it) => (
                    <div key={it.food_name} className="contents">
                      <dt className={`text-xs font-bold uppercase tracking-wider ${
                        it.item_role === 'MAIN' ? 'text-ember-light' : 'text-beige-deep'}`}>
                        {roleLabel(t, it.item_role)}
                      </dt>
                      <dd className={it.item_role === 'MAIN' ? 'font-semibold' : ''}>
                        {it.food_name}
                        {it.portion_size && <span className="text-beige-deep"> {it.portion_size}</span>}
                      </dd>
                    </div>
                  ))}
                </dl>
              </section>

              {/* ── Informasi gizi ───────────────────────────────────────── */}
              <section className="gtable">
                <div className="flex items-baseline justify-between border-b border-edge px-4 py-2">
                  <h3 className="font-display text-lg font-semibold">{t('c02.nutrition')}</h3>
                  <span className="text-xs text-beige-deep">{t('c02.per_portion')}</span>
                </div>
                <div className="grid grid-cols-2">
                  <Nutrient label={t('c02.energy')} value={`${meal.nutrition.calories_kcal} kkal`} />
                  <Nutrient label={t('c02.protein')} value={grams(meal.nutrition.protein_mg)} />
                  <Nutrient label={t('c02.carbs')} value={grams(meal.nutrition.carbohydrate_mg)} />
                  <Nutrient label={t('c02.fat')} value={grams(meal.nutrition.fat_mg)} />
                  <Nutrient label={t('c02.fibre')} value={grams(meal.nutrition.fibre_mg)} />
                  <Nutrient label={t('c02.sodium')} value={`${meal.nutrition.sodium_mg} mg`} last />
                </div>
                <p className="m-0 border-t border-edge px-4 py-2 text-xs text-beige-deep">
                  {t('c02.sum_note', meal.items.length)}
                </p>
              </section>

              {/* An incomplete panel says so. Under-reporting a calorie count
                  to someone eating to a target is the failure mode here. */}
              {!meal.nutrition.complete && (
                <div className="note-emph">{t('c02.estimated_note')}</div>
              )}

              {/* ── Alergen ──────────────────────────────────────────────── */}
              <section className="flex flex-wrap items-center gap-2">
                <h3 className="kicker">{t('c02.allergens')}</h3>
                {meal.allergens.length === 0 ? (
                  <span className="text-sm text-beige-deep">{t('c02.no_allergens')}</span>
                ) : (
                  meal.allergens.map((a) => (
                    <span key={a.code}
                          className="rounded border border-berry-light px-3 py-1 text-sm font-semibold">
                      {locale === 'id' ? a.name_id : a.name_en}
                    </span>
                  ))
                )}
              </section>
            </div>

            <BottomBar
              kicker={quote ? `${quote.tier}` : t('c01.summary')}
              total={quote ? <MoneyView formatted={quote.unit_price} /> : <>—</>}
            >
              <Stepper value={qty} onChange={setQty} min={1} max={99}
                       label={t('menu.qty')} />
              <button
                type="button"
                className="btn-primary"
                disabled={soldOut}
                onClick={() => {
                  cart.setQty({
                    mealID: meal.id, name: title, slot: meal.slot,
                    serviceDate: meal.service_date, dietType: meal.diet_type,
                    dietTypeID: meal.diet_type_id, photoKey: meal.hero_photo_key,
                  }, qty)
                  nav('/cart')
                }}
              >
                {soldOut ? t('menu.sold_out') : t('c01.add')}
              </button>
            </BottomBar>
          </>
        )}
      </State>
    </Phone>
  )
}

function Nutrient({ label, value, last }: { label: string; value: string; last?: boolean }) {
  return (
    <div className={`px-4 py-2 odd:border-r odd:border-edge ${last ? '' : 'border-b border-edge'}`}>
      <div className="text-xs font-semibold text-beige-deep">{label}</div>
      <div className="font-display text-lg font-bold">{value}</div>
    </div>
  )
}

/** Milligrams to grams with one decimal, the Indonesian way (comma).
 *
 * Nutrition is stored in integer milligrams (docs/02 D-24) precisely so this
 * is the only place a decimal point appears, and it is a DISPLAY decimal — no
 * arithmetic is done on it.
 */
function grams(mg: number): string {
  return `${(mg / 1000).toFixed(1).replace('.', ',')} g`
}

/** The four item roles the schema allows, mapped to catalogue keys.
 *  `satisfies` keeps this honest: a typo in a key is a compile error, and a
 *  new role in the CHECK constraint shows up here as a missing case. */
const ROLE_KEY = {
  MAIN: 'c02.role_main',
  SIDE: 'c02.role_side',
  DESSERT: 'c02.role_dessert',
  DRINK: 'c02.role_drink',
} satisfies Record<string, MessageKey>

function roleLabel(t: ReturnType<typeof useT>, role: string): string {
  return t(ROLE_KEY[role as keyof typeof ROLE_KEY] ?? ROLE_KEY.SIDE)
}
