import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiFailure, Page, request } from '../lib/api'
import { SearchBox, State } from '../components/ui'
import { useT } from '../lib/i18n'
import type { MessageKey } from '../lib/messages'

type OrderRow = {
  id: string; order_code: string; order_type: string; status: string
  total: string; payment_amount: string; placed_at?: string; delivery_count: number
}

// Status codes map to catalogue KEYS, not to strings: the label has to change
// with the language, and the code is what the API sends.
const STATUS_KEY: Record<string, MessageKey> = {
  AWAITING_PAYMENT: 'status.awaiting_payment',
  PAYMENT_SUBMITTED: 'status.payment_submitted',
  PAID: 'status.paid',
  COMPLETED: 'status.completed',
  EXPIRED: 'status.expired',
  CANCELLED: 'status.cancelled',
  REFUNDED: 'status.refunded',
}

export default function Orders() {
  const t = useT()
  const [page, setPage] = useState<Page<OrderRow> | null>(null)
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    request<Page<OrderRow>>(`/orders?q=${encodeURIComponent(q)}`)
      .then(setPage)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('orders.load_failed')))
      .finally(() => setLoading(false))
  }, [q])

  return (
    <div>
      <h1 className="text-3xl mb-6">{t('orders.title')}</h1>
      <SearchBox value={q} onChange={setQ} placeholder={t('orders.search_placeholder')}
                 resultCount={page?.total} />

      <State loading={loading} error={error} empty={(page?.items.length ?? 0) === 0}
             emptyText={t('orders.empty')}>
        <ul className="grid gap-3">
          {page?.items.map((o) => (
            <li key={o.id} className="card flex flex-wrap items-center gap-3">
              <Link to={`/orders/${o.id}`} className="font-display text-lg underline">
                {o.order_code}
              </Link>
              <span className="badge">
                {STATUS_KEY[o.status] ? t(STATUS_KEY[o.status]!) : o.status}
              </span>
              <span className="text-sm text-ink-muted">
                {o.order_type === 'PACKAGE'
                  ? t('orders.package')
                  : `${o.delivery_count} ${t('orders.deliveries')}`}
              </span>
              <span className="ml-auto tabular-nums">{o.total}</span>
            </li>
          ))}
        </ul>
      </State>
    </div>
  )
}
