import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiFailure, login } from '../lib/api'
import { FieldError, SubmitButton } from '../components/ui'

export default function Login() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      await login(email, password)
      nav('/menu')
    } catch (err) {
      // The API deliberately gives one message for every failure so it cannot
      // be used to discover which addresses have accounts. We show it as-is.
      setError(err instanceof ApiFailure ? err.message : 'Tidak dapat masuk.')
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="mx-auto max-w-md">
      <h1 className="text-3xl mb-6">Masuk</h1>
      <form onSubmit={submit} noValidate>
        <div className="mb-4">
          <label className="label" htmlFor="email">Email</label>
          <input id="email" className="field" type="email" autoComplete="email"
                 value={email} onChange={(e) => setEmail(e.target.value)} required />
        </div>
        <div className="mb-4">
          <label className="label" htmlFor="password">Kata sandi</label>
          <input id="password" className="field" type="password" autoComplete="current-password"
                 value={password} onChange={(e) => setPassword(e.target.value)} required />
        </div>
        <FieldError message={error ?? undefined} />
        <SubmitButton pending={pending}>Masuk</SubmitButton>
      </form>
      <p className="mt-6 text-sm">
        Belum punya akun? <Link className="underline" to="/register">Daftar</Link>
      </p>
    </div>
  )
}
