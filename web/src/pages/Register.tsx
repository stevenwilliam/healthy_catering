import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiFailure, registerCustomer } from '../lib/api'
import { FieldError, SubmitButton } from '../components/ui'

export default function Register() {
  const [form, setForm] = useState({ email: '', password: '', full_name: '', phone: '' })
  const [pending, setPending] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [fields, setFields] = useState<Record<string, string>>({})

  function set(k: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [k]: e.target.value })
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setPending(true); setError(null); setFields({})

    // Client-side validation is for FEEDBACK only. The server re-checks all of
    // it, because this form can be bypassed with curl (CLAUDE.md §4).
    if (form.password.length < 12) {
      setFields({ password: 'Gunakan minimal 12 karakter.' })
      setPending(false)
      return
    }

    try {
      await registerCustomer(form)
      setDone(true)
    } catch (err) {
      if (err instanceof ApiFailure) {
        setError(err.message)
        setFields(err.details)
      } else {
        setError('Pendaftaran gagal.')
      }
    } finally {
      setPending(false)
    }
  }

  if (done) {
    return (
      <div className="mx-auto max-w-md">
        <h1 className="text-3xl mb-4">Cek email Anda</h1>
        <p>Kami mengirim tautan konfirmasi. Konfirmasi email sebelum pesanan pertama Anda.</p>
        <p className="mt-6"><Link className="underline" to="/login">Kembali ke halaman masuk</Link></p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-md">
      <h1 className="text-3xl mb-6">Daftar</h1>
      <form onSubmit={submit} noValidate>
        <div className="mb-4">
          <label className="label" htmlFor="name">Nama lengkap</label>
          <input id="name" className="field" value={form.full_name} onChange={set('full_name')} required />
          <FieldError message={fields.full_name} />
        </div>
        <div className="mb-4">
          <label className="label" htmlFor="email">Email</label>
          <input id="email" className="field" type="email" autoComplete="email"
                 value={form.email} onChange={set('email')} required />
          <FieldError message={fields.email} />
        </div>
        <div className="mb-4">
          <label className="label" htmlFor="phone">Nomor HP</label>
          <input id="phone" className="field" inputMode="tel" placeholder="0812 3456 7890"
                 value={form.phone} onChange={set('phone')} />
          <FieldError message={fields.phone} />
        </div>
        <div className="mb-4">
          <label className="label" htmlFor="password">Kata sandi</label>
          <input id="password" className="field" type="password" autoComplete="new-password"
                 value={form.password} onChange={set('password')} required />
          <p className="text-xs text-ink-muted mt-1">Minimal 12 karakter.</p>
          <FieldError message={fields.password} />
        </div>
        <FieldError message={error ?? undefined} />
        <SubmitButton pending={pending}>Daftar</SubmitButton>
      </form>
    </div>
  )
}
