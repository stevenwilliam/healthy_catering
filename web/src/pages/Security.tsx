import { FormEvent, useEffect, useState } from 'react'
import {
  ApiFailure,
  MfaStatus,
  mfaConfirm,
  mfaDisable,
  mfaStart,
  mfaStatus,
} from '../lib/api'
import { CopyButton, FieldError, SubmitButton } from '../components/ui'

// Security is where a signed-in user manages their second factor.
export default function Security() {
  const [status, setStatus] = useState<MfaStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  // Enrolment in progress.
  const [secret, setSecret] = useState<string | null>(null)
  const [otpauth, setOtpauth] = useState('')
  const [code, setCode] = useState('')
  const [recovery, setRecovery] = useState<string[] | null>(null)

  const [password, setPassword] = useState('')

  async function refresh() {
    try {
      setStatus(await mfaStatus())
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : 'Tidak dapat memuat status.')
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  async function start() {
    setPending(true)
    setError(null)
    try {
      const out = await mfaStart()
      setSecret(out.secret)
      setOtpauth(out.otpauth_url)
      setCode('')
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : 'Tidak dapat memulai pendaftaran.')
    } finally {
      setPending(false)
    }
  }

  async function confirm(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      const out = await mfaConfirm(code)
      setRecovery(out.recovery_codes)
      setSecret(null)
      await refresh()
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : 'Kode tidak dapat diverifikasi.')
      setCode('')
    } finally {
      setPending(false)
    }
  }

  async function disable(e: FormEvent) {
    e.preventDefault()
    setPending(true)
    setError(null)
    try {
      await mfaDisable(password)
      setPassword('')
      setRecovery(null)
      await refresh()
    } catch (err) {
      setError(err instanceof ApiFailure ? err.message : 'Tidak dapat menonaktifkan.')
    } finally {
      setPending(false)
    }
  }

  if (!status) return <p>Memuat…</p>

  if (!status.available) {
    return (
      <div className="mx-auto max-w-xl">
        <h1 className="text-3xl mb-4">Keamanan</h1>
        <p className="text-sm text-ink-muted">
          Verifikasi dua langkah belum dikonfigurasi di server ini. Hubungi
          administrator sistem.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-xl">
      <h1 className="text-3xl mb-2">Keamanan</h1>
      <p className="text-sm text-ink-muted mb-6">
        Verifikasi dua langkah menambahkan kode dari ponsel Anda saat masuk.
        {status.required && ' Untuk peran Anda, ini wajib.'}
      </p>

      <FieldError message={error ?? undefined} />

      {/* The recovery codes are shown once and only once. */}
      {recovery && (
        <div className="mb-8 rounded border border-nourish-deep/60 p-4">
          <h2 className="text-xl mb-2">Simpan kode pemulihan Anda</h2>
          <p className="text-sm text-ink-muted mb-4">
            Setiap kode hanya dapat dipakai satu kali. Ini satu-satunya kali kode
            ditampilkan — tanpa kode ini, ponsel yang hilang berarti akun yang hilang.
          </p>
          <ul className="grid grid-cols-2 gap-2 font-mono text-sm">
            {recovery.map((c) => (
              <li key={c} className="rounded bg-white border border-nourish-deep/60 px-3 py-2">{c}</li>
            ))}
          </ul>
          <div className="mt-4">
            <CopyButton value={recovery.join('\n')} label="Salin semua" />
          </div>
        </div>
      )}

      {status.enabled ? (
        <div>
          <p className="mb-4">
            <span className="font-medium">Aktif.</span>{' '}
            {typeof status.recovery_codes_left === 'number' &&
              `${status.recovery_codes_left} kode pemulihan tersisa.`}
          </p>
          {status.required ? (
            <p className="text-sm text-ink-muted">
              Verifikasi dua langkah wajib untuk peran Anda dan tidak dapat dimatikan.
            </p>
          ) : (
            <form onSubmit={disable} className="mt-6">
              <label className="label" htmlFor="pw">
                Masukkan kata sandi untuk menonaktifkan
              </label>
              <input
                id="pw"
                className="field"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
              <SubmitButton pending={pending}>Nonaktifkan</SubmitButton>
            </form>
          )}
        </div>
      ) : secret ? (
        <div>
          <h2 className="text-xl mb-2">Langkah 1 — pindai atau ketik</h2>
          <p className="text-sm text-ink-muted mb-3">
            Tambahkan ini ke Google Authenticator, Authy atau sejenisnya.
          </p>
          <p className="font-mono text-sm break-all rounded bg-white border border-nourish-deep/60 px-3 py-2 mb-2">
            {secret}
          </p>
          <p className="text-xs text-ink-muted mb-6 break-all">{otpauth}</p>

          <h2 className="text-xl mb-2">Langkah 2 — buktikan</h2>
          <form onSubmit={confirm} noValidate>
            <label className="label" htmlFor="code">Kode enam digit</label>
            <input
              id="code"
              className="field tracking-[0.4em] text-center text-xl"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={8}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
            />
            <p className="mt-2 mb-4 text-xs text-ink-muted">
              Tidak ada yang berubah sampai kode ini benar.
            </p>
            <SubmitButton pending={pending}>Aktifkan</SubmitButton>
          </form>
        </div>
      ) : (
        <button type="button" className="btn-primary" disabled={pending} onClick={() => void start()}>
          Aktifkan verifikasi dua langkah
        </button>
      )}
    </div>
  )
}
