import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiFailure, completeMfa, login } from '../lib/api'
import { FieldError, SubmitButton } from '../components/ui'

export default function Login() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // The second step. Holding the challenge in component state — not
  // localStorage — means closing the tab abandons the half-finished sign-in
  // rather than leaving it lying around.
  const [challenge, setChallenge] = useState<string | null>(null)
  const [code, setCode] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      const session = await login(email, password)
      if (session.mfa_required && session.mfa_token) {
        setChallenge(session.mfa_token)
        setCode('')
        return
      }
      nav('/menu')
    } catch (err) {
      // The API deliberately gives one message for every failure so it cannot
      // be used to discover which addresses have accounts. We show it as-is.
      setError(err instanceof ApiFailure ? err.message : 'Tidak dapat masuk.')
    } finally {
      setPending(false)
    }
  }

  async function submitCode(e: FormEvent) {
    e.preventDefault()
    if (!challenge) return
    setPending(true)
    setError(null)
    try {
      await completeMfa(challenge, code)
      nav('/menu')
    } catch (err) {
      const message = err instanceof ApiFailure ? err.message : 'Kode tidak dapat diverifikasi.'
      setError(message)
      // An expired challenge cannot be retried with another code, so send them
      // back to the password step instead of leaving them typing into a form
      // that can never succeed.
      if (err instanceof ApiFailure && err.status === 401 && /kedaluwarsa|expired/i.test(message)) {
        setChallenge(null)
      }
      setCode('')
    } finally {
      setPending(false)
    }
  }

  if (challenge) {
    return (
      <div className="mx-auto max-w-md">
        <h1 className="text-3xl mb-2">Verifikasi dua langkah</h1>
        {/* The API's mfa_hint is English, for API clients. This UI is
            Indonesian and writes its own copy rather than rendering a server
            string in the wrong language. */}
        <p className="text-sm text-ink-muted mb-6">
          Masukkan kode enam digit dari aplikasi autentikator Anda.
        </p>
        <form onSubmit={submitCode} noValidate>
          <div className="mb-4">
            <label className="label" htmlFor="code">Kode</label>
            <input
              id="code"
              className="field tracking-[0.4em] text-center text-xl"
              // Not type="number": a leading zero matters and the spinner is
              // useless here. inputMode brings up the numeric keypad on a phone.
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              autoFocus
              maxLength={14}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
            />
            <p className="mt-2 text-xs text-ink-muted">
              Kehilangan ponsel? Masukkan salah satu kode pemulihan Anda di sini.
            </p>
          </div>
          <FieldError message={error ?? undefined} />
          <SubmitButton pending={pending}>Verifikasi</SubmitButton>
        </form>
        <p className="mt-6 text-sm">
          <button type="button" className="underline" onClick={() => { setChallenge(null); setError(null) }}>
            Kembali
          </button>
        </p>
      </div>
    )
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
