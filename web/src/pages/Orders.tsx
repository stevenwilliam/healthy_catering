import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiFailure, Page, request } from '../lib/api'
import { SearchBox, State } from '../components/ui'

type OrderRow = {
  id: string; order_code: string; order_type: string; status: string
  total: string; payment_amount: string; placed_at?: string; delivery_count: number
}

const STATUS_LABEL: Record<string, string> = {
  AWAITING_PAYMENT: 'Menunggu pembayaran',
  PAYMENT_SUBMITTED: 'Bukti terkirim',
  PAID: 'Lunas',
  COMPLETED: 'Selesai',
  EXPIRED: 'Kedaluwarsa',
  CANCELLED: 'Dibatalkan',
  REFUNDED: 'Dikembalikan',
}

export default function Orders() {
  const [page, setPage] = useState<Page<OrderRow> | null>(null)
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    request<Page<OrderRow>>(`/orders?q=${encodeURIComponent(q)}`)
      .then(setPage)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : 'Gagal memuat pesanan.'))
      .finally(() => setLoading(false))
  }, [q])

  return (
    <div>
      <h1 className="text-3xl mb-6">Pesanan saya</h1>
      <SearchBox value={q} onChange={setQ} placeholder="Cari kode pesanan atau status"
                 resultCount={page?.total} />

      <State loading={loading} error={error} empty={(page?.items.length ?? 0) === 0}
             emptyText="Belum ada pesanan.">
        <ul className="grid gap-3">
          {page?.items.map((o) => (
            <li key={o.id} className="card flex flex-wrap items-center gap-3">
              <Link to={`/orders/${o.id}`} className="font-display text-lg underline">
                {o.order_code}
              </Link>
              <span className="badge">{STATUS_LABEL[o.status] ?? o.status}</span>
              <span className="text-sm text-ink-muted">
                {o.order_type === 'PACKAGE' ? 'Paket' : `${o.delivery_count} pengiriman`}
              </span>
              <span className="ml-auto tabular-nums">{o.total}</span>
            </li>
          ))}
        </ul>
      </State>
    </div>
  )
}
