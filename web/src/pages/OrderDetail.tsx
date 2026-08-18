import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiFailure, request } from '../lib/api'
import { CopyButton, State, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'

type Detail = {
  id: string; order_code: string; status: string
  total: string; payment_amount: string; unique_code?: number
  bank_name?: string; bank_account_number?: string; bank_account_holder?: string
  payment_deadline?: string
  lines: { line_no: number; qty: number; unit_price: string; line_total: string
           service_date?: string; meal: { name?: string; items?: { name: string }[] } }[]
  deliveries: { id: string; delivery_code: string; service_date: string; slot: string
                status: string; kitchen?: string }[]
}

export default function OrderDetail() {
  const t = useT()
  const { id } = useParams()
  const [order, setOrder] = useState<Detail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  function load() {
    request<Detail>(`/orders/${id}`)
      .then(setOrder)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('order.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [id])

  async function upload() {
    const file = fileRef.current?.files?.[0]
    if (!file) return
    setUploading(true); setError(null)
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

  return (
    <State loading={loading} error={error && !order ? error : null}>
      {order && (
        <div>
          <h1 className="text-3xl mb-2">{order.order_code}</h1>
          <p className="mb-6"><span className="badge">{order.status}</span></p>

          {order.status === 'AWAITING_PAYMENT' && (
            <section className="card mb-8">
              <h2 className="text-xl mb-3">{t('order.how_to_pay')}</h2>
              <p className="mb-2">{t('order.transfer_exactly')}</p>
              <p className="text-2xl font-display mb-3">{order.payment_amount}</p>
              {/* The suffix only works if customers do not round it. */}
              <p className="text-sm text-ink-muted mb-4 max-w-prose">
                {t('order.unique_code_note')}
              </p>
              <div className="mb-4">
                <p className="text-sm text-ink-muted">{t('order.bank')} {order.bank_name}</p>
                {/* The account number is the field customers mistype, so it is
                    set large, spaced, and copyable rather than buried in a
                    sentence. */}
                <p className="font-mono text-xl tracking-wider my-1">
                  {order.bank_account_number}
                </p>
                {order.bank_account_number && (
                  <CopyButton value={order.bank_account_number} label={t('order.copy_account')} />
                )}
                <p className="text-sm mt-2">{t('order.account_holder')} {order.bank_account_holder}</p>
              </div>

              <label className="label" htmlFor="proof">{t('order.upload_proof')}</label>
              <input id="proof" ref={fileRef} type="file" className="field mb-3"
                     accept="image/jpeg,image/png,image/webp,application/pdf" />
              <p className="text-xs text-ink-muted mb-3">{t('order.proof_formats')}</p>
              <SubmitButton pending={uploading} type="button" onClick={upload}>
                {t('order.send_proof')}
              </SubmitButton>
              {error && <p className="error" role="alert">{error}</p>}
            </section>
          )}

          <section className="mb-8">
            <h2 className="text-xl mb-3">{t('order.lines')}</h2>
            <ul className="grid gap-2">
              {order.lines.map((l) => (
                <li key={l.line_no} className="card">
                  <p className="font-medium">{l.meal?.name ?? t('orders.package')}</p>
                  {l.service_date && <p className="text-sm text-ink-muted">{l.service_date}</p>}
                  <p className="text-sm">{l.qty} × {l.unit_price} = {l.line_total}</p>
                </li>
              ))}
            </ul>
            <p className="mt-3 text-right text-lg">{t('order.total')} {order.total}</p>
          </section>

          {order.deliveries.length > 0 && (
            <section>
              <h2 className="text-xl mb-3">{t('order.deliveries')}</h2>
              <ul className="grid gap-2">
                {order.deliveries.map((d) => (
                  <li key={d.id} className="card flex flex-wrap gap-3 items-center">
                    <span>{d.service_date} · {d.slot}</span>
                    <span className="badge">{d.status}</span>
                    {d.kitchen && <span className="text-sm text-ink-muted">{d.kitchen}</span>}
                  </li>
                ))}
              </ul>
            </section>
          )}
        </div>
      )}
    </State>
  )
}
