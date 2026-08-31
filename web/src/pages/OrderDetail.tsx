import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiFailure, request } from '../lib/api'
import { CopyButton, State, SubmitButton } from '../components/ui'
import { AppBar, BottomBar, Phone, humanLeft } from '../components/mobile'
import { useI18n, useT } from '../lib/i18n'
import { localeTag } from './Menu'

/** Artboard 05 — manual transfer, and the three digits that identify it.
 *
 * The unique suffix is the whole mechanism. Bank transfers arrive with a name
 * and an amount and nothing else, so the last three rupiah are what let
 * finance match a payment to an order without asking the customer. That is why
 * the amount is shown with the suffix picked out, why it is copyable as one
 * string, and why the screen says "exact" rather than "about".
 *
 * The deadline is real: past it the order is cancelled and the slot is
 * released (docs/02 D-13). It counts down rather than showing a timestamp,
 * because "2 jam 47 menit" is actionable and "18:32" needs arithmetic.
 */

type Detail = {
  id: string
  order_code: string
  status: string
  total: string
  payment_amount: string
  unique_code?: number
  bank_name?: string
  bank_account_number?: string
  bank_account_holder?: string
  payment_deadline?: string
  lines: {
    line_no: number; qty: number; unit_price: string; line_total: string
    service_date?: string; meal: { name?: string; items?: { name: string }[] }
  }[]
  deliveries: {
    id: string; delivery_code: string; service_date: string; slot: string
    status: string; kitchen?: string
  }[]
}

export default function OrderDetail() {
  const t = useT()
  const { locale } = useI18n()
  const { id } = useParams()
  const [order, setOrder] = useState<Detail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  function load() {
    request<Detail>(`/orders/${id}`)
      .then(setOrder)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('order.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  async function upload() {
    const file = fileRef.current?.files?.[0]
    if (!file) return
    setUploading(true)
    setError(null)
    try {
      const form = new FormData()
      form.append('file', file)
      await request(`/orders/${id}/proof`, { method: 'POST', form })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('order.upload_failed'))
    } finally {
      setUploading(false)
    }
  }

  const awaiting = order?.status === 'AWAITING_PAYMENT' || order?.status === 'PENDING_PAYMENT'
  const amount = order?.payment_amount ?? order?.total ?? ''

  return (
    <Phone>
      <AppBar title={t('c05.title')} />

      <State loading={loading} error={error}>
        {order && (
          <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
            <div>
              <span className="kicker">{t('c05.order', order.order_code)}</span>
              <h2 className="mt-2 text-2xl leading-tight">{t('c05.transfer', amount)}</h2>
            </div>

            {awaiting && order.payment_deadline && (
              <Countdown deadline={order.payment_deadline} locale={localeTag(locale)} />
            )}

            {/* ── Rekening tujuan ───────────────────────────────────────── */}
            {order.bank_account_number && (
              <section className="gtable">
                <h3 className="border-b border-edge px-4 py-3 font-display text-xl font-semibold">
                  {t('c05.account')}
                </h3>
                <div className="flex flex-col gap-4 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className="text-sm font-semibold text-beige-deep">
                        {order.bank_name} · {order.bank_account_holder}
                      </div>
                      <div className="font-display text-2xl font-bold tracking-wide">
                        {order.bank_account_number}
                      </div>
                    </div>
                    <CopyButton value={order.bank_account_number} label={t('c05.copy')} />
                  </div>

                  <div className="h-px bg-rule" />

                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className="text-sm font-semibold text-beige-deep">{t('c05.exact')}</div>
                      {/* The suffix is picked out in ember (7.27 on the
                          ground) AND named in the sentence below, so the
                          instruction never depends on seeing a colour. */}
                      <div className="font-display text-2xl font-bold">
                        <SuffixAmount amount={amount} code={order.unique_code} />
                      </div>
                    </div>
                    <CopyButton value={digitsOf(amount)} label={t('c05.copy')} />
                  </div>

                  {order.unique_code !== undefined && (
                    <p className="m-0 text-sm text-beige-deep">{t('c05.suffix_note')}</p>
                  )}
                </div>
              </section>
            )}

            {/* ── Bukti transfer ────────────────────────────────────────── */}
            <section className="flex flex-col gap-3">
              <h3 className="kicker">{t('c05.proof')}</h3>
              <label className="flex cursor-pointer flex-col items-center gap-3 rounded border-2 border-dashed border-rule p-5 text-center">
                <span aria-hidden="true"
                      className="flex h-touch w-touch items-center justify-center rounded border-2 border-beige text-onbar font-bold">
                  ↑
                </span>
                <span className="text-sm font-medium leading-snug">
                  {t('c05.upload')}<br />
                  <span className="text-beige-deep">{t('c05.formats')}</span>
                </span>
                <input
                  ref={fileRef}
                  type="file"
                  className="sr-only"
                  accept="image/jpeg,image/png"
                  onChange={() => void upload()}
                />
              </label>
              {uploading && <p className="text-sm text-beige-deep">{t('ui.processing')}</p>}
            </section>

            <div className="flex items-start gap-3 rounded border border-rule p-4">
              <span className="kicker-info whitespace-nowrap">{t('c05.waiting')}</span>
              <p className="m-0 text-sm leading-snug">{t('c05.waiting_note')}</p>
            </div>

            {/* ── What was ordered ──────────────────────────────────────── */}
            <section>
              <h3 className="kicker mb-2">{t('order.lines')}</h3>
              <ul className="m-0 flex list-none flex-col gap-2 p-0 text-sm">
                {order.lines.map((l) => (
                  <li key={l.line_no} className="flex justify-between gap-3 border-b border-rule pb-2">
                    <span>
                      {l.qty} × {l.meal.name ?? l.meal.items?.[0]?.name}
                      {l.service_date && (
                        <span className="text-beige-deep"> · {l.service_date}</span>
                      )}
                    </span>
                    <span className="font-semibold">{l.line_total}</span>
                  </li>
                ))}
              </ul>
            </section>

            {order.deliveries.length > 0 && (
              <section>
                <h3 className="kicker mb-2">{t('nav.deliveries')}</h3>
                <ul className="m-0 flex list-none flex-col gap-2 p-0 text-sm">
                  {order.deliveries.map((d) => (
                    <li key={d.id} className="flex justify-between gap-3">
                      <span>{d.service_date} · {d.slot}</span>
                      <span className="text-beige-deep">
                        {d.kitchen ? `${d.kitchen} · ` : ''}{d.status}
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            )}

            {error && <p className="error" role="alert">{error}</p>}
          </div>
        )}
      </State>

      {order && awaiting && (
        <BottomBar kicker={t('c04.pay')} total={amount}>
          {/* This button does not move money. It tells finance to look, which
              is why it is disabled once a proof is already in — pressing it
              twice does not make anyone check faster. */}
          <SubmitButton
            pending={confirming}
            type="button"
            className="btn-primary"
            onClick={() => { setConfirming(true); fileRef.current?.click() }}
          >
            {t('c05.done')}
          </SubmitButton>
        </BottomBar>
      )}
    </Phone>
  )
}

/** The amount with its identifying suffix picked out. */
function SuffixAmount({ amount, code }: { amount: string; code?: number }) {
  if (code === undefined) return <>{amount}</>
  const suffix = String(code).padStart(3, '0')
  const i = amount.lastIndexOf(suffix)
  if (i < 0) return <>{amount}</>
  return (
    <>
      {amount.slice(0, i)}
      <span className="text-ember-light">{suffix}</span>
      {amount.slice(i + suffix.length)}
    </>
  )
}

/** The digits alone, for the clipboard — a banking app wants 480148, not
 *  "Rp 480.148". */
function digitsOf(formatted: string): string {
  return formatted.replace(/\D/g, '')
}

/** Counts down to the payment deadline. */
function Countdown({ deadline, locale }: { deadline: string; locale: string }) {
  const t = useT()
  const [left, setLeft] = useState(() => Date.parse(deadline) - Date.now())

  useEffect(() => {
    const id = window.setInterval(() => setLeft(Date.parse(deadline) - Date.now()), 30_000)
    return () => window.clearInterval(id)
  }, [deadline])

  if (Number.isNaN(left)) return null
  return (
    <div className="note-emph">
      <p className="m-0">{t('c05.deadline', humanLeft(Math.max(0, left), locale))}</p>
    </div>
  )
}
