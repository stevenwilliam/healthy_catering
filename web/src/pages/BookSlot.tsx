import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiFailure, Page, newIdempotencyKey, request } from '../lib/api'
import { State, SubmitButton } from '../components/ui'
import { AppBar, BottomBar, DayChips, Phone, dayLong } from '../components/mobile'
import { useI18n, useT } from '../lib/i18n'
import { addDays, localeTag, serviceDateWIB, type Meal } from './Menu'

/** Artboard M3 — spending credit on a published slot.
 *
 * A slot the customer's own area cannot take is shown, dimmed, with the
 * reason — "Kapasitas penuh untuk wilayahmu". Hiding it would leave them
 * wondering whether the kitchen cooks that day at all, and the answer is
 * different: it does, just not for them.
 *
 * Booking is idempotent. A double-tapped "Kunci jadwal" must spend ONE credit;
 * the key is what makes the second press a no-op rather than a second booking
 * the customer then has to ask us to reverse.
 */

type Mine = {
  id: string
  package_name: string
  remaining_credits: number
  expires_at?: string
}

type SlotOffer = {
  slot_id: string
  alias: string
  serviceable: boolean
  reason?: string
}

type Address = { ID: string; Label: string; Latitude: number; Longitude: number }

export default function BookSlot() {
  const t = useT()
  const { locale } = useI18n()
  const nav = useNavigate()

  const [pkg, setPkg] = useState<Mine | null>(null)
  const [meals, setMeals] = useState<Meal[]>([])
  const [address, setAddress] = useState<Address | null>(null)
  const [offers, setOffers] = useState<SlotOffer[]>([])
  const [day, setDay] = useState('')
  const [slotID, setSlotID] = useState('')
  const [loading, setLoading] = useState(true)
  const [booking, setBooking] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const from = serviceDateWIB()
    Promise.all([
      request<Page<Mine>>('/my/packages'),
      request<Meal[]>(`/menu?from=${from}&to=${addDays(from, 14)}`),
      request<Address[]>('/addresses'),
    ])
      .then(([p, m, a]) => {
        const items = Array.isArray(p?.items) ? p.items : []
        setPkg(items.find((x) => x.remaining_credits > 0) ?? null)
        setMeals(Array.isArray(m) ? m : [])
        setAddress(Array.isArray(a) ? a[0] ?? null : null)
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('packages.load_failed')))
      .finally(() => setLoading(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const days = useMemo(
    () => [...new Set(meals.map((m) => m.service_date))].sort(),
    [meals],
  )

  useEffect(() => {
    if (days.length && !days.includes(day)) setDay(days[0]!)
  }, [days, day])

  useEffect(() => {
    if (!address || !day) return
    let live = true
    request<SlotOffer[]>(
      `/delivery-slots/availability?lat=${address.Latitude}&lng=${address.Longitude}&date=${day}&qty=1`,
    )
      .then((o) => {
        if (!live) return
        setOffers(o)
        setSlotID(o.find((s) => s.serviceable)?.slot_id ?? '')
      })
      .catch(() => setOffers([]))
    return () => { live = false }
  }, [address, day])

  // What is actually cooking in each slot that day — the artboard names the
  // dish beside the time, because "11.30" alone is not a choice.
  const dishFor = (slotId: string) =>
    meals.find((m) => m.service_date === day && m.slot_id === slotId)

  async function book() {
    if (!pkg || !slotID || !address) return
    const meal = dishFor(slotID)
    if (!meal) return
    setBooking(true)
    setError(null)
    try {
      await request(`/my/packages/${pkg.id}/book`, {
        method: 'POST',
        body: { scheduled_meal_id: meal.id, address_id: address.ID },
        // One credit per tap, however many taps (PROMPT §14).
        idempotencyKey: newIdempotencyKey(),
      })
      nav('/credits')
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('packages.book_failed'))
    } finally {
      setBooking(false)
    }
  }

  const chosen = offers.find((s) => s.slot_id === slotID)
  const canBook = !!pkg && !!chosen?.serviceable && !!dishFor(slotID)

  return (
    <Phone>
      <AppBar
        title={t('m3.title')}
        trailing={pkg && (
          <span className="rounded-full bg-canvas px-4 py-1 text-onbar font-bold text-beige">
            {t('m3.credits', pkg.remaining_credits)}
          </span>
        )}
      />

      <State loading={loading} error={error}>
        <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
          {!pkg ? (
            <div className="flex flex-1 flex-col items-center justify-center gap-4 py-16 text-center">
              <p className="text-beige-deep">{t('c06.none')}</p>
              <Link to="/packages" className="btn-primary">{t('c06.buy')}</Link>
            </div>
          ) : days.length === 0 ? (
            <p className="py-16 text-center text-beige-deep">{t('m3.none_published')}</p>
          ) : (
            <>
              <DayChips days={days.slice(0, 6)} value={day} onChange={setDay} />

              <h2 className="kicker">{t('m3.pick_time', dayLong(day, localeTag(locale)))}</h2>

              <div className="flex flex-col gap-3" role="radiogroup">
                {offers.map((s) => {
                  const dish = dishFor(s.slot_id)
                  const on = s.slot_id === slotID
                  const usable = s.serviceable && !!dish
                  return (
                    <button
                      key={s.slot_id}
                      type="button"
                      role="radio"
                      aria-checked={on}
                      disabled={!usable}
                      onClick={() => setSlotID(s.slot_id)}
                      className={`flex items-center justify-between gap-3 rounded p-4 text-left ${
                        on ? 'border-2 border-beige' : 'border border-rule'
                      } ${usable ? '' : 'opacity-60'}`}
                    >
                      <div className="min-w-0">
                        <div className="font-display text-xl font-bold">{s.alias}</div>
                        <div className={`mt-1 text-sm ${
                          usable ? 'text-beige-deep' : 'text-berry-light'}`}>
                          {!s.serviceable
                            ? t('m3.area_full')
                            : dish
                              ? (dish.name ?? dish.items[0]?.food_name)
                              : t('m3.none_published')}
                        </div>
                      </div>
                      <span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full ${
                        on ? 'bg-beige font-bold text-nourish-deep' : 'border-2 border-rule'}`}>
                        {on ? '✓' : ''}
                      </span>
                    </button>
                  )
                })}
              </div>

              <div className="note-info">{t('m3.change_note')}</div>

              {error && <p className="error" role="alert">{error}</p>}
            </>
          )}
        </div>
      </State>

      {pkg && (
        <BottomBar
          kicker={t('m3.selected', canBook ? 1 : 0)}
          total={t('m3.remaining', Math.max(0, pkg.remaining_credits - (canBook ? 1 : 0)))}
        >
          <SubmitButton pending={booking} type="button" onClick={book} disabled={!canBook}>
            {t('m3.lock')}
          </SubmitButton>
        </BottomBar>
      )}
    </Phone>
  )
}
