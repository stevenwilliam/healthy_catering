import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiFailure, loadSession, newIdempotencyKey, request } from '../lib/api'
import { Money as MoneyView, SearchBox, State, SubmitButton } from '../components/ui'

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
      .catch((e) => setError(e instanceof ApiFailure ? e.message : 'Gagal memuat menu.'))
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
      setError(e instanceof ApiFailure ? e.message : 'Tidak dapat membuat pesanan.')
    } finally {
      setPlacing(false)
    }
  }

  const session = loadSession()
  const unverified = session !== null && !session.email_verified

  return (
    <div>
      <h1 className="text-3xl mb-6">Menu minggu ini</h1>

      {/* Tell them BEFORE they build a cart. Discovering the rule at checkout,
          after choosing meals and an address, is the worst moment to learn it
          (99 §8: a disabled state explains itself). */}
      {unverified && (
        <p className="card mb-4 border-brown-deep" role="status">
          Konfirmasi email Anda sebelum pesanan pertama. Kami sudah mengirim tautannya
          ke <strong>{session.email}</strong>.
        </p>
      )}
      <SearchBox value={q} onChange={setQ} placeholder="Cari lauk, mis. ayam" resultCount={meals.length} />

      <State loading={loading} error={error} empty={meals.length === 0}
             emptyText="Menu belum dipublikasikan untuk minggu ini.">
        {/* Room for the sticky summary, or it sits ON TOP of the last card —
            which on a phone hides a meal the customer is trying to order. */}
        <div className={`grid gap-4 sm:grid-cols-2 lg:grid-cols-3 ${totalMeals > 0 ? 'pb-64 sm:pb-40' : ''}`}>
          {meals.map((m) => {
            const soldOut = m.qty_capacity !== undefined && m.qty_reserved >= (m.qty_capacity ?? 0)
            return (
              <article key={m.id} className="card">
                <p className="text-sm text-ink-muted">{m.service_date} · {m.slot} · {m.diet_type}</p>
                <h2 className="text-lg mt-1">{m.name ?? m.items[0]?.food_name}</h2>
                <ul className="mt-2 mb-3 list-disc pl-5 text-sm">
                  {m.items.map((it) => <li key={it.food_name}>{it.food_name}</li>)}
                </ul>
                <p className="flex flex-wrap gap-2 mb-3">
                  <span className="badge">{m.nutrition.calories_kcal} kkal</span>
                  <span className="badge">{Math.round(m.nutrition.protein_mg / 1000)} g protein</span>
                  {!m.nutrition.complete && <span className="badge bg-ember-light">perkiraan</span>}
                </p>
                {soldOut ? (
                  <p className="text-sm text-ink-muted">Habis untuk tanggal ini.</p>
                ) : (
                  <label className="flex items-center gap-2 text-sm">
                    Jumlah
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
        <aside
          className="card fixed inset-x-4 bottom-4 z-10 shadow-lg sm:sticky sm:inset-x-auto sm:mt-8"
          aria-label="Ringkasan pesanan"
        >
          <h2 className="text-lg mb-2">{totalMeals} porsi</h2>
          {quote && (
            <p className="mb-3 text-sm">
              {quote.is_promo && (
                <>
                  <s className="text-ink-muted mr-2">{quote.normal_price}</s>
                  <span className="badge mr-2">{quote.promo_label}</span>
                </>
              )}
              <MoneyView formatted={quote.unit_price} /> per porsi · tarif {quote.tier}
              {quote.savings && <> · hemat {quote.savings}</>}
            </p>
          )}

          <label className="label" htmlFor="addr">Antar ke</label>
          <select id="addr" className="field mb-3" value={addressID}
                  onChange={(e) => setAddressID(e.target.value)}>
            {addresses.map((a) => (
              <option key={a.ID} value={a.ID}>{a.Label} — {a.AddressLine}</option>
            ))}
          </select>

          {addresses.length === 0 ? (
            <p className="text-sm text-ink-muted">
              Tambahkan alamat pengiriman dulu — kami perlu titik peta Anda.
            </p>
          ) : (
            <SubmitButton pending={placing} onClick={placeOrder} type="button"
                          disabled={unverified}>
              Pesan sekarang
            </SubmitButton>
          )}
          {error && <p className="error" role="alert">{error}</p>}
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
