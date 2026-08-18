import { useEffect, useState } from 'react'
import { ApiFailure, Page, request } from '../lib/api'
import { SearchBox, State, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'
import ExportCsv from '../components/ExportCsv'

type QueueItem = {
  payment_id: string; order_code: string; customer_name: string; customer_email: string
  expected_amount: string; unique_code?: number; status: string
  waiting_minutes: number; proof_count: number; bank_name: string
}

export default function AdminPayments() {
  const t = useT()
  const [page, setPage] = useState<Page<QueueItem> | null>(null)
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [proofs, setProofs] = useState<Record<string, string[]>>({})
  const [rejecting, setRejecting] = useState<string | null>(null)
  const [reason, setReason] = useState('')

  function load() {
    setLoading(true)
    request<Page<QueueItem>>(`/admin/payments?q=${encodeURIComponent(q)}`)
      .then(setPage)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('pay.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [q]) // eslint-disable-line react-hooks/exhaustive-deps

  async function verify(id: string) {
    setBusy(id); setError(null)
    try {
      await request(`/admin/payments/${id}/verify`, { method: 'POST', body: {} })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('pay.verify_failed'))
    } finally { setBusy(null) }
  }

  async function reject(id: string) {
    setBusy(id); setError(null)
    try {
      await request(`/admin/payments/${id}/reject`, { method: 'POST', body: { reason } })
      setRejecting(null); setReason(''); load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('pay.reject_failed'))
    } finally { setBusy(null) }
  }

  async function viewProof(id: string) {
    // A private bucket: the only way to see a proof is a short-lived
    // presigned URL, minted per view (99 §7).
    const out = await request<{ urls: string[] }>(`/admin/payments/${id}/proof`)
    setProofs({ ...proofs, [id]: out.urls })
  }

  return (
    <div>
      <h1 className="text-3xl mb-2">{t('pay.title')}</h1>
      {/* Oldest first: the customer who has waited longest is the one about to
          telephone. */}
      <p className="text-sm text-ink-muted mb-6">{t('pay.oldest_first')}</p>

      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="grow"><SearchBox value={q} onChange={setQ} placeholder={t('pay.search_placeholder')}
                 resultCount={page?.total} /></div>
        <ExportCsv path="/admin/payments" filename="payment-queue" params={{ q }} />
      </div>

      <State loading={loading} error={error} empty={(page?.items.length ?? 0) === 0}
             emptyText={t('pay.empty')}>
        <ul className="grid gap-3">
          {page?.items.map((p) => (
            <li key={p.payment_id} className="card">
              <div className="flex flex-wrap items-baseline gap-3">
                <span className="font-display text-lg">{p.order_code}</span>
                <span>{p.customer_name}</span>
                <span className="text-sm text-ink-muted">{p.customer_email}</span>
                <span className="ml-auto text-lg tabular-nums">{p.expected_amount}</span>
              </div>
              <p className="text-sm text-ink-muted mt-1">
                {p.bank_name} · {t('pay.waiting')} {p.waiting_minutes} {t('pay.minutes')} ·{' '}
                {p.proof_count} {t('pay.proofs')}
                {p.unique_code ? ` · ${t('pay.unique_code')} ${p.unique_code}` : ''}
              </p>

              <div className="mt-3 flex flex-wrap items-center gap-2">
                <button className="btn-ghost" onClick={() => viewProof(p.payment_id)}>
                  {t('pay.view_proof')}
                </button>
                <SubmitButton pending={busy === p.payment_id} type="button"
                              onClick={() => verify(p.payment_id)}>
                  {t('pay.verify')}
                </SubmitButton>
                {rejecting === p.payment_id ? (
                  <span className="flex flex-wrap items-center gap-2">
                    <input className="field w-64" placeholder={t('pay.reject_reason')}
                           value={reason} onChange={(e) => setReason(e.target.value)} />
                    <button className="btn-danger" onClick={() => reject(p.payment_id)}>
                      {t('pay.reject')}
                    </button>
                    <button className="btn-ghost" onClick={() => setRejecting(null)}>
                      {t('ui.cancel')}
                    </button>
                  </span>
                ) : (
                  <button className="btn-ghost" onClick={() => setRejecting(p.payment_id)}>
                    {t('pay.reject')}
                  </button>
                )}
              </div>

              {proofs[p.payment_id]?.map((u) => (
                <a key={u} href={u} target="_blank" rel="noreferrer"
                   className="mt-3 block underline text-sm">
                  Buka bukti transfer (tautan berlaku 10 menit)
                </a>
              ))}
            </li>
          ))}
        </ul>
      </State>
    </div>
  )
}
