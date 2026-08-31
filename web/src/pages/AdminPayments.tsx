import { useEffect, useState } from 'react'
import { ApiFailure, Page, request } from '../lib/api'
import { SearchBox, State, SubmitButton } from '../components/ui'
import { Board, TopBar, Tabs } from '../components/backoffice'
import ExportCsv from '../components/ExportCsv'
import { useT } from '../lib/i18n'

/** Artboard S4 — the payment verification queue.
 *
 * A queue on the left, one payment open on the right. That split is the whole
 * point: verifying a bank transfer means comparing four things — what was
 * billed, what arrived, who sent it, and when — against an image the customer
 * uploaded. Doing that from a list row means holding numbers in your head.
 *
 * Both buttons write to the audit log with actor and reason, which is why
 * rejection asks for one rather than offering a bare "Tolak".
 */

type QueueItem = {
  payment_id: string
  order_code: string
  customer_name: string
  customer_email: string
  expected_amount: string
  unique_code?: number
  status: string
  waiting_minutes: number
  proof_count: number
  bank_name: string
}

export default function AdminPayments() {
  const t = useT()
  const [page, setPage] = useState<Page<QueueItem> | null>(null)
  const [status, setStatus] = useState('SUBMITTED')
  const [q, setQ] = useState('')
  const [selected, setSelected] = useState<string>('')
  const [proofs, setProofs] = useState<Record<string, string[]>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [rejecting, setRejecting] = useState(false)
  const [reason, setReason] = useState('')

  function load() {
    setLoading(true)
    request<Page<QueueItem>>(
      `/admin/payments?status=${status}&q=${encodeURIComponent(q)}`)
      .then((p) => {
        setPage(p)
        setSelected((cur) => p.items.some((i) => i.payment_id === cur)
          ? cur
          : p.items[0]?.payment_id ?? '')
      })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('pay.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [q, status]) // eslint-disable-line react-hooks/exhaustive-deps

  const items = page?.items ?? []
  const open = items.find((i) => i.payment_id === selected)

  // The proof image is fetched per payment and cached: it is an authenticated
  // endpoint, so it cannot be an <img src> — a plain request lands on the
  // login page and renders as a broken image.
  useEffect(() => {
    if (!open || proofs[open.payment_id]) return
    let live = true
    // The endpoint answers { urls, expires_in_seconds } — a presigned link
    // that dies in ten minutes, because the bucket is private and a URL
    // pasted into a chat must stop working (99 §7).
    request<{ urls: string[] }>(`/admin/payments/${open.payment_id}/proof`)
      .then((r) => {
        if (live) setProofs((p) => ({ ...p, [open.payment_id]: r?.urls ?? [] }))
      })
      .catch(() => { /* an unreadable proof must not blank the panel */ })
    return () => { live = false }
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  async function verify() {
    if (!open) return
    setBusy(true)
    setError(null)
    try {
      await request(`/admin/payments/${open.payment_id}/verify`, { method: 'POST', body: {} })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('pay.verify_failed'))
    } finally { setBusy(false) }
  }

  async function reject() {
    if (!open || !reason.trim()) return
    setBusy(true)
    setError(null)
    try {
      await request(`/admin/payments/${open.payment_id}/reject`, {
        method: 'POST', body: { reason: reason.trim() },
      })
      setRejecting(false)
      setReason('')
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('pay.reject_failed'))
    } finally { setBusy(false) }
  }

  return (
    <Board>
      <TopBar
        title={t('pay.title')}
        actions={items.length > 0 && status === 'SUBMITTED' && (
          <span className="pill-emph">{t('pay.tab_waiting', items.length)}</span>
        )}
        note={t('pay.audit_note')}
      />

      <div className="grid xl:grid-cols-[1fr_470px]">
        {/* ── The queue ───────────────────────────────────────────────────── */}
        <div className="min-w-0 border-rule xl:border-r">
          <div className="flex flex-wrap items-center gap-2 border-b border-rule px-6 py-4">
            <Tabs
              label={t('pay.title')}
              value={status}
              onChange={setStatus}
              options={[
                { id: 'SUBMITTED', name: t('pay.tab_waiting', items.length) },
                { id: 'REJECTED', name: t('pay.tab_rejected', '') },
                { id: 'VERIFIED', name: t('pay.tab_verified', '') },
              ]}
            />
            <span className="ml-auto"><ExportCsv path="/admin/payments"
              params={{ status, q }} filename={`payments-${status.toLowerCase()}`} /></span>
          </div>

          <div className="px-6 py-4">
            <SearchBox value={q} onChange={setQ} resultCount={items.length}
                       placeholder={t('pay.search_placeholder')} />

            <State loading={loading} error={error} empty={items.length === 0}
                   emptyText={t('pay.empty')}>
              <div className="overflow-x-auto">
                <div className="gtable min-w-[44rem]">
                  <div className="grid"
                       style={{ gridTemplateColumns: '1.15fr 1.35fr 1fr 0.85fr 0.9fr' }}>
                    <div className="gtable-head">{t('pay.col_order')}</div>
                    <div className="gtable-head">{t('pay.col_customer')}</div>
                    <div className="gtable-head">{t('pay.col_amount')}</div>
                    <div className="gtable-head">{t('pay.col_waiting')}</div>
                    <div className="gtable-head">{t('pay.col_match')}</div>
                    {items.map((p, i) => {
                      const on = p.payment_id === selected
                      const last = i === items.length - 1
                      // The selected row is the canvas's #468973 FILL, and its
                      // text goes large so beige on it clears AA (§4.1).
                      const cell = [
                        'gtable-cell', last ? 'is-last' : '',
                        on ? 'gtable-row-selected' : '',
                      ].join(' ')
                      return (
                        <div
                          key={p.payment_id}
                          className="contents"
                          onClick={() => setSelected(p.payment_id)}
                        >
                          <div className={`${cell} cursor-pointer font-semibold`}>
                            <button type="button" className="text-left"
                                    aria-pressed={on}>{p.order_code}</button>
                          </div>
                          <div className={cell}>{p.customer_name}</div>
                          <div className={`${cell} font-semibold`}>{p.expected_amount}</div>
                          <div className={cell}>{p.waiting_minutes} {t('pay.minutes')}</div>
                          <div className={cell}>
                            {p.proof_count > 0
                              ? <span className="pill-ok">{t('pay.match_ok')}</span>
                              : <span className="pill-archived">{t('pay.match_none')}</span>}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              </div>
            </State>
          </div>
        </div>

        {/* ── The one being reviewed ──────────────────────────────────────── */}
        <aside className="min-w-0">
          {!open ? (
            <p className="p-6 text-sm text-beige-deep">{t('pay.select_row')}</p>
          ) : (
            <div className="flex h-full flex-col">
              <div className="border-b border-rule px-6 py-5">
                <span className="kicker">
                  {open.order_code} · {t('pay.waiting')} {open.waiting_minutes} {t('pay.minutes')}
                </span>
                <h2 className="mb-1 mt-2 text-2xl">{open.customer_name}</h2>
                <p className="text-sm text-beige-deep">
                  {t('pay.retail')} · {open.customer_email} · {open.bank_name}
                </p>
              </div>

              <div className="flex flex-1 flex-col gap-4 p-6">
                {/* The proof. A labelled placeholder when there is none —
                    "no proof yet" is a state finance must be able to see. */}
                <div className="flex h-52 items-end overflow-hidden rounded-card bg-bar p-4">
                  {proofs[open.payment_id]?.[0] ? (
                    <img src={proofs[open.payment_id]![0]} alt={t('pay.proof_label')}
                         className="h-full w-full object-contain" />
                  ) : (
                    <span className="kicker text-beige">{t('pay.proof_label')}</span>
                  )}
                </div>

                {/* The four-way comparison the artboard draws. What we can
                    state is what the ORDER says; the bank side is read off the
                    proof image by the person, which is the whole job. */}
                <div className="gtable">
                  <div className="grid grid-cols-2">
                    <Cell label={t('pay.billed')} value={open.expected_amount} big border />
                    <Cell label={t('pay.col_amount')} value={open.expected_amount} big />
                    <Cell label={t('pay.col_customer')} value={open.customer_name} border last />
                    <Cell label={t('pay.unique_code')}
                          value={open.unique_code !== undefined ? String(open.unique_code) : '—'} last />
                  </div>
                </div>

                {open.unique_code !== undefined && (
                  <div className="note-info">
                    <p className="m-0">
                      {t('pay.suffix_match', String(open.unique_code).padStart(3, '0'))}
                    </p>
                  </div>
                )}

                {error && <p className="error" role="alert">{error}</p>}
              </div>

              <div className="flex flex-col gap-3 border-t border-rule p-6">
                {rejecting ? (
                  <>
                    <label className="label" htmlFor="reason">{t('pay.reject_reason')}</label>
                    <input id="reason" className="field" value={reason} autoFocus
                           onChange={(e) => setReason(e.target.value)} />
                    <div className="flex gap-3">
                      {/* A reason is REQUIRED: the audit row is the record of
                          why a customer's money was refused. */}
                      <SubmitButton pending={busy} type="button" className="btn-danger flex-1"
                                    disabled={!reason.trim()} onClick={reject}>
                        {t('pay.reject')}
                      </SubmitButton>
                      <button type="button" className="btn-ghost"
                              onClick={() => { setRejecting(false); setReason('') }}>
                        {t('ui.cancel')}
                      </button>
                    </div>
                  </>
                ) : (
                  <div className="flex gap-3">
                    <SubmitButton pending={busy} type="button" className="btn-primary flex-1"
                                  onClick={verify}>
                      {t('pay.verify')}
                    </SubmitButton>
                    <button type="button" className="btn-danger"
                            onClick={() => setRejecting(true)}>
                      {t('pay.reject')}
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}
        </aside>
      </div>
    </Board>
  )
}

function Cell({ label, value, big, border, last }: {
  label: string; value: string; big?: boolean; border?: boolean; last?: boolean
}) {
  return (
    <div className={[
      'px-4 py-3',
      border ? 'border-r border-rule' : '',
      last ? '' : 'border-b border-rule',
    ].join(' ')}>
      <div className="text-xs text-beige-deep">{label}</div>
      <div className={big ? 'font-display text-lg font-bold' : 'font-semibold'}>{value}</div>
    </div>
  )
}
