import { useEffect, useState } from 'react'
import { ApiFailure, Page, request } from '../lib/api'
import { SearchBox, State } from '../components/ui'
import { useT } from '../lib/i18n'
import ExportCsv from '../components/ExportCsv'
import type { MessageKey } from '../lib/messages'

type Row = {
  id: string; delivery_code: string; service_date: string; slot: string
  status: string; kitchen: string; customer_name: string; phone: string
  address_line: string; district: string; meals: number
  distance_m: number; assignment_mode: string; assignment_reason: string
}

// Which button to offer next, driven by the same machine the server enforces.
const NEXT: Record<string, { to: string; label: string }[]> = {
  SCHEDULED: [{ to: 'PREPARING', label: 'deliv.start_cooking' }],
  PREPARING: [{ to: 'OUT_FOR_DELIVERY', label: 'deliv.depart' }],
  OUT_FOR_DELIVERY: [
    { to: 'DELIVERED', label: 'deliv.delivered' },
    { to: 'FAILED', label: 'deliv.failed' },
  ],
}

export default function AdminDeliveries() {
  const t = useT()
  const [page, setPage] = useState<Page<Row> | null>(null)
  const [q, setQ] = useState('')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  function load() {
    setLoading(true)
    request<Page<Row>>(`/admin/deliveries?from=${date}&to=${date}&q=${encodeURIComponent(q)}`)
      .then(setPage)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('deliv.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [q, date]) // eslint-disable-line react-hooks/exhaustive-deps

  async function advance(id: string, to: string) {
    setBusy(id); setError(null)
    let reason = ''
    if (to === 'FAILED') {
      reason = window.prompt('Apa yang terjadi? (tidak ada orang, alamat salah, rusak)') ?? ''
      if (!reason) { setBusy(null); return }
    }
    try {
      await request(`/admin/deliveries/${id}/status`, { method: 'POST', body: { status: to, reason } })
      load()
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : t('deliv.status_failed'))
    } finally { setBusy(null) }
  }

  return (
    <div>
      <h1 className="text-3xl mb-6">{t('deliv.title')}</h1>

      <div className="mb-4 max-w-xs">
        <label className="label" htmlFor="date">{t('deliv.date')}</label>
        <input id="date" type="date" className="field" value={date}
               onChange={(e) => setDate(e.target.value)} />
      </div>
      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="grow"><SearchBox value={q} onChange={setQ} placeholder={t('deliv.search_placeholder')}
                 resultCount={page?.total} /></div>
        <ExportCsv path="/admin/deliveries" filename="deliveries" params={{ q, from: date, to: date }} />
      </div>

      <State loading={loading} error={error} empty={(page?.items.length ?? 0) === 0}
             emptyText={t('deliv.empty')}>
        <ul className="grid gap-3">
          {page?.items.map((d) => (
            <li key={d.id} className="card">
              <div className="flex flex-wrap items-baseline gap-3">
                <span className="font-display">{d.delivery_code}</span>
                <span className="badge">{d.status}</span>
                <span>{d.slot}</span>
                <span className="text-sm text-beige-deep">{d.kitchen}</span>
                {d.assignment_mode === 'MANUAL' && <span className="badge">{t('deliv.manual')}</span>}
                <span className="ml-auto text-sm">{d.meals} {t('menu.portions')}</span>
              </div>
              <p className="text-sm mt-1">{d.customer_name} · {d.phone}</p>
              <p className="text-sm text-beige-deep">{d.address_line}, {d.district}</p>
              <p className="text-xs text-beige-deep mt-1">{d.assignment_reason}</p>

              <div className="mt-3 flex flex-wrap gap-2">
                {(NEXT[d.status] ?? []).map((n) => (
                  <button key={n.to} className="btn-ghost" disabled={busy === d.id}
                          onClick={() => advance(d.id, n.to)}>
                    {t(n.label as MessageKey)}
                  </button>
                ))}
                {!NEXT[d.status] && (
                  <span className="text-sm text-beige-deep">{t('deliv.no_next')}</span>
                )}
              </div>
            </li>
          ))}
        </ul>
      </State>
    </div>
  )
}
