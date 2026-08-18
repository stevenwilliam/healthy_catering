import { FormEvent, useEffect, useState } from 'react'
import { ApiFailure, request } from '../lib/api'
import { FieldError, State, SubmitButton } from '../components/ui'
import { useT } from '../lib/i18n'

type Address = {
  ID: string; Label: string; RecipientName: string; RecipientPhone: string
  AddressLine: string; District: string; City: string
  Latitude: number; Longitude: number; DriverNote: string
}

type Saved = {
  id: string; serviceable: boolean
  kitchen_name?: string; distance_km?: number; delivery_fee?: string; message: string
}

export default function Addresses() {
  const t = useT()
  const [list, setList] = useState<Address[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [fields, setFields] = useState<Record<string, string>>({})
  const [pending, setPending] = useState(false)
  const [result, setResult] = useState<Saved | null>(null)

  const [form, setForm] = useState({
    label: '', recipient_name: '', recipient_phone: '', address_line: '',
    district: '', city: '', latitude: '', longitude: '', driver_note: '',
  })

  function set(k: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [k]: e.target.value })
  }

  function load() {
    request<Address[]>('/addresses')
      .then(setList)
      .catch((e) => setError(e instanceof ApiFailure ? e.message : t('addresses.load_failed')))
      .finally(() => setLoading(false))
  }
  useEffect(load, [])

  async function submit(e: FormEvent) {
    e.preventDefault()
    setPending(true); setError(null); setFields({}); setResult(null)
    try {
      const saved = await request<Saved>('/addresses', {
        method: 'POST',
        body: {
          ...form,
          latitude: Number(form.latitude),
          longitude: Number(form.longitude),
        },
      })
      setResult(saved)
      load()
    } catch (err) {
      if (err instanceof ApiFailure) { setError(err.message); setFields(err.details) }
      else setError(t('addresses.save_failed'))
    } finally {
      setPending(false)
    }
  }

  return (
    <div>
      <h1 className="text-3xl mb-6">{t('addresses.title')}</h1>

      <State loading={loading} error={null} empty={list.length === 0}
             emptyText={t('addresses.empty')}>
        <ul className="grid gap-3 sm:grid-cols-2 mb-8">
          {list.map((a) => (
            <li key={a.ID} className="card">
              <h2 className="text-lg">{a.Label}</h2>
              <p className="text-sm">{a.RecipientName} · {a.RecipientPhone}</p>
              <p className="text-sm text-ink-muted">{a.AddressLine}, {a.District} {a.City}</p>
              {a.DriverNote && <p className="text-sm mt-1">Catatan: {a.DriverNote}</p>}
            </li>
          ))}
        </ul>
      </State>

      <h2 className="text-2xl mb-4">{t('addresses.add')}</h2>

      {/* Google Maps is not wired yet (blocked on the API keys), so the pin is
          entered as coordinates for now. The rule is unchanged: no pin, no
          address — the pin is what we route by. */}
      <p className="mb-4 text-sm text-ink-muted max-w-prose">
        Titik peta wajib diisi. Kami mengantar berdasarkan titik ini, bukan teks alamat.
        Pemilih peta menyusul setelah kunci Google Maps tersedia.
      </p>

      <form onSubmit={submit} className="grid gap-4 sm:grid-cols-2 max-w-3xl" noValidate>
        <div>
          <label className="label" htmlFor="label">{t('addresses.label')}</label>
          <input id="label" className="field" placeholder={t('addresses.label_placeholder')} value={form.label} onChange={set('label')} required />
          <FieldError message={fields.label} />
        </div>
        <div>
          <label className="label" htmlFor="rname">{t('addresses.recipient')}</label>
          <input id="rname" className="field" value={form.recipient_name} onChange={set('recipient_name')} required />
          <FieldError message={fields.recipient_name} />
        </div>
        <div>
          <label className="label" htmlFor="rphone">{t('addresses.recipient_phone')}</label>
          <input id="rphone" className="field" inputMode="tel" value={form.recipient_phone} onChange={set('recipient_phone')} required />
          <FieldError message={fields.recipient_phone} />
        </div>
        <div>
          <label className="label" htmlFor="district">{t('addresses.district')}</label>
          <input id="district" className="field" value={form.district} onChange={set('district')} />
        </div>
        <div className="sm:col-span-2">
          <label className="label" htmlFor="line">{t('addresses.line')}</label>
          <input id="line" className="field" value={form.address_line} onChange={set('address_line')} required />
          <FieldError message={fields.address_line} />
        </div>
        <div>
          <label className="label" htmlFor="lat">Latitude</label>
          <input id="lat" className="field" inputMode="decimal" placeholder="-6.2607"
                 value={form.latitude} onChange={set('latitude')} required />
          <FieldError message={fields.latitude} />
        </div>
        <div>
          <label className="label" htmlFor="lng">Longitude</label>
          <input id="lng" className="field" inputMode="decimal" placeholder="106.8145"
                 value={form.longitude} onChange={set('longitude')} required />
          <FieldError message={fields.longitude} />
        </div>
        <div className="sm:col-span-2">
          <label className="label" htmlFor="note">{t('addresses.note')}</label>
          <input id="note" className="field" placeholder={t('addresses.note_placeholder')}
                 value={form.driver_note} onChange={set('driver_note')} />
        </div>
        <div className="sm:col-span-2">
          <FieldError message={error ?? undefined} />
          <SubmitButton pending={pending}>{t('addresses.save')}</SubmitButton>
        </div>
      </form>

      {/* Serviceability is answered on SAVE, not at checkout — telling someone
          their area is uncovered while they are paying is the worst moment. */}
      {result && (
        <div className={`card mt-6 max-w-3xl ${result.serviceable ? '' : 'border-berry-deep'}`} role="status">
          {result.serviceable ? (
            <p>
              Dilayani oleh <strong>{result.kitchen_name}</strong>
              {result.distance_km ? ` · ${result.distance_km.toFixed(1)} km` : ''}
              {result.delivery_fee ? ` · ongkir ${result.delivery_fee}` : ''}
            </p>
          ) : (
            <p>{result.message}</p>
          )}
        </div>
      )}
    </div>
  )
}
