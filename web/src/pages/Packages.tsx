import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiFailure, newIdempotencyKey, Page, request } from '../lib/api'
import { SearchBox, State, SubmitButton } from '../components/ui'

type Pkg = { id: string; name: string; description: string; meal_credits: number; validity_days: number }
type Mine = {
  id: string; package_name: string; status: string
  purchased_credits: number; remaining_credits: number
  expires_at?: string; price_paid: string
}
type Entry = { entry_type: string; qty: number; running_balance: number; note: string; occurred_at: string }

export default function Packages() {
  const nav = useNavigate()
  const [available, setAvailable] = useState<Pkg[]>([])
  const [mine, setMine] = useState<Mine[]>([])
  const [ledger, setLedger] = useState<Record<string, Entry[]>>({})
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [buying, setBuying] = useState<string | null>(null)

  function load() {
    Promise.all([
      request<Page<Pkg>>(`/packages?q=${encodeURIComponent(q)}`),
      request<Page<Mine>>('/my/packages'),
    ])
      .then(([a, m]) => { setAvailable(a.items); setMine(m.items) })
      .catch((e) => setError(e instanceof ApiFailure ? e.message : 'Gagal memuat paket.'))
      .finally(() => setLoading(false))
  }
  useEffect(load, [q]) // eslint-disable-line react-hooks/exhaustive-deps

  async function buy(id: string) {
    setBuying(id); setError(null)
    try {
      const out = await request<{ order_id: string }>(`/packages/${id}/buy`, {
        method: 'POST',
        idempotencyKey: newIdempotencyKey(),
      })
      nav(`/orders/${out.order_id}`)
    } catch (e) {
      setError(e instanceof ApiFailure ? e.message : 'Pembelian gagal.')
    } finally {
      setBuying(null)
    }
  }

  async function showLedger(id: string) {
    if (ledger[id]) { setLedger({ ...ledger, [id]: [] }); return }
    const entries = await request<Entry[]>(`/my/packages/${id}/ledger`)
    setLedger({ ...ledger, [id]: entries })
  }

  return (
    <div>
      <h1 className="text-3xl mb-6">Paket kredit</h1>

      <State loading={loading} error={error} empty={false}>
        {mine.length > 0 && (
          <section className="mb-10">
            <h2 className="text-xl mb-3">Paket saya</h2>
            <ul className="grid gap-3">
              {mine.map((p) => (
                <li key={p.id} className="card">
                  <div className="flex flex-wrap items-center gap-3">
                    <span className="font-display text-lg">{p.package_name}</span>
                    <span className="badge">{p.status}</span>
                    <span>{p.remaining_credits} / {p.purchased_credits} kredit</span>
                    {p.expires_at && (
                      <span className="text-sm text-ink-muted">berlaku sampai {p.expires_at}</span>
                    )}
                    <button className="btn-ghost ml-auto" onClick={() => showLedger(p.id)}>
                      Riwayat kredit
                    </button>
                  </div>

                  {/* The ledger drill-down PROMPT §7 asks for: every movement,
                      with a running balance, so a disputed number can be
                      traced rather than argued about. */}
                  {ledger[p.id]?.length ? (
                    <table className="mt-4 w-full text-sm">
                      <caption className="sr-only">Riwayat kredit {p.package_name}</caption>
                      <thead>
                        <tr className="text-left border-b border-nourish-deep/30">
                          <th scope="col" className="py-1">Waktu</th>
                          <th scope="col">Jenis</th>
                          <th scope="col" className="text-right">Perubahan</th>
                          <th scope="col" className="text-right">Saldo</th>
                          <th scope="col">Catatan</th>
                        </tr>
                      </thead>
                      <tbody>
                        {ledger[p.id]!.map((e, i) => (
                          <tr key={i} className="border-b border-nourish-deep/10">
                            <td className="py-1">{e.occurred_at.slice(0, 16)}</td>
                            <td>{e.entry_type}</td>
                            <td className="text-right tabular-nums">{e.qty > 0 ? `+${e.qty}` : e.qty}</td>
                            <td className="text-right tabular-nums">{e.running_balance}</td>
                            <td className="text-ink-muted">{e.note}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : null}
                </li>
              ))}
            </ul>
          </section>
        )}

        <h2 className="text-xl mb-3">Beli paket</h2>
        <SearchBox value={q} onChange={setQ} placeholder="Cari paket" resultCount={available.length} />
        <ul className="grid gap-4 sm:grid-cols-3">
          {available.map((p) => (
            <li key={p.id} className="card">
              <h3 className="text-lg">{p.name}</h3>
              <p className="text-sm text-ink-muted mb-2">{p.description}</p>
              <p className="text-sm mb-3">{p.meal_credits} kredit · berlaku {p.validity_days} hari</p>
              <SubmitButton pending={buying === p.id} type="button" onClick={() => buy(p.id)}>
                Beli
              </SubmitButton>
            </li>
          ))}
        </ul>

        {/* D-31 stated where a customer will actually read it, not only in the
            terms page they never open. */}
        <p className="mt-6 text-sm text-ink-muted max-w-prose">
          Satu kredit berlaku untuk satu paket makan, berapa pun jumlah lauknya.
          Masa aktif dimulai saat pembayaran kami konfirmasi. Kredit yang tidak
          terpakai hangus saat masa aktif berakhir dan tidak dapat diuangkan.
        </p>
      </State>
    </div>
  )
}
