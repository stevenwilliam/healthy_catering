import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiFailure, newIdempotencyKey, request } from '../lib/api'
import { Money as MoneyView, State, SubmitButton } from '../components/ui'
import { AppBar, BottomBar, Phone, dayLong } from '../components/mobile'
import { useCart } from '../lib/cart'
import { useI18n, useT } from '../lib/i18n'
import { localeTag } from './Menu'

/** Artboard 04 — where it goes, when, and to whom.
 *
 * The slot chips are the substance. A slot the kitchen cannot serve for THIS
 * address on THIS date is shown struck through with the reason, never hidden:
 * a control that vanishes reads as a bug, and the customer cannot tell "full"
 * from "we never deliver then". Availability comes from the same routing the
 * order will use when it commits, so the picker cannot offer something
 * checkout will then refuse.
 *
 * The map is a locator, not an editor. Dragging a pin needs the browser Maps
 * key, which the SPA does not yet receive (RUN-WHEN-BACK §B3) — so the pin
 * shows where the saved address is and sends the customer to the address form
 * to move it, rather than pretending to be draggable.
 */

/** Exactly what GET /addresses returns. `postgres.Address` carries no json
 *  tags, so these are Go field names, not snake_case — getting that wrong
 *  leaves every field undefined and the card renders blank. */
type Address = {
  ID: string
  Label: string
  RecipientName: string
  RecipientPhone: string
  AddressLine: string
  District: string
  City: string
  Latitude: number
  Longitude: number
  DriverNote: string
}

type SlotOffer = {
  slot_id: string
  alias: string
  serviceable: boolean
  kitchen_name?: string
  delivery_fee?: string
  reason?: string
}

type Quote = { line_total: string }

export default function Checkout() {
  const t = useT()
  const { locale } = useI18n()
  const nav = useNavigate()
  const cart = useCart()

  const [addresses, setAddresses] = useState<Address[]>([])
  const [addressID, setAddressID] = useState('')
  const [slots, setSlots] = useState<SlotOffer[]>([])
  const [slotID, setSlotID] = useState('')
  // Prefilled from the address's standing note: the customer sees what the
  // courier would otherwise be told, and can amend it for this drop.
  const [note, setNote] = useState('')
  const [noteTouched, setNoteTouched] = useState(false)
  const [quote, setQuote] = useState<Quote | null>(null)
  const [loading, setLoading] = useState(true)
  const [placing, setPlacing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const date = cart.lines[0]?.serviceDate ?? ''

  useEffect(() => {
    request<Address[]>('/addresses')
      .then((a) => {
        setAddresses(a)
        setAddressID(a[0]?.ID ?? '')
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('menu.load_failed')))
      .finally(() => setLoading(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const address = useMemo(
    () => addresses.find((a) => a.ID === addressID),
    [addresses, addressID],
  )

  useEffect(() => {
    if (!noteTouched) setNote(address?.DriverNote ?? '')
  }, [address, noteTouched])

  useEffect(() => {
    if (!address || !date) return
    let live = true
    request<SlotOffer[]>(
      `/delivery-slots/availability?lat=${address.Latitude}&lng=${address.Longitude}` +
      `&date=${date}&qty=${cart.totalMeals}`,
    )
      .then((o) => {
        if (!live) return
        setSlots(o)
        // Preselect the first slot that actually works, never a dead one.
        setSlotID((cur) => o.some((s) => s.slot_id === cur && s.serviceable)
          ? cur
          : o.find((s) => s.serviceable)?.slot_id ?? '')
      })
      .catch(() => setSlots([]))
    return () => { live = false }
  }, [address, date, cart.totalMeals])

  useEffect(() => {
    const first = cart.lines[0]
    if (!first) return
    request<Quote>(
      `/quote?diet_type_id=${first.dietTypeID}&qty=${cart.totalMeals}&date=${first.serviceDate}`,
    ).then(setQuote).catch(() => setQuote(null))
  }, [cart.totalMeals, cart.lines])

  const chosen = slots.find((s) => s.slot_id === slotID)

  async function place() {
    if (!addressID) return
    setPlacing(true)
    setError(null)
    try {
      const out = await request<{ order_id: string }>('/orders', {
        method: 'POST',
        body: {
          lines: cart.lines.map((l) => ({
            scheduled_meal_id: l.mealID, qty: l.qty, address_id: addressID,
          })),
          driver_note: note.trim(),
        },
        // A double-tapped checkout must create ONE order (PROMPT §14).
        idempotencyKey: newIdempotencyKey(),
      })
      cart.clear()
      nav(`/orders/${out.order_id}`)
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('menu.order_failed'))
    } finally {
      setPlacing(false)
    }
  }

  if (!loading && cart.lines.length === 0) {
    return (
      <Phone>
        <AppBar title={t('c04.title')} />
        <div className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
          <p className="text-beige-deep">{t('c03.empty')}</p>
          <Link to="/menu" className="btn-primary">{t('c01.title')}</Link>
        </div>
      </Phone>
    )
  }

  return (
    <Phone>
      <AppBar title={t('c04.title')} />

      <State loading={loading} error={null}>
        <div className="flex flex-1 flex-col overflow-y-auto">
          {/* ── The locator ─────────────────────────────────────────────── */}
          <div className="relative h-36 shrink-0 bg-bar">
            <div
              aria-hidden="true"
              className="absolute inset-0"
              style={{
                background:
                  'repeating-linear-gradient(90deg, rgba(28,61,52,0.18) 0 1px, transparent 1px 46px),' +
                  'repeating-linear-gradient(0deg, rgba(28,61,52,0.18) 0 1px, transparent 1px 46px)',
              }}
            />
            <div className="absolute left-1/2 top-1/2 flex -translate-x-1/2 -translate-y-1/2 flex-col items-center">
              <span className="h-6 w-6 rounded-full border-4 border-canvas bg-beige" />
              <span className="h-4 w-0.5 bg-canvas" />
            </div>
            <Link to="/addresses" className="btn-primary absolute bottom-3 right-3">
              {t('c04.move_pin')}
            </Link>
          </div>

          <div className="flex flex-col gap-4 p-4">
            {/* ── Address ───────────────────────────────────────────────── */}
            {addresses.length === 0 ? (
              <div className="note-warn flex items-center justify-between gap-3">
                <p className="m-0">{t('c04.no_address')}</p>
                <Link to="/addresses" className="btn-ghost">{t('nav.addresses')}</Link>
              </div>
            ) : (
              <div className="flex items-start gap-3 rounded border border-beige p-4">
                <div className="min-w-0 flex-1">
                  <div className="mb-1 flex items-baseline gap-2">
                    <span className="text-base font-bold">{address?.Label}</span>
                    {address?.ID === addresses[0]?.ID && (
                      <span className="kicker-emph">{t('c04.primary')}</span>
                    )}
                  </div>
                  <p className="m-0 text-sm leading-snug">
                    {address?.AddressLine}, {address?.District} {address?.City}
                  </p>
                  <p className="mt-1 text-sm text-beige-deep">
                    {address?.RecipientName} · {address?.RecipientPhone}
                    {address?.DriverNote && <> · {address.DriverNote}</>}
                  </p>
                </div>
                <Link to="/addresses" className="btn-ghost">{t('c04.change')}</Link>
              </div>
            )}

            {addresses.length > 1 && (
              <div>
                <label className="label" htmlFor="addr">{t('menu.deliver_to')}</label>
                <select id="addr" className="field" value={addressID}
                        onChange={(e) => setAddressID(e.target.value)}>
                  {addresses.map((a) => (
                    <option key={a.ID} value={a.ID}>{a.Label} — {a.AddressLine}</option>
                  ))}
                </select>
              </div>
            )}

            {/* ── Slots ─────────────────────────────────────────────────── */}
            {date && (
              <section>
                <h2 className="kicker">{t('c04.slot_for', dayLong(date, localeTag(locale)))}</h2>
                <div className="mt-3 flex flex-wrap gap-2" role="radiogroup">
                  {slots.map((s) => (
                    <button
                      key={s.slot_id}
                      type="button"
                      role="radio"
                      aria-checked={s.slot_id === slotID}
                      disabled={!s.serviceable}
                      onClick={() => setSlotID(s.slot_id)}
                      className={[
                        'min-h-touch flex-1 rounded px-4 text-sm font-semibold',
                        s.slot_id === slotID ? 'bg-beige text-nourish-deep' : 'border border-edge',
                        s.serviceable ? '' : 'text-beige-deep line-through opacity-60',
                      ].join(' ')}
                    >
                      {s.alias}
                    </button>
                  ))}
                </div>
                {/* Every unavailable slot says WHY, in words. */}
                {slots.filter((s) => !s.serviceable).map((s) => (
                  <p key={s.slot_id} className="mt-2 text-sm text-beige-deep">
                    {t('c04.slot_full', s.alias)}
                  </p>
                ))}
              </section>
            )}

            {/* ── What happens next ─────────────────────────────────────── */}
            <div className="rounded border border-ocean-light p-4">
              <span className="kicker-info">{t('c04.kitchen_confirm')}</span>
              <p className="m-0 mt-1 text-sm leading-snug">{t('c04.kitchen_note')}</p>
            </div>

            {/* ── Courier note ──────────────────────────────────────────── */}
            <div>
              <label className="label" htmlFor="note">{t('c04.courier_note')}</label>
              <textarea
                id="note"
                className="field"
                rows={2}
                maxLength={280}
                value={note}
                onChange={(e) => { setNoteTouched(true); setNote(e.target.value) }}
                placeholder={t('c04.courier_placeholder')}
              />
            </div>

            {error && <p className="error" role="alert">{error}</p>}
          </div>
        </div>
      </State>

      <BottomBar
        kicker={t('c04.pay')}
        total={quote ? <MoneyView formatted={quote.line_total} /> : <>—</>}
      >
        <SubmitButton
          pending={placing}
          type="button"
          onClick={place}
          disabled={addresses.length === 0 || !chosen?.serviceable}
        >
          {t('c04.continue')}
        </SubmitButton>
      </BottomBar>
    </Phone>
  )
}
