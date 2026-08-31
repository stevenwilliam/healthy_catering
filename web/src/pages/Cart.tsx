import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { request } from '../lib/api'
import { Money as MoneyView } from '../components/ui'
import { AppBar, BottomBar, Phone, Stepper } from '../components/mobile'
import { MealPhoto } from '../components/MealPhoto'
import { useCart } from '../lib/cart'
import { useT } from '../lib/i18n'
import { dayLong } from '../components/mobile'
import { useI18n } from '../lib/i18n'
import { localeTag } from './Menu'

/** Artboard 03 — the cart, and the tier ladder that explains its price.
 *
 * The ladder is the point of this screen. Tiering is worked out on the WHOLE
 * order (docs/02 B-11), so a per-line price is not a fact a customer can act
 * on — what they can act on is "four more portions and every one drops". The
 * artboard shows the bands, marks the one in force, and says what the next one
 * costs to reach. That is the screen doing arithmetic so the customer does not
 * have to guess at it.
 *
 * Every figure comes from /quote or the public price list. Nothing on this
 * page multiplies money (CLAUDE.md §4) — the "add N more" count is a QUANTITY
 * subtraction, which is integer and is not money.
 */

type Quote = {
  unit_price: string
  line_total: string
  normal_price: string
  is_promo: boolean
  promo_label?: string
  tier: string
  savings?: string
}

type PublicPrices = {
  tiers: { label: string; min_qty: number; max_qty?: number }[]
  prices: { diet_slug: string; diet_name: string; tier_label: string; tier_min_qty: number; unit_price_idr: number }[]
}

export default function Cart() {
  const t = useT()
  const { locale } = useI18n()
  const nav = useNavigate()
  const cart = useCart()
  const [quote, setQuote] = useState<Quote | null>(null)
  const [ladder, setLadder] = useState<PublicPrices | null>(null)

  useEffect(() => {
    // Shape-checked, not trusted. A price endpoint that answers with an
    // unexpected body — a proxy error page, an older deploy, a partial
    // failure that returns {} — must cost the LADDER, not the cart. Reading
    // `.map` off undefined here white-screens a customer mid-checkout, which
    // is the worst possible moment for this screen to disappear.
    request<PublicPrices>('/public/prices')
      .then((p) => setLadder(
        Array.isArray(p?.tiers) && Array.isArray(p?.prices) ? p : null))
      .catch(() => setLadder(null))
  }, [])

  useEffect(() => {
    const first = cart.lines[0]
    if (!first) { setQuote(null); return }
    let live = true
    request<Quote>(
      `/quote?diet_type_id=${first.dietTypeID}&qty=${cart.totalMeals}&date=${first.serviceDate}`,
    ).then((q) => { if (live) setQuote(q) }).catch(() => setQuote(null))
    return () => { live = false }
  }, [cart.totalMeals, cart.lines])

  // The bands, with the one in force marked, and the price each band charges
  // for the diet already in the basket.
  const bands = useMemo(() => {
    if (!ladder) return []
    const slug = cart.lines[0]?.dietType ?? ''
    return ladder.tiers.map((tier) => {
      const row = ladder.prices.find(
        (p) => p.tier_label === tier.label && (p.diet_name === slug || !slug))
      return {
        label: tier.label,
        min: tier.min_qty,
        max: tier.max_qty,
        priceIDR: row?.unit_price_idr,
        active: cart.totalMeals >= tier.min_qty
          && (tier.max_qty === undefined || cart.totalMeals <= tier.max_qty),
      }
    })
  }, [ladder, cart.totalMeals, cart.lines])

  // The next band down, and how many portions away it is. Integer quantity
  // arithmetic — never money.
  const nextBand = useMemo(() => {
    const upcoming = bands.filter((b) => b.min > cart.totalMeals)
    const cheapest = upcoming.find((b) => b.priceIDR !== undefined)
    if (!cheapest) return null
    return { need: cheapest.min - cart.totalMeals, priceIDR: cheapest.priceIDR! }
  }, [bands, cart.totalMeals])

  if (cart.lines.length === 0) {
    return (
      <Phone>
        <AppBar title={t('c03.title')} />
        <div className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
          <p className="text-beige-deep">{t('c03.empty')}</p>
          <Link to="/menu" className="btn-primary">{t('c01.title')}</Link>
        </div>
      </Phone>
    )
  }

  return (
    <Phone>
      <AppBar title={t('c03.title')} />

      <div className="flex flex-1 flex-col gap-5 overflow-y-auto p-4">
        {/* ── Lines ─────────────────────────────────────────────────────── */}
        <ul className="m-0 flex list-none flex-col gap-3 p-0">
          {cart.lines.map((l) => (
            <li key={l.mealID} className="flex gap-3 border-b border-edge pb-3">
              <div className="w-16 shrink-0 overflow-hidden rounded">
                <MealPhoto photoKey={l.photoKey} diet={l.dietType} alt={l.name}
                           className="h-16" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="font-display text-lg font-semibold leading-tight">{l.name}</div>
                <div className="mt-1 text-sm text-beige-deep">
                  {dayLong(l.serviceDate, localeTag(locale))} · {l.slot} · {l.dietType}
                </div>
                <div className="mt-2 flex items-center justify-between gap-3">
                  <Stepper
                    size="sm"
                    value={l.qty}
                    min={0}
                    onChange={(n) => cart.setQty(l, n)}
                    label={`${t('menu.qty')} — ${l.name}`}
                  />
                  {/* Per-line money is deliberately NOT shown. The unit price
                      depends on the whole basket, so a line total here would
                      be a number that changes when an unrelated line changes —
                      which reads as a bug even when it is correct. */}
                  <button type="button" className="min-h-touch text-sm underline"
                          onClick={() => cart.remove(l.mealID)}>
                    {t('c03.remove')}
                  </button>
                </div>
              </div>
            </li>
          ))}
        </ul>

        {/* ── The tier ladder ───────────────────────────────────────────── */}
        {bands.length > 0 && (
          <section className="gtable">
            <h2 className="border-b border-edge px-4 py-3 font-display text-xl font-semibold">
              {t('c03.per_portion_price')}
            </h2>
            {bands.map((b) => (
              <div
                key={b.label}
                className={`flex justify-between px-4 py-2 text-sm ${
                  b.active ? 'bg-beige font-bold text-nourish-deep' : 'text-beige-deep'}`}
              >
                <span>
                  {t('c03.tier_band', b.max === undefined ? `${b.min}+` : `${b.min}–${b.max}`)}
                  {b.active && <> · {t('c03.tier_active')}</>}
                </span>
                <span>{b.priceIDR !== undefined ? <MoneyView amount={b.priceIDR} /> : '—'}</span>
              </div>
            ))}
          </section>
        )}

        {nextBand && (
          <div className="note-info flex items-center justify-between gap-3">
            <p className="m-0">
              {t('c03.upsell', nextBand.need, idr(nextBand.priceIDR))}
            </p>
            <Link to="/menu" className="btn-ghost whitespace-nowrap">{t('c03.topup')}</Link>
          </div>
        )}

        {/* ── Totals ───────────────────────────────────────────────────── */}
        {quote && (
          <dl className="m-0 flex flex-col gap-2 text-sm">
            <Row label={t('c03.subtotal', cart.totalMeals)}>
              <MoneyView formatted={quote.line_total} />
            </Row>
            {quote.is_promo && (
              <Row label={quote.promo_label ?? ''}>
                <s className="text-beige-deep"><MoneyView formatted={quote.normal_price} /></s>
              </Row>
            )}
            {quote.savings && (
              <Row label={t('c03.saving_label')}><MoneyView formatted={quote.savings} /></Row>
            )}
            {/* Delivery and VAT are stated on the ORDER, not guessed here:
                shipping depends on the routed kitchen and VAT is already
                inside the price. Checkout returns both. */}
            <Row label={t('c03.shipping', cart.dropCount)} muted>—</Row>
          </dl>
        )}
      </div>

      <BottomBar
        kicker={t('c03.total')}
        total={quote ? <MoneyView formatted={quote.line_total} /> : <>—</>}
      >
        <button type="button" className="btn-primary" onClick={() => nav('/checkout')}>
          {t('c03.checkout')}
        </button>
      </BottomBar>
    </Phone>
  )
}

function Row({ label, children, muted }: {
  label: string; children: React.ReactNode; muted?: boolean
}) {
  return (
    <div className={`flex justify-between gap-3 ${muted ? 'text-beige-deep' : ''}`}>
      <dt>{label}</dt>
      <dd className="m-0 font-semibold">{children}</dd>
    </div>
  )
}

/** Whole rupiah, grouped the Indonesian way. Formatting only — the value is
 *  an integer from the server and stays one. */
function idr(v: number): string {
  return `Rp ${v.toLocaleString('id-ID')}`
}
