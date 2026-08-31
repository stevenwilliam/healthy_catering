import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiFailure, Page, newIdempotencyKey, request } from '../lib/api'
import { Money as MoneyView, State, SubmitButton } from '../components/ui'
import { AppBar, BottomBar, Phone } from '../components/mobile'
import { useT } from '../lib/i18n'

/** Artboard M2 — buying prepaid meal credit.
 *
 * Three things on this screen are commitments rather than marketing, and all
 * three are stated before the button, not after it: what a portion costs in
 * this bundle, how long the credit lasts, and that it is neither refundable
 * nor extendable. A customer who only reads the price and the CTA has still
 * seen the expiry, because it sits inside the card they are choosing.
 *
 * The saving is computed from integer rupiah the server sent — a subtraction
 * of two prices, never a percentage of one (CLAUDE.md §4).
 */

type Pkg = {
  id: string
  name: string
  description: string
  meal_credits: number
  validity_days: number
}

type PublicPrices = {
  packages: {
    name: string; meal_credits: number; validity_days: number; price_idr?: number
  }[]
  prices: { tier_min_qty: number; unit_price_idr: number }[]
}

export default function Packages() {
  const t = useT()
  const nav = useNavigate()
  const [available, setAvailable] = useState<Pkg[]>([])
  const [pub, setPub] = useState<PublicPrices | null>(null)
  const [chosen, setChosen] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [buying, setBuying] = useState(false)

  useEffect(() => {
    Promise.all([
      request<Page<Pkg>>('/packages'),
      request<PublicPrices>('/public/prices'),
    ])
      .then(([a, p]) => {
        // Same reasoning as the cart: an odd /public/prices body costs the
        // savings line, never the list of packages.
        const items = Array.isArray(a?.items) ? a.items : []
        setAvailable(items)
        setPub(Array.isArray(p?.packages) && Array.isArray(p?.prices) ? p : null)
        setChosen(items[1]?.id ?? items[0]?.id ?? '')
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('packages.load_failed')))
      .finally(() => setLoading(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // The single-portion price is the FIRST tier's — that is what a package is
  // being compared against, and it is what "hemat X dibanding harga satuan"
  // means. Missing prices simply hide the saving rather than guessing at it.
  const singleIDR = useMemo(
    () => pub?.prices?.length
      ? [...pub.prices].sort((a, b) => a.tier_min_qty - b.tier_min_qty)[0]?.unit_price_idr
      : undefined,
    [pub],
  )

  const priceOf = (p: Pkg) =>
    pub?.packages?.find((x) => x.name === p.name)?.price_idr

  // The artboard marks the middle bundle "Paling laris". It is the one most
  // people buy, and it is marked from position rather than from a flag the
  // schema does not have — recorded as such rather than pretending it is data.
  const bestseller = available.length >= 3 ? available[1]?.id : undefined

  const selected = available.find((p) => p.id === chosen)
  const selectedPrice = selected ? priceOf(selected) : undefined

  async function buy() {
    if (!chosen) return
    setBuying(true)
    setError(null)
    try {
      const out = await request<{ order_id: string }>(`/packages/${chosen}/buy`, {
        method: 'POST',
        body: {},
        idempotencyKey: newIdempotencyKey(),
      })
      nav(`/orders/${out.order_id}`)
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('packages.buy_failed'))
    } finally {
      setBuying(false)
    }
  }

  return (
    <Phone>
      <AppBar
        title={t('m2.title')}
        trailing={<Link to="/credits" className="btn-ghost">{t('c06.title')}</Link>}
      />

      <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
        <div>
          <h2 className="text-2xl leading-tight">{t('m2.headline')}</h2>
          <p className="mt-1 text-sm text-beige-deep">{t('m2.sub')}</p>
        </div>

        <State loading={loading} error={error} empty={available.length === 0}>
          <div className="flex flex-col gap-3" role="radiogroup" aria-label={t('m2.title')}>
            {available.map((p) => {
              const price = priceOf(p)
              const perPortion = price !== undefined && p.meal_credits > 0
                // Integer division, floored: the per-portion figure shown can
                // never overstate what a portion costs.
                ? Math.floor(price / p.meal_credits)
                : undefined
              const saving = price !== undefined && singleIDR !== undefined
                ? singleIDR * p.meal_credits - price
                : undefined
              const on = p.id === chosen

              return (
                <button
                  key={p.id}
                  type="button"
                  role="radio"
                  aria-checked={on}
                  onClick={() => setChosen(p.id)}
                  className={`relative rounded-card p-4 text-left ${
                    on ? 'border-2 border-beige' : 'border border-rule'}`}
                >
                  {p.id === bestseller && (
                    <span className="pill-emph absolute -top-3 right-4">{t('m2.bestseller')}</span>
                  )}
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="font-display text-2xl font-bold leading-tight">
                        {t('m2.portions', p.meal_credits)}
                      </div>
                      {perPortion !== undefined && (
                        <div className="mt-1 text-sm text-beige-deep">
                          {t('m2.per_portion', idr(perPortion))}
                        </div>
                      )}
                    </div>
                    <div className="text-right">
                      <div className="font-display text-xl font-bold">
                        {price !== undefined ? <MoneyView amount={price} /> : '—'}
                      </div>
                      <div className="text-xs text-beige-deep">
                        {t('m2.valid_days', p.validity_days)}
                      </div>
                    </div>
                  </div>
                  {saving !== undefined && saving > 0 && (
                    <div className="mt-3 rounded bg-ocean-light/10 px-3 py-2 text-sm font-medium">
                      {t('m2.saving', idr(saving))}
                    </div>
                  )}
                </button>
              )
            })}
          </div>

          {/* Non-refundable and time-limited, said before the button. */}
          <div className="note-emph mt-4">{t('m2.terms')}</div>
        </State>

        {error && <p className="error" role="alert">{error}</p>}
      </div>

      {selected && (
        <BottomBar
          kicker={t('m2.portions', selected.meal_credits)}
          total={selectedPrice !== undefined ? <MoneyView amount={selectedPrice} /> : <>—</>}
        >
          <SubmitButton pending={buying} type="button" onClick={buy}>
            {t('m2.buy')}
          </SubmitButton>
        </BottomBar>
      )}
    </Phone>
  )
}

function idr(v: number): string {
  return `Rp ${v.toLocaleString('id-ID')}`
}
