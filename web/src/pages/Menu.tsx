import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiFailure, loadSession, newIdempotencyKey, request } from '../lib/api'
import { Money as MoneyView, SearchBox, State, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'

type Meal = {
  id: string
  service_date: string
  diet_type: string
  slot: string
  name?: string
  qty_capacity?: number
  qty_reserved: number
  items: { food_name: string; item_role: string }[]
  nutrition: { calories_kcal: number; protein_mg: number; complete: boolean }
}

type Address = { ID: string; Label: string; AddressLine: string }

type Quote = {
  unit_price: string
  line_total: string
  normal_price: string
  is_promo: boolean
  promo_label?: string
  tier: string
  savings?: string
}

export default function Menu() {
  const t = useT()
  const nav = useNavigate()
  const [meals, setMeals] = useState<Meal[]>([])
  const [addresses, setAddresses] = useState<Address[]>([])
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // The cart is a map of meal id -> quantity. Deliberately simple: an order can
  // span several dates, and the tier is worked out on the TOTAL (docs/02 B-11).
  const [cart, setCart] = useState<Record<string, number>>({})
  const [addressID, setAddressID] = useState('')
  const [quote, setQuote] = useState<Quote | null>(null)
  const [placing, setPlacing] = useState(false)

  const totalMeals = useMemo(
    () => Object.values(cart).reduce((a, b) => a + b, 0),
    [cart],
  )

  useEffect(() => {
    const from = new Date().toISOString().slice(0, 10)
    const to = new Date(Date.now() + 7 * 864e5).toISOString().slice(0, 10)
    Promise.all([
      request<Meal[]>(`/menu?from=${from}&to=${to}${q ? `&q=${encodeURIComponent(q)}` : ''}`),
      request<Address[]>('/addresses'),
    ])
      .then(([m, a]) => {
        setMeals(m)
        setAddresses(a)
        if (a.length && !addressID) setAddressID(a[0]!.ID)
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('menu.load_failed')))
      .finally(() => setLoading(false))
  }, [q]) // eslint-disable-line react-hooks/exhaustive-deps

  // Re-quote whenever the total changes: the price depends on the whole order,
  // so showing a per-line price alone would be misleading.
  useEffect(() => {
    if (totalMeals === 0) { setQuote(null); return }
    const first = meals.find((m) => cart[m.id])
    if (!first) return
    request<Quote>(
      `/quote?diet_type_id=${dietIdOf(meals, first.id)}&qty=${totalMeals}&date=${first.service_date}`,
    ).then(setQuote).catch(() => setQuote(null))
  }, [totalMeals]) // eslint-disable-line react-hooks/exhaustive-deps

  async function placeOrder() {
    setPlacing(true)
    setError(null)
    try {
      const lines = Object.entries(cart)
        .filter(([, qty]) => qty > 0)
        .map(([scheduled_meal_id, qty]) => ({ scheduled_meal_id, qty, address_id: addressID }))

      const out = await request<{ order_id: string }>('/orders', {
        method: 'POST',
        body: { lines },
        // A double-tapped button must create ONE order (PROMPT §14).
        idempotencyKey: newIdempotencyKey(),
      })
      nav(`/orders/${out.order_id}`)
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('menu.order_failed'))
    } finally {
      setPlacing(false)
    }
  }

  const session = loadSession()
  const unverified = session !== null && !session.email_verified

  return (
    <div>
      <h1 className="text-3xl mb-6">{t('menu.title')}</h1>

      {/* Tell them BEFORE they build a cart. Discovering the rule at checkout,
          after choosing meals and an address, is the worst moment to learn it
          (99 §8: a disabled state explains itself). */}
      {unverified && (
        <p className="card mb-4 border-brown-deep" role="status">
          {t('menu.verify_email')} <strong>{session.email}</strong>.
        </p>
      )}
      <SearchBox value={q} onChange={setQ} placeholder={t('menu.search_placeholder')} resultCount={meals.length} />

      <State loading={loading} error={error} empty={meals.length === 0}
             emptyText={t('menu.empty')}>
        {/* Room for the sticky summary, or it sits ON TOP of the last card —
            which on a phone hides a meal the customer is trying to order. */}
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {meals.map((m) => {
            const soldOut = m.qty_capacity !== undefined && m.qty_reserved >= (m.qty_capacity ?? 0)
            return (
              // The meal card from artboard 01: a kicker for the slot, the
              // dish in Erode, the nutrition line as quiet meta, and the
              // price and action on one baseline at the foot.
              <article key={m.id} className="card flex flex-col gap-3">
                <span className="kicker-emph">{m.slot} · {m.diet_type}</span>
                <h2 className="text-xl leading-tight">{m.name ?? m.items[0]?.food_name}</h2>
                <p className="text-sm text-beige-deep">
                  {m.items.length} {t('cal.components')} · {m.nutrition.calories_kcal}{' '}
                  {t('menu.kcal')} · {Math.round(m.nutrition.protein_mg / 1000)}{' '}
                  {t('menu.protein')}
                  {!m.nutrition.complete && <> · {t('menu.estimated')}</>}
                </p>
                <ul className="list-disc pl-5 text-sm text-beige-deep">
                  {m.items.map((it) => <li key={it.food_name}>{it.food_name}</li>)}
                </ul>
                <p className="text-xs text-beige-deep">{m.service_date}</p>
                {soldOut ? (
                  // Sold out is a word AND a pill, never a colour alone.
                  <p className="mt-auto"><span className="pill-full">{t('menu.sold_out')}</span></p>
                ) : (
                  <label className="mt-auto flex items-center gap-2 text-sm">
                    {t('menu.qty')}
                    <input
                      type="number" min={0} max={999} className="field w-24"
                      value={cart[m.id] ?? 0}
                      onChange={(e) =>
                        setCart({ ...cart, [m.id]: Math.max(0, Number(e.target.value) || 0) })
                      }
                    />
                  </label>
                )}
              </article>
            )
          })}
        </div>
      </State>

      {totalMeals > 0 && (
        // The sticky bottom bar from artboards 01/03 (docs/10 §4.10): the
        // running total on the left, the action on the right, both on the
        // mid-green bar. The total is Erode at 24px because every string on a
        // bar has to clear WCAG's large-text threshold — beige there is only
        // 3.93 (docs/10 §2.7), which is exactly why it is not 15px Inter.
        <aside className="bottombar -mx-4 mt-8" aria-label={t('menu.summary_aria')}>
          <div className="min-w-0">
            <div className="kicker text-beige">
              {totalMeals} {t('menu.portions')}
              {quote && <> · {t('menu.tier')} {quote.tier}</>}
            </div>
            <div className="bottombar-total">
              {quote
                ? <MoneyView formatted={quote.line_total} />
                : <>—</>}
            </div>
            {quote && (
              <div className="text-xs text-beige">
                {quote.is_promo && (
                  <s className="mr-2 opacity-80">{quote.normal_price}</s>
                )}
                <MoneyView formatted={quote.unit_price} /> {t('menu.per_portion')}
                {quote.savings && <> · {t('menu.savings')} {quote.savings}</>}
              </div>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-3">
            <label className="sr-only" htmlFor="addr">{t('menu.deliver_to')}</label>
            <select id="addr" className="field max-w-[14rem]" value={addressID}
                    onChange={(e) => setAddressID(e.target.value)}>
              {addresses.map((a) => (
                <option key={a.ID} value={a.ID}>{a.Label} — {a.AddressLine}</option>
              ))}
            </select>
            {addresses.length === 0 ? (
              <p className="text-bar font-bold text-beige">{t('menu.need_address')}</p>
            ) : (
              <SubmitButton pending={placing} onClick={placeOrder} type="button"
                            disabled={unverified}>
                {t('menu.order_now')}
              </SubmitButton>
            )}
          </div>
          {error && <p className="error basis-full" role="alert">{error}</p>}
        </aside>
      )}
    </div>
  )
}

// The menu response carries the diet type name but the quote wants the id; the
// list endpoint returns it, so read it from the same record.
function dietIdOf(meals: Meal[], mealID: string): string {
  const m = meals.find((x) => x.id === mealID) as unknown as { diet_type_id?: string }
  return m?.diet_type_id ?? ''
}
